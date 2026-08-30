package embyproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"litepan/internal/domain"
	"litepan/internal/httpx"
	"litepan/internal/playback"
	"litepan/internal/proxybase"
	"litepan/internal/settings"
	"litepan/pkg/jsonvalue"
)

const (
	maskedSecret = "******"
)

type Service struct {
	settings *settings.Service
	playback *playback.Service
	log      *slog.Logger
	client   *http.Client

	servePlayback func(http.ResponseWriter, *http.Request, playback.Request, playback.Intent) error

	mu       sync.Mutex
	runtimes map[string]*runtime
}

type runtime struct {
	server *http.Server
	port   int
	err    string
}

type Options struct {
	Settings *settings.Service
	Playback *playback.Service
	Log      *slog.Logger
}

type Config struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	EmbyURL   string `json:"emby_url"`
	APIKey    string `json:"api_key"`
	Port      string `json:"proxy_port"`
	ProxyURL  string `json:"proxy_url"`
	Running   bool   `json:"running"`
	LastError string `json:"last_error,omitempty"`
}

type storedConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	EmbyURL string `json:"emby_url"`
	APIKey  string `json:"api_key"`
	Port    string `json:"proxy_port"`
}

type UpdateRequest struct {
	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	EmbyURL string                   `json:"emby_url"`
	APIKey  string                   `json:"api_key"`
	Port    jsonvalue.FlexibleString `json:"proxy_port"`
}

type State struct {
	Enabled bool     `json:"enabled"`
	Items   []Config `json:"items"`
}

type RefreshResult struct {
	ConfigID    string `json:"config_id"`
	ConfigName  string `json:"config_name"`
	Mode        string `json:"mode"`
	TaskID      string `json:"task_id,omitempty"`
	LibraryID   string `json:"library_id,omitempty"`
	LibraryName string `json:"library_name,omitempty"`
}

type RefreshRequest struct {
	ConfigID  string `json:"config_id"`
	Mode      string `json:"mode"`
	LibraryID string `json:"library_id"`
}

type Library struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CollectionType string `json:"collection_type,omitempty"`
}

func New(opts Options) *Service {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	client := httpx.NewClient(httpx.ClientOptions{DisableCompression: true})
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Service{
		settings: opts.Settings,
		playback: opts.Playback,
		log:      log,
		client:   client,
		runtimes: map[string]*runtime{},
		servePlayback: func(w http.ResponseWriter, r *http.Request, req playback.Request, intent playback.Intent) error {
			if opts.Playback == nil {
				return domain.Errf(domain.CodeNotImplement)
			}
			return opts.Playback.ServeHTTP(w, r, req, intent)
		},
	}
}

func (s *Service) Snapshots(r *http.Request) []Config {
	configs := s.configsFromSettings()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range configs {
		configs[i].APIKey = maskSecret(configs[i].APIKey)
		if configs[i].Port != "" {
			configs[i].ProxyURL = proxybase.PublicBase(r, configs[i].Port)
		}
		if rt := s.runtimes[configs[i].ID]; rt != nil {
			configs[i].Running = rt.server != nil
			configs[i].LastError = rt.err
		}
	}
	return configs
}

func (s *Service) State(r *http.Request) State {
	items := s.Snapshots(r)
	return State{Enabled: s.enabled() && len(items) > 0, Items: items}
}

func (s *Service) Snapshot(r *http.Request) Config {
	configs := s.Snapshots(r)
	if len(configs) > 0 {
		return configs[0]
	}
	return Config{}
}

func (s *Service) Replace(ctx context.Context, enabled bool, inputs []UpdateRequest) (State, error) {
	if s.settings == nil {
		return State{}, domain.Errf(domain.CodeNotImplement)
	}
	stored := s.configsFromSettings()
	storedByID := make(map[string]Config, len(stored))
	for _, cfg := range stored {
		storedByID[cfg.ID] = cfg
	}
	configs := make([]Config, 0, len(inputs))
	seenIDs := make(map[string]struct{}, len(inputs))
	seenNames := make(map[string]struct{}, len(inputs))
	seenPorts := make(map[string]struct{}, len(inputs))
	for _, in := range inputs {
		cfg, err := ConfigFromUpdate(in)
		if err != nil {
			return State{}, err
		}
		if cfg.ID == "" {
			cfg.ID = uuid.NewString()
		}
		if _, ok := seenIDs[cfg.ID]; ok {
			return State{}, domain.Errorf(domain.CodeValidation, "Emby 配置重复")
		}
		seenIDs[cfg.ID] = struct{}{}
		nameKey := strings.ToLower(cfg.Name)
		if _, ok := seenNames[nameKey]; ok {
			return State{}, domain.Errorf(domain.CodeValidation, "Emby 配置名称不能重复")
		}
		seenNames[nameKey] = struct{}{}
		if old, ok := storedByID[cfg.ID]; ok && isStoredSecretInput(cfg.APIKey, old.APIKey) {
			cfg.APIKey = old.APIKey
		}
		if cfg.EmbyURL == "" || cfg.APIKey == "" {
			return State{}, domain.Errorf(domain.CodeValidation, "请填写 Emby 地址和 API Key")
		}
		if enabled && cfg.Port == "" {
			return State{}, domain.Errorf(domain.CodeValidation, "启用 Emby 反代前，请为所有配置填写反代端口")
		}
		if cfg.Port != "" {
			if _, ok := seenPorts[cfg.Port]; ok {
				return State{}, domain.Errorf(domain.CodeValidation, "多个 Emby 反代不能使用同一个端口")
			}
			seenPorts[cfg.Port] = struct{}{}
			if err := s.checkFnosPortConflict(cfg.Port); err != nil {
				return State{}, err
			}
		}
		configs = append(configs, cfg)
	}
	if len(configs) == 0 {
		enabled = false
	}
	raw, err := json.Marshal(storedConfigs(configs))
	if err != nil {
		return State{}, domain.Wrap(domain.CodeInternal, err)
	}
	if err := s.settings.Update(ctx, map[string]string{
		settings.KeyEmbyEnabled:        strconv.FormatBool(enabled),
		settings.KeyEmbyProxyInstances: string(raw),
	}); err != nil {
		return State{}, err
	}
	if err := s.Sync(ctx); err != nil {
		return s.State(nil), err
	}
	return s.State(nil), nil
}

func storedConfigs(configs []Config) []storedConfig {
	out := make([]storedConfig, 0, len(configs))
	for _, cfg := range configs {
		out = append(out, storedConfig{
			ID: cfg.ID, Name: cfg.Name, EmbyURL: cfg.EmbyURL, APIKey: cfg.APIKey, Port: cfg.Port,
		})
	}
	return out
}

func (s *Service) checkFnosPortConflict(port string) error {
	if s.settings == nil || port == "" {
		return nil
	}
	if !s.settings.Bool(settings.KeyFnosEnabled) {
		return nil
	}
	fnosPort := strings.TrimSpace(s.settings.String(settings.KeyFnosProxyPort))
	if fnosPort != "" && fnosPort == port {
		return domain.Errorf(domain.CodeValidation, "反代端口与飞牛影视反代端口冲突")
	}
	return nil
}

func (s *Service) UsesPort(port string) bool {
	if !s.enabled() {
		return false
	}
	port = strings.TrimSpace(port)
	for _, cfg := range s.configsFromSettings() {
		if cfg.Port == port {
			return true
		}
	}
	return false
}

func (s *Service) Test(ctx context.Context) error {
	cfg, err := s.resolveConfig("")
	if err != nil {
		return err
	}
	return s.TestConfig(ctx, cfg)
}

func (s *Service) TestUpdate(ctx context.Context, in UpdateRequest) error {
	cfg, err := ConfigFromUpdate(in)
	if err != nil {
		return err
	}
	if old, findErr := s.resolveConfig(in.ID); findErr == nil && isStoredSecretInput(in.APIKey, old.APIKey) {
		cfg.APIKey = old.APIKey
	}
	return s.TestConfig(ctx, cfg)
}

func (s *Service) TestConfig(ctx context.Context, cfg Config) error {
	if strings.TrimSpace(cfg.EmbyURL) == "" {
		return domain.Errorf(domain.CodeValidation, "请先填写 Emby 地址")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return domain.Errorf(domain.CodeValidation, "请先填写 Emby API Key")
	}
	testCtx, cancel := context.WithTimeout(ctx, proxybase.TestRequestTimeout)
	defer cancel()
	testURL := cfg.EmbyURL + "/System/Info?" + url.Values{"api_key": {cfg.APIKey}}.Encode()
	req, err := http.NewRequestWithContext(testCtx, http.MethodGet, testURL, nil)
	if err != nil {
		return domain.Errorf(domain.CodeValidation, "Emby 地址无效")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return embyTestConnectError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return embyTestHTTPError(resp.StatusCode)
	}
	return nil
}

func (s *Service) ListLibraries(ctx context.Context, configIDs ...string) ([]Library, error) {
	configID := ""
	if len(configIDs) > 0 {
		configID = configIDs[0]
	}
	cfg, err := s.resolveConfig(configID)
	if err != nil {
		return nil, err
	}
	return s.listLibraries(ctx, cfg)
}

func (s *Service) listLibraries(ctx context.Context, cfg Config) ([]Library, error) {
	if strings.TrimSpace(cfg.EmbyURL) == "" {
		return nil, domain.Errorf(domain.CodeValidation, "请先填写 Emby 地址")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, domain.Errorf(domain.CodeValidation, "请先填写 Emby API Key")
	}
	base := strings.TrimRight(cfg.EmbyURL, "/")
	query := url.Values{"api_key": {cfg.APIKey}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/Library/SelectableMediaFolders?"+query, nil)
	if err != nil {
		return nil, domain.Wrap(domain.CodeInternal, err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, embyTestConnectError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, embyTestHTTPError(resp.StatusCode)
	}
	var payload []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, domain.Wrap(domain.CodeInternal, err)
	}
	out := make([]Library, 0, len(payload))
	for _, item := range payload {
		id := strings.TrimSpace(anyString(item["Id"]))
		name := strings.TrimSpace(anyString(item["Name"]))
		if id == "" || name == "" {
			continue
		}
		out = append(out, Library{
			ID:             id,
			Name:           name,
			CollectionType: strings.TrimSpace(anyString(item["CollectionType"])),
		})
	}
	return out, nil
}

func (s *Service) RefreshLibrary(ctx context.Context, req RefreshRequest) (RefreshResult, error) {
	cfg, err := s.resolveConfig(req.ConfigID)
	if err != nil {
		return RefreshResult{}, err
	}
	if strings.TrimSpace(cfg.EmbyURL) == "" {
		return RefreshResult{}, domain.Errorf(domain.CodeValidation, "请先填写 Emby 地址")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return RefreshResult{}, domain.Errorf(domain.CodeValidation, "请先填写 Emby API Key")
	}
	base := strings.TrimRight(cfg.EmbyURL, "/")
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "global"
	}
	if mode == "library" {
		result, err := s.refreshLibraryByID(ctx, cfg, strings.TrimSpace(req.LibraryID))
		return withRefreshConfig(result, cfg), err
	}
	result, err := s.refreshAllLibraries(ctx, base, cfg.APIKey)
	return withRefreshConfig(result, cfg), err
}

func withRefreshConfig(result RefreshResult, cfg Config) RefreshResult {
	result.ConfigID = cfg.ID
	result.ConfigName = cfg.Name
	return result
}

func (s *Service) refreshAllLibraries(ctx context.Context, base, apiKey string) (RefreshResult, error) {
	query := url.Values{"api_key": {apiKey}}.Encode()
	taskID, err := s.findLibraryRefreshTask(ctx, base, query)
	if err == nil && taskID != "" {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, base+"/ScheduledTasks/Running/"+taskID+"?"+query, nil)
		if reqErr != nil {
			return RefreshResult{}, domain.Wrap(domain.CodeInternal, reqErr)
		}
		resp, doErr := s.client.Do(req)
		if doErr != nil {
			return RefreshResult{}, embyTestConnectError(doErr)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 400 {
			return RefreshResult{Mode: "scheduled_task", TaskID: taskID}, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/Library/Refresh?"+query, nil)
	if err != nil {
		return RefreshResult{}, domain.Wrap(domain.CodeInternal, err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return RefreshResult{}, embyTestConnectError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return RefreshResult{}, embyTestHTTPError(resp.StatusCode)
	}
	return RefreshResult{Mode: "global"}, nil
}

func (s *Service) refreshLibraryByID(ctx context.Context, cfg Config, libraryID string) (RefreshResult, error) {
	if libraryID == "" {
		return RefreshResult{}, domain.Errorf(domain.CodeValidation, "请选择 Emby 媒体库")
	}
	libraries, err := s.listLibraries(ctx, cfg)
	if err != nil {
		return RefreshResult{}, err
	}
	var selected *Library
	for i := range libraries {
		if libraries[i].ID == libraryID {
			selected = &libraries[i]
			break
		}
	}
	if selected == nil {
		return RefreshResult{}, domain.Errorf(domain.CodeValidation, "所选 Emby 媒体库不存在")
	}
	base := strings.TrimRight(cfg.EmbyURL, "/")
	query := url.Values{
		"Recursive":           {"true"},
		"ImageRefreshMode":    {"Default"},
		"MetadataRefreshMode": {"Default"},
		"ReplaceAllImages":    {"false"},
		"ReplaceAllMetadata":  {"false"},
		"api_key":             {cfg.APIKey},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/Items/"+url.PathEscape(libraryID)+"/Refresh?"+query, nil)
	if err != nil {
		return RefreshResult{}, domain.Wrap(domain.CodeInternal, err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return RefreshResult{}, embyTestConnectError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return RefreshResult{}, embyTestHTTPError(resp.StatusCode)
	}
	return RefreshResult{
		Mode:        "library",
		LibraryID:   selected.ID,
		LibraryName: selected.Name,
	}, nil
}

func (s *Service) findLibraryRefreshTask(ctx context.Context, base, query string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/ScheduledTasks?"+query, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", domain.Errorf(domain.CodeDriverError, "读取 Emby 计划任务失败")
	}
	var tasks []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return "", err
	}
	for _, task := range tasks {
		name := strings.ToLower(strings.TrimSpace(anyString(task["Name"])))
		key := strings.TrimSpace(anyString(task["Key"]))
		if strings.Contains(name, "扫描媒体库") || strings.Contains(name, "refresh library") || strings.Contains(name, "scan media library") || key == "RefreshLibrary" {
			return strings.TrimSpace(anyString(task["Id"])), nil
		}
	}
	return "", nil
}

func ConfigFromUpdate(in UpdateRequest) (Config, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Config{}, domain.Errorf(domain.CodeValidation, "请输入 Emby 配置名称")
	}
	if len([]rune(name)) > 40 {
		return Config{}, domain.Errorf(domain.CodeValidation, "Emby 配置名称不能超过 40 个字符")
	}
	embyURL, err := normalizeEmbyURL(in.EmbyURL, false)
	if err != nil {
		return Config{}, err
	}
	port, err := proxybase.NormalizeOptionalPort(in.Port.String())
	if err != nil {
		return Config{}, err
	}
	return Config{
		ID:      strings.TrimSpace(in.ID),
		Name:    name,
		EmbyURL: embyURL,
		APIKey:  strings.TrimSpace(in.APIKey),
		Port:    port,
	}, nil
}

func (s *Service) Start(ctx context.Context) {
	if err := s.Sync(ctx); err != nil {
		s.log.Warn("Emby 反代启动失败", "error", err)
	}
}

func (s *Service) Sync(ctx context.Context) error {
	configs := s.configsFromSettings()
	enabled := s.enabled() && len(configs) > 0
	wanted := make(map[string]Config, len(configs))
	if enabled {
		for _, cfg := range configs {
			wanted[cfg.ID] = cfg
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, rt := range s.runtimes {
		cfg, ok := wanted[id]
		port, _ := strconv.Atoi(cfg.Port)
		if !ok || port == 0 || rt.port != port {
			s.stopRuntimeLocked(ctx, id)
		}
	}
	var firstErr error
	for _, cfg := range configs {
		port, _ := strconv.Atoi(cfg.Port)
		if !enabled || port == 0 {
			continue
		}
		if rt := s.runtimes[cfg.ID]; rt != nil && rt.server != nil && rt.port == port {
			rt.err = ""
			continue
		}
		if err := s.startRuntimeLocked(cfg, port); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Service) Shutdown(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.runtimes {
		s.stopRuntimeLocked(ctx, id)
	}
}

func (s *Service) startRuntimeLocked(cfg Config, port int) error {
	rt := &runtime{port: port}
	s.runtimes[cfg.ID] = rt
	if cfg.EmbyURL == "" || cfg.APIKey == "" {
		rt.err = "启用反代时需要填写 Emby 地址和 API Key"
		return domain.Errorf(domain.CodeValidation, "%s", rt.err)
	}
	if err := s.checkFnosPortConflict(cfg.Port); err != nil {
		rt.err = err.Error()
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.handleConfig(cfg.ID, w, r)
	})
	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		rt.err = fmt.Sprintf("Emby 反代端口 %d 监听失败：%v", port, err)
		return domain.Errorf(domain.CodeDriverError, "%s", rt.err)
	}
	rt.server = srv
	go func(id, name string, active *runtime) {
		s.log.Info("Emby 反代已监听", "name", name, "addr", srv.Addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			if s.runtimes[id] == active {
				active.err = err.Error()
				active.server = nil
				active.port = 0
			}
			s.mu.Unlock()
			s.log.Error("Emby 反代服务异常退出", "name", name, "error", err)
		}
	}(cfg.ID, cfg.Name, rt)
	return nil
}

func (s *Service) stopRuntimeLocked(ctx context.Context, id string) {
	rt := s.runtimes[id]
	if rt == nil {
		return
	}
	delete(s.runtimes, id)
	if rt.server == nil {
		return
	}
	srv := rt.server
	stopCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(stopCtx); err != nil {
		_ = srv.Close()
	}
}

func (s *Service) configsFromSettings() []Config {
	if s.settings == nil {
		return nil
	}
	var configs []Config
	if err := json.Unmarshal([]byte(s.settings.String(settings.KeyEmbyProxyInstances)), &configs); err != nil {
		s.log.Error("Emby 反代配置解析失败", "error", err)
		return nil
	}
	for i := range configs {
		configs[i].ID = strings.TrimSpace(configs[i].ID)
		configs[i].Name = strings.TrimSpace(configs[i].Name)
		configs[i].EmbyURL = strings.TrimRight(strings.TrimSpace(configs[i].EmbyURL), "/")
		configs[i].APIKey = strings.TrimSpace(configs[i].APIKey)
		configs[i].Port = strings.TrimSpace(configs[i].Port)
	}
	return configs
}

func (s *Service) enabled() bool {
	return s.settings != nil && s.settings.Bool(settings.KeyEmbyEnabled)
}

func (s *Service) resolveConfig(id string) (Config, error) {
	id = strings.TrimSpace(id)
	configs := s.configsFromSettings()
	if id != "" {
		for _, cfg := range configs {
			if cfg.ID == id {
				return cfg, nil
			}
		}
		return Config{}, domain.Errorf(domain.CodeValidation, "所选 Emby 配置不存在")
	}
	if len(configs) > 0 {
		return configs[0], nil
	}
	return Config{}, domain.Errorf(domain.CodeValidation, "请先配置 Emby")
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.resolveConfig("")
	if err != nil {
		http.Error(w, "Emby proxy is not configured", http.StatusNotFound)
		return
	}
	s.handleWithConfig(cfg, w, r)
}

func (s *Service) handleConfig(configID string, w http.ResponseWriter, r *http.Request) {
	cfg, err := s.resolveConfig(configID)
	if err != nil {
		http.Error(w, "Emby proxy is not configured", http.StatusNotFound)
		return
	}
	s.handleWithConfig(cfg, w, r)
}

func (s *Service) handleWithConfig(cfg Config, w http.ResponseWriter, r *http.Request) {
	if !s.enabled() || cfg.EmbyURL == "" || cfg.APIKey == "" {
		http.Error(w, "Emby proxy is not enabled", http.StatusNotFound)
		return
	}
	fullPath := strings.TrimPrefix(r.URL.Path, "/")
	s.proxyRequest(w, r, cfg, fullPath)
}

func (s *Service) proxyRequest(w http.ResponseWriter, r *http.Request, cfg Config, fullPath string) {
	target, err := targetURL(cfg, fullPath, r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	var body io.Reader
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		data, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadRequest)
			return
		}
		if len(data) > 0 {
			body = bytes.NewReader(data)
		}
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	copyRequestHeaders(req.Header, r.Header, false)
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	headers := responseHeaders(resp.Header)
	if loc := headers.Get("Location"); loc != "" {
		headers.Set("Location", rewriteLocation(loc, cfg, r))
	}
	writeHeaders(w, headers)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.CopyBuffer(w, resp.Body, make([]byte, 128*1024))
}

func targetURL(cfg Config, fullPath, rawQuery string) (string, error) {
	baseStr := strings.TrimRight(strings.TrimSpace(cfg.EmbyURL), "/")
	path := strings.TrimPrefix(strings.TrimSpace(fullPath), "/")
	if strings.HasSuffix(strings.ToLower(baseStr), "/emby") && strings.HasPrefix(strings.ToLower(path), "emby/") {
		path = path[len("emby/"):]
	}
	base, err := url.Parse(baseStr + "/")
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	target := base.ResolveReference(ref)
	target.RawQuery = rawQuery
	return target.String(), nil
}

func normalizeEmbyURL(raw string, required bool) (string, error) {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	if v == "" {
		if required {
			return "", domain.Errorf(domain.CodeValidation, "请填写 Emby 地址")
		}
		return "", nil
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", domain.Errorf(domain.CodeValidation, "Emby 地址格式不正确，示例：http://192.168.1.10:8096")
	}
	return v, nil
}

func anyString(v any) string {
	switch got := v.(type) {
	case string:
		return got
	case json.Number:
		return got.String()
	case fmt.Stringer:
		return got.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

func responseHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, values := range src {
		if _, skip := proxybase.HopByHopHeaderNames[strings.ToLower(k)]; skip {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
	return dst
}

func copyRequestHeaders(dst, src http.Header, identity bool) {
	for k, values := range src {
		if _, skip := proxybase.HopByHopHeaderNames[strings.ToLower(k)]; skip {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
	if identity {
		dst.Set("Accept-Encoding", "identity")
	}
}

func writeHeaders(w http.ResponseWriter, headers http.Header) {
	for k, values := range headers {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
}

func rewriteLocation(location string, cfg Config, r *http.Request) string {
	embyURL := strings.TrimRight(cfg.EmbyURL, "/")
	if strings.HasPrefix(location, embyURL) {
		return strings.TrimRight(proxybase.PublicBase(r, cfg.Port), "/") + strings.TrimPrefix(location, embyURL)
	}
	return location
}

func maskSecret(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	n := len(v)
	if n <= 4 {
		if n <= 2 {
			return strings.Repeat("*", n)
		}
		return v[:1] + strings.Repeat("*", n-2) + v[n-1:]
	}
	if n <= 8 {
		return v[:2] + "****" + v[n-2:]
	}
	return v[:4] + "****" + v[n-4:]
}

func isStoredSecretInput(input, stored string) bool {
	input = strings.TrimSpace(input)
	stored = strings.TrimSpace(stored)
	if input == "" || stored == "" {
		return false
	}
	if input == stored || input == maskedSecret {
		return true
	}
	return input == maskSecret(stored)
}

// isExpectedClientDisconnect 识别播放器探测、跳转 Range 或重建播放链路时主动取消的旧请求。
// 这类错误不代表解析或上游故障，不应记为 Warn，也不再尝试补写 500 响应。
func isExpectedClientDisconnect(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{"broken pipe", "connection reset by peer", "use of closed network connection"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func embyTestHTTPError(status int) *domain.AppError {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.Errorf(domain.CodeDriverError, "Emby API Key 不正确")
	case http.StatusNotFound:
		return domain.Errorf(domain.CodeDriverError, "Emby 地址不正确，请检查服务地址")
	default:
		if status >= 500 {
			return domain.Errorf(domain.CodeDriverError, "Emby 服务异常，请稍后重试")
		}
		return domain.Errorf(domain.CodeDriverError, "Emby 地址无法访问")
	}
}

func embyTestConnectError(err error) *domain.AppError {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.Errorf(domain.CodeDriverError, "Emby 地址连接超时，请检查网络与服务是否在线")
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return domain.Errorf(domain.CodeDriverError, "Emby 地址无法连接，请检查地址和端口是否正确")
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "cannot resolve"), strings.Contains(msg, "lookup"):
		return domain.Errorf(domain.CodeDriverError, "Emby 地址无法解析，请检查主机名或 IP")
	case strings.Contains(msg, "context deadline exceeded"), strings.Contains(msg, "timeout"):
		return domain.Errorf(domain.CodeDriverError, "Emby 地址连接超时，请检查网络与服务是否在线")
	default:
		return domain.Errorf(domain.CodeDriverError, "Emby 地址无法连接，请检查地址是否正确")
	}
}

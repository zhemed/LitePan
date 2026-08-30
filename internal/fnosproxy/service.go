package fnosproxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"litepan/internal/domain"
	"litepan/internal/httpx"
	"litepan/internal/playback"
	"litepan/internal/proxybase"
	"litepan/internal/settings"
	"litepan/pkg/jsonvalue"
)

var (
	baseHTMLPlayerPathRE = regexp.MustCompile(`(?i)^(?:/?emby)?/?web/modules/htmlvideoplayer/basehtmlplayer\.js$`)
	htmlCrossOriginRE    = regexp.MustCompile(`mediaSource\.IsRemote\s*&&\s*(?:"DirectPlay"\s*===\s*playMethod|playMethod\s*===\s*"DirectPlay")\s*\?\s*null\s*:\s*"anonymous"`)
)

type Service struct {
	settings       *settings.Service
	playback       *playback.Service
	log            *slog.Logger
	client         *http.Client
	portUsedByEmby func(string) bool

	servePlayback func(w http.ResponseWriter, r *http.Request, req playback.Request, intent playback.Intent) error

	mu     sync.Mutex
	server *http.Server
	port   int
	err    string
}

type Options struct {
	Settings       *settings.Service
	Playback       *playback.Service
	Log            *slog.Logger
	PortUsedByEmby func(string) bool
}

type Config struct {
	Enabled   bool   `json:"enabled"`
	Name      string `json:"name"`
	FnosURL   string `json:"fnos_url"`
	Port      string `json:"proxy_port"`
	ProxyURL  string `json:"proxy_url"`
	Running   bool   `json:"running"`
	LastError string `json:"last_error,omitempty"`
}

type UpdateRequest struct {
	Enabled bool                     `json:"enabled"`
	Name    string                   `json:"name"`
	FnosURL string                   `json:"fnos_url"`
	Port    jsonvalue.FlexibleString `json:"proxy_port"`
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
		settings:       opts.Settings,
		playback:       opts.Playback,
		log:            log,
		client:         client,
		portUsedByEmby: opts.PortUsedByEmby,
		servePlayback: func(w http.ResponseWriter, r *http.Request, req playback.Request, intent playback.Intent) error {
			if opts.Playback == nil {
				return domain.Errf(domain.CodeNotImplement)
			}
			return opts.Playback.ServeHTTP(w, r, req, intent)
		},
	}
}

func (s *Service) Snapshot(r *http.Request) Config {
	cfg := s.configFromSettings()
	if cfg.Port != "" {
		cfg.ProxyURL = proxybase.PublicBase(r, cfg.Port)
	}
	s.mu.Lock()
	cfg.Running = s.server != nil
	cfg.LastError = s.err
	s.mu.Unlock()
	return cfg
}

func (s *Service) Update(ctx context.Context, in UpdateRequest) (Config, error) {
	if s.settings == nil {
		return Config{}, domain.Errf(domain.CodeNotImplement)
	}
	fnosURL, err := normalizeFnosURL(in.FnosURL, false)
	if err != nil {
		return Config{}, err
	}
	port, err := proxybase.NormalizeOptionalPort(in.Port.String())
	if err != nil {
		return Config{}, err
	}
	if in.Enabled && port != "" {
		if fnosURL == "" {
			return Config{}, domain.Errorf(domain.CodeValidation, "启用飞牛反代并填写端口时，需要填写飞牛影视地址")
		}
		if err := s.checkPortConflict(port); err != nil {
			return Config{}, err
		}
	}
	if err := s.settings.Update(ctx, map[string]string{
		settings.KeyFnosEnabled:   strconv.FormatBool(in.Enabled),
		settings.KeyFnosName:      normalizeConfigName(in.Name),
		settings.KeyFnosURL:       fnosURL,
		settings.KeyFnosProxyPort: port,
	}); err != nil {
		return Config{}, err
	}
	if err := s.Sync(ctx); err != nil {
		return s.Snapshot(nil), err
	}
	return s.Snapshot(nil), nil
}

func (s *Service) checkPortConflict(port string) error {
	if s.settings == nil || port == "" {
		return nil
	}
	if s.portUsedByEmby != nil && s.portUsedByEmby(port) {
		return domain.Errorf(domain.CodeValidation, "反代端口与 Emby 反代端口冲突")
	}
	return nil
}

func (s *Service) Test(ctx context.Context) error {
	return s.TestConfig(ctx, s.configFromSettings())
}

func (s *Service) TestUpdate(ctx context.Context, in UpdateRequest) error {
	cfg, err := ConfigFromUpdate(in)
	if err != nil {
		return err
	}
	return s.TestConfig(ctx, cfg)
}

func (s *Service) TestConfig(ctx context.Context, cfg Config) error {
	if strings.TrimSpace(cfg.FnosURL) == "" {
		return domain.Errorf(domain.CodeValidation, "请先填写飞牛影视地址")
	}
	testCtx, cancel := context.WithTimeout(ctx, proxybase.TestRequestTimeout)
	defer cancel()
	candidates := []string{
		cfg.FnosURL + "/System/Info/Public",
		cfg.FnosURL + "/System/Info",
		cfg.FnosURL + "/",
	}
	var lastErr error
	for _, testURL := range candidates {
		req, err := http.NewRequestWithContext(testCtx, http.MethodGet, testURL, nil)
		if err != nil {
			return domain.Errorf(domain.CodeValidation, "飞牛影视地址无效")
		}
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 500 {
			return nil
		}
		lastErr = fnosTestHTTPError(resp.StatusCode)
	}
	if lastErr != nil {
		if ae, ok := lastErr.(*domain.AppError); ok {
			return ae
		}
		return fnosTestConnectError(lastErr)
	}
	return domain.Errorf(domain.CodeDriverError, "飞牛影视地址无法访问")
}

func ConfigFromUpdate(in UpdateRequest) (Config, error) {
	fnosURL, err := normalizeFnosURL(in.FnosURL, false)
	if err != nil {
		return Config{}, err
	}
	port, err := proxybase.NormalizeOptionalPort(in.Port.String())
	if err != nil {
		return Config{}, err
	}
	return Config{
		Enabled: in.Enabled,
		Name:    normalizeConfigName(in.Name),
		FnosURL: fnosURL,
		Port:    port,
	}, nil
}

func (s *Service) Start(ctx context.Context) {
	if err := s.Sync(ctx); err != nil {
		s.log.Warn("飞牛反代启动失败", "error", err)
	}
}

func (s *Service) Sync(ctx context.Context) error {
	cfg := s.configFromSettings()
	port, _ := strconv.Atoi(cfg.Port)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil && (port == 0 || !cfg.Enabled || s.port != port) {
		s.stopLocked(ctx)
	}
	s.err = ""
	if !cfg.Enabled || port == 0 {
		return nil
	}
	if cfg.FnosURL == "" {
		s.err = "启用反代时需要填写飞牛影视地址"
		return domain.Errorf(domain.CodeValidation, "%s", s.err)
	}
	if err := s.checkPortConflict(cfg.Port); err != nil {
		s.err = err.Error()
		return err
	}
	if s.server != nil && s.port == port {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		s.err = fmt.Sprintf("飞牛反代端口 %d 监听失败：%v", port, err)
		return domain.Errorf(domain.CodeDriverError, "%s", s.err)
	}
	s.server = srv
	s.port = port
	go func() {
		s.log.Info("飞牛反代已监听", "addr", srv.Addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			s.err = err.Error()
			s.server = nil
			s.port = 0
			s.mu.Unlock()
			s.log.Error("飞牛反代服务异常退出", "error", err)
		}
	}()
	return nil
}

func (s *Service) Shutdown(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked(ctx)
}

func (s *Service) stopLocked(ctx context.Context) {
	if s.server == nil {
		return
	}
	srv := s.server
	s.server = nil
	s.port = 0
	stopCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(stopCtx); err != nil {
		_ = srv.Close()
	}
}

func (s *Service) configFromSettings() Config {
	if s.settings == nil {
		return Config{}
	}
	return Config{
		Enabled: s.settings.Bool(settings.KeyFnosEnabled),
		Name:    normalizeConfigName(s.settings.String(settings.KeyFnosName)),
		FnosURL: strings.TrimRight(strings.TrimSpace(s.settings.String(settings.KeyFnosURL)), "/"),
		Port:    strings.TrimSpace(s.settings.String(settings.KeyFnosProxyPort)),
	}
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	cfg := s.configFromSettings()
	if !cfg.Enabled || cfg.FnosURL == "" {
		http.Error(w, "飞牛反代未启用", http.StatusNotFound)
		return
	}
	fullPath := strings.TrimPrefix(r.URL.Path, "/")
	if baseHTMLPlayerPathRE.MatchString(fullPath) && r.Method == http.MethodGet {
		s.modifyBaseHTMLPlayer(w, r, cfg, fullPath)
		return
	}
	s.proxyRequest(w, r, cfg, fullPath)
}

func (s *Service) modifyBaseHTMLPlayer(w http.ResponseWriter, r *http.Request, cfg Config, fullPath string) {
	resp, body, err := s.requestUpstream(r, cfg, fullPath, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	crossOriginGuard := []byte(";try{(function(){var s=Element.prototype.setAttribute;Element.prototype.setAttribute=function(n,v){if(this&&this.tagName&&/^(VIDEO|AUDIO)$/i.test(this.tagName)&&String(n).toLowerCase()==='crossorigin')return;return s.call(this,n,v)};try{Object.defineProperty(HTMLMediaElement.prototype,'crossOrigin',{get:function(){return null},set:function(){return null},configurable:true})}catch(e){}})()}catch(e){};")
	body = bytes.ReplaceAll(body, []byte(`mediaSource.IsRemote&&"DirectPlay"===playMethod?null:"anonymous"`), []byte("null"))
	body = htmlCrossOriginRE.ReplaceAll(body, []byte("null"))
	if !bytes.Contains(body, []byte("HTMLMediaElement.prototype,'crossOrigin'")) {
		body = append(crossOriginGuard, body...)
	}
	headers := responseHeaders(resp.Header)
	headers.Set("Content-Length", strconv.Itoa(len(body)))
	headers.Set("Cache-Control", "no-store")
	headers.Del("Content-Encoding")
	writeHeaders(w, headers)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *Service) proxyRequest(w http.ResponseWriter, r *http.Request, cfg Config, fullPath string) {
	targetValue, err := targetURL(cfg, fullPath, r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	target, err := url.Parse(targetValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	proxy := &httputil.ReverseProxy{
		Transport: s.client.Transport,
		Rewrite: func(req *httputil.ProxyRequest) {
			outURL := *target
			req.Out.URL = &outURL
			req.Out.Host = target.Host
			req.SetXForwarded()
		},
		ModifyResponse: func(resp *http.Response) error {
			if loc := resp.Header.Get("Location"); loc != "" {
				resp.Header.Set("Location", rewriteLocation(loc, cfg, r))
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			if req.Context().Err() != nil {
				return
			}
			s.log.Warn("飞牛反代请求失败", "path", req.URL.Path, "error", err)
			http.Error(w, "飞牛影视服务暂时无法访问", http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
}

func (s *Service) requestUpstream(r *http.Request, cfg Config, fullPath string, identity bool) (*http.Response, []byte, error) {
	target, err := targetURL(cfg, fullPath, r.URL.RawQuery)
	if err != nil {
		return nil, nil, err
	}
	var body io.Reader
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		buf, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(buf))
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		return nil, nil, err
	}
	copyRequestHeaders(req.Header, r.Header, identity)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, data, nil
}

func (s *Service) servePlaybackHTTP(w http.ResponseWriter, r *http.Request, req playback.Request, intent playback.Intent) error {
	if s == nil || s.servePlayback == nil {
		return domain.Errf(domain.CodeNotImplement)
	}
	return s.servePlayback(w, r, req, intent)
}

func targetURL(cfg Config, fullPath, rawQuery string) (string, error) {
	baseStr := strings.TrimRight(strings.TrimSpace(cfg.FnosURL), "/")
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

func normalizeFnosURL(raw string, required bool) (string, error) {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	if v == "" {
		if required {
			return "", domain.Errorf(domain.CodeValidation, "请填写飞牛影视地址")
		}
		return "", nil
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", domain.Errorf(domain.CodeValidation, "飞牛影视地址格式不正确，示例：http://192.168.1.10:8005")
	}
	return v, nil
}

// normalizeConfigName 归一化配置名称：空名回落默认显示名，避免界面出现空白名称。
func normalizeConfigName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "飞牛影视"
	}
	return name
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

func writeUpstreamBody(w http.ResponseWriter, resp *http.Response, body []byte) {
	writeHeaders(w, responseHeaders(resp.Header))
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func rewriteLocation(location string, cfg Config, r *http.Request) string {
	fnosURL := strings.TrimRight(cfg.FnosURL, "/")
	if strings.HasPrefix(location, fnosURL) {
		return strings.TrimRight(proxybase.PublicBase(r, cfg.Port), "/") + strings.TrimPrefix(location, fnosURL)
	}
	return location
}

// embyMediaStreamNonNullFields 是部分客户端要求存在且非 null 的字段。
var embyMediaStreamNonNullFields = []string{"Type", "Language", "DisplayLanguage", "Title", "DisplayTitle"}

// normalizeEmbyMediaStreams 补齐必填字符串字段并返回是否发生修改。
func normalizeEmbyMediaStreams(mediaSource map[string]any) bool {
	if mediaSource == nil {
		return false
	}
	raw, ok := mediaSource["MediaStreams"].([]any)
	if !ok {
		return false
	}
	changed := false
	for _, item := range raw {
		stream, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, field := range embyMediaStreamNonNullFields {
			if v, exists := stream[field]; !exists || v == nil {
				stream[field] = ""
				changed = true
			}
		}
	}
	return changed
}

func fnosTestHTTPError(status int) *domain.AppError {
	switch status {
	case http.StatusNotFound:
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址不正确，请检查服务地址（默认端口 8005）")
	default:
		if status >= 500 {
			return domain.Errorf(domain.CodeDriverError, "飞牛影视服务异常，请稍后重试")
		}
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址无法访问")
	}
}

func fnosTestConnectError(err error) *domain.AppError {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址连接超时，请检查网络与服务是否在线")
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址无法连接，请检查地址和端口是否正确（默认 8005）")
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "cannot resolve"), strings.Contains(msg, "lookup"):
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址无法解析，请检查主机名或 IP")
	case strings.Contains(msg, "context deadline exceeded"), strings.Contains(msg, "timeout"):
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址连接超时，请检查网络与服务是否在线")
	default:
		return domain.Errorf(domain.CodeDriverError, "飞牛影视地址无法连接，请检查地址是否正确")
	}
}

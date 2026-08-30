package embyproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"litepan/internal/settings"
	"litepan/internal/store"
)

func TestUpdateRequestAcceptsNumericPort(t *testing.T) {
	var in UpdateRequest
	if err := json.Unmarshal([]byte(`{"proxy_port":18097}`), &in); err != nil {
		t.Fatal(err)
	}
	if in.Port.String() != "18097" {
		t.Fatalf("端口=%q", in.Port.String())
	}
}

func testEmbyProxyService(t *testing.T, embyURL string) *Service {
	t.Helper()
	ctx := context.Background()
	db, err := store.Open(ctx, store.Options{Memory: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	repos := store.New(db)
	settingsSvc, err := settings.New(ctx, repos.Configs)
	if err != nil {
		t.Fatal(err)
	}
	if err := settingsSvc.Update(ctx, map[string]string{
		settings.KeyEmbyEnabled: "true",
		settings.KeyEmbyProxyInstances: fmt.Sprintf(
			`[{"id":"default","name":"Emby","emby_url":%q,"api_key":"test-key","proxy_port":"8097"}]`,
			embyURL,
		),
	}); err != nil {
		t.Fatal(err)
	}
	return New(Options{Settings: settingsSvc})
}

func TestListLibrariesAndRefreshSpecificLibrary(t *testing.T) {
	var gotSelectable bool
	var gotRefreshPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/Library/SelectableMediaFolders"):
			gotSelectable = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"Id":"lib-1","Name":"电影"},{"Id":"lib-2","Name":"剧集"}]`)
		case strings.HasSuffix(r.URL.Path, "/Items/lib-2/Refresh"):
			gotRefreshPath = r.URL.RequestURI()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	svc := testEmbyProxyService(t, upstream.URL)
	libraries, err := svc.ListLibraries(context.Background())
	if err != nil {
		t.Fatalf("ListLibraries 返回错误: %v", err)
	}
	if !gotSelectable {
		t.Fatalf("未请求 SelectableMediaFolders")
	}
	if len(libraries) != 2 || libraries[1].ID != "lib-2" {
		t.Fatalf("媒体库列表异常: %#v", libraries)
	}
	result, err := svc.RefreshLibrary(context.Background(), RefreshRequest{Mode: "library", LibraryID: "lib-2"})
	if err != nil {
		t.Fatalf("RefreshLibrary 返回错误: %v", err)
	}
	if !strings.Contains(gotRefreshPath, "/Items/lib-2/Refresh?") {
		t.Fatalf("指定库刷新路径异常: %q", gotRefreshPath)
	}
	if result.Mode != "library" || result.LibraryID != "lib-2" || result.LibraryName != "剧集" {
		t.Fatalf("刷新结果异常: %#v", result)
	}
}

func TestReplaceConfigsKeepsFirstAndMaskedSecret(t *testing.T) {
	svc := testEmbyProxyService(t, "http://emby.test:8096")
	state, err := svc.Replace(context.Background(), false, []UpdateRequest{
		{Name: "主 Emby", EmbyURL: "http://primary.test:8096", APIKey: "primary-secret"},
		{Name: "备用 Emby", EmbyURL: "http://backup.test:8096", APIKey: "backup-secret"},
	})
	if err != nil {
		t.Fatalf("保存多条 Emby 配置: %v", err)
	}
	configs := state.Items
	if state.Enabled || len(configs) != 2 || configs[0].ID == "" || configs[1].ID == "" {
		t.Fatalf("配置状态异常: %#v", state)
	}
	if configs[0].APIKey == "primary-secret" || !strings.Contains(configs[0].APIKey, "****") {
		t.Fatalf("API Key 未脱敏: %q", configs[0].APIKey)
	}

	configs[0].Name = "家庭 Emby"
	updated, err := svc.Replace(context.Background(), false, []UpdateRequest{
		{ID: configs[0].ID, Name: configs[0].Name, EmbyURL: configs[0].EmbyURL, APIKey: configs[0].APIKey},
		{ID: configs[1].ID, Name: configs[1].Name, EmbyURL: configs[1].EmbyURL, APIKey: configs[1].APIKey},
	})
	if err != nil {
		t.Fatalf("使用脱敏 Key 更新配置: %v", err)
	}
	raw, err := svc.resolveConfig(updated.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if raw.APIKey != "primary-secret" || raw.Name != "家庭 Emby" {
		t.Fatalf("更新后配置=%#v", raw)
	}
}

func TestRefreshWithoutConfigIDUsesFirst(t *testing.T) {
	var firstCalls, secondCalls atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer primary.Close()
	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer secondary.Close()

	svc := testEmbyProxyService(t, primary.URL)
	state, err := svc.Replace(context.Background(), false, []UpdateRequest{
		{Name: "Emby A", EmbyURL: primary.URL, APIKey: "key-a"},
		{Name: "Emby B", EmbyURL: secondary.URL, APIKey: "key-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.RefreshLibrary(context.Background(), RefreshRequest{Mode: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ConfigID != state.Items[0].ID || result.ConfigName != "Emby A" {
		t.Fatalf("旧自动联动未使用第一条配置: %#v", result)
	}
	if firstCalls.Load() == 0 || secondCalls.Load() != 0 {
		t.Fatalf("请求分发错误: first=%d second=%d", firstCalls.Load(), secondCalls.Load())
	}
}

func TestReplaceConfigsRejectsDuplicateEnabledPort(t *testing.T) {
	svc := testEmbyProxyService(t, "http://emby.test:8096")
	_, err := svc.Replace(context.Background(), true, []UpdateRequest{
		{Name: "Emby A", EmbyURL: "http://a.test:8096", APIKey: "a", Port: "18097"},
		{Name: "Emby B", EmbyURL: "http://b.test:8096", APIKey: "b", Port: "18097"},
	})
	if err == nil || !strings.Contains(err.Error(), "同一个端口") {
		t.Fatalf("重复端口错误=%v", err)
	}
}

func TestReplaceConfigsRejectsMissingPortWhenEnabled(t *testing.T) {
	svc := testEmbyProxyService(t, "http://emby.test:8096")
	_, err := svc.Replace(context.Background(), true, []UpdateRequest{
		{Name: "Emby A", EmbyURL: "http://a.test:8096", APIKey: "a"},
	})
	if err == nil || !strings.Contains(err.Error(), "所有配置填写反代端口") {
		t.Fatalf("缺少端口错误=%v", err)
	}
}

func TestIsExpectedClientDisconnect(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if !isExpectedClientDisconnect(canceled, context.Canceled) {
		t.Fatal("context canceled 应识别为客户端取消")
	}
	for _, err := range []error{
		fmt.Errorf("write tcp: broken pipe"),
		fmt.Errorf("readfrom tcp: connection reset by peer"),
		fmt.Errorf("use of closed network connection"),
	} {
		if !isExpectedClientDisconnect(context.Background(), err) {
			t.Fatalf("%q 应识别为客户端取消", err)
		}
	}
	if isExpectedClientDisconnect(context.Background(), fmt.Errorf("upstream returned 401")) {
		t.Fatal("真实上游错误不应被当成客户端取消")
	}
}

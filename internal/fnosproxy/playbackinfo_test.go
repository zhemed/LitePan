package fnosproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"litepan/internal/proxybase"
)

func TestUpdateRequestAcceptsNumericPort(t *testing.T) {
	var in UpdateRequest
	if err := json.Unmarshal([]byte(`{"proxy_port":18997}`), &in); err != nil {
		t.Fatal(err)
	}
	if in.Port.String() != "18997" {
		t.Fatalf("端口=%q", in.Port.String())
	}
}

func TestEmbyClientNameRecognizesMediaBrowserPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Items/demo/PlaybackInfo", nil)
	req.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Infuse", Device="Mac", Token="secret"`)
	if got := proxybase.EmbyClientName(req); got != "Infuse" {
		t.Fatalf("客户端名=%q，期望 Infuse", got)
	}
}

func TestNormalizeEmbyMediaStreams(t *testing.T) {
	tests := []struct {
		name        string
		stream      map[string]any
		wantChanged bool
	}{
		{
			name:        "缺 Title 与 DisplayLanguage",
			stream:      map[string]any{"Type": "Audio", "Language": "chi", "DisplayTitle": "[Mandarin]"},
			wantChanged: true,
		},
		{
			name:        "字段显式为 null",
			stream:      map[string]any{"Type": "Video", "Language": nil, "DisplayLanguage": nil, "Title": nil, "DisplayTitle": nil},
			wantChanged: true,
		},
		{
			name:        "字段齐全无需修改",
			stream:      map[string]any{"Type": "Video", "Language": "", "DisplayLanguage": "", "Title": "", "DisplayTitle": "4K HDR"},
			wantChanged: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ms := map[string]any{"MediaStreams": []any{tc.stream}}
			changed := normalizeEmbyMediaStreams(ms)
			if changed != tc.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, tc.wantChanged)
			}
			for _, field := range embyMediaStreamNonNullFields {
				v, ok := tc.stream[field]
				if !ok {
					t.Errorf("字段 %q 补齐后仍缺失", field)
					continue
				}
				if _, isStr := v.(string); !isStr {
					t.Errorf("字段 %q 补齐后应为 string，实际 %T", field, v)
				}
			}
		})
	}
}

func TestNormalizeEmbyMediaStreams_JSONParsable(t *testing.T) {
	ms := map[string]any{
		"MediaStreams": []any{
			map[string]any{"Type": "Video", "Codec": "hevc"},
			map[string]any{"Type": "Audio", "Language": "chi", "DisplayTitle": "DTS"},
		},
	}
	normalizeEmbyMediaStreams(ms)

	raw, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded struct {
		MediaStreams []map[string]json.RawMessage `json:"MediaStreams"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for i, stream := range decoded.MediaStreams {
		for _, field := range embyMediaStreamNonNullFields {
			v, ok := stream[field]
			if !ok || string(v) == "null" {
				t.Errorf("stream[%d] 字段 %q 缺失或为 null: ok=%v val=%s", i, field, ok, v)
			}
		}
	}
}

func TestNormalizeEmbyMediaStreams_NoStreams(t *testing.T) {
	for _, ms := range []map[string]any{
		nil,
		{},
		{"MediaStreams": "not-an-array"},
		{"MediaStreams": []any{}},
	} {
		if normalizeEmbyMediaStreams(ms) {
			t.Errorf("空/非法 MediaStreams 不应报告修改: %v", ms)
		}
	}
}

func TestProxyRequestRewritesLocation(t *testing.T) {
	var upstreamURL string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" || r.URL.RawQuery != "next=%2Fhome" {
			t.Errorf("上游请求地址 = %q?%s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Header.Get("X-Forwarded-Host") == "" || r.Header.Get("X-Forwarded-Proto") != "http" {
			t.Errorf("缺少标准转发头: host=%q proto=%q", r.Header.Get("X-Forwarded-Host"), r.Header.Get("X-Forwarded-Proto"))
		}
		if got := r.Header.Get("Authorization"); got != `MediaBrowser Token="test-token"` {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Location", upstreamURL+"/home")
		w.WriteHeader(http.StatusFound)
	}))
	defer upstream.Close()
	upstreamURL = upstream.URL

	service := New(Options{})
	var cfg Config
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service.proxyRequest(w, r, cfg, strings.TrimPrefix(r.URL.Path, "/"))
	}))
	defer proxyServer.Close()
	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatalf("解析反代地址失败: %v", err)
	}
	cfg = Config{FnosURL: upstream.URL, Port: proxyURL.Port()}

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodGet, proxyServer.URL+"/login?next=%2Fhome", nil)
	if err != nil {
		t.Fatalf("创建反代请求失败: %v", err)
	}
	req.Header.Set("Authorization", `MediaBrowser Token="test-token"`)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求反代失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("状态码 = %d，期望 %d", resp.StatusCode, http.StatusFound)
	}
	if got, want := resp.Header.Get("Location"), proxyServer.URL+"/home"; got != want {
		t.Fatalf("Location = %q，期望 %q", got, want)
	}
}

func TestNormalizeClientKeywords(t *testing.T) {
	if got, want := proxybase.NormalizeClientKeywords(" Infuse；VidHub;infuse;; "), "Infuse;VidHub"; got != want {
		t.Fatalf("规范化结果=%q，期望 %q", got, want)
	}
}

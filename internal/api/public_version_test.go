package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"litepan/internal/adminauth"
	"litepan/internal/store"
)

// systemConfigResp 对应 writeOK 的 {success, data} 封装。
type systemConfigResp struct {
	Success bool `json:"success"`
	Data    struct {
		Version                string `json:"version"`
		IndexAccountSwitchMode string `json:"index_account_switch_mode"`
		CompactHomeEnabled     bool   `json:"compact_home_enabled"`
		HeaderEffectsEnabled   bool   `json:"header_effects_enabled"`
	} `json:"data"`
}

// TestPublicSystemConfigExposesInjectedVersion 锁定版本号的单一来源约定：
// /public/system-config 必须回传装配期注入的 buildinfo.Version，前端据此运行期显示，
// 不再自带任何版本字面量（见 web/src/stores/appInfo.ts）。
func TestPublicSystemConfigExposesInjectedVersion(t *testing.T) {
	const wantVersion = "v9.9.9-test"

	ctx := context.Background()
	db, err := store.Open(ctx, store.Options{Memory: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st := store.New(db)

	h := &Handler{
		version:   wantVersion,
		adminAuth: adminauth.New(st.Configs, []byte("test-secret-key-min-16b"), nil),
	}

	recorder := httptest.NewRecorder()
	h.publicSystemConfig(recorder, httptest.NewRequest(http.MethodGet, "/api/public/system-config", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}

	var got systemConfigResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, recorder.Body.String())
	}

	t.Run("回传注入的版本号", func(t *testing.T) {
		if !got.Success {
			t.Fatalf("success = false (body=%s)", recorder.Body.String())
		}
		if got.Data.Version != wantVersion {
			t.Fatalf("version = %q, want %q", got.Data.Version, wantVersion)
		}
	})

	t.Run("既有三个字段未丢失", func(t *testing.T) {
		// 用 map 复核键存在性，避免零值（"" / false）被误判为缺失。
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		var data map[string]json.RawMessage
		if err := json.Unmarshal(raw["data"], &data); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		for _, key := range []string{"version", "index_account_switch_mode", "compact_home_enabled", "header_effects_enabled"} {
			if _, ok := data[key]; !ok {
				t.Errorf("响应缺少字段 %q", key)
			}
		}
	})
}

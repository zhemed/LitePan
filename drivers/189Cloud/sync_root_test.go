package cloud189

import (
	"context"
	"net/http"
	"testing"

	"litepan/internal/domain"
)

// 同步盘（真实 ID 为 0）用独立标识在内部区分，出网前必须还原为 "0"；
// 普通目录 ID 与根别名行为不变。
func TestAPIParentIDMapsSyncRootWithoutChangingOthers(t *testing.T) {
	d := &Driver{}
	cases := map[string]string{
		syncRootID: "0",
		"12345":    "12345",
		"0":        d.rootID(), // 根别名仍归一到 rootID
		"":         d.rootID(),
		"/":        d.rootID(),
		"root":     d.rootID(),
	}
	for in, want := range cases {
		if got := d.apiParentID(in); got != want {
			t.Fatalf("apiParentID(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// 家庭空间（family）不走同步盘分支：syncRootID 在 family 下按普通 ID 透传。
func TestAPIParentIDKeepsSyncRootForFamily(t *testing.T) {
	d := &Driver{}
	d.add.SpaceType = "family"
	if got := d.apiParentID(syncRootID); got != syncRootID {
		t.Fatalf("family 下 apiParentID(%q) = %q，应原样透传", syncRootID, got)
	}
}

// 同步盘本身算根：不允许重命名/删除它。
func TestContainsRootIncludesSyncRoot(t *testing.T) {
	if !containsRoot([]string{syncRootID}, "-11") {
		t.Fatal("同步盘应被判定为根")
	}
	if containsRoot([]string{"12345"}, "-11") {
		t.Fatal("普通目录不应被判定为根")
	}
	if !containsRoot([]string{"12345", "0"}, "-11") {
		t.Fatal("别名 0 应被判定为根")
	}
}

// GetFileInfo 对同步盘直接返回目录项（不发网络请求）。
func TestGetFileInfoReturnsSyncRootWithoutNetwork(t *testing.T) {
	d := &Driver{}
	item, err := d.GetFileInfo(context.Background(), syncRootID)
	if err != nil {
		t.Fatalf("GetFileInfo(sync:0) 不应报错：%v", err)
	}
	if item == nil || item.ID != syncRootID || item.Name != "同步盘" || !item.IsDir {
		t.Fatalf("同步盘目录项 = %+v", item)
	}
	if item.IDKind != domain.IDStable {
		t.Fatalf("同步盘 ID 类型 = %v，期望 IDStable", item.IDKind)
	}
}

// 400 + 失效 payload 也算会话失效（上游会用 400 表示 invalidsessionkey）；
// 403/429 等仍不作失效处理。
func TestIs189AuthExpiredResponse(t *testing.T) {
	expiredBody := []byte(`{"error":"invalidsessionkey"}`)
	cases := []struct {
		status int
		body   []byte
		want   bool
	}{
		{http.StatusUnauthorized, []byte("{}"), true},
		{http.StatusOK, expiredBody, true},
		{http.StatusBadRequest, expiredBody, true},
		{http.StatusBadRequest, []byte(`{"error":"bad request"}`), false},
		{http.StatusForbidden, expiredBody, false},
		{http.StatusTooManyRequests, expiredBody, false},
		{http.StatusOK, []byte("{}"), false},
	}
	for _, tc := range cases {
		if got := is189AuthExpiredResponse(tc.status, tc.body); got != tc.want {
			t.Fatalf("status=%d body=%s → %v，期望 %v", tc.status, tc.body, got, tc.want)
		}
	}
}

package pan115open

import (
	"strconv"
	"testing"
)

// 明文 pickcode 缓存不得无上限增长：达到上限后新增键会整体清空重来。
func TestRememberPickCodeClearsCacheAtCapacity(t *testing.T) {
	d := &Driver{}
	for i := 0; i < maxPickCodeCacheEntries; i++ {
		d.rememberPickCode(fileEntry{Fid: "f" + strconv.Itoa(i), PickCode: "pc" + strconv.Itoa(i)})
	}
	if got := len(d.pickBy); got != maxPickCodeCacheEntries {
		t.Fatalf("缓存条目 = %d，期望 %d", got, maxPickCodeCacheEntries)
	}

	d.rememberPickCode(fileEntry{Fid: "新键", PickCode: "新凭据"})
	if got := len(d.pickBy); got != 1 {
		t.Fatalf("超限后应整体清空再写入新键，实际条目 = %d", got)
	}
	if got := d.cachedPickCode("新键"); got != "新凭据" {
		t.Fatalf("新键凭据 = %q，期望 新凭据", got)
	}
}

// 已存在的键在满容量时只覆盖取值，不触发清空（否则冷热交替会反复丢整个缓存）。
func TestRememberPickCodeKeepsCacheWhenUpdatingExistingKey(t *testing.T) {
	d := &Driver{}
	for i := 0; i < maxPickCodeCacheEntries; i++ {
		d.rememberPickCode(fileEntry{Fid: "f" + strconv.Itoa(i), PickCode: "pc" + strconv.Itoa(i)})
	}
	d.rememberPickCode(fileEntry{Fid: "f0", PickCode: "新凭据"})
	if got := len(d.pickBy); got != maxPickCodeCacheEntries {
		t.Fatalf("更新已有键不应清空缓存，实际条目 = %d", got)
	}
	if got := d.cachedPickCode("f0"); got != "新凭据" {
		t.Fatalf("已有键应被覆盖为 新凭据，实际 %q", got)
	}
}

// 缺少文件 ID 或 pickcode 的条目不入缓存，避免把空键写进去。
func TestRememberPickCodeSkipsIncompleteEntries(t *testing.T) {
	d := &Driver{}
	d.rememberPickCode(fileEntry{PickCode: "只有凭据"})
	d.rememberPickCode(fileEntry{Fid: "f1"})
	d.rememberPickCode(fileEntry{Fid: "  ", PickCode: "  "})
	if got := len(d.pickBy); got != 0 {
		t.Fatalf("不完整条目不应进缓存，实际条目 = %d", got)
	}
}

// 删除成功后必须清掉对应凭据；删除失败（未调用本函数）时缓存保留，重试仍可用。
func TestForgetPickCodesRemovesOnlyDeletedIDs(t *testing.T) {
	d := &Driver{}
	d.rememberPickCode(fileEntry{Fid: "f1", PickCode: "pc1"})
	d.rememberPickCode(fileEntry{Fid: "f2", PickCode: "pc2"})

	d.forgetPickCodes([]string{"f1", "不存在的键"})
	if got := d.cachedPickCode("f1"); got != "" {
		t.Fatalf("已删除文件的凭据应被清掉，实际 %q", got)
	}
	if got := d.cachedPickCode("f2"); got != "pc2" {
		t.Fatalf("未删除文件的凭据应保留，实际 %q", got)
	}
	if got := len(d.pickBy); got != 1 {
		t.Fatalf("缓存条目 = %d，期望 1", got)
	}

	d.forgetPickCodes(nil)
	if got := len(d.pickBy); got != 1 {
		t.Fatalf("空列表不应改动缓存，实际条目 = %d", got)
	}
}

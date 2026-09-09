package upload

import (
	"sort"

	"context"
	"testing"

	"litepan/internal/domain"
)

type targetDirTestFiles struct {
	items       map[string][]domain.FileItem
	listCalls   map[string]int
	createCalls []string
}

func (f *targetDirTestFiles) List(_ context.Context, _ int64, parentID string, _ bool) ([]domain.FileItem, error) {
	if f.listCalls == nil {
		f.listCalls = make(map[string]int)
	}
	f.listCalls[parentID]++
	return append([]domain.FileItem(nil), f.items[parentID]...), nil
}

func (f *targetDirTestFiles) CreateFolder(_ context.Context, _ int64, parentID, name string) (*domain.FileItem, error) {
	item := domain.FileItem{ID: parentID + "-" + name, Name: name, IsDir: true}
	f.createCalls = append(f.createCalls, parentID+"/"+name)
	f.items[parentID] = append(f.items[parentID], item)
	return &item, nil
}

func TestFindByClientTaskIDRestoredAndDeleted(t *testing.T) {
	repo := &failingUploadTaskRepo{
		rows: map[string]*domain.UploadTaskRecord{
			"task-1": {
				TaskID:         "task-1",
				ClientTaskID:   "client:1",
				AccountID:      1,
				AccountName:    "目标盘",
				DriverType:     "mock",
				FileName:       "demo.mkv",
				TargetPath:     "0",
				Status:         StatusSuccess,
				QueueOrder:     1,
				ConflictPolicy: "overwrite",
			},
		},
	}
	m := NewManager(Options{Repo: repo, DataDir: t.TempDir()})

	got := m.FindByClientTaskID("client:1")
	if got == nil || got.TaskID != "task-1" {
		t.Fatalf("restored task not found by clientTaskID: %#v", got)
	}

	found, err := m.Delete(context.Background(), "task-1", false)
	if err != nil || !found {
		t.Fatalf("delete task failed: found=%v err=%v", found, err)
	}
	if got := m.FindByClientTaskID("client:1"); got != nil {
		t.Fatalf("deleted task should not remain indexed: %#v", got)
	}
}

func TestEnsureUploadTargetDirCachesFullPathAndPrefix(t *testing.T) {
	files := &targetDirTestFiles{
		items: map[string][]domain.FileItem{
			"root": {{ID: "tv", Name: "tv", IsDir: true}},
			"tv":   {{ID: "s1", Name: "season1", IsDir: true}},
		},
	}
	cache := newUploadTargetDirCache()

	id, err := ensureUploadTargetDir(context.Background(), files, cache, 1, "root", "tv/season1")
	if err != nil || id != "s1" {
		t.Fatalf("resolve season1 failed: id=%q err=%v", id, err)
	}
	if files.listCalls["root"] != 1 || files.listCalls["tv"] != 1 {
		t.Fatalf("unexpected first list calls: %#v", files.listCalls)
	}

	id, err = ensureUploadTargetDir(context.Background(), files, cache, 1, "root", "tv/season1")
	if err != nil || id != "s1" {
		t.Fatalf("resolve season1 second time failed: id=%q err=%v", id, err)
	}
	if files.listCalls["root"] != 1 || files.listCalls["tv"] != 1 {
		t.Fatalf("full-path cache did not hit: %#v", files.listCalls)
	}

	id, err = ensureUploadTargetDir(context.Background(), files, cache, 1, "root", "tv/season2")
	if err != nil {
		t.Fatalf("resolve season2 failed: %v", err)
	}
	if id == "" {
		t.Fatal("season2 folder id should not be empty")
	}
	if files.listCalls["root"] != 1 {
		t.Fatalf("prefix cache should skip root relist: %#v", files.listCalls)
	}
	if files.listCalls["tv"] != 2 {
		t.Fatalf("season2 should only list tv once more: %#v", files.listCalls)
	}
	if len(files.createCalls) != 1 || files.createCalls[0] != "tv/season2" {
		t.Fatalf("unexpected create calls: %#v", files.createCalls)
	}
}

// 预解析：乱序+重复的 rel_dir 应去重并按字典序（父前缀先行）解析，
// 解析后同前缀零新增 List（缓存命中）。
func TestWarmTargetDirsDedupesAndSortsPrefixes(t *testing.T) {
	files := &targetDirTestFiles{
		items: map[string][]domain.FileItem{
			"root": {{ID: "msg", Name: "msg", IsDir: true}},
			"msg":  {{ID: "attach", Name: "attach", IsDir: true}},
		},
	}
	cache := newUploadTargetDirCache()
	relDirs := []string{
		"msg/attach/h2/2026-08",
		"msg/attach/h1/2026-08",
		"msg/attach/h2/2026-08", // 重复
		"msg/attach/h1/2026-07",
		"msg/resource",
	}
	warmTargetDirs(context.Background(), files, cache, 1, "root", relDirs)

	// fake 生成的目录 ID 为 parentID+"-"+name；List 以解析后的 ID 为父。
	// root 1；msg 2（attach 前缀 + resource 前缀）；attach 2（h1/h2）；
	// attach-h1 2（2026-07 与 2026-08 各一次）；attach-h2 1。
	if files.listCalls["root"] != 1 || files.listCalls["msg"] != 2 || files.listCalls["attach"] != 2 {
		t.Fatalf("prefix dedupe failed: %#v", files.listCalls)
	}
	if files.listCalls["attach-h1"] != 2 || files.listCalls["attach-h2"] != 1 {
		t.Fatalf("hash dir lists wrong: %#v", files.listCalls)
	}
	listsBefore := files.listCalls["root"] + files.listCalls["msg"] + files.listCalls["attach"] + files.listCalls["attach-h1"] + files.listCalls["attach-h2"]

	// 预热后再解析任意目录：全部缓存命中，零新增 List。
	for _, relDir := range relDirs {
		if _, err := ensureUploadTargetDir(context.Background(), files, cache, 1, "root", relDir); err != nil {
			t.Fatalf("resolve after warm failed: %v", err)
		}
	}
	listsAfter := files.listCalls["root"] + files.listCalls["msg"] + files.listCalls["attach"] + files.listCalls["attach-h1"] + files.listCalls["attach-h2"]
	if listsAfter != listsBefore {
		t.Fatalf("cache should absorb post-warm resolves: before=%d after=%d", listsBefore, listsAfter)
	}
}

// 批次预热收集：去重 rel_dir、按 (账号,根目录) 分组、跳过空 RelDir。
func TestCollectBatchWarmDirs(t *testing.T) {
	created := []*taskState{
		{Task: Task{TaskID: "1", AccountID: 1, TargetPath: "0", RelDir: "a/b"}},
		{Task: Task{TaskID: "2", AccountID: 1, TargetPath: "0", RelDir: "a/c"}},
		{Task: Task{TaskID: "3", AccountID: 1, TargetPath: "0", RelDir: "a/b"}}, // 重复
		{Task: Task{TaskID: "4", AccountID: 2, TargetPath: "0", RelDir: "x"}},
		{Task: Task{TaskID: "5", AccountID: 1, TargetPath: "0", RelDir: ""}}, // 根目录文件跳过
	}
	got := collectBatchWarmDirs(created)
	if len(got) != 2 {
		t.Fatalf("groups = %d，期望 2（按账号分组）", len(got))
	}
	acc1 := got[batchWarmKey{accountID: 1, rootID: "0"}]
	// collectBatchWarmDirs 输出经 map 遍历，顺序随机（消费方 warmTargetDirs 会
	// 排序）——断言集合内容而非顺序（修复潜在 flaky）。
	sorted := append([]string(nil), acc1...)
	sort.Strings(sorted)
	if len(acc1) != 2 || sorted[0] != "a/b" || sorted[1] != "a/c" {
		t.Fatalf("account1 dirs = %#v", acc1)
	}
	acc2 := got[batchWarmKey{accountID: 2, rootID: "0"}]
	if len(acc2) != 1 || acc2[0] != "x" {
		t.Fatalf("account2 dirs = %#v", acc2)
	}
}

// 0.0.22：ResolveUploadTargetDir —— 共享缓存跨请求命中（零 List）与
// createdPrefixes（仅本次新建前缀）语义。
func TestResolveUploadTargetDirSharedCacheAndCreatedPrefixes(t *testing.T) {
	files := &targetDirTestFiles{
		items: map[string][]domain.FileItem{
			"root": {{ID: "old", Name: "old", IsDir: true}},
		},
	}
	cache := newUploadTargetDirCache()
	resolve := func(relDir string) (string, map[string]bool, error) {
		return ensureUploadTargetDirDetailed(context.Background(), files, cache, 1, "root", relDir)
	}
	totalLists := func() int {
		n := 0
		for _, c := range files.listCalls {
			n += c
		}
		return n
	}
	baseline := totalLists()

	// 第一次解析 msg/attach：msg 已存在、attach 新建
	id, created, err := resolve("old/attach/2026-08")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if created["old"] {
		t.Fatalf("existing prefix must not be marked created: %v", created)
	}
	if !created["old/attach"] || !created["old/attach/2026-08"] {
		t.Fatalf("created prefixes missing: %v", created)
	}

	// 第二次解析同前缀 deeper 路径：前缀全部命中缓存，root 零新 List
	id2, created2, err := resolve("old/attach/2026-09")
	if err != nil {
		t.Fatalf("resolve2: %v", err)
	}
	if created2["old"] || created2["old/attach"] {
		t.Fatalf("cache-hit prefixes must not be created: %v", created2)
	}
	if !created2["old/attach/2026-09"] {
		t.Fatalf("new leaf prefix should be created: %v", created2)
	}
	if got := totalLists() - baseline; got != 4 {
		t.Fatalf("expected 4 Lists total (3 first resolve + 1 new leaf on second), got %d", got)
	}
	if id == "" || id2 == "" || id == id2 {
		t.Fatalf("unexpected ids %q %q", id, id2)
	}
}

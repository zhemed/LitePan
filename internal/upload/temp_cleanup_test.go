package upload

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCleanupOrphanTempFilesKeepsActiveAndFreshPaths 锁定临时目录清理的判定：
// 只有「过期且不被任何在途任务占用」的临时文件才会被删；
// 在途任务正在用的（localPath）与未过期的都必须保留。
//
// 背景：2026-09-15 删除 TempRegistry 链时，行为等价性就是靠这个判定论证并锁定的
// —— 该链的写入者只有已删除的 FUSE 写路径，Track 零调用 ⇒ 其 map 恒为空，
// 因此清理逻辑只依赖 m.tasks 的 localPath。
func TestCleanupOrphanTempFilesKeepsActiveAndFreshPaths(t *testing.T) {
	m := &Manager{
		dataDir: t.TempDir(),
		tasks:   map[string]*taskState{},
	}
	dir := m.TempDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir temp dir: %v", err)
	}

	write := func(name string, age time.Duration) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		old := time.Now().Add(-age)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
		return p
	}

	stale := write("stale.bin.part", 2*time.Hour)
	fresh := write("fresh.bin.part", time.Minute)
	active := write("active.bin.part", 2*time.Hour)
	m.tasks["t-active"] = &taskState{localPath: active}

	deleted, err := m.CleanupOrphanTempFiles(time.Hour)
	if err != nil {
		t.Fatalf("CleanupOrphanTempFiles: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("应只删除 1 个过期且无主的临时文件，实际 %d", deleted)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("过期且无主的临时文件应被删除: err=%v", err)
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatalf("在途任务占用的临时文件必须保留: err=%v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("未过期的临时文件必须保留: err=%v", err)
	}

	// maxAge=0 表示不限龄：除在途占用外都应清掉。
	deleted, err = m.CleanupOrphanTempFiles(0)
	if err != nil {
		t.Fatalf("CleanupOrphanTempFiles(0): %v", err)
	}
	if deleted != 1 {
		t.Fatalf("第二个过期文件应被清理，实际删除 %d", deleted)
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatalf("在途任务占用的临时文件在 maxAge=0 时也必须保留: err=%v", err)
	}
}

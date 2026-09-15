package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"litepan/internal/domain"
	"litepan/internal/store"
)

func TestSnapshotToCreatesConsistentDatabase(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db, err := store.Open(ctx, store.Options{Path: filepath.Join(root, "source.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	st := store.New(db)
	if err := st.Configs.Set(ctx, "snapshot_test", "ok"); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "snapshot.db")
	if err := db.SnapshotTo(ctx, destination); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Open(ctx, store.Options{Path: destination})
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if err := snapshot.IntegrityCheck(ctx); err != nil {
		t.Fatal(err)
	}
	value, ok, err := store.New(snapshot).Configs.Get(ctx, "snapshot_test")
	if err != nil || !ok || value != "ok" {
		t.Fatalf("snapshot value=%q ok=%v err=%v", value, ok, err)
	}
}

// TestBackupCountsAfterMigrations 锁定 BackupCounts 只能统计**迁移后真实存在**的表。
// 回归背景：该函数曾把已删功能的表（fuse_mounts）写死在 tables 列表里，
// 表被迁移 DROP 后创建备份会直接 `no such table` —— 这个用例就是那条路径的哨兵。
func TestBackupCountsAfterMigrations(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(ctx, store.Options{Memory: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	counts, err := db.BackupCounts(ctx)
	if err != nil {
		t.Fatalf("BackupCounts 在迁移后的库上必须可用（创建备份的关键路径）: %v", err)
	}
	if counts.Accounts != 0 || counts.Tasks != 0 {
		t.Fatalf("空库计数应为 0/0，实际 accounts=%d tasks=%d", counts.Accounts, counts.Tasks)
	}

	// 写入一条自动化规则，确认 Tasks 确实来自仍存在的表（而非恒为 0 的假实现）。
	st := store.New(db)
	if _, err := st.AutomationRules.Create(ctx, &domain.AutomationRule{
		Name:        "counts-probe",
		TriggerType: domain.AutomationTriggerInterval,
		Status:      domain.AutomationStatusRunning,
	}); err != nil {
		t.Fatalf("create automation rule: %v", err)
	}
	counts, err = db.BackupCounts(ctx)
	if err != nil {
		t.Fatalf("BackupCounts after insert: %v", err)
	}
	if counts.Tasks != 1 {
		t.Fatalf("Tasks 应统计 automation_rules 的 1 行，实际 %d", counts.Tasks)
	}
}

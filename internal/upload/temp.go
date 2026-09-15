package upload

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

const (
	TempCleanupInterval = time.Hour
	TempMaxAge          = 24 * time.Hour
)

func TempDir(dataDir string) string {
	return filepath.Join(dataDir, "upload_tasks")
}

func CleanupTempDir(dir string, active map[string]struct{}, maxAge time.Duration) (int, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	deleted := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Clean(filepath.Join(dir, e.Name()))
		if active != nil {
			if _, ok := active[path]; ok {
				continue
			}
		}
		if maxAge > 0 {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if now.Sub(info.ModTime()) < maxAge {
				continue
			}
		}
		if err := os.Remove(path); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (m *Manager) activeTempPaths() map[string]struct{} {
	active := make(map[string]struct{})
	m.mu.Lock()
	for _, st := range m.tasks {
		if st.localPath == "" {
			continue
		}
		active[filepath.Clean(st.localPath)] = struct{}{}
	}
	m.mu.Unlock()
	return active
}

func (m *Manager) CleanupOrphanTempFiles(maxAge time.Duration) (int, error) {
	active := m.activeTempPaths()
	return CleanupTempDir(m.TempDir(), active, maxAge)
}

func (m *Manager) StartTempCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(TempCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := m.CleanupOrphanTempFiles(TempMaxAge)
				if err != nil {
					m.log.Warn("upload temp cleanup failed", "err", err)
					continue
				}
				if n > 0 {
					m.log.Info("upload temp cleanup done", "deleted", n)
				}
			}
		}
	}()
}

func (m *Manager) initTempCleanup() {
	if n, err := m.CleanupOrphanTempFiles(0); err != nil {
		m.log.Warn("upload temp startup cleanup failed", "err", err)
	} else if n > 0 {
		m.log.Info("upload temp startup cleanup done", "deleted", n)
	}
}

package upload

import (
	"context"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"litepan/internal/domain"
)

// 10 分钟：大批量上传（数百文件/目录）运行期远超旧值 30s，过期会导致
// 同一目录被反复重新 List（每次都过账号间隔门，加剧吞吐塌陷）。
const uploadTargetCacheTTL = 10 * time.Minute

type uploadTargetFiles interface {
	List(ctx context.Context, accountID int64, parentID string, forceRefresh bool) ([]domain.FileItem, error)
	CreateFolder(ctx context.Context, accountID int64, parentID, name string) (*domain.FileItem, error)
}

type uploadTargetCacheKey struct {
	accountID int64
	rootID    string
	relDir    string
}

type uploadTargetCacheEntry struct {
	folderID  string
	expiresAt time.Time
}

type uploadTargetDirCache struct {
	mu      sync.Mutex
	entries map[uploadTargetCacheKey]uploadTargetCacheEntry
}

func newUploadTargetDirCache() *uploadTargetDirCache {
	return &uploadTargetDirCache{entries: make(map[uploadTargetCacheKey]uploadTargetCacheEntry)}
}

func (c *uploadTargetDirCache) get(accountID int64, rootID, relDir string, now time.Time) (string, bool) {
	if c == nil || relDir == "" {
		return "", false
	}
	key := uploadTargetCacheKey{accountID: accountID, rootID: rootID, relDir: relDir}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if now.After(entry.expiresAt) {
		delete(c.entries, key)
		return "", false
	}
	return entry.folderID, true
}

func (c *uploadTargetDirCache) put(accountID int64, rootID, relDir, folderID string, now time.Time) {
	if c == nil || relDir == "" || folderID == "" {
		return
	}
	key := uploadTargetCacheKey{accountID: accountID, rootID: rootID, relDir: relDir}
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[uploadTargetCacheKey]uploadTargetCacheEntry)
	}
	c.entries[key] = uploadTargetCacheEntry{
		folderID:  folderID,
		expiresAt: now.Add(uploadTargetCacheTTL),
	}
	c.mu.Unlock()
}

func ensureUploadTargetDir(ctx context.Context, files uploadTargetFiles, cache *uploadTargetDirCache, accountID int64, rootID, relDir string) (string, error) {
	if files == nil {
		return "", domain.Errorf(domain.CodeInternal, "文件服务未就绪")
	}
	relDir = strings.Trim(relDir, "/")
	if relDir == "" {
		return rootID, nil
	}
	now := time.Now()
	cur := rootID
	parts := strings.Split(relDir, "/")
	prefixParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		prefixParts = append(prefixParts, part)
		prefix := strings.Join(prefixParts, "/")
		if cachedID, ok := cache.get(accountID, rootID, prefix, now); ok {
			cur = cachedID
			continue
		}
		items, err := files.List(ctx, accountID, cur, false)
		if err != nil {
			return "", err
		}
		next := ""
		for _, item := range items {
			if item.IsDir && item.Name == part {
				next = item.ID
				break
			}
		}
		if next == "" {
			created, err := files.CreateFolder(ctx, accountID, cur, part)
			if err != nil {
				return "", err
			}
			next = created.ID
		}
		cur = next
		cache.put(accountID, rootID, prefix, cur, now)
	}
	return cur, nil
}

// warmTargetDirs 按字典序预解析目录（父前缀天然先行），全部命中缓存后
// 上传 worker 不再边传边解析。目录已在缓存中的前缀零成本跳过。
func warmTargetDirs(ctx context.Context, files uploadTargetFiles, cache *uploadTargetDirCache, accountID int64, rootID string, relDirs []string) {
	sorted := append([]string(nil), relDirs...)
	sort.Strings(sorted)
	for _, relDir := range sorted {
		if ctx.Err() != nil {
			return
		}
		if _, err := ensureUploadTargetDir(ctx, files, cache, accountID, rootID, relDir); err != nil {
			// 预解析失败不致命：上传 worker 走原有即时解析兜底。
			return
		}
	}
}

func joinUploadDisplayPath(base, relDir string) string {	base = "/" + strings.Trim(strings.TrimSpace(base), "/")
	if base == "/" {
		base = ""
	}
	relDir = strings.Trim(strings.TrimSpace(relDir), "/")
	if relDir == "" {
		if base == "" {
			return "/"
		}
		return base
	}
	joined := path.Join(base, relDir)
	if joined == "." || joined == "" {
		return "/"
	}
	if !strings.HasPrefix(joined, "/") {
		return "/" + joined
	}
	return joined
}

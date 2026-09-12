package upload

import (
	"context"
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

// ResolveUploadTargetDir 共享缓存目录解析：预解析（warmTargetDirs）与批次
// 创建 walk（api.ensureLocalUploadTargetDir）统一走 manager 持有的同一缓存
// 实例（TTL 10min，跨请求命中，驱动无关——115 等任何 LocalUploader 同益）。
// 返回 folderID 与本次调用**新建**的前缀集合（键=相对前缀），供调用方维持
// BatchRootOwned 语义（仅本次新建的根计入）。
func (m *Manager) ResolveUploadTargetDir(ctx context.Context, accountID int64, rootID, relDir string) (string, map[string]bool, error) {
	if m == nil || m.files == nil {
		return "", nil, domain.Errorf(domain.CodeInternal, "上传服务未就绪")
	}
	return ensureUploadTargetDirDetailed(ctx, m.files, m.targetDirCache, accountID, rootID, relDir)
}

func ensureUploadTargetDir(ctx context.Context, files uploadTargetFiles, cache *uploadTargetDirCache, accountID int64, rootID, relDir string) (string, error) {
	folderID, _, err := ensureUploadTargetDirDetailed(ctx, files, cache, accountID, rootID, relDir)
	return folderID, err
}

func ensureUploadTargetDirDetailed(ctx context.Context, files uploadTargetFiles, cache *uploadTargetDirCache, accountID int64, rootID, relDir string) (string, map[string]bool, error) {
	if files == nil {
		return "", nil, domain.Errorf(domain.CodeInternal, "文件服务未就绪")
	}
	relDir = strings.Trim(relDir, "/")
	if relDir == "" {
		return rootID, nil, nil
	}
	createdPrefixes := make(map[string]bool)
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
			return "", nil, err
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
				return "", nil, err
			}
			next = created.ID
			createdPrefixes[prefix] = true
		}
		cur = next
		cache.put(accountID, rootID, prefix, cur, now)
	}
	return cur, createdPrefixes, nil
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

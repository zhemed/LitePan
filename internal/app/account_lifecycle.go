package app

import (
	"context"
	"fmt"

	"litepan/internal/favorites"
	"litepan/internal/fusemount"
	"litepan/internal/fusereadcache"
	"litepan/internal/offlinedownload"
	"litepan/internal/quarktv"
	"litepan/internal/upload"
)

type accountLifecycle struct {
	fuse      *fusemount.Service
	readCache *fusereadcache.Service
	favorites *favorites.Service
	offline   *offlinedownload.Service
	uploads   *upload.Manager
	quarktv   *quarktv.Service
}

func (a accountLifecycle) OnAccountDisabled(ctx context.Context, accountID int64) {
	if accountID <= 0 {
		return
	}
}

func (a accountLifecycle) OnAccountEnabled(ctx context.Context, accountID int64) {
	if accountID <= 0 {
		return
	}
}

func (a accountLifecycle) OnAccountDeleted(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}
	if a.fuse != nil {
		if err := a.fuse.OnAccountDeleted(ctx, accountID); err != nil {
			return err
		}
	}
	if a.readCache != nil {
		if err := a.readCache.InvalidateAccount(ctx, accountID); err != nil {
			return fmt.Errorf("清理 FUSE 读缓存失败: %w", err)
		}
	}
	if a.favorites != nil {
		if err := a.favorites.Delete(ctx, accountID); err != nil {
			return fmt.Errorf("清理收藏夹失败: %w", err)
		}
	}
	if a.offline != nil {
		if _, err := a.offline.RemoveTasksByAccount(ctx, accountID); err != nil {
			return fmt.Errorf("清理离线下载任务失败: %w", err)
		}
	}
	if a.uploads != nil {
		if _, err := a.uploads.RemoveTasksByAccount(ctx, accountID); err != nil {
			return fmt.Errorf("清理上传任务失败: %w", err)
		}
	}
	if a.quarktv != nil {
		if err := a.quarktv.DeleteBinding(ctx, accountID); err != nil {
			return fmt.Errorf("清理夸克 TV 绑定失败: %w", err)
		}
	}
	return nil
}

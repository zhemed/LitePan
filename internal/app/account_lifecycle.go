package app

import (
	"context"
	"fmt"

	"litepan/internal/favorites"
	"litepan/internal/upload"
)

type accountLifecycle struct {
	favorites *favorites.Service
	uploads   *upload.Manager
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
	if a.favorites != nil {
		if err := a.favorites.Delete(ctx, accountID); err != nil {
			return fmt.Errorf("清理收藏夹失败: %w", err)
		}
	}
	if a.uploads != nil {
		if _, err := a.uploads.RemoveTasksByAccount(ctx, accountID); err != nil {
			return fmt.Errorf("清理上传任务失败: %w", err)
		}
	}
	return nil
}

package domain

import "time"

const NotificationCategoryCacheScopeWarn = "cache_scope_warn"

type Notification struct {
	ID        int64
	Level     string
	Category  string
	Title     string
	Message   string
	AccountID int64
	RefID     int64
	IsRead    bool
	CreatedAt time.Time
}

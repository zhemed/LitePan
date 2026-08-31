package settings

import "litepan/internal/domain"

// 全局设置：默认值在代码，DB 仅存用户改过的项。

// 设置键。oauth 复用 domain 常量，保证与驱动层读取一致。
const (
	KeyOAuthServerURL              = domain.SettingOAuthServerURL
	KeyCacheEnabled                = "cache_enabled"
	KeyCacheTTL                    = "cache_ttl"
	KeyCacheMaxItems               = "cache_max_items"
	KeyCacheMemoryLimitMB          = "cache_memory_limit_mb"
	KeyCachePersistenceEnabled     = "cache_persistence_enabled"
	KeyCachePersistenceIntervalMin = "cache_persistence_interval_minutes"
	KeyUploadTaskConcurrency       = "upload_task_concurrency"
	KeyFuseReadCacheEnabled        = "fuse_read_cache_enabled"
	KeyFuseReadCacheMaxGB          = "fuse_read_cache_max_gb"
	KeyFuseReadCacheRetentionDays  = "fuse_read_cache_retention_days"
	KeyFuseReadCacheEvictionPolicy = "fuse_read_cache_eviction_policy"
	KeyAuthActiveRefresh           = "auth_active_refresh_enabled"
	KeyAccountShowProfile          = "account_show_profile"
	KeyAccountShowMembership       = "account_show_membership"
	KeyLogLevel                    = "log_level"
	KeyLogRetentionDays            = "log_retention_days"
	KeyLogErrorAckAt               = "log_error_ack_at"
	KeyLocalUploadEnabled          = "local_upload_enabled"
	KeyLocalUploadMappings         = "local_upload_mappings"
)

// Type 决定后台表单控件与校验方式。
type Type string

const (
	TypeString Type = "string"
	TypeInt    Type = "int"
	TypeBool   Type = "bool"
	TypeSelect Type = "select"
)

// Option 是 select 类型的可选项。
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Spec 声明单个全局设置的元数据，驱动后台表单渲染与写入校验。
type Spec struct {
	Key         string
	Type        Type
	Category    string
	Label       string
	Description string
	Default     string // 默认值的规范字符串形式（与 configs 表存储一致）
	Unit        string
	Min, Max    *int     // 仅 TypeInt
	Options     []Option // 仅 TypeSelect
	Sensitive   bool
	Hidden      bool
	// normalize 对字符串值做规范化/兜底（如 OAuth 地址校验），nil 表示不处理。
	normalize func(string) string
}

// Category 是设置分组，用于后台分区展示。
type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func intp(n int) *int { return &n }

// defaultSpecs 是全部全局设置的有序声明。新增全局设置只改这里。
func boolSpec(key, category, label, description, def string) Spec {
	return Spec{Key: key, Type: TypeBool, Category: category, Label: label, Description: description, Default: def}
}

func stringSpec(key, category, label, description, def string) Spec {
	return Spec{Key: key, Type: TypeString, Category: category, Label: label, Description: description, Default: def}
}

func intSpec(key, category, label, description, def, unit string, min, max int) Spec {
	return Spec{Key: key, Type: TypeInt, Category: category, Label: label, Description: description, Default: def, Unit: unit, Min: intp(min), Max: intp(max)}
}

func selectSpec(key, category, label, description, def string, options []Option) Spec {
	return Spec{Key: key, Type: TypeSelect, Category: category, Label: label, Description: description, Default: def, Options: options}
}

func defaultSpecs() []Spec {
	return []Spec{
		boolSpec(KeyCacheEnabled, "performance", "启用元数据缓存", "关闭后每次浏览目录都会直接请求网盘，不经过本地缓存；建议保持开启以提升列表加载速度。", "true"),
		intSpec(KeyCacheTTL, "performance", "全局缓存时间", "目录列表缓存的有效时长，超时后自动重新获取；设为 0 表示不缓存。", "30", "分钟", 0, 1440),
		intSpec(KeyCacheMaxItems, "performance", "缓存条目上限", "内存中最多保留的目录条目数，超出后按最近最少使用（LRU）淘汰旧条目。", "10000", "条", 1000, 1000000),
		intSpec(KeyCacheMemoryLimitMB, "performance", "缓存内存上限", "目录缓存可占用的内存软上限，接近上限时会主动淘汰较旧的条目以控制内存。", "128", "MB", 64, 16384),
		boolSpec(KeyCachePersistenceEnabled, "performance", "启用缓存持久化", "开启后定期将未过期的目录缓存写入磁盘，重启后可快速恢复，无需重新拉取。", "true"),
		intSpec(KeyCachePersistenceIntervalMin, "performance", "持久化间隔", "将内存中的缓存同步到磁盘的时间间隔，修改后立即生效。", "10", "分钟", 1, 1440),
		intSpec(KeyUploadTaskConcurrency, "performance", "任务并发数", "同时进行的最大上传任务数；修改后新调度的任务立即生效。", "3", "个", 1, 5),
		boolSpec(KeyFuseReadCacheEnabled, "performance", "FUSE 读缓存", "开启后通过 FUSE 读取的文件块会缓存到本地磁盘，加速重复读取；与目录元数据缓存相互独立。需在「文件共享 → 本地挂载」页配置。", "false"),
		intSpec(KeyFuseReadCacheMaxGB, "performance", "FUSE 读缓存容量上限", "本地磁盘块缓存允许占用的最大空间，超出后按淘汰策略清理。需在「文件共享 → 本地挂载」页配置。", "10", "GB", 1, 500),
		intSpec(KeyFuseReadCacheRetentionDays, "performance", "FUSE 读缓存保留天数", "未被访问的缓存块超过该天数后自动清理，用于回收长期未用的磁盘空间。需在「文件共享 → 本地挂载」页配置。", "7", "天", 1, 90),
		selectSpec(KeyFuseReadCacheEvictionPolicy, "performance", "FUSE 读缓存淘汰策略", "磁盘占用达到上限时的清理方式。需在「文件共享 → 本地挂载」页配置。", "lru", []Option{
			{Value: "lru", Label: "最近最少使用（LRU）"},
			{Value: "large_file", Label: "大文件优先"},
		}),
		boolSpec(KeyAuthActiveRefresh, "system", "智能主动认证刷新", "开启后系统会根据 Token 有效期在后台自动刷新并检查 Cookie 健康度；关闭后仅在访问失效时被动刷新。", "true"),
		boolSpec(KeyAccountShowProfile, "account_display", "显示账号信息", "开启后在账号卡片第二行展示昵称、手机号或邮箱；关闭则显示账号创建时间。", "true"),
		boolSpec(KeyAccountShowMembership, "account_display", "显示会员标签", "开启后在账号名称旁展示网盘返回的会员等级，如 VIP / SVIP。", "true"),
		selectSpec(KeyLogLevel, "system", "日志级别", "控制台及落盘日志的最低输出级别；日常建议保持 Info，需要排查问题时再切换到 Debug。", "info", []Option{
			{Value: "debug", Label: "Debug（调试）"},
			{Value: "info", Label: "Info（常规）"},
			{Value: "warn", Label: "Warn（警告）"},
			{Value: "error", Label: "Error（错误）"},
		}),
		intSpec(KeyLogRetentionDays, "system", "日志保留天数", "本地日志文件的最长保留天数，超出后自动清理；在日志页面也可手动触发清理。", "30", "天", 1, 365),
		{
			Key:         KeyOAuthServerURL,
			Type:        TypeString,
			Category:    "system",
			Label:       "OAuth 代理服务地址",
			Description: "用于账号授权时「自动获取 Token」的转发服务；留空或格式错误时自动回落至默认值，本地调试可填 http://127.0.0.1:8000。",
			Default:     domain.DefaultOAuthServerURL,
			normalize:   domain.NormalizeOAuthServerURL,
		},
		{
			Key:     KeyLogErrorAckAt,
			Type:    TypeString,
			Default: "",
			Hidden:  true,
		},		{
			Key:     KeyLocalUploadEnabled,
			Type:    TypeBool,
			Default: "false",
			Hidden:  true,
		},
		{
			Key:     KeyLocalUploadMappings,
			Type:    TypeString,
			Default: "[]",
			Hidden:  true,
		},
	}
}

// categories 返回有序分组定义；只保留当前实际用到的分组。
func categories() []Category {
	return []Category{
		{ID: "system", Label: "系统设置"},
		{ID: "account_display", Label: "网盘账号显示"},
		{ID: "performance", Label: "性能设置"},
	}
}
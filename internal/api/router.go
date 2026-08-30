package api

import (
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"litepan/internal/account"
	"litepan/internal/accountprofile"
	"litepan/internal/adminauth"
	"litepan/internal/announcement"
	"litepan/internal/apikey"
	"litepan/internal/auth"
	"litepan/internal/automation"
	"litepan/internal/backuprestore"
	"litepan/internal/cache"
		"litepan/internal/domain"
	"litepan/internal/favorites"
	"litepan/internal/file"
	"litepan/internal/fusemount"
	"litepan/internal/logx"
	"litepan/internal/notification"
	"litepan/internal/offlinedownload"
	"litepan/internal/playback"
	"litepan/internal/settings"
	"litepan/internal/upload"
)

//go:embed web
var webFS embed.FS

// Deps 是 API 层所需依赖（只依赖接口/服务，不感知具体存储实现）。
type Deps struct {
	Logs              *logx.Manager
	AccountSvc        *account.Service
	AccountProfile    *accountprofile.Service
	Accounts          domain.AccountRepository
	Configs           domain.ConfigRepository
	Settings          *settings.Service
	Cache             *cache.Service
	ListHitTracker    *cache.HitTracker
	Files             *file.Service
	Favorites         *favorites.Service
	Uploads           *upload.Manager
	OfflineDownloads  *offlinedownload.Service
	Playback          *playback.Service
	Automation        *automation.Service
	Fuse              *fusemount.Service
	ApiKeys           *apikey.Service
	Auth              *auth.Service
	AuthSched         *auth.Scheduler
	AdminAuth         *adminauth.Service
	Notifications     *notification.Service
	Announcement      *announcement.Service
	BackupRestore     *backuprestore.Service
	DataDir           string
	OnSettingsUpdated func(map[string]string)
}

// Handler 持有处理请求所需的依赖。
type Handler struct {
	bootID            string
	logs              *logx.Manager
	log               *slog.Logger
	accountSvc        *account.Service
	accountProfile    *accountprofile.Service
	settings          *settings.Service
	cache             *cache.Service
	listHits          *cache.HitTracker
	files             *file.Service
	favorites         *favorites.Service
	uploads           *upload.Manager
	offlineDownloads  *offlinedownload.Service
	playback          *playback.Service
	automation        *automation.Service
	fuse              *fusemount.Service
	apiKeys           *apikey.Service
	auth              *auth.Service
	authSched         *auth.Scheduler
	adminAuth         *adminauth.Service
	notifications     *notification.Service
	announcement      *announcement.Service
	backupRestore     *backuprestore.Service
	dataDir           string
	onSettingsUpdated func(map[string]string)

	devMu       sync.Mutex
	devUnlocked bool
}

// NewRouter 装配并返回 HTTP 路由（含内嵌管理页面）。
func NewRouter(d Deps) http.Handler {
	apiLog := slog.Default()
	if d.Logs != nil {
		apiLog = d.Logs.For(logx.ModuleAPI)
	}
	h := &Handler{
		bootID:            uuid.NewString(),
		logs:              d.Logs,
		log:               apiLog,
		accountSvc:        d.AccountSvc,
		accountProfile:    d.AccountProfile,
		settings:          d.Settings,
		cache:             d.Cache,
		listHits:          d.ListHitTracker,
		files:             d.Files,
		favorites:         d.Favorites,
		uploads:           d.Uploads,
		offlineDownloads:  d.OfflineDownloads,
		playback:          d.Playback,
		automation:        d.Automation,
		fuse:              d.Fuse,
		apiKeys:           d.ApiKeys,
		auth:              d.Auth,
		authSched:         d.AuthSched,
		adminAuth:         d.AdminAuth,
		notifications:     d.Notifications,
		announcement:      d.Announcement,
		backupRestore:     d.BackupRestore,
		dataDir:           d.DataDir,
		onSettingsUpdated: d.OnSettingsUpdated,
	}

	r := chi.NewRouter()
	r.Use(trackResponseCommit)
	r.Use(chimw.RequestID)
	r.Use(h.attachRequestLogger)
	r.Use(chimw.Recoverer)

	r.Get("/install-docker.sh", h.serveInstallDocker)
	r.Get("/install-docker-fixed.sh", h.serveInstallDocker)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.health)
		r.Get("/auth/status", h.authStatus)
		r.Post("/auth/login", h.authLogin)
		r.Post("/auth/logout", h.authLogout)
		r.Post("/auth/reset-password", h.authResetPassword)
		r.Route("/public", func(r chi.Router) {
			r.Use(h.requirePublicOrAdmin)
			r.Get("/accounts", h.publicAccounts)
			r.Get("/system-config", h.publicSystemConfig)
			r.Get("/cache/hit-rate", h.publicCacheHitRate)
		})
		r.Route("/open", func(r chi.Router) {
			r.Post("/automation/events", h.automationWebhook)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.requireAdmin)
			r.Get("/logs", h.listLogs)
			r.Get("/logs/stats", h.logStats)
			r.Post("/logs/ack-errors", h.ackRecentErrors)
			r.Post("/logs/cleanup", h.cleanupLogs)
			r.Post("/logs/cleanup/keep-today", h.cleanupLogsKeepToday)
			r.Post("/logs/cleanup/all", h.cleanupLogsAll)
			r.Route("/admin", func(r chi.Router) {
				r.Get("/system-config", h.adminSystemConfig)
				r.Route("/backups", func(r chi.Router) {
					r.Get("/", h.listBackups)
					r.Post("/", h.createBackup)
					r.Post("/import", h.importBackup)
					r.Get("/status", h.backupRestoreStatus)
					r.Post("/status/ack", h.acknowledgeRestoreStatus)
					r.Delete("/pending", h.cancelPendingRestore)
					r.Post("/restart", h.restartForRestore)
					r.Get("/{id}/download", h.downloadBackup)
					r.Post("/{id}/restore", h.prepareBackupRestore)
					r.Delete("/{id}", h.deleteBackup)
				})
				r.Post("/update-credentials", h.adminUpdateCredentials)
				r.Get("/local-fs/browse", h.browseLocalFS)
				r.Get("/drivers", h.listDrivers)
				r.Get("/dev/state", h.getDevState)
				r.Post("/dev/unlock", h.unlockDevMode)
				r.Get("/accounts", h.listAccounts)
				r.Post("/accounts", h.createAccount)
				r.Get("/accounts/{id}", h.getAccount)
				r.Put("/accounts/{id}", h.updateAccount)
				r.Delete("/accounts/{id}", h.deleteAccount)
				r.Post("/accounts/{id}/toggle", h.toggleAccount)
				r.Post("/accounts/{id}/set-default", h.setDefaultAccount)
				r.Post("/accounts/{id}/refresh-auth", h.refreshAccountAuth)
				r.Post("/accounts/{id}/refresh-profile", h.refreshAccountProfile)
				r.Get("/settings", h.getSettings)
				r.Put("/settings", h.updateSettings)
				r.Route("/api-keys", func(r chi.Router) {
					r.Get("/", h.listApiKeys)
					r.Post("/", h.createApiKey)
					r.Put("/{id}", h.updateApiKey)
					r.Post("/{id}/toggle", h.toggleApiKey)
					r.Delete("/{id}", h.deleteApiKey)
				})
				r.Get("/cache/stats", h.cacheStats)
				r.Get("/cache/stats/{id}", h.accountCacheStats)
				r.Post("/clear-cache", h.clearCache)
				r.Get("/notifications", h.listNotifications)
				r.Get("/notifications/unread-count", h.notificationUnreadCount)
				r.Post("/notifications/read-all", h.markAllNotificationsRead)
				r.Delete("/notifications", h.deleteAllNotifications)
				r.Post("/notifications/{id}/read", h.markNotificationRead)
				r.Delete("/notifications/{id}", h.deleteNotification)
				r.Get("/announcement", h.getAnnouncement)
				r.Post("/announcement/read", h.markAnnouncementRead)
				r.Route("/tools/local-upload", func(r chi.Router) {
					r.Get("/config", h.getLocalUploadConfig)
					r.Put("/config", h.updateLocalUploadConfig)
					r.Post("/browse", h.browseLocalUpload)
					r.Post("/upload", h.createLocalUploadTasks)
				})
				r.Route("/automation", func(r chi.Router) {
					r.Get("/rules", h.listAutomationRules)
					r.Post("/rules", h.createAutomationRule)
					r.Put("/rules/{id}", h.updateAutomationRule)
					r.Delete("/rules/{id}", h.deleteAutomationRule)
					r.Post("/rules/{id}/toggle", h.toggleAutomationRule)
					r.Post("/rules/{id}/run", h.runAutomationRule)
					r.Post("/validate", h.validateAutomationRule)
					r.Get("/runs", h.listAutomationRuns)
					r.Post("/runs/clear", h.clearAutomationRuns)
					r.Get("/options", h.automationOptions)
				})
				r.Route("/fuse", func(r chi.Router) {
					r.Get("/status", h.fuseStatus)
					r.Put("/config", h.updateFuseConfig)
					r.Get("/read-cache", h.getFuseReadCache)
					r.Put("/read-cache", h.updateFuseReadCache)
					r.Post("/read-cache/clear", h.clearFuseReadCache)
					r.Get("/mounts", h.listFuseMounts)
					r.Post("/mounts", h.createFuseMount)
					r.Put("/mounts/{id}", h.updateFuseMount)
					r.Delete("/mounts/{id}", h.deleteFuseMount)
					r.Post("/mounts/{id}/mount", h.mountFuse)
					r.Post("/mounts/{id}/unmount", h.unmountFuse)
				})
			})
			r.Post("/oauth/start", h.startOAuth)
			r.Get("/oauth/status/{session_id}", h.oauthStatus)
			r.Post("/oauth/confirm-received/{session_id}", h.oauthConfirmReceived)
			r.Post("/qr/start", h.startQRLogin)
			r.Post("/qr/poll", h.pollQRLogin)
		})
		r.Route("/files", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(h.requirePublicOrAdmin)
				r.Get("/list", h.listFiles)
				r.Get("/info", h.fileInfo)
				r.Get("/download", h.downloadFile)
				r.Head("/download", h.downloadFile)
			})
			r.Group(func(r chi.Router) {
				r.Use(h.requireAdmin)
				r.Delete("/delete", h.deleteFiles)
				r.Post("/move", h.moveFiles)
				r.Post("/copy", h.copyFiles)
				r.Put("/rename", h.renameFile)
				r.Get("/favorites", h.getFavorites)
				r.Put("/favorites", h.saveFavorites)
				r.Post("/name-align/preview", h.previewNameAlign)
				r.Post("/name-align/apply", h.applyNameAlign)
				r.Post("/create-folder", h.createFolder)
				r.Post("/upload-task", h.createUploadTask)
				r.Get("/upload/runtime", h.getUploadRuntime)
				r.Put("/upload/runtime", h.updateUploadRuntime)
				r.Get("/upload/tasks", h.listUploadTasks)
				r.Get("/upload/tasks/stream", h.streamUploadTasks)
				r.Get("/upload/tasks/{taskID}", h.getUploadTask)
				r.Post("/upload/tasks/{taskID}/pause", h.pauseUploadTask)
				r.Post("/upload/tasks/{taskID}/resume", h.resumeUploadTask)
				r.Delete("/upload/tasks/{taskID}", h.deleteUploadTask)
				r.Post("/upload/tasks/batch-delete", h.batchDeleteUploadTasks)
				r.Route("/offline-download", func(r chi.Router) {
					r.Get("/capabilities", h.offlineDownloadCapabilities)
					r.Post("/urls", h.addOfflineURLs)
					r.Post("/torrent/prepare", h.prepareOfflineTorrent)
					r.Post("/torrent", h.addOfflineTorrent)
					r.Get("/tasks", h.listOfflineDownloadTasks)
					r.Post("/tasks/refresh", h.refreshOfflineDownloadTasks)
					r.Post("/tasks/batch-delete", h.batchDeleteOfflineDownloadTasks)
					r.Delete("/tasks/{taskID}", h.deleteOfflineDownloadTask)
				})
			})
		})
	})

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err) // 编译期内嵌，理论上不会失败
	}
	r.Handle("/*", spaHandler(sub))

	return r
}

func (h *Handler) serveInstallDocker(w http.ResponseWriter, r *http.Request) {
	candidates := []string{
		h.dataDir + "/install-docker.sh",
		h.dataDir + "/install-docker-fixed.sh",
		"/app/data/install-docker.sh",
		"/app/data/install-docker-fixed.sh",
	}
	var data []byte
	var lastErr error
	for _, p := range candidates {
		if p == "" || p == "/" {
			continue
		}
		if b, err := os.ReadFile(p); err == nil {
			data = b
			lastErr = nil
			break
		} else {
			lastErr = err
		}
	}
	if len(data) == 0 {
		msg := "install-docker.sh not found"
		if lastErr != nil {
			msg += ": " + lastErr.Error()
		}
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	index, _ := fs.ReadFile(fsys, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" {
			upath = "index.html"
		}
		if _, statErr := fs.Stat(fsys, upath); statErr == nil {
			setStaticCacheHeader(w, upath)
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, statErr := fs.Stat(fsys, upath+".gz"); statErr == nil {
			serveCompressedAsset(w, r, fsys, upath)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method != http.MethodHead {
			_, _ = w.Write(index)
		}
	})
}

func serveCompressedAsset(w http.ResponseWriter, r *http.Request, fsys fs.FS, upath string) {
	compressed, err := fsys.Open(upath + ".gz")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer compressed.Close()
	info, err := compressed.Stat()
	if err != nil {
		http.Error(w, "静态资源读取失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", staticContentType(upath))
	w.Header().Set("Vary", "Accept-Encoding")
	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		if r.Method != http.MethodHead {
			_, _ = io.Copy(w, compressed)
		}
		return
	}

	if r.Method == http.MethodHead {
		return
	}
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		http.Error(w, "静态资源解压失败", http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	_, _ = io.Copy(w, reader)
}

func setStaticCacheHeader(w http.ResponseWriter, upath string) {
	if strings.HasPrefix(upath, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

func staticContentType(name string) string {
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		return contentType
	}
	switch path.Ext(name) {
	case ".mjs", ".js":
		return "text/javascript; charset=utf-8"
	case ".wasm":
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, value := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(value), ";")
		coding := strings.ToLower(strings.TrimSpace(parts[0]))
		quality := 1.0
		for _, parameter := range parts[1:] {
			key, raw, found := strings.Cut(strings.TrimSpace(parameter), "=")
			if !found || !strings.EqualFold(key, "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				quality = 0
			} else {
				quality = parsed
			}
		}
		if coding == "gzip" {
			return quality > 0
		}
		if coding == "*" {
			wildcard = quality > 0
		}
	}
	return wildcard
}

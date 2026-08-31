package app

import (
		"net/http"
	"time"

	"litepan/internal/adminauth"
		"litepan/internal/api"
	"litepan/internal/apikey"
	"litepan/internal/backuprestore"
	"litepan/internal/buildinfo"
	"litepan/internal/cache"
	"litepan/internal/config"
	"litepan/internal/logx"
	"litepan/internal/notification"
	"litepan/internal/settings"
)

func wireHTTPServer(cfg config.Config, logs *logx.Manager, st *storeBundle, core *coreBundle, svc *servicesBundle, onRestart func()) (*http.Server, error) {
	notifySvc := notification.NewService(notification.Options{
		Repo:     st.store.Notifications,
		Accounts: st.store.Accounts,
		Log:      logs.For(logx.ModuleSystem),
	})
	notifySvc.Register(core.bus)

	apiKeySvc := apikey.New(apikey.Options{
		Repo:     st.store.ApiKeys,
		Settings: st.settings,
		Secret:   core.secret,
	})
	if svc.automation != nil {
		svc.automation.SetApiKeys(apiKeySvc)
	}
	backupRestoreSvc, err := backuprestore.New(backuprestore.Options{
		DataDir:   cfg.DataDir,
		DBPath:    cfg.DBPath,
		Version:   buildinfo.Version,
		DB:        st.db,
		Configs:   st.store.Configs,
		Secret:    core.secret,
		Log:       logs.For(logx.ModuleSystem),
		OnRestart: onRestart,
	})
	if err != nil {
		return nil, err
	}
	router := api.NewRouter(api.Deps{
		Logs:             logs,
		AccountSvc:       svc.account,
		AccountProfile:   svc.accountProfile,
		Accounts:         st.store.Accounts,
		Configs:          st.store.Configs,
		Settings:         st.settings,
		Cache:            core.cache,
		ListHitTracker:   core.listHits,
		Files:     svc.files,
		Favorites: svc.favorites,
		Uploads:   svc.uploads,
		Playback:  svc.playback,
		Automation:       svc.automation,
		Fuse:             svc.fuse,
		ApiKeys:          apiKeySvc,
		Auth:             core.auth,
		AuthSched:        core.sched,
		AdminAuth:        adminauth.New(st.store.Configs, core.secret, logs.For(logx.ModuleAPI)),
		Notifications:    notifySvc,
		BackupRestore:    backupRestoreSvc,
		DataDir:          cfg.DataDir,
		OnSettingsUpdated: cacheSettingsHook(core.cache, st.settings, cfg.DataDir),
	})

	return &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}, nil
}

func cacheSettingsHook(cacheSvc *cache.Service, settingsSvc *settings.Service, dataDir string) func(map[string]string) {
	return func(changed map[string]string) {
		if !settingsTouchesCache(changed) {
			return
		}
		applyCacheRuntime(cacheSvc, settingsSvc, dataDir)
	}
}

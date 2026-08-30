package app

import (
	"context"

	"litepan/internal/account"
	"litepan/internal/accountprofile"
	"litepan/internal/automation"
	"litepan/internal/config"
	"litepan/internal/crosstransfer"
	"litepan/internal/domain"
	"litepan/internal/favorites"
	"litepan/internal/file"
	"litepan/internal/fusemount"
	"litepan/internal/fusereadcache"
	"litepan/internal/logx"
	"litepan/internal/offlinedownload"
	"litepan/internal/playback"
	"litepan/internal/settings"
	"litepan/internal/upload"
)

type servicesBundle struct {
	files            *file.Service
	uploads          *upload.Manager
	offlineDownloads *offlinedownload.Service
	playback         *playback.Service
	account          *account.Service
	accountProfile   *accountprofile.Service
	automation       *automation.Service
	fuse             *fusemount.Service
	fuseReadCache    *fusereadcache.Service
	crossTransfer    *crosstransfer.Service
	favorites        *favorites.Service
}

func wireServices(cfg config.Config, logs *logx.Manager, st *storeBundle, core *coreBundle) *servicesBundle {
	var startupGate <-chan struct{}
	if core != nil && core.sched != nil {
		startupGate = core.sched.StartupReady()
	}
	favoritesSvc := favorites.NewService(cfg.DBPath, logs.For(logx.ModuleSystem))
	fileSvc := file.NewService(core.exec, core.cache, st.store.Accounts, core.bus, st.settings, core.listHits)
	fileSvc.SetLogger(logs.For(logx.ModuleFileOp))
	playbackSvc := playback.NewService(core.exec, core.cache)
	fuseReadCache := wireFuseReadCacheOrNil(context.Background(), cfg, logs, st, core.bus)
	offlineDownloadSvc := offlinedownload.New(offlinedownload.Options{
		Exec:     core.exec,
		Accounts: st.store.Accounts,
		Repo:     st.store.OfflineDownloads,
		Folders:  fileSvc,
		Settings: st.settings,
		DataDir:  cfg.DataDir,
		Bus:      core.bus,
		Log:      logs.For(logx.ModuleFileOp),
	})
	fusemount.ApplyConfiguredMountRoot(context.Background(), st.store.Configs)
	fuseSvc := fusemount.New(fusemount.Options{
		Repo:      st.store.FuseMounts,
		Configs:   st.store.Configs,
		Accounts:  st.store.Accounts,
		Notify:    st.store.Notifications,
		Files:     fileSvc,
		Playback:  playbackSvc,
		ReadCache: fuseReadCache,
		Bus:       core.bus,
		Log:       logs.For(logx.ModuleSystem),
	})
	fuseSvc.SetStartupGate(startupGate)
	fuseSvc.Register(core.bus)
	_ = fuseSvc.PrepareMountRoot()
	lifecycle := &accountLifecycle{
		fuse:      fuseSvc,
		readCache: fuseReadCache,
		favorites: favoritesSvc,
		offline:   offlineDownloadSvc,
	}
	accountSvc := account.NewService(account.Options{
		Accounts:      st.store.Accounts,
		AuthStates:    st.store.AuthStates,
		Drivers:       core.drivers,
		Auth:          core.auth,
		Playback:      playbackSvc,
		MetadataCache: core.cache,
		Lifecycle:     lifecycle,
		OAuthURL: func(context.Context) string {
			return domain.NormalizeOAuthServerURL(st.settings.String(settings.KeyOAuthServerURL))
		},
	})
	accountProfileSvc := accountprofile.New(core.exec)
	uploadSvc := upload.NewManager(upload.Options{
		Exec:        core.exec,
		Files:       fileSvc,
		Playback:    playbackSvc,
		Accounts:    accountSvc,
		Repo:        st.store.UploadTasks,
		Settings:    st.settings,
		Bus:         core.bus,
		DataDir:     cfg.DataDir,
		Log:         logs.For(logx.ModuleFileOp),
		StartupGate: startupGate,
	})
	lifecycle.uploads = uploadSvc
	offlineDownloadSvc.SetUploads(uploadSvc)
	fuseSvc.SetUploads(uploadSvc)
	crossTransferSvc := crosstransfer.New(crosstransfer.Options{
		Exec:    core.exec,
		Files:   fileSvc,
		Uploads: uploadSvc,
		Log:     logs.For(logx.ModuleAPI),
	})
	automationSvc := automation.New(automation.Options{
		Rules: st.store.AutomationRules,
		Runs:  st.store.AutomationRuns,
		Files: fileSvc,
		Log:   logs.For(logx.ModuleSystem),
	})
	automationSvc.SetStartupGate(startupGate)
	automationSvc.Register(core.bus)
	return &servicesBundle{
		files:            fileSvc,
		uploads:          uploadSvc,
		offlineDownloads: offlineDownloadSvc,
		playback:         playbackSvc,
		account:          accountSvc,
		accountProfile:   accountProfileSvc,
		automation:       automationSvc,
		fuse:             fuseSvc,
		fuseReadCache:    fuseReadCache,
		crossTransfer:    crossTransferSvc,
		favorites:        favoritesSvc,
	}
}

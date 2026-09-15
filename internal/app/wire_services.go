package app

import (
	"context"

	"litepan/internal/account"
	"litepan/internal/accountprofile"
	"litepan/internal/automation"
	"litepan/internal/config"
		"litepan/internal/domain"
	"litepan/internal/favorites"
	"litepan/internal/file"
	"litepan/internal/logx"
	"litepan/internal/playback"
	"litepan/internal/settings"
	"litepan/internal/upload"
)

type servicesBundle struct {
	files          *file.Service
	uploads        *upload.Manager
	playback       *playback.Service
	account        *account.Service
	accountProfile *accountprofile.Service
	automation     *automation.Service
	favorites      *favorites.Service
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
	lifecycle := &accountLifecycle{
		favorites: favoritesSvc,
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
	automationSvc := automation.New(automation.Options{
		Rules:    st.store.AutomationRules,
		Runs:     st.store.AutomationRuns,
		Files:    fileSvc,
		Settings: st.settings,
		DataDir:  cfg.DataDir,
		Uploads:  uploadSvc,
		Log:      logs.For(logx.ModuleSystem),
	})
	automationSvc.SetStartupGate(startupGate)
	return &servicesBundle{
		files:          fileSvc,
		uploads:        uploadSvc,
		playback:       playbackSvc,
		account:        accountSvc,
		accountProfile: accountProfileSvc,
		automation:     automationSvc,
		favorites:      favoritesSvc,
	}
}

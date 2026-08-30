package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"litepan/internal/app"
	"litepan/internal/config"
	"litepan/internal/logx"

	_ "litepan/drivers" //
)

const shutdownBudget = 25 * time.Second

func main() {
	cfg := config.Load()

	listen := flag.String("listen", cfg.ListenAddr, "HTTP 监听地址")
	dataDir := flag.String("data-dir", cfg.DataDir, "数据目录")
	flag.Parse()

	cfg.ListenAddr = *listen
	if *dataDir != cfg.DataDir {
		cfg.DataDir = *dataDir
		cfg.DBPath = filepath.Join(*dataDir, "litepan.db")
	}

	logs, err := logx.New(logx.Options{
		Dir:   filepath.Join(cfg.DataDir, "log"),
		Level: cfg.LogLevel,
	})
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx, app.Options{Config: cfg, Logs: logs})
	if err != nil {
		logs.Root().Error("startup failed", "err", err)
		os.Exit(1)
	}

	runErr := a.Run(ctx)
	restarting := errors.Is(runErr, app.ErrRestartRequested)
	stop()

	shCtx, cancel := context.WithTimeout(context.Background(), shutdownBudget)
	defer cancel()
	if err := a.Shutdown(shCtx); err != nil {
		logs.Root().Error("shutdown error", "err", err)
		os.Exit(1)
	}
	if runErr != nil && !restarting {
		logs.Root().Error("run error", "err", runErr)
		os.Exit(1)
	}
}

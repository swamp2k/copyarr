package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/swamp2k/copyarr/internal/api"
	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
	"github.com/swamp2k/copyarr/internal/engine"
	"github.com/swamp2k/copyarr/internal/logging"
)

var (
	version  = "dev"
	revision = "unknown"
)

func main() {
	cfgPath := flag.String("config", "/config/config.json", "config path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		slog.Error("create data dir failed", "err", err)
		os.Exit(1)
	}
	origin, err := config.EnsureWritableRcloneConfig(&cfg)
	if err != nil {
		slog.Error("prepare writable rclone config failed", "err", err)
		os.Exit(1)
	}
	slog.Info("rclone config ready", "path", cfg.RcloneConfig, "origin", string(origin))
	database, err := db.Open(filepath.Join(cfg.DataDir, "copyarr.db"))
	if err != nil {
		slog.Error("open db failed", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	loggingEnabled := true
	if v, ok, metaErr := database.Meta("settings:logging_enabled"); metaErr == nil && ok && v == "false" {
		loggingEnabled = false
	}
	logging.Install(database, loggingEnabled)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	eng := engine.New(cfg, database, version, revision)
	eng.Run(ctx)

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: api.New(eng).Handler()}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.Info("copyarr starting", "version", version, "revision", revision, "listen", cfg.ListenAddr, "rules", len(cfg.Rules))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("http server failed", "err", err)
		os.Exit(1)
	}
}

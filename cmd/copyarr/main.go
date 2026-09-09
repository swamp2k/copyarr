package main

import (
	"context"
	"flag"
	"io"
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
	if err := prepareWritableRcloneConfig(&cfg); err != nil {
		slog.Error("prepare writable rclone config failed", "err", err)
		os.Exit(1)
	}
	database, err := db.Open(filepath.Join(cfg.DataDir, "copyarr.db"))
	if err != nil {
		slog.Error("open db failed", "err", err)
		os.Exit(1)
	}
	defer database.Close()

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

func prepareWritableRcloneConfig(cfg *config.Config) error {
	dir := filepath.Join(cfg.DataDir, "rclone")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(dir, "rclone.conf")
	if _, err := os.Stat(dst); err == nil {
		cfg.RcloneConfig = dst
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if cfg.RcloneConfig != "" {
		if src, err := os.Open(cfg.RcloneConfig); err == nil {
			defer src.Close()
			out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, src)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			cfg.RcloneConfig = dst
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	if err := os.WriteFile(dst, nil, 0o600); err != nil {
		return err
	}
	cfg.RcloneConfig = dst
	return nil
}

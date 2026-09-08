package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
	rc "github.com/swamp2k/copyarr/internal/rclone"
	rt "github.com/swamp2k/copyarr/internal/rtorrent"
)

type Engine struct {
	cfg      config.Config
	db       *db.DB
	rc       rc.Client
	wake     chan struct{}
	mu       sync.Mutex
	scanning bool
}

func New(cfg config.Config, d *db.DB) *Engine {
	return &Engine{
		cfg:  cfg,
		db:   d,
		rc:   rc.Client{ConfigPath: cfg.RcloneConfig},
		wake: make(chan struct{}, 1),
	}
}

func (e *Engine) Run(ctx context.Context) {
	go e.worker(ctx)
	go e.scheduler(ctx)
}

func (e *Engine) TriggerScan() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

func (e *Engine) scheduler(ctx context.Context) {
	e.scanAll(ctx)
	t := time.NewTicker(e.cfg.ScanInterval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.scanAll(ctx)
		case <-e.wake:
			e.scanAll(ctx)
		}
	}
}

func objectKey(rel string, size int64, mod time.Time) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s", rel, size, mod.UTC().Format(time.RFC3339Nano))))
	return hex.EncodeToString(h[:])
}

func (e *Engine) scanAll(ctx context.Context) {
	e.mu.Lock()
	if e.scanning {
		e.mu.Unlock()
		return
	}
	e.scanning = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.scanning = false
		e.mu.Unlock()
	}()

	for _, r := range e.cfg.Rules {
		if !r.Enabled {
			continue
		}
		if err := e.scanRule(ctx, r); err != nil {
			slog.Error("scan failed", "rule", r.ID, "err", err)
		}
		if err := e.cleanupRule(ctx, r); err != nil {
			slog.Error("cleanup failed", "rule", r.ID, "err", err)
		}
	}
}

func (e *Engine) scanRule(ctx context.Context, r config.Rule) error {
	items, err := e.rc.ListFiles(ctx, r.Source)
	if err != nil {
		return err
	}
	metaKey := "rule:" + r.ID + ":initialized"
	_, initialized, err := e.db.Meta(metaKey)
	if err != nil {
		return err
	}

	var completed []rt.Torrent
	rtOK := false
	if r.RTorrent != nil {
		completed, err = rt.New(*r.RTorrent).Completed(ctx)
		if err != nil {
			if r.RTorrent.Required {
				return fmt.Errorf("rtorrent required: %w", err)
			}
			slog.Warn("rtorrent unavailable, using stability fallback", "rule", r.ID, "err", err)
		} else {
			rtOK = true
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, it := range items {
		if it.IsDir {
			continue
		}
		k := objectKey(it.Path, it.Size, it.ModTime)
		o, isNew, err := e.db.UpsertSeen(r.ID, k, it.Path, it.Size, it.ModTime.UTC().Format(time.RFC3339Nano), now, "discovered")
		if err != nil {
			return err
		}
		if !initialized && r.InitialBehavior == "ignore_existing" {
			if isNew {
				_ = e.db.SetStateByKey(r.ID, k, "ignored", "")
			}
			continue
		}
		if !isNew && o.State != "discovered" && o.State != "retry_wait" {
			continue
		}

		ready := false
		reason := ""
		if rtOK && matchesCompleted(it.Path, *r.RTorrent, completed) {
			ready = true
			reason = "rtorrent_complete"
		} else {
			stableSince := now
			if !isNew {
				stableSince = o.StableSince
			}
			t, parseErr := time.Parse(time.RFC3339Nano, stableSince)
			if parseErr == nil {
				ready = time.Since(t) >= time.Duration(r.StabilitySeconds)*time.Second
				if ready {
					reason = "stable"
				}
			}
		}
		if ready {
			id := o.ID
			if isNew {
				id, err = e.db.IDByKey(r.ID, k)
				if err != nil {
					return err
				}
			}
			if err := e.db.Queue(id); err != nil {
				return err
			}
			slog.Info("queued object", "rule", r.ID, "path", it.Path, "reason", reason, "size", it.Size)
		}
	}

	if !initialized {
		if err := e.db.SetMeta(metaKey, "1"); err != nil {
			return err
		}
		slog.Info("rule initialized", "rule", r.ID, "objects", len(items), "behavior", r.InitialBehavior)
	}
	return nil
}

func matchesCompleted(rel string, c config.RTorrent, torrents []rt.Torrent) bool {
	base := strings.TrimSuffix(c.SourceBasePath, "/")
	for _, t := range torrents {
		p := t.BasePath
		if base != "" && strings.HasPrefix(p, base) {
			p = strings.TrimPrefix(p, base)
			p = strings.TrimPrefix(p, "/")
		}
		p = strings.TrimSuffix(p, "/")
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

func (e *Engine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		o, err := e.db.NextQueued()
		if db.IsNoRows(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
			slog.Error("queue read failed", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}
		r, ok := e.rule(o.RuleID)
		if !ok {
			_ = e.db.Fail(o.ID, "rule no longer exists")
			continue
		}
		if err := e.transfer(ctx, r, o); err != nil {
			slog.Error("transfer failed", "object", o.String(), "err", err)
			_ = e.db.Fail(o.ID, err.Error())
			time.Sleep(5 * time.Second)
			_ = e.db.Queue(o.ID)
		}
	}
}

func (e *Engine) transfer(ctx context.Context, r config.Rule, o db.Object) error {
	if err := e.db.Start(o.ID); err != nil {
		return err
	}
	src := rc.Target(r.Source, o.RelPath)
	final := rc.Target(r.Destination, o.RelPath)
	stage := final + fmt.Sprintf(".copyarr-stage-%d", o.ID)

	slog.Info("copying", "source", src, "stage", stage, "size", o.Size)
	if err := e.rc.CopyTo(ctx, src, stage, r.RcloneArgs); err != nil {
		return err
	}
	st, err := e.rc.Stat(ctx, stage)
	if err != nil {
		return fmt.Errorf("verify stat: %w", err)
	}
	if st.Size != o.Size {
		return fmt.Errorf("size mismatch: source=%d staged=%d", o.Size, st.Size)
	}
	if err := e.rc.MoveTo(ctx, stage, final); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	st, err = e.rc.Stat(ctx, final)
	if err != nil {
		return fmt.Errorf("verify committed: %w", err)
	}
	if st.Size != o.Size {
		return fmt.Errorf("committed size mismatch: source=%d dest=%d", o.Size, st.Size)
	}
	if r.Mode == "move" {
		if err := e.rc.DeleteFile(ctx, src); err != nil {
			return fmt.Errorf("destination committed but source delete failed: %w", err)
		}
	}
	return e.db.Complete(o.ID, final)
}

func (e *Engine) cleanupRule(ctx context.Context, r config.Rule) error {
	if r.CleanupDays <= 0 {
		return nil
	}
	before := time.Now().UTC().Add(-time.Duration(r.CleanupDays) * 24 * time.Hour).Format(time.RFC3339Nano)
	items, err := e.db.CleanupCandidates(r.ID, before)
	if err != nil {
		return err
	}
	for _, o := range items {
		st, err := e.rc.Stat(ctx, o.DestPath)
		if err != nil {
			continue
		}
		if st.Size != o.Size {
			slog.Warn("cleanup skipped modified destination", "path", o.DestPath, "expected", o.Size, "actual", st.Size)
			continue
		}
		if err := e.rc.DeleteFile(ctx, o.DestPath); err != nil {
			slog.Warn("cleanup delete failed", "path", o.DestPath, "err", err)
			continue
		}
		_ = e.db.SetState(o.ID, "cleaned", "")
	}
	return nil
}

func (e *Engine) rule(id string) (config.Rule, bool) {
	for _, r := range e.cfg.Rules {
		if r.ID == id {
			return r, true
		}
	}
	return config.Rule{}, false
}

func (e *Engine) Rules() []config.Rule { return e.cfg.Rules }
func (e *Engine) DB() *db.DB           { return e.db }

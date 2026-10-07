// Package privateering periodically pushes a snapshot of Copyarr's torrents
// and managed files to a Nexus instance (see nexus/worker/privateering). It
// is entirely best-effort: a failed push is logged and never affects
// Copyarr's own scan/transfer loop.
package privateering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Torrent mirrors nexus/worker/privateering/contract.ts PrivateeringTorrent.
type Torrent struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	SizeBytes   int64   `json:"sizeBytes"`
	Completed   bool    `json:"completed"`
	ProgressPct float64 `json:"progressPct"`
	AddedAt     *string `json:"addedAt"`
}

// File mirrors nexus/worker/privateering/contract.ts PrivateeringFile.
type File struct {
	Path        string  `json:"path"`
	SizeBytes   int64   `json:"sizeBytes"`
	Status      string  `json:"status"`
	CommittedAt *string `json:"committedAt"`
}

// Snapshot mirrors nexus/worker/privateering/contract.ts PrivateeringIngestPayload.
type Snapshot struct {
	Torrents     []Torrent `json:"torrents"`
	CopyarrFiles []File    `json:"copyarrFiles"`
}

// SnapshotSource builds the current snapshot to push. Implemented by
// *engine.Engine.
type SnapshotSource interface {
	PrivateeringSnapshot(ctx context.Context) (Snapshot, error)
}

// checkInterval is how often the Pusher wakes up to re-read settings and
// decide whether a push is due. Settings changes made in the UI therefore
// take effect within this window, without a restart.
const checkInterval = 30 * time.Second

// Pusher sends a full snapshot to Nexus on the interval configured in
// Settings, re-reading Settings on every tick so changes made in Copyarr's
// web UI apply without a restart.
type Pusher struct {
	store    *Store
	source   SnapshotSource
	http     *http.Client
	lastPush time.Time
}

// New builds a Pusher backed by the given settings store and snapshot
// source.
func New(store *Store, source SnapshotSource) *Pusher {
	return &Pusher{
		store:  store,
		source: source,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Run checks the configured push interval every checkInterval and pushes a
// snapshot when due, until ctx is cancelled. It is a no-op whenever the
// Nexus URL or token is unset. It never returns an error; failures are
// logged.
func (p *Pusher) Run(ctx context.Context) {
	t := time.NewTicker(checkInterval)
	defer t.Stop()
	for {
		p.maybePush(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (p *Pusher) maybePush(ctx context.Context) {
	settings, err := p.store.Get()
	if err != nil {
		slog.Warn("privateering: load settings failed", "err", err)
		return
	}
	if !settings.Enabled() {
		return
	}
	if !p.lastPush.IsZero() && time.Since(p.lastPush) < settings.Interval() {
		return
	}
	p.lastPush = time.Now()
	p.pushOnce(ctx, settings)
}

// PushNow immediately pushes a snapshot using the current settings,
// regardless of the configured interval. Used by the "send now" / test
// connection action in the UI. Returns an error if settings are incomplete
// or the push fails, so the caller can surface it.
func (p *Pusher) PushNow(ctx context.Context) error {
	settings, err := p.store.Get()
	if err != nil {
		return err
	}
	if !settings.Enabled() {
		return fmt.Errorf("Nexus URL and token must both be set")
	}
	p.lastPush = time.Now()
	return p.push(ctx, settings)
}

func (p *Pusher) pushOnce(ctx context.Context, settings Settings) {
	if err := p.push(ctx, settings); err != nil {
		slog.Warn("privateering: push failed", "err", err)
	}
}

func (p *Pusher) push(ctx context.Context, settings Settings) error {
	snap, err := p.source.PrivateeringSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("build snapshot: %w", err)
	}
	body, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	url := strings.TrimRight(settings.URL, "/") + "/api/privateering/ingest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.Token)
	res, err := p.http.Do(req)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("push rejected: status %d", res.StatusCode)
	}
	slog.Debug("privateering: snapshot pushed", "torrents", len(snap.Torrents), "copyarr_files", len(snap.CopyarrFiles))
	return nil
}

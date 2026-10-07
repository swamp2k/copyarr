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

// Pusher sends a full snapshot to Nexus on a fixed interval.
type Pusher struct {
	url      string
	token    string
	interval time.Duration
	source   SnapshotSource
	http     *http.Client
}

// New builds a Pusher. url must be the Nexus base URL (e.g.
// https://nexus.example); the /api/privateering/ingest path is appended.
func New(url, token string, interval time.Duration, source SnapshotSource) *Pusher {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Pusher{
		url:      strings.TrimRight(url, "/"),
		token:    token,
		interval: interval,
		source:   source,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// Run pushes an initial snapshot and then one every interval, until ctx is
// cancelled. It never returns an error; failures are logged.
func (p *Pusher) Run(ctx context.Context) {
	p.pushOnce(ctx)
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.pushOnce(ctx)
		}
	}
}

func (p *Pusher) pushOnce(ctx context.Context) {
	snap, err := p.source.PrivateeringSnapshot(ctx)
	if err != nil {
		slog.Warn("privateering: build snapshot failed", "err", err)
		return
	}
	body, err := json.Marshal(snap)
	if err != nil {
		slog.Warn("privateering: marshal snapshot failed", "err", err)
		return
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, p.url+"/api/privateering/ingest", bytes.NewReader(body))
	if err != nil {
		slog.Warn("privateering: build request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	res, err := p.http.Do(req)
	if err != nil {
		slog.Warn("privateering: push failed", "err", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		slog.Warn("privateering: push rejected", "status", res.StatusCode, "err", fmt.Errorf("unexpected status"))
		return
	}
	slog.Debug("privateering: snapshot pushed", "torrents", len(snap.Torrents), "copyarr_files", len(snap.CopyarrFiles))
}

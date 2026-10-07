package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
)

func TestPrivateeringSnapshotIncludesObjectsNoRtorrent(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if _, _, err := d.UpsertSeen("r1", "key1", "Show/episode.mkv", 1024, "mod", "now", "committed"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Rules: []config.Rule{{ID: "r1", Enabled: true}}}
	e := New(cfg, d, "test", "test")

	snap, err := e.PrivateeringSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Torrents) != 0 {
		t.Fatalf("expected no torrents without an rtorrent config, got %d", len(snap.Torrents))
	}
	if len(snap.CopyarrFiles) != 1 {
		t.Fatalf("expected 1 copyarr file, got %d", len(snap.CopyarrFiles))
	}
	f := snap.CopyarrFiles[0]
	if f.Path != "Show/episode.mkv" || f.SizeBytes != 1024 || f.Status != "committed" {
		t.Fatalf("unexpected file: %+v", f)
	}
}

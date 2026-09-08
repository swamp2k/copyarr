package engine

import (
	"testing"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
	rt "github.com/swamp2k/copyarr/internal/rtorrent"
)

func TestPathWithinRoot(t *testing.T) {
	cases := []struct {
		rel, root string
		want      bool
	}{
		{"Show/file.mkv", "Show", true},
		{"Show", "Show", true},
		{"Show2/file.mkv", "Show", false},
		{"file.mkv", "file.mkv", true},
	}
	for _, tc := range cases {
		if got := pathWithinRoot(tc.rel, tc.root); got != tc.want {
			t.Fatalf("pathWithinRoot(%q,%q)=%v want %v", tc.rel, tc.root, got, tc.want)
		}
	}
}

func TestTorrentRelativeRoot(t *testing.T) {
	cfg := config.RTorrent{SourceBasePath: "/media/user/complete"}
	tr := rt.Torrent{BasePath: "/media/user/complete/My Show"}
	if got := torrentRelativeRoot(cfg, tr); got != "My Show" {
		t.Fatalf("got %q", got)
	}
}

func TestTorrentJobKeyStableAcrossOrder(t *testing.T) {
	a := []db.JobItem{
		{RelPath: "Show/b.mkv", Size: 2, ModTime: "b"},
		{RelPath: "Show/a.mkv", Size: 1, ModTime: "a"},
	}
	b := []db.JobItem{a[1], a[0]}
	if torrentJobKey("HASH", a) != torrentJobKey("HASH", b) {
		t.Fatal("torrent job key depends on item order")
	}
}

package engine

import (
	"path/filepath"
	"testing"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
)

func TestLogRetentionDefaultsToSevenDays(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	e := New(config.Config{}, d, "test", "test")
	if got := e.LogRetentionDays(); got != 7 {
		t.Fatalf("LogRetentionDays=%d want 7", got)
	}
	if err := e.SetLogRetentionDays(14); err != nil {
		t.Fatal(err)
	}
	if got := e.LogRetentionDays(); got != 14 {
		t.Fatalf("LogRetentionDays=%d want 14", got)
	}
	if err := e.SetLogRetentionDays(0); err == nil {
		t.Fatal("expected zero-day retention to be rejected")
	}
}

package db

import (
	"path/filepath"
	"testing"
)

func TestUpsertSeenIgnoresMtimeJitterForSamePathAndSize(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	first, isNew, err := d.UpsertSeen("r", "key-one", "file.bin", 1234, "2026-09-08T12:00:00Z", "2026-09-08T12:00:01Z", "discovered")
	if err != nil || !isNew {
		t.Fatalf("first insert err=%v isNew=%v", err, isNew)
	}
	if err := d.SetState(first.ID, "done", ""); err != nil {
		t.Fatal(err)
	}

	second, isNew, err := d.UpsertSeen("r", "key-two", "file.bin", 1234, "2026-09-08T12:00:05Z", "2026-09-08T12:10:01Z", "discovered")
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Fatal("mtime jitter created a new generation")
	}
	if second.ID != first.ID {
		t.Fatalf("expected existing object %d, got %d", first.ID, second.ID)
	}
	if second.State != "done" {
		t.Fatalf("expected terminal state to be preserved, got %q", second.State)
	}
}

func TestUpsertSeenCreatesNewGenerationWhenSizeChanges(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	first, _, err := d.UpsertSeen("r", "key-one", "file.bin", 1234, "2026-09-08T12:00:00Z", "2026-09-08T12:00:01Z", "discovered")
	if err != nil {
		t.Fatal(err)
	}
	second, isNew, err := d.UpsertSeen("r", "key-two", "file.bin", 5678, "2026-09-08T12:00:05Z", "2026-09-08T12:10:01Z", "discovered")
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Fatal("size change did not create a new generation")
	}
	if second.ID == first.ID {
		t.Fatal("size change reused old generation")
	}
}

package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func insertTestJob(t *testing.T, d *DB, state string, attempts int) int64 {
	t.Helper()
	ts := now()
	res, err := d.Exec(`INSERT INTO jobs(rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,attempts,created_at,updated_at)
VALUES('r', ?, 'file', 'test', 'test.bin', ?, 'test', 1, 1, ?, ?, ?)`,
		ts, state, attempts, ts, ts)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestScheduledRetryPromotesOnlyWhenDue(t *testing.T) {
	d := openTestDB(t)
	id := insertTestJob(t, d, "copying", 1)

	future := time.Now().UTC().Add(30 * time.Minute)
	if err := d.FailJob(id, "boom", &future); err != nil {
		t.Fatal(err)
	}
	if err := d.PromoteDueRetries(); err != nil {
		t.Fatal(err)
	}

	var state string
	var next *string
	if err := d.QueryRow(`SELECT state,next_retry_at FROM jobs WHERE id=?`, id).Scan(&state, &next); err != nil {
		t.Fatal(err)
	}
	if state != "retry_wait" || next == nil {
		t.Fatalf("state=%q next=%v want retry_wait with timestamp", state, next)
	}

	past := retryTimestamp(time.Now().UTC().Add(-time.Second))
	if _, err := d.Exec(`UPDATE jobs SET next_retry_at=? WHERE id=?`, past, id); err != nil {
		t.Fatal(err)
	}
	if err := d.PromoteDueRetries(); err != nil {
		t.Fatal(err)
	}
	if err := d.QueryRow(`SELECT state FROM jobs WHERE id=?`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "queued" {
		t.Fatalf("state=%q want queued", state)
	}
}

func TestManualRetryResetsAttemptBudget(t *testing.T) {
	d := openTestDB(t)
	id := insertTestJob(t, d, "failed", 99)
	if err := d.RetryJob(id); err != nil {
		t.Fatal(err)
	}
	var state string
	var attempts int
	if err := d.QueryRow(`SELECT state,attempts FROM jobs WHERE id=?`, id).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "queued" || attempts != 0 {
		t.Fatalf("state=%q attempts=%d want queued/0", state, attempts)
	}
}

func TestOpenMigratesExistingJobsTable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`CREATE TABLE jobs(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_id TEXT NOT NULL,
		job_key TEXT NOT NULL,
		kind TEXT NOT NULL,
		display_name TEXT NOT NULL,
		rel_root TEXT NOT NULL,
		state TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		total_bytes INTEGER NOT NULL DEFAULT 0,
		item_count INTEGER NOT NULL DEFAULT 0,
		attempts INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		started_at TEXT,
		completed_at TEXT,
		dest_path TEXT NOT NULL DEFAULT '',
		UNIQUE(rule_id,job_key)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	d, err := Open(p)
	if err != nil {
		t.Fatalf("Open old schema: %v", err)
	}
	defer d.Close()

	rows, err := d.Query("PRAGMA table_info(jobs)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var def any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "next_retry_at" {
			found = true
		}
	}
	if !found {
		t.Fatal("next_retry_at column was not added")
	}
}

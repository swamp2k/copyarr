package db

import (
	"testing"
	"time"
)

func TestDeleteLogsBefore(t *testing.T) {
	d := openTestDB(t)
	old := time.Now().UTC().Add(-8 * 24 * time.Hour).Format(time.RFC3339Nano)
	fresh := time.Now().UTC().Add(-6 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := d.Exec(`INSERT INTO logs(ts,level,source,message,fields_json) VALUES(?,?,?,?,''),(?,?,?,?, '')`,
		old, "INFO", "test", "old", fresh, "INFO", "test", "fresh"); err != nil {
		t.Fatal(err)
	}
	deleted, err := d.DeleteLogsBefore(time.Now().UTC().Add(-7 * 24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d want 1", deleted)
	}
	logs, err := d.ListLogs(10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Message != "fresh" {
		t.Fatalf("remaining logs=%+v", logs)
	}
}

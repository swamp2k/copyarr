package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

type Object struct {
	ID          int64   `json:"id"`
	RuleID      string  `json:"rule_id"`
	ObjectKey   string  `json:"object_key"`
	RelPath     string  `json:"rel_path"`
	Size        int64   `json:"size"`
	ModTime     string  `json:"mod_time"`
	State       string  `json:"state"`
	FirstSeen   string  `json:"first_seen"`
	LastSeen    string  `json:"last_seen"`
	StableSince string  `json:"stable_since"`
	Attempts    int     `json:"attempts"`
	LastError   string  `json:"last_error"`
	CompletedAt *string `json:"completed_at,omitempty"`
	DestPath    string  `json:"dest_path"`
}

func Open(path string) (*DB, error) {
	s, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	d := &DB{s}
	if err = d.migrate(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate() error {
	_, err := d.Exec(`PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS objects(
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 rule_id TEXT NOT NULL,
 object_key TEXT NOT NULL,
 rel_path TEXT NOT NULL,
 size INTEGER NOT NULL,
 mod_time TEXT NOT NULL,
 state TEXT NOT NULL,
 first_seen TEXT NOT NULL,
 last_seen TEXT NOT NULL,
 stable_since TEXT NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0,
 last_error TEXT NOT NULL DEFAULT '',
 completed_at TEXT,
 dest_path TEXT NOT NULL DEFAULT '',
 UNIQUE(rule_id,object_key)
);
CREATE INDEX IF NOT EXISTS idx_objects_state ON objects(state,id);`)
	return err
}

func (d *DB) Meta(key string) (string, bool, error) {
	var v string
	err := d.QueryRow(`SELECT value FROM meta WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

func (d *DB) SetMeta(key, v string) error {
	_, e := d.Exec(`INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, v)
	return e
}

func (d *DB) UpsertSeen(rule, key, rel string, size int64, mod, timeNow, state string) (Object, bool, error) {
	var o Object
	err := d.QueryRow(`SELECT id,rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since,attempts,last_error,completed_at,dest_path
FROM objects WHERE rule_id=? AND object_key=?`, rule, key).Scan(
		&o.ID, &o.RuleID, &o.ObjectKey, &o.RelPath, &o.Size, &o.ModTime, &o.State,
		&o.FirstSeen, &o.LastSeen, &o.StableSince, &o.Attempts, &o.LastError, &o.CompletedAt, &o.DestPath,
	)
	if err == sql.ErrNoRows {
		_, e := d.Exec(`INSERT INTO objects(rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since)
VALUES(?,?,?,?,?,?,?,?,?)`, rule, key, rel, size, mod, state, timeNow, timeNow, timeNow)
		return o, true, e
	}
	if err != nil {
		return o, false, err
	}
	_, err = d.Exec(`UPDATE objects SET last_seen=? WHERE id=?`, timeNow, o.ID)
	return o, false, err
}

func (d *DB) SetState(id int64, state, errMsg string) error {
	_, e := d.Exec(`UPDATE objects SET state=?,last_error=? WHERE id=?`, state, errMsg, id)
	return e
}

func (d *DB) SetStateByKey(rule, key, state, msg string) error {
	_, e := d.Exec(`UPDATE objects SET state=?,last_error=? WHERE rule_id=? AND object_key=?`, state, msg, rule, key)
	return e
}

func (d *DB) IDByKey(rule, key string) (int64, error) {
	var id int64
	err := d.QueryRow(`SELECT id FROM objects WHERE rule_id=? AND object_key=?`, rule, key).Scan(&id)
	return id, err
}

func (d *DB) Queue(id int64) error {
	_, e := d.Exec(`UPDATE objects SET state='queued',last_error='' WHERE id=? AND state IN ('discovered','retry_wait')`, id)
	return e
}

func (d *DB) NextQueued() (Object, error) {
	var o Object
	err := d.QueryRow(`SELECT id,rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since,attempts,last_error,completed_at,dest_path
FROM objects WHERE state='queued' ORDER BY id LIMIT 1`).Scan(
		&o.ID, &o.RuleID, &o.ObjectKey, &o.RelPath, &o.Size, &o.ModTime, &o.State,
		&o.FirstSeen, &o.LastSeen, &o.StableSince, &o.Attempts, &o.LastError, &o.CompletedAt, &o.DestPath,
	)
	return o, err
}

func (d *DB) Start(id int64) error {
	_, e := d.Exec(`UPDATE objects SET state='copying',attempts=attempts+1,last_error='' WHERE id=?`, id)
	return e
}

func (d *DB) Complete(id int64, dest string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, e := d.Exec(`UPDATE objects SET state='done',completed_at=?,dest_path=?,last_error='' WHERE id=?`, now, dest, id)
	return e
}

func (d *DB) Fail(id int64, msg string) error {
	_, e := d.Exec(`UPDATE objects SET state='retry_wait',last_error=? WHERE id=?`, msg, id)
	return e
}

func (d *DB) List(limit int) ([]Object, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := d.Query(`SELECT id,rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since,attempts,last_error,completed_at,dest_path
FROM objects ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var o Object
		if err := rows.Scan(&o.ID, &o.RuleID, &o.ObjectKey, &o.RelPath, &o.Size, &o.ModTime, &o.State,
			&o.FirstSeen, &o.LastSeen, &o.StableSince, &o.Attempts, &o.LastError, &o.CompletedAt, &o.DestPath); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (d *DB) CleanupCandidates(rule, before string) ([]Object, error) {
	rows, err := d.Query(`SELECT id,rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since,attempts,last_error,completed_at,dest_path
FROM objects WHERE rule_id=? AND state='done' AND completed_at IS NOT NULL AND completed_at<? AND dest_path<>''`, rule, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var o Object
		if err := rows.Scan(&o.ID, &o.RuleID, &o.ObjectKey, &o.RelPath, &o.Size, &o.ModTime, &o.State,
			&o.FirstSeen, &o.LastSeen, &o.StableSince, &o.Attempts, &o.LastError, &o.CompletedAt, &o.DestPath); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func IsNoRows(err error) bool { return err == sql.ErrNoRows }

func (o Object) String() string { return fmt.Sprintf("%s:%s", o.RuleID, o.RelPath) }

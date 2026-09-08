package db

import (
	"database/sql"
	"fmt"
	"strings"
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

type Job struct {
	ID          int64   `json:"id"`
	RuleID      string  `json:"rule_id"`
	JobKey      string  `json:"job_key"`
	Kind        string  `json:"kind"`
	DisplayName string  `json:"display_name"`
	RelRoot     string  `json:"rel_root"`
	State       string  `json:"state"`
	Reason      string  `json:"reason"`
	TotalBytes  int64   `json:"total_bytes"`
	ItemCount   int     `json:"item_count"`
	Attempts    int     `json:"attempts"`
	LastError   string  `json:"last_error"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	DestPath    string  `json:"dest_path"`
}

type JobItem struct {
	ObjectID int64  `json:"object_id"`
	RelPath  string `json:"rel_path"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time"`
}

type QueueStats struct {
	Jobs  int   `json:"jobs"`
	Bytes int64 `json:"bytes"`
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
	if err = d.RecoverInterrupted(); err != nil {
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
CREATE INDEX IF NOT EXISTS idx_objects_state ON objects(state,id);
CREATE INDEX IF NOT EXISTS idx_objects_rule_path ON objects(rule_id,rel_path);

CREATE TABLE IF NOT EXISTS jobs(
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
);
CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state,id);

CREATE TABLE IF NOT EXISTS job_items(
 job_id INTEGER NOT NULL,
 object_id INTEGER NOT NULL,
 rel_path TEXT NOT NULL,
 size INTEGER NOT NULL,
 mod_time TEXT NOT NULL,
 PRIMARY KEY(job_id,object_id),
 FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE,
 FOREIGN KEY(object_id) REFERENCES objects(id)
);
CREATE INDEX IF NOT EXISTS idx_job_items_object ON job_items(object_id);
`)
	return err
}

func (d *DB) RecoverInterrupted() error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE jobs SET state='queued', updated_at=? WHERE state='copying'`, now()); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE objects SET state='discovered', last_error='' 
WHERE state IN ('queued','copying','retry_wait')
AND id NOT IN (SELECT object_id FROM job_items)`); err != nil {
		return err
	}
	return tx.Commit()
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

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
		tx, e := d.Begin()
		if e != nil {
			return o, false, e
		}
		defer tx.Rollback()

		// A growing/changing upload creates a new generation. Older unprocessed
		// generations for the same path must never become "stable" later.
		if _, e = tx.Exec(`UPDATE objects SET state='superseded',last_error=''
WHERE rule_id=? AND rel_path=? AND object_key<>? AND state IN ('discovered','retry_wait')`, rule, rel, key); e != nil {
			return o, false, e
		}

		res, e := tx.Exec(`INSERT INTO objects(rule_id,object_key,rel_path,size,mod_time,state,first_seen,last_seen,stable_since)
VALUES(?,?,?,?,?,?,?,?,?)`, rule, key, rel, size, mod, state, timeNow, timeNow, timeNow)
		if e != nil {
			return o, false, e
		}
		id, e := res.LastInsertId()
		if e != nil {
			return o, false, e
		}
		if e = tx.Commit(); e != nil {
			return o, false, e
		}
		o = Object{
			ID: id, RuleID: rule, ObjectKey: key, RelPath: rel, Size: size, ModTime: mod,
			State: state, FirstSeen: timeNow, LastSeen: timeNow, StableSince: timeNow,
		}
		return o, true, nil
	}
	if err != nil {
		return o, false, err
	}
	_, err = d.Exec(`UPDATE objects SET last_seen=? WHERE id=?`, timeNow, o.ID)
	o.LastSeen = timeNow
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

func (d *DB) CreateJob(rule, key, kind, displayName, relRoot, reason string, items []JobItem) (Job, bool, error) {
	var j Job
	err := d.QueryRow(`SELECT id,rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,attempts,last_error,created_at,started_at,completed_at,dest_path
FROM jobs WHERE rule_id=? AND job_key=?`, rule, key).Scan(
		&j.ID, &j.RuleID, &j.JobKey, &j.Kind, &j.DisplayName, &j.RelRoot, &j.State, &j.Reason,
		&j.TotalBytes, &j.ItemCount, &j.Attempts, &j.LastError, &j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.DestPath,
	)
	if err == nil {
		return j, false, nil
	}
	if err != sql.ErrNoRows {
		return j, false, err
	}
	if len(items) == 0 {
		return j, false, fmt.Errorf("cannot create empty job")
	}

	var total int64
	for _, item := range items {
		total += item.Size
	}
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return j, false, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO jobs(rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,created_at,updated_at)
VALUES(?,?,?,?,?,'queued',?,?,?,?,?)`, rule, key, kind, displayName, relRoot, reason, total, len(items), ts, ts)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return d.jobByKey(rule, key)
		}
		return j, false, err
	}
	jobID, err := res.LastInsertId()
	if err != nil {
		return j, false, err
	}
	for _, item := range items {
		if _, err = tx.Exec(`INSERT INTO job_items(job_id,object_id,rel_path,size,mod_time) VALUES(?,?,?,?,?)`,
			jobID, item.ObjectID, item.RelPath, item.Size, item.ModTime); err != nil {
			return j, false, err
		}
		if _, err = tx.Exec(`UPDATE objects SET state='queued',last_error='' WHERE id=? AND state IN ('discovered','retry_wait')`, item.ObjectID); err != nil {
			return j, false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return j, false, err
	}
	j = Job{
		ID: jobID, RuleID: rule, JobKey: key, Kind: kind, DisplayName: displayName,
		RelRoot: relRoot, State: "queued", Reason: reason, TotalBytes: total,
		ItemCount: len(items), CreatedAt: ts,
	}
	return j, true, nil
}

func (d *DB) jobByKey(rule, key string) (Job, bool, error) {
	var j Job
	err := d.QueryRow(`SELECT id,rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,attempts,last_error,created_at,started_at,completed_at,dest_path
FROM jobs WHERE rule_id=? AND job_key=?`, rule, key).Scan(
		&j.ID, &j.RuleID, &j.JobKey, &j.Kind, &j.DisplayName, &j.RelRoot, &j.State, &j.Reason,
		&j.TotalBytes, &j.ItemCount, &j.Attempts, &j.LastError, &j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.DestPath,
	)
	if err == sql.ErrNoRows {
		return j, false, nil
	}
	return j, err == nil, err
}

func (d *DB) NextQueuedJob() (Job, error) {
	var j Job
	err := d.QueryRow(`SELECT id,rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,attempts,last_error,created_at,started_at,completed_at,dest_path
FROM jobs WHERE state='queued' ORDER BY id LIMIT 1`).Scan(
		&j.ID, &j.RuleID, &j.JobKey, &j.Kind, &j.DisplayName, &j.RelRoot, &j.State, &j.Reason,
		&j.TotalBytes, &j.ItemCount, &j.Attempts, &j.LastError, &j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.DestPath,
	)
	return j, err
}

func (d *DB) JobItems(jobID int64) ([]JobItem, error) {
	rows, err := d.Query(`SELECT object_id,rel_path,size,mod_time FROM job_items WHERE job_id=? ORDER BY rel_path`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobItem
	for rows.Next() {
		var i JobItem
		if err := rows.Scan(&i.ObjectID, &i.RelPath, &i.Size, &i.ModTime); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (d *DB) StartJob(id int64) error {
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE jobs SET state='copying',attempts=attempts+1,started_at=?,updated_at=?,last_error='' WHERE id=?`, ts, ts, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE objects SET state='copying',attempts=attempts+1,last_error='' WHERE id IN (SELECT object_id FROM job_items WHERE job_id=?)`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) CompleteObject(id int64, dest string) error {
	ts := now()
	_, e := d.Exec(`UPDATE objects SET state='done',completed_at=?,dest_path=?,last_error='' WHERE id=?`, ts, dest, id)
	return e
}

func (d *DB) CompleteJob(id int64, dest string) error {
	ts := now()
	_, e := d.Exec(`UPDATE jobs SET state='done',completed_at=?,updated_at=?,dest_path=?,last_error='' WHERE id=?`, ts, ts, dest, id)
	return e
}

func (d *DB) FailJob(id int64, msg string) error {
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE jobs SET state='retry_wait',last_error=?,updated_at=? WHERE id=?`, msg, ts, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE objects SET state='retry_wait',last_error=? WHERE id IN (SELECT object_id FROM job_items WHERE job_id=?)`, msg, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) RequeueJob(id int64) error {
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE jobs SET state='queued',updated_at=?,last_error='' WHERE id=? AND state IN ('retry_wait','paused')`, ts, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("job %d is not retryable/resumable", id)
	}
	if _, err = tx.Exec(`UPDATE objects SET state='queued',last_error='' WHERE id IN (SELECT object_id FROM job_items WHERE job_id=?) AND state IN ('retry_wait','paused')`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) PauseJob(id int64) error {
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE jobs SET state='paused',updated_at=? WHERE id=? AND state IN ('queued','retry_wait')`, ts, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("job %d cannot be paused in its current state", id)
	}
	if _, err = tx.Exec(`UPDATE objects SET state='paused' WHERE id IN (SELECT object_id FROM job_items WHERE job_id=?) AND state IN ('queued','retry_wait')`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) CancelJob(id int64) error {
	ts := now()
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE jobs SET state='cancelled',updated_at=? WHERE id=? AND state IN ('queued','retry_wait','paused')`, ts, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("job %d cannot be cancelled in its current state", id)
	}
	if _, err = tx.Exec(`UPDATE objects SET state='cancelled' WHERE id IN (SELECT object_id FROM job_items WHERE job_id=?) AND state IN ('queued','retry_wait','paused')`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) QueueStats() (QueueStats, error) {
	var s QueueStats
	err := d.QueryRow(`SELECT COUNT(*),COALESCE(SUM(total_bytes),0) FROM jobs WHERE state='queued'`).Scan(&s.Jobs, &s.Bytes)
	return s, err
}

func (d *DB) JobCounts() (map[string]int, error) {
	rows, err := d.Query(`SELECT state,COUNT(*) FROM jobs GROUP BY state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		out[state] = count
	}
	return out, rows.Err()
}

func (d *DB) ListJobs(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.Query(`SELECT id,rule_id,job_key,kind,display_name,rel_root,state,reason,total_bytes,item_count,attempts,last_error,created_at,started_at,completed_at,dest_path
FROM jobs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Job, 0)
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.RuleID, &j.JobKey, &j.Kind, &j.DisplayName, &j.RelRoot, &j.State, &j.Reason,
			&j.TotalBytes, &j.ItemCount, &j.Attempts, &j.LastError, &j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.DestPath); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
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
	out := make([]Object, 0)
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

package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
	rc "github.com/swamp2k/copyarr/internal/rclone"
	rt "github.com/swamp2k/copyarr/internal/rtorrent"
)

type ActiveTransfer struct {
	JobID            int64   `json:"job_id"`
	RuleID           string  `json:"rule_id"`
	Name             string  `json:"name"`
	Kind             string  `json:"kind"`
	Path             string  `json:"path"`
	TotalBytes       int64   `json:"total_bytes"`
	TransferredBytes int64   `json:"transferred_bytes"`
	ProgressPercent  float64 `json:"progress_percent"`
	SpeedBps         float64 `json:"speed_bps"`
	ETASeconds       *int64  `json:"eta_seconds,omitempty"`
	StartedAt        string  `json:"started_at"`
}

type Status struct {
	Service  string          `json:"service"`
	Scanning bool            `json:"scanning"`
	Active   *ActiveTransfer `json:"active,omitempty"`
	Queue    db.QueueStats   `json:"queue"`
	Jobs     map[string]int  `json:"jobs"`
}

type Engine struct {
	cfg      config.Config
	db       *db.DB
	rc       rc.Client
	wake     chan struct{}
	mu       sync.Mutex
	scanning bool
	active   *ActiveTransfer
}

type seenItem struct {
	item rc.Item
	obj  db.Object
}

func New(cfg config.Config, d *db.DB) *Engine {
	return &Engine{
		cfg:  cfg,
		db:   d,
		rc:   rc.Client{ConfigPath: cfg.RcloneConfig},
		wake: make(chan struct{}, 1),
	}
}

func (e *Engine) Run(ctx context.Context) {
	go e.worker(ctx)
	go e.scheduler(ctx)
}

func (e *Engine) TriggerScan() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

func (e *Engine) scheduler(ctx context.Context) {
	e.scanAll(ctx)
	t := time.NewTicker(e.cfg.ScanInterval())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.scanAll(ctx)
		case <-e.wake:
			e.scanAll(ctx)
		}
	}
}

func objectKey(rel string, size int64, mod time.Time) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s", rel, size, mod.UTC().Format(time.RFC3339Nano))))
	return hex.EncodeToString(h[:])
}

func (e *Engine) scanAll(ctx context.Context) {
	e.mu.Lock()
	if e.scanning {
		e.mu.Unlock()
		return
	}
	e.scanning = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.scanning = false
		e.mu.Unlock()
	}()

	for _, r := range e.cfg.Rules {
		if !r.Enabled {
			continue
		}
		if err := e.scanRule(ctx, r); err != nil {
			slog.Error("scan failed", "rule", r.ID, "err", err)
		}
		if err := e.cleanupRule(ctx, r); err != nil {
			slog.Error("cleanup failed", "rule", r.ID, "err", err)
		}
	}
}

func (e *Engine) scanRule(ctx context.Context, r config.Rule) error {
	items, err := e.rc.ListFiles(ctx, r.Source)
	if err != nil {
		return err
	}
	metaKey := "rule:" + r.ID + ":initialized"
	_, initialized, err := e.db.Meta(metaKey)
	if err != nil {
		return err
	}

	var torrents []rt.Torrent
	rtOK := false
	if r.RTorrent != nil {
		torrents, err = rt.New(*r.RTorrent).Torrents(ctx)
		if err != nil {
			if r.RTorrent.Required {
				return fmt.Errorf("rtorrent required: %w", err)
			}
			slog.Warn("rtorrent unavailable, using stability fallback", "rule", r.ID, "err", err)
		} else {
			rtOK = true
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	seen := make([]seenItem, 0, len(items))
	for _, it := range items {
		if it.IsDir {
			continue
		}
		k := objectKey(it.Path, it.Size, it.ModTime)
		o, isNew, err := e.db.UpsertSeen(r.ID, k, it.Path, it.Size, it.ModTime.UTC().Format(time.RFC3339Nano), now, "discovered")
		if err != nil {
			return err
		}
		if !initialized && r.InitialBehavior == "ignore_existing" {
			if isNew {
				if err := e.db.SetStateByKey(r.ID, k, "ignored", ""); err != nil {
					return err
				}
			}
			continue
		}
		seen = append(seen, seenItem{item: it, obj: o})
	}

	if !initialized {
		if err := e.db.SetMeta(metaKey, "1"); err != nil {
			return err
		}
		slog.Info("rule initialized", "rule", r.ID, "objects", len(items), "behavior", r.InitialBehavior)
		return nil
	}

	assigned := map[string]bool{}

	// Completed rTorrent payloads are queued as one persistent job. Incomplete
	// torrents are deliberately not allowed to fall through to stability.
	if rtOK && r.RTorrent != nil {
		for _, t := range torrents {
			if !t.Complete {
				continue
			}
			root := torrentRelativeRoot(*r.RTorrent, t)
			if root == "" {
				continue
			}
			group := matchingSeen(root, seen)
			if len(group) == 0 || !allProcessable(group) {
				continue
			}
			jobItems := make([]db.JobItem, 0, len(group))
			for _, s := range group {
				jobItems = append(jobItems, db.JobItem{
					ObjectID: s.obj.ID,
					RelPath:  s.item.Path,
					Size:     s.item.Size,
					ModTime:  s.item.ModTime.UTC().Format(time.RFC3339Nano),
				})
			}
			key := torrentJobKey(t.Hash, jobItems)
			display := t.Name
			if display == "" {
				display = path.Base(root)
			}
			job, created, err := e.db.CreateJob(r.ID, key, "torrent", display, root, "rtorrent_complete", jobItems)
			if err != nil {
				return err
			}
			if created {
				slog.Info("queued torrent job", "rule", r.ID, "job", job.ID, "name", display, "items", len(jobItems), "bytes", job.TotalBytes)
			}
			for _, s := range group {
				assigned[s.item.Path] = true
			}
		}
	}

	for _, s := range seen {
		if assigned[s.item.Path] || !processable(s.obj.State) {
			continue
		}

		if rtOK && r.RTorrent != nil {
			if t, _, ok := torrentForPath(s.item.Path, *r.RTorrent, torrents); ok {
				// If rTorrent knows this path, readiness belongs to rTorrent.
				// This includes incomplete torrents and completed torrents whose
				// grouped job already exists or cannot safely be reconstructed.
				_ = t
				continue
			}
		}

		stableSince := s.obj.StableSince
		t, parseErr := time.Parse(time.RFC3339Nano, stableSince)
		if parseErr != nil || time.Since(t) < time.Duration(r.StabilitySeconds)*time.Second {
			continue
		}
		jobItem := db.JobItem{
			ObjectID: s.obj.ID,
			RelPath:  s.item.Path,
			Size:     s.item.Size,
			ModTime:  s.item.ModTime.UTC().Format(time.RFC3339Nano),
		}
		job, created, err := e.db.CreateJob(r.ID, "file:"+s.obj.ObjectKey, "file", path.Base(s.item.Path), s.item.Path, "stable", []db.JobItem{jobItem})
		if err != nil {
			return err
		}
		if created {
			slog.Info("queued file job", "rule", r.ID, "job", job.ID, "path", s.item.Path, "reason", "stable", "size", s.item.Size)
		}
	}
	return nil
}

func processable(state string) bool {
	return state == "discovered" || state == "retry_wait"
}

func allProcessable(items []seenItem) bool {
	for _, s := range items {
		if !processable(s.obj.State) {
			return false
		}
	}
	return true
}

func matchingSeen(root string, seen []seenItem) []seenItem {
	var out []seenItem
	for _, s := range seen {
		if pathWithinRoot(s.item.Path, root) {
			out = append(out, s)
		}
	}
	return out
}

func pathWithinRoot(rel, root string) bool {
	root = strings.Trim(root, "/")
	rel = strings.Trim(rel, "/")
	return rel == root || strings.HasPrefix(rel, root+"/")
}

func torrentRelativeRoot(c config.RTorrent, t rt.Torrent) string {
	p := strings.TrimSpace(t.BasePath)
	base := strings.TrimSuffix(strings.TrimSpace(c.SourceBasePath), "/")
	if base != "" {
		if p == base {
			return path.Base(p)
		}
		prefix := base + "/"
		if strings.HasPrefix(p, prefix) {
			p = strings.TrimPrefix(p, prefix)
		}
	}
	return strings.Trim(p, "/")
}

func torrentForPath(rel string, c config.RTorrent, torrents []rt.Torrent) (rt.Torrent, string, bool) {
	// Prefer the most specific root if paths happen to nest.
	type candidate struct {
		t    rt.Torrent
		root string
	}
	var matches []candidate
	for _, t := range torrents {
		root := torrentRelativeRoot(c, t)
		if root != "" && pathWithinRoot(rel, root) {
			matches = append(matches, candidate{t: t, root: root})
		}
	}
	if len(matches) == 0 {
		return rt.Torrent{}, "", false
	}
	sort.Slice(matches, func(i, j int) bool { return len(matches[i].root) > len(matches[j].root) })
	return matches[0].t, matches[0].root, true
}

func torrentJobKey(hash string, items []db.JobItem) string {
	cp := append([]db.JobItem(nil), items...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].RelPath < cp[j].RelPath })
	h := sha256.New()
	_, _ = h.Write([]byte(hash))
	for _, item := range cp {
		_, _ = fmt.Fprintf(h, "\x00%s\x00%d\x00%s", item.RelPath, item.Size, item.ModTime)
	}
	return "torrent:" + hash + ":" + hex.EncodeToString(h.Sum(nil))
}

func (e *Engine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, err := e.db.NextQueuedJob()
		if db.IsNoRows(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		if err != nil {
			slog.Error("queue read failed", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}
		r, ok := e.rule(job.RuleID)
		if !ok {
			_ = e.db.FailJob(job.ID, "rule no longer exists")
			continue
		}
		items, err := e.db.JobItems(job.ID)
		if err != nil {
			_ = e.db.FailJob(job.ID, err.Error())
			continue
		}
		if err := e.transferJob(ctx, r, job, items); err != nil {
			slog.Error("transfer failed", "job", job.ID, "name", job.DisplayName, "err", err)
			_ = e.db.FailJob(job.ID, err.Error())
			time.Sleep(5 * time.Second)
			_ = e.db.RequeueJob(job.ID)
		}
	}
}

func (e *Engine) transferJob(ctx context.Context, r config.Rule, job db.Job, items []db.JobItem) error {
	if len(items) == 0 {
		return fmt.Errorf("job has no items")
	}
	if err := e.db.StartJob(job.ID); err != nil {
		return err
	}

	isDir := job.Kind == "torrent" && (len(items) > 1 || items[0].RelPath != job.RelRoot)
	src := rc.Target(r.Source, job.RelRoot)
	final := rc.Target(r.Destination, job.RelRoot)
	stageRel := path.Join(".copyarr-staging", strconv.FormatInt(job.ID, 10), job.RelRoot)
	stage := rc.Target(r.Destination, stageRel)

	e.beginActive(job)
	stopProgress := e.monitorProgress(ctx, stage, job.TotalBytes)
	defer func() {
		stopProgress()
		e.clearActive(job.ID)
	}()

	slog.Info("copying job", "job", job.ID, "kind", job.Kind, "name", job.DisplayName, "source", src, "stage", stage, "items", len(items), "bytes", job.TotalBytes)
	if isDir {
		if err := e.rc.CopyDir(ctx, src, stage, r.RcloneArgs); err != nil {
			return err
		}
	} else {
		if err := e.rc.CopyTo(ctx, src, stage, r.RcloneArgs); err != nil {
			return err
		}
	}

	if err := e.verifyManifest(ctx, stage, job.RelRoot, items, isDir); err != nil {
		return fmt.Errorf("verify staging: %w", err)
	}
	if err := e.rc.MoveTo(ctx, stage, final); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	if err := e.verifyManifest(ctx, final, job.RelRoot, items, isDir); err != nil {
		return fmt.Errorf("verify committed: %w", err)
	}

	for _, item := range items {
		if err := e.db.CompleteObject(item.ObjectID, rc.Target(r.Destination, item.RelPath)); err != nil {
			return err
		}
	}
	if r.Mode == "move" {
		if err := e.rc.DeleteFile(ctx, src); err != nil && !isDir {
			return fmt.Errorf("destination committed but source delete failed: %w", err)
		}
		if isDir {
			// rclone purge is intentionally not used in the MVP; directory move
			// sources need an explicit safe implementation before enabling it.
			return fmt.Errorf("move mode for directory jobs is not yet supported safely")
		}
	}
	e.updateActive(job.ID, job.TotalBytes, 0)
	if err := e.db.CompleteJob(job.ID, final); err != nil {
		return err
	}
	slog.Info("job completed", "job", job.ID, "name", job.DisplayName, "items", len(items), "bytes", job.TotalBytes, "destination", final)
	return nil
}

func (e *Engine) verifyManifest(ctx context.Context, target, relRoot string, expected []db.JobItem, isDir bool) error {
	if !isDir {
		st, err := e.rc.Stat(ctx, target)
		if err != nil {
			return err
		}
		if st.Size != expected[0].Size {
			return fmt.Errorf("size mismatch: expected=%d actual=%d", expected[0].Size, st.Size)
		}
		return nil
	}

	actual, err := e.rc.ListTargetFiles(ctx, target)
	if err != nil {
		return err
	}
	want := make(map[string]int64, len(expected))
	for _, item := range expected {
		rel := strings.TrimPrefix(item.RelPath, strings.TrimSuffix(relRoot, "/")+"/")
		want[rel] = item.Size
	}
	if len(actual) != len(want) {
		return fmt.Errorf("manifest count mismatch: expected=%d actual=%d", len(want), len(actual))
	}
	for _, item := range actual {
		size, ok := want[item.Path]
		if !ok {
			return fmt.Errorf("unexpected file in payload: %s", item.Path)
		}
		if size != item.Size {
			return fmt.Errorf("size mismatch for %s: expected=%d actual=%d", item.Path, size, item.Size)
		}
		delete(want, item.Path)
	}
	if len(want) != 0 {
		return fmt.Errorf("manifest missing %d files", len(want))
	}
	return nil
}

func (e *Engine) beginActive(job db.Job) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = &ActiveTransfer{
		JobID: job.ID, RuleID: job.RuleID, Name: job.DisplayName, Kind: job.Kind,
		Path: job.RelRoot, TotalBytes: job.TotalBytes, StartedAt: now,
	}
}

func (e *Engine) monitorProgress(ctx context.Context, target string, total int64) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var prevBytes int64
		prevAt := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				bytes, err := e.rc.TargetBytes(ctx, target)
				if err != nil {
					continue
				}
				now := time.Now()
				elapsed := now.Sub(prevAt).Seconds()
				speed := float64(0)
				if elapsed > 0 && bytes >= prevBytes {
					speed = float64(bytes-prevBytes) / elapsed
				}
				e.mu.Lock()
				if e.active != nil {
					e.active.TransferredBytes = bytes
					if total > 0 {
						e.active.ProgressPercent = float64(bytes) * 100 / float64(total)
						if e.active.ProgressPercent > 100 {
							e.active.ProgressPercent = 100
						}
					}
					e.active.SpeedBps = speed
					e.active.ETASeconds = nil
					if speed > 0 && bytes < total {
						eta := int64(float64(total-bytes) / speed)
						e.active.ETASeconds = &eta
					}
				}
				e.mu.Unlock()
				prevBytes = bytes
				prevAt = now
			}
		}
	}()
	return func() {
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
}

func (e *Engine) updateActive(jobID, bytes int64, speed float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active == nil || e.active.JobID != jobID {
		return
	}
	e.active.TransferredBytes = bytes
	e.active.ProgressPercent = 100
	e.active.SpeedBps = speed
	e.active.ETASeconds = nil
}

func (e *Engine) clearActive(jobID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active != nil && e.active.JobID == jobID {
		e.active = nil
	}
}

func (e *Engine) Status() (Status, error) {
	queue, err := e.db.QueueStats()
	if err != nil {
		return Status{}, err
	}
	counts, err := e.db.JobCounts()
	if err != nil {
		return Status{}, err
	}
	e.mu.Lock()
	scanning := e.scanning
	var active *ActiveTransfer
	if e.active != nil {
		cp := *e.active
		active = &cp
	}
	e.mu.Unlock()
	return Status{Service: "copyarr", Scanning: scanning, Active: active, Queue: queue, Jobs: counts}, nil
}

func (e *Engine) cleanupRule(ctx context.Context, r config.Rule) error {
	if r.CleanupDays <= 0 {
		return nil
	}
	before := time.Now().UTC().Add(-time.Duration(r.CleanupDays) * 24 * time.Hour).Format(time.RFC3339Nano)
	items, err := e.db.CleanupCandidates(r.ID, before)
	if err != nil {
		return err
	}
	for _, o := range items {
		st, err := e.rc.Stat(ctx, o.DestPath)
		if err != nil {
			continue
		}
		if st.Size != o.Size {
			slog.Warn("cleanup skipped modified destination", "path", o.DestPath, "expected", o.Size, "actual", st.Size)
			continue
		}
		if err := e.rc.DeleteFile(ctx, o.DestPath); err != nil {
			slog.Warn("cleanup delete failed", "path", o.DestPath, "err", err)
			continue
		}
		_ = e.db.SetState(o.ID, "cleaned", "")
	}
	return nil
}

func (e *Engine) rule(id string) (config.Rule, bool) {
	for _, r := range e.cfg.Rules {
		if r.ID == id {
			return r, true
		}
	}
	return config.Rule{}, false
}

func (e *Engine) Rules() []config.Rule { return e.cfg.Rules }
func (e *Engine) DB() *db.DB           { return e.db }

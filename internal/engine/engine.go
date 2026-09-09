package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	Phase            string  `json:"phase"`
	Verification     string  `json:"verification"`
	MultiThread      bool    `json:"multi_thread"`
	TotalBytes       int64   `json:"total_bytes"`
	TransferredBytes int64   `json:"transferred_bytes"`
	ProgressPercent  float64 `json:"progress_percent"`
	SpeedBps         float64 `json:"speed_bps"`
	ETASeconds       *int64  `json:"eta_seconds,omitempty"`
	StartedAt        string  `json:"started_at"`
	AttemptNumber     int     `json:"attempt_number"`
	MaxAttempts       int     `json:"max_attempts"`
}

type RuleScanStatus struct {
	RuleID              string  `json:"rule_id"`
	LastScanStartedAt   *string `json:"last_scan_started_at,omitempty"`
	LastScanCompletedAt *string `json:"last_scan_completed_at,omitempty"`
	LastError           string  `json:"last_error,omitempty"`
	SourceObjects       int     `json:"source_objects"`
	RTorrentParsed      int     `json:"rtorrent_parsed"`
	RTorrentComplete    int     `json:"rtorrent_complete"`
}

type JobView struct {
	db.Job
	AttemptNumber int `json:"attempt_number"`
	MaxAttempts   int `json:"max_attempts"`
}

type Status struct {
	Service   string                   `json:"service"`
	Version   string                   `json:"version"`
	Revision  string                   `json:"revision"`
	Scanning  bool                     `json:"scanning"`
	Active    *ActiveTransfer          `json:"active,omitempty"`
	Queue     db.QueueStats            `json:"queue"`
	Jobs      map[string]int           `json:"jobs"`
	RuleScans map[string]RuleScanStatus `json:"rule_scans"`
}

type RetryPolicy struct {
	RetryCount       int `json:"retry_count"`
	RetryWaitSeconds int `json:"retry_wait_seconds"`
}

type Engine struct {
	cfg        config.Config
	db         *db.DB
	rc         rc.Client
	rules      []config.Rule
	version    string
	revision   string
	wake       chan struct{}
	mu         sync.Mutex
	scanning   bool
	active         *ActiveTransfer
	activeCancel   context.CancelFunc
	controlActions map[int64]string
	retryPolicies  map[string]RetryPolicy
	ruleScans      map[string]RuleScanStatus
}

type seenItem struct {
	item rc.Item
	obj  db.Object
}

func New(cfg config.Config, d *db.DB, version, revision string) *Engine {
	if version == "" {
		version = "dev"
	}
	if revision == "" {
		revision = "unknown"
	}
	e := &Engine{
		cfg:            cfg,
		db:             d,
		rc:             rc.Client{ConfigPath: cfg.RcloneConfig},
		rules:          append([]config.Rule(nil), cfg.Rules...),
		version:        version,
		revision:       revision,
		wake:           make(chan struct{}, 1),
		ruleScans:      make(map[string]RuleScanStatus),
		controlActions: make(map[int64]string),
		retryPolicies:  make(map[string]RetryPolicy),
	}
	if stored, err := d.MetaPrefix("jobdef:"); err == nil {
		for key, raw := range stored {
			var r config.Rule
			if json.Unmarshal([]byte(raw), &r) != nil {
				continue
			}
			if err := config.NormalizeRule(&r, len(e.rules)); err != nil {
				slog.Warn("ignoring invalid persisted job definition", "key", key, "err", err)
				continue
			}
			e.upsertRuleLocked(r)
		}
	}
	for _, r := range e.rules {
		p := RetryPolicy{RetryCount: r.RetryLimit(), RetryWaitSeconds: int(r.RetryWait() / time.Second)}
		if raw, ok, err := d.Meta("rule:" + r.ID + ":retry_policy"); err == nil && ok {
			var count, wait int
			if _, scanErr := fmt.Sscanf(raw, "%d,%d", &count, &wait); scanErr == nil && count >= 0 && wait >= 0 {
				p = RetryPolicy{RetryCount: count, RetryWaitSeconds: wait}
			}
		}
		e.retryPolicies[r.ID] = p
	}
	return e
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

	for _, r := range e.rulesSnapshot() {
		if !r.Enabled {
			continue
		}
		e.scanStarted(r.ID)
		var scanErr error
		if err := e.scanRule(ctx, r); err != nil {
			scanErr = err
			slog.Error("scan failed", "rule", r.ID, "err", err)
		}
		if err := e.cleanupRule(ctx, r); err != nil {
			slog.Error("cleanup failed", "rule", r.ID, "err", err)
			if scanErr == nil {
				scanErr = fmt.Errorf("cleanup: %w", err)
			}
		}
		e.scanFinished(r.ID, scanErr)
	}
}

func (e *Engine) scanRule(ctx context.Context, r config.Rule) error {
	items, err := e.rc.ListFiles(ctx, r.Source)
	if err != nil {
		return err
	}
	e.scanCounts(r.ID, len(items), 0, 0)
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
			completeCount := 0
			for _, t := range torrents {
				if t.Complete {
					completeCount++
				}
			}
			e.scanCounts(r.ID, len(items), len(torrents), completeCount)
			slog.Info("rtorrent scan", "rule", r.ID, "parsed", len(torrents), "complete", completeCount)
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

		if err := e.db.PromoteDueRetries(); err != nil {
			slog.Error("retry scheduler failed", "err", err)
			time.Sleep(2 * time.Second)
			continue
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
			_ = e.db.FailJob(job.ID, "rule no longer exists", nil)
			continue
		}
		items, err := e.db.JobItems(job.ID)
		if err != nil {
			_ = e.db.FailJob(job.ID, err.Error(), nil)
			continue
		}

		err = e.transferJob(ctx, r, job, items)
		if err == nil {
			e.discardControlAction(job.ID)
			continue
		}

		if action := e.consumeControlAction(job.ID); action != "" {
			switch action {
			case "pause":
				if stateErr := e.db.PauseJob(job.ID); stateErr != nil {
					slog.Error("pause state update failed", "job", job.ID, "err", stateErr)
				} else {
					slog.Info("job paused", "job", job.ID, "name", job.DisplayName)
				}
			case "cancel":
				if stateErr := e.db.CancelJob(job.ID); stateErr != nil {
					slog.Error("cancel state update failed", "job", job.ID, "err", stateErr)
				} else {
					if cleanupErr := e.cleanupJobStage(context.Background(), r, job.ID); cleanupErr != nil {
						slog.Warn("cancelled job staging cleanup failed", "job", job.ID, "err", cleanupErr)
					}
					slog.Info("job cancelled", "job", job.ID, "name", job.DisplayName)
				}
			}
			continue
		}

		attempt := job.Attempts + 1
		slog.Error("transfer failed", "job", job.ID, "name", job.DisplayName, "attempt", attempt, "max_attempts", r.RetryLimit()+1, "err", err)
		if attempt <= r.RetryLimit() {
			retryAt := time.Now().UTC().Add(r.RetryWait())
			if dbErr := e.db.FailJob(job.ID, err.Error(), &retryAt); dbErr != nil {
				slog.Error("schedule retry failed", "job", job.ID, "err", dbErr)
			} else {
				slog.Info("job retry scheduled", "job", job.ID, "retry_at", retryAt.Format(time.RFC3339), "next_attempt", attempt+1, "max_attempts", r.RetryLimit()+1)
			}
		} else {
			if dbErr := e.db.FailJob(job.ID, err.Error(), nil); dbErr != nil {
				slog.Error("mark job failed failed", "job", job.ID, "err", dbErr)
			} else {
				slog.Error("job retries exhausted", "job", job.ID, "attempts", attempt, "max_attempts", r.RetryLimit()+1)
			}
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
	stageRootRel := path.Join(".copyarr-staging", strconv.FormatInt(job.ID, 10))
	stageRel := path.Join(stageRootRel, job.RelRoot)
	stageRoot := rc.Target(r.Destination, stageRootRel)
	stage := rc.Target(r.Destination, stageRel)

	transferCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	e.beginActive(job, r, cancel)
	defer e.clearActive(job.ID)

	if job.Attempts > 0 {
		slog.Info("cleaning stale staging before retry", "job", job.ID, "stage_root", stageRoot, "attempts", job.Attempts)
		if err := e.rc.PurgeIfExists(transferCtx, stageRoot); err != nil {
			return fmt.Errorf("clean stale staging: %w", err)
		}
	}
	e.setPhase(job.ID, "transferring")

	slog.Info("copying job", "job", job.ID, "kind", job.Kind, "name", job.DisplayName, "source", src, "stage", stage, "items", len(items), "bytes", job.TotalBytes, "verification", r.Verification, "multi_thread_streams", r.MultiThreadStreams)
	var usedMT bool
	if isDir {
		var err error
		usedMT, err = e.rc.CopyDirWithMultiThreadFallback(transferCtx, src, stage, r.RcloneArgs, r.MultiThreadStreams, r.MultiThreadCutoff, func(p rc.Progress) {
			e.updateRcloneProgress(job.ID, p)
		})
		if err != nil {
			return err
		}
	} else {
		var err error
		usedMT, err = e.rc.CopyToWithMultiThreadFallback(transferCtx, src, stage, r.RcloneArgs, r.MultiThreadStreams, r.MultiThreadCutoff, func(p rc.Progress) {
			e.updateRcloneProgress(job.ID, p)
		})
		if err != nil {
			return err
		}
	}
	e.setTransferMode(job.ID, usedMT)
	e.setPhase(job.ID, "finalizing")

	if r.Verification == "size" {
		e.setPhase(job.ID, "verifying_staging")
		if err := e.verifyManifest(transferCtx, stage, job.RelRoot, items, isDir); err != nil {
			return fmt.Errorf("verify staging: %w", err)
		}
	}

	e.setPhase(job.ID, "committing")
	if err := e.rc.MoveTo(transferCtx, stage, final); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	if r.Verification == "size" {
		e.setPhase(job.ID, "verifying_final")
		if err := e.verifyManifest(transferCtx, final, job.RelRoot, items, isDir); err != nil {
			return fmt.Errorf("verify committed: %w", err)
		}
	}

	for _, item := range items {
		if err := e.db.CompleteObject(item.ObjectID, rc.Target(r.Destination, item.RelPath)); err != nil {
			return err
		}
	}
	if r.Mode == "move" {
		if isDir {
			if err := e.rc.Purge(transferCtx, src); err != nil {
				return fmt.Errorf("destination committed but source purge failed: %w", err)
			}
		} else if err := e.rc.DeleteFile(transferCtx, src); err != nil {
			return fmt.Errorf("destination committed but source delete failed: %w", err)
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

func (e *Engine) beginActive(job db.Job, r config.Rule, cancel context.CancelFunc) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = &ActiveTransfer{
		JobID: job.ID, RuleID: job.RuleID, Name: job.DisplayName, Kind: job.Kind,
		Path: job.RelRoot, Phase: "preparing", Verification: r.Verification,
		MultiThread: r.MultiThreadStreams > 1, TotalBytes: job.TotalBytes, StartedAt: now,
		AttemptNumber: job.Attempts + 1, MaxAttempts: r.RetryLimit() + 1,
	}
	e.activeCancel = cancel
}

func (e *Engine) setPhase(jobID int64, phase string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active != nil && e.active.JobID == jobID {
		e.active.Phase = phase
	}
}

func (e *Engine) setTransferMode(jobID int64, multiThread bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active != nil && e.active.JobID == jobID {
		e.active.MultiThread = multiThread
	}
}

func (e *Engine) updateRcloneProgress(jobID int64, p rc.Progress) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active == nil || e.active.JobID != jobID {
		return
	}
	bytes := p.Bytes
	if bytes < 0 {
		bytes = 0
	}
	if e.active.TotalBytes > 0 && bytes > e.active.TotalBytes {
		bytes = e.active.TotalBytes
	}
	e.active.TransferredBytes = bytes
	if e.active.TotalBytes > 0 {
		e.active.ProgressPercent = float64(bytes) * 100 / float64(e.active.TotalBytes)
		if e.active.ProgressPercent > 100 {
			e.active.ProgressPercent = 100
		}
	}
	e.active.SpeedBps = p.Speed
	e.active.ETASeconds = p.ETASeconds
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
		e.activeCancel = nil
	}
}

func (e *Engine) ControlJob(jobID int64, action string) error {
	e.mu.Lock()
	if e.active != nil && e.active.JobID == jobID {
		if action != "pause" && action != "cancel" {
			e.mu.Unlock()
			return fmt.Errorf("active job %d only supports pause or cancel", jobID)
		}
		if e.active.Phase != "transferring" {
			phase := e.active.Phase
			e.mu.Unlock()
			return fmt.Errorf("job %d cannot be %sd during phase %s", jobID, action, phase)
		}
		e.controlActions[jobID] = action
		cancel := e.activeCancel
		e.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return nil
	}
	e.mu.Unlock()

	switch action {
	case "retry":
		return e.db.RetryJob(jobID)
	case "resume":
		return e.db.RequeueJob(jobID)
	case "pause":
		return e.db.PauseJob(jobID)
	case "cancel":
		if err := e.db.CancelJob(jobID); err != nil {
			return err
		}
		if j, ok := e.jobByID(jobID); ok {
			if r, found := e.rule(j.RuleID); found {
				_ = e.cleanupJobStage(context.Background(), r, jobID)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported job action %q", action)
	}
}

func (e *Engine) consumeControlAction(jobID int64) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	action := e.controlActions[jobID]
	delete(e.controlActions, jobID)
	return action
}

func (e *Engine) discardControlAction(jobID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.controlActions, jobID)
}

func (e *Engine) cleanupJobStage(ctx context.Context, r config.Rule, jobID int64) error {
	stageRootRel := path.Join(".copyarr-staging", strconv.FormatInt(jobID, 10))
	return e.rc.PurgeIfExists(ctx, rc.Target(r.Destination, stageRootRel))
}

func (e *Engine) jobByID(jobID int64) (db.Job, bool) {
	jobs, err := e.db.ListJobs(1000)
	if err != nil {
		return db.Job{}, false
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return j, true
		}
	}
	return db.Job{}, false
}

func (e *Engine) Jobs(limit int) ([]JobView, error) {
	jobs, err := e.db.ListJobs(limit)
	if err != nil {
		return nil, err
	}
	out := make([]JobView, 0, len(jobs))
	for _, j := range jobs {
		maxAttempts := 1
		if r, ok := e.rule(j.RuleID); ok {
			maxAttempts = r.RetryLimit() + 1
		}
		attempt := j.Attempts
		if j.State == "queued" {
			attempt = j.Attempts + 1
		}
		out = append(out, JobView{Job: j, AttemptNumber: attempt, MaxAttempts: maxAttempts})
	}
	return out, nil
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
	ruleScans := make(map[string]RuleScanStatus, len(e.ruleScans))
	for k, v := range e.ruleScans {
		ruleScans[k] = v
	}
	version := e.version
	revision := e.revision
	e.mu.Unlock()
	return Status{
		Service: "copyarr", Version: version, Revision: revision, Scanning: scanning,
		Active: active, Queue: queue, Jobs: counts, RuleScans: ruleScans,
	}, nil
}

func (e *Engine) scanStarted(ruleID string) {
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.ruleScans[ruleID]
	s.RuleID = ruleID
	s.LastScanStartedAt = &ts
	s.LastError = ""
	e.ruleScans[ruleID] = s
}

func (e *Engine) scanCounts(ruleID string, sourceObjects, rtParsed, rtComplete int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.ruleScans[ruleID]
	s.RuleID = ruleID
	s.SourceObjects = sourceObjects
	s.RTorrentParsed = rtParsed
	s.RTorrentComplete = rtComplete
	e.ruleScans[ruleID] = s
}

func (e *Engine) scanFinished(ruleID string, err error) {
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.ruleScans[ruleID]
	s.RuleID = ruleID
	s.LastScanCompletedAt = &ts
	if err != nil {
		s.LastError = err.Error()
	} else {
		s.LastError = ""
	}
	e.ruleScans[ruleID] = s
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

func (e *Engine) rulesSnapshot() []config.Rule {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]config.Rule(nil), e.rules...)
}

func (e *Engine) upsertRuleLocked(r config.Rule) {
	for i := range e.rules {
		if e.rules[i].ID == r.ID {
			e.rules[i] = r
			return
		}
	}
	e.rules = append(e.rules, r)
}

func (e *Engine) rule(id string) (config.Rule, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range e.rules {
		if r.ID == id {
			if p, ok := e.retryPolicies[id]; ok {
				count := p.RetryCount
				wait := p.RetryWaitSeconds
				r.RetryCount = &count
				r.RetryWaitSeconds = &wait
			}
			return r, true
		}
	}
	return config.Rule{}, false
}

func (e *Engine) RetryPolicy(ruleID string) (RetryPolicy, bool) {
	if _, ok := e.rule(ruleID); !ok {
		return RetryPolicy{}, false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ok := e.retryPolicies[ruleID]
	return p, ok
}

func (e *Engine) UpdateRetryPolicy(ruleID string, count, waitSeconds int) error {
	if count < 0 || waitSeconds < 0 {
		return fmt.Errorf("retry_count and retry_wait_seconds must be >= 0")
	}
	if _, ok := e.rule(ruleID); !ok {
		return fmt.Errorf("rule %q not found", ruleID)
	}
	p := RetryPolicy{RetryCount: count, RetryWaitSeconds: waitSeconds}
	if err := e.db.SetMeta("rule:"+ruleID+":retry_policy", fmt.Sprintf("%d,%d", count, waitSeconds)); err != nil {
		return err
	}
	e.mu.Lock()
	e.retryPolicies[ruleID] = p
	e.mu.Unlock()
	return nil
}

func (e *Engine) SaveJobDefinition(r config.Rule) error {
	if err := config.NormalizeRule(&r, 0); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := e.db.SetMeta("jobdef:"+r.ID, string(raw)); err != nil {
		return err
	}
	p := RetryPolicy{RetryCount: r.RetryLimit(), RetryWaitSeconds: int(r.RetryWait() / time.Second)}
	if err := e.db.SetMeta("rule:"+r.ID+":retry_policy", fmt.Sprintf("%d,%d", p.RetryCount, p.RetryWaitSeconds)); err != nil {
		return err
	}
	e.mu.Lock()
	e.upsertRuleLocked(r)
	e.retryPolicies[r.ID] = p
	e.mu.Unlock()
	return nil
}

func (e *Engine) DeleteJobDefinition(id string) error {
	for _, base := range e.cfg.Rules {
		if base.ID == id {
			return fmt.Errorf("job %q comes from config.json; disable or edit it instead of deleting it", id)
		}
	}
	if err := e.db.DeleteMeta("jobdef:" + id); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	out := e.rules[:0]
	for _, r := range e.rules {
		if r.ID != id {
			out = append(out, r)
		}
	}
	e.rules = out
	delete(e.retryPolicies, id)
	return nil
}

func (e *Engine) Rules() []config.Rule {
	rules := e.rulesSnapshot()
	out := make([]config.Rule, 0, len(rules))
	for _, base := range rules {
		r, _ := e.rule(base.ID)
		if r.RTorrent != nil {
			cp := *r.RTorrent
			cp.Password = ""
			r.RTorrent = &cp
		}
		out = append(out, r)
	}
	return out
}

func (e *Engine) Remotes(ctx context.Context) ([]rc.RemoteInfo, error) {
	return e.rc.ListRemotes(ctx)
}

func (e *Engine) RemoteProviders(ctx context.Context) (json.RawMessage, error) {
	return e.rc.Providers(ctx)
}

func (e *Engine) CreateRemote(ctx context.Context, name, typ string, params map[string]string) (*rc.ConfigQuestion, error) {
	return e.rc.CreateRemote(ctx, name, typ, params)
}

func (e *Engine) UpdateRemote(ctx context.Context, name string, params map[string]string) (*rc.ConfigQuestion, error) {
	return e.rc.UpdateRemote(ctx, name, params)
}

func (e *Engine) DeleteRemote(ctx context.Context, name string) error {
	return e.rc.DeleteRemote(ctx, name)
}

func (e *Engine) DB() *db.DB { return e.db }

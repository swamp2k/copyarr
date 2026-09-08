package rclone

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
		"os/exec"
	"path"
	"strings"
	"time"

	"github.com/swamp2k/copyarr/internal/config"
)

type Client struct {
	ConfigPath string
}

type Progress struct {
	Bytes      int64
	TotalBytes int64
	Speed      float64
	ETASeconds *int64
}

type ProgressFunc func(Progress)

type Item struct {
	Path    string    `json:"Path"`
	Name    string    `json:"Name"`
	Size    int64     `json:"Size"`
	ModTime time.Time `json:"ModTime"`
	IsDir   bool      `json:"IsDir"`
}

func Target(e config.Endpoint, rel string) string {
	p := strings.TrimSuffix(e.Path, "/")
	if rel != "" {
		p = path.Join(p, rel)
	}
	if e.Remote == "" {
		return p
	}
	return e.Remote + ":" + p
}

func (c Client) run(ctx context.Context, args ...string) ([]byte, error) {
	base := []string{}
	if c.ConfigPath != "" {
		base = append(base, "--config", c.ConfigPath)
	}
	cmd := exec.CommandContext(ctx, "rclone", append(base, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone %v: %w: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}


func (c Client) runCopyJSONStats(ctx context.Context, args []string, progress ProgressFunc) error {
	base := []string{}
	if c.ConfigPath != "" {
		base = append(base, "--config", c.ConfigPath)
	}

	// rclone emits periodic transfer stats as NDJSON records when JSON logging
	// is enabled. Stats are INFO by default, so explicitly promote them to
	// NOTICE to make sure they are emitted without enabling verbose logging.
	args = append(args, "--stats", "2s", "--stats-log-level", "NOTICE", "--use-json-log")
	cmd := exec.CommandContext(ctx, "rclone", append(base, args...)...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	var nonStats []string
	scanner := bufio.NewScanner(stderr)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		var raw struct {
			Stats *struct {
				Bytes      int64    `json:"bytes"`
				TotalBytes int64    `json:"totalBytes"`
				Speed      float64  `json:"speed"`
				ETA        *float64 `json:"eta"`
			} `json:"stats"`
		}
		if json.Unmarshal([]byte(line), &raw) == nil && raw.Stats != nil {
			p := Progress{
				Bytes:      raw.Stats.Bytes,
				TotalBytes: raw.Stats.TotalBytes,
				Speed:      raw.Stats.Speed,
			}
			if raw.Stats.ETA != nil && *raw.Stats.ETA >= 0 {
				eta := int64(*raw.Stats.ETA)
				p.ETASeconds = &eta
			}
			if progress != nil {
				progress(p)
			}
			continue
		}
		nonStats = append(nonStats, line)
	}

	err = cmd.Wait()
	if scanErr := scanner.Err(); scanErr != nil && err == nil {
		return scanErr
	}
	if err != nil {
		return fmt.Errorf("rclone %v: %w: %s", args, err, strings.TrimSpace(strings.Join(nonStats, "\n")))
	}
	return nil
}

func (c Client) runCopy(ctx context.Context, args []string, progress ProgressFunc) error {
	err := c.runCopyJSONStats(ctx, args, progress)
	if err == nil {
		return nil
	}
	s := strings.ToLower(err.Error())
	if !strings.Contains(s, "unknown flag: --use-json-log") {
		return err
	}

	// Telemetry must never prevent a transfer from running on an older rclone.
	base := []string{}
	if c.ConfigPath != "" {
		base = append(base, "--config", c.ConfigPath)
	}
	cmd := exec.CommandContext(ctx, "rclone", append(base, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone %v: %w: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (c Client) ListFiles(ctx context.Context, e config.Endpoint) ([]Item, error) {
	out, err := c.run(ctx, "lsjson", Target(e, ""), "--recursive", "--files-only")
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c Client) CopyTo(ctx context.Context, src, dst string, extra []string) error {
	args := []string{"copyto", src, dst, "--partial-suffix", ".copyarr-part"}
	args = append(args, extra...)
	_, err := c.run(ctx, args...)
	return err
}

func (c Client) CopyDir(ctx context.Context, src, dst string, extra []string) error {
	args := []string{"copy", src, dst, "--partial-suffix", ".copyarr-part"}
	args = append(args, extra...)
	_, err := c.run(ctx, args...)
	return err
}

func (c Client) CopyToWithMultiThreadFallback(ctx context.Context, src, dst string, extra []string, streams int, cutoff string, progress ProgressFunc) (bool, error) {
	withMT := append([]string{"copyto", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	if streams > 1 {
		withMT = append(withMT, "--multi-thread-streams", fmt.Sprint(streams))
		if cutoff != "" {
			withMT = append(withMT, "--multi-thread-cutoff", cutoff)
		}
	}
	err := c.runCopy(ctx, withMT, progress)
	if err == nil || streams <= 1 {
		return streams > 1, err
	}
	if !isMultiThreadUnsupported(err) {
		return true, err
	}
	fallback := append([]string{"copyto", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	return false, c.runCopy(ctx, fallback, progress)
}

func (c Client) CopyDirWithMultiThreadFallback(ctx context.Context, src, dst string, extra []string, streams int, cutoff string, progress ProgressFunc) (bool, error) {
	withMT := append([]string{"copy", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	if streams > 1 {
		withMT = append(withMT, "--multi-thread-streams", fmt.Sprint(streams))
		if cutoff != "" {
			withMT = append(withMT, "--multi-thread-cutoff", cutoff)
		}
	}
	err := c.runCopy(ctx, withMT, progress)
	if err == nil || streams <= 1 {
		return streams > 1, err
	}
	if !isMultiThreadUnsupported(err) {
		return true, err
	}
	fallback := append([]string{"copy", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	return false, c.runCopy(ctx, fallback, progress)
}

func isMultiThreadUnsupported(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "multi-thread") ||
		strings.Contains(s, "multithread") ||
		strings.Contains(s, "multi thread") ||
		strings.Contains(s, "not supported")
}

func (c Client) ListTargetFiles(ctx context.Context, target string) ([]Item, error) {
	out, err := c.run(ctx, "lsjson", target, "--recursive", "--files-only")
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c Client) TargetBytes(ctx context.Context, target string) (int64, error) {
	items, err := c.ListTargetFiles(ctx, target)
	if err == nil {
		var total int64
		for _, item := range items {
			total += item.Size
		}
		return total, nil
	}
	st, statErr := c.Stat(ctx, target)
	if statErr != nil {
		return 0, err
	}
	return st.Size, nil
}

func (c Client) MoveTo(ctx context.Context, src, dst string) error {
	_, err := c.run(ctx, "moveto", src, dst)
	return err
}

func (c Client) DeleteFile(ctx context.Context, target string) error {
	_, err := c.run(ctx, "deletefile", target)
	return err
}

func (c Client) Purge(ctx context.Context, target string) error {
	_, err := c.run(ctx, "purge", target)
	return err
}

func (c Client) PurgeIfExists(ctx context.Context, target string) error {
	_, err := c.run(ctx, "purge", target)
	if err == nil {
		return nil
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "not found") || strings.Contains(s, "directory not found") || strings.Contains(s, "object not found") {
		return nil
	}
	return err
}

func (c Client) Stat(ctx context.Context, target string) (Item, error) {
	out, err := c.run(ctx, "lsjson", target, "--stat")
	if err != nil {
		return Item{}, err
	}
	var item Item
	if err := json.Unmarshal(out, &item); err != nil {
		return Item{}, err
	}
	return item, nil
}

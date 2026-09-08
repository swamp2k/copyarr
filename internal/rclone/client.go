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
	"sync"
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


func (c Client) runCopy(ctx context.Context, args []string, progress ProgressFunc) error {
	base := []string{}
	if c.ConfigPath != "" {
		base = append(base, "--config", c.ConfigPath)
	}
	args = append(args, "--stats", "2s", "--stats-one-line-json")
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

	var mu sync.Mutex
	var nonStats []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			var raw map[string]any
			if json.Unmarshal([]byte(line), &raw) == nil {
				p := Progress{}
				if v, ok := raw["bytes"].(float64); ok {
					p.Bytes = int64(v)
				}
				if v, ok := raw["totalBytes"].(float64); ok {
					p.TotalBytes = int64(v)
				}
				if v, ok := raw["speed"].(float64); ok {
					p.Speed = v
				}
				if v, ok := raw["eta"].(float64); ok && v >= 0 {
					eta := int64(v)
					p.ETASeconds = &eta
				}
				if progress != nil && (p.Bytes > 0 || p.TotalBytes > 0 || p.Speed > 0) {
					progress(p)
					continue
				}
			}
			mu.Lock()
			nonStats = append(nonStats, line)
			mu.Unlock()
		}
	}()

	err = cmd.Wait()
	<-done
	if err != nil {
		mu.Lock()
		msg := strings.Join(nonStats, "\n")
		mu.Unlock()
		return fmt.Errorf("rclone %v: %w: %s", args, err, strings.TrimSpace(msg))
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

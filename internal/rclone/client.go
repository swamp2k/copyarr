package rclone

import (
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

func (c Client) MoveTo(ctx context.Context, src, dst string) error {
	_, err := c.run(ctx, "moveto", src, dst)
	return err
}

func (c Client) DeleteFile(ctx context.Context, target string) error {
	_, err := c.run(ctx, "deletefile", target)
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

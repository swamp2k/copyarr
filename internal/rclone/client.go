package rclone

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
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
type RawLogFunc func(string)

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
	return c.runWith(ctx, c.ConfigPath, args...)
}

// runWith is run() against an explicit config file, so probes can execute
// against a throwaway config without touching the live one.
func (c Client) runWith(ctx context.Context, configPath string, args ...string) ([]byte, error) {
	base := []string{}
	if configPath != "" {
		base = append(base, "--config", configPath)
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

func (c Client) runCopyJSONStats(ctx context.Context, args []string, progress ProgressFunc, rawLog RawLogFunc) error {
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
		if rawLog != nil {
			rawLog(line)
		}
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

func (c Client) runCopy(ctx context.Context, args []string, progress ProgressFunc, rawLog RawLogFunc) error {
	err := c.runCopyJSONStats(ctx, args, progress, rawLog)
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
	err = cmd.Run()
	if rawLog != nil {
		for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
			if strings.TrimSpace(line) != "" {
				rawLog(line)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("rclone %v: %w: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

type RemoteInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ConfigQuestion struct {
	State  string         `json:"State"`
	Option map[string]any `json:"Option"`
	Error  string         `json:"Error"`
}

func (c Client) ListRemotes(ctx context.Context) ([]RemoteInfo, error) {
	out, err := c.run(ctx, "listremotes")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	remotes := make([]RemoteInfo, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSuffix(strings.TrimSpace(line), ":")
		if name == "" {
			continue
		}
		typ := ""
		if redacted, e := c.run(ctx, "config", "redacted", name); e == nil {
			for _, l := range strings.Split(string(redacted), "\n") {
				l = strings.TrimSpace(l)
				if strings.HasPrefix(l, "type = ") {
					typ = strings.TrimSpace(strings.TrimPrefix(l, "type = "))
					break
				}
			}
		}
		remotes = append(remotes, RemoteInfo{Name: name, Type: typ})
	}
	return remotes, nil
}

func (c Client) CreateRemote(ctx context.Context, name, typ string, params map[string]string) (*ConfigQuestion, error) {
	args := []string{"config", "create", name, typ, "--non-interactive", "--obscure"}
	for k, v := range params {
		args = append(args, k, v)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var q ConfigQuestion
	if json.Unmarshal(out, &q) == nil && q.State != "" {
		return &q, nil
	}
	return nil, nil
}

func (c Client) UpdateRemote(ctx context.Context, name string, params map[string]string) (*ConfigQuestion, error) {
	args := []string{"config", "update", name, "--non-interactive", "--obscure"}
	for k, v := range params {
		args = append(args, k, v)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var q ConfigQuestion
	if json.Unmarshal(out, &q) == nil && q.State != "" {
		return &q, nil
	}
	return nil, nil
}

func (c Client) DeleteRemote(ctx context.Context, name string) error {
	_, err := c.run(ctx, "config", "delete", name)
	return err
}

func (c Client) Providers(ctx context.Context) (json.RawMessage, error) {
	out, err := c.run(ctx, "config", "providers")
	if err != nil {
		return nil, err
	}
	if !json.Valid(out) {
		return nil, fmt.Errorf("rclone providers returned invalid JSON")
	}
	return json.RawMessage(out), nil
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

func (c Client) CopyToWithMultiThreadFallback(ctx context.Context, src, dst string, extra []string, streams int, cutoff string, progress ProgressFunc, rawLog RawLogFunc) (bool, error) {
	withMT := append([]string{"copyto", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	if streams > 1 {
		withMT = append(withMT, "--multi-thread-streams", fmt.Sprint(streams))
		if cutoff != "" {
			withMT = append(withMT, "--multi-thread-cutoff", cutoff)
		}
	}
	err := c.runCopy(ctx, withMT, progress, rawLog)
	if err == nil || streams <= 1 {
		return streams > 1, err
	}
	if !isMultiThreadUnsupported(err) {
		return true, err
	}
	fallback := append([]string{"copyto", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	return false, c.runCopy(ctx, fallback, progress, rawLog)
}

func (c Client) CopyDirWithMultiThreadFallback(ctx context.Context, src, dst string, extra []string, streams int, cutoff string, progress ProgressFunc, rawLog RawLogFunc) (bool, error) {
	withMT := append([]string{"copy", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	if streams > 1 {
		withMT = append(withMT, "--multi-thread-streams", fmt.Sprint(streams))
		if cutoff != "" {
			withMT = append(withMT, "--multi-thread-cutoff", cutoff)
		}
	}
	err := c.runCopy(ctx, withMT, progress, rawLog)
	if err == nil || streams <= 1 {
		return streams > 1, err
	}
	if !isMultiThreadUnsupported(err) {
		return true, err
	}
	fallback := append([]string{"copy", src, dst, "--partial-suffix", ".copyarr-part"}, extra...)
	return false, c.runCopy(ctx, fallback, progress, rawLog)
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

// probeRemoteName is the throwaway remote name used when validating a remote
// definition that has not been saved to the live config yet.
const probeRemoteName = "copyarr_probe"

// TestResult reports whether a remote answered a listing request.
type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Entries int    `json:"entries"`
}

// RemoteDetail is a saved remote's configuration with rclone's own redaction
// applied, so an edit form can be prefilled without exposing secrets.
type RemoteDetail struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Parameters map[string]string `json:"parameters"`
	Redacted   []string          `json:"redacted"`
}

// rcloneLogPrefix matches the timestamp and severity rclone stamps on every log
// line. It is deliberately unanchored: depending on whether stderr arrived as
// its own line or folded into the wrapping "rclone %v: %w: %s" message, the
// prefix can sit mid-string rather than at the start.
var rcloneLogPrefix = regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} (?:DEBUG|INFO|NOTICE|WARNING|ERROR|CRITICAL)\s*:\s*`)

// summarizeError reduces rclone's multi-line stderr to the one line most
// likely to explain the failure to a user.
func summarizeError(err error) string {
	var lines []string
	for _, l := range strings.Split(err.Error(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "rclone failed without any output"
	}
	pick := lines[len(lines)-1]
	for _, l := range lines {
		if strings.Contains(l, "Failed to") || strings.Contains(l, "ERROR") || strings.Contains(l, "CRITICAL") {
			pick = l
			break
		}
	}
	if loc := rcloneLogPrefix.FindStringIndex(pick); loc != nil {
		pick = pick[loc[1]:]
	}
	// The throwaway probe remote is an implementation detail; naming it in an
	// error would only puzzle whoever is filling in the form.
	pick = strings.ReplaceAll(pick, `"`+probeRemoteName+`:"`, "this remote")
	pick = strings.TrimSpace(pick)
	if len(pick) > 400 {
		pick = pick[:400] + "..."
	}
	return pick
}

func remoteTarget(name, p string) string {
	return name + ":" + strings.TrimSpace(p)
}

// probe lists one level of a target with retries disabled so a bad credential
// or unreachable host reports back quickly instead of blocking the UI.
func (c Client) probe(ctx context.Context, configPath, target string) TestResult {
	out, err := c.runWith(ctx, configPath, "lsjson", target,
		"--max-depth", "1",
		"--retries", "1",
		"--low-level-retries", "1",
		"--timeout", "20s",
		"--contimeout", "10s",
	)
	if err != nil {
		return TestResult{Message: summarizeError(err)}
	}
	var items []Item
	if json.Unmarshal(out, &items) != nil {
		return TestResult{OK: true, Message: "Connected."}
	}
	noun := "entries"
	if len(items) == 1 {
		noun = "entry"
	}
	return TestResult{OK: true, Entries: len(items), Message: fmt.Sprintf("Connected - %d %s at this path.", len(items), noun)}
}

// TestRemote checks a remote that already exists in the live config.
func (c Client) TestRemote(ctx context.Context, name, p string) TestResult {
	return c.probe(ctx, c.ConfigPath, remoteTarget(name, p))
}

// TestConfig checks a remote definition that has not been saved yet by
// materialising it in a throwaway config file and listing its root.
func (c Client) TestConfig(ctx context.Context, typ string, params map[string]string, p string) (TestResult, error) {
	dir, err := os.MkdirTemp("", "copyarr-probe-")
	if err != nil {
		return TestResult{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := filepath.Join(dir, "rclone.conf")
	if err := os.WriteFile(cfg, nil, 0o600); err != nil {
		return TestResult{}, err
	}

	args := []string{"config", "create", probeRemoteName, typ, "--non-interactive", "--obscure"}
	for _, k := range sortedKeys(params) {
		args = append(args, k, params[k])
	}
	out, err := c.runWith(ctx, cfg, args...)
	if err != nil {
		return TestResult{Message: summarizeError(err)}, nil
	}
	var q ConfigQuestion
	if json.Unmarshal(out, &q) == nil && q.State != "" {
		return TestResult{Message: "This provider needs an interactive or OAuth step that Copyarr cannot complete yet."}, nil
	}
	return c.probe(ctx, cfg, remoteTarget(probeRemoteName, p)), nil
}

// RemoteConfig reads a saved remote back through rclone's redacting printer.
// Redacted values come back as XXX, which are reported separately instead of
// being handed to the caller as if they were real values.
func (c Client) RemoteConfig(ctx context.Context, name string) (RemoteDetail, error) {
	out, err := c.run(ctx, "config", "redacted", name)
	if err != nil {
		return RemoteDetail{}, err
	}
	return parseRedactedConfig(name, out), nil
}

// parseRedactedConfig turns "rclone config redacted" output into a RemoteDetail.
// rclone replaces every value it considers sensitive with XXX - which is
// broader than passwords, covering host and user on sftp for example. Those
// keys are reported by name only; their values are never carried over, so a
// stored secret cannot reach the browser through this path.
func parseRedactedConfig(name string, out []byte) RemoteDetail {
	detail := RemoteDetail{Name: name, Parameters: map[string]string{}}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch {
		case key == "type":
			detail.Type = value
		case value == "XXX":
			detail.Redacted = append(detail.Redacted, key)
		default:
			detail.Parameters[key] = value
		}
	}
	sort.Strings(detail.Redacted)
	return detail
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

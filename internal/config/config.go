package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	DataDir             string `json:"data_dir"`
	ListenAddr          string `json:"listen_addr"`
	RcloneConfig        string `json:"rclone_config"`
	ScanIntervalSeconds int    `json:"scan_interval_seconds"`
	Rules               []Rule `json:"rules"`

	// Privateering: optional background push of a torrents/files snapshot to a
	// Nexus instance. Enabled only when both values are set. Prefer the
	// NEXUS_URL / NEXUS_PRIVATEERING_TOKEN environment variables for the
	// token so secrets never need to live in config.json.
	NexusURL                        string `json:"nexus_url,omitempty"`
	NexusPrivateeringToken           string `json:"nexus_privateering_token,omitempty"`
	PrivateeringPushIntervalSeconds int    `json:"privateering_push_interval_seconds,omitempty"`
}

// PrivateeringEnabled reports whether both the Nexus URL and token are set.
func (c Config) PrivateeringEnabled() bool {
	return c.NexusURL != "" && c.NexusPrivateeringToken != ""
}

// PrivateeringPushInterval returns the configured push interval, defaulting
// to 5 minutes.
func (c Config) PrivateeringPushInterval() time.Duration {
	if c.PrivateeringPushIntervalSeconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.PrivateeringPushIntervalSeconds) * time.Second
}

type Endpoint struct {
	Remote string `json:"remote"`
	Path   string `json:"path"`
}

type RTorrent struct {
	URL            string `json:"url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	View           string `json:"view"`
	SourceBasePath string `json:"source_base_path"`
	Required       bool   `json:"required"`
}

type Rule struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Enabled            bool      `json:"enabled"`
	Source             Endpoint  `json:"source"`
	Destination        Endpoint  `json:"destination"`
	Mode               string    `json:"mode"`
	InitialBehavior    string    `json:"initial_behavior"`
	StabilitySeconds   int       `json:"stability_seconds"`
	CleanupDays        int       `json:"cleanup_days"`
	Verification       string    `json:"verification"`
	MultiThreadStreams int       `json:"multi_thread_streams"`
	MultiThreadCutoff  string    `json:"multi_thread_cutoff"`
	RetryCount          *int      `json:"retry_count"`
	RetryWaitSeconds    *int      `json:"retry_wait_seconds"`
	RTorrent           *RTorrent `json:"rtorrent,omitempty"`
	RcloneArgs         []string  `json:"rclone_args,omitempty"`
	Includes           []string  `json:"includes,omitempty"`
	Excludes           []string  `json:"excludes,omitempty"`
}

func Load(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.DataDir == "" {
		c.DataDir = "/data"
	}
	if c.ListenAddr == "" {
		c.ListenAddr = ":8686"
	}
	if c.ScanIntervalSeconds <= 0 {
		c.ScanIntervalSeconds = 300
	}
	if v := os.Getenv("NEXUS_URL"); v != "" {
		c.NexusURL = v
	}
	if v := os.Getenv("NEXUS_PRIVATEERING_TOKEN"); v != "" {
		c.NexusPrivateeringToken = v
	}
	for i := range c.Rules {
		if err := NormalizeRule(&c.Rules[i], i); err != nil {
			return c, err
		}
	}
	return c, nil
}

func NormalizeRule(r *Rule, index int) error {
	if r.ID == "" {
		return fmt.Errorf("rule %d missing id", index)
	}
	if r.Name == "" {
		r.Name = r.ID
	}
	if r.Mode == "" {
		r.Mode = "copy"
	}
	if r.Mode != "copy" && r.Mode != "move" {
		return fmt.Errorf("rule %s has unsupported mode %q", r.ID, r.Mode)
	}
	if r.InitialBehavior == "" {
		r.InitialBehavior = "ignore_existing"
	}
	if r.StabilitySeconds <= 0 {
		r.StabilitySeconds = 600
	}
	if r.Verification == "" {
		r.Verification = "size"
	}
	if r.Verification != "none" && r.Verification != "size" {
		return fmt.Errorf("rule %s has unsupported verification %q (use none or size)", r.ID, r.Verification)
	}
	if r.MultiThreadStreams < 0 {
		return fmt.Errorf("rule %s has invalid multi_thread_streams", r.ID)
	}
	if r.MultiThreadStreams == 0 {
		r.MultiThreadStreams = 4
	}
	if r.MultiThreadCutoff == "" {
		r.MultiThreadCutoff = "256M"
	}
	if r.RetryCount == nil {
		v := 3
		r.RetryCount = &v
	}
	if *r.RetryCount < 0 {
		return fmt.Errorf("rule %s has invalid retry_count", r.ID)
	}
	if r.RetryWaitSeconds == nil {
		v := 300
		r.RetryWaitSeconds = &v
	}
	if *r.RetryWaitSeconds < 0 {
		return fmt.Errorf("rule %s has invalid retry_wait_seconds", r.ID)
	}
	if r.RTorrent != nil && r.RTorrent.View == "" {
		r.RTorrent.View = "main"
	}
	return nil
}

func (r Rule) RetryLimit() int {
	if r.RetryCount == nil {
		return 3
	}
	return *r.RetryCount
}

func (r Rule) RetryWait() time.Duration {
	if r.RetryWaitSeconds == nil {
		return 300 * time.Second
	}
	return time.Duration(*r.RetryWaitSeconds) * time.Second
}

func (c Config) ScanInterval() time.Duration {
	return time.Duration(c.ScanIntervalSeconds) * time.Second
}

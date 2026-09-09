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
	RetryCount          int       `json:"retry_count"`
	RetryWaitSeconds    int       `json:"retry_wait_seconds"`
	RTorrent           *RTorrent `json:"rtorrent,omitempty"`
	RcloneArgs         []string  `json:"rclone_args,omitempty"`
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
	for i := range c.Rules {
		r := &c.Rules[i]
		if r.ID == "" {
			return c, fmt.Errorf("rule %d missing id", i)
		}
		if r.Mode == "" {
			r.Mode = "copy"
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
			return c, fmt.Errorf("rule %s has unsupported verification %q (use none or size)", r.ID, r.Verification)
		}
		if r.MultiThreadStreams < 0 {
			return c, fmt.Errorf("rule %s has invalid multi_thread_streams", r.ID)
		}
		if r.MultiThreadStreams == 0 {
			r.MultiThreadStreams = 4
		}
		if r.MultiThreadCutoff == "" {
			r.MultiThreadCutoff = "256M"
		}
		if r.RetryCount < 0 {
			return c, fmt.Errorf("rule %s has invalid retry_count", r.ID)
		}
		if r.RetryCount == 0 {
			r.RetryCount = 3
		}
		if r.RetryWaitSeconds < 0 {
			return c, fmt.Errorf("rule %s has invalid retry_wait_seconds", r.ID)
		}
		if r.RetryWaitSeconds == 0 {
			r.RetryWaitSeconds = 300
		}
		if r.RTorrent != nil && r.RTorrent.View == "" {
			r.RTorrent.View = "main"
		}
	}
	return c, nil
}

func (c Config) ScanInterval() time.Duration {
	return time.Duration(c.ScanIntervalSeconds) * time.Second
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConf = "[seedbox]\ntype = sftp\nhost = seedbox.example.net\nuser = martin\npass = obscured-value\n"

// writeReadOnlySource stands in for the /config mount on Unraid: readable, and
// the test asserts afterwards that Copyarr never wrote to it.
func writeReadOnlySource(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "rclone.conf")
	if err := os.WriteFile(path, []byte(sampleConf), 0o444); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEnsureWritableRcloneConfigImportsOnce(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := writeReadOnlySource(t, configDir)

	cfg := Config{DataDir: dataDir, RcloneConfig: source}
	origin, err := EnsureWritableRcloneConfig(&cfg)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if origin != RcloneConfigImported {
		t.Errorf("origin = %q, want %q", origin, RcloneConfigImported)
	}

	want := WritableRcloneConfigPath(dataDir)
	if cfg.RcloneConfig != want {
		t.Errorf("RcloneConfig = %q, want %q", cfg.RcloneConfig, want)
	}
	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read imported config: %v", err)
	}
	if string(got) != sampleConf {
		t.Errorf("imported content = %q, want %q", got, sampleConf)
	}

	// The second start must reuse the copy, not import over it again.
	if err := os.WriteFile(want, []byte(sampleConf+"\n[added-later]\ntype = local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg2 := Config{DataDir: dataDir, RcloneConfig: source}
	origin, err = EnsureWritableRcloneConfig(&cfg2)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if origin != RcloneConfigExisting {
		t.Errorf("second origin = %q, want %q", origin, RcloneConfigExisting)
	}
	after, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "[added-later]") {
		t.Error("second call overwrote the writable config instead of reusing it")
	}
}

func TestEnsureWritableRcloneConfigNeverWritesTheSource(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := writeReadOnlySource(t, configDir)
	before, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{DataDir: filepath.Join(root, "data"), RcloneConfig: source}
	if _, err := EnsureWritableRcloneConfig(&cfg); err != nil {
		t.Fatal(err)
	}

	// Simulate the app writing a new remote into the config it now owns.
	if err := os.WriteFile(cfg.RcloneConfig, []byte("[new]\ntype = local\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sampleConf {
		t.Errorf("source config was modified:\n got %q\nwant %q", got, sampleConf)
	}
	after, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("source config mtime changed; it must only ever be read")
	}
	if cfg.RcloneConfig == source {
		t.Error("engine is still pointed at the read-only source config")
	}
}

func TestEnsureWritableRcloneConfigCreatesEmptyWhenNoSource(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"no source configured", ""},
		{"source path does not exist", filepath.Join(t.TempDir(), "missing", "rclone.conf")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := filepath.Join(t.TempDir(), "data")
			cfg := Config{DataDir: dataDir, RcloneConfig: tc.source}
			origin, err := EnsureWritableRcloneConfig(&cfg)
			if err != nil {
				t.Fatal(err)
			}
			if origin != RcloneConfigCreated {
				t.Errorf("origin = %q, want %q", origin, RcloneConfigCreated)
			}
			if cfg.RcloneConfig != WritableRcloneConfigPath(dataDir) {
				t.Errorf("RcloneConfig = %q", cfg.RcloneConfig)
			}
			body, err := os.ReadFile(cfg.RcloneConfig)
			if err != nil {
				t.Fatalf("expected an empty config to exist: %v", err)
			}
			if len(body) != 0 {
				t.Errorf("expected an empty config, got %q", body)
			}
		})
	}
}

func TestEnsureWritableRcloneConfigLeavesNoStagingFiles(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := writeReadOnlySource(t, configDir)
	dataDir := filepath.Join(root, "data")

	cfg := Config{DataDir: dataDir, RcloneConfig: source}
	if _, err := EnsureWritableRcloneConfig(&cfg); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Dir(cfg.RcloneConfig))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "rclone.conf" {
		t.Errorf("expected only rclone.conf after import, got %v", names)
	}
}

func TestEnsureWritableRcloneConfigRequiresDataDir(t *testing.T) {
	cfg := Config{}
	if _, err := EnsureWritableRcloneConfig(&cfg); err == nil {
		t.Fatal("expected an error when data_dir is empty")
	}
}

func TestEnsureWritableRcloneConfigIsIdempotentWhenAlreadyPointedAtTheCopy(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	cfg := Config{DataDir: dataDir, RcloneConfig: WritableRcloneConfigPath(dataDir)}

	if origin, err := EnsureWritableRcloneConfig(&cfg); err != nil {
		t.Fatal(err)
	} else if origin != RcloneConfigCreated {
		t.Errorf("origin = %q, want %q", origin, RcloneConfigCreated)
	}
	if err := os.WriteFile(cfg.RcloneConfig, []byte(sampleConf), 0o600); err != nil {
		t.Fatal(err)
	}
	if origin, err := EnsureWritableRcloneConfig(&cfg); err != nil {
		t.Fatal(err)
	} else if origin != RcloneConfigExisting {
		t.Errorf("second origin = %q, want %q", origin, RcloneConfigExisting)
	}
	got, err := os.ReadFile(cfg.RcloneConfig)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sampleConf {
		t.Error("re-running the migration clobbered the config it already owned")
	}
}

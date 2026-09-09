package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RcloneConfigOrigin reports what EnsureWritableRcloneConfig did, so startup can
// log which config the process ended up owning.
type RcloneConfigOrigin string

const (
	// RcloneConfigExisting means a writable config was already in the data dir.
	RcloneConfigExisting RcloneConfigOrigin = "existing"
	// RcloneConfigImported means the configured source was copied in once.
	RcloneConfigImported RcloneConfigOrigin = "imported"
	// RcloneConfigCreated means there was nothing to import, so an empty
	// config was created.
	RcloneConfigCreated RcloneConfigOrigin = "created"
)

// WritableRcloneConfigPath is where Copyarr keeps the rclone config it owns.
func WritableRcloneConfigPath(dataDir string) string {
	return filepath.Join(dataDir, "rclone", "rclone.conf")
}

// EnsureWritableRcloneConfig points cfg.RcloneConfig at a config file Copyarr
// can write to, under the data directory.
//
// On Unraid the /config mount is read-only, so the config shipped there can be
// read but never modified. The first time Copyarr starts it imports that file
// into the data directory and uses the copy from then on; the original is only
// ever opened for reading and is left exactly as it was.
//
// The import is staged through a temporary file and renamed into place, so an
// interrupted copy can never be mistaken on the next start for a completed
// import and leave the process running against a truncated config.
func EnsureWritableRcloneConfig(cfg *Config) (RcloneConfigOrigin, error) {
	if cfg.DataDir == "" {
		return "", fmt.Errorf("data_dir must be set before preparing the rclone config")
	}
	dst := WritableRcloneConfigPath(cfg.DataDir)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("create rclone config dir: %w", err)
	}

	switch info, err := os.Stat(dst); {
	case err == nil:
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory, expected an rclone config file", dst)
		}
		cfg.RcloneConfig = dst
		return RcloneConfigExisting, nil
	case !os.IsNotExist(err):
		return "", fmt.Errorf("stat %s: %w", dst, err)
	}

	source := cfg.RcloneConfig
	cfg.RcloneConfig = dst

	if source != "" && source != dst {
		imported, err := importRcloneConfig(source, dst)
		if err != nil {
			return "", err
		}
		if imported {
			return RcloneConfigImported, nil
		}
	}

	if err := os.WriteFile(dst, nil, 0o600); err != nil {
		return "", fmt.Errorf("create %s: %w", dst, err)
	}
	return RcloneConfigCreated, nil
}

// importRcloneConfig copies source to dst atomically. It reports false when
// there was nothing to import, and never writes to source.
func importRcloneConfig(source, dst string) (bool, error) {
	src, err := os.Open(source)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", source, err)
	}
	defer func() { _ = src.Close() }()

	if info, err := src.Stat(); err != nil {
		return false, fmt.Errorf("stat %s: %w", source, err)
	} else if info.IsDir() {
		return false, fmt.Errorf("%s is a directory, expected an rclone config file", source)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".rclone.conf.import-*")
	if err != nil {
		return false, fmt.Errorf("stage rclone config import: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("copy %s: %w", source, err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("chmod staged rclone config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close staged rclone config: %w", err)
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return false, fmt.Errorf("install %s: %w", dst, err)
	}
	return true, nil
}

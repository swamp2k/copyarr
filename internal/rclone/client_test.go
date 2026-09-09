package rclone

import (
	"errors"
	"strings"
	"testing"
)

func TestSummarizeErrorPrefersTheFailureLine(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "picks the failure line out of noisy stderr",
			err: errors.New("rclone [lsjson probe:]: exit status 1: " +
				"2026/09/09 12:00:00 NOTICE: Config file not found\n" +
				"2026/09/09 12:00:01 ERROR : Failed to create file system for \"probe:\": dial tcp: i/o timeout\n" +
				"2026/09/09 12:00:01 Failed to lsjson with 1 error"),
			want: "Failed to create file system for \"probe:\": dial tcp: i/o timeout",
		},
		{
			// Exactly what rclone v1.75 emits for an unreachable sftp host.
			name: "strips the log prefix and the internal probe name",
			err: errors.New("rclone [config create copyarr_probe sftp]: exit status 1: " +
				"2026/09/09 18:22:33 CRITICAL: Failed to create file system for \"copyarr_probe:\": " +
				"NewFs: couldn't connect SSH: dial tcp 192.0.2.1:22: i/o timeout"),
			want: "Failed to create file system for this remote: NewFs: couldn't connect SSH: dial tcp 192.0.2.1:22: i/o timeout",
		},
		{
			name: "falls back to the last line when nothing looks like a failure",
			err:  errors.New("rclone: exit status 1: first\nsecond\nthird"),
			want: "third",
		},
		{
			name: "single line errors survive unchanged",
			err:  errors.New("boom"),
			want: "boom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarizeError(tc.err); got != tc.want {
				t.Errorf("summarizeError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSummarizeErrorTruncatesRunawayOutput(t *testing.T) {
	got := summarizeError(errors.New(strings.Repeat("x", 900)))
	if len(got) != 403 {
		t.Fatalf("expected a 400 char body plus an ellipsis, got %d chars", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected the truncated message to be marked, got %q", got)
	}
}

func TestRemoteTargetKeepsAbsolutePaths(t *testing.T) {
	tests := []struct {
		name, remote, path, want string
	}{
		{"bare remote", "seedbox", "", "seedbox:"},
		{"relative path", "seedbox", "downloads", "seedbox:downloads"},
		{"absolute path is preserved for local backends", "disk", "/mnt/media", "disk:/mnt/media"},
		{"surrounding whitespace is dropped", "seedbox", "  /incoming  ", "seedbox:/incoming"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := remoteTarget(tc.remote, tc.path); got != tc.want {
				t.Errorf("remoteTarget(%q, %q) = %q, want %q", tc.remote, tc.path, got, tc.want)
			}
		})
	}
}

func TestSortedKeysIsDeterministic(t *testing.T) {
	got := sortedKeys(map[string]string{"user": "m", "host": "h", "pass": "p"})
	want := []string{"host", "pass", "user"}
	if len(got) != len(want) {
		t.Fatalf("sortedKeys() returned %d keys, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedKeys() = %v, want %v", got, want)
		}
	}
}

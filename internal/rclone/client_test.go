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

// Real "rclone config redacted" output for an sftp remote created with a
// password. rclone redacts every value it marks sensitive, not just secrets.
const redactedSFTP = `[box1]
type = sftp
host = XXX
user = XXX
pass = XXX
port = 2222
`

func TestParseRedactedConfigNeverCarriesRedactedValues(t *testing.T) {
	detail := parseRedactedConfig("box1", []byte(redactedSFTP))

	if detail.Name != "box1" {
		t.Errorf("Name = %q", detail.Name)
	}
	if detail.Type != "sftp" {
		t.Errorf("Type = %q, want sftp", detail.Type)
	}

	// Redacted keys are reported by name only.
	wantRedacted := []string{"host", "pass", "user"}
	if len(detail.Redacted) != len(wantRedacted) {
		t.Fatalf("Redacted = %v, want %v", detail.Redacted, wantRedacted)
	}
	for i, k := range wantRedacted {
		if detail.Redacted[i] != k {
			t.Fatalf("Redacted = %v, want %v (sorted)", detail.Redacted, wantRedacted)
		}
	}

	// Nothing redacted may appear in the parameters that reach the browser,
	// not even rclone's XXX placeholder masquerading as a real value.
	for _, k := range wantRedacted {
		if v, present := detail.Parameters[k]; present {
			t.Errorf("redacted key %q leaked into Parameters as %q", k, v)
		}
	}
	for k, v := range detail.Parameters {
		if v == "XXX" {
			t.Errorf("parameter %q kept rclone's redaction placeholder as a value", k)
		}
	}

	// Non-sensitive values still come through so the edit form can prefill.
	if detail.Parameters["port"] != "2222" {
		t.Errorf("port = %q, want 2222", detail.Parameters["port"])
	}
	if _, present := detail.Parameters["type"]; present {
		t.Error("type belongs in Type, not Parameters")
	}
}

func TestParseRedactedConfigIgnoresSectionsAndComments(t *testing.T) {
	in := []byte("# a comment\n\n[box1]\ntype = local\nnot a pair\ncopy_links = true\n")
	detail := parseRedactedConfig("box1", in)
	if detail.Type != "local" {
		t.Errorf("Type = %q", detail.Type)
	}
	if len(detail.Parameters) != 1 || detail.Parameters["copy_links"] != "true" {
		t.Errorf("Parameters = %v, want only copy_links", detail.Parameters)
	}
}

func TestParseRedactedConfigKeepsValuesContainingEquals(t *testing.T) {
	// Base64 and connection strings routinely contain "=", so only the first
	// separator may be treated as the key/value split.
	detail := parseRedactedConfig("r", []byte("[r]\ntype = s3\nsecret = abc==def\n"))
	if got := detail.Parameters["secret"]; got != "abc==def" {
		t.Errorf("secret = %q, want abc==def", got)
	}
}

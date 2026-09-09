package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRetryDefaultsWhenOmitted(t *testing.T) {
	p := writeConfig(t, `{"rules":[{"id":"r","enabled":true,"source":{"path":"/src"},"destination":{"path":"/dst"}}]}`)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	r := c.Rules[0]
	if got := r.RetryLimit(); got != 3 {
		t.Fatalf("RetryLimit()=%d want 3", got)
	}
	if got := r.RetryWait(); got != 300*time.Second {
		t.Fatalf("RetryWait()=%s want 5m", got)
	}
}

func TestZeroRetriesIsExplicitlySupported(t *testing.T) {
	p := writeConfig(t, `{"rules":[{"id":"r","enabled":true,"source":{"path":"/src"},"destination":{"path":"/dst"},"retry_count":0,"retry_wait_seconds":0}]}`)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	r := c.Rules[0]
	if got := r.RetryLimit(); got != 0 {
		t.Fatalf("RetryLimit()=%d want 0", got)
	}
	if got := r.RetryWait(); got != 0 {
		t.Fatalf("RetryWait()=%s want 0", got)
	}
}

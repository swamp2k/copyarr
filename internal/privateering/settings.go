package privateering

import (
	"strconv"
	"strings"
	"time"
)

const (
	metaURL      = "privateering:url"
	metaToken    = "privateering:token"
	metaInterval = "privateering:interval_seconds"

	defaultIntervalSeconds = 300 // 5 minutes
)

// Settings is the user-configured Nexus push target, persisted in Copyarr's
// own SQLite database (via MetaStore) and edited from Copyarr's web UI.
type Settings struct {
	URL             string `json:"nexus_url"`
	Token           string `json:"-"`
	IntervalSeconds int    `json:"push_interval_seconds"`
}

// Enabled reports whether both the URL and token are set.
func (s Settings) Enabled() bool {
	return s.URL != "" && s.Token != ""
}

// Interval returns the configured push interval, defaulting to 5 minutes.
func (s Settings) Interval() time.Duration {
	if s.IntervalSeconds <= 0 {
		return defaultIntervalSeconds * time.Second
	}
	return time.Duration(s.IntervalSeconds) * time.Second
}

// MetaStore is the subset of *db.DB the settings store needs. It is defined
// here (rather than importing internal/db) so this package stays free of a
// dependency on Copyarr's storage layer; *db.DB already satisfies it.
type MetaStore interface {
	Meta(key string) (string, bool, error)
	SetMeta(key, value string) error
	DeleteMeta(key string) error
}

// Store persists Settings in the meta key/value table.
type Store struct {
	db MetaStore
}

func NewStore(db MetaStore) *Store { return &Store{db: db} }

// Get returns the current settings, including the token in plaintext. Never
// expose the result of this call over an API response.
func (s *Store) Get() (Settings, error) {
	var out Settings
	if v, ok, err := s.db.Meta(metaURL); err != nil {
		return out, err
	} else if ok {
		out.URL = v
	}
	if v, ok, err := s.db.Meta(metaToken); err != nil {
		return out, err
	} else if ok {
		out.Token = v
	}
	if v, ok, err := s.db.Meta(metaInterval); err != nil {
		return out, err
	} else if ok {
		if n, err := strconv.Atoi(v); err == nil {
			out.IntervalSeconds = n
		}
	}
	return out, nil
}

// Update applies a partial change. url and intervalSeconds always replace
// the stored value. token only replaces the stored token when setToken is
// true (an empty token with setToken=true clears it) - this lets the save
// endpoint leave an already-configured token alone when the form field was
// left blank.
func (s *Store) Update(url string, setToken bool, token string, intervalSeconds int) error {
	url = NormalizeURL(url)
	if err := s.db.SetMeta(metaURL, url); err != nil {
		return err
	}
	if setToken {
		if token == "" {
			if err := s.db.DeleteMeta(metaToken); err != nil {
				return err
			}
		} else if err := s.db.SetMeta(metaToken, token); err != nil {
			return err
		}
	}
	if err := s.db.SetMeta(metaInterval, strconv.Itoa(intervalSeconds)); err != nil {
		return err
	}
	return nil
}

// NormalizeURL accepts either a Nexus base URL (https://nexus.example) or
// the full ingest URL copied from Nexus's own settings page
// (https://nexus.example/api/privateering/ingest) and returns the base URL
// the Pusher should talk to.
func NormalizeURL(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimRight(u, "/")
	u = strings.TrimSuffix(u, "/api/privateering/ingest")
	return strings.TrimRight(u, "/")
}

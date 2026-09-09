package logging

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/swamp2k/copyarr/internal/db"
)

var enabled atomic.Bool

var store struct {
	sync.RWMutex
	db *db.DB
}

func Install(d *db.DB, on bool) {
	store.Lock()
	store.db = d
	store.Unlock()
	enabled.Store(on)
	base := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(&handler{base: base}))
}

func SetEnabled(on bool) { enabled.Store(on) }
func Enabled() bool      { return enabled.Load() }

func Raw(level, source, message string, jobID *int64) {
	if !enabled.Load() {
		return
	}
	store.RLock()
	d := store.db
	store.RUnlock()
	if d == nil {
		return
	}
	_ = d.AddLog(strings.ToUpper(level), source, message, jobID, "")
}

type handler struct {
	base  slog.Handler
	attrs []slog.Attr
	group string
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.base.Handle(ctx, r); err != nil {
		return err
	}
	if !enabled.Load() {
		return nil
	}

	fields := map[string]any{}
	for _, a := range h.attrs {
		addAttr(fields, h.group, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		addAttr(fields, h.group, a)
		return true
	})

	var jobID *int64
	if v, ok := fields["job"]; ok {
		switch x := v.(type) {
		case int64:
			j := x
			jobID = &j
		case int:
			j := int64(x)
			jobID = &j
		case float64:
			j := int64(x)
			jobID = &j
		}
	}

	raw, _ := json.Marshal(fields)
	store.RLock()
	d := store.db
	store.RUnlock()
	if d != nil {
		_ = d.AddLog(r.Level.String(), "copyarr", r.Message, jobID, string(raw))
	}
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.base = h.base.WithAttrs(attrs)
	cp.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &cp
}

func (h *handler) WithGroup(name string) slog.Handler {
	cp := *h
	cp.base = h.base.WithGroup(name)
	if cp.group == "" {
		cp.group = name
	} else {
		cp.group += "." + name
	}
	return &cp
}

func addAttr(dst map[string]any, group string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	switch a.Value.Kind() {
	case slog.KindInt64:
		dst[key] = a.Value.Int64()
	case slog.KindUint64:
		dst[key] = a.Value.Uint64()
	case slog.KindFloat64:
		dst[key] = a.Value.Float64()
	case slog.KindBool:
		dst[key] = a.Value.Bool()
	case slog.KindString:
		dst[key] = a.Value.String()
	case slog.KindDuration:
		dst[key] = a.Value.Duration().String()
	case slog.KindTime:
		dst[key] = a.Value.Time().Format("2006-01-02T15:04:05.000000000Z07:00")
	case slog.KindGroup:
		for _, ga := range a.Value.Group() {
			addAttr(dst, key, ga)
		}
	default:
		dst[key] = a.Value.Any()
	}
}

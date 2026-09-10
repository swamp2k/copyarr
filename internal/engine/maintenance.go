package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/swamp2k/copyarr/internal/db"
	"github.com/swamp2k/copyarr/internal/logging"
)

const defaultLogRetentionDays = 7

type ExtendedSettings struct {
	LoggingEnabled   bool `json:"logging_enabled"`
	LogRetentionDays int  `json:"log_retention_days"`
}

type ExecutionRow struct {
	JobView
	TransferAt      *string `json:"transfer_at,omitempty"`
	DurationSeconds float64 `json:"duration_seconds"`
	AvgSpeedBps     float64 `json:"avg_speed_bps"`
	PeakSpeedBps    float64 `json:"peak_speed_bps"`
}

func (e *Engine) SettingsExtended() ExtendedSettings {
	return ExtendedSettings{
		LoggingEnabled:   logging.Enabled(),
		LogRetentionDays: e.LogRetentionDays(),
	}
}

func (e *Engine) LogRetentionDays() int {
	v, ok, err := e.db.Meta("settings:log_retention_days")
	if err != nil || !ok {
		return defaultLogRetentionDays
	}
	days, err := strconv.Atoi(v)
	if err != nil || days < 1 {
		return defaultLogRetentionDays
	}
	return days
}

func (e *Engine) SetLogRetentionDays(days int) error {
	if days < 1 || days > 3650 {
		return fmt.Errorf("log retention must be between 1 and 3650 days")
	}
	if err := e.db.SetMeta("settings:log_retention_days", strconv.Itoa(days)); err != nil {
		return err
	}
	if _, err := e.cleanupExpiredLogs(); err != nil {
		return err
	}
	slog.Info("log retention changed", "days", days)
	return nil
}

func (e *Engine) cleanupExpiredLogs() (int64, error) {
	before := time.Now().UTC().Add(-time.Duration(e.LogRetentionDays()) * 24 * time.Hour)
	return e.db.DeleteLogsBefore(before)
}

// RunMaintenance performs an immediate retention pass on startup and then
// repeats it hourly. It is separate from transfer scheduling so log cleanup can
// never block scans or the queue worker.
func (e *Engine) RunMaintenance(ctx context.Context) {
	if n, err := e.cleanupExpiredLogs(); err != nil {
		slog.Warn("log retention cleanup failed", "err", err)
	} else if n > 0 {
		slog.Info("expired log entries removed", "count", n, "retention_days", e.LogRetentionDays())
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := e.cleanupExpiredLogs(); err != nil {
					slog.Warn("log retention cleanup failed", "err", err)
				} else if n > 0 {
					slog.Info("expired log entries removed", "count", n, "retention_days", e.LogRetentionDays())
				}
			}
		}
	}()
}

func (e *Engine) ExecutionRows(limit int) ([]ExecutionRow, error) {
	jobs, err := e.Jobs(limit)
	if err != nil {
		return nil, err
	}
	peaks, err := e.db.PeakJobSpeeds()
	if err != nil {
		return nil, err
	}
	out := make([]ExecutionRow, 0, len(jobs))
	for _, j := range jobs {
		row := ExecutionRow{JobView: j, PeakSpeedBps: peaks[j.ID]}
		row.TransferAt = j.StartedAt
		if row.TransferAt == nil {
			created := j.CreatedAt
			row.TransferAt = &created
		}
		if j.StartedAt != nil {
			start, err := time.Parse(time.RFC3339Nano, *j.StartedAt)
			if err == nil {
				end := time.Now().UTC()
				bytesForAverage := int64(0)
				if j.CompletedAt != nil {
					if parsed, parseErr := time.Parse(time.RFC3339Nano, *j.CompletedAt); parseErr == nil {
						end = parsed
						bytesForAverage = j.TotalBytes
					}
				} else {
					stats, statErr := e.db.ListJobStats(j.ID, 5000)
					if statErr == nil && len(stats) > 0 {
						bytesForAverage = stats[len(stats)-1].TransferredBytes
					}
				}
				secs := end.Sub(start).Seconds()
				if secs > 0 {
					row.DurationSeconds = secs
					if bytesForAverage > 0 {
						row.AvgSpeedBps = float64(bytesForAverage) / secs
					}
				}
			}
		}
		out = append(out, row)
	}
	return out, nil
}

var _ = db.Job{}

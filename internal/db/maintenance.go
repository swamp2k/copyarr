package db

import "time"

// DeleteLogsBefore removes persisted log entries older than the supplied time.
// Transfer history and job_stats are deliberately independent of log retention.
func (d *DB) DeleteLogsBefore(before time.Time) (int64, error) {
	res, err := d.Exec(`DELETE FROM logs WHERE ts < ?`, before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PeakJobSpeeds returns the highest sampled transfer speed for every job that
// has telemetry. Jobs created before telemetry was introduced simply won't be
// present in the map.
func (d *DB) PeakJobSpeeds() (map[int64]float64, error) {
	rows, err := d.Query(`SELECT job_id,COALESCE(MAX(speed_bps),0) FROM job_stats GROUP BY job_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]float64)
	for rows.Next() {
		var id int64
		var speed float64
		if err := rows.Scan(&id, &speed); err != nil {
			return nil, err
		}
		out[id] = speed
	}
	return out, rows.Err()
}

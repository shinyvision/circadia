package storage

import (
	"fmt"
	"time"
)

type SleepSession struct {
	ID          int
	StartTime   time.Time
	EndTime     time.Time
	SnoozeCount int
}

func AddSleepSession(startTime, endTime time.Time, snoozeCount int) error {
	query := `
	INSERT INTO sleep_history (start_time, end_time, snooze_count)
	VALUES (?, ?, ?)
	`
	_, err := DB.Exec(query, startTime, endTime, snoozeCount)
	if err != nil {
		return fmt.Errorf("failed to add sleep session: %w", err)
	}

	pruneQuery := `DELETE FROM sleep_history WHERE start_time < ?`
	cutoff := time.Now().AddDate(0, 0, -30)
	_, err = DB.Exec(pruneQuery, cutoff)
	if err != nil {
		fmt.Printf("Warning: failed to prune old sleep history: %v\n", err)
	}

	return nil
}

func GetHistory(days int) ([]SleepSession, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
	SELECT id, start_time, end_time, snooze_count
	FROM sleep_history
	WHERE start_time >= ?
	ORDER BY start_time ASC
	`
	rows, err := DB.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query sleep history: %w", err)
	}
	defer rows.Close()

	var sessions []SleepSession
	for rows.Next() {
		var s SleepSession
		if err := rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.SnoozeCount); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func GetLastSleepSession() (*SleepSession, error) {
	query := `
	SELECT id, start_time, end_time, snooze_count
	FROM sleep_history
	ORDER BY end_time DESC
	LIMIT 1
	`
	var s SleepSession
	err := DB.QueryRow(query).Scan(&s.ID, &s.StartTime, &s.EndTime, &s.SnoozeCount)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

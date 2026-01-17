package storage

import (
	"fmt"
)

type Alarm struct {
	ID      int64
	Hour    int
	Minute  int
	Enabled bool
}

func AddAlarm(hour, minute int) error {
	_, err := DB.Exec("INSERT INTO alarms (hour, minute, enabled) VALUES (?, ?, ?)", hour, minute, true)
	if err != nil {
		return fmt.Errorf("failed to add alarm: %w", err)
	}
	return nil
}

func GetAlarms() ([]Alarm, error) {
	rows, err := DB.Query("SELECT id, hour, minute, enabled FROM alarms ORDER BY hour, minute ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query alarms: %w", err)
	}
	defer rows.Close()

	var alarms []Alarm
	for rows.Next() {
		var a Alarm
		if err := rows.Scan(&a.ID, &a.Hour, &a.Minute, &a.Enabled); err != nil {
			return nil, err
		}
		alarms = append(alarms, a)
	}
	return alarms, nil
}

func UpdateAlarm(id int64, hour, minute int, enabled bool) error {
	_, err := DB.Exec("UPDATE alarms SET hour = ?, minute = ?, enabled = ? WHERE id = ?", hour, minute, enabled, id)
	if err != nil {
		return fmt.Errorf("failed to update alarm: %w", err)
	}
	return nil
}

func DeleteAlarm(id int64) error {
	_, err := DB.Exec("DELETE FROM alarms WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete alarm: %w", err)
	}
	return nil
}

func ToggleAlarm(id int64, enabled bool) error {
	_, err := DB.Exec("UPDATE alarms SET enabled = ? WHERE id = ?", enabled, id)
	if err != nil {
		return fmt.Errorf("failed to toggle alarm: %w", err)
	}
	return nil
}

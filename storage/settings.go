package storage

import (
	"fmt"
	"time"
)

func GetSetting(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func SetSetting(key, value string) error {
	query := `
    INSERT INTO settings (key, value) 
    VALUES (?, ?)
    ON CONFLICT(key) DO UPDATE SET value = excluded.value;
    `
	_, err := DB.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}

func GetBedtime() (string, error) {
	return GetSetting("bedtime")
}

func SetBedtime(timeStr string) error {
	return SetSetting("bedtime", timeStr)
}

func GetNotifyBedtime() (bool, error) {
	val, err := GetSetting("notify_bedtime")
	if err != nil {
		return false, err
	}
	return val == "true", nil
}

func SetNotifyBedtime(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return SetSetting("notify_bedtime", val)
}

func GetSmartWakeUp() (bool, error) {
	val, err := GetSetting("smart_wake_up")
	if err != nil {
		return true, nil
	}
	return val == "true", nil
}

func SetSmartWakeUp(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return SetSetting("smart_wake_up", val)
}

func GetSleepStartTime() (time.Time, error) {
	val, err := GetSetting("sleep_start_time")
	if err != nil || val == "" {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}

func SetSleepStartTime(t time.Time) error {
	return SetSetting("sleep_start_time", t.Format(time.RFC3339))
}

func ClearSleepStartTime() error {
	return SetSetting("sleep_start_time", "")
}

func GetAlarmAudioPath() (string, error) {
	val, err := GetSetting("alarm_audio_path")
	if err != nil {
		return "", nil
	}
	return val, nil
}

func SetAlarmAudioPath(path string) error {
	return SetSetting("alarm_audio_path", path)
}

func GetSnoozeDuration() (int, error) {
	val, err := GetSetting("snooze_duration")
	if err != nil {
		return 15, nil
	}
	var d int
	_, err = fmt.Sscanf(val, "%d", &d)
	if err != nil {
		return 15, nil
	}
	return d, nil
}

func SetSnoozeDuration(minutes int) error {
	return SetSetting("snooze_duration", fmt.Sprintf("%d", minutes))
}

func GetSnoozeEnabled() (bool, error) {
	val, err := GetSetting("snooze_enabled")
	if err != nil {
		return true, nil
	}
	return val == "true", nil
}

func SetSnoozeEnabled(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return SetSetting("snooze_enabled", val)
}

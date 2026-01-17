package daemon

import (
	"fmt"
	"log"
	"time"

	"circadia/internal/ipc"
	"circadia/storage"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

var OnAlarmTriggered func(h, m int)
var OnSleepModeChanged func(enabled bool)
var OnSmartWakeUpToggled func(enabled bool)

var globalApp *gio.Application

func Start(app *gio.Application) {
	globalApp = app
	log.Println("Daemon starting...")

	err := ipc.StartListener(func(msg string) {
		log.Printf("Received signal: %s", msg)
		if msg == "bedtimeChanged" {
			resetNotificationState()
		} else if len(msg) > 15 && msg[:15] == "alarmTriggered:" {
			if OnAlarmTriggered != nil {
				var h, m int
				fmt.Sscanf(msg[15:], "%d:%d", &h, &m)
				glib.IdleAdd(func() {
					OnAlarmTriggered(h, m)
				})
			}
		} else if len(msg) > 17 && msg[:17] == "sleepModeChanged:" {
			var enabled bool
			fmt.Sscanf(msg[17:], "%t", &enabled)
			if OnSleepModeChanged != nil {
				glib.IdleAdd(func() {
					OnSleepModeChanged(enabled)
				})
			}
		} else if len(msg) > 19 && msg[:19] == "smartWakeUpToggled:" {
			var enabled bool
			fmt.Sscanf(msg[19:], "%t", &enabled)
			if OnSmartWakeUpToggled != nil {
				glib.IdleAdd(func() {
					OnSmartWakeUpToggled(enabled)
					RestartTicker()
				})
			}
		}
	})
	if err != nil {
		log.Printf("Failed to start IPC listener: %v", err)
	}

	startTicker(app)
}

var lastNotifiedTime string

func resetNotificationState() {
	lastNotifiedTime = ""
}

func checkAndNotify(app *gio.Application) {
	enabled, err := storage.GetNotifyBedtime()
	if err != nil || !enabled {
		return
	}

	bedtimeStr, err := storage.GetBedtime()
	if err != nil {
		return
	}

	bedtime, err := time.Parse("15:04", bedtimeStr)
	if err != nil {
		log.Printf("Invalid bedtime format: %v", err)
		return
	}

	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), bedtime.Hour(), bedtime.Minute(), 0, 0, now.Location())

	diff := now.Sub(target)
	if diff >= 0 && diff < time.Minute {
		if lastNotifiedTime != "bedtime:"+bedtimeStr {
			sendNotification(app, "It's Bedtime", "Sleep tight!")
			lastNotifiedTime = "bedtime:" + bedtimeStr
		}
	}

	target30 := target.Add(-30 * time.Minute)
	diff30 := now.Sub(target30)
	if diff30 >= 0 && diff30 < time.Minute {
		if lastNotifiedTime != "30min:"+bedtimeStr {
			sendNotification(app, "Wind Down", "Bedtime in 30 minutes.")
			lastNotifiedTime = "30min:" + bedtimeStr
		}
	}

	checkAlarms(app, now)
}

var audioPreloadActive bool
var lastAudioPreloadSuccess time.Time

func runPreloadLoop() {
	if audioPreloadActive {
		return
	}
	audioPreloadActive = true
	log.Println("Starting Audio Preload Loop...")

	go func() {
		defer func() { audioPreloadActive = false }()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		timeout := time.After(40 * time.Minute)

		if err := ForceSpeakerOutput(); err == nil {
			log.Println("Audio Preload Success!")
			lastAudioPreloadSuccess = time.Now()
			return
		}

		for {
			select {
			case <-timeout:
				log.Println("Audio Preload Timeout")
				return
			case <-ticker.C:
				log.Println("Audio Preload Attempt...")
				if err := ForceSpeakerOutput(); err == nil {
					log.Println("Audio Preload Success!")
					lastAudioPreloadSuccess = time.Now()
					return
				}
			}
		}
	}()
}

var lastTriggeredTime time.Time

func checkAlarms(app *gio.Application, now time.Time) {
	alarms, err := storage.GetAlarms()
	if err != nil {
		log.Printf("Error checking alarms: %v", err)
		return
	}

	if !audioPreloadActive {
		smartEnabled, _ := storage.GetSmartWakeUp()
		preloadWindow := 5 * time.Minute
		if smartEnabled {
			preloadWindow = 35 * time.Minute
		}

		if lastAudioPreloadSuccess.IsZero() || time.Since(lastAudioPreloadSuccess) >= preloadWindow {
			for _, a := range alarms {
				if !a.Enabled {
					continue
				}
				target := time.Date(now.Year(), now.Month(), now.Day(), a.Hour, a.Minute, 0, 0, now.Location())
				if target.Before(now) {
					target = target.Add(24 * time.Hour)
				}
				diff := target.Sub(now)
				if diff > 0 && diff <= preloadWindow {
					runPreloadLoop()
					break
				}
			}
		}
	}

	if !lastTriggeredTime.IsZero() && now.Sub(lastTriggeredTime) < 1*time.Minute {
		return
	}

	currentHour := now.Hour()
	currentMinute := now.Minute()

	for _, alarm := range alarms {
		if alarm.Enabled && alarm.Hour == currentHour && alarm.Minute == currentMinute {
			lastTriggeredTime = now
			triggerAlarm(app, alarm)
			return
		}
	}
}

var activeAlarmID int64 = -1

func IsRinging() bool {
	return activeAlarmID != -1
}

func triggerAlarm(app *gio.Application, alarm storage.Alarm) {
	if activeAlarmID == alarm.ID {
		return
	}
	activeAlarmID = alarm.ID

	log.Printf("ALARM TRIGGERED: %d:%02d", alarm.Hour, alarm.Minute)

	StartAlarmSound()

	app.Activate()

	if err := ipc.SendSignal(fmt.Sprintf("alarmTriggered:%d:%02d", alarm.Hour, alarm.Minute)); err != nil {
		log.Printf("Failed to signal alarm to UI: %v", err)
	}
}

var snoozeTimer *time.Timer

func StopAlarm() {
	StopAlarmSound()
	activeAlarmID = -1
	if snoozeTimer != nil {
		snoozeTimer.Stop()
		snoozeTimer = nil
	}
}

func sendNotification(app *gio.Application, title, body string) {
	notification := gio.NewNotification(title)
	notification.SetBody(body)
	app.SendNotification(title, notification)
}

func ToggleSleepMode(enabled bool) {
	log.Printf("Sleep Mode Toggled: %v", enabled)

	go func() {
		msg := fmt.Sprintf("sleepModeChanged:%v", enabled)
		if err := ipc.SendSignal(msg); err != nil {
			log.Println("IPC Signal Error:", err)
		}
	}()
}

var (
	alarmTicker *time.Ticker
	tickerQuit  chan struct{}
)

func startTicker(app *gio.Application) {
	RestartTicker()
}

func RestartTicker() {
	if alarmTicker != nil {
		alarmTicker.Stop()
		close(tickerQuit)
	}

	tickerQuit = make(chan struct{})

	smartEnabled, _ := storage.GetSmartWakeUp()
	interval := 30 * time.Second
	if smartEnabled {
		interval = 29 * time.Minute
	}

	log.Printf("Starting Ticker with interval: %v (Smart Wake Up: %v)", interval, smartEnabled)
	alarmTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-tickerQuit:
				return
			case <-alarmTicker.C:
				if globalApp != nil {
					glib.IdleAdd(func() {
						checkAndNotify(globalApp)
						checkSmartWakeUp()
					})
				}
			}
		}
	}()
}

func checkSmartWakeUp() {
	log.Println("Checking Smart Wake Up...")

	smartEnabled, err := storage.GetSmartWakeUp()
	if err != nil {
		log.Println("Error reading smart wake up setting:", err)
		return
	}
	if !smartEnabled {
		return
	}

	alarms, err := storage.GetAlarms()
	if err != nil {
		log.Println("Error reading alarms for smart wake up:", err)
		return
	}

	now := time.Now()

	for _, a := range alarms {
		if !a.Enabled {
			continue
		}

		target := time.Date(now.Year(), now.Month(), now.Day(), a.Hour, a.Minute, 0, 0, now.Location())

		if target.Before(now) {
			target = target.Add(24 * time.Hour)
		}

		diff := target.Sub(now)
		if diff > 0 && diff <= 30*time.Minute {
			log.Printf("Smart Wake Up Triggered for alarm at %02d:%02d (in %v)", a.Hour, a.Minute, diff)

			if globalApp != nil {
				glib.IdleAdd(func() {
					triggerAlarm(globalApp, a)
					// Sleep Mode remains active until alarm is stopped
				})
			}
			return
		}
	}
}

var OnSleepSessionSaved func()

func IsSleepModeEnabled() bool {
	t, err := storage.GetSleepStartTime()
	return err == nil && !t.IsZero()
}

var debugMode bool

func SetDebugMode(enabled bool) {
	debugMode = enabled
	log.Printf("Daemon Debug Mode: %v", debugMode)
}

func SnoozeAlarm() {
	StopAlarmSound()
	activeAlarmID = -1

	if snoozeTimer != nil {
		snoozeTimer.Stop()
	}

	log.Println("Snoozing...")
	durationMin, err := storage.GetSnoozeDuration()
	if err != nil {
		durationMin = 15
	}

	log.Printf("Snoozing for %d minutes...", durationMin)
	snoozeTimer = time.AfterFunc(time.Duration(durationMin)*time.Minute, func() {
		log.Println("Snooze finished! Ringing again.")

		StartAlarmSound()

		if globalApp != nil {
			glib.IdleAdd(func() {
				globalApp.Activate()
			})
		}

		h, m := time.Now().Hour(), time.Now().Minute()
		if err := ipc.SendSignal(fmt.Sprintf("alarmTriggered:%d:%02d", h, m)); err != nil {
			log.Printf("Failed to signal alarm to UI: %v", err)
		}
	})
}

func FinalizeSleepSession(startTime, endTime time.Time, snoozeCount int, bypassDurationCheck bool) error {
	duration := endTime.Sub(startTime)

	// Only save if duration > 1 hour OR if check is bypassed (e.g. Alarm Stop)
	if bypassDurationCheck || duration > 1*time.Hour {
		// Save to DB
		if err := storage.AddSleepSession(startTime, endTime, snoozeCount); err != nil {
			log.Printf("Failed to save sleep session: %v", err)
			return err
		}
		log.Printf("Sleep Session Saved: %v - %v (Snoozes: %d)", startTime, endTime, snoozeCount)

		if OnSleepSessionSaved != nil {
			glib.IdleAdd(func() {
				OnSleepSessionSaved()
			})
		}

		return nil
	}

	log.Printf("Sleep Session discarded (too short): %v", duration)
	return nil
}

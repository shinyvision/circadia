package main

import (
	"circadia/daemon"
	"circadia/storage"
	"circadia/ui"
	"circadia/ui/pages"
	"log"
	"os"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func NewWindow(app *gtk.Application, debugMode bool) *gtk.ApplicationWindow {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Circadia")
	window.SetResizable(true)
	window.SetDecorated(false)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)

	overlay := gtk.NewOverlay()
	overlay.SetChild(mainBox)
	window.SetChild(overlay)

	tabs, btnSetAlarm, btnHistory, btnSettings := ui.NewTabSelector()
	mainBox.Append(tabs)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)
	mainBox.Append(scrolled)

	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	goodnightBox := gtk.NewBox(gtk.OrientationVertical, 0)
	goodnightBox.AddCSSClass("goodnight-overlay")
	goodnightBox.SetHExpand(true)
	goodnightBox.SetVExpand(true)
	goodnightBox.SetHAlign(gtk.AlignFill)
	goodnightBox.SetVAlign(gtk.AlignFill)
	goodnightBox.SetVisible(false)

	goodnightLabel := gtk.NewLabel("")
	goodnightLabel.AddCSSClass("goodnight-text")
	goodnightLabel.SetHExpand(true)
	goodnightLabel.SetVExpand(true)
	goodnightLabel.SetHAlign(gtk.AlignCenter)
	goodnightLabel.SetVAlign(gtk.AlignCenter)

	goodnightBox.Append(goodnightLabel)
	overlay.AddOverlay(goodnightBox)

	mainBox.AddCSSClass("content-transition")

	fab := gtk.NewButton()

	iconPath := "assets/icons/ui/moon.svg"
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		iconPath = "/app/share/circadia/assets/icons/ui/moon.svg"
	}

	icon := gtk.NewImageFromFile(iconPath)
	icon.SetPixelSize(35)

	fab.SetChild(icon)
	fab.AddCSSClass("fab-btn")
	fab.SetHAlign(gtk.AlignEnd)
	fab.SetVAlign(gtk.AlignEnd)
	fab.SetVisible(true)

	var snoozeCount int

	ringingBox := gtk.NewBox(gtk.OrientationVertical, 0)
	ringingBox.AddCSSClass("ringing-overlay")
	ringingBox.SetHExpand(true)
	ringingBox.SetVExpand(true)
	ringingBox.SetVisible(false)
	ringingBox.AddCSSClass("black-background")

	overlay.AddOverlay(ringingBox)

	var typewriterSourceId glib.SourceHandle

	startTypewriter := func(text string) {
		if typewriterSourceId > 0 {
			glib.SourceRemove(typewriterSourceId)
			typewriterSourceId = 0
		}

		runes := []rune(text)
		idx := 0
		goodnightLabel.SetText("")

		typewriterSourceId = glib.TimeoutAdd(100, func() bool {
			if idx >= len(runes) {
				typewriterSourceId = 0
				return false
			}
			current := goodnightLabel.Text()
			goodnightLabel.SetText(current + string(runes[idx]))
			idx++
			return true
		})
	}

	fab.ConnectClicked(func() {
		isActive := fab.HasCSSClass("fab-active")
		newState := !isActive
		daemon.ToggleSleepMode(newState)
	})

	overlay.AddOverlay(fab)

	showModal := func(widget *gtk.Widget) func() {
		dimmer := gtk.NewBox(gtk.OrientationVertical, 0)
		dimmer.AddCSSClass("modal-dimmer")
		dimmer.AddCSSClass("modal-dimmer-in")
		dimmer.SetHExpand(true)
		dimmer.SetVExpand(true)

		widget.SetVExpand(true)
		widget.SetVAlign(gtk.AlignCenter)
		widget.SetHExpand(true)
		widget.SetHAlign(gtk.AlignCenter)

		widget.AddCSSClass("modal-content-in")

		if widget.Parent() != nil {
			log.Println("Warning: Widget passed to showModal already has a parent. Unparenting.")
			widget.Unparent()
		}

		dimmer.Append(widget)

		overlay.AddOverlay(dimmer)

		return func() {
			dimmer.RemoveCSSClass("modal-dimmer-in")
			dimmer.AddCSSClass("modal-dimmer-out")

			widget.RemoveCSSClass("modal-content-in")
			widget.AddCSSClass("modal-content-out")

			glib.TimeoutAdd(300, func() bool {
				overlay.RemoveOverlay(dimmer)
				widget.RemoveCSSClass("modal-content-out")
				return false
			})
		}
	}

	setAlarmPage := pages.NewSetAlarmPage(showModal)
	stack.AddNamed(setAlarmPage, "set_alarm")

	sleepHistoryCtrl := pages.NewSleepHistoryPage()
	stack.AddNamed(sleepHistoryCtrl.Box, "sleep_history")

	settingsPage := pages.NewSettingsPage(debugMode)
	stack.AddNamed(settingsPage, "settings")

	scrolled.SetChild(stack)

	btnSetAlarm.ConnectClicked(func() {
		stack.SetVisibleChildName("set_alarm")
		btnSetAlarm.AddCSSClass("active")
		btnHistory.RemoveCSSClass("active")
		btnSettings.RemoveCSSClass("active")
	})

	btnHistory.ConnectClicked(func() {
		sleepHistoryCtrl.Refresh()
		stack.SetVisibleChildName("sleep_history")
		btnHistory.AddCSSClass("active")
		btnSetAlarm.RemoveCSSClass("active")
		btnSettings.RemoveCSSClass("active")
	})

	btnSettings.ConnectClicked(func() {
		stack.SetVisibleChildName("settings")
		btnSettings.AddCSSClass("active")
		btnSetAlarm.RemoveCSSClass("active")
		btnHistory.RemoveCSSClass("active")
	})

	ringingPage := pages.NewRingingPage(func(action string) {
		ringingBox.SetVisible(false)

		if action == "snooze" {
			log.Println("Alarm Snoozed")
			snoozeCount++

			if daemon.IsSleepModeEnabled() {
				log.Println("Sleep Mode is Active - Restoring Goodnight Overlay")
				goodnightLabel.SetText("Goodnight")
				goodnightLabel.SetVisible(true)
				goodnightBox.SetVisible(true)
				fab.SetVisible(true)
				fab.AddCSSClass("fab-active")

				mainBox.AddCSSClass("hidden")
				tabs.SetVisible(false)
			} else {
				tabs.SetVisible(true)
				fab.SetVisible(true)
			}

		} else if action == "stop" {
			log.Println("Alarm Stopped")
			fab.SetVisible(true)

			if startTime, err := storage.GetSleepStartTime(); err == nil && !startTime.IsZero() {
				endTime := time.Now()

				if err := daemon.FinalizeSleepSession(startTime, endTime, snoozeCount, true); err != nil {
					log.Printf("Error finalizing sleep session: %v", err)
				}

				storage.ClearSleepStartTime()

				daemon.ToggleSleepMode(false)

				goodnightBox.SetVisible(false)
				goodnightLabel.SetText("")
				fab.RemoveCSSClass("fab-active")
			}

			snoozeCount = 0

			tabs.SetVisible(true)
			stack.SetVisibleChildName("set_alarm")
		}
	})

	ringingBox.Append(ringingPage)

	daemon.OnAlarmTriggered = func(h, m int) {
		log.Printf("UI Handling Alarm: %d:%02d", h, m)
		tabs.SetVisible(false)

		ringingBox.SetVisible(true)

		fab.SetVisible(false)

		// Only present if not already active (focused)
		if !window.IsActive() {
			if !window.IsVisible() {
				window.SetVisible(true)
			}
			window.Present()
		}
	}

	daemon.OnSleepSessionSaved = func() {
		log.Println("Sleep session saved, refreshing history...")
		sleepHistoryCtrl.Refresh()
	}

	daemon.OnSleepModeChanged = func(enabled bool) {
		log.Printf("UI Sleep Mode: %v", enabled)
		if enabled {
			fab.AddCSSClass("fab-active")

			if err := storage.SetSleepStartTime(time.Now()); err != nil {
				log.Printf("Failed to set sleep start time: %v", err)
			}
			snoozeCount = 0

			goodnightBox.SetVisible(true)
			goodnightBox.AddCSSClass("visible")
			mainBox.AddCSSClass("hidden")

			glib.TimeoutAdd(500, func() bool {
				goodnightLabel.SetText("")
				startTypewriter("Goodnight")
				return false
			})

		} else {
			fab.RemoveCSSClass("fab-active")

			if startTime, err := storage.GetSleepStartTime(); err == nil && !startTime.IsZero() {
				endTime := time.Now()
				if err := daemon.FinalizeSleepSession(startTime, endTime, snoozeCount, false); err != nil {
					log.Printf("Manual disable save error: %v", err)
				}
			}

			storage.ClearSleepStartTime()

			goodnightBox.RemoveCSSClass("visible")
			mainBox.RemoveCSSClass("hidden")

			glib.TimeoutAdd(750, func() bool {
				if !fab.HasCSSClass("fab-active") {
					goodnightBox.SetVisible(false)
					goodnightLabel.SetText("")
				}
				return false
			})
		}
	}

	if daemon.IsRinging() {
		tabs.SetVisible(false)
		ringingBox.SetVisible(true)
		fab.SetVisible(false)
	} else {
		stack.SetVisibleChildName("set_alarm")
	}

	window.ConnectCloseRequest(func() bool {
		log.Println("Window close requested - Hiding window")
		window.SetVisible(false)
		return true
	})

	return window
}

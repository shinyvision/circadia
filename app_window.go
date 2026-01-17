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

func NewWindow(app *gtk.Application) *gtk.ApplicationWindow {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Circadia")
	window.SetDefaultSize(375, 812) // iPhone X dimensions approximate
	window.SetResizable(true)
	window.SetDecorated(false) // No title bar for mobile-like feel

	// Main container - vertical box
	// Tabs will be at top (sticky), scrolled window below
	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)

	// Create Overlay
	overlay := gtk.NewOverlay()
	overlay.SetChild(mainBox)
	window.SetChild(overlay)

	// Tabs - Edge to Edge (Sticky at top)
	tabs, btnSetAlarm, btnHistory, btnSettings := ui.NewTabSelector()
	mainBox.Append(tabs)

	// Scrolled area for content
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true) // Take remaining space
	mainBox.Append(scrolled)

	// Stack for pages
	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	// Sleep Mode Overlay
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

	// Apply transition class to mainBox to allow fading
	mainBox.AddCSSClass("content-transition")

	// FAB Logic
	fab := gtk.NewButton()

	// Icon Path Logic
	iconPath := "assets/icons/ui/moon.svg"
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		// Try fallback global path for Flatpak
		iconPath = "/app/share/circadia/assets/icons/ui/moon.svg"
	}

	icon := gtk.NewImageFromFile(iconPath)
	icon.SetPixelSize(35)

	fab.SetChild(icon)
	fab.AddCSSClass("fab-btn")
	fab.SetHAlign(gtk.AlignEnd)
	fab.SetVAlign(gtk.AlignEnd)
	fab.SetVisible(true) // Always visible now

	// Sleep Tracking
	var snoozeCount int

	// Ringing Overlay (Must be added AFTER goodnightBox to be on top)
	ringingBox := gtk.NewBox(gtk.OrientationVertical, 0)
	ringingBox.AddCSSClass("ringing-overlay") // We'll need to define this in CSS if not exists, or just use background
	ringingBox.SetHExpand(true)
	ringingBox.SetVExpand(true)
	ringingBox.SetVisible(false)
	// We need a white/background to cover the goodnight overlay
	// Actually RingingPage probably has its own background?
	// Let's ensure it covers.
	ringingBox.AddCSSClass("black-background")

	overlay.AddOverlay(ringingBox)

	// Typewriter state
	var typewriterSourceId glib.SourceHandle

	startTypewriter := func(text string) {
		// Cancel existing if any
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

	// Click Handler for FAB
	fab.ConnectClicked(func() {
		isActive := fab.HasCSSClass("fab-active")
		newState := !isActive
		daemon.ToggleSleepMode(newState)
	})

	overlay.AddOverlay(fab)

	// Modal Logic
	showModal := func(widget *gtk.Widget) func() {
		dimmer := gtk.NewBox(gtk.OrientationVertical, 0)
		// ... (rest of showModal)
		dimmer.AddCSSClass("modal-dimmer")
		dimmer.AddCSSClass("modal-dimmer-in") // Fade In Logic
		dimmer.SetHExpand(true)
		dimmer.SetVExpand(true)

		// Center the widget logic is inside the widget mostly, but we ensure dimmer fills overlay
		// and we put widget inside dimmer.
		// To center it within the dimmer (VBox), we set Expand and Align on the widget.
		widget.SetVExpand(true)
		widget.SetVAlign(gtk.AlignCenter)
		widget.SetHExpand(true)
		widget.SetHAlign(gtk.AlignCenter)

		// Apply Scale In Logic
		widget.AddCSSClass("modal-content-in")

		// Safety check: Ensure widget has no parent
		if widget.Parent() != nil {
			log.Println("Warning: Widget passed to showModal already has a parent. Unparenting.")
			widget.Unparent()
		}

		dimmer.Append(widget)

		overlay.AddOverlay(dimmer)

		return func() {
			// Trigger Close Animation
			dimmer.RemoveCSSClass("modal-dimmer-in")
			dimmer.AddCSSClass("modal-dimmer-out")

			widget.RemoveCSSClass("modal-content-in")
			widget.AddCSSClass("modal-content-out")

			// Wait for animation to finish (300ms) before removing
			glib.TimeoutAdd(300, func() bool {
				overlay.RemoveOverlay(dimmer)
				// Cleanup style classes in case widget is reused (though we typically recreate)
				widget.RemoveCSSClass("modal-content-out")
				return false // Don't repeat
			})
		}
	}

	// Pages
	setAlarmPage := pages.NewSetAlarmPage(showModal)
	stack.AddNamed(setAlarmPage, "set_alarm")

	sleepHistoryPage := pages.NewSleepHistoryPage()
	stack.AddNamed(sleepHistoryPage, "sleep_history")

	settingsPage := pages.NewSettingsPage()
	stack.AddNamed(settingsPage, "settings")

	scrolled.SetChild(stack)

	// Connect Signals
	btnSetAlarm.ConnectClicked(func() {
		stack.SetVisibleChildName("set_alarm")
		btnSetAlarm.AddCSSClass("active")
		btnHistory.RemoveCSSClass("active")
		btnSettings.RemoveCSSClass("active")
	})

	btnHistory.ConnectClicked(func() {
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

	// Add listener for Alarm Trigger
	// We need a way to trigger this from outside.
	// Let's create a global hook or export a function on the window.
	// Go doesn't allow easy method extension on standard types without embedding.
	// But we can return a custom struct or just use a closure?
	// For now, let's rely on main.go wiring.

	// Add Ringing Logic
	// We instantiate RingingPage but we put it in the ringingBox overlay
	ringingPage := pages.NewRingingPage(func(action string) {
		// Action is "snooze" or "stop"
		ringingBox.SetVisible(false) // Hide ringing

		if action == "snooze" {
			log.Println("Alarm Snoozed")
			snoozeCount++
			// Restore tabs visibility as alarm UI is hidden
			tabs.SetVisible(true)
			// Sleep mode remains active if it was active
			// Go back to underlying state (which might be Goodnight overlay)
		} else if action == "stop" {
			log.Println("Alarm Stopped")
			// Calculate Sleep Duration if Sleep Mode was active
			if startTime, err := storage.GetSleepStartTime(); err == nil && !startTime.IsZero() {
				// We have a valid start time
				endTime := time.Now()
				// Save to DB
				if err := storage.AddSleepSession(startTime, endTime, snoozeCount); err != nil {
					log.Printf("Failed to save sleep session: %v", err)
				}
				log.Printf("Sleep Session Saved: %v - %v (Snoozes: %d)", startTime, endTime, snoozeCount)

				// Reset Start Time
				storage.ClearSleepStartTime()

				// Auto Disable Sleep Mode
				daemon.ToggleSleepMode(false)

				// Requirements: "goodnight overlay gets hidden immediately without any transition"
				// ToggleSleepMode(false) triggers OnSleepModeChanged(false), which has a transition.
				// We should override that or handle it there.
				// We can force hide it here:
				goodnightBox.SetVisible(false)
				goodnightLabel.SetText("")
				fab.RemoveCSSClass("fab-active") // Sync UI state immediately
			}

			// Reset Snooze Count
			snoozeCount = 0

			// Show tabs and Set Alarm Page
			tabs.SetVisible(true)
			stack.SetVisibleChildName("set_alarm")
		}
	})

	// Add ringingPage to our overlay box
	ringingBox.Append(ringingPage)

	// Remove old stack ringing page logic if present (it was lines 210-216)

	// Set up Daemon Callback
	daemon.OnAlarmTriggered = func(h, m int) {
		log.Printf("UI Handling Alarm: %d:%02d", h, m)
		tabs.SetVisible(false) // Hide menu
		// stack.SetVisibleChildName("ringing") // old way

		// Show Ringing Overlay
		ringingBox.SetVisible(true)

		// Ensure window comes to front even if already open
		window.Present()
	}

	daemon.OnSleepModeChanged = func(enabled bool) {
		log.Printf("UI Sleep Mode: %v", enabled)
		if enabled {
			fab.AddCSSClass("fab-active")

			// Start Tracking
			if err := storage.SetSleepStartTime(time.Now()); err != nil {
				log.Printf("Failed to set sleep start time: %v", err)
			}
			snoozeCount = 0 // Reset snooze count on new sleep session

			// Show overlay
			goodnightBox.SetVisible(true)
			goodnightBox.AddCSSClass("visible")
			mainBox.AddCSSClass("hidden")

			// Start Typewriter
			glib.TimeoutAdd(500, func() bool {
				goodnightLabel.SetText("")
				startTypewriter("Goodnight")
				return false
			})

		} else {
			fab.RemoveCSSClass("fab-active")

			// Clear start time to prevent recording if manually disabled
			storage.ClearSleepStartTime()

			// Initial Hide Overlay Animation
			goodnightBox.RemoveCSSClass("visible")
			mainBox.RemoveCSSClass("hidden")

			// Determine if we should clear start time?
			// If disabled manually by user without alarm, we should probably just clear it?
			// Or should we log a "woke up" event?
			// For now, let's assume manual toggle off = cancel or wake up.
			// But the requirements specifics were about "Stops alarm -> add record".
			// If user manually toggles off, maybe we don't save? Or we do?
			// Let's stick to the Alarm Stop requirement for saving for now.

			// If this was triggered by Alarm Stop, goodnightBox might already be hidden by that logic?
			// But let's keep the transition logic as graceful fallback.

			glib.TimeoutAdd(750, func() bool {
				// Only hide if still disabled
				if !fab.HasCSSClass("fab-active") {
					goodnightBox.SetVisible(false)
					goodnightLabel.SetText("")
				}
				return false
			})
		}
	}

	daemon.OnSmartWakeUpToggled = func(enabled bool) {
		// log.Printf("UI Smart Wake Up Toggled: %v", enabled)
		// fab.SetVisible(true) // Always visible now
	}

	// Initial State Check
	if daemon.IsRinging() {
		tabs.SetVisible(false)
		ringingBox.SetVisible(true)
		fab.SetVisible(false) // Hide FAB while ringing
	} else {
		stack.SetVisibleChildName("set_alarm")
	}

	// Hide on Close behavior to keep deamon alive and window ready
	window.ConnectCloseRequest(func() bool {
		log.Println("Window close requested - Hiding window")
		window.SetVisible(false)
		return true // Stop other handlers (prevent destruction)
	})

	return window
}

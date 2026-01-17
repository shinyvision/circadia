package pages

import (
	"circadia/internal/ipc"
	"circadia/storage"
	"circadia/ui"
	"log"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func NewSetAlarmPage(showModal func(*gtk.Widget) func()) *gtk.Box {
	contentBox := gtk.NewBox(gtk.OrientationVertical, 10)
	contentBox.SetMarginTop(20)
	contentBox.SetMarginBottom(20)
	contentBox.SetMarginStart(20)
	contentBox.SetMarginEnd(20)

	initialBedtime, err := storage.GetBedtime()
	if err != nil {
		initialBedtime = "23:00"
		log.Printf("Failed to get bedtime: %v", err)
	}

	initialNotify, err := storage.GetNotifyBedtime()
	if err != nil {
		initialNotify = true
		log.Printf("Failed to get notify: %v", err)
	}

	bedtimeCard := ui.NewBedtimeCard(
		initialBedtime,
		initialNotify,
		func(timeStr string) {
			if err := storage.SetBedtime(timeStr); err != nil {
				log.Printf("Error saving bedtime: %v", err)
			}
			go func() {
				if err := ipc.SendSignal("bedtimeChanged"); err != nil {
					log.Printf("IPC Error: %v", err)
				}
			}()
		},
		func(enabled bool) {
			if err := storage.SetNotifyBedtime(enabled); err != nil {
				log.Printf("Error saving notification toggle: %v", err)
			}
			go func() {
				if err := ipc.SendSignal("bedtimeNotificationsChanged"); err != nil {
					log.Printf("IPC Error: %v", err)
				}
			}()
		},
		showModal,
	)
	contentBox.Append(bedtimeCard)

	alarmContainer := gtk.NewBox(gtk.OrientationVertical, 0)
	contentBox.Append(alarmContainer)

	var refreshAlarms func()
	refreshAlarms = func() {
		for {
			child := alarmContainer.FirstChild()
			if child == nil {
				break
			}
			alarmContainer.Remove(child)
		}

		alarms, err := storage.GetAlarms()
		if err != nil {
			log.Printf("Error loading alarms: %v", err)
			alarms = []storage.Alarm{}
		}

		card := ui.NewWakeUpCard(alarms, func(a storage.Alarm) {
			var closeOverlay func()

			editor := ui.NewAlarmEditor(a, func(updated storage.Alarm) {
				if err := storage.UpdateAlarm(updated.ID, updated.Hour, updated.Minute, updated.Enabled); err != nil {
					log.Printf("Error updating alarm: %v", err)
				}
				if closeOverlay != nil {
					closeOverlay()
				}
				refreshAlarms()
			}, func() {
				if closeOverlay != nil {
					closeOverlay()
				}
			})

			if showModal != nil {
				closeOverlay = showModal(&editor.Widget)
			}
		}, func(a storage.Alarm, enabled bool) {
			if err := storage.ToggleAlarm(a.ID, enabled); err != nil {
				log.Printf("Error toggling alarm: %v", err)
			}
			refreshAlarms()
		}, func(a storage.Alarm) {
			var closeOverlay func()

			vbox := gtk.NewBox(gtk.OrientationVertical, 20)
			vbox.AddCSSClass("modal-content")
			vbox.SetHAlign(gtk.AlignCenter)
			vbox.SetVAlign(gtk.AlignCenter)

			title := gtk.NewLabel("Delete Alarm?")
			title.AddCSSClass("h2")
			vbox.Append(title)

			msg := gtk.NewLabel("Are you sure you want to delete this alarm?")
			vbox.Append(msg)

			actionBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
			actionBox.AddCSSClass("modal-actions")
			actionBox.SetHAlign(gtk.AlignCenter)

			cancelBtn := gtk.NewButtonWithLabel("Cancel")
			cancelBtn.AddCSSClass("modal-btn")
			cancelBtn.ConnectClicked(func() {
				if closeOverlay != nil {
					closeOverlay()
				}
			})
			actionBox.Append(cancelBtn)

			confirmBtn := gtk.NewButtonWithLabel("Delete")
			confirmBtn.AddCSSClass("modal-btn")
			confirmBtn.AddCSSClass("destructive-action")
			confirmBtn.ConnectClicked(func() {
				if err := storage.DeleteAlarm(a.ID); err != nil {
					log.Printf("Error deleting alarm: %v", err)
				}
				if closeOverlay != nil {
					closeOverlay()
				}
				refreshAlarms()
			})
			actionBox.Append(confirmBtn)
			vbox.Append(actionBox)

			if showModal != nil {
				closeOverlay = showModal(&vbox.Widget)
			}
		})
		alarmContainer.Append(card)
	}

	refreshAlarms()

	smartWakeCard := ui.NewSmartWakeUpCard()
	contentBox.Append(smartWakeCard)

	btn := gtk.NewButtonWithLabel("Set Alarm")
	btn.AddCSSClass("action-button")
	btn.ConnectClicked(func() {
		var closeOverlay func()
		nowH, nowM := 8, 0

		onSave := func(h, m int) {
			if err := storage.AddAlarm(h, m); err != nil {
				log.Printf("Error adding alarm: %v", err)
			}
			if closeOverlay != nil {
				closeOverlay()
			}
			refreshAlarms()
		}

		widget := ui.NewTimePickerWidget("Set Alarm", nowH, nowM, onSave, func() {
			if closeOverlay != nil {
				closeOverlay()
			}
		})

		if showModal != nil {
			closeOverlay = showModal(&widget.Widget)
		}
	})
	contentBox.Append(btn)

	return contentBox
}

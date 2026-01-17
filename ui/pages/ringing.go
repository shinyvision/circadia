package pages

import (
	"fmt"
	"log"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"circadia/daemon"
	"circadia/storage"
)

func NewRingingPage(onAction func(string)) *gtk.Box {
	vbox := gtk.NewBox(gtk.OrientationVertical, 20)
	vbox.SetHAlign(gtk.AlignCenter)
	vbox.SetVAlign(gtk.AlignCenter)
	vbox.SetHExpand(true)
	vbox.SetVExpand(true)
	vbox.AddCSSClass("black-background")

	icon := gtk.NewImageFromIconName("alarm-symbolic")
	icon.SetPixelSize(64)
	vbox.Append(icon)

	label := gtk.NewLabel("Alarm")
	label.AddCSSClass("h1")
	vbox.Append(label)

	msg := gtk.NewLabel("Wake Up!")
	msg.AddCSSClass("h2")
	vbox.Append(msg)

	snoozeEnabled, _ := storage.GetSnoozeEnabled()
	if snoozeEnabled {
		dur, _ := storage.GetSnoozeDuration()
		snoozeBtn := gtk.NewButtonWithLabel(fmt.Sprintf("Snooze (%d min)", dur))
		snoozeBtn.AddCSSClass("pill-button")
		snoozeBtn.ConnectClicked(func() {
			log.Println("Snooze clicked")
			daemon.SnoozeAlarm()
			if onAction != nil {
				onAction("snooze")
			}
		})
		vbox.Append(snoozeBtn)
	}

	stopBtn := gtk.NewButtonWithLabel("Stop")
	stopBtn.AddCSSClass("pill-button")
	stopBtn.AddCSSClass("destructive-action")
	stopBtn.ConnectClicked(func() {
		log.Println("Stop clicked")
		daemon.StopAlarm()
		if onAction != nil {
			onAction("stop")
		}
	})
	vbox.Append(stopBtn)

	return vbox
}

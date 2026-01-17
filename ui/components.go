package ui

import (
	"circadia/internal/ipc"
	"circadia/storage"
	"fmt"
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func CreateCardBox() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.AddCSSClass("card-box")
	return box
}

func NewTabSelector() (*gtk.Box, *gtk.Button, *gtk.Button, *gtk.Button) {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("nav-box")

	btn1 := gtk.NewButtonWithLabel("Set Alarm")
	btn1.AddCSSClass("nav-button")
	btn1.AddCSSClass("active")
	btn1.SetHExpand(true)

	btn2 := gtk.NewButtonWithLabel("Sleep History")
	btn2.AddCSSClass("nav-button")
	btn2.SetHExpand(true)

	btn3 := gtk.NewButtonWithLabel("Settings")
	btn3.AddCSSClass("nav-button")
	btn3.SetHExpand(true)

	box.Append(btn1)
	box.Append(btn2)
	box.Append(btn3)

	return box, btn1, btn2, btn3
}

func NewBedtimeCard(initialTime string, initialNotify bool, onTimeChange func(string), onToggle func(bool), showModal func(*gtk.Widget) func()) *gtk.Box {
	card := CreateCardBox()

	title := gtk.NewLabel("Bedtime")
	title.SetCSSClasses([]string{"h2"})
	title.SetHAlign(gtk.AlignStart)
	card.Append(title)

	row := gtk.NewBox(gtk.OrientationHorizontal, 0)

	timeBtn := gtk.NewButtonWithLabel(initialTime)
	timeBtn.SetCSSClasses([]string{"time-display-btn"})
	timeBtn.SetHExpand(true)
	timeBtn.SetHAlign(gtk.AlignStart)

	var initH, initM int
	fmt.Sscanf(initialTime, "%d:%d", &initH, &initM)

	currentH, currentM := initH, initM

	timeBtn.ConnectClicked(func() {
		var closeOverlay func()

		widget := NewTimePickerWidget("Set Bedtime", currentH, currentM, func(h, m int) {
			currentH, currentM = h, m
			newTime := fmt.Sprintf("%02d:%02d", h, m)
			timeBtn.SetLabel(newTime)
			if onTimeChange != nil {
				onTimeChange(newTime)
			}
			if closeOverlay != nil {
				closeOverlay()
			}
		}, func() {
			if closeOverlay != nil {
				closeOverlay()
			}
		})

		if showModal != nil {
			closeOverlay = showModal(&widget.Widget)
		}
	})

	toggle := gtk.NewSwitch()
	toggle.SetActive(initialNotify)
	toggle.SetVAlign(gtk.AlignCenter)

	toggle.ConnectStateSet(func(state bool) bool {
		if onToggle != nil {
			onToggle(state)
		}
		return false
	})

	row.Append(timeBtn)
	row.Append(toggle)
	card.Append(row)

	sub := gtk.NewLabel("Tonight")
	sub.SetCSSClasses([]string{"caption"})
	sub.SetHAlign(gtk.AlignStart)
	card.Append(sub)

	return card
}

func NewTimePickerWidget(titleText string, hour, min int, onSave func(h, m int), onCancel func()) *gtk.Box {
	vbox := gtk.NewBox(gtk.OrientationVertical, 20)
	vbox.AddCSSClass("modal-content")
	vbox.SetHAlign(gtk.AlignCenter)
	vbox.SetVAlign(gtk.AlignCenter)

	title := gtk.NewLabel(titleText)
	title.AddCSSClass("h2")
	vbox.Append(title)

	h := hour
	m := min

	pickerRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
	pickerRow.SetHAlign(gtk.AlignCenter)

	var hLabel, mLabel *gtk.Label
	updateLabels := func() {
		hLabel.SetText(fmt.Sprintf("%02d", h))
		mLabel.SetText(fmt.Sprintf("%02d", m))
	}

	hBox := gtk.NewBox(gtk.OrientationVertical, 10)
	hUp := gtk.NewButtonWithLabel("+")
	hUp.AddCSSClass("picker-btn")
	hLabel = gtk.NewLabel(fmt.Sprintf("%02d", h))
	hLabel.AddCSSClass("picker-val")
	hDown := gtk.NewButtonWithLabel("-")
	hDown.AddCSSClass("picker-btn")

	connectRepeatingButton(hUp, func() {
		h = (h + 1) % 24
		updateLabels()
	})
	connectRepeatingButton(hDown, func() {
		h = (h - 1 + 24) % 24
		updateLabels()
	})

	hBox.Append(hUp)
	hBox.Append(hLabel)
	hBox.Append(hDown)

	sep := gtk.NewLabel(":")
	sep.AddCSSClass("picker-val")
	sep.SetVAlign(gtk.AlignCenter)

	mBox := gtk.NewBox(gtk.OrientationVertical, 10)
	mUp := gtk.NewButtonWithLabel("+")
	mUp.AddCSSClass("picker-btn")
	mLabel = gtk.NewLabel(fmt.Sprintf("%02d", m))
	mLabel.AddCSSClass("picker-val")
	mDown := gtk.NewButtonWithLabel("-")
	mDown.AddCSSClass("picker-btn")

	connectRepeatingButton(mUp, func() {
		m = (m + 1) % 60
		updateLabels()
	})
	connectRepeatingButton(mDown, func() {
		m = (m - 1 + 60) % 60
		updateLabels()
	})

	mBox.Append(mUp)
	mBox.Append(mLabel)
	mBox.Append(mDown)

	pickerRow.Append(hBox)
	pickerRow.Append(sep)
	pickerRow.Append(mBox)

	actionBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	actionBox.AddCSSClass("modal-actions")
	actionBox.SetHAlign(gtk.AlignCenter)

	cancel := gtk.NewButtonWithLabel("Cancel")
	cancel.AddCSSClass("modal-btn")
	cancel.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	save := gtk.NewButtonWithLabel("Save")
	save.AddCSSClass("modal-btn")
	save.AddCSSClass("suggested-action")
	save.ConnectClicked(func() {
		if onSave != nil {
			onSave(h, m)
		}
	})

	actionBox.Append(cancel)
	actionBox.Append(save)

	vbox.Append(pickerRow)
	vbox.Append(actionBox)

	return vbox
}

func connectRepeatingButton(btn *gtk.Button, action func()) {
	click := gtk.NewGestureClick()
	click.SetPropagationPhase(gtk.PhaseCapture)
	click.ConnectPressed(func(n int, x, y float64) {
		action()
	})
	btn.AddController(click)

	longPress := gtk.NewGestureLongPress()
	longPress.SetTouchOnly(false)
	longPress.SetPropagationPhase(gtk.PhaseCapture)
	btn.AddController(longPress)

	var sourceId glib.SourceHandle

	stopTimer := func() {
		if sourceId > 0 {
			glib.SourceRemove(sourceId)
			sourceId = 0
		}
	}

	longPress.ConnectPressed(func(x, y float64) {
		stopTimer()
		sourceId = glib.TimeoutAdd(100, func() bool {
			action()
			return true
		})
	})

	longPress.ConnectEnd(func(seq *gdk.EventSequence) {
		stopTimer()
	})

	longPress.ConnectCancelled(func() {
		stopTimer()
	})
}

func NewAlarmEditor(alarm storage.Alarm, onSave func(a storage.Alarm), onCancel func()) *gtk.Box {
	vbox := gtk.NewBox(gtk.OrientationVertical, 20)
	vbox.AddCSSClass("modal-content")
	vbox.SetHAlign(gtk.AlignCenter)
	vbox.SetVAlign(gtk.AlignCenter)

	title := gtk.NewLabel("Edit Alarm")
	title.AddCSSClass("h2")
	vbox.Append(title)

	h, m := alarm.Hour, alarm.Minute
	enabled := alarm.Enabled

	pickerRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
	pickerRow.SetHAlign(gtk.AlignCenter)

	var hLabel, mLabel *gtk.Label
	updateLabels := func() {
		hLabel.SetText(fmt.Sprintf("%02d", h))
		mLabel.SetText(fmt.Sprintf("%02d", m))
	}

	hBox := gtk.NewBox(gtk.OrientationVertical, 10)
	hUp := gtk.NewButtonWithLabel("+")
	hUp.AddCSSClass("picker-btn")
	hLabel = gtk.NewLabel(fmt.Sprintf("%02d", h))
	hLabel.AddCSSClass("picker-val")
	hDown := gtk.NewButtonWithLabel("-")
	hDown.AddCSSClass("picker-btn")
	connectRepeatingButton(hUp, func() { h = (h + 1) % 24; updateLabels() })
	connectRepeatingButton(hDown, func() { h = (h - 1 + 24) % 24; updateLabels() })
	hBox.Append(hUp)
	hBox.Append(hLabel)
	hBox.Append(hDown)

	sep := gtk.NewLabel(":")
	sep.AddCSSClass("picker-val")
	sep.SetVAlign(gtk.AlignCenter)

	mBox := gtk.NewBox(gtk.OrientationVertical, 10)
	mUp := gtk.NewButtonWithLabel("+")
	mUp.AddCSSClass("picker-btn")
	mLabel = gtk.NewLabel(fmt.Sprintf("%02d", m))
	mLabel.AddCSSClass("picker-val")
	mDown := gtk.NewButtonWithLabel("-")
	mDown.AddCSSClass("picker-btn")
	connectRepeatingButton(mUp, func() { m = (m + 1) % 60; updateLabels() })
	connectRepeatingButton(mDown, func() { m = (m - 1 + 60) % 60; updateLabels() })
	mBox.Append(mUp)
	mBox.Append(mLabel)
	mBox.Append(mDown)

	pickerRow.Append(hBox)
	pickerRow.Append(sep)
	pickerRow.Append(mBox)
	vbox.Append(pickerRow)

	toggleRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
	toggleRow.SetHAlign(gtk.AlignCenter)
	toggleLabel := gtk.NewLabel("Enabled")
	toggleSwitch := gtk.NewSwitch()
	toggleSwitch.SetActive(enabled)
	toggleSwitch.ConnectStateSet(func(state bool) bool {
		enabled = state
		return false
	})
	toggleRow.Append(toggleLabel)
	toggleRow.Append(toggleSwitch)
	vbox.Append(toggleRow)

	actionBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	actionBox.AddCSSClass("modal-actions")
	actionBox.SetHAlign(gtk.AlignCenter)

	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("modal-btn")
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})
	actionBox.Append(cancelBtn)

	saveBtn := gtk.NewButtonWithLabel("Save")
	saveBtn.AddCSSClass("modal-btn")
	saveBtn.AddCSSClass("suggested-action")
	saveBtn.ConnectClicked(func() {
		if onSave != nil {
			alarm.Hour = h
			alarm.Minute = m
			alarm.Enabled = enabled
			onSave(alarm)
		}
	})
	actionBox.Append(saveBtn)

	vbox.Append(actionBox)
	return vbox
}

func createAlarmRow(alarm storage.Alarm, onEdit func(), onToggle func(bool), onDelete func()) *gtk.Box {
	row := gtk.NewBox(gtk.OrientationHorizontal, 10)
	row.AddCSSClass("alarm-row")

	switchBtn := gtk.NewSwitch()
	switchBtn.SetActive(alarm.Enabled)
	switchBtn.AddCSSClass("alarm-switch")
	switchBtn.SetVAlign(gtk.AlignCenter)
	switchBtn.ConnectStateSet(func(state bool) bool {
		if onToggle != nil {
			onToggle(state)
		}
		return false
	})
	row.Append(switchBtn)

	timeStr := fmt.Sprintf("%02d:%02d", alarm.Hour, alarm.Minute)

	timeBtnRaw := gtk.NewButton()
	timeBtnRaw.AddCSSClass("flat")
	timeBtnRaw.SetHExpand(true)

	timeLabel := gtk.NewLabel(timeStr)
	timeLabel.AddCSSClass("h2")
	timeLabel.SetHAlign(gtk.AlignStart)

	timeBtnRaw.SetChild(timeLabel)
	timeBtnRaw.ConnectClicked(func() {
		if onEdit != nil {
			onEdit()
		}
	})
	row.Append(timeBtnRaw)

	trashBtn := gtk.NewButtonFromIconName("user-trash-symbolic")
	trashBtn.AddCSSClass("flat")
	trashBtn.SetVAlign(gtk.AlignCenter)
	trashBtn.ConnectClicked(func() {
		if onDelete != nil {
			onDelete()
		}
	})
	row.Append(trashBtn)

	return row
}

func NewWakeUpCard(alarms []storage.Alarm, onEdit func(alarm storage.Alarm), onToggle func(alarm storage.Alarm, enabled bool), onDelete func(alarm storage.Alarm)) *gtk.Box {
	card := CreateCardBox()

	header := gtk.NewBox(gtk.OrientationHorizontal, 0)
	title := gtk.NewLabel("Wake Up")
	title.AddCSSClass("h2")
	title.SetHAlign(gtk.AlignStart)
	title.SetMarginBottom(10)
	title.SetHExpand(true)
	header.Append(title)

	card.Append(header)

	list := gtk.NewBox(gtk.OrientationVertical, 10)
	list.AddCSSClass("alarm-list")

	for _, a := range alarms {
		currentAlarm := a
		row := createAlarmRow(a, func() {
			if onEdit != nil {
				onEdit(currentAlarm)
			}
		}, func(enabled bool) {
			if onToggle != nil {
				onToggle(currentAlarm, enabled)
			}
		}, func() {
			if onDelete != nil {
				onDelete(currentAlarm)
			}
		})
		list.Append(row)
	}

	card.Append(list)

	return card
}

func NewSmartWakeUpCard() *gtk.Box {
	card := CreateCardBox()

	header := gtk.NewBox(gtk.OrientationHorizontal, 0)

	title := gtk.NewLabel("Smart Wake Up")
	title.AddCSSClass("h2")
	title.SetHExpand(true)
	title.SetHAlign(gtk.AlignStart)

	enabled, err := storage.GetSmartWakeUp()
	if err != nil {
		log.Println("Error loading smart wake up setting:", err)
	}

	toggle := gtk.NewSwitch()
	toggle.SetActive(enabled)
	toggle.SetVAlign(gtk.AlignCenter)

	toggle.ConnectStateSet(func(state bool) bool {
		if err := storage.SetSmartWakeUp(state); err != nil {
			log.Println("Error saving smart wake up:", err)
		}
		go func() {
			if err := ipc.SendSignal(fmt.Sprintf("smartWakeUpToggled:%v", state)); err != nil {
				log.Println("IPC Error:", err)
			}
		}()
		return false
	})

	header.Append(title)
	header.Append(toggle)
	card.Append(header)

	desc := gtk.NewLabel("Press on the moon icon when you're ready to sleep. We'll wake you within 30 minutes of your set time.")
	desc.AddCSSClass("caption")
	desc.SetHAlign(gtk.AlignStart)
	desc.SetWrap(true)
	desc.SetMaxWidthChars(40)
	card.Append(desc)

	return card
}

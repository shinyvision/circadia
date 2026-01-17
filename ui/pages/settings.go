package pages

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"circadia/daemon"
	"circadia/storage"
	"circadia/ui"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func NewSettingsPage() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	audioCard := ui.CreateCardBox()
	box.Append(audioCard)

	audioHeader := gtk.NewLabel("Alarm Sound")
	audioHeader.AddCSSClass("h2")
	audioHeader.SetHAlign(gtk.AlignStart)
	audioHeader.SetMarginBottom(10)
	audioCard.Append(audioHeader)

	fileRow := gtk.NewBox(gtk.OrientationHorizontal, 15)
	fileRow.SetHExpand(true)

	btnPreview := gtk.NewButton()
	iconPlay := gtk.NewImageFromIconName("media-playback-start-symbolic")
	iconPlay.SetPixelSize(24)
	btnPreview.SetChild(iconPlay)
	btnPreview.AddCSSClass("flat")
	btnPreview.SetSizeRequest(40, 40)

	fileRow.Append(btnPreview)

	currentPath, _ := storage.GetAlarmAudioPath()
	displayPath := "Default"
	if currentPath != "" {
		displayPath = filepath.Base(currentPath)
	}
	fileLabel := gtk.NewLabel(displayPath)
	fileLabel.AddCSSClass("body-text")
	fileLabel.SetHAlign(gtk.AlignStart)
	fileLabel.SetHExpand(true)
	fileRow.Append(fileLabel)

	btnReset := gtk.NewButtonFromIconName("edit-undo-symbolic")
	btnReset.SetTooltipText("Reset to Default")
	btnReset.AddCSSClass("flat")
	btnReset.SetVAlign(gtk.AlignCenter)

	btnReset.SetSensitive(currentPath != "")

	fileRow.Append(btnReset)
	audioCard.Append(fileRow)

	btnChoose := gtk.NewButtonWithLabel("Choose File")
	btnChoose.AddCSSClass("pill-button")
	btnChoose.AddCSSClass("suggested-action")
	btnChoose.SetHExpand(true)
	audioCard.Append(btnChoose)

	btnChoose.ConnectClicked(func() {
		dialog := gtk.NewFileDialog()
		dialog.SetTitle("Select alarm audio")
		dialog.SetAcceptLabel("_Open")
		dialog.SetModal(true)

		filter := gtk.NewFileFilter()
		filter.SetName("Audio Files")
		filter.AddMIMEType("audio/mpeg")
		filter.AddMIMEType("audio/ogg")
		filter.AddMIMEType("audio/wav")
		filter.AddMIMEType("application/ogg")
		filter.AddPattern("*.mp3")
		filter.AddPattern("*.ogg")
		filter.AddPattern("*.wav")

		filters := gio.NewListStore(gtk.GTypeFileFilter)
		filters.Append(filter.Object)
		dialog.SetFilters(filters)
		dialog.SetDefaultFilter(filter)

		var parent *gtk.Window
		if root := btnChoose.Root(); root != nil {
			if w, ok := root.Cast().(*gtk.Window); ok {
				parent = w
			}
		}

		dialog.Open(context.TODO(), parent, func(res gio.AsyncResulter) {
			file, err := dialog.OpenFinish(res)
			if err != nil {
				log.Printf("File dialog cancelled or error: %v", err)
				return
			}

			path := file.Path()
			if path == "" {
				return
			}

			if err := storage.SetAlarmAudioPath(path); err != nil {
				log.Printf("Failed to save audio path: %v", err)
				return
			}

			fileLabel.SetText(filepath.Base(path))
			btnReset.SetSensitive(true)
		})
	})

	btnReset.ConnectClicked(func() {
		storage.SetAlarmAudioPath("")
		fileLabel.SetText("Default")
		btnReset.SetSensitive(false)
	})

	isPreviewing := false
	btnPreview.ConnectClicked(func() {
		btnPreview.SetSensitive(false)

		if isPreviewing {
			go func() {
				daemon.StopAlarmSound()
				glib.IdleAdd(func() bool {
					iconPlay.SetFromIconName("media-playback-start-symbolic")
					isPreviewing = false
					btnPreview.SetSensitive(true)
					return false
				})
			}()
		} else {
			path, _ := storage.GetAlarmAudioPath()

			go func() {
				err := daemon.PreviewAudio(path)
				glib.IdleAdd(func() bool {
					if err != nil {
						fmt.Printf("Preview error: %v\n", err)
						iconPlay.SetFromIconName("media-playback-start-symbolic")
						isPreviewing = false
					} else {
						iconPlay.SetFromIconName("media-playback-stop-symbolic")
						isPreviewing = true
					}
					btnPreview.SetSensitive(true)
					return false
				})
			}()
		}
	})

	snoozeCard := ui.CreateCardBox()
	box.Append(snoozeCard)

	snoozeHeader := gtk.NewLabel("Snooze Settings")
	snoozeHeader.AddCSSClass("h2")
	snoozeHeader.SetHAlign(gtk.AlignStart)
	snoozeHeader.SetMarginBottom(10)
	snoozeCard.Append(snoozeHeader)

	toggleRow := gtk.NewBox(gtk.OrientationHorizontal, 10)

	lblEnable := gtk.NewLabel("Enable Snooze")
	lblEnable.AddCSSClass("body-text")
	lblEnable.SetHExpand(true)
	lblEnable.SetHAlign(gtk.AlignStart)

	snoozeEnabled, _ := storage.GetSnoozeEnabled()
	checkEnable := gtk.NewSwitch()
	checkEnable.SetActive(snoozeEnabled)
	checkEnable.SetVAlign(gtk.AlignCenter)

	toggleRow.Append(lblEnable)
	toggleRow.Append(checkEnable)
	snoozeCard.Append(toggleRow)

	snoozeDur, _ := storage.GetSnoozeDuration()
	sliderBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	sliderBox.SetMarginTop(10)

	lblDur := gtk.NewLabel(fmt.Sprintf("%d min", snoozeDur))
	lblDur.AddCSSClass("h3")
	lblDur.SetWidthChars(6)

	scale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 5, 30, 1)
	scale.SetValue(float64(snoozeDur))
	scale.SetHExpand(true)
	scale.SetDrawValue(false)
	scale.SetSizeRequest(-1, 40)

	scale.SetSensitive(snoozeEnabled)

	sliderBox.Append(scale)
	sliderBox.Append(lblDur)
	snoozeCard.Append(sliderBox)

	checkEnable.ConnectStateSet(func(state bool) bool {
		storage.SetSnoozeEnabled(state)
		scale.SetSensitive(state)
		return false
	})

	scale.ConnectValueChanged(func() {
		val := int(scale.Value())
		lblDur.SetText(fmt.Sprintf("%d min", val))
		storage.SetSnoozeDuration(val)
	})

	return box
}

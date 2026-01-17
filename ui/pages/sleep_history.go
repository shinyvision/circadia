package pages

import (
	"fmt"
	"math"
	"time"

	"circadia/storage"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type SleepHistoryController struct {
	Box *gtk.Box
}

func NewSleepHistoryPage() *SleepHistoryController {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)

	controller := &SleepHistoryController{Box: box}
	controller.Refresh()

	return controller
}

func (c *SleepHistoryController) Refresh() {
	// Clear all existing children
	// Clear all existing children
	for {
		child := c.Box.FirstChild()
		if child == nil {
			break
		}
		c.Box.Remove(child)
	}

	history, err := storage.GetHistory(30)
	if err != nil {
		errLabel := gtk.NewLabel("Failed to load history")
		c.Box.Append(errLabel)
		return
	}

	graphArea := gtk.NewDrawingArea()
	graphArea.SetContentHeight(200)
	graphArea.SetDrawFunc(func(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		drawHistoryGraph(cr, width, height, history)
	})
	c.Box.Append(graphArea)

	avgSnooze := calculateAverageSnooze(history)
	avgDuration := calculateAverageSleepDuration(history)
	avgCard := createAverageCard(avgDuration, avgSnooze)
	c.Box.Append(avgCard)

	lastSession, err := storage.GetLastSleepSession()
	if err == nil && lastSession != nil {
		card := createLastNightCard(lastSession)
		c.Box.Append(card)
	} else {
		noData := gtk.NewLabel("No sleep data recorded yet.")
		noData.SetHAlign(gtk.AlignStart)
		c.Box.Append(noData)
	}
}

func drawHistoryGraph(cr *cairo.Context, width, height int, history []storage.SleepSession) {
	cr.SetSourceRGB(0.1, 0.1, 0.12)
	cr.Paint()

	if len(history) == 0 {
		return
	}

	pad := 20.0
	w := float64(width) - 2*pad
	h := float64(height) - 2*pad

	var maxDuration float64
	for _, s := range history {
		dur := s.EndTime.Sub(s.StartTime).Hours()
		if dur > maxDuration {
			maxDuration = dur
		}
	}
	if maxDuration == 0 {
		maxDuration = 8
	} else {
		maxDuration = math.Ceil(maxDuration)
	}

	cr.SetSourceRGB(0.5, 0.5, 0.5)
	cr.SetLineWidth(1)

	cr.MoveTo(pad, pad)
	cr.LineTo(pad, float64(height)-pad)
	cr.Stroke()

	cr.MoveTo(pad, float64(height)-pad)
	cr.LineTo(float64(width)-pad, float64(height)-pad)
	cr.Stroke()

	cr.SetSourceRGB(0.4, 0.7, 1.0)
	cr.SetLineWidth(2)

	stepX := w / float64(len(history)-1)
	if len(history) == 1 {
		stepX = w / 2
	}

	first := true
	for i, s := range history {
		dur := s.EndTime.Sub(s.StartTime).Hours()

		x := pad + float64(i)*stepX

		y := (float64(height) - pad) - (dur / maxDuration * h)

		if first {
			cr.MoveTo(x, y)
			first = false
		} else {
			cr.LineTo(x, y)
		}

		cr.Save()
		cr.Arc(x, y, 3, 0, 2*math.Pi)
		cr.Fill()
		cr.Restore()
		cr.MoveTo(x, y)
	}
	cr.Stroke()
}

func calculateAverageSnooze(history []storage.SleepSession) float64 {
	if len(history) == 0 {
		return 0
	}
	total := 0
	for _, s := range history {
		total += s.SnoozeCount
	}
	return float64(total) / float64(len(history))
}

func calculateAverageSleepDuration(history []storage.SleepSession) time.Duration {
	if len(history) == 0 {
		return 0
	}
	var total time.Duration
	for _, s := range history {
		total += s.EndTime.Sub(s.StartTime)
	}
	return total / time.Duration(len(history))
}

func createAverageCard(avgDuration time.Duration, avgSnooze float64) *gtk.Box {
	card := gtk.NewBox(gtk.OrientationVertical, 10)
	card.AddCSSClass("card-box")

	title := gtk.NewLabel("Your Monthly Average")
	title.AddCSSClass("h2")
	title.SetHAlign(gtk.AlignStart)
	title.SetMarginBottom(10)
	card.Append(title)

	grid := gtk.NewGrid()
	grid.SetColumnSpacing(40)
	card.Append(grid)

	vbox1 := gtk.NewBox(gtk.OrientationVertical, 5)
	vbox1.SetHExpand(true)
	vbox1.SetHAlign(gtk.AlignFill)
	lblTitle1 := gtk.NewLabel("Sleep time")
	lblTitle1.SetOpacity(0.7)
	lblTitle1.SetHAlign(gtk.AlignStart)

	hours := int(avgDuration.Hours())
	minutes := int(avgDuration.Minutes()) % 60
	lblValue1 := gtk.NewLabel(fmt.Sprintf("%d:%02d hours", hours, minutes))
	lblValue1.SetHAlign(gtk.AlignStart)

	vbox1.Append(lblTitle1)
	vbox1.Append(lblValue1)
	grid.Attach(vbox1, 0, 0, 1, 1)

	vbox2 := gtk.NewBox(gtk.OrientationVertical, 5)
	vbox2.SetHExpand(true)
	vbox2.SetHAlign(gtk.AlignFill)
	lblTitle2 := gtk.NewLabel("Snoozes")
	lblTitle2.SetOpacity(0.7)
	lblTitle2.SetHAlign(gtk.AlignStart)

	lblValue2 := gtk.NewLabel(fmt.Sprintf("%.2f", avgSnooze))
	if avgSnooze == float64(int(avgSnooze)) {
		lblValue2.SetText(fmt.Sprintf("%d", int(avgSnooze)))
	}
	if avgSnooze == math.Trunc(avgSnooze) {
		lblValue2.SetText(fmt.Sprintf("%.0f", avgSnooze))
	}

	lblValue2.SetHAlign(gtk.AlignStart)

	vbox2.Append(lblTitle2)
	vbox2.Append(lblValue2)
	grid.Attach(vbox2, 1, 0, 1, 1)

	return card
}

func createLastNightCard(s *storage.SleepSession) *gtk.Box {
	if s == nil {
		return nil
	}

	card := gtk.NewBox(gtk.OrientationVertical, 10)
	card.AddCSSClass("card-box")

	header := gtk.NewLabel("Last Night")
	header.AddCSSClass("h2")
	header.SetHAlign(gtk.AlignStart)
	header.SetMarginBottom(10)
	card.Append(header)

	dur := s.EndTime.Sub(s.StartTime)
	hours := int(dur.Hours())
	minutes := int(dur.Minutes()) % 60
	durLabel := gtk.NewLabel(fmt.Sprintf("Slept for %02d:%02d hours", hours, minutes))
	durLabel.SetHAlign(gtk.AlignStart)
	card.Append(durLabel)

	var snoozeText string
	if s.SnoozeCount == 0 {
		snoozeText = "No snoozes"
	} else if s.SnoozeCount == 1 {
		snoozeText = "Snoozed once"
	} else {
		snoozeText = fmt.Sprintf("Snoozed %d times", s.SnoozeCount)
	}

	snoozeLabel := gtk.NewLabel(snoozeText)
	snoozeLabel.SetHAlign(gtk.AlignStart)
	card.Append(snoozeLabel)

	return card
}

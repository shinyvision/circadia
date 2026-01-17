package pages

import (
	"circadia/storage"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestSleepHistoryRefresh_ReplacesContent(t *testing.T) {
	// Initialize in-memory DB
	storage.InitDB(":memory:")

	// Create a dummy session so we have data
	start := time.Now().Add(-8 * time.Hour)
	end := time.Now()
	storage.AddSleepSession(start, end, 0)

	// Initialize GTK (required for widget creation)
	gtk.Init()

	// Create Controller
	controller := NewSleepHistoryPage()

	// Check initial child count
	initialCount := countChildren(controller.Box)
	if initialCount == 0 {
		t.Error("Expected children after initialization, got 0")
	}

	// Call Refresh explicitly
	controller.Refresh()

	// Check child count again
	// It should be roughly same (replacing), NOT double
	refreshedCount := countChildren(controller.Box)

	if refreshedCount != initialCount {
		t.Errorf("Expected child count to remain consistent after refresh (replacing content). Initial: %d, Refreshed: %d", initialCount, refreshedCount)
	}

	// If it was appending, it would be close to 2x initialCount
	if refreshedCount >= initialCount*2 {
		t.Error("Refresh seems to be appending instead of replacing!")
	}
}

func countChildren(box *gtk.Box) int {
	count := 0
	child := box.FirstChild()
	for child != nil {
		count++
		if w, ok := child.(*gtk.Widget); ok {
			child = w.NextSibling()
		} else {
			break
		}
	}
	return count
}

package daemon

import (
	"circadia/storage"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	// Use in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open valid in-memory db: %v", err)
	}

	storage.DB = db

	// Initialize tables
	queryHistory := `
	CREATE TABLE IF NOT EXISTS sleep_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		snooze_count INTEGER
	);
	`
	if _, err := db.Exec(queryHistory); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
}

func TestFinalizeSleepSession_ShortDuration(t *testing.T) {
	setupTestDB(t)

	startTime := time.Now()
	endTime := startTime.Add(30 * time.Minute) // 30 mins < 1 hour
	snoozeCount := 0

	err := FinalizeSleepSession(startTime, endTime, snoozeCount, false)
	if err != nil {
		t.Fatalf("FinalizeSleepSession failed: %v", err)
	}

	// Verify NO session was saved
	sessions, err := storage.GetHistory(1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

func TestFinalizeSleepSession_LongDuration(t *testing.T) {
	setupTestDB(t)

	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour) // 2 hours > 1 hour
	snoozeCount := 2

	err := FinalizeSleepSession(startTime, endTime, snoozeCount, false)
	if err != nil {
		t.Fatalf("FinalizeSleepSession failed: %v", err)
	}

	// Verify session WAS saved
	sessions, err := storage.GetHistory(1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	} else {
		s := sessions[0]
		// In-memory DB time precision might vary, checking rough equality or just presence is mostly enough here
		// But let's check basic fields
		if s.SnoozeCount != snoozeCount {
			t.Errorf("Expected snooze count %d, got %d", snoozeCount, s.SnoozeCount)
		}
	}
}

func TestFinalizeSleepSession_ForceSave(t *testing.T) {
	setupTestDB(t)

	startTime := time.Now()
	endTime := startTime.Add(10 * time.Minute) // Short duration
	snoozeCount := 1

	// Force bypass = true
	err := FinalizeSleepSession(startTime, endTime, snoozeCount, true)
	if err != nil {
		t.Fatalf("FinalizeSleepSession failed: %v", err)
	}

	// Verify session WAS saved despite short duration
	sessions, err := storage.GetHistory(1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

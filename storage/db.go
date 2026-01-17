package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() error {
	dbPath, err := xdg.DataFile("circadia/user.db")
	if err != nil {
		return fmt.Errorf("could not resolve data file path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	log.Printf("Opening database at %s", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	DB = db

	return migrate()
}

func migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`
	_, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("could not create settings table: %w", err)
	}

	queryAlarms := `
	CREATE TABLE IF NOT EXISTS alarms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hour INTEGER,
		minute INTEGER,
		enabled BOOLEAN
	);
	`
	_, err = DB.Exec(queryAlarms)
	if err != nil {
		return fmt.Errorf("could not create alarms table: %w", err)
	}

	queryHistory := `
	CREATE TABLE IF NOT EXISTS sleep_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		snooze_count INTEGER
	);
	`
	_, err = DB.Exec(queryHistory)
	if err != nil {
		return fmt.Errorf("could not create sleep_history table: %w", err)
	}

	if err := SetDefault("bedtime", "23:00"); err != nil {
		return err
	}
	if err := SetDefault("notify_bedtime", "true"); err != nil {
		return err
	}

	return nil
}

func SetDefault(key, value string) error {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", key).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", key, value)
		return err
	}
	return nil
}

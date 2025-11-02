package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDatabase initializes SQLite database and creates tables
func InitDatabase(dbPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=1")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("Database initialized at: %s", dbPath)
	return nil
}

// createTables creates all necessary tables
func createTables() error {
	createButtonsTable := `
	CREATE TABLE IF NOT EXISTS hardware_buttons (
		id TEXT PRIMARY KEY,
		mac_address TEXT UNIQUE NOT NULL,
		button_id TEXT NOT NULL DEFAULT '1',
		name TEXT NOT NULL DEFAULT '',
		room_code TEXT NOT NULL DEFAULT '',
		team_id TEXT NOT NULL DEFAULT '',
		team_name TEXT NOT NULL DEFAULT '',
		is_active INTEGER NOT NULL DEFAULT 1,
		press_count INTEGER NOT NULL DEFAULT 0,
		last_press DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := DB.Exec(createButtonsTable); err != nil {
		return fmt.Errorf("failed to create hardware_buttons table: %w", err)
	}

	// Create index on MAC address for faster lookups
	createIndex := `CREATE INDEX IF NOT EXISTS idx_mac_address ON hardware_buttons(mac_address);`
	if _, err := DB.Exec(createIndex); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Create index on room_code for filtering
	createRoomIndex := `CREATE INDEX IF NOT EXISTS idx_room_code ON hardware_buttons(room_code);`
	if _, err := DB.Exec(createRoomIndex); err != nil {
		return fmt.Errorf("failed to create room_code index: %w", err)
	}

	log.Println("Database tables created successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}


// ABOUTME: Database connection management and initialization
// ABOUTME: Handles SQLite connection, XDG paths, and migrations

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	// Use 0700 (owner only) for privacy - RSS reading habits are personal data
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func GetDefaultDBPath() string {
	return filepath.Join(getDataDir(), "digest", "digest.db")
}

func GetDefaultOPMLPath() string {
	return filepath.Join(getDataDir(), "digest", "feeds.opml")
}

func getDataDir() string {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return dataDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".local", "share")
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		folder TEXT DEFAULT '',
		etag TEXT,
		last_modified TEXT,
		last_fetched_at DATETIME,
		last_error TEXT,
		error_count INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS entries (
		id TEXT PRIMARY KEY,
		feed_id TEXT NOT NULL,
		guid TEXT NOT NULL,
		title TEXT,
		link TEXT,
		author TEXT,
		published_at DATETIME,
		content TEXT,
		read BOOLEAN DEFAULT FALSE,
		read_at DATETIME,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
		UNIQUE(feed_id, guid)
	);

	CREATE INDEX IF NOT EXISTS idx_entries_feed_id ON entries(feed_id);
	CREATE INDEX IF NOT EXISTS idx_entries_read ON entries(read);
	CREATE INDEX IF NOT EXISTS idx_entries_published_at ON entries(published_at);

	CREATE TABLE IF NOT EXISTS entry_tags (
		entry_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		PRIMARY KEY (entry_id, tag),
		FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE
	);
	`
	_, err := db.Exec(schema)
	return err
}

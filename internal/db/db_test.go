// ABOUTME: Tests for database connection and path helpers
// ABOUTME: Validates XDG path resolution and connection lifecycle

package db

import (
	"path/filepath"
	"testing"
)

func TestGetDefaultDBPath(t *testing.T) {
	path := GetDefaultDBPath()

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	if filepath.Base(path) != "digest.db" {
		t.Errorf("expected digest.db, got %s", filepath.Base(path))
	}
}

func TestGetDefaultOPMLPath(t *testing.T) {
	path := GetDefaultOPMLPath()

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	if filepath.Base(path) != "feeds.opml" {
		t.Errorf("expected feeds.opml, got %s", filepath.Base(path))
	}
}

func TestInitDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer conn.Close()

	// Verify tables exist
	var count int
	err = conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='feeds'").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Error("feeds table not created")
	}
}

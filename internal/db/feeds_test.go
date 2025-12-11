// ABOUTME: Tests for feed database operations
// ABOUTME: Validates CRUD operations for feeds table

package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/models"
)

func TestCreateFeed(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	feed.Title = strPtr("Test Feed")

	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Verify it exists
	got, err := GetFeedByURL(conn, feed.URL)
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %s, got %s", feed.ID, got.ID)
	}
}

func TestGetFeedByPrefix(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	// Use first 8 chars of UUID
	prefix := feed.ID[:8]
	got, err := GetFeedByPrefix(conn, prefix)
	if err != nil {
		t.Fatalf("GetFeedByPrefix failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %s, got %s", feed.ID, got.ID)
	}
}

func TestListFeeds(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed2 := models.NewFeed("https://example.com/feed2.xml")
	_ = CreateFeed(conn, feed1)
	_ = CreateFeed(conn, feed2)

	feeds, err := ListFeeds(conn)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}
}

func TestDeleteFeed(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	err := DeleteFeed(conn, feed.ID)
	if err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	_, err = GetFeedByURL(conn, feed.URL)
	if err == nil {
		t.Error("expected feed to be deleted")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	return conn
}

func strPtr(s string) *string {
	return &s
}

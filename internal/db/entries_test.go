// ABOUTME: Tests for entry database operations
// ABOUTME: Validates CRUD operations for entries table

package db

import (
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

func TestCreateEntry(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "entry-guid-123", "Test Entry")
	err = CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Verify it exists by ID
	got, err := GetEntryByID(conn, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID failed: %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("expected ID %s, got %s", entry.ID, got.ID)
	}
	if got.GUID != entry.GUID {
		t.Errorf("expected GUID %s, got %s", entry.GUID, got.GUID)
	}
	if got.Read != false {
		t.Errorf("expected Read to be false, got %v", got.Read)
	}
}

func TestCreateEntry_Duplicate(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry1 := models.NewEntry(feed.ID, "entry-guid-123", "Test Entry 1")
	err = CreateEntry(conn, entry1)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Try to create another entry with same feed_id and guid
	entry2 := models.NewEntry(feed.ID, "entry-guid-123", "Test Entry 2")
	err = CreateEntry(conn, entry2)
	if err == nil {
		t.Error("expected CreateEntry to fail with duplicate feed_id+guid, but it succeeded")
	}
}

func TestListEntries_Unread(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create entries
	entry1 := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	entry2 := models.NewEntry(feed.ID, "guid-2", "Entry 2")
	entry3 := models.NewEntry(feed.ID, "guid-3", "Entry 3")

	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)
	_ = CreateEntry(conn, entry3)

	// Mark entry2 as read
	err = MarkEntryRead(conn, entry2.ID)
	if err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	// List unread entries only
	unreadOnly := true
	entries, err := ListEntries(conn, &feed.ID, &unreadOnly, nil, nil)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 unread entries, got %d", len(entries))
	}

	// Verify the unread entries are entry1 and entry3
	for _, e := range entries {
		if e.ID == entry2.ID {
			t.Error("expected entry2 to be filtered out as it's marked read")
		}
	}
}

func TestMarkEntryRead(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	err = CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Mark as read
	err = MarkEntryRead(conn, entry.ID)
	if err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	// Verify it's marked as read
	got, err := GetEntryByID(conn, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID failed: %v", err)
	}
	if got.Read != true {
		t.Errorf("expected Read to be true, got %v", got.Read)
	}
	if got.ReadAt == nil {
		t.Error("expected ReadAt to be set, got nil")
	}
}

func TestEntryExists(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	err = CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Check if entry exists
	exists, err := EntryExists(conn, feed.ID, "guid-1")
	if err != nil {
		t.Fatalf("EntryExists failed: %v", err)
	}
	if !exists {
		t.Error("expected entry to exist, but EntryExists returned false")
	}

	// Check if non-existent entry exists
	exists, err = EntryExists(conn, feed.ID, "non-existent-guid")
	if err != nil {
		t.Fatalf("EntryExists failed: %v", err)
	}
	if exists {
		t.Error("expected entry not to exist, but EntryExists returned true")
	}
}

func TestCountUnreadEntries(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create two feeds
	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed2 := models.NewFeed("https://example.com/feed2.xml")
	_ = CreateFeed(conn, feed1)
	_ = CreateFeed(conn, feed2)

	// Create entries for feed1
	entry1 := models.NewEntry(feed1.ID, "guid-1", "Entry 1")
	entry2 := models.NewEntry(feed1.ID, "guid-2", "Entry 2")
	entry3 := models.NewEntry(feed1.ID, "guid-3", "Entry 3")
	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)
	_ = CreateEntry(conn, entry3)

	// Create entries for feed2
	entry4 := models.NewEntry(feed2.ID, "guid-4", "Entry 4")
	entry5 := models.NewEntry(feed2.ID, "guid-5", "Entry 5")
	_ = CreateEntry(conn, entry4)
	_ = CreateEntry(conn, entry5)

	// Mark one entry from feed1 as read
	_ = MarkEntryRead(conn, entry2.ID)

	// Count unread entries for feed1
	count, err := CountUnreadEntries(conn, &feed1.ID)
	if err != nil {
		t.Fatalf("CountUnreadEntries failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread entries for feed1, got %d", count)
	}

	// Count all unread entries
	count, err = CountUnreadEntries(conn, nil)
	if err != nil {
		t.Fatalf("CountUnreadEntries failed: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 total unread entries, got %d", count)
	}
}

func TestGetEntryByPrefix(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	err = CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Use first 8 chars of UUID
	prefix := entry.ID[:8]
	got, err := GetEntryByPrefix(conn, prefix)
	if err != nil {
		t.Fatalf("GetEntryByPrefix failed: %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("expected ID %s, got %s", entry.ID, got.ID)
	}
}

func TestMarkEntryUnread(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	err = CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Mark as read
	err = MarkEntryRead(conn, entry.ID)
	if err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	// Mark as unread
	err = MarkEntryUnread(conn, entry.ID)
	if err != nil {
		t.Fatalf("MarkEntryUnread failed: %v", err)
	}

	// Verify it's marked as unread
	got, err := GetEntryByID(conn, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID failed: %v", err)
	}
	if got.Read != false {
		t.Errorf("expected Read to be false, got %v", got.Read)
	}
	if got.ReadAt != nil {
		t.Errorf("expected ReadAt to be nil, got %v", got.ReadAt)
	}
}

func TestListEntries_WithFilters(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed
	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	// Create entries with different published times
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	entry1 := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	entry1.PublishedAt = &past
	entry2 := models.NewEntry(feed.ID, "guid-2", "Entry 2")
	entry2.PublishedAt = &now
	entry3 := models.NewEntry(feed.ID, "guid-3", "Entry 3")
	entry3.PublishedAt = &now

	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)
	_ = CreateEntry(conn, entry3)

	// Test with since filter
	sinceTime := now.Add(-1 * time.Hour)
	entries, err := ListEntries(conn, nil, nil, &sinceTime, nil)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries since %v, got %d", sinceTime, len(entries))
	}

	// Test with limit
	limit := 2
	entries, err = ListEntries(conn, nil, nil, nil, &limit)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(entries))
	}
}

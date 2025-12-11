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
	entries, err := ListEntries(conn, &feed.ID, nil, &unreadOnly, nil, nil, nil)
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
	entries, err := ListEntries(conn, nil, nil, nil, &sinceTime, nil, nil)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries since %v, got %d", sinceTime, len(entries))
	}

	// Test with limit
	limit := 2
	entries, err = ListEntries(conn, nil, nil, nil, nil, nil, &limit)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(entries))
	}

	// Test with until filter (should only return entry1 which is in the past)
	untilTime := now.Add(-12 * time.Hour)
	entries, err = ListEntries(conn, nil, nil, nil, nil, &untilTime, nil)
	if err != nil {
		t.Fatalf("ListEntries with until failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry before %v, got %d", untilTime, len(entries))
	}

	// Test with since AND until (yesterday's window)
	sinceYesterday := now.Add(-25 * time.Hour)
	untilYesterday := now.Add(-23 * time.Hour)
	entries, err = ListEntries(conn, nil, nil, nil, &sinceYesterday, &untilYesterday, nil)
	if err != nil {
		t.Fatalf("ListEntries with since and until failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry in yesterday window, got %d", len(entries))
	}
}

func TestListEntries_ByFeedIDs(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create two feeds
	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed2 := models.NewFeed("https://example.com/feed2.xml")
	feed3 := models.NewFeed("https://example.com/feed3.xml")
	_ = CreateFeed(conn, feed1)
	_ = CreateFeed(conn, feed2)
	_ = CreateFeed(conn, feed3)

	// Create entries for each feed
	entry1 := models.NewEntry(feed1.ID, "guid-1", "Entry from Feed 1")
	entry2 := models.NewEntry(feed2.ID, "guid-2", "Entry from Feed 2")
	entry3 := models.NewEntry(feed3.ID, "guid-3", "Entry from Feed 3")
	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)
	_ = CreateEntry(conn, entry3)

	// Test filtering by multiple feed IDs
	feedIDs := []string{feed1.ID, feed2.ID}
	entries, err := ListEntries(conn, nil, feedIDs, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ListEntries with feedIDs failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries from feed1 and feed2, got %d", len(entries))
	}

	// Verify only entries from feed1 and feed2 are returned
	for _, entry := range entries {
		if entry.FeedID != feed1.ID && entry.FeedID != feed2.ID {
			t.Errorf("unexpected entry from feed %s", entry.FeedID)
		}
	}

	// Test with single feed in array (should work same as feedID)
	feedIDs = []string{feed3.ID}
	entries, err = ListEntries(conn, nil, feedIDs, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ListEntries with single feedID in array failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry from feed3, got %d", len(entries))
	}

	// Test feedIDs takes precedence over feedID
	entries, err = ListEntries(conn, &feed1.ID, []string{feed2.ID, feed3.ID}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("ListEntries with both feedID and feedIDs failed: %v", err)
	}

	// Should return entries from feed2 and feed3, not feed1
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (feedIDs should override feedID), got %d", len(entries))
	}
	for _, entry := range entries {
		if entry.FeedID == feed1.ID {
			t.Error("feedIDs should take precedence over feedID")
		}
	}
}

func TestMarkEntriesReadBefore(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	// Create a feed
	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	// Create entries with different published times
	now := time.Now()
	twoDaysAgo := now.Add(-48 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	entry1 := models.NewEntry(feed.ID, "guid-old", "Old Entry")
	entry1.PublishedAt = &twoDaysAgo
	entry2 := models.NewEntry(feed.ID, "guid-yesterday", "Yesterday Entry")
	entry2.PublishedAt = &yesterday
	entry3 := models.NewEntry(feed.ID, "guid-today", "Today Entry")
	entry3.PublishedAt = &now

	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)
	_ = CreateEntry(conn, entry3)

	// Mark entries older than yesterday as read
	cutoff := now.Add(-24 * time.Hour)
	count, err := MarkEntriesReadBefore(conn, cutoff)
	if err != nil {
		t.Fatalf("MarkEntriesReadBefore failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 entry marked as read, got %d", count)
	}

	// Verify the old entry is marked as read
	oldEntry, _ := GetEntryByID(conn, entry1.ID)
	if !oldEntry.Read {
		t.Error("expected old entry to be marked as read")
	}

	// Verify the yesterday entry is still unread
	yesterdayEntry, _ := GetEntryByID(conn, entry2.ID)
	if yesterdayEntry.Read {
		t.Error("expected yesterday entry to still be unread")
	}

	// Verify today's entry is still unread
	todayEntry, _ := GetEntryByID(conn, entry3.ID)
	if todayEntry.Read {
		t.Error("expected today entry to still be unread")
	}

	// Mark remaining entries as read before now
	count, err = MarkEntriesReadBefore(conn, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("MarkEntriesReadBefore second call failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 entries marked as read, got %d", count)
	}

	// Verify all entries are now read
	unreadCount, _ := CountUnreadEntries(conn, nil)
	if unreadCount != 0 {
		t.Errorf("expected 0 unread entries, got %d", unreadCount)
	}
}

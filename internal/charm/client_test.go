// ABOUTME: Tests for the Charm KV client wrapper
// ABOUTME: Uses real local KV storage with sync disabled for fast, isolated tests

//go:build !race

package charm

import (
	"os"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

// newTestClient creates a fresh client for testing with auto-sync disabled.
// Each call creates a new database with unique name to isolate tests.
func newTestClient(t *testing.T) *Client {
	t.Helper()

	// Create a unique database name for this test
	dbName := "digest-test-" + t.Name()

	// Create temp dir for the test database
	tmpDir, err := os.MkdirTemp("", "charm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	// Set data dir for charm to use temp dir
	os.Setenv("CHARM_DATA_DIR", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("CHARM_DATA_DIR")
	})

	return NewTestClientWithDBName(dbName, false)
}

func TestNewFeed(t *testing.T) {
	url := "https://example.com/feed.xml"
	feed := NewFeed(url)

	if feed.ID == "" {
		t.Error("expected feed ID to be generated")
	}
	if feed.URL != url {
		t.Errorf("expected URL %q, got %q", url, feed.URL)
	}
	if feed.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewEntry(t *testing.T) {
	feedID := "feed-123"
	guid := "guid-456"
	title := "Test Entry"

	entry := NewEntry(feedID, guid, title)

	if entry.ID == "" {
		t.Error("expected entry ID to be generated")
	}
	if entry.FeedID != feedID {
		t.Errorf("expected FeedID %q, got %q", feedID, entry.FeedID)
	}
	if entry.GUID != guid {
		t.Errorf("expected GUID %q, got %q", guid, entry.GUID)
	}
	if entry.Title == nil || *entry.Title != title {
		t.Errorf("expected Title %q, got %v", title, entry.Title)
	}
	if entry.Read {
		t.Error("expected entry to be unread by default")
	}
	if entry.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestFeedCRUD(t *testing.T) {
	c := newTestClient(t)

	// Create
	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Read
	got, err := c.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if got.URL != feed.URL {
		t.Errorf("expected URL %q, got %q", feed.URL, got.URL)
	}

	// Update
	title := "Updated Title"
	feed.Title = &title
	if err := c.UpdateFeed(feed); err != nil {
		t.Fatalf("UpdateFeed: %v", err)
	}

	got, err = c.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed after update: %v", err)
	}
	if got.Title == nil || *got.Title != title {
		t.Errorf("expected Title %q, got %v", title, got.Title)
	}

	// List
	feeds, err := c.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(feeds) != 1 {
		t.Errorf("expected 1 feed, got %d", len(feeds))
	}

	// Delete
	if err := c.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed: %v", err)
	}

	_, err = c.GetFeed(feed.ID)
	if err == nil {
		t.Error("expected error getting deleted feed")
	}
}

func TestGetFeedByURL(t *testing.T) {
	c := newTestClient(t)

	url := "https://example.com/feed.xml"
	feed := NewFeed(url)
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	got, err := c.GetFeedByURL(url)
	if err != nil {
		t.Fatalf("GetFeedByURL: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %q, got %q", feed.ID, got.ID)
	}

	// Not found
	_, err = c.GetFeedByURL("https://notfound.com/feed.xml")
	if err == nil {
		t.Error("expected error for non-existent URL")
	}
}

func TestGetFeedByPrefix(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Get by 8-char prefix
	prefix := feed.ID[:8]
	got, err := c.GetFeedByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetFeedByPrefix: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %q, got %q", feed.ID, got.ID)
	}

	// Prefix too short
	_, err = c.GetFeedByPrefix("abc")
	if err == nil {
		t.Error("expected error for short prefix")
	}

	// No match
	_, err = c.GetFeedByPrefix("zzzzzzzz")
	if err == nil {
		t.Error("expected error for non-matching prefix")
	}
}

func TestEntryCRUD(t *testing.T) {
	c := newTestClient(t)

	// Create feed first
	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Create entry
	entry := NewEntry(feed.ID, "guid-123", "Test Entry")
	if err := c.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Read
	got, err := c.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if *got.Title != *entry.Title {
		t.Errorf("expected Title %q, got %q", *entry.Title, *got.Title)
	}

	// Update
	content := "Updated content"
	entry.Content = &content
	if err := c.UpdateEntry(entry); err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}

	got, err = c.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry after update: %v", err)
	}
	if got.Content == nil || *got.Content != content {
		t.Errorf("expected Content %q, got %v", content, got.Content)
	}

	// Delete
	if err := c.DeleteEntry(entry.ID); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}

	_, err = c.GetEntry(entry.ID)
	if err == nil {
		t.Error("expected error getting deleted entry")
	}
}

func TestListEntriesWithFilter(t *testing.T) {
	c := newTestClient(t)

	// Create two feeds
	feed1 := NewFeed("https://example.com/feed1.xml")
	feed2 := NewFeed("https://example.com/feed2.xml")
	if err := c.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}
	if err := c.CreateFeed(feed2); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Create entries with different timestamps
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Feed 1: two entries (one read, one unread)
	entry1 := NewEntry(feed1.ID, "guid-1", "Entry 1")
	entry1.PublishedAt = &now
	entry1.Read = true
	readAt := now
	entry1.ReadAt = &readAt

	entry2 := NewEntry(feed1.ID, "guid-2", "Entry 2")
	entry2.PublishedAt = &yesterday

	// Feed 2: one entry from last week
	entry3 := NewEntry(feed2.ID, "guid-3", "Entry 3")
	entry3.PublishedAt = &lastWeek

	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Test: all entries
	entries, err := c.ListEntries(nil)
	if err != nil {
		t.Fatalf("ListEntries (all): %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Test: filter by feed
	entries, err = c.ListEntries(&EntryFilter{FeedID: &feed1.ID})
	if err != nil {
		t.Fatalf("ListEntries (by feed): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries for feed1, got %d", len(entries))
	}

	// Test: unread only
	unreadOnly := true
	entries, err = c.ListEntries(&EntryFilter{UnreadOnly: &unreadOnly})
	if err != nil {
		t.Fatalf("ListEntries (unread): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 unread entries, got %d", len(entries))
	}

	// Test: since yesterday
	since := yesterday.Add(-time.Hour)
	entries, err = c.ListEntries(&EntryFilter{Since: &since})
	if err != nil {
		t.Fatalf("ListEntries (since): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries since yesterday, got %d", len(entries))
	}

	// Test: limit and offset
	limit := 2
	entries, err = c.ListEntries(&EntryFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("ListEntries (limit): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(entries))
	}

	offset := 1
	entries, err = c.ListEntries(&EntryFilter{Offset: &offset})
	if err != nil {
		t.Fatalf("ListEntries (offset): %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries with offset, got %d", len(entries))
	}

	// Test: FeedIDs (multiple feeds)
	entries, err = c.ListEntries(&EntryFilter{FeedIDs: []string{feed1.ID, feed2.ID}})
	if err != nil {
		t.Fatalf("ListEntries (feedIDs): %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries with feedIDs, got %d", len(entries))
	}
}

func TestMarkEntryReadUnread(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := NewEntry(feed.ID, "guid-1", "Test Entry")
	if err := c.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Initially unread
	got, _ := c.GetEntry(entry.ID)
	if got.Read {
		t.Error("expected entry to be unread initially")
	}

	// Mark read
	if err := c.MarkEntryRead(entry.ID); err != nil {
		t.Fatalf("MarkEntryRead: %v", err)
	}

	got, _ = c.GetEntry(entry.ID)
	if !got.Read {
		t.Error("expected entry to be read")
	}
	if got.ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}

	// Mark unread
	if err := c.MarkEntryUnread(entry.ID); err != nil {
		t.Fatalf("MarkEntryUnread: %v", err)
	}

	got, _ = c.GetEntry(entry.ID)
	if got.Read {
		t.Error("expected entry to be unread")
	}
	if got.ReadAt != nil {
		t.Error("expected ReadAt to be nil")
	}
}

func TestCascadeDelete(t *testing.T) {
	c := newTestClient(t)

	// Create feed with entries
	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry1 := NewEntry(feed.ID, "guid-1", "Entry 1")
	entry2 := NewEntry(feed.ID, "guid-2", "Entry 2")
	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Verify entries exist
	entries, _ := c.ListEntries(&EntryFilter{FeedID: &feed.ID})
	if len(entries) != 2 {
		t.Errorf("expected 2 entries before delete, got %d", len(entries))
	}

	// Delete feed (should cascade to entries)
	if err := c.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed: %v", err)
	}

	// Verify entries are gone
	entries, _ = c.ListEntries(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after cascade delete, got %d", len(entries))
	}
}

func TestEntryExists(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	guid := "unique-guid-123"
	entry := NewEntry(feed.ID, guid, "Test Entry")
	if err := c.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Entry exists
	exists, err := c.EntryExists(feed.ID, guid)
	if err != nil {
		t.Fatalf("EntryExists: %v", err)
	}
	if !exists {
		t.Error("expected entry to exist")
	}

	// Different guid doesn't exist
	exists, err = c.EntryExists(feed.ID, "different-guid")
	if err != nil {
		t.Fatalf("EntryExists: %v", err)
	}
	if exists {
		t.Error("expected entry not to exist")
	}

	// Different feed doesn't have this guid
	exists, err = c.EntryExists("other-feed-id", guid)
	if err != nil {
		t.Fatalf("EntryExists: %v", err)
	}
	if exists {
		t.Error("expected entry not to exist for other feed")
	}
}

func TestGetFeedStats(t *testing.T) {
	c := newTestClient(t)

	// Create feed with entries
	feed := NewFeed("https://example.com/feed.xml")
	title := "Test Feed"
	feed.Title = &title
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry1 := NewEntry(feed.ID, "guid-1", "Entry 1")
	entry2 := NewEntry(feed.ID, "guid-2", "Entry 2")
	entry2.Read = true
	readAt := time.Now()
	entry2.ReadAt = &readAt

	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	stats, err := c.GetFeedStats()
	if err != nil {
		t.Fatalf("GetFeedStats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 feed stat, got %d", len(stats))
	}

	stat := stats[0]
	if stat.FeedID != feed.ID {
		t.Errorf("expected FeedID %q, got %q", feed.ID, stat.FeedID)
	}
	if stat.EntryCount != 2 {
		t.Errorf("expected EntryCount 2, got %d", stat.EntryCount)
	}
	if stat.UnreadCount != 1 {
		t.Errorf("expected UnreadCount 1, got %d", stat.UnreadCount)
	}
}

func TestGetOverallStats(t *testing.T) {
	c := newTestClient(t)

	// Create two feeds with entries
	feed1 := NewFeed("https://example.com/feed1.xml")
	feed2 := NewFeed("https://example.com/feed2.xml")
	if err := c.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}
	if err := c.CreateFeed(feed2); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry1 := NewEntry(feed1.ID, "guid-1", "Entry 1")
	entry2 := NewEntry(feed1.ID, "guid-2", "Entry 2")
	entry2.Read = true
	entry3 := NewEntry(feed2.ID, "guid-3", "Entry 3")

	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	stats, err := c.GetOverallStats()
	if err != nil {
		t.Fatalf("GetOverallStats: %v", err)
	}

	if stats.TotalFeeds != 2 {
		t.Errorf("expected TotalFeeds 2, got %d", stats.TotalFeeds)
	}
	if stats.TotalEntries != 3 {
		t.Errorf("expected TotalEntries 3, got %d", stats.TotalEntries)
	}
	if stats.UnreadCount != 2 {
		t.Errorf("expected UnreadCount 2, got %d", stats.UnreadCount)
	}
}

func TestMarkEntriesReadBefore(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Create entries with different timestamps
	entry1 := NewEntry(feed.ID, "guid-1", "Today")
	entry1.PublishedAt = &now

	entry2 := NewEntry(feed.ID, "guid-2", "Yesterday")
	entry2.PublishedAt = &yesterday

	entry3 := NewEntry(feed.ID, "guid-3", "Two days ago")
	entry3.PublishedAt = &twoDaysAgo

	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Mark entries before yesterday as read
	cutoff := now.Add(-12 * time.Hour) // Before today but after yesterday
	count, err := c.MarkEntriesReadBefore(cutoff)
	if err != nil {
		t.Fatalf("MarkEntriesReadBefore: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 entries marked read, got %d", count)
	}

	// Verify
	unreadOnly := true
	entries, _ := c.ListEntries(&EntryFilter{UnreadOnly: &unreadOnly})
	if len(entries) != 1 {
		t.Errorf("expected 1 unread entry, got %d", len(entries))
	}
}

func TestUpdateFeedFetchState(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	etag := "abc123"
	lastMod := "Wed, 18 Dec 2024 00:00:00 GMT"
	fetchedAt := time.Now()

	if err := c.UpdateFeedFetchState(feed.ID, &etag, &lastMod, fetchedAt); err != nil {
		t.Fatalf("UpdateFeedFetchState: %v", err)
	}

	got, _ := c.GetFeed(feed.ID)
	if got.ETag == nil || *got.ETag != etag {
		t.Errorf("expected ETag %q, got %v", etag, got.ETag)
	}
	if got.LastModified == nil || *got.LastModified != lastMod {
		t.Errorf("expected LastModified %q, got %v", lastMod, got.LastModified)
	}
	if got.LastFetchedAt == nil {
		t.Error("expected LastFetchedAt to be set")
	}
}

func TestUpdateFeedError(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Record error
	errMsg := "connection timeout"
	if err := c.UpdateFeedError(feed.ID, errMsg); err != nil {
		t.Fatalf("UpdateFeedError: %v", err)
	}

	got, _ := c.GetFeed(feed.ID)
	if got.LastError == nil || *got.LastError != errMsg {
		t.Errorf("expected LastError %q, got %v", errMsg, got.LastError)
	}
	if got.ErrorCount != 1 {
		t.Errorf("expected ErrorCount 1, got %d", got.ErrorCount)
	}

	// Record another error
	if err := c.UpdateFeedError(feed.ID, "another error"); err != nil {
		t.Fatalf("UpdateFeedError: %v", err)
	}

	got, _ = c.GetFeed(feed.ID)
	if got.ErrorCount != 2 {
		t.Errorf("expected ErrorCount 2, got %d", got.ErrorCount)
	}
}

func TestCountUnreadEntries(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry1 := NewEntry(feed.ID, "guid-1", "Entry 1")
	entry2 := NewEntry(feed.ID, "guid-2", "Entry 2")
	entry2.Read = true

	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Count all unread
	count, err := c.CountUnreadEntries(nil)
	if err != nil {
		t.Fatalf("CountUnreadEntries: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 unread entry, got %d", count)
	}

	// Count unread for specific feed
	count, err = c.CountUnreadEntries(&feed.ID)
	if err != nil {
		t.Fatalf("CountUnreadEntries: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 unread entry for feed, got %d", count)
	}
}

func TestGetEntryByPrefix(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := NewEntry(feed.ID, "guid-1", "Test Entry")
	if err := c.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Get by 8-char prefix
	prefix := entry.ID[:8]
	got, err := c.GetEntryByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetEntryByPrefix: %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("expected ID %q, got %q", entry.ID, got.ID)
	}

	// Prefix too short
	_, err = c.GetEntryByPrefix("abc")
	if err == nil {
		t.Error("expected error for short prefix")
	}

	// No match
	_, err = c.GetEntryByPrefix("zzzzzzzz")
	if err == nil {
		t.Error("expected error for non-matching prefix")
	}
}

func TestListEntriesSortOrder(t *testing.T) {
	c := newTestClient(t)

	feed := NewFeed("https://example.com/feed.xml")
	if err := c.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Create entries with different timestamps
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	entry1 := NewEntry(feed.ID, "guid-1", "Oldest")
	entry1.PublishedAt = &twoDaysAgo

	entry2 := NewEntry(feed.ID, "guid-2", "Middle")
	entry2.PublishedAt = &yesterday

	entry3 := NewEntry(feed.ID, "guid-3", "Newest")
	entry3.PublishedAt = &now

	// Insert in non-chronological order
	if err := c.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := c.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// List should be sorted newest first
	entries, err := c.ListEntries(nil)
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if *entries[0].Title != "Newest" {
		t.Errorf("expected first entry to be 'Newest', got %q", *entries[0].Title)
	}
	if *entries[1].Title != "Middle" {
		t.Errorf("expected second entry to be 'Middle', got %q", *entries[1].Title)
	}
	if *entries[2].Title != "Oldest" {
		t.Errorf("expected third entry to be 'Oldest', got %q", *entries[2].Title)
	}
}

func TestListFeedsSortOrder(t *testing.T) {
	c := newTestClient(t)

	// Create feeds with different creation times
	// Note: We need to manipulate CreatedAt since NewFeed uses time.Now()
	feed1 := NewFeed("https://example.com/feed1.xml")
	feed1.CreatedAt = time.Now().Add(-48 * time.Hour)

	feed2 := NewFeed("https://example.com/feed2.xml")
	feed2.CreatedAt = time.Now().Add(-24 * time.Hour)

	feed3 := NewFeed("https://example.com/feed3.xml")
	feed3.CreatedAt = time.Now()

	// Insert in non-chronological order
	if err := c.CreateFeed(feed2); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}
	if err := c.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}
	if err := c.CreateFeed(feed3); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// List should be sorted newest first
	feeds, err := c.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}

	if len(feeds) != 3 {
		t.Fatalf("expected 3 feeds, got %d", len(feeds))
	}

	if feeds[0].URL != "https://example.com/feed3.xml" {
		t.Errorf("expected first feed to be feed3, got %q", feeds[0].URL)
	}
	if feeds[1].URL != "https://example.com/feed2.xml" {
		t.Errorf("expected second feed to be feed2, got %q", feeds[1].URL)
	}
	if feeds[2].URL != "https://example.com/feed1.xml" {
		t.Errorf("expected third feed to be feed1, got %q", feeds[2].URL)
	}
}

func TestModelsMarking(t *testing.T) {
	// Test models.Entry marking methods
	entry := &models.Entry{
		ID:     "test-id",
		FeedID: "feed-id",
		GUID:   "guid-123",
		Read:   false,
	}

	// Mark read
	entry.MarkRead()
	if !entry.Read {
		t.Error("expected entry to be read after MarkRead")
	}
	if entry.ReadAt == nil {
		t.Error("expected ReadAt to be set after MarkRead")
	}

	// Mark unread
	entry.MarkUnread()
	if entry.Read {
		t.Error("expected entry to be unread after MarkUnread")
	}
	if entry.ReadAt != nil {
		t.Error("expected ReadAt to be nil after MarkUnread")
	}
}

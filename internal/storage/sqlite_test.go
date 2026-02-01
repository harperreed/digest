// ABOUTME: Tests for SQLite storage implementation
// ABOUTME: Covers all CRUD operations for feeds and entries with FTS5 search

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

func TestNewSQLiteStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestFeedCRUD(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create a feed
	feed := models.NewFeed("https://example.com/feed.xml")
	title := "Example Feed"
	feed.Title = &title
	feed.Folder = "Tech"

	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Get by ID
	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.URL != feed.URL {
		t.Errorf("URL mismatch: got %q, want %q", got.URL, feed.URL)
	}
	if got.Title == nil || *got.Title != title {
		t.Errorf("Title mismatch: got %v, want %q", got.Title, title)
	}
	if got.Folder != "Tech" {
		t.Errorf("Folder mismatch: got %q, want %q", got.Folder, "Tech")
	}

	// Get by URL
	got, err = store.GetFeedByURL("https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, feed.ID)
	}

	// Get by prefix
	prefix := feed.ID[:8]
	got, err = store.GetFeedByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetFeedByPrefix failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, feed.ID)
	}

	// Update feed
	newTitle := "Updated Feed"
	feed.Title = &newTitle
	if err := store.UpdateFeed(feed); err != nil {
		t.Fatalf("UpdateFeed failed: %v", err)
	}

	got, err = store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed after update failed: %v", err)
	}
	if got.Title == nil || *got.Title != newTitle {
		t.Errorf("Title not updated: got %v, want %q", got.Title, newTitle)
	}

	// List feeds
	feeds, err := store.ListFeeds()
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 1 {
		t.Errorf("ListFeeds count: got %d, want 1", len(feeds))
	}

	// Delete feed
	if err := store.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	_, err = store.GetFeed(feed.ID)
	if err == nil {
		t.Error("expected error getting deleted feed")
	}
}

func TestEntryCRUD(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create a feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create an entry
	entry := models.NewEntry(feed.ID, "guid-123", "Test Article")
	link := "https://example.com/post/1"
	author := "John Doe"
	content := "This is the article content about technology."
	pubTime := time.Now().Add(-time.Hour)
	entry.Link = &link
	entry.Author = &author
	entry.Content = &content
	entry.PublishedAt = &pubTime

	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Get by ID
	got, err := store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if got.GUID != "guid-123" {
		t.Errorf("GUID mismatch: got %q, want %q", got.GUID, "guid-123")
	}
	if got.Title == nil || *got.Title != "Test Article" {
		t.Errorf("Title mismatch: got %v, want %q", got.Title, "Test Article")
	}
	if got.Read {
		t.Error("new entry should not be read")
	}

	// Get by prefix
	prefix := entry.ID[:8]
	got, err = store.GetEntryByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetEntryByPrefix failed: %v", err)
	}
	if got.ID != entry.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, entry.ID)
	}

	// Entry exists
	exists, err := store.EntryExists(feed.ID, "guid-123")
	if err != nil {
		t.Fatalf("EntryExists failed: %v", err)
	}
	if !exists {
		t.Error("expected entry to exist")
	}

	exists, err = store.EntryExists(feed.ID, "nonexistent-guid")
	if err != nil {
		t.Fatalf("EntryExists for nonexistent failed: %v", err)
	}
	if exists {
		t.Error("expected entry to not exist")
	}

	// Mark as read
	if err := store.MarkEntryRead(entry.ID); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	got, err = store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry after mark read failed: %v", err)
	}
	if !got.Read {
		t.Error("entry should be read")
	}
	if got.ReadAt == nil {
		t.Error("ReadAt should be set")
	}

	// Mark as unread
	if err := store.MarkEntryUnread(entry.ID); err != nil {
		t.Fatalf("MarkEntryUnread failed: %v", err)
	}

	got, err = store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry after mark unread failed: %v", err)
	}
	if got.Read {
		t.Error("entry should be unread")
	}
	if got.ReadAt != nil {
		t.Error("ReadAt should be nil")
	}

	// Delete entry
	if err := store.DeleteEntry(entry.ID); err != nil {
		t.Fatalf("DeleteEntry failed: %v", err)
	}

	_, err = store.GetEntry(entry.ID)
	if err == nil {
		t.Error("expected error getting deleted entry")
	}
}

func TestListEntriesFilter(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create multiple entries
	now := time.Now()
	entries := []struct {
		guid  string
		title string
		pub   time.Time
		read  bool
	}{
		{"guid-1", "Article 1", now.Add(-1 * time.Hour), false},
		{"guid-2", "Article 2", now.Add(-2 * time.Hour), true},
		{"guid-3", "Article 3", now.Add(-3 * time.Hour), false},
		{"guid-4", "Article 4", now.Add(-25 * time.Hour), false},
	}

	for _, e := range entries {
		entry := models.NewEntry(feed.ID, e.guid, e.title)
		entry.PublishedAt = &e.pub
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
		if e.read {
			if err := store.MarkEntryRead(entry.ID); err != nil {
				t.Fatalf("MarkEntryRead failed: %v", err)
			}
		}
	}

	// Test unread only filter
	unreadOnly := true
	filter := &EntryFilter{UnreadOnly: &unreadOnly}
	result, err := store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries unread failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 unread entries, got %d", len(result))
	}

	// Test since filter
	since := now.Add(-4 * time.Hour)
	filter = &EntryFilter{Since: &since}
	result, err = store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries since failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 recent entries, got %d", len(result))
	}

	// Test limit
	limit := 2
	filter = &EntryFilter{Limit: &limit}
	result, err = store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries limit failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries with limit, got %d", len(result))
	}

	// Test offset
	offset := 1
	filter = &EntryFilter{Limit: &limit, Offset: &offset}
	result, err = store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries offset failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries with offset, got %d", len(result))
	}

	// Test feed filter
	filter = &EntryFilter{FeedID: &feed.ID}
	result, err = store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries feed filter failed: %v", err)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 entries for feed, got %d", len(result))
	}
}

func TestMarkEntriesReadBefore(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create entries
	now := time.Now()
	for i := 1; i <= 5; i++ {
		entry := models.NewEntry(feed.ID, "guid-"+string(rune('0'+i)), "Article")
		pub := now.Add(time.Duration(-i) * 24 * time.Hour)
		entry.PublishedAt = &pub
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	// Mark entries older than 3 days as read
	cutoff := now.Add(-2 * 24 * time.Hour)
	count, err := store.MarkEntriesReadBefore(cutoff)
	if err != nil {
		t.Fatalf("MarkEntriesReadBefore failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 entries marked, got %d", count)
	}

	// Verify unread count
	unreadCount, err := store.CountUnreadEntries(nil)
	if err != nil {
		t.Fatalf("CountUnreadEntries failed: %v", err)
	}
	if unreadCount != 2 {
		t.Errorf("expected 2 unread entries, got %d", unreadCount)
	}
}

func TestFeedFetchState(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Update fetch state
	etag := "abc123"
	lastMod := "Wed, 01 Jan 2025 00:00:00 GMT"
	fetchedAt := time.Now()
	if err := store.UpdateFeedFetchState(feed.ID, &etag, &lastMod, fetchedAt); err != nil {
		t.Fatalf("UpdateFeedFetchState failed: %v", err)
	}

	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.ETag == nil || *got.ETag != etag {
		t.Errorf("ETag mismatch: got %v, want %q", got.ETag, etag)
	}
	if got.LastModified == nil || *got.LastModified != lastMod {
		t.Errorf("LastModified mismatch: got %v, want %q", got.LastModified, lastMod)
	}

	// Update with error
	errMsg := "connection timeout"
	if err := store.UpdateFeedError(feed.ID, errMsg); err != nil {
		t.Fatalf("UpdateFeedError failed: %v", err)
	}

	got, err = store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.LastError == nil || *got.LastError != errMsg {
		t.Errorf("LastError mismatch: got %v, want %q", got.LastError, errMsg)
	}
	if got.ErrorCount != 1 {
		t.Errorf("ErrorCount mismatch: got %d, want 1", got.ErrorCount)
	}

	// Update with another error
	if err := store.UpdateFeedError(feed.ID, "another error"); err != nil {
		t.Fatalf("UpdateFeedError failed: %v", err)
	}

	got, err = store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.ErrorCount != 2 {
		t.Errorf("ErrorCount mismatch: got %d, want 2", got.ErrorCount)
	}
}

func TestCascadeDelete(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed with entries
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := models.NewEntry(feed.ID, "guid-"+string(rune('0'+i)), "Article")
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	// Verify entries exist
	entries, err := store.ListEntries(&EntryFilter{FeedID: &feed.ID})
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}

	// Delete feed
	if err := store.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	// Verify entries are gone
	entries, err = store.ListEntries(nil)
	if err != nil {
		t.Fatalf("ListEntries after delete failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after cascade delete, got %d", len(entries))
	}
}

func TestStats(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feeds
	feed1 := models.NewFeed("https://example1.com/feed.xml")
	title1 := "Feed 1"
	feed1.Title = &title1
	if err := store.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	feed2 := models.NewFeed("https://example2.com/feed.xml")
	title2 := "Feed 2"
	feed2.Title = &title2
	if err := store.CreateFeed(feed2); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create entries
	for i := 0; i < 3; i++ {
		entry := models.NewEntry(feed1.ID, "guid-1-"+string(rune('0'+i)), "Article")
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		entry := models.NewEntry(feed2.ID, "guid-2-"+string(rune('0'+i)), "Article")
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
		if i == 0 {
			if err := store.MarkEntryRead(entry.ID); err != nil {
				t.Fatalf("MarkEntryRead failed: %v", err)
			}
		}
	}

	// Check overall stats
	overall, err := store.GetOverallStats()
	if err != nil {
		t.Fatalf("GetOverallStats failed: %v", err)
	}
	if overall.TotalFeeds != 2 {
		t.Errorf("TotalFeeds: got %d, want 2", overall.TotalFeeds)
	}
	if overall.TotalEntries != 5 {
		t.Errorf("TotalEntries: got %d, want 5", overall.TotalEntries)
	}
	if overall.UnreadCount != 4 {
		t.Errorf("UnreadCount: got %d, want 4", overall.UnreadCount)
	}

	// Check feed stats
	feedStats, err := store.GetFeedStats()
	if err != nil {
		t.Fatalf("GetFeedStats failed: %v", err)
	}
	if len(feedStats) != 2 {
		t.Errorf("expected 2 feed stats, got %d", len(feedStats))
	}
}

func TestSearch(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create entries with content
	entry1 := models.NewEntry(feed.ID, "guid-1", "Golang Tutorial")
	content1 := "Learn how to build web applications with Go"
	entry1.Content = &content1
	if err := store.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry2 := models.NewEntry(feed.ID, "guid-2", "Python Basics")
	content2 := "Introduction to Python programming"
	entry2.Content = &content2
	if err := store.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry3 := models.NewEntry(feed.ID, "guid-3", "Web Development")
	content3 := "Building modern web apps with golang"
	entry3.Content = &content3
	if err := store.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Search for "golang"
	results, err := store.Search("golang", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'golang', got %d", len(results))
	}

	// Search for "python"
	results, err = store.Search("python", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'python', got %d", len(results))
	}
}

func TestCompact(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Just verify it doesn't error
	if err := store.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
}

func TestGetByIDOrPrefix(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Test GetFeedByURLOrPrefix with URL
	got, err := store.GetFeedByURLOrPrefix("https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("GetFeedByURLOrPrefix with URL failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, feed.ID)
	}

	// Test GetFeedByURLOrPrefix with prefix
	got, err = store.GetFeedByURLOrPrefix(feed.ID[:8])
	if err != nil {
		t.Fatalf("GetFeedByURLOrPrefix with prefix failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, feed.ID)
	}

	// Create entry
	entry := models.NewEntry(feed.ID, "guid-1", "Test")
	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Test GetEntryByIDOrPrefix with full ID
	gotEntry, err := store.GetEntryByIDOrPrefix(entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByIDOrPrefix with ID failed: %v", err)
	}
	if gotEntry.ID != entry.ID {
		t.Errorf("ID mismatch: got %q, want %q", gotEntry.ID, entry.ID)
	}

	// Test GetEntryByIDOrPrefix with prefix
	gotEntry, err = store.GetEntryByIDOrPrefix(entry.ID[:8])
	if err != nil {
		t.Fatalf("GetEntryByIDOrPrefix with prefix failed: %v", err)
	}
	if gotEntry.ID != entry.ID {
		t.Errorf("ID mismatch: got %q, want %q", gotEntry.ID, entry.ID)
	}
}

func TestPrefixTooShort(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	_, err := store.GetFeedByPrefix("abc")
	if err == nil {
		t.Error("expected error for prefix too short")
	}

	_, err = store.GetEntryByPrefix("abc")
	if err == nil {
		t.Error("expected error for prefix too short")
	}
}

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}

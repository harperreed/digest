// ABOUTME: Tests for MarkdownStore file-based storage backend
// ABOUTME: Covers all CRUD operations for feeds and entries, stats, search, and edge cases

package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

// newTestMarkdownStore creates a MarkdownStore in a temporary directory for testing.
func newTestMarkdownStore(t *testing.T) *MarkdownStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewMarkdownStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test markdown store: %v", err)
	}
	return store
}

func TestNewMarkdownStore(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "digest-data")

	store, err := NewMarkdownStore(dataDir)
	if err != nil {
		t.Fatalf("NewMarkdownStore failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("NewMarkdownStore returned nil")
	}

	// Verify data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatal("Data directory was not created")
	}
}

func TestMarkdownClose(t *testing.T) {
	store := newTestMarkdownStore(t)
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMarkdownFeedCRUD(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownEntryCRUD(t *testing.T) {
	store := newTestMarkdownStore(t)
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
	if got.Link == nil || *got.Link != link {
		t.Errorf("Link mismatch: got %v, want %q", got.Link, link)
	}
	if got.Author == nil || *got.Author != author {
		t.Errorf("Author mismatch: got %v, want %q", got.Author, author)
	}
	if got.Content == nil || *got.Content != content {
		t.Errorf("Content mismatch: got %v, want %q", got.Content, content)
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

	// Update entry
	newEntryTitle := "Updated Title"
	entry.Title = &newEntryTitle
	newContent := "Updated content"
	entry.Content = &newContent
	if err := store.UpdateEntry(entry); err != nil {
		t.Fatalf("UpdateEntry failed: %v", err)
	}

	got, err = store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry after update failed: %v", err)
	}
	if got.Title == nil || *got.Title != newEntryTitle {
		t.Errorf("Title mismatch: got %v, want %q", got.Title, newEntryTitle)
	}
	if got.Content == nil || *got.Content != newContent {
		t.Errorf("Content mismatch: got %v, want %q", got.Content, newContent)
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

func TestMarkdownListEntriesFilter(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownListEntriesUntil(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	now := time.Now()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-1 * time.Hour)

	entry1 := models.NewEntry(feed.ID, "guid-1", "Old Article")
	entry1.PublishedAt = &old
	if err := store.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry2 := models.NewEntry(feed.ID, "guid-2", "Recent Article")
	entry2.PublishedAt = &recent
	if err := store.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Test until filter
	cutoff := now.Add(-24 * time.Hour)
	filter := &EntryFilter{Until: &cutoff}
	result, err := store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries until failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 old entry, got %d", len(result))
	}
}

func TestMarkdownListEntriesMultipleFeedIDs(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create feeds
	feed1 := models.NewFeed("https://example1.com/feed.xml")
	if err := store.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	feed2 := models.NewFeed("https://example2.com/feed.xml")
	if err := store.CreateFeed(feed2); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	feed3 := models.NewFeed("https://example3.com/feed.xml")
	if err := store.CreateFeed(feed3); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create entries
	entry1 := models.NewEntry(feed1.ID, "guid-1", "Article 1")
	if err := store.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry2 := models.NewEntry(feed2.ID, "guid-2", "Article 2")
	if err := store.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	entry3 := models.NewEntry(feed3.ID, "guid-3", "Article 3")
	if err := store.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	// Filter by multiple feed IDs
	filter := &EntryFilter{FeedIDs: []string{feed1.ID, feed2.ID}}
	result, err := store.ListEntries(filter)
	if err != nil {
		t.Fatalf("ListEntries with FeedIDs failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries for 2 feeds, got %d", len(result))
	}
}

func TestMarkdownMarkEntriesReadBefore(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownFeedFetchState(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownCascadeDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownStats(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownSearch(t *testing.T) {
	store := newTestMarkdownStore(t)
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

	// Search for "golang" (case insensitive)
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

	// Search for title match (case insensitive)
	results, err = store.Search("GOLANG", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'GOLANG', got %d", len(results))
	}
}

func TestMarkdownCompact(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Should be a no-op but not error
	if err := store.Compact(); err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
}

func TestMarkdownGetByIDOrPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownPrefixTooShort(t *testing.T) {
	store := newTestMarkdownStore(t)
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

func TestMarkdownCountUnreadEntriesByFeed(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create feeds
	feed1 := models.NewFeed("https://example1.com/feed.xml")
	if err := store.CreateFeed(feed1); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	feed2 := models.NewFeed("https://example2.com/feed.xml")
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
	}

	// Count unread for specific feed
	count, err := store.CountUnreadEntries(&feed1.ID)
	if err != nil {
		t.Fatalf("CountUnreadEntries failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 unread for feed1, got %d", count)
	}

	count, err = store.CountUnreadEntries(&feed2.ID)
	if err != nil {
		t.Fatalf("CountUnreadEntries failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread for feed2, got %d", count)
	}
}

func TestMarkdownDuplicateFeedURL(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed1 := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed1); err != nil {
		t.Fatalf("First CreateFeed failed: %v", err)
	}

	feed2 := models.NewFeed("https://example.com/feed.xml")
	err := store.CreateFeed(feed2)
	if err == nil {
		t.Error("expected error creating feed with duplicate URL")
	}
}

func TestMarkdownFeedDirectoryCreated(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	title := "My Test Feed"
	feed.Title = &title
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// The feed directory should exist
	feedDir := filepath.Join(store.dataDir, "my-test-feed")
	if _, err := os.Stat(feedDir); os.IsNotExist(err) {
		t.Error("feed directory should be created")
	}
}

func TestMarkdownFeedDirectoryRemovedOnDelete(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	title := "Delete Me"
	feed.Title = &title
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Verify directory exists
	feedDir := filepath.Join(store.dataDir, "delete-me")
	if _, err := os.Stat(feedDir); os.IsNotExist(err) {
		t.Fatal("feed directory should exist")
	}

	if err := store.DeleteFeed(feed.ID); err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	// Directory should be gone
	if _, err := os.Stat(feedDir); !os.IsNotExist(err) {
		t.Error("feed directory should be removed on delete")
	}
}

func TestMarkdownEntryWithMultilineContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Entry with various markdown formatting
	content := "# Heading\n\nSome paragraph.\n\n```go\nfunc main() {\n\tprintln(\"hello\")\n}\n```\n\n- list item 1\n- list item 2\n\n> Blockquote here"
	entry := models.NewEntry(feed.ID, "guid-1", "Markdown Content")
	entry.Content = &content
	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	got, err := store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if got.Content == nil || *got.Content != content {
		t.Errorf("multiline content mismatch:\nwant: %q\ngot:  %q", content, *got.Content)
	}
}

func TestMarkdownEntryWithYAMLLikeContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Entry content that looks like YAML frontmatter
	content := "---\ntitle: Something\nauthor: Someone\n---\nActual content here."
	entry := models.NewEntry(feed.ID, "guid-1", "YAML-like content")
	entry.Content = &content
	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	got, err := store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if got.Content == nil || *got.Content != content {
		t.Errorf("content with YAML-like content was corrupted:\nwant: %q\ngot:  %q", content, *got.Content)
	}
}

func TestMarkdownFeedNoTitle(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Feed without title should use URL hostname for directory name
	feed := models.NewFeed("https://blog.example.com/rss")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if got.URL != "https://blog.example.com/rss" {
		t.Errorf("URL mismatch: got %q", got.URL)
	}

	// A directory named something like "blog-example-com" should exist
	dirEntries, _ := os.ReadDir(store.dataDir)
	found := false
	for _, de := range dirEntries {
		if de.IsDir() && strings.Contains(de.Name(), "blog-example-com") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected feed directory based on hostname")
	}
}

func TestMarkdownEntryFileName(t *testing.T) {
	entry := models.NewEntry("feed-id", "guid-1", "My Great Article")
	fileName := entryFileName(entry)

	if !strings.HasSuffix(fileName, ".md") {
		t.Errorf("expected .md suffix, got %q", fileName)
	}
	if !strings.Contains(fileName, "my-great-article") {
		t.Errorf("expected slug in filename, got %q", fileName)
	}
	if !strings.Contains(fileName, entry.ID[:8]) {
		t.Errorf("expected ID prefix in filename, got %q", fileName)
	}
}

func TestMarkdownDeleteNonexistentFeed(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	err := store.DeleteFeed("nonexistent-id")
	if err == nil {
		t.Error("expected error deleting nonexistent feed")
	}
}

func TestMarkdownDeleteNonexistentEntry(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	err := store.DeleteEntry("nonexistent-id")
	if err == nil {
		t.Error("expected error deleting nonexistent entry")
	}
}

func TestMarkdownUpdateNonexistentFeed(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	err := store.UpdateFeed(feed)
	if err == nil {
		t.Error("expected error updating nonexistent feed")
	}
}

func TestMarkdownSearchLimit(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Create 5 entries all matching "golang"
	for i := 0; i < 5; i++ {
		entry := models.NewEntry(feed.ID, "guid-"+string(rune('0'+i)), "Golang Article")
		content := "Learn golang programming"
		entry.Content = &content
		if err := store.CreateEntry(entry); err != nil {
			t.Fatalf("CreateEntry failed: %v", err)
		}
	}

	// Search with limit
	results, err := store.Search("golang", 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results with limit, got %d", len(results))
	}
}

func TestMarkdownEntryNilContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Entry with no content
	entry := models.NewEntry(feed.ID, "guid-1", "Title Only")
	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	got, err := store.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if got.Title == nil || *got.Title != "Title Only" {
		t.Errorf("Title mismatch: got %v", got.Title)
	}
}

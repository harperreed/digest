// ABOUTME: Integration tests for full RSS feed workflow
// ABOUTME: Tests end-to-end scenarios including fetch, parse, OPML, and caching

package test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/opml"
	"github.com/harper/digest/internal/parse"
)

// TestFullWorkflow tests the complete RSS feed workflow from fetch to database
func TestFullWorkflow(t *testing.T) {
	// Create temp directory for database and OPML
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	opmlPath := filepath.Join(tmpDir, "feeds.opml")

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create OPML document
	doc := opml.NewDocument("Test Feeds")

	// Add xkcd feed to database and OPML
	feedURL := "https://xkcd.com/rss.xml"
	feed := models.NewFeed(feedURL)
	feed.Title = stringPtr("XKCD")

	if err := db.CreateFeed(database, feed); err != nil {
		t.Fatalf("failed to create feed in database: %v", err)
	}

	if err := doc.AddFeed(feedURL, "XKCD", "Comics"); err != nil {
		t.Fatalf("failed to add feed to OPML: %v", err)
	}

	// Write OPML to file
	if err := doc.WriteFile(opmlPath); err != nil {
		t.Fatalf("failed to write OPML file: %v", err)
	}

	// Fetch the feed
	t.Logf("Fetching feed from %s", feedURL)
	result, err := fetch.Fetch(feedURL, nil, nil)
	if err != nil {
		t.Fatalf("failed to fetch feed: %v", err)
	}

	if result.NotModified {
		t.Fatal("unexpected NotModified response on initial fetch")
	}

	if len(result.Body) == 0 {
		t.Fatal("fetched feed body is empty")
	}

	t.Logf("Fetched %d bytes", len(result.Body))

	// Parse the feed
	parsedFeed, err := parse.Parse(result.Body)
	if err != nil {
		t.Fatalf("failed to parse feed: %v", err)
	}

	if parsedFeed.Title == "" {
		t.Error("parsed feed has no title")
	}

	if len(parsedFeed.Entries) == 0 {
		t.Fatal("parsed feed has no entries")
	}

	t.Logf("Parsed feed: %s with %d entries", parsedFeed.Title, len(parsedFeed.Entries))

	// Create entries in database
	entriesCreated := 0
	for _, parsedEntry := range parsedFeed.Entries {
		// Check if entry already exists
		exists, err := db.EntryExists(database, feed.ID, parsedEntry.GUID)
		if err != nil {
			t.Fatalf("failed to check entry existence: %v", err)
		}
		if exists {
			continue
		}

		// Create new entry
		entry := models.NewEntry(feed.ID, parsedEntry.GUID, parsedEntry.Title)
		entry.Link = &parsedEntry.Link
		entry.Author = &parsedEntry.Author
		entry.PublishedAt = parsedEntry.PublishedAt
		entry.Content = &parsedEntry.Content

		if err := db.CreateEntry(database, entry); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
		entriesCreated++
	}

	t.Logf("Created %d entries in database", entriesCreated)

	// Verify database contains entries
	entries, err := db.ListEntries(database, &feed.ID, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list entries: %v", err)
	}

	if len(entries) != entriesCreated {
		t.Errorf("expected %d entries, got %d", entriesCreated, len(entries))
	}

	// Count unread entries before marking any as read
	unreadBefore, err := db.CountUnreadEntries(database, nil)
	if err != nil {
		t.Fatalf("failed to count unread entries: %v", err)
	}

	if unreadBefore != entriesCreated {
		t.Errorf("expected %d unread entries, got %d", entriesCreated, unreadBefore)
	}

	// Mark first entry as read
	if len(entries) > 0 {
		firstEntry := entries[0]
		if err := db.MarkEntryRead(database, firstEntry.ID); err != nil {
			t.Fatalf("failed to mark entry as read: %v", err)
		}

		// Verify unread count decreased
		unreadAfter, err := db.CountUnreadEntries(database, nil)
		if err != nil {
			t.Fatalf("failed to count unread entries after marking read: %v", err)
		}

		expectedUnread := unreadBefore - 1
		if unreadAfter != expectedUnread {
			t.Errorf("expected %d unread entries after marking one read, got %d", expectedUnread, unreadAfter)
		}

		// Verify the entry is actually marked as read
		updatedEntry, err := db.GetEntryByID(database, firstEntry.ID)
		if err != nil {
			t.Fatalf("failed to get updated entry: %v", err)
		}

		if !updatedEntry.Read {
			t.Error("entry should be marked as read")
		}

		if updatedEntry.ReadAt == nil {
			t.Error("entry ReadAt timestamp should be set")
		}
	}

	t.Log("Full workflow test completed successfully")
}

// TestOPMLRoundTrip tests OPML creation, writing, and reading
func TestOPMLRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	opmlPath := filepath.Join(tmpDir, "test_roundtrip.opml")

	// Create document with folders and feeds
	doc := opml.NewDocument("Test RSS Feeds")

	// Add some folders
	if err := doc.AddFolder("Tech"); err != nil {
		t.Fatalf("failed to add Tech folder: %v", err)
	}

	if err := doc.AddFolder("Comics"); err != nil {
		t.Fatalf("failed to add Comics folder: %v", err)
	}

	// Add feeds to folders
	if err := doc.AddFeed("https://example.com/tech/feed.xml", "Tech Blog", "Tech"); err != nil {
		t.Fatalf("failed to add feed to Tech folder: %v", err)
	}

	if err := doc.AddFeed("https://xkcd.com/rss.xml", "XKCD", "Comics"); err != nil {
		t.Fatalf("failed to add feed to Comics folder: %v", err)
	}

	// Add a root-level feed (no folder)
	if err := doc.AddFeed("https://example.com/root.xml", "Root Feed", ""); err != nil {
		t.Fatalf("failed to add root feed: %v", err)
	}

	// Write to temp file
	if err := doc.WriteFile(opmlPath); err != nil {
		t.Fatalf("failed to write OPML file: %v", err)
	}

	// Parse back from file
	loadedDoc, err := opml.ParseFile(opmlPath)
	if err != nil {
		t.Fatalf("failed to parse OPML file: %v", err)
	}

	// Verify document title
	if loadedDoc.Title != "Test RSS Feeds" {
		t.Errorf("expected title 'Test RSS Feeds', got %s", loadedDoc.Title)
	}

	// Verify folders
	folders := loadedDoc.Folders()
	if len(folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(folders))
	}

	// Verify all feeds preserved
	allFeeds := loadedDoc.AllFeeds()
	if len(allFeeds) != 3 {
		t.Errorf("expected 3 feeds, got %d", len(allFeeds))
	}

	// Verify feeds in specific folder
	techFeeds := loadedDoc.FeedsInFolder("Tech")
	if len(techFeeds) != 1 {
		t.Errorf("expected 1 feed in Tech folder, got %d", len(techFeeds))
	}

	if len(techFeeds) > 0 && techFeeds[0].URL != "https://example.com/tech/feed.xml" {
		t.Errorf("unexpected feed URL in Tech folder: %s", techFeeds[0].URL)
	}

	// Verify root-level feeds
	rootFeeds := loadedDoc.FeedsInFolder("")
	if len(rootFeeds) != 1 {
		t.Errorf("expected 1 root-level feed, got %d", len(rootFeeds))
	}

	t.Log("OPML round-trip test completed successfully")
}

// TestCachedSync tests HTTP caching with ETag and conditional requests
func TestCachedSync(t *testing.T) {
	feedURL := "https://xkcd.com/rss.xml"

	// First fetch without cache headers
	t.Logf("Fetching %s for the first time", feedURL)
	result1, err := fetch.Fetch(feedURL, nil, nil)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}

	if result1.NotModified {
		t.Error("first fetch should not be NotModified")
	}

	t.Logf("First fetch: %d bytes, ETag=%q, Last-Modified=%q",
		len(result1.Body), result1.ETag, result1.LastModified)

	// Check if feed supports caching
	if result1.ETag == "" && result1.LastModified == "" {
		t.Log("Feed does not support ETag or Last-Modified headers (caching not available)")
		t.Log("This is informational - the feed server doesn't provide caching headers")
		return
	}

	// Wait a moment to ensure we're not hitting rate limits
	time.Sleep(2 * time.Second)

	// Second fetch with cache headers
	t.Log("Fetching again with cache headers")
	var etag *string
	var lastModified *string

	if result1.ETag != "" {
		etag = &result1.ETag
	}
	if result1.LastModified != "" {
		lastModified = &result1.LastModified
	}

	result2, err := fetch.Fetch(feedURL, etag, lastModified)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}

	// The server should return 304 Not Modified if content hasn't changed
	// However, this depends on the server behavior and timing
	if result2.NotModified {
		t.Log("Feed returned 304 Not Modified - caching working as expected")
	} else {
		t.Logf("Feed returned fresh content (not cached): %d bytes", len(result2.Body))
		t.Log("This might indicate the feed was updated between requests")

		// Verify we got valid content
		if len(result2.Body) == 0 {
			t.Error("second fetch should return body if not cached")
		}
	}

	t.Log("Cached sync test completed successfully")
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

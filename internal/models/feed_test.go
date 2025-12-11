// ABOUTME: Test suite for Feed model, validating feed creation and cache header management
// ABOUTME: Ensures feed instances have proper IDs, timestamps, and can store HTTP caching metadata

package models

import (
	"testing"
	"time"
)

func TestNewFeed(t *testing.T) {
	url := "https://example.com/feed.xml"
	feed := NewFeed(url)

	// Verify URL is set correctly
	if feed.URL != url {
		t.Errorf("expected URL to be %q, got %q", url, feed.URL)
	}

	// Verify ID is generated (non-empty)
	if feed.ID == "" {
		t.Error("expected feed ID to be generated, got empty string")
	}

	// Verify CreatedAt is set (not zero)
	if feed.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero time")
	}

	// Verify CreatedAt is recent (within last second)
	now := time.Now()
	if feed.CreatedAt.After(now) || feed.CreatedAt.Before(now.Add(-time.Second)) {
		t.Errorf("expected CreatedAt to be recent, got %v", feed.CreatedAt)
	}
}

func TestFeed_SetCacheHeaders(t *testing.T) {
	feed := NewFeed("https://example.com/feed.xml")

	etag := `"abc123"`
	lastModified := "Mon, 02 Jan 2006 15:04:05 GMT"

	feed.SetCacheHeaders(etag, lastModified)

	// Verify ETag is set
	if feed.ETag == nil || *feed.ETag != etag {
		t.Errorf("expected ETag to be %q, got %v", etag, feed.ETag)
	}

	// Verify LastModified is set
	if feed.LastModified == nil || *feed.LastModified != lastModified {
		t.Errorf("expected LastModified to be %q, got %v", lastModified, feed.LastModified)
	}
}

func TestNewEntry(t *testing.T) {
	feedID := "feed-123"
	guid := "entry-guid-456"
	title := "Test Entry Title"

	entry := NewEntry(feedID, guid, title)

	// Verify FeedID is set correctly
	if entry.FeedID != feedID {
		t.Errorf("expected FeedID to be %q, got %q", feedID, entry.FeedID)
	}

	// Verify GUID is set correctly
	if entry.GUID != guid {
		t.Errorf("expected GUID to be %q, got %q", guid, entry.GUID)
	}

	// Verify Title is set correctly
	if entry.Title == nil || *entry.Title != title {
		t.Errorf("expected Title to be %q, got %v", title, entry.Title)
	}

	// Verify ID is generated (non-empty)
	if entry.ID == "" {
		t.Error("expected entry ID to be generated, got empty string")
	}

	// Verify CreatedAt is set (not zero)
	if entry.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero time")
	}

	// Verify CreatedAt is recent (within last second)
	now := time.Now()
	if entry.CreatedAt.After(now) || entry.CreatedAt.Before(now.Add(-time.Second)) {
		t.Errorf("expected CreatedAt to be recent, got %v", entry.CreatedAt)
	}

	// Verify Read is initially false
	if entry.Read {
		t.Error("expected Read to be false initially, got true")
	}

	// Verify ReadAt is initially nil
	if entry.ReadAt != nil {
		t.Errorf("expected ReadAt to be nil initially, got %v", entry.ReadAt)
	}
}

func TestEntry_MarkRead(t *testing.T) {
	entry := NewEntry("feed-123", "guid-456", "Test Entry")

	// Mark the entry as read
	entry.MarkRead()

	// Verify Read is now true
	if !entry.Read {
		t.Error("expected Read to be true after MarkRead, got false")
	}

	// Verify ReadAt is set (not nil)
	if entry.ReadAt == nil {
		t.Error("expected ReadAt to be set after MarkRead, got nil")
	}

	// Verify ReadAt is recent (within last second)
	now := time.Now()
	if entry.ReadAt.After(now) || entry.ReadAt.Before(now.Add(-time.Second)) {
		t.Errorf("expected ReadAt to be recent, got %v", entry.ReadAt)
	}
}

func TestEntry_MarkUnread(t *testing.T) {
	entry := NewEntry("feed-123", "guid-456", "Test Entry")

	// First mark as read
	entry.MarkRead()

	// Verify it's marked as read
	if !entry.Read {
		t.Error("expected Read to be true after MarkRead, got false")
	}

	// Now mark as unread
	entry.MarkUnread()

	// Verify Read is now false
	if entry.Read {
		t.Error("expected Read to be false after MarkUnread, got true")
	}

	// Verify ReadAt is now nil
	if entry.ReadAt != nil {
		t.Errorf("expected ReadAt to be nil after MarkUnread, got %v", entry.ReadAt)
	}
}

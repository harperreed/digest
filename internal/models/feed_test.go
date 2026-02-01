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

func TestFeed_GetTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    *string
		expected string
	}{
		{
			name:     "nil title",
			title:    nil,
			expected: DefaultFeedTitle,
		},
		{
			name:     "empty title",
			title:    stringPtr(""),
			expected: DefaultFeedTitle,
		},
		{
			name:     "valid title",
			title:    stringPtr("My Feed"),
			expected: "My Feed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := NewFeed("https://example.com/feed.xml")
			feed.Title = tt.title

			got := feed.GetTitle()
			if got != tt.expected {
				t.Errorf("GetTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFeed_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		title    *string
		url      string
		expected string
	}{
		{
			name:     "nil title returns URL",
			title:    nil,
			url:      "https://example.com/feed.xml",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "empty title returns URL",
			title:    stringPtr(""),
			url:      "https://example.com/feed.xml",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "valid title returns title",
			title:    stringPtr("My Feed"),
			url:      "https://example.com/feed.xml",
			expected: "My Feed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := NewFeed(tt.url)
			feed.Title = tt.title

			got := feed.GetDisplayName()
			if got != tt.expected {
				t.Errorf("GetDisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateFeedURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com/feed.xml",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/feed.xml",
			wantErr: false,
		},
		{
			name:    "ftp scheme not allowed",
			url:     "ftp://example.com/feed.xml",
			wantErr: true,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "file scheme not allowed",
			url:     "file:///feed.xml",
			wantErr: true,
			errMsg:  "URL must use http or https scheme",
		},
		{
			name:    "missing host",
			url:     "https:///feed.xml",
			wantErr: true,
			errMsg:  "URL must have a host",
		},
		{
			name:    "invalid URL format",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateFeedURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFeedURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateFeedURL(%q) error = %v, want error containing %q", tt.url, err, tt.errMsg)
				}
			}
			if err == nil && result == nil {
				t.Errorf("ValidateFeedURL(%q) returned nil result without error", tt.url)
			}
		})
	}
}

func TestFeed_SetCacheHeaders_Empty(t *testing.T) {
	feed := NewFeed("https://example.com/feed.xml")

	// Set empty values should not update
	feed.SetCacheHeaders("", "")

	if feed.ETag != nil {
		t.Errorf("expected ETag to be nil with empty string, got %v", feed.ETag)
	}
	if feed.LastModified != nil {
		t.Errorf("expected LastModified to be nil with empty string, got %v", feed.LastModified)
	}
}

func TestFeed_SetCacheHeaders_Partial(t *testing.T) {
	feed := NewFeed("https://example.com/feed.xml")

	// Set only etag
	feed.SetCacheHeaders("etag-value", "")

	if feed.ETag == nil || *feed.ETag != "etag-value" {
		t.Errorf("expected ETag to be 'etag-value', got %v", feed.ETag)
	}
	if feed.LastModified != nil {
		t.Errorf("expected LastModified to be nil, got %v", feed.LastModified)
	}

	// Set only lastModified on a new feed
	feed2 := NewFeed("https://example.com/feed2.xml")
	feed2.SetCacheHeaders("", "last-modified-value")

	if feed2.ETag != nil {
		t.Errorf("expected ETag to be nil, got %v", feed2.ETag)
	}
	if feed2.LastModified == nil || *feed2.LastModified != "last-modified-value" {
		t.Errorf("expected LastModified to be 'last-modified-value', got %v", feed2.LastModified)
	}
}

func TestEntry_GetTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    *string
		expected string
	}{
		{
			name:     "nil title",
			title:    nil,
			expected: DefaultEntryTitle,
		},
		{
			name:     "empty title",
			title:    stringPtr(""),
			expected: DefaultEntryTitle,
		},
		{
			name:     "valid title",
			title:    stringPtr("My Entry"),
			expected: "My Entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewEntry("feed-123", "guid-456", "Original")
			entry.Title = tt.title

			got := entry.GetTitle()
			if got != tt.expected {
				t.Errorf("GetTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Helper functions for tests
func stringPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

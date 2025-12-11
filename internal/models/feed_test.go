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

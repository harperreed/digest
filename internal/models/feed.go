// ABOUTME: Feed model representing an RSS/Atom feed source with HTTP caching support
// ABOUTME: Tracks feed metadata, fetch history, and conditional request headers (ETag, Last-Modified)

package models

import (
	"time"

	"github.com/google/uuid"
)

// Feed represents an RSS/Atom feed subscription
type Feed struct {
	ID            string     // Unique identifier for the feed
	URL           string     // Feed URL
	Title         *string    // Feed title (from RSS/Atom metadata)
	Folder        string     // Folder for organization (empty = root)
	ETag          *string    // HTTP ETag header for conditional requests
	LastModified  *string    // HTTP Last-Modified header for conditional requests
	LastFetchedAt *time.Time // Timestamp of last successful fetch
	LastError     *string    // Last error message (if any)
	ErrorCount    int        // Consecutive error count for backoff strategy
	CreatedAt     time.Time  // Feed creation timestamp
}

// NewFeed creates a new Feed instance with a generated ID and timestamp
func NewFeed(url string) *Feed {
	return &Feed{
		ID:        uuid.New().String(),
		URL:       url,
		CreatedAt: time.Now(),
	}
}

// SetCacheHeaders updates the feed's HTTP caching headers for conditional requests
func (f *Feed) SetCacheHeaders(etag, lastModified string) {
	if etag != "" {
		f.ETag = &etag
	}
	if lastModified != "" {
		f.LastModified = &lastModified
	}
}

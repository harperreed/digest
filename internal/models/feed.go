// ABOUTME: Feed model representing an RSS/Atom feed source with HTTP caching support
// ABOUTME: Tracks feed metadata, fetch history, and conditional request headers (ETag, Last-Modified)

package models

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

const DefaultFeedTitle = "Untitled Feed"

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

// GetTitle returns the feed title, or "Untitled Feed" if not set
func (f *Feed) GetTitle() string {
	if f.Title != nil && *f.Title != "" {
		return *f.Title
	}
	return DefaultFeedTitle
}

// GetDisplayName returns the feed title if set, otherwise the URL
func (f *Feed) GetDisplayName() string {
	if f.Title != nil && *f.Title != "" {
		return *f.Title
	}
	return f.URL
}

// ValidateFeedURL checks that a URL is valid for use as a feed URL.
// Returns the parsed URL if valid, or an error describing the problem.
func ValidateFeedURL(urlStr string) (*url.URL, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL must have a host")
	}
	return parsedURL, nil
}

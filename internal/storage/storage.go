// ABOUTME: Storage interface and types for digest data persistence
// ABOUTME: Defines the contract for feed and entry storage operations

package storage

import (
	"time"

	"github.com/harper/digest/internal/models"
)

// EntryFilter specifies criteria for listing entries.
type EntryFilter struct {
	FeedID     *string
	FeedIDs    []string
	UnreadOnly *bool
	Since      *time.Time
	Until      *time.Time
	Limit      *int
	Offset     *int
}

// FeedStatsRow represents statistics for a single feed.
type FeedStatsRow struct {
	FeedID        string
	FeedURL       string
	FeedTitle     *string
	LastFetchedAt *time.Time
	ErrorCount    int
	LastError     *string
	EntryCount    int
	UnreadCount   int
}

// OverallStats represents overall statistics.
type OverallStats struct {
	TotalFeeds   int
	TotalEntries int
	UnreadCount  int
}

// Store defines the storage interface for digest data.
type Store interface {
	// Close closes the store and releases resources.
	Close() error

	// Feed Operations

	// CreateFeed stores a new feed.
	CreateFeed(feed *models.Feed) error

	// GetFeed retrieves a feed by ID.
	GetFeed(id string) (*models.Feed, error)

	// GetFeedByURL finds a feed by its URL.
	GetFeedByURL(url string) (*models.Feed, error)

	// GetFeedByPrefix finds a feed by ID prefix (min 6 chars).
	GetFeedByPrefix(prefix string) (*models.Feed, error)

	// ListFeeds returns all feeds, sorted by creation date (newest first).
	ListFeeds() ([]*models.Feed, error)

	// UpdateFeed updates an existing feed.
	UpdateFeed(feed *models.Feed) error

	// DeleteFeed removes a feed and all its entries (cascade).
	DeleteFeed(id string) error

	// UpdateFeedFetchState updates feed caching headers and clears errors.
	UpdateFeedFetchState(feedID string, etag, lastModified *string, fetchedAt time.Time) error

	// UpdateFeedError records a fetch error for a feed.
	UpdateFeedError(feedID string, errMsg string) error

	// Entry Operations

	// CreateEntry stores a new entry.
	CreateEntry(entry *models.Entry) error

	// GetEntry retrieves an entry by ID.
	GetEntry(id string) (*models.Entry, error)

	// GetEntryByPrefix finds an entry by ID prefix (min 6 chars).
	GetEntryByPrefix(prefix string) (*models.Entry, error)

	// ListEntries returns entries matching the filter, sorted by published date.
	ListEntries(filter *EntryFilter) ([]*models.Entry, error)

	// UpdateEntry updates an existing entry.
	UpdateEntry(entry *models.Entry) error

	// DeleteEntry removes an entry.
	DeleteEntry(id string) error

	// MarkEntryRead marks an entry as read.
	MarkEntryRead(id string) error

	// MarkEntryUnread marks an entry as unread.
	MarkEntryUnread(id string) error

	// MarkEntriesReadBefore marks all unread entries before the given time as read.
	MarkEntriesReadBefore(before time.Time) (int64, error)

	// EntryExists checks if an entry exists with the given feed_id and guid.
	EntryExists(feedID, guid string) (bool, error)

	// CountUnreadEntries counts unread entries, optionally filtered by feedID.
	CountUnreadEntries(feedID *string) (int, error)

	// Statistics

	// GetFeedStats retrieves statistics for all feeds.
	GetFeedStats() ([]FeedStatsRow, error)

	// GetOverallStats retrieves overall statistics.
	GetOverallStats() (*OverallStats, error)

	// Retrieval helpers

	// GetEntryByIDOrPrefix tries to get an entry by exact ID first,
	// then falls back to prefix matching if not found.
	GetEntryByIDOrPrefix(ref string) (*models.Entry, error)

	// GetFeedByURLOrPrefix tries to get a feed by exact URL first,
	// then falls back to prefix matching if not found.
	GetFeedByURLOrPrefix(ref string) (*models.Feed, error)

	// Maintenance

	// Compact performs database maintenance (VACUUM).
	Compact() error

	// Search performs full-text search on entries.
	Search(query string, limit int) ([]*models.Entry, error)
}

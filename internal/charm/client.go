// ABOUTME: Charm KV client wrapper using transactional Do API
// ABOUTME: Short-lived connections to avoid lock contention with other MCP servers

package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/google/uuid"
	"github.com/harper/digest/internal/models"
)

const (
	// Key prefixes for KV store
	FeedPrefix  = "feed:"
	EntryPrefix = "entry:"

	// Default Charm server
	DefaultCharmHost = "charm.2389.dev"

	// DBName is the name of the charm kv database for digest.
	DBName = "digest"
)

// Client holds configuration for KV operations.
// Unlike the previous implementation, it does NOT hold a persistent connection.
// Each operation opens the database, performs the operation, and closes it.
type Client struct {
	dbName   string
	autoSync bool
}

// NewClient creates a new client.
func NewClient() (*Client, error) {
	// Set Charm server before operations
	if os.Getenv("CHARM_HOST") == "" {
		os.Setenv("CHARM_HOST", DefaultCharmHost)
	}

	return &Client{
		dbName:   DBName,
		autoSync: true, // Auto-sync enabled for seamless multi-device sync
	}, nil
}

// DoReadOnly executes a function with read-only database access.
// Use this for batch read operations that need multiple Gets.
func (c *Client) DoReadOnly(fn func(k *kv.KV) error) error {
	return kv.DoReadOnly(c.dbName, fn)
}

// Do executes a function with write access to the database.
// Use this for batch write operations.
func (c *Client) Do(fn func(k *kv.KV) error) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := fn(k); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// SetAutoSync enables or disables automatic sync after writes.
func (c *Client) SetAutoSync(enabled bool) {
	c.autoSync = enabled
}

// Sync manually triggers a sync with the Charm server.
func (c *Client) Sync() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Sync()
	})
}

// Reset wipes all local data (for sync wipe command).
func (c *Client) Reset() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Reset()
	})
}

// ID returns the user's Charm ID for status display.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	return cc.ID()
}

// Close is a no-op for backwards compatibility.
// With Do API, connections are automatically closed after each operation.
func (c *Client) Close() error {
	return nil
}

// GetCharmClient returns a new Charm client for low-level operations.
func GetCharmClient() (*client.Client, error) {
	return client.NewClientWithDefaults()
}

// Feed Operations

func feedKey(id string) []byte {
	return []byte(FeedPrefix + id)
}

// CreateFeed stores a new feed.
func (c *Client) CreateFeed(feed *models.Feed) error {
	data, err := json.Marshal(feed)
	if err != nil {
		return fmt.Errorf("marshal feed: %w", err)
	}

	return c.Do(func(k *kv.KV) error {
		return k.Set(feedKey(feed.ID), data)
	})
}

// GetFeed retrieves a feed by ID.
func (c *Client) GetFeed(id string) (*models.Feed, error) {
	var feed models.Feed
	err := c.DoReadOnly(func(k *kv.KV) error {
		data, err := k.Get(feedKey(id))
		if err != nil {
			return fmt.Errorf("get feed: %w", err)
		}
		if data == nil {
			return fmt.Errorf("feed not found: %s", id)
		}
		return json.Unmarshal(data, &feed)
	})
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

// GetFeedByURL finds a feed by its URL.
func (c *Client) GetFeedByURL(url string) (*models.Feed, error) {
	feeds, err := c.ListFeeds()
	if err != nil {
		return nil, err
	}

	for _, feed := range feeds {
		if feed.URL == url {
			return feed, nil
		}
	}
	return nil, fmt.Errorf("feed not found with URL: %s", url)
}

// GetFeedByPrefix finds a feed by ID prefix (min 6 chars).
func (c *Client) GetFeedByPrefix(prefix string) (*models.Feed, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	feeds, err := c.ListFeeds()
	if err != nil {
		return nil, err
	}

	var matches []*models.Feed
	for _, feed := range feeds {
		if strings.HasPrefix(feed.ID, prefix) {
			matches = append(matches, feed)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no feed found with prefix %s", prefix)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d feeds", prefix, len(matches))
	}
	return matches[0], nil
}

// ListFeeds returns all feeds, sorted by creation date (newest first).
func (c *Client) ListFeeds() ([]*models.Feed, error) {
	var feeds []*models.Feed
	prefix := []byte(FeedPrefix)
	warnedCorruption := false

	err := c.DoReadOnly(func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return fmt.Errorf("list keys: %w", err)
		}

		feeds = make([]*models.Feed, 0, len(keys))

		for _, key := range keys {
			if !strings.HasPrefix(string(key), string(prefix)) {
				continue
			}

			data, err := k.Get(key)
			if err != nil {
				if !warnedCorruption {
					fmt.Fprintf(os.Stderr, "Warning: some feeds may be corrupted\n")
					warnedCorruption = true
				}
				continue
			}

			var feed models.Feed
			if err := json.Unmarshal(data, &feed); err != nil {
				if !warnedCorruption {
					fmt.Fprintf(os.Stderr, "Warning: some feeds may be corrupted\n")
					warnedCorruption = true
				}
				continue
			}
			feeds = append(feeds, &feed)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by created_at descending
	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].CreatedAt.After(feeds[j].CreatedAt)
	})

	return feeds, nil
}

// UpdateFeed updates an existing feed.
func (c *Client) UpdateFeed(feed *models.Feed) error {
	return c.CreateFeed(feed) // Same as create (overwrite)
}

// DeleteFeed removes a feed and all its entries (cascade).
func (c *Client) DeleteFeed(id string) error {
	return c.Do(func(k *kv.KV) error {
		// First delete all entries for this feed
		if err := c.deleteEntriesForFeedWithKV(k, id); err != nil {
			return fmt.Errorf("delete feed entries: %w", err)
		}

		// Then delete the feed itself
		if err := k.Delete(feedKey(id)); err != nil {
			return fmt.Errorf("delete feed: %w", err)
		}
		return nil
	})
}

// deleteEntriesForFeedWithKV removes all entries for a feed (internal helper).
func (c *Client) deleteEntriesForFeedWithKV(k *kv.KV, feedID string) error {
	keys, err := k.Keys()
	if err != nil {
		return err
	}

	prefix := []byte(EntryPrefix)
	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := k.Get(key)
		if err != nil {
			continue
		}

		var entry models.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		if entry.FeedID == feedID {
			_ = k.Delete(key)
		}
	}
	return nil
}

// UpdateFeedFetchState updates feed caching headers and clears errors.
func (c *Client) UpdateFeedFetchState(feedID string, etag, lastModified *string, fetchedAt time.Time) error {
	feed, err := c.GetFeed(feedID)
	if err != nil {
		return err
	}

	feed.ETag = etag
	feed.LastModified = lastModified
	feed.LastFetchedAt = &fetchedAt
	feed.LastError = nil
	feed.ErrorCount = 0

	return c.UpdateFeed(feed)
}

// UpdateFeedError records a fetch error for a feed.
func (c *Client) UpdateFeedError(feedID string, errMsg string) error {
	feed, err := c.GetFeed(feedID)
	if err != nil {
		return err
	}

	feed.LastError = &errMsg
	feed.ErrorCount++

	return c.UpdateFeed(feed)
}

// Entry Operations

func entryKey(id string) []byte {
	return []byte(EntryPrefix + id)
}

// CreateEntry stores a new entry.
func (c *Client) CreateEntry(entry *models.Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	return c.Do(func(k *kv.KV) error {
		return k.Set(entryKey(entry.ID), data)
	})
}

// GetEntry retrieves an entry by ID.
func (c *Client) GetEntry(id string) (*models.Entry, error) {
	var entry models.Entry
	err := c.DoReadOnly(func(k *kv.KV) error {
		data, err := k.Get(entryKey(id))
		if err != nil {
			return fmt.Errorf("get entry: %w", err)
		}
		if data == nil {
			return fmt.Errorf("entry not found: %s", id)
		}
		return json.Unmarshal(data, &entry)
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetEntryByPrefix finds an entry by ID prefix (min 6 chars).
func (c *Client) GetEntryByPrefix(prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	var matches []*models.Entry
	entryPfx := []byte(EntryPrefix)

	err := c.DoReadOnly(func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return err
		}

		matches = make([]*models.Entry, 0, 2) // Expect 0-2 matches for prefix lookup

		for _, key := range keys {
			if !strings.HasPrefix(string(key), string(entryPfx)) {
				continue
			}

			// Extract ID from key
			id := strings.TrimPrefix(string(key), EntryPrefix)
			if !strings.HasPrefix(id, prefix) {
				continue
			}

			data, err := k.Get(key)
			if err != nil {
				continue
			}

			var entry models.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}
			matches = append(matches, &entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no entry found with prefix %s", prefix)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d entries", prefix, len(matches))
	}
	return matches[0], nil
}

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

// entryMatchesFilter checks if an entry passes the filter criteria.
func entryMatchesFilter(entry *models.Entry, filter *EntryFilter, feedIDSet map[string]bool) bool {
	if filter == nil {
		return true
	}

	// FeedIDs takes precedence over FeedID
	if len(filter.FeedIDs) > 0 && !feedIDSet[entry.FeedID] {
		return false
	}
	if len(filter.FeedIDs) == 0 && filter.FeedID != nil && entry.FeedID != *filter.FeedID {
		return false
	}

	if filter.UnreadOnly != nil && *filter.UnreadOnly && entry.Read {
		return false
	}

	if filter.Since != nil && entry.PublishedAt != nil && entry.PublishedAt.Before(*filter.Since) {
		return false
	}

	if filter.Until != nil && entry.PublishedAt != nil && !entry.PublishedAt.Before(*filter.Until) {
		return false
	}

	return true
}

// applyPagination applies offset and limit to the entries slice.
func applyPagination(entries []*models.Entry, filter *EntryFilter) []*models.Entry {
	if filter == nil {
		return entries
	}

	offset := 0
	if filter.Offset != nil {
		offset = *filter.Offset
	}

	if offset > len(entries) {
		return nil
	}
	entries = entries[offset:]

	if filter.Limit != nil && *filter.Limit < len(entries) {
		entries = entries[:*filter.Limit]
	}

	return entries
}

// ListEntries returns entries matching the filter, sorted by published date.
func (c *Client) ListEntries(filter *EntryFilter) ([]*models.Entry, error) {
	var entries []*models.Entry
	prefix := []byte(EntryPrefix)
	warnedCorruption := false

	// Build feed ID set for efficient lookup
	feedIDSet := make(map[string]bool)
	if filter != nil {
		for _, id := range filter.FeedIDs {
			feedIDSet[id] = true
		}
	}

	err := c.DoReadOnly(func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return fmt.Errorf("list keys: %w", err)
		}

		entries = make([]*models.Entry, 0, len(keys))

		for _, key := range keys {
			if !strings.HasPrefix(string(key), string(prefix)) {
				continue
			}

			data, err := k.Get(key)
			if err != nil {
				if !warnedCorruption {
					fmt.Fprintf(os.Stderr, "Warning: some entries may be corrupted\n")
					warnedCorruption = true
				}
				continue
			}

			var entry models.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				if !warnedCorruption {
					fmt.Fprintf(os.Stderr, "Warning: some entries may be corrupted\n")
					warnedCorruption = true
				}
				continue
			}

			if entryMatchesFilter(&entry, filter, feedIDSet) {
				entries = append(entries, &entry)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by published_at descending
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].PublishedAt == nil {
			return false
		}
		if entries[j].PublishedAt == nil {
			return true
		}
		return entries[i].PublishedAt.After(*entries[j].PublishedAt)
	})

	return applyPagination(entries, filter), nil
}

// UpdateEntry updates an existing entry.
func (c *Client) UpdateEntry(entry *models.Entry) error {
	return c.CreateEntry(entry) // Same as create (overwrite)
}

// DeleteEntry removes an entry.
func (c *Client) DeleteEntry(id string) error {
	return c.Do(func(k *kv.KV) error {
		return k.Delete(entryKey(id))
	})
}

// MarkEntryRead marks an entry as read.
func (c *Client) MarkEntryRead(id string) error {
	entry, err := c.GetEntry(id)
	if err != nil {
		return err
	}

	entry.MarkRead()
	return c.UpdateEntry(entry)
}

// MarkEntryUnread marks an entry as unread.
func (c *Client) MarkEntryUnread(id string) error {
	entry, err := c.GetEntry(id)
	if err != nil {
		return err
	}

	entry.MarkUnread()
	return c.UpdateEntry(entry)
}

// MarkEntriesReadBefore marks all unread entries before the given time as read.
// Returns the count of entries marked.
func (c *Client) MarkEntriesReadBefore(before time.Time) (int64, error) {
	var count int64
	now := time.Now()

	err := c.Do(func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return err
		}

		prefix := []byte(EntryPrefix)
		for _, key := range keys {
			if !strings.HasPrefix(string(key), string(prefix)) {
				continue
			}

			data, err := k.Get(key)
			if err != nil {
				continue
			}

			var entry models.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}

			if entry.Read {
				continue
			}

			if entry.PublishedAt != nil && entry.PublishedAt.Before(before) {
				entry.Read = true
				entry.ReadAt = &now

				updatedData, err := json.Marshal(&entry)
				if err != nil {
					continue
				}

				if err := k.Set(key, updatedData); err != nil {
					continue
				}
				count++
			}
		}
		return nil
	})

	return count, err
}

// EntryExists checks if an entry exists with the given feed_id and guid.
func (c *Client) EntryExists(feedID, guid string) (bool, error) {
	var exists bool
	prefix := []byte(EntryPrefix)

	err := c.DoReadOnly(func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return err
		}

		for _, key := range keys {
			if !strings.HasPrefix(string(key), string(prefix)) {
				continue
			}

			data, err := k.Get(key)
			if err != nil {
				continue
			}

			var entry models.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}

			if entry.FeedID == feedID && entry.GUID == guid {
				exists = true
				return nil
			}
		}
		return nil
	})

	return exists, err
}

// CountUnreadEntries counts unread entries, optionally filtered by feedID.
func (c *Client) CountUnreadEntries(feedID *string) (int, error) {
	unreadOnly := true
	filter := &EntryFilter{
		FeedID:     feedID,
		UnreadOnly: &unreadOnly,
	}

	entries, err := c.ListEntries(filter)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// Stats

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

// GetFeedStats retrieves statistics for all feeds.
func (c *Client) GetFeedStats() ([]FeedStatsRow, error) {
	feeds, err := c.ListFeeds()
	if err != nil {
		return nil, err
	}

	entries, err := c.ListEntries(nil)
	if err != nil {
		return nil, err
	}

	// Count entries per feed
	entryCounts := make(map[string]int)
	unreadCounts := make(map[string]int)

	for _, entry := range entries {
		entryCounts[entry.FeedID]++
		if !entry.Read {
			unreadCounts[entry.FeedID]++
		}
	}

	stats := make([]FeedStatsRow, 0, len(feeds))
	for _, feed := range feeds {
		stats = append(stats, FeedStatsRow{
			FeedID:        feed.ID,
			FeedURL:       feed.URL,
			FeedTitle:     feed.Title,
			LastFetchedAt: feed.LastFetchedAt,
			ErrorCount:    feed.ErrorCount,
			LastError:     feed.LastError,
			EntryCount:    entryCounts[feed.ID],
			UnreadCount:   unreadCounts[feed.ID],
		})
	}

	return stats, nil
}

// OverallStats represents overall statistics.
type OverallStats struct {
	TotalFeeds   int
	TotalEntries int
	UnreadCount  int
}

// GetOverallStats retrieves overall statistics.
func (c *Client) GetOverallStats() (*OverallStats, error) {
	feeds, err := c.ListFeeds()
	if err != nil {
		return nil, err
	}

	entries, err := c.ListEntries(nil)
	if err != nil {
		return nil, err
	}

	unreadCount := 0
	for _, entry := range entries {
		if !entry.Read {
			unreadCount++
		}
	}

	return &OverallStats{
		TotalFeeds:   len(feeds),
		TotalEntries: len(entries),
		UnreadCount:  unreadCount,
	}, nil
}

// ============================================================================
// Retrieval helpers (exact ID or prefix)
// ============================================================================

// GetEntryByIDOrPrefix tries to get an entry by exact ID first,
// then falls back to prefix matching if not found.
func (c *Client) GetEntryByIDOrPrefix(ref string) (*models.Entry, error) {
	entry, err := c.GetEntry(ref)
	if err == nil {
		return entry, nil
	}

	// Try prefix match
	entry, err = c.GetEntryByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("entry not found: %s", ref)
	}
	return entry, nil
}

// GetFeedByURLOrPrefix tries to get a feed by exact URL first,
// then falls back to prefix matching if not found.
func (c *Client) GetFeedByURLOrPrefix(ref string) (*models.Feed, error) {
	feed, err := c.GetFeedByURL(ref)
	if err == nil {
		return feed, nil
	}

	// Try prefix match
	feed, err = c.GetFeedByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("feed not found: %s", ref)
	}
	return feed, nil
}

// ============================================================================
// Helper for creating new entries (preserves existing interface)

// NewFeed creates a new feed with generated ID.
func NewFeed(url string) *models.Feed {
	return &models.Feed{
		ID:        uuid.New().String(),
		URL:       url,
		CreatedAt: time.Now(),
	}
}

// NewEntry creates a new entry with generated ID.
func NewEntry(feedID, guid, title string) *models.Entry {
	now := time.Now()
	return &models.Entry{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		GUID:      guid,
		Title:     &title,
		Read:      false,
		CreatedAt: now,
	}
}

// ============================================================================
// Legacy compatibility layer
// ============================================================================

var globalClient *Client

// InitClient initializes the global charm client.
// With the new architecture, this just creates a Client instance.
func InitClient() (*Client, error) {
	if globalClient != nil {
		return globalClient, nil
	}
	var err error
	globalClient, err = NewClient()
	return globalClient, err
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	return InitClient()
}

// NewTestClient creates a Client for testing.
// The db parameter is ignored (kept for backward compatibility).
// The autoSync parameter controls whether writes trigger sync (usually false for tests).
func NewTestClient(db *kv.KV, autoSync bool) *Client {
	return &Client{
		dbName:   DBName,
		autoSync: autoSync,
	}
}

// NewTestClientWithDBName creates a Client for testing with a custom database name.
// Use this when you need isolated test databases.
func NewTestClientWithDBName(dbName string, autoSync bool) *Client {
	return &Client{
		dbName:   dbName,
		autoSync: autoSync,
	}
}

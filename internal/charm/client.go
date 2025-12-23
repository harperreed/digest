// ABOUTME: Charm KV client wrapper for digest with automatic cloud sync
// ABOUTME: Provides thread-safe initialization and sync-on-write for seamless multi-device sync

package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
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
)

var (
	globalClient *Client
	clientOnce   sync.Once
	clientErr    error
)

// Client wraps Charm KV with digest-specific operations.
type Client struct {
	kv       *kv.KV
	mu       sync.RWMutex
	autoSync bool
}

// entryMeta contains entry metadata without full content for cloud sync.
// Content is excluded to reduce charm cloud storage usage - articles can be
// re-fetched from their original URLs when needed.
type entryMeta struct {
	ID          string     `json:"id"`
	FeedID      string     `json:"feed_id"`
	GUID        string     `json:"guid"`
	Title       *string    `json:"title,omitempty"`
	Link        *string    `json:"link,omitempty"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Read        bool       `json:"read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	// Content is intentionally excluded - stored locally only
}

// InitClient initializes the global Charm client.
// Thread-safe via sync.Once.
func InitClient() (*Client, error) {
	clientOnce.Do(func() {
		// Set Charm server before opening KV
		if os.Getenv("CHARM_HOST") == "" {
			os.Setenv("CHARM_HOST", DefaultCharmHost)
		}

		db, err := kv.OpenWithDefaultsFallback("digest")
		if err != nil {
			clientErr = fmt.Errorf("open charm kv: %w", err)
			return
		}

		globalClient = &Client{
			kv:       db,
			autoSync: true,
		}

		// Pull remote data on startup (skip in read-only mode)
		if globalClient.autoSync && !globalClient.kv.IsReadOnly() {
			_ = globalClient.kv.Sync()
		}
	})

	if clientErr != nil {
		return nil, clientErr
	}
	return globalClient, nil
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	return InitClient()
}

// Close closes the KV store.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kv != nil {
		return c.kv.Close()
	}
	return nil
}

// syncIfEnabled calls Sync() if auto-sync is enabled.
// CRITICAL: Must be called after every write operation!
// NOTE: Caller must NOT hold c.mu lock - this function accesses c.autoSync directly
// since it's only modified during initialization or via SetAutoSync.
func (c *Client) syncIfEnabled() error {
	if c.autoSync && !c.kv.IsReadOnly() {
		if err := c.kv.Sync(); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
	}
	return nil
}

// SetAutoSync enables or disables automatic sync after writes.
func (c *Client) SetAutoSync(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoSync = enabled
}

// IsReadOnly returns true if the database is open in read-only mode.
// This happens when another process (like an MCP server) holds the lock.
func (c *Client) IsReadOnly() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kv.IsReadOnly()
}

// Sync manually triggers a sync with the Charm server.
func (c *Client) Sync() error {
	if c.kv.IsReadOnly() {
		return nil // Skip sync in read-only mode
	}
	return c.kv.Sync()
}

// Reset wipes all local data (for sync wipe command).
func (c *Client) Reset() error {
	return c.kv.Reset()
}

// ID returns the user's Charm ID for status display.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	return cc.ID()
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	data, err := json.Marshal(feed)
	if err != nil {
		return fmt.Errorf("marshal feed: %w", err)
	}

	if err := c.kv.Set(feedKey(feed.ID), data); err != nil {
		return fmt.Errorf("set feed: %w", err)
	}

	if err := c.syncIfEnabled(); err != nil {
		// Log but don't fail - sync errors shouldn't break local operations
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return nil
}

// GetFeed retrieves a feed by ID.
func (c *Client) GetFeed(id string) (*models.Feed, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := c.kv.Get(feedKey(id))
	if err != nil {
		return nil, fmt.Errorf("get feed: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("feed not found: %s", id)
	}

	var feed models.Feed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("unmarshal feed: %w", err)
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
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	feeds := make([]*models.Feed, 0, len(keys))
	prefix := []byte(FeedPrefix)
	warnedCorruption := false

	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := c.kv.Get(key)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	// First delete all entries for this feed
	if err := c.deleteEntriesForFeedLocked(id); err != nil {
		return fmt.Errorf("delete feed entries: %w", err)
	}

	// Then delete the feed itself
	if err := c.kv.Delete(feedKey(id)); err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}

	if err := c.syncIfEnabled(); err != nil {
		// Log but don't fail - sync errors shouldn't break local operations
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return nil
}

// deleteEntriesForFeedLocked removes all entries for a feed (must hold lock).
func (c *Client) deleteEntriesForFeedLocked(feedID string) error {
	keys, err := c.kv.Keys()
	if err != nil {
		return err
	}

	prefix := []byte(EntryPrefix)
	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var entry models.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		if entry.FeedID == feedID {
			_ = c.kv.Delete(key)
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

// toEntryMeta converts an Entry to entryMeta, excluding Content for cloud sync.
func toEntryMeta(e *models.Entry) entryMeta {
	return entryMeta{
		ID:          e.ID,
		FeedID:      e.FeedID,
		GUID:        e.GUID,
		Title:       e.Title,
		Link:        e.Link,
		Author:      e.Author,
		PublishedAt: e.PublishedAt,
		Read:        e.Read,
		ReadAt:      e.ReadAt,
		CreatedAt:   e.CreatedAt,
	}
}

// CreateEntry stores a new entry (metadata only, content excluded from cloud sync).
func (c *Client) CreateEntry(entry *models.Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	// Store metadata only - Content is excluded to save charm cloud storage
	meta := toEntryMeta(entry)
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if err := c.kv.Set(entryKey(entry.ID), data); err != nil {
		return fmt.Errorf("set entry: %w", err)
	}

	if err := c.syncIfEnabled(); err != nil {
		// Log but don't fail - sync errors shouldn't break local operations
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return nil
}

// GetEntry retrieves an entry by ID.
func (c *Client) GetEntry(id string) (*models.Entry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := c.kv.Get(entryKey(id))
	if err != nil {
		return nil, fmt.Errorf("get entry: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("entry not found: %s", id)
	}

	var entry models.Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}
	return &entry, nil
}

// GetEntryByPrefix finds an entry by ID prefix (min 6 chars).
func (c *Client) GetEntryByPrefix(prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	keys, err := c.kv.Keys()
	if err != nil {
		return nil, err
	}

	matches := make([]*models.Entry, 0, 2) // Expect 0-2 matches for prefix lookup
	entryPfx := []byte(EntryPrefix)

	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(entryPfx)) {
			continue
		}

		// Extract ID from key
		id := strings.TrimPrefix(string(key), EntryPrefix)
		if !strings.HasPrefix(id, prefix) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var entry models.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		matches = append(matches, &entry)
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
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	entries := make([]*models.Entry, 0, len(keys))
	prefix := []byte(EntryPrefix)
	warnedCorruption := false

	// Build feed ID set for efficient lookup
	feedIDSet := make(map[string]bool)
	if filter != nil {
		for _, id := range filter.FeedIDs {
			feedIDSet[id] = true
		}
	}

	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := c.kv.Get(key)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	if err := c.kv.Delete(entryKey(id)); err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}

	if err := c.syncIfEnabled(); err != nil {
		// Log but don't fail - sync errors shouldn't break local operations
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return nil
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return 0, fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	keys, err := c.kv.Keys()
	if err != nil {
		return 0, err
	}

	var count int64
	prefix := []byte(EntryPrefix)
	now := time.Now()

	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := c.kv.Get(key)
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

			if err := c.kv.Set(key, updatedData); err != nil {
				continue
			}
			count++
		}
	}

	if err := c.syncIfEnabled(); err != nil {
		// Log but don't fail - sync errors shouldn't break local operations
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return count, nil
}

// EntryExists checks if an entry exists with the given feed_id and guid.
func (c *Client) EntryExists(feedID, guid string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys, err := c.kv.Keys()
	if err != nil {
		return false, err
	}

	prefix := []byte(EntryPrefix)
	for _, key := range keys {
		if !strings.HasPrefix(string(key), string(prefix)) {
			continue
		}

		data, err := c.kv.Get(key)
		if err != nil {
			continue
		}

		var entry models.Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		if entry.FeedID == feedID && entry.GUID == guid {
			return true, nil
		}
	}
	return false, nil
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

// NewTestClient creates a Client for testing with a provided KV store.
// The autoSync parameter controls whether writes trigger sync (usually false for tests).
func NewTestClient(db *kv.KV, autoSync bool) *Client {
	return &Client{
		kv:       db,
		autoSync: autoSync,
	}
}

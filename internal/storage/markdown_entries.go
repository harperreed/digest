// ABOUTME: Entry CRUD operations for MarkdownStore
// ABOUTME: Persists entries as markdown files with YAML frontmatter in per-feed directories

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/harper/suite/mdstore"

	"github.com/harper/digest/internal/models"
)

// CreateEntry stores a new entry.
func (s *MarkdownStore) CreateEntry(entry *models.Entry) error {
	slug, err := s.feedSlugByID(entry.FeedID)
	if err != nil {
		return fmt.Errorf("create entry: %w", err)
	}

	feedDir := s.feedDirPath(slug)
	if err := mdstore.EnsureDir(feedDir); err != nil {
		return fmt.Errorf("create feed directory: %w", err)
	}

	fileName := entryFileName(entry)
	filePath := filepath.Join(feedDir, fileName)

	return writeEntryFile(filePath, entry)
}

// GetEntry retrieves an entry by ID.
func (s *MarkdownStore) GetEntry(id string) (*models.Entry, error) {
	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	for _, fe := range entries {
		feedDir := s.feedDirPath(fe.Slug)
		fp, err := findEntryFile(feedDir, id)
		if err != nil {
			continue
		}
		return readEntryFile(fp)
	}
	return nil, fmt.Errorf("entry not found")
}

// GetEntryByPrefix finds an entry by ID prefix (min 6 chars).
func (s *MarkdownStore) GetEntryByPrefix(prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	var matches []*models.Entry
	for _, fe := range entries {
		feedDir := s.feedDirPath(fe.Slug)
		feedEntries, err := readAllEntries(feedDir)
		if err != nil {
			continue
		}
		for _, e := range feedEntries {
			if strings.HasPrefix(e.ID, prefix) {
				matches = append(matches, e)
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no entry found with prefix %s", prefix)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d entries", prefix, len(matches))
	}
	return matches[0], nil
}

// ListEntries returns entries matching the filter, sorted by published date.
func (s *MarkdownStore) ListEntries(filter *EntryFilter) ([]*models.Entry, error) {
	feeds, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	feedSlugs := s.selectFeedSlugs(feeds, filter)

	allEntries := s.collectEntries(feedSlugs)

	// Apply filters
	if filter != nil {
		allEntries = applyEntryFilters(allEntries, filter)
	}

	// Sort by published date, newest first
	sort.Slice(allEntries, func(i, j int) bool {
		ti := entryPublishedTime(allEntries[i])
		tj := entryPublishedTime(allEntries[j])
		return ti.After(tj)
	})

	allEntries = applyPagination(allEntries, filter)

	return allEntries, nil
}

// selectFeedSlugs determines which feed slugs to include based on the filter.
func (s *MarkdownStore) selectFeedSlugs(feeds []feedEntry, filter *EntryFilter) map[string]bool {
	feedSlugs := make(map[string]bool)

	switch {
	case filter != nil && len(filter.FeedIDs) > 0:
		feedIDSet := make(map[string]bool)
		for _, id := range filter.FeedIDs {
			feedIDSet[id] = true
		}
		for _, fe := range feeds {
			if feedIDSet[fe.ID] {
				feedSlugs[fe.Slug] = true
			}
		}
	case filter != nil && filter.FeedID != nil:
		for _, fe := range feeds {
			if fe.ID == *filter.FeedID {
				feedSlugs[fe.Slug] = true
			}
		}
	default:
		for _, fe := range feeds {
			feedSlugs[fe.Slug] = true
		}
	}

	return feedSlugs
}

// collectEntries reads all entries from the given feed slugs.
// Feeds whose directories cannot be read are silently skipped.
func (s *MarkdownStore) collectEntries(feedSlugs map[string]bool) []*models.Entry {
	var allEntries []*models.Entry
	for slug := range feedSlugs {
		feedDir := s.feedDirPath(slug)
		entries, err := readAllEntries(feedDir)
		if err != nil {
			continue
		}
		allEntries = append(allEntries, entries...)
	}
	return allEntries
}

// applyPagination applies limit and offset from the filter to the entry slice.
func applyPagination(entries []*models.Entry, filter *EntryFilter) []*models.Entry {
	if filter == nil {
		return entries
	}
	if filter.Offset != nil && *filter.Offset > 0 {
		if *filter.Offset >= len(entries) {
			return nil
		}
		entries = entries[*filter.Offset:]
	}
	if filter.Limit != nil && *filter.Limit >= 0 {
		if *filter.Limit < len(entries) {
			entries = entries[:*filter.Limit]
		}
	}
	return entries
}

// applyEntryFilters applies non-pagination filters to entries.
func applyEntryFilters(entries []*models.Entry, filter *EntryFilter) []*models.Entry {
	var result []*models.Entry
	for _, e := range entries {
		if filter.UnreadOnly != nil && *filter.UnreadOnly && e.Read {
			continue
		}
		if filter.Since != nil {
			pubTime := entryPublishedTime(e)
			if !timeAfterOrEqual(pubTime, *filter.Since) {
				continue
			}
		}
		if filter.Until != nil {
			pubTime := entryPublishedTime(e)
			if !timeBefore(pubTime, *filter.Until) {
				continue
			}
		}
		result = append(result, e)
	}
	return result
}

// entryPublishedTime returns the published time or created time as fallback.
func entryPublishedTime(e *models.Entry) time.Time {
	if e.PublishedAt != nil {
		return *e.PublishedAt
	}
	return e.CreatedAt
}

// UpdateEntry updates an existing entry.
func (s *MarkdownStore) UpdateEntry(entry *models.Entry) error {
	slug, err := s.feedSlugByID(entry.FeedID)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}

	feedDir := s.feedDirPath(slug)
	fp, err := findEntryFile(feedDir, entry.ID)
	if err != nil {
		return fmt.Errorf("entry not found: %s", entry.ID)
	}

	return writeEntryFile(fp, entry)
}

// DeleteEntry removes an entry.
func (s *MarkdownStore) DeleteEntry(id string) error {
	feeds, err := s.readFeeds()
	if err != nil {
		return err
	}

	for _, fe := range feeds {
		feedDir := s.feedDirPath(fe.Slug)
		fp, err := findEntryFile(feedDir, id)
		if err != nil {
			continue
		}
		if err := os.Remove(fp); err != nil {
			return fmt.Errorf("delete entry file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("entry not found: %s", id)
}

// MarkEntryRead marks an entry as read.
func (s *MarkdownStore) MarkEntryRead(id string) error {
	entry, err := s.GetEntry(id)
	if err != nil {
		return fmt.Errorf("entry not found: %s", id)
	}

	now := time.Now()
	entry.Read = true
	entry.ReadAt = &now

	return s.UpdateEntry(entry)
}

// MarkEntryUnread marks an entry as unread.
func (s *MarkdownStore) MarkEntryUnread(id string) error {
	entry, err := s.GetEntry(id)
	if err != nil {
		return fmt.Errorf("entry not found: %s", id)
	}

	entry.Read = false
	entry.ReadAt = nil

	return s.UpdateEntry(entry)
}

// MarkEntriesReadBefore marks all unread entries before the given time as read.
func (s *MarkdownStore) MarkEntriesReadBefore(before time.Time) (int64, error) {
	feeds, err := s.readFeeds()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var count int64

	for _, fe := range feeds {
		feedDir := s.feedDirPath(fe.Slug)
		entries, err := readAllEntries(feedDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.Read {
				continue
			}
			pubTime := entryPublishedTime(entry)
			if pubTime.Before(before) {
				entry.Read = true
				entry.ReadAt = &now
				fp, findErr := findEntryFile(feedDir, entry.ID)
				if findErr != nil {
					continue
				}
				if writeErr := writeEntryFile(fp, entry); writeErr != nil {
					continue
				}
				count++
			}
		}
	}

	return count, nil
}

// EntryExists checks if an entry exists with the given feed_id and guid.
func (s *MarkdownStore) EntryExists(feedID, guid string) (bool, error) {
	slug, err := s.feedSlugByID(feedID)
	if err != nil {
		return false, err
	}

	feedDir := s.feedDirPath(slug)
	entries, err := readAllEntries(feedDir)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		if e.FeedID == feedID && e.GUID == guid {
			return true, nil
		}
	}
	return false, nil
}

// CountUnreadEntries counts unread entries, optionally filtered by feedID.
func (s *MarkdownStore) CountUnreadEntries(feedID *string) (int, error) {
	feeds, err := s.readFeeds()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, fe := range feeds {
		if feedID != nil && fe.ID != *feedID {
			continue
		}
		feedDir := s.feedDirPath(fe.Slug)
		entries, err := readAllEntries(feedDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.Read {
				count++
			}
		}
	}
	return count, nil
}

// GetFeedStats retrieves statistics for all feeds.
func (s *MarkdownStore) GetFeedStats() ([]FeedStatsRow, error) {
	feedEntries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	var stats []FeedStatsRow
	for _, fe := range feedEntries {
		feed, err := fe.toModel()
		if err != nil {
			continue
		}

		feedDir := s.feedDirPath(fe.Slug)
		entries, _ := readAllEntries(feedDir)

		entryCount := len(entries)
		unreadCount := 0
		for _, e := range entries {
			if !e.Read {
				unreadCount++
			}
		}

		stats = append(stats, FeedStatsRow{
			FeedID:        feed.ID,
			FeedURL:       feed.URL,
			FeedTitle:     feed.Title,
			LastFetchedAt: feed.LastFetchedAt,
			ErrorCount:    feed.ErrorCount,
			LastError:     feed.LastError,
			EntryCount:    entryCount,
			UnreadCount:   unreadCount,
		})
	}
	return stats, nil
}

// GetOverallStats retrieves overall statistics.
func (s *MarkdownStore) GetOverallStats() (*OverallStats, error) {
	feedEntries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	stats := &OverallStats{
		TotalFeeds: len(feedEntries),
	}

	for _, fe := range feedEntries {
		feedDir := s.feedDirPath(fe.Slug)
		entries, _ := readAllEntries(feedDir)
		stats.TotalEntries += len(entries)
		for _, e := range entries {
			if !e.Read {
				stats.UnreadCount++
			}
		}
	}

	return stats, nil
}

// GetEntryByIDOrPrefix tries to get an entry by exact ID first,
// then falls back to prefix matching if not found.
func (s *MarkdownStore) GetEntryByIDOrPrefix(ref string) (*models.Entry, error) {
	entry, err := s.GetEntry(ref)
	if err == nil {
		return entry, nil
	}

	entry, err = s.GetEntryByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("entry not found: %s", ref)
	}
	return entry, nil
}

// GetFeedByURLOrPrefix tries to get a feed by exact URL first,
// then falls back to prefix matching if not found.
func (s *MarkdownStore) GetFeedByURLOrPrefix(ref string) (*models.Feed, error) {
	feed, err := s.GetFeedByURL(ref)
	if err == nil {
		return feed, nil
	}

	feed, err = s.GetFeedByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("feed not found: %s", ref)
	}
	return feed, nil
}

// Compact is a no-op for markdown storage.
func (s *MarkdownStore) Compact() error {
	return nil
}

// Search performs case-insensitive string matching on entry title and content.
func (s *MarkdownStore) Search(query string, limit int) ([]*models.Entry, error) {
	feeds, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []*models.Entry

	for _, fe := range feeds {
		feedDir := s.feedDirPath(fe.Slug)
		entries, err := readAllEntries(feedDir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			titleMatch := e.Title != nil && strings.Contains(strings.ToLower(*e.Title), queryLower)
			contentMatch := e.Content != nil && strings.Contains(strings.ToLower(*e.Content), queryLower)
			if titleMatch || contentMatch {
				results = append(results, e)
			}
		}
	}

	// Sort by published time, newest first
	sort.Slice(results, func(i, j int) bool {
		ti := entryPublishedTime(results[i])
		tj := entryPublishedTime(results[j])
		return ti.After(tj)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

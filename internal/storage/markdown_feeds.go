// ABOUTME: Feed CRUD operations for MarkdownStore
// ABOUTME: Persists feeds in _feeds.yaml and manages per-feed directories

package storage

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/harperreed/mdstore"

	"github.com/harper/digest/internal/models"
)

// CreateFeed stores a new feed.
func (s *MarkdownStore) CreateFeed(feed *models.Feed) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readFeeds()
		if err != nil {
			return err
		}

		// Check for duplicate URL
		for _, e := range entries {
			if e.URL == feed.URL {
				return fmt.Errorf("insert feed: feed URL %q already exists", feed.URL)
			}
		}

		slug := s.feedSlugForModel(feed)
		entries = append(entries, fromFeedModel(feed, slug))
		if err := s.writeFeeds(entries); err != nil {
			return fmt.Errorf("write feeds: %w", err)
		}

		// Create feed directory
		feedDir := s.feedDirPath(slug)
		if err := mdstore.EnsureDir(feedDir); err != nil {
			return fmt.Errorf("create feed directory: %w", err)
		}

		return nil
	})
}

// GetFeed retrieves a feed by ID.
func (s *MarkdownStore) GetFeed(id string) (*models.Feed, error) {
	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.ID == id {
			feed, err := e.toModel()
			if err != nil {
				return nil, fmt.Errorf("parse feed entry: %w", err)
			}
			return feed, nil
		}
	}
	return nil, fmt.Errorf("feed not found")
}

// GetFeedByURL finds a feed by its URL.
func (s *MarkdownStore) GetFeedByURL(url string) (*models.Feed, error) {
	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.URL == url {
			feed, err := e.toModel()
			if err != nil {
				return nil, fmt.Errorf("parse feed entry: %w", err)
			}
			return feed, nil
		}
	}
	return nil, fmt.Errorf("feed not found")
}

// GetFeedByPrefix finds a feed by ID prefix (min 6 chars).
func (s *MarkdownStore) GetFeedByPrefix(prefix string) (*models.Feed, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	var matches []*models.Feed
	for _, e := range entries {
		if strings.HasPrefix(e.ID, prefix) {
			feed, err := e.toModel()
			if err != nil {
				continue
			}
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
func (s *MarkdownStore) ListFeeds() ([]*models.Feed, error) {
	entries, err := s.readFeeds()
	if err != nil {
		return nil, err
	}

	var feeds []*models.Feed
	for _, e := range entries {
		feed, err := e.toModel()
		if err != nil {
			// Skip malformed entries
			continue
		}
		feeds = append(feeds, feed)
	}

	// Sort by creation date, newest first
	SortFeeds(feeds)
	return feeds, nil
}

// UpdateFeed updates an existing feed.
func (s *MarkdownStore) UpdateFeed(feed *models.Feed) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readFeeds()
		if err != nil {
			return err
		}

		found := false
		for i, e := range entries {
			if e.ID == feed.ID {
				entries[i] = fromFeedModel(feed, e.Slug)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("feed not found: %s", feed.ID)
		}

		return s.writeFeeds(entries)
	})
}

// DeleteFeed removes a feed and all its entries (cascade).
func (s *MarkdownStore) DeleteFeed(id string) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readFeeds()
		if err != nil {
			return err
		}

		found := false
		var slug string
		newEntries := make([]feedEntry, 0, len(entries))
		for _, e := range entries {
			if e.ID == id {
				found = true
				slug = e.Slug
				continue
			}
			newEntries = append(newEntries, e)
		}

		if !found {
			return fmt.Errorf("feed not found: %s", id)
		}

		if err := s.writeFeeds(newEntries); err != nil {
			return fmt.Errorf("write feeds: %w", err)
		}

		// Remove feed directory and all entry files
		feedDir := s.feedDirPath(slug)
		if err := os.RemoveAll(feedDir); err != nil {
			return fmt.Errorf("remove feed directory: %w", err)
		}

		return nil
	})
}

// UpdateFeedFetchState updates feed caching headers and clears errors.
func (s *MarkdownStore) UpdateFeedFetchState(feedID string, etag, lastModified *string, fetchedAt time.Time) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readFeeds()
		if err != nil {
			return err
		}

		found := false
		for i, e := range entries {
			if e.ID == feedID {
				entries[i].ETag = etag
				entries[i].LastModified = lastModified
				fetchStr := mdstore.FormatTime(fetchedAt.UTC())
				entries[i].LastFetchedAt = &fetchStr
				entries[i].LastError = nil
				entries[i].ErrorCount = 0
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("feed not found: %s", feedID)
		}

		return s.writeFeeds(entries)
	})
}

// UpdateFeedError records a fetch error for a feed.
func (s *MarkdownStore) UpdateFeedError(feedID string, errMsg string) error {
	return mdstore.WithLock(s.dataDir, func() error {
		entries, err := s.readFeeds()
		if err != nil {
			return err
		}

		found := false
		for i, e := range entries {
			if e.ID == feedID {
				entries[i].LastError = &errMsg
				entries[i].ErrorCount = e.ErrorCount + 1
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("feed not found: %s", feedID)
		}

		return s.writeFeeds(entries)
	})
}

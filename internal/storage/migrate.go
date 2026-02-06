// ABOUTME: Data migration between digest storage backends
// ABOUTME: Copies feeds and entries from source to destination store

package storage

import (
	"fmt"
	"os"
)

// MigrateSummary holds counts of migrated entities.
type MigrateSummary struct {
	Feeds   int
	Entries int
}

// MigrateData copies all data from src to dst storage.
// It iterates through feeds and entries in order,
// creating each entity in the destination. The destination should be empty
// before calling this function.
func MigrateData(src, dst Store) (*MigrateSummary, error) {
	summary := &MigrateSummary{}

	// List all feeds
	feeds, err := src.ListFeeds()
	if err != nil {
		return nil, fmt.Errorf("list source feeds: %w", err)
	}

	for _, feed := range feeds {
		if err := dst.CreateFeed(feed); err != nil {
			return nil, fmt.Errorf("create feed %q: %w", feed.URL, err)
		}
		summary.Feeds++

		if err := migrateFeedEntries(src, dst, feed.ID, summary); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

// migrateFeedEntries copies all entries for a single feed.
func migrateFeedEntries(src, dst Store, feedID string, summary *MigrateSummary) error {
	entries, err := src.ListEntries(&EntryFilter{FeedID: &feedID})
	if err != nil {
		return fmt.Errorf("list entries for feed %s: %w", feedID, err)
	}

	for _, entry := range entries {
		if err := dst.CreateEntry(entry); err != nil {
			return fmt.Errorf("create entry %s in feed %s: %w", entry.ID, feedID, err)
		}
		summary.Entries++
	}
	return nil
}

// IsDirNonEmpty checks whether a directory exists and contains any files or subdirectories.
// Returns false if the directory does not exist or is empty.
func IsDirNonEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read directory %q: %w", path, err)
	}
	return len(entries) > 0, nil
}

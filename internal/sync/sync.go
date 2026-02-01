// ABOUTME: Shared feed synchronization logic used by CLI and MCP server
// ABOUTME: Fetches, parses, and stores feed entries

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/parse"
	"github.com/harper/digest/internal/storage"
)

// SyncResult contains the outcome of a feed sync operation
type SyncResult struct {
	NewEntries int
	WasCached  bool
}

// SyncFeed fetches and processes a single feed, storing new entries.
// If force is true, ignores cache headers and re-fetches unconditionally.
func SyncFeed(ctx context.Context, store storage.Store, feed *models.Feed, force bool) (*SyncResult, error) {
	// Get cache headers (skip if force)
	var etag, lastModified *string
	if !force {
		etag = feed.ETag
		lastModified = feed.LastModified
	}

	// Fetch the feed
	result, err := fetch.Fetch(ctx, feed.URL, etag, lastModified)
	if err != nil {
		errMsg := err.Error()
		if updateErr := store.UpdateFeedError(feed.ID, errMsg); updateErr != nil {
			return nil, fmt.Errorf("fetch failed (%v) and error update failed: %w", err, updateErr)
		}
		return nil, err
	}

	// Handle 304 Not Modified
	if result.NotModified {
		return &SyncResult{NewEntries: 0, WasCached: true}, nil
	}

	// Parse the feed
	parsed, err := parse.Parse(result.Body)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse feed: %v", err)
		if updateErr := store.UpdateFeedError(feed.ID, errMsg); updateErr != nil {
			return nil, fmt.Errorf("parse failed (%v) and error update failed: %w", err, updateErr)
		}
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Update feed title if empty
	titleUpdated := false
	if feed.Title == nil || *feed.Title == "" {
		feed.Title = &parsed.Title
		titleUpdated = true
	}

	// Process entries
	newCount := 0
	for _, parsedEntry := range parsed.Entries {
		exists, err := store.EntryExists(feed.ID, parsedEntry.GUID)
		if err != nil {
			return nil, fmt.Errorf("failed to check entry existence: %w", err)
		}
		if exists {
			continue
		}

		entry := storage.NewEntry(feed.ID, parsedEntry.GUID, parsedEntry.Title)
		entry.Link = &parsedEntry.Link
		entry.Author = &parsedEntry.Author
		entry.PublishedAt = parsedEntry.PublishedAt
		entry.Content = &parsedEntry.Content

		if err := store.CreateEntry(entry); err != nil {
			return nil, fmt.Errorf("failed to create entry: %w", err)
		}
		newCount++
	}

	// Update feed fetch state
	fetchedAt := time.Now()
	if err := store.UpdateFeedFetchState(feed.ID, &result.ETag, &result.LastModified, fetchedAt); err != nil {
		return nil, fmt.Errorf("failed to update feed state: %w", err)
	}

	// Update feed if title changed
	if titleUpdated {
		if err := store.UpdateFeed(feed); err != nil {
			return nil, fmt.Errorf("failed to update feed title: %w", err)
		}
	}

	return &SyncResult{NewEntries: newCount, WasCached: false}, nil
}

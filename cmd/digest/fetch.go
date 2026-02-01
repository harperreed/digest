// ABOUTME: Fetch command to retrieve new entries from RSS/Atom feeds with HTTP caching support
// ABOUTME: Handles batch fetching of all feeds or individual feed fetch with colored progress output

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/parse"
	"github.com/harper/digest/internal/storage"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [url]",
	Short: "Fetch new entries from feeds",
	Long: `Fetch new entries from all subscribed feeds or a specific feed by URL.

Uses HTTP caching headers (ETag, Last-Modified) to avoid re-fetching unchanged content.
Use --force to ignore cache headers and fetch unconditionally.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		// Get all feeds from storage
		feeds, err := store.ListFeeds()
		if err != nil {
			return fmt.Errorf("failed to list feeds: %w", err)
		}

		if len(feeds) == 0 {
			fmt.Println("No feeds found. Add a feed with 'digest feed add <url>'")
			return nil
		}

		// Filter to specific URL if provided
		if len(args) == 1 {
			targetURL := args[0]
			filtered := []*models.Feed{}
			for _, feed := range feeds {
				if feed.URL == targetURL {
					filtered = append(filtered, feed)
					break
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("feed not found: %s", targetURL)
			}
			feeds = filtered
		}

		// Sync each feed
		totalNew := 0
		totalCached := 0
		totalErrors := 0

		green := color.New(color.FgGreen).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		faint := color.New(color.Faint).SprintFunc()

		for _, feed := range feeds {
			displayName := feedDisplayName(feed)
			fmt.Printf("Syncing %s... ", displayName)

			newCount, wasCached, err := syncFeed(feed, force)
			if err != nil {
				fmt.Printf("%s %s\n", red("x"), err.Error())
				totalErrors++
				continue
			}

			if wasCached {
				fmt.Printf("%s (cached)\n", faint("-"))
				totalCached++
			} else if newCount > 0 {
				fmt.Printf("%s %d new\n", green("v"), newCount)
				totalNew += newCount
			} else {
				fmt.Printf("%s no new entries\n", green("v"))
			}
		}

		// Print summary
		fmt.Println()
		fmt.Printf("Summary: %d feed(s) synced\n", len(feeds))
		if totalNew > 0 {
			fmt.Printf("  %s %d new entries\n", green("v"), totalNew)
		}
		if totalCached > 0 {
			fmt.Printf("  %s %d cached (not modified)\n", faint("-"), totalCached)
		}
		if totalErrors > 0 {
			fmt.Printf("  %s %d errors\n", red("x"), totalErrors)
		}

		return nil
	},
}

// syncFeed fetches and processes a single feed, returning the count of new entries
func syncFeed(feed *models.Feed, force bool) (newCount int, wasCached bool, err error) {
	// Get cache headers from feed (skip if force)
	var etag, lastModified *string
	if !force {
		etag = feed.ETag
		lastModified = feed.LastModified
	}

	// Fetch the feed
	result, err := fetch.Fetch(context.Background(), feed.URL, etag, lastModified)
	if err != nil {
		// Update error state
		if updateErr := store.UpdateFeedError(feed.ID, err.Error()); updateErr != nil {
			return 0, false, fmt.Errorf("fetch failed (%v) and error update failed: %w", err, updateErr)
		}
		return 0, false, err
	}

	// Handle 304 Not Modified
	if result.NotModified {
		return 0, true, nil
	}

	// Parse the feed
	parsed, err := parse.Parse(result.Body)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse feed: %v", err)
		if updateErr := store.UpdateFeedError(feed.ID, errMsg); updateErr != nil {
			return 0, false, fmt.Errorf("parse failed (%v) and error update failed: %w", err, updateErr)
		}
		return 0, false, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Update feed title if empty and persist
	titleUpdated := false
	if feed.Title == nil || *feed.Title == "" {
		feed.Title = &parsed.Title
		titleUpdated = true
	}

	// Process entries
	newCount = 0
	for _, parsedEntry := range parsed.Entries {
		// Check if entry already exists
		exists, err := store.EntryExists(feed.ID, parsedEntry.GUID)
		if err != nil {
			return newCount, false, fmt.Errorf("failed to check entry existence: %w", err)
		}

		if exists {
			continue
		}

		// Create new entry
		entry := storage.NewEntry(feed.ID, parsedEntry.GUID, parsedEntry.Title)
		entry.Link = &parsedEntry.Link
		entry.Author = &parsedEntry.Author
		entry.PublishedAt = parsedEntry.PublishedAt
		entry.Content = &parsedEntry.Content

		if err := store.CreateEntry(entry); err != nil {
			return newCount, false, fmt.Errorf("failed to create entry: %w", err)
		}

		newCount++
	}

	// Update feed fetch state
	fetchedAt := time.Now()
	if err := store.UpdateFeedFetchState(feed.ID, &result.ETag, &result.LastModified, fetchedAt); err != nil {
		return newCount, false, fmt.Errorf("failed to update feed state: %w", err)
	}

	// If title was updated, persist
	if titleUpdated {
		if err := store.UpdateFeed(feed); err != nil {
			return newCount, false, fmt.Errorf("failed to update feed title: %w", err)
		}
	}

	return newCount, false, nil
}

// feedDisplayName returns a human-readable name for the feed
func feedDisplayName(feed *models.Feed) string {
	if feed.Title != nil && *feed.Title != "" {
		return *feed.Title
	}
	return feed.URL
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().BoolP("force", "f", false, "ignore cache headers and force fetch")
}

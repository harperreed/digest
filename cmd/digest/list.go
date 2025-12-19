// ABOUTME: List command for viewing feed entries with filtering options
// ABOUTME: Displays entries with read status, title, and published date using color formatting

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/charm"
	"github.com/harper/digest/internal/timeutil"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List feed entries",
	Long:    "List feed entries with optional filtering by feed and read status",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		feedFilter, _ := cmd.Flags().GetString("feed")
		category, _ := cmd.Flags().GetString("category")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		today, _ := cmd.Flags().GetBool("today")
		yesterday, _ := cmd.Flags().GetBool("yesterday")
		week, _ := cmd.Flags().GetBool("week")

		// Build entry filter
		filter := &charm.EntryFilter{
			Limit:  &limit,
			Offset: &offset,
		}

		// Set unreadOnly based on --all flag
		if !all {
			unreadOnly := true
			filter.UnreadOnly = &unreadOnly
		}

		if feedFilter != "" && category != "" {
			return fmt.Errorf("cannot use --feed and --category together")
		}

		if feedFilter != "" {
			// Try exact URL match first
			feed, err := charmClient.GetFeedByURL(feedFilter)
			if err != nil {
				// Try prefix match
				feed, err = charmClient.GetFeedByPrefix(feedFilter)
				if err != nil {
					return fmt.Errorf("failed to find feed: %w", err)
				}
			}
			filter.FeedID = &feed.ID
		}

		if category != "" {
			// Get all feeds in this category from OPML
			categoryFeeds := opmlDoc.FeedsInFolder(category)
			if len(categoryFeeds) == 0 {
				return fmt.Errorf("no feeds found in category %q", category)
			}

			// Get feed IDs from Charm
			for _, opmlFeed := range categoryFeeds {
				charmFeed, err := charmClient.GetFeedByURL(opmlFeed.URL)
				if err != nil {
					continue // Skip feeds not in Charm
				}
				filter.FeedIDs = append(filter.FeedIDs, charmFeed.ID)
			}

			if len(filter.FeedIDs) == 0 {
				return fmt.Errorf("no synced feeds found in category %q", category)
			}
		}

		// Calculate date filters based on smart view flags
		if today {
			s := timeutil.StartOfToday()
			filter.Since = &s
		} else if yesterday {
			s := timeutil.StartOfYesterday()
			u := timeutil.EndOfYesterday()
			filter.Since = &s
			filter.Until = &u
		} else if week {
			s := timeutil.StartOfWeek()
			filter.Since = &s
		}

		// List entries
		entries, err := charmClient.ListEntries(filter)
		if err != nil {
			return fmt.Errorf("failed to list entries: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No entries found")
			return nil
		}

		// Create color functions
		faint := color.New(color.Faint).SprintFunc()

		// Display entries
		for _, entry := range entries {
			// ID (first 8 chars, faint) - with bounds check for safety
			idShort := entry.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Print(faint(idShort))
			fmt.Print(" ")

			// Read status (checkmark or space)
			if entry.Read {
				fmt.Print("✓ ")
			} else {
				fmt.Print("  ")
			}

			// Title
			title := "Untitled"
			if entry.Title != nil {
				title = *entry.Title
			}
			fmt.Print(title)

			// Published date (RFC822 format, faint)
			if entry.PublishedAt != nil {
				dateStr := entry.PublishedAt.Format("02 Jan 06 15:04 MST")
				fmt.Print(" ")
				fmt.Print(faint(dateStr))
			}

			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolP("all", "a", false, "show all entries including read")
	listCmd.Flags().StringP("feed", "f", "", "filter by feed URL or prefix")
	listCmd.Flags().StringP("category", "c", "", "filter by feed category/folder")
	listCmd.Flags().IntP("limit", "n", 20, "max entries to show")
	listCmd.Flags().IntP("offset", "o", 0, "number of entries to skip (for pagination)")
	listCmd.Flags().Bool("today", false, "show only today's entries")
	listCmd.Flags().Bool("yesterday", false, "show only yesterday's entries")
	listCmd.Flags().Bool("week", false, "show only this week's entries")

	listCmd.MarkFlagsMutuallyExclusive("today", "yesterday", "week")
	listCmd.MarkFlagsMutuallyExclusive("feed", "category")
}

// ABOUTME: Read command for viewing article content
// ABOUTME: Displays full article details with markdown rendering and marks as read

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/content"
	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/sync"
)

var readCmd = &cobra.Command{
	Use:   "read <entry-id>",
	Short: "Read an article",
	Long:  "Display the full content of an article and mark it as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entryRef := args[0]
		noMark, _ := cmd.Flags().GetBool("no-mark")

		// Get entry by ID or prefix
		entry, err := db.GetEntryByID(dbConn, entryRef)
		if err != nil {
			// Only try prefix match if entry was not found (not for other DB errors)
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get entry: %w", err)
			}
			entry, err = db.GetEntryByPrefix(dbConn, entryRef)
			if err != nil {
				return fmt.Errorf("entry not found: %s", entryRef)
			}
		}

		// Get feed for context
		feed, err := db.GetFeedByID(dbConn, entry.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed: %w", err)
		}

		// Color helpers
		bold := color.New(color.Bold).SprintFunc()
		faint := color.New(color.Faint).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		// Display article header
		fmt.Println(strings.Repeat("─", 60))

		// Title
		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("%s\n\n", bold(title))

		// Feed
		feedTitle := feed.URL
		if feed.Title != nil {
			feedTitle = *feed.Title
		}
		fmt.Printf("%s %s\n", faint("Feed:"), feedTitle)

		// Author
		if entry.Author != nil && *entry.Author != "" {
			fmt.Printf("%s %s\n", faint("Author:"), *entry.Author)
		}

		// Published date
		if entry.PublishedAt != nil {
			fmt.Printf("%s %s\n", faint("Published:"), entry.PublishedAt.Format("Mon, 02 Jan 2006 15:04 MST"))
		}

		// Link
		if entry.Link != nil {
			fmt.Printf("%s %s\n", faint("Link:"), cyan(*entry.Link))
		}

		fmt.Println(strings.Repeat("─", 60))

		// Content
		if entry.Content != nil && *entry.Content != "" {
			// Convert HTML to markdown if needed
			markdown := content.ToMarkdown(*entry.Content)

			// Render with glamour for terminal display
			rendered, err := glamour.Render(markdown, "dark")
			if err != nil {
				// Fall back to plain markdown if rendering fails
				fmt.Printf("%s\n", faint("(markdown rendering unavailable, showing plain text)"))
				fmt.Printf("\n%s\n", markdown)
			} else {
				fmt.Print(rendered)
			}
		} else {
			fmt.Println("\n(No content available)")
		}

		fmt.Println()

		// Mark as read unless --no-mark flag is set
		if !noMark && !entry.Read {
			if err := db.MarkEntryRead(dbConn, entry.ID); err != nil {
				return fmt.Errorf("failed to mark entry as read: %w", err)
			}

			// Queue read state sync if configured
			ctx := context.Background()
			cfg, _ := sync.LoadConfig()
			if cfg != nil && cfg.IsConfigured() {
				syncer, err := sync.NewSyncer(cfg, dbConn)
				if err == nil {
					defer syncer.Close()
					feedURL := feed.URL
					if err := syncer.QueueReadStateChange(ctx, feedURL, entry.GUID, true, time.Now()); err != nil {
						log.Printf("warning: failed to queue read state sync: %v", err)
					}
				}
			}

			fmt.Printf("%s\n", faint("Marked as read"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	readCmd.Flags().Bool("no-mark", false, "don't mark the article as read")
}

// ABOUTME: Feed management commands for adding, listing, and removing RSS/Atom feeds
// ABOUTME: Handles feed CRUD operations and syncs changes to both database and OPML file

package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/models"
)

var feedCmd = &cobra.Command{
	Use:     "feed",
	Aliases: []string{"f"},
	Short:   "Manage RSS/Atom feeds",
	Long:    "Add, list, and remove RSS/Atom feeds from your subscriptions",
}

var feedAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a new RSS/Atom feed",
	Long:  "Add a new feed to your subscriptions and sync to OPML",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		folder, _ := cmd.Flags().GetString("folder")
		title, _ := cmd.Flags().GetString("title")

		// Check if feed already exists
		existingFeed, err := db.GetFeedByURL(dbConn, url)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check for existing feed: %w", err)
		}
		if existingFeed != nil {
			return fmt.Errorf("feed already exists: %s", url)
		}

		// Create new feed
		feed := models.NewFeed(url)
		if title != "" {
			feed.Title = &title
		}

		// Save to database
		if err := db.CreateFeed(dbConn, feed); err != nil {
			return fmt.Errorf("failed to create feed in database: %w", err)
		}

		// Add to OPML
		opmlTitle := title
		if opmlTitle == "" {
			opmlTitle = url
		}
		if err := opmlDoc.AddFeed(url, opmlTitle, folder); err != nil {
			return fmt.Errorf("failed to add feed to OPML: %w", err)
		}

		// Save OPML
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		if folder != "" {
			fmt.Printf("Added feed to folder '%s': %s\n", folder, url)
		} else {
			fmt.Printf("Added feed: %s\n", url)
		}
		fmt.Printf("Feed ID: %s\n", feed.ID)

		return nil
	},
}

var feedListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all feeds",
	Long:    "List all subscribed feeds with their folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		feeds := opmlDoc.AllFeeds()
		if len(feeds) == 0 {
			fmt.Println("No feeds found. Add a feed with 'digest feed add <url>'")
			return nil
		}

		fmt.Printf("Found %d feed(s):\n\n", len(feeds))
		for _, feed := range feeds {
			if feed.Folder != "" {
				fmt.Printf("[%s] %s\n", feed.Folder, feed.Title)
			} else {
				fmt.Printf("%s\n", feed.Title)
			}
			fmt.Printf("  URL: %s\n\n", feed.URL)
		}

		return nil
	},
}

var feedRemoveCmd = &cobra.Command{
	Use:   "remove <url>",
	Short: "Remove a feed",
	Long:  "Remove a feed from your subscriptions and OPML",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		// Get feed from database
		feed, err := db.GetFeedByURL(dbConn, url)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("feed not found: %s", url)
			}
			return fmt.Errorf("failed to get feed: %w", err)
		}

		// Delete from database
		if err := db.DeleteFeed(dbConn, feed.ID); err != nil {
			return fmt.Errorf("failed to delete feed from database: %w", err)
		}

		// Remove from OPML
		if err := opmlDoc.RemoveFeed(url); err != nil {
			return fmt.Errorf("failed to remove feed from OPML: %w", err)
		}

		// Save OPML
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		fmt.Printf("Removed feed: %s\n", url)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(feedCmd)
	feedCmd.AddCommand(feedAddCmd)
	feedCmd.AddCommand(feedListCmd)
	feedCmd.AddCommand(feedRemoveCmd)

	feedAddCmd.Flags().StringP("folder", "f", "", "folder to organize feed in")
	feedAddCmd.Flags().StringP("title", "t", "", "feed title (defaults to URL)")
}

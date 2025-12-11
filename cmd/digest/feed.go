// ABOUTME: Feed management commands for adding, listing, and removing RSS/Atom feeds
// ABOUTME: Handles feed CRUD operations and syncs changes to both database and OPML file

package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/discover"
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
	Long:  "Add a new feed to your subscriptions and sync to OPML. Automatically discovers feed URLs from HTML pages.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputURL := args[0]
		folder, _ := cmd.Flags().GetString("folder")
		title, _ := cmd.Flags().GetString("title")
		noDiscover, _ := cmd.Flags().GetBool("no-discover")

		var feedURL, feedTitle string

		if noDiscover {
			// Skip discovery, use URL as-is
			feedURL = inputURL
			feedTitle = title
		} else {
			// Discover feed from URL
			fmt.Printf("Discovering feed at %s...\n", inputURL)
			discovered, err := discover.Discover(inputURL)
			if err != nil {
				return fmt.Errorf("could not find feed at %s: %w", inputURL, err)
			}

			feedURL = discovered.URL
			if title != "" {
				feedTitle = title
			} else {
				feedTitle = discovered.Title
			}

			// Inform user if URL changed
			if feedURL != inputURL {
				fmt.Printf("Found feed: %s\n", feedURL)
			}
		}

		// Check if feed already exists
		existingFeed, err := db.GetFeedByURL(dbConn, feedURL)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check for existing feed: %w", err)
		}
		if existingFeed != nil {
			return fmt.Errorf("feed already exists: %s", feedURL)
		}

		// Create new feed
		feed := models.NewFeed(feedURL)
		if feedTitle != "" {
			feed.Title = &feedTitle
		}

		// Save to database
		if err := db.CreateFeed(dbConn, feed); err != nil {
			return fmt.Errorf("failed to create feed in database: %w", err)
		}

		// Add to OPML
		opmlTitle := feedTitle
		if opmlTitle == "" {
			opmlTitle = feedURL
		}
		if err := opmlDoc.AddFeed(feedURL, opmlTitle, folder); err != nil {
			return fmt.Errorf("failed to add feed to OPML: %w", err)
		}

		// Save OPML
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		if folder != "" {
			fmt.Printf("Added feed to folder '%s': %s\n", folder, feedTitle)
		} else {
			fmt.Printf("Added feed: %s\n", feedTitle)
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

var feedMoveCmd = &cobra.Command{
	Use:   "move <url> <category>",
	Short: "Move a feed to a different category",
	Long:  "Move a feed to a different category/folder. Use empty quotes \"\" for root level.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		newFolder := args[1]

		// Verify feed exists in database
		_, err := db.GetFeedByURL(dbConn, url)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("feed not found: %s", url)
			}
			return fmt.Errorf("failed to get feed: %w", err)
		}

		// Move feed in OPML
		if err := opmlDoc.MoveFeed(url, newFolder); err != nil {
			return fmt.Errorf("failed to move feed: %w", err)
		}

		// Save OPML
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		if newFolder == "" {
			fmt.Printf("Moved feed to root level: %s\n", url)
		} else {
			fmt.Printf("Moved feed to '%s': %s\n", newFolder, url)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(feedCmd)
	feedCmd.AddCommand(feedAddCmd)
	feedCmd.AddCommand(feedListCmd)
	feedCmd.AddCommand(feedRemoveCmd)
	feedCmd.AddCommand(feedMoveCmd)

	feedAddCmd.Flags().StringP("folder", "f", "", "folder to organize feed in")
	feedAddCmd.Flags().StringP("title", "t", "", "feed title (defaults to discovered title)")
	feedAddCmd.Flags().Bool("no-discover", false, "skip feed discovery and use URL as-is")
}

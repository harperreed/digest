// ABOUTME: Feed management commands for adding, listing, and removing RSS/Atom feeds
// ABOUTME: Handles feed CRUD operations and syncs changes to both Charm KV and OPML file

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/charm"
	"github.com/harper/digest/internal/discover"
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
	Long:  "Add a new feed to your subscriptions. Automatically discovers feed URLs from HTML pages.",
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
		existingFeed, err := charmClient.GetFeedByURL(feedURL)
		if err == nil && existingFeed != nil {
			return fmt.Errorf("feed already exists: %s", feedURL)
		}

		// Create new feed
		feed := charm.NewFeed(feedURL)
		feed.Folder = folder
		if feedTitle != "" {
			feed.Title = &feedTitle
		}

		// Save to Charm KV (auto-syncs to cloud)
		if err := charmClient.CreateFeed(feed); err != nil {
			return fmt.Errorf("failed to create feed: %w", err)
		}

		// Add to OPML for import/export compatibility
		opmlTitle := feedTitle
		if opmlTitle == "" {
			opmlTitle = feedURL
		}
		if err := opmlDoc.AddFeed(feedURL, opmlTitle, folder); err != nil {
			// Non-fatal: OPML is for import/export, Charm is source of truth
			fmt.Printf("Note: Could not add to OPML: %v\n", err)
		} else {
			if err := saveOPML(); err != nil {
				fmt.Printf("Note: Could not save OPML: %v\n", err)
			}
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
		feeds, err := charmClient.ListFeeds()
		if err != nil {
			return fmt.Errorf("failed to list feeds: %w", err)
		}

		if len(feeds) == 0 {
			fmt.Println("No feeds found. Add a feed with 'digest feed add <url>'")
			return nil
		}

		fmt.Printf("Found %d feed(s):\n\n", len(feeds))
		for _, feed := range feeds {
			title := feed.URL
			if feed.Title != nil {
				title = *feed.Title
			}

			if feed.Folder != "" {
				fmt.Printf("[%s] %s\n", feed.Folder, title)
			} else {
				fmt.Printf("%s\n", title)
			}
			fmt.Printf("  URL: %s\n", feed.URL)
			fmt.Printf("  ID: %s\n\n", feed.ID)
		}

		return nil
	},
}

var feedRemoveCmd = &cobra.Command{
	Use:   "remove <url>",
	Short: "Remove a feed",
	Long:  "Remove a feed from your subscriptions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		// Get feed from Charm
		feed, err := charmClient.GetFeedByURL(url)
		if err != nil {
			return fmt.Errorf("feed not found: %s", url)
		}

		// Delete from Charm (cascade deletes entries, auto-syncs)
		if err := charmClient.DeleteFeed(feed.ID); err != nil {
			return fmt.Errorf("failed to delete feed: %w", err)
		}

		// Remove from OPML
		if err := opmlDoc.RemoveFeed(url); err != nil {
			// Non-fatal
			fmt.Printf("Note: Could not remove from OPML: %v\n", err)
		} else {
			if err := saveOPML(); err != nil {
				fmt.Printf("Note: Could not save OPML: %v\n", err)
			}
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

		// Get feed from Charm
		feed, err := charmClient.GetFeedByURL(url)
		if err != nil {
			return fmt.Errorf("feed not found: %s", url)
		}

		// Update folder
		feed.Folder = newFolder
		if err := charmClient.UpdateFeed(feed); err != nil {
			return fmt.Errorf("failed to update feed: %w", err)
		}

		// Move feed in OPML
		if err := opmlDoc.MoveFeed(url, newFolder); err != nil {
			fmt.Printf("Note: Could not move in OPML: %v\n", err)
		} else {
			if err := saveOPML(); err != nil {
				fmt.Printf("Note: Could not save OPML: %v\n", err)
			}
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

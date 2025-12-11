// ABOUTME: List command for viewing feed entries with filtering options
// ABOUTME: Displays entries with read status, title, and published date using color formatting

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List feed entries",
	Long:    "List feed entries with optional filtering by feed and read status",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		feedFilter, _ := cmd.Flags().GetString("feed")
		limit, _ := cmd.Flags().GetInt("limit")

		// Get feedID if --feed is specified
		var feedID *string
		if feedFilter != "" {
			// Try exact URL match first
			feed, err := db.GetFeedByURL(dbConn, feedFilter)
			if err != nil {
				// Try prefix match
				feed, err = db.GetFeedByPrefix(dbConn, feedFilter)
				if err != nil {
					return fmt.Errorf("failed to find feed: %w", err)
				}
			}
			feedID = &feed.ID
		}

		// Set unreadOnly based on --all flag
		unreadOnly := !all

		// List entries
		entries, err := db.ListEntries(dbConn, feedID, &unreadOnly, nil, &limit)
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
	listCmd.Flags().IntP("limit", "n", 20, "max entries to show")
}

// ABOUTME: Read and unread commands for managing entry state
// ABOUTME: Marks entries as read or unread with prefix-based lookup

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
)

var readCmd = &cobra.Command{
	Use:   "read <entry-prefix>",
	Short: "Mark an entry as read",
	Long:  "Mark an entry as read by providing its ID prefix (minimum 6 characters)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get entry by prefix
		entry, err := db.GetEntryByPrefix(dbConn, args[0])
		if err != nil {
			return fmt.Errorf("failed to find entry: %w", err)
		}

		// Mark as read
		if err := db.MarkEntryRead(dbConn, entry.ID); err != nil {
			return fmt.Errorf("failed to mark entry as read: %w", err)
		}

		// Print confirmation with title
		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("✓ Marked as read: %s\n", title)

		return nil
	},
}

var unreadCmd = &cobra.Command{
	Use:   "unread <entry-prefix>",
	Short: "Mark an entry as unread",
	Long:  "Mark an entry as unread by providing its ID prefix (minimum 6 characters)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get entry by prefix
		entry, err := db.GetEntryByPrefix(dbConn, args[0])
		if err != nil {
			return fmt.Errorf("failed to find entry: %w", err)
		}

		// Mark as unread
		if err := db.MarkEntryUnread(dbConn, entry.ID); err != nil {
			return fmt.Errorf("failed to mark entry as unread: %w", err)
		}

		// Print confirmation with title
		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("  Marked as unread: %s\n", title)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(unreadCmd)
}

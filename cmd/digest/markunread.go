// ABOUTME: Mark-unread command for marking entries as unread
// ABOUTME: Supports marking a single entry as unread by ID

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var markUnreadCmd = &cobra.Command{
	Use:   "mark-unread <entry-id>",
	Short: "Mark an entry as unread",
	Long:  "Mark a single entry as unread by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entryRef := args[0]

		// Get entry by ID or prefix
		entry, err := store.GetEntry(entryRef)
		if err != nil {
			// Try prefix match
			entry, err = store.GetEntryByPrefix(entryRef)
			if err != nil {
				return fmt.Errorf("entry not found: %s", entryRef)
			}
		}

		if !entry.Read {
			fmt.Println("Entry is already marked as unread")
			return nil
		}

		if err := store.MarkEntryUnread(entry.ID); err != nil {
			return fmt.Errorf("failed to mark entry as unread: %w", err)
		}

		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("Marked as unread: %s\n", title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(markUnreadCmd)
}

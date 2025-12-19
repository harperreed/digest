// ABOUTME: Mark-read command for marking entries as read
// ABOUTME: Supports single entry by ID or bulk operations by date

package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/timeutil"
)

var markReadCmd = &cobra.Command{
	Use:   "mark-read [entry-id]",
	Short: "Mark entries as read",
	Long:  "Mark a single entry as read by ID, or use --before to mark all entries older than a date",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		before, _ := cmd.Flags().GetString("before")

		// Single entry mode
		if len(args) == 1 {
			if before != "" {
				return fmt.Errorf("cannot use --before with an entry ID")
			}

			entryRef := args[0]

			// Get entry by ID or prefix
			entry, err := charmClient.GetEntry(entryRef)
			if err != nil {
				// Try prefix match
				entry, err = charmClient.GetEntryByPrefix(entryRef)
				if err != nil {
					return fmt.Errorf("entry not found: %s", entryRef)
				}
			}

			if entry.Read {
				fmt.Println("Entry is already marked as read")
				return nil
			}

			if err := charmClient.MarkEntryRead(entry.ID); err != nil {
				return fmt.Errorf("failed to mark entry as read: %w", err)
			}

			title := "Untitled"
			if entry.Title != nil {
				title = *entry.Title
			}
			fmt.Printf("Marked as read: %s\n", title)
			return nil
		}

		// Bulk mode requires --before
		if before == "" {
			return fmt.Errorf("provide an entry ID or use --before for bulk marking")
		}

		// Parse the period
		cutoff, ok := timeutil.ParsePeriod(before)
		if !ok {
			// Try parsing as ISO date
			parsed, err := time.Parse("2006-01-02", before)
			if err != nil {
				return fmt.Errorf("invalid period %q: use yesterday, week, month, or YYYY-MM-DD", before)
			}
			cutoff = parsed
		}

		// Mark entries as read
		count, err := charmClient.MarkEntriesReadBefore(cutoff)
		if err != nil {
			return fmt.Errorf("failed to mark entries as read: %w", err)
		}

		if count == 0 {
			fmt.Println("No entries to mark as read")
		} else {
			fmt.Printf("Marked %d entries as read\n", count)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(markReadCmd)

	markReadCmd.Flags().StringP("before", "b", "", "mark entries older than: yesterday, week, month, or YYYY-MM-DD")
}

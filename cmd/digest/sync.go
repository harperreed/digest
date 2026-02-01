// ABOUTME: Sync subcommand for database maintenance
// ABOUTME: Provides compact command for database optimization

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Database maintenance commands",
	Long: `Database maintenance commands for digest.

Commands:
  compact - Strip article content to reduce database size

Examples:
  digest sync compact`,
}

var syncCompactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Strip article content to reduce database size",
	Long: `Remove article content from stored entries to reduce database size.

Content can be re-fetched from original URLs when needed.
This is useful if your database has grown too large from storing full articles.

After compaction, a VACUUM is performed to reclaim disk space.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			return fmt.Errorf("storage not initialized")
		}

		fmt.Println("Compacting database (stripping article content)...")

		// Get all entries
		entries, err := store.ListEntries(nil)
		if err != nil {
			return fmt.Errorf("failed to list entries: %w", err)
		}

		// Count entries with content
		var withContent int
		for _, e := range entries {
			if e.Content != nil && *e.Content != "" {
				withContent++
			}
		}

		if withContent == 0 {
			color.Green("  v Database already compact (no content to strip)")
			return nil
		}

		fmt.Printf("  Found %d entries with content to strip\n", withContent)

		// Strip content and update
		for _, e := range entries {
			if e.Content != nil && *e.Content != "" {
				e.Content = nil
				if err := store.UpdateEntry(e); err != nil {
					color.Yellow("  Warning: failed to update entry %s: %v", e.ID[:8], err)
				}
			}
		}

		color.Green("  v Stripped content from %d entries", withContent)

		// VACUUM to reclaim space
		fmt.Println("  Running VACUUM...")
		if err := store.Compact(); err != nil {
			color.Yellow("  Warning: VACUUM failed: %v", err)
		} else {
			color.Green("  v VACUUM complete")
		}

		color.Green("\nCompaction complete.")
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncCompactCmd)
	rootCmd.AddCommand(syncCmd)
}

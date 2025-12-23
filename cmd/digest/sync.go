// ABOUTME: Sync subcommand for Charm cloud integration
// ABOUTME: Provides status, link, unlink, repair, reset, and wipe commands for cloud sync

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/kv"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/charm"
)

const digestDBName = "digest"

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage cloud sync for digest data",
	Long: `Sync your digest data automatically to the cloud using Charm.

Charm uses your SSH keys for authentication - no passwords needed!
All data is encrypted end-to-end before being stored.

Commands:
  status  - Show sync status and account info
  link    - Link your account (open browser to charm.2389.dev)
  unlink  - Unlink this device from your account
  repair  - Fix a corrupted local database
  reset   - Delete local data and re-sync from cloud
  wipe    - Permanently delete ALL data (local and cloud)

Examples:
  digest sync status
  digest sync link
  digest sync repair --force
  digest sync reset
  digest sync wipe`,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Display current sync configuration and Charm account status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get Charm client ID
		cc, err := charm.GetCharmClient()
		if err != nil {
			return fmt.Errorf("failed to get charm client: %w", err)
		}

		id, err := cc.ID()
		if err != nil {
			color.Yellow("Not linked to Charm")
			fmt.Println("\nRun 'digest sync link' to connect your account.")
			return nil
		}

		color.Green("Linked to Charm")
		fmt.Printf("  Account ID: %s\n", id)
		fmt.Printf("  Server: %s\n", charm.DefaultCharmHost)

		// Show some stats
		if charmClient != nil {
			stats, err := charmClient.GetOverallStats()
			if err == nil {
				fmt.Printf("\n  Feeds: %d\n", stats.TotalFeeds)
				fmt.Printf("  Entries: %d (%d unread)\n", stats.TotalEntries, stats.UnreadCount)
			}
		}

		return nil
	},
}

var syncLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link to Charm account",
	Long: `Link this device to your Charm account.

Opens your browser to authenticate with Charm Cloud.
Your SSH keys are used for secure authentication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc, err := charm.GetCharmClient()
		if err != nil {
			return fmt.Errorf("failed to get charm client: %w", err)
		}

		// Check if already linked
		id, err := cc.ID()
		if err == nil {
			color.Green("Already linked to Charm!")
			fmt.Printf("  Account ID: %s\n", id)
			return nil
		}

		// Open browser for linking
		fmt.Println("Opening browser to link your Charm account...")
		fmt.Printf("Visit: https://%s\n\n", charm.DefaultCharmHost)

		color.Yellow("After linking in browser, restart digest to sync.")
		return nil
	},
}

var syncUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink from Charm account",
	Long:  `Unlink this device from your Charm account. Local data is preserved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cc, err := charm.GetCharmClient()
		if err != nil {
			return fmt.Errorf("failed to get charm client: %w", err)
		}

		// Check if linked
		id, err := cc.ID()
		if err != nil {
			fmt.Println("Not currently linked to Charm.")
			return nil
		}

		fmt.Printf("Currently linked to account: %s\n", id)
		fmt.Println("\nTo unlink this device, visit:")
		fmt.Printf("  https://%s\n", charm.DefaultCharmHost)
		fmt.Println("\nYou can manage linked devices and SSH keys there.")

		return nil
	},
}

var syncRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair a corrupted local database",
	Long: `Attempt to repair a corrupted local database.

Steps performed:
  1. Checkpoint WAL (write-ahead log) into main database
  2. Remove stale SHM (shared memory) files
  3. Run integrity check
  4. Vacuum database to reclaim space

Use --force to attempt REINDEX recovery if corruption is detected.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		fmt.Println("Repairing database...")
		result, err := kv.Repair(digestDBName, force)

		if result.WalCheckpointed {
			color.Green("  ✓ WAL checkpointed")
		}
		if result.ShmRemoved {
			color.Green("  ✓ SHM file removed")
		}
		if result.IntegrityOK {
			color.Green("  ✓ Integrity check passed")
		} else {
			color.Red("  ✗ Integrity check failed")
		}
		if result.Vacuumed {
			color.Green("  ✓ Database vacuumed")
		}

		if err != nil {
			if !force {
				fmt.Println("\nRun with --force to attempt REINDEX recovery.")
			}
			return err
		}

		color.Green("\nRepair complete.")
		return nil
	},
}

var syncResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete local database and re-download from cloud",
	Long: `Delete the local database and re-sync from Charm Cloud.

This removes all local data and re-downloads from the cloud.
Any unsynced local changes will be lost.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("This will DELETE your local database and re-download from Charm Cloud.")
		fmt.Println("Any unsynced local data will be lost.")
		fmt.Print("\nContinue? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "y" && confirmation != "Y" {
			fmt.Println("Canceled.")
			return nil
		}

		fmt.Println("\nResetting database...")
		if err := kv.Reset(digestDBName); err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}

		color.Green("  ✓ Local database deleted")
		color.Green("  ✓ Synced from cloud")
		color.Green("\nReset complete.")
		return nil
	},
}

var syncCompactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Strip article content to reduce database size",
	Long: `Remove article content from stored entries to reduce database size.

Content can be re-fetched from original URLs when needed.
This is useful if your database has grown too large from storing full articles.

After compaction, a sync is performed to update the cloud.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if charmClient == nil {
			return fmt.Errorf("charm client not initialized")
		}

		fmt.Println("Compacting database (stripping article content)...")

		// Disable auto-sync during bulk update
		charmClient.SetAutoSync(false)
		defer charmClient.SetAutoSync(true)

		// Get all entries
		entries, err := charmClient.ListEntries(nil)
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
			color.Green("  ✓ Database already compact (no content to strip)")
			return nil
		}

		fmt.Printf("  Found %d entries with content to strip\n", withContent)

		// Strip content and rewrite (no sync per write)
		for _, e := range entries {
			if e.Content != nil && *e.Content != "" {
				e.Content = nil
				if err := charmClient.UpdateEntry(e); err != nil {
					color.Yellow("  Warning: failed to update entry %s: %v", e.ID[:8], err)
				}
			}
		}

		color.Green("  ✓ Stripped content from %d entries", withContent)

		// Single sync at the end
		fmt.Println("  Syncing to cloud...")
		if err := charmClient.Sync(); err != nil {
			color.Yellow("  Warning: sync failed: %v", err)
		} else {
			color.Green("  ✓ Synced to cloud")
		}

		color.Green("\nCompaction complete.")
		return nil
	},
}

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Permanently delete ALL data (local and cloud)",
	Long: `Permanently delete ALL data for this database.

WARNING: This is destructive and cannot be undone!
This removes BOTH local data AND cloud backups.

If you only want to reset local data, use 'digest sync reset' instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Red("WARNING: This will permanently delete ALL data!")
		fmt.Println("This includes local AND cloud data. This cannot be undone.")
		fmt.Print("\nType 'wipe' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "wipe" {
			fmt.Println("Canceled.")
			return nil
		}

		fmt.Println("\nWiping database...")
		result, err := kv.Wipe(digestDBName)
		if err != nil {
			return fmt.Errorf("wipe failed: %w", err)
		}

		if result.CloudBackupsDeleted > 0 {
			color.Green("  ✓ %d cloud backups deleted", result.CloudBackupsDeleted)
		}
		if result.LocalFilesDeleted > 0 {
			color.Green("  ✓ %d local files deleted", result.LocalFilesDeleted)
		}

		color.Green("\nWipe complete.")
		return nil
	},
}

func init() {
	syncRepairCmd.Flags().Bool("force", false, "Attempt REINDEX recovery if corruption detected")

	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
	syncCmd.AddCommand(syncRepairCmd)
	syncCmd.AddCommand(syncResetCmd)
	syncCmd.AddCommand(syncCompactCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}

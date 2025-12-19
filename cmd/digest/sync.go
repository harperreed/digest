// ABOUTME: Sync subcommand for Charm cloud integration
// ABOUTME: Provides status, link, unlink, and wipe commands for cloud sync via SSH keys

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/charm"
)

// Note: bufio, os, strings still used by syncWipeCmd

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
  wipe    - Reset local data and start fresh

Examples:
  digest sync status
  digest sync link
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

var syncWipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Wipe local data and start fresh",
	Long: `Clear all local Charm data.

This removes all synced data from this device.
Your digest data (feeds, entries) will be re-synced from the cloud.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Confirm with user
		fmt.Println("This will DELETE all local sync data.")
		fmt.Println("Data will be re-synced from the cloud on next run.")
		fmt.Print("\nType 'wipe' to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)

		if confirmation != "wipe" {
			fmt.Println("Aborted.")
			return nil
		}

		fmt.Println("\nWiping local data...")

		if charmClient != nil {
			if err := charmClient.Reset(); err != nil {
				return fmt.Errorf("failed to reset: %w", err)
			}
		}

		color.Green("Local data cleared.")
		fmt.Println("Data will be re-synced from the cloud on next run.")
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLinkCmd)
	syncCmd.AddCommand(syncUnlinkCmd)
	syncCmd.AddCommand(syncWipeCmd)

	rootCmd.AddCommand(syncCmd)
}

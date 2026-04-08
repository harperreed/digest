// ABOUTME: Profile management commands for isolated feed collections
// ABOUTME: Handles listing and removing named profiles

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage feed profiles",
	Long:  "List and manage isolated feed collection profiles",
}

var profileListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all profiles",
	Long:    "List all available feed profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		dataDir := cfg.GetDataDir()
		entries, err := os.ReadDir(dataDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No profiles found.")
				return nil
			}
			return fmt.Errorf("failed to read data directory: %w", err)
		}

		var profiles []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			profiles = append(profiles, entry.Name())
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found.")
			return nil
		}

		fmt.Printf("Found %d profile(s):\n\n", len(profiles))
		for _, name := range profiles {
			if name == profileName {
				fmt.Printf("* %s (active)\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}

		return nil
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile and all its data",
	Long:  "Remove a profile directory and all feeds, entries, and OPML data within it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if strings.EqualFold(name, "default") {
			return fmt.Errorf("cannot remove the default profile")
		}

		if err := config.ValidateProfileName(name); err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		profileDir, err := cfg.ProfileDataDir(name)
		if err != nil {
			return err
		}
		if _, err := os.Stat(profileDir); os.IsNotExist(err) {
			return fmt.Errorf("profile %q does not exist", name)
		}

		// Confirmation prompt
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Printf("This will permanently remove profile %q and all its data.\n", name)
			fmt.Printf("Directory: %s\n", profileDir)
			fmt.Print("Are you sure? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Canceled.")
				return nil
			}
		}

		if err := os.RemoveAll(profileDir); err != nil {
			return fmt.Errorf("failed to remove profile: %w", err)
		}

		fmt.Printf("Removed profile: %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileRemoveCmd)

	profileRemoveCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
}

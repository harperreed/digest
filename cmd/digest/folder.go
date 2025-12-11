// ABOUTME: Folder management commands for organizing feeds into categories
// ABOUTME: Handles folder CRUD operations and syncs changes to OPML file

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Manage feed folders",
	Long:  "Create, list, and manage folders for organizing feeds",
}

var folderAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new folder",
	Long:  "Create a new folder to organize feeds",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Add folder to OPML
		if err := opmlDoc.AddFolder(name); err != nil {
			return fmt.Errorf("failed to add folder: %w", err)
		}

		// Save OPML
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		fmt.Printf("Created folder: %s\n", name)
		return nil
	},
}

var folderListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all folders",
	Long:    "List all folders and the count of feeds in each",
	RunE: func(cmd *cobra.Command, args []string) error {
		folders := opmlDoc.Folders()

		if len(folders) == 0 {
			fmt.Println("No folders found. Create a folder with 'digest folder add <name>'")
			return nil
		}

		fmt.Printf("Found %d folder(s):\n\n", len(folders))
		for _, folder := range folders {
			feeds := opmlDoc.FeedsInFolder(folder)
			fmt.Printf("%s (%d feed(s))\n", folder, len(feeds))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(folderCmd)
	folderCmd.AddCommand(folderAddCmd)
	folderCmd.AddCommand(folderListCmd)
}

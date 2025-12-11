// ABOUTME: Open command for launching entry links in browser
// ABOUTME: Opens the entry's link and marks the entry as read

package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
)

var openCmd = &cobra.Command{
	Use:   "open <entry-prefix>",
	Short: "Open entry link in browser and mark as read",
	Long:  "Open an entry's link in your default browser and mark the entry as read by providing its ID prefix (minimum 6 characters)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get entry by prefix
		entry, err := db.GetEntryByPrefix(dbConn, args[0])
		if err != nil {
			return fmt.Errorf("failed to find entry: %w", err)
		}

		// Check that link is not nil/empty
		if entry.Link == nil || *entry.Link == "" {
			return fmt.Errorf("entry has no link")
		}

		// Open browser
		if err := openBrowser(*entry.Link); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
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
		fmt.Printf("✓ Opened and marked as read: %s\n", title)

		return nil
	},
}

// openBrowser opens a URL in the default browser for the current platform
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Start the browser (don't wait for it to complete)
	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(openCmd)
}

// ABOUTME: Open command for launching entry links in browser
// ABOUTME: Opens the entry's link and marks the entry as read

package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <entry-prefix>",
	Short: "Open entry link in browser and mark as read",
	Long:  "Open an entry's link in your default browser and mark the entry as read by providing its ID prefix (minimum 6 characters)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get entry by prefix
		entry, err := store.GetEntryByPrefix(args[0])
		if err != nil {
			return fmt.Errorf("failed to find entry: %w", err)
		}

		// Check that link is not nil/empty
		if entry.Link == nil || *entry.Link == "" {
			return fmt.Errorf("entry has no link")
		}

		// Validate URL format and scheme for security
		parsedURL, err := url.Parse(*entry.Link)
		if err != nil {
			return fmt.Errorf("entry has malformed link: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("entry link must be http or https, got: %s", parsedURL.Scheme)
		}

		// Open browser with validated URL
		if err := openBrowser(parsedURL.String()); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
		}

		// Mark as read
		if err := store.MarkEntryRead(entry.ID); err != nil {
			return fmt.Errorf("failed to mark entry as read: %w", err)
		}

		// Print confirmation with title
		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("v Opened and marked as read: %s\n", title)

		return nil
	},
}

// openBrowser opens a URL in the default browser for the current platform
func openBrowser(urlStr string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", urlStr)
	case "linux":
		cmd = exec.Command("xdg-open", urlStr)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", urlStr)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Start the browser
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}

	// Reap the process asynchronously to prevent zombie processes
	go cmd.Wait()

	return nil
}

func init() {
	rootCmd.AddCommand(openCmd)
}

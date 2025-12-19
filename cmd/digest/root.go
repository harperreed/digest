// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes Charm KV client

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/charm"
	"github.com/harper/digest/internal/opml"
)

var (
	opmlPath    string
	opmlDoc     *opml.Document
	charmClient *charm.Client
)

var rootCmd = &cobra.Command{
	Use:   "digest",
	Short: "RSS/Atom feed tracker with MCP integration",
	Long: `
██████╗ ██╗ ██████╗ ███████╗███████╗████████╗
██╔══██╗██║██╔════╝ ██╔════╝██╔════╝╚══██╔══╝
██║  ██║██║██║  ███╗█████╗  ███████╗   ██║
██║  ██║██║██║   ██║██╔══╝  ╚════██║   ██║
██████╔╝██║╚██████╔╝███████╗███████║   ██║
╚═════╝ ╚═╝ ╚═════╝ ╚══════╝╚══════╝   ╚═╝

RSS/Atom feed tracker for humans and AI agents.

Track feeds, sync content, and expose via MCP for Claude.
Now with automatic cloud sync via Charm!`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set default OPML path if not provided
		if opmlPath == "" {
			opmlPath = GetDefaultOPMLPath()
		}

		// Initialize Charm client
		var err error
		charmClient, err = charm.InitClient()
		if err != nil {
			return fmt.Errorf("failed to initialize charm: %w", err)
		}

		// Load or create OPML document
		if _, err := os.Stat(opmlPath); os.IsNotExist(err) {
			opmlDoc = opml.NewDocument("digest feeds")
		} else {
			opmlDoc, err = opml.ParseFile(opmlPath)
			if err != nil {
				return fmt.Errorf("failed to load OPML: %w", err)
			}
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if charmClient != nil {
			if err := charmClient.Close(); err != nil {
				return fmt.Errorf("failed to close charm client: %w", err)
			}
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path (default: ~/.local/share/digest/feeds.opml)")
}

func saveOPML() error {
	if opmlDoc == nil {
		return fmt.Errorf("OPML document not initialized")
	}
	if err := opmlDoc.WriteFile(opmlPath); err != nil {
		return fmt.Errorf("failed to write OPML file: %w", err)
	}
	return nil
}

// GetDefaultOPMLPath returns the default OPML file path.
func GetDefaultOPMLPath() string {
	return filepath.Join(getDataDir(), "digest", "feeds.opml")
}

func getDataDir() string {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return dataDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".local", "share")
}

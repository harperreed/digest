// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes storage via config

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/config"
	"github.com/harper/digest/internal/opml"
	"github.com/harper/digest/internal/storage"
)

var (
	opmlPath string
	opmlDoc  *opml.Document
	store    storage.Store
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
Data stored locally. Configure backend via config.json.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip storage init for migrate command (it manages its own storage)
		if cmd.Name() == "migrate" || cmd.Name() == "setup" {
			return nil
		}

		// Set default OPML path if not provided
		if opmlPath == "" {
			opmlPath = GetDefaultOPMLPath()
		}

		// Load config and open storage via configured backend
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		store, err = cfg.OpenStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
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
		if store != nil {
			if err := store.Close(); err != nil {
				return fmt.Errorf("failed to close storage: %w", err)
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

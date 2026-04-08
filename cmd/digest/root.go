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
	opmlPath    string
	profileName string
	opmlDoc     *opml.Document
	store       storage.Store
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
		// Skip storage init for commands that don't need it
		switch cmd.Name() {
		case "setup", "migrate", "version", "help", "completion":
			return nil
		}
		// Profile subcommands don't need storage
		if cmd.Parent() != nil && cmd.Parent().Name() == "profile" {
			return nil
		}

		// Load config and open profile-scoped storage
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Migrate flat-layout data files into "default" profile subdirectory (idempotent)
		if err := cfg.MigrateToProfileLayout(); err != nil {
			return fmt.Errorf("failed to migrate to profile layout: %w", err)
		}

		// Set default OPML path to profile-scoped directory if not explicitly provided
		if opmlPath == "" {
			profileDir, err := cfg.ProfileDataDir(profileName)
			if err != nil {
				return fmt.Errorf("invalid profile: %w", err)
			}
			opmlPath = filepath.Join(profileDir, "feeds.opml")
		}

		store, err = cfg.OpenProfileStorage(profileName)
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
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path (default: <data-dir>/<profile>/feeds.opml)")
	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "default", "profile name (e.g., work, personal). Profiles keep separate sets of feeds. Omit for default profile")
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

// GetDefaultOPMLPath returns the default OPML file path for the default profile.
func GetDefaultOPMLPath() string {
	cfg, err := config.Load()
	if err != nil {
		return filepath.Join(getDataDir(), "digest", "default", "feeds.opml")
	}
	profileDir, err := cfg.ProfileDataDir("default")
	if err != nil {
		return filepath.Join(getDataDir(), "digest", "default", "feeds.opml")
	}
	return filepath.Join(profileDir, "feeds.opml")
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

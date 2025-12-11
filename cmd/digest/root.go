// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes database/OPML

package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/opml"
)

var (
	dbPath   string
	opmlPath string
	dbConn   *sql.DB
	opmlDoc  *opml.Document
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

Track feeds, sync content, and expose via MCP for Claude.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set default paths if not provided
		if dbPath == "" {
			dbPath = db.GetDefaultDBPath()
		}
		if opmlPath == "" {
			opmlPath = db.GetDefaultOPMLPath()
		}

		// Initialize database
		var err error
		dbConn, err = db.InitDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
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
		if dbConn != nil {
			if err := dbConn.Close(); err != nil {
				return fmt.Errorf("failed to close database: %w", err)
			}
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database file path (default: ~/.local/share/digest/digest.db)")
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

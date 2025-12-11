// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes database/OPML

package main

import (
	"github.com/spf13/cobra"
)

var (
	dbPath   string
	opmlPath string
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
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database file path (default: ~/.local/share/digest/digest.db)")
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path (default: ~/.local/share/digest/feeds.opml)")
}

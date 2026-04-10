// ABOUTME: MCP server command for digest CLI
// ABOUTME: Starts stdio-based MCP server for AI agent integration

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI agents",
	Long: `Start the Model Context Protocol (MCP) server on stdio.

This allows AI agents like Claude to interact with your RSS feeds,
query entries, manage subscriptions, and more through structured tools.

The server communicates via JSON-RPC on stdin/stdout.
Supports --profile / -p to set the default profile for the session.
All tools accept an optional "profile" parameter to target a different profile per call.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create MCP server with config and default profile
		server, err := mcp.NewServer(cfg, profileName)
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}
		defer server.Close()

		// Start serving on stdio
		if err := server.ServeStdio(); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

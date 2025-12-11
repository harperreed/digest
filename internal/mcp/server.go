// ABOUTME: MCP server implementation for digest
// ABOUTME: Provides tools, resources, and prompts for AI agents to interact with RSS feeds

package mcp

import (
	"database/sql"

	"github.com/harper/digest/internal/opml"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with digest-specific context
type Server struct {
	mcpServer *server.MCPServer
	db        *sql.DB
	opmlDoc   *opml.Document
	opmlPath  string
}

// NewServer creates a new MCP server instance
func NewServer(db *sql.DB, opmlDoc *opml.Document, opmlPath string) *Server {
	s := &Server{
		db:       db,
		opmlDoc:  opmlDoc,
		opmlPath: opmlPath,
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"digest",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	// Register handlers (stubs for now)
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

// ServeStdio starts the MCP server on stdio
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers MCP tools (stub)
func (s *Server) registerTools() {
	// Will be implemented in next tasks
}

// registerResources registers MCP resources (stub)
func (s *Server) registerResources() {
	// Will be implemented in next tasks
}

// registerPrompts registers MCP prompts (stub)
func (s *Server) registerPrompts() {
	// Will be implemented in next tasks
}

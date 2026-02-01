// ABOUTME: MCP server implementation for digest
// ABOUTME: Provides tools, resources, and prompts for AI agents to interact with RSS feeds

package mcp

import (
	"sync"

	"github.com/harper/digest/internal/opml"
	"github.com/harper/digest/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with digest-specific context
type Server struct {
	mcpServer *server.MCPServer
	store     storage.Store
	opmlDoc   *opml.Document
	opmlPath  string
	opmlMu    sync.RWMutex
}

// NewServer creates a new MCP server instance
func NewServer(store storage.Store, opmlDoc *opml.Document, opmlPath string) *Server {
	s := &Server{
		store:    store,
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

// registerTools is implemented in tools.go
// registerResources is implemented in resources.go
// registerPrompts is implemented in prompts.go

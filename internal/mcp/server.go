// ABOUTME: MCP server implementation for digest
// ABOUTME: Provides tools, resources, and prompts for AI agents to interact with RSS feeds

package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/harper/digest/internal/config"
	"github.com/harper/digest/internal/opml"
	"github.com/harper/digest/internal/storage"
	"github.com/mark3labs/mcp-go/server"
)

// profileContext holds the store, OPML doc, and OPML path for a single profile.
type profileContext struct {
	store    storage.Store
	opmlDoc  *opml.Document
	opmlPath string
	opmlMu   sync.RWMutex
}

// Server wraps the MCP server with digest-specific context.
type Server struct {
	mcpServer      *server.MCPServer
	cfg            *config.Config
	defaultProfile string
	profiles       map[string]*profileContext
	profilesMu     sync.Mutex
}

// NewServer creates a new MCP server instance with a given config and default profile.
// It eagerly loads the default profile to catch configuration errors at startup.
func NewServer(cfg *config.Config, defaultProfile string) (*Server, error) {
	s := &Server{
		cfg:            cfg,
		defaultProfile: defaultProfile,
		profiles:       make(map[string]*profileContext),
	}

	// Eagerly load the default profile to catch errors at startup
	if _, err := s.getProfile(defaultProfile); err != nil {
		return nil, fmt.Errorf("failed to load default profile %q: %w", defaultProfile, err)
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"digest",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	// Register handlers
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s, nil
}

// getProfile returns the profileContext for the named profile.
// If name is empty, the defaultProfile is used.
// Profiles are lazily opened and cached after first access.
func (s *Server) getProfile(name string) (*profileContext, error) {
	if name == "" {
		name = s.defaultProfile
	}

	s.profilesMu.Lock()
	defer s.profilesMu.Unlock()

	if pc, ok := s.profiles[name]; ok {
		return pc, nil
	}

	// Open store for this profile
	store, err := s.cfg.OpenProfileStorage(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage for profile %q: %w", name, err)
	}

	// Get profile data directory
	profileDir, err := s.cfg.ProfileDataDir(name)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to get profile dir for %q: %w", name, err)
	}

	// Load or create OPML document
	opmlPath := filepath.Join(profileDir, "feeds.opml")
	var opmlDoc *opml.Document
	if _, err := os.Stat(opmlPath); os.IsNotExist(err) {
		opmlDoc = opml.NewDocument("digest feeds")
	} else {
		opmlDoc, err = opml.ParseFile(opmlPath)
		if err != nil {
			store.Close()
			return nil, fmt.Errorf("failed to load OPML for profile %q: %w", name, err)
		}
	}

	pc := &profileContext{
		store:    store,
		opmlDoc:  opmlDoc,
		opmlPath: opmlPath,
	}
	s.profiles[name] = pc
	return pc, nil
}

// Close closes all cached profile stores.
func (s *Server) Close() error {
	s.profilesMu.Lock()
	defer s.profilesMu.Unlock()

	var firstErr error
	for name, pc := range s.profiles {
		if err := pc.store.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close store for profile %q: %w", name, err)
		}
	}
	return firstErr
}

// ServeStdio starts the MCP server on stdio
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools is implemented in tools.go
// registerResources is implemented in resources.go
// registerPrompts is implemented in prompts.go

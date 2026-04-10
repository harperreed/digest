# MCP Profile Access Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-tool `profile` parameter to all MCP tools so AI agents can target any profile without restarting the server.

**Architecture:** Replace the single `store`/`opmlDoc`/`opmlPath` in the MCP `Server` struct with a `config.Config` + lazy profile cache. Every tool handler extracts an optional `profile` string from its arguments and resolves it through a `getProfile` helper that caches opened stores and OPML docs. A new `list_profiles` tool lets agents discover available profiles.

**Tech Stack:** Go, mcp-go, Cobra CLI, SQLite storage, OPML

---

## File Structure

| File | Role | Change |
|------|------|--------|
| `internal/mcp/server.go` | Server struct, constructor, profile cache | Rewrite struct, new `NewServer(cfg, defaultProfile)`, add `getProfile`, `Close` |
| `internal/mcp/tools.go` | Tool registration + handlers | Add `profile` property to all 10 tools, add `list_profiles` tool, extract profile in each handler |
| `internal/mcp/resources.go` | Resource handlers | Use `getProfile("")` for default profile |
| `internal/mcp/prompts.go` | Prompt handlers | No store usage — no changes needed |
| `internal/mcp/server_test.go` | Integration tests | Update `testServer` for new constructor, add profile-switching test |
| `cmd/digest/mcp.go` | CLI command wiring | Pass `cfg` + `profileName`, call `Close` |

---

### Task 1: Rewrite Server struct and constructor

**Files:**
- Modify: `internal/mcp/server.go`
- Test: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing test for new constructor**

In `server_test.go`, add a test that constructs a server with `config.Config` and a profile name. The current `testServer` helper uses `NewServer(store, opmlDoc, opmlPath)` — write a new test that calls the new signature.

```go
func TestNewServerWithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config pointing at tmpDir
	cfg := &config.Config{}
	cfg.SetDataDir(tmpDir)
	cfg.SetBackend("sqlite")

	// Create default profile dir with OPML
	defaultDir := filepath.Join(tmpDir, "default")
	require.NoError(t, os.MkdirAll(defaultDir, 0700))
	opmlDoc := opml.NewDocument("test feeds")
	require.NoError(t, opmlDoc.WriteFile(filepath.Join(defaultDir, "feeds.opml")))

	s, err := NewServer(cfg, "default")
	require.NoError(t, err)
	require.NotNil(t, s)
	defer s.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/harper/Public/src/personal/suite/digest && uv run -- go test ./internal/mcp/ -run TestNewServerWithConfig -v`
Expected: FAIL — `NewServer` signature doesn't match.

- [ ] **Step 3: Rewrite server.go**

Replace the entire `Server` struct and constructor. Key changes:

```go
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

// NewServer creates a new MCP server instance.
// It eagerly loads the default profile so startup failures surface immediately.
func NewServer(cfg *config.Config, defaultProfile string) (*Server, error) {
	s := &Server{
		cfg:            cfg,
		defaultProfile: defaultProfile,
		profiles:       make(map[string]*profileContext),
	}

	// Eagerly load default profile
	if _, err := s.getProfile(defaultProfile); err != nil {
		return nil, fmt.Errorf("failed to load default profile %q: %w", defaultProfile, err)
	}

	s.mcpServer = server.NewMCPServer(
		"digest",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s, nil
}

// getProfile returns a cached profileContext, opening it lazily if needed.
// An empty name resolves to the default profile.
func (s *Server) getProfile(name string) (*profileContext, error) {
	if name == "" {
		name = s.defaultProfile
	}

	s.profilesMu.Lock()
	defer s.profilesMu.Unlock()

	if pc, ok := s.profiles[name]; ok {
		return pc, nil
	}

	store, err := s.cfg.OpenProfileStorage(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile %q: %w", name, err)
	}

	profileDir, err := s.cfg.ProfileDataDir(name)
	if err != nil {
		store.Close()
		return nil, err
	}

	opmlPath := filepath.Join(profileDir, "feeds.opml")
	var opmlDoc *opml.Document
	if _, err := os.Stat(opmlPath); os.IsNotExist(err) {
		opmlDoc = opml.NewDocument("digest feeds")
	} else {
		opmlDoc, err = opml.ParseFile(opmlPath)
		if err != nil {
			store.Close()
			return nil, fmt.Errorf("failed to parse OPML for profile %q: %w", name, err)
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
			firstErr = fmt.Errorf("failed to close profile %q: %w", name, err)
		}
	}
	s.profiles = make(map[string]*profileContext)
	return firstErr
}

// ServeStdio starts the MCP server on stdio.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -run TestNewServerWithConfig -v`
Expected: PASS

- [ ] **Step 5: Update testServer helper to use new constructor**

The existing `testServer` helper must change to create a `config.Config` and use the new `NewServer`. This will temporarily break existing tests — that's expected and will be fixed in subsequent steps.

```go
func testServer(t *testing.T) (*Server, storage.Store, string) {
	t.Helper()

	tmpDir := t.TempDir()

	// Create config
	cfg := &config.Config{}
	cfg.SetDataDir(tmpDir)
	cfg.SetBackend("sqlite")

	// Create default profile dir with OPML
	defaultDir := filepath.Join(tmpDir, "default")
	require.NoError(t, os.MkdirAll(defaultDir, 0700))

	// Write initial OPML
	initialOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test Feeds</title></head>
  <body>
    <outline text="Tech" title="Tech">
      <outline text="Example Blog" title="Example Blog" type="rss" xmlUrl="https://example.com/feed.xml"/>
    </outline>
  </body>
</opml>`
	opmlPath := filepath.Join(defaultDir, "feeds.opml")
	require.NoError(t, os.WriteFile(opmlPath, []byte(initialOPML), 0644))

	// Create server
	s, err := NewServer(cfg, "default")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	// Get the store from the profile context for test assertions
	pc, err := s.getProfile("default")
	require.NoError(t, err)

	return s, pc.store, opmlPath
}
```

Remove the standalone `newTestStore` function — it's no longer needed since `NewServer` creates the store internally.

- [ ] **Step 6: Run full test suite to see what breaks**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -v -count=1 2>&1 | head -100`
Expected: Compilation errors in tools.go/resources.go because they still reference `s.store`, `s.opmlDoc`, `s.opmlPath`, `s.opmlMu`. That's fine — Tasks 2-4 fix those.

- [ ] **Step 7: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "refactor(mcp): replace single store with profile-aware server constructor"
```

---

### Task 2: Add profile parameter extraction helper

**Files:**
- Modify: `internal/mcp/tools.go`

- [ ] **Step 1: Add extractProfile helper function**

Add a helper at the top of the handlers section in `tools.go` that extracts the `profile` string from a `CallToolRequest`:

```go
// extractProfile returns the profile name from the request arguments,
// or empty string if not specified (which resolves to default).
func extractProfile(req mcp.CallToolRequest) string {
	args := req.GetArguments()
	if args == nil {
		return ""
	}
	if p, ok := args["profile"]; ok {
		if s, ok := p.(string); ok {
			return s
		}
	}
	return ""
}
```

- [ ] **Step 2: Add the profile property definition**

Add a reusable property map for the `profile` parameter that every tool will include:

```go
// profileProperty is the shared schema for the optional profile parameter on all tools.
var profileProperty = map[string]interface{}{
	"type":        "string",
	"description": "Target profile name. Defaults to the server's startup profile if omitted.",
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/mcp/tools.go
git commit -m "feat(mcp): add profile parameter extraction helper"
```

---

### Task 3: Wire profile into all tool registrations and handlers

**Files:**
- Modify: `internal/mcp/tools.go`

This is the bulk of the work. Every tool registration adds the `profile` property, and every handler calls `extractProfile` + `s.getProfile` instead of using `s.store`/`s.opmlDoc`/`s.opmlPath`/`s.opmlMu` directly.

- [ ] **Step 1: Update registerListFeedsTool and handleListFeeds**

Add `"profile": profileProperty` to the tool's Properties map.

Replace the handler to use profile context:

```go
func (s *Server) handleListFeeds(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pc, err := s.getProfile(extractProfile(req))
	if err != nil {
		return nil, err
	}

	// Get all feeds from OPML
	pc.opmlMu.RLock()
	opmlFeeds := pc.opmlDoc.AllFeeds()
	folders := pc.opmlDoc.Folders()
	pc.opmlMu.RUnlock()

	// Get all feeds from storage
	storedFeeds, err := pc.store.ListFeeds()
	if err != nil {
		return nil, fmt.Errorf("failed to list feeds: %w", err)
	}

	// ... rest of handler unchanged, just uses pc.store instead of s.store
```

- [ ] **Step 2: Update registerAddFeedTool and handleAddFeed**

Add `"profile": profileProperty` to Properties. Update handler:

```go
func (s *Server) handleAddFeed(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pc, err := s.getProfile(extractProfile(req))
	if err != nil {
		return nil, err
	}

	// ... replace all s.store → pc.store
	// ... replace all s.opmlMu → pc.opmlMu
	// ... replace all s.opmlDoc → pc.opmlDoc
	// ... replace all s.opmlPath → pc.opmlPath
```

- [ ] **Step 3: Update registerRemoveFeedTool and handleRemoveFeed**

Same pattern: add profile property, replace `s.store` → `pc.store`, `s.opmlMu` → `pc.opmlMu`, `s.opmlDoc` → `pc.opmlDoc`, `s.opmlPath` → `pc.opmlPath`.

- [ ] **Step 4: Update registerMoveFeedTool and handleMoveFeed**

Same pattern.

- [ ] **Step 5: Update registerSyncFeedsTool and handleSyncFeeds**

Add profile property. Replace `s.store` → `pc.store` in the handler. The `syncFeed` helper also uses `s.store` — it needs to accept a `storage.Store` parameter instead:

```go
func (s *Server) syncFeed(ctx context.Context, store storage.Store, feed *models.Feed, force bool) (int, bool, error) {
	// ... replace all s.store → store
```

And the caller passes `pc.store`:

```go
newCount, wasCached, err := s.syncFeed(ctx, pc.store, feed, force)
```

- [ ] **Step 6: Update registerListEntriesTool and handleListEntries**

Add profile property. Replace `s.store` → `pc.store`.

- [ ] **Step 7: Update registerGetEntryTool and handleGetEntry**

Add profile property. Replace `s.store` → `pc.store`.

- [ ] **Step 8: Update registerMarkReadTool and handleMarkRead**

Add profile property. Replace `s.store` → `pc.store`.

- [ ] **Step 9: Update registerMarkUnreadTool and handleMarkUnread**

Add profile property. Replace `s.store` → `pc.store`.

- [ ] **Step 10: Update registerBulkMarkReadTool and handleBulkMarkRead**

Add profile property. Replace `s.store` → `pc.store`.

- [ ] **Step 11: Compile check**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go build ./internal/mcp/`
Expected: may still fail if resources.go references old fields — that's Task 4.

- [ ] **Step 12: Commit**

```bash
git add internal/mcp/tools.go
git commit -m "feat(mcp): add profile parameter to all MCP tool handlers"
```

---

### Task 4: Wire profile into resources

**Files:**
- Modify: `internal/mcp/resources.go`

Resources use the default profile (MCP resource URIs don't carry per-request parameters).

- [ ] **Step 1: Update all resource handlers**

Replace every `s.store` reference with profile context:

```go
func (s *Server) registerFeedsResource() {
	s.mcpServer.AddResource(
		mcp.Resource{
			URI:         "digest://feeds",
			Name:        "All Feeds",
			Description: "List all subscribed RSS/Atom feeds with metadata including title, URL, last fetch time, and error status",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			pc, err := s.getProfile("")
			if err != nil {
				return nil, fmt.Errorf("failed to get profile: %w", err)
			}

			feeds, err := pc.store.ListFeeds()
			// ... rest unchanged, just uses pc.store
```

Apply the same pattern to `registerEntriesUnreadResource`, `registerEntriesTodayResource`, `registerStatsResource`, and `calculateStats`.

For `calculateStats`, change it to accept a `storage.Store` parameter:

```go
func (s *Server) calculateStats(store storage.Store) (*StatsData, error) {
	overallStats, err := store.GetOverallStats()
	// ... replace all s.store → store
```

And the caller passes `pc.store`:

```go
stats, err := s.calculateStats(pc.store)
```

- [ ] **Step 2: Compile check**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go build ./internal/mcp/`
Expected: PASS — no more references to `s.store`, `s.opmlDoc`, `s.opmlPath`, `s.opmlMu`.

- [ ] **Step 3: Run existing tests**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -v -count=1 2>&1 | tail -30`
Expected: All existing tests PASS with the updated `testServer` helper.

- [ ] **Step 4: Commit**

```bash
git add internal/mcp/resources.go
git commit -m "feat(mcp): use profile context in MCP resource handlers"
```

---

### Task 5: Add list_profiles tool

**Files:**
- Modify: `internal/mcp/tools.go`
- Test: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestListProfiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	cfg.SetDataDir(tmpDir)
	cfg.SetBackend("sqlite")

	// Create multiple profile directories
	for _, name := range []string{"default", "work", "personal"} {
		dir := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(dir, 0700))
		doc := opml.NewDocument("test")
		require.NoError(t, doc.WriteFile(filepath.Join(dir, "feeds.opml")))
	}
	// Create a non-profile file to ensure it's excluded
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644))

	s, err := NewServer(cfg, "default")
	require.NoError(t, err)
	defer s.Close()

	result, err := s.handleListProfiles(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Parse output
	var output ListProfilesOutput
	text := result.Content[0].(mcp.TextContent).Text
	require.NoError(t, json.Unmarshal([]byte(text), &output))

	require.Equal(t, 3, output.Count)
	require.Equal(t, "default", output.Default)

	names := make([]string, len(output.Profiles))
	for i, p := range output.Profiles {
		names[i] = p.Name
	}
	require.Contains(t, names, "default")
	require.Contains(t, names, "work")
	require.Contains(t, names, "personal")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -run TestListProfiles -v`
Expected: FAIL — `handleListProfiles` doesn't exist.

- [ ] **Step 3: Implement list_profiles tool**

Add types and handler in `tools.go`:

```go
type ProfileInfo struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type ListProfilesOutput struct {
	Profiles []ProfileInfo `json:"profiles"`
	Count    int           `json:"count"`
	Default  string        `json:"default"`
}
```

Add registration in `registerTools()`:

```go
s.registerListProfilesTool()
```

Add the registration and handler:

```go
func (s *Server) registerListProfilesTool() {
	tool := mcp.Tool{
		Name:        "list_profiles",
		Description: "List available feed profiles. Profiles are isolated collections of feeds and entries. Returns profile names and which is the current default.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
	s.mcpServer.AddTool(tool, s.handleListProfiles)
}

func (s *Server) handleListProfiles(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dataDir := s.cfg.GetDataDir()
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	profiles := make([]ProfileInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := config.ValidateProfileName(name); err != nil {
			continue
		}
		profiles = append(profiles, ProfileInfo{
			Name:      name,
			IsDefault: name == s.defaultProfile,
		})
	}

	output := ListProfilesOutput{
		Profiles: profiles,
		Count:    len(profiles),
		Default:  s.defaultProfile,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -run TestListProfiles -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/tools.go internal/mcp/server_test.go
git commit -m "feat(mcp): add list_profiles tool for profile discovery"
```

---

### Task 6: Write cross-profile integration test

**Files:**
- Test: `internal/mcp/server_test.go`

- [ ] **Step 1: Write test that uses profile parameter across tool calls**

```go
func TestProfileParameterIsolation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	cfg.SetDataDir(tmpDir)
	cfg.SetBackend("sqlite")

	// Create two profiles
	for _, name := range []string{"default", "work"} {
		dir := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(dir, 0700))
		doc := opml.NewDocument("test")
		require.NoError(t, doc.WriteFile(filepath.Join(dir, "feeds.opml")))
	}

	s, err := NewServer(cfg, "default")
	require.NoError(t, err)
	defer s.Close()

	// Add a feed to the "work" profile
	addReq := mcp.CallToolRequest{}
	addReq.Params.Name = "add_feed"
	addReq.Params.Arguments = map[string]interface{}{
		"url":     "https://work.example.com/feed.xml",
		"title":   "Work Feed",
		"profile": "work",
	}

	// We need a test HTTP server for the feed
	// Actually, add_feed doesn't fetch — it just creates in DB + OPML
	result, err := s.handleAddFeed(context.Background(), addReq)
	require.NoError(t, err)
	require.NotNil(t, result)

	// List feeds on default profile — should be empty (no feeds added there)
	listDefault := mcp.CallToolRequest{}
	listDefault.Params.Name = "list_feeds"
	listDefault.Params.Arguments = map[string]interface{}{}

	defaultResult, err := s.handleListFeeds(context.Background(), listDefault)
	require.NoError(t, err)
	var defaultOutput ListFeedsOutput
	text := defaultResult.Content[0].(mcp.TextContent).Text
	require.NoError(t, json.Unmarshal([]byte(text), &defaultOutput))
	require.Equal(t, 0, defaultOutput.Count, "default profile should have no feeds")

	// List feeds on work profile — should have the feed we added
	listWork := mcp.CallToolRequest{}
	listWork.Params.Name = "list_feeds"
	listWork.Params.Arguments = map[string]interface{}{
		"profile": "work",
	}

	workResult, err := s.handleListFeeds(context.Background(), listWork)
	require.NoError(t, err)
	var workOutput ListFeedsOutput
	text = workResult.Content[0].(mcp.TextContent).Text
	require.NoError(t, json.Unmarshal([]byte(text), &workOutput))
	require.Equal(t, 1, workOutput.Count, "work profile should have one feed")
	require.Equal(t, "https://work.example.com/feed.xml", workOutput.Feeds[0].URL)
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -run TestProfileParameterIsolation -v`
Expected: PASS

- [ ] **Step 3: Write test for invalid profile name**

```go
func TestInvalidProfileNameReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	cfg.SetDataDir(tmpDir)
	cfg.SetBackend("sqlite")

	defaultDir := filepath.Join(tmpDir, "default")
	require.NoError(t, os.MkdirAll(defaultDir, 0700))
	doc := opml.NewDocument("test")
	require.NoError(t, doc.WriteFile(filepath.Join(defaultDir, "feeds.opml")))

	s, err := NewServer(cfg, "default")
	require.NoError(t, err)
	defer s.Close()

	// Try to list feeds with path traversal profile
	listReq := mcp.CallToolRequest{}
	listReq.Params.Name = "list_feeds"
	listReq.Params.Arguments = map[string]interface{}{
		"profile": "../../../etc",
	}

	_, err = s.handleListFeeds(context.Background(), listReq)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid profile name")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/mcp/ -run TestInvalidProfileNameReturnsError -v`
Expected: PASS (validation comes from `config.ValidateProfileName` called by `OpenProfileStorage`)

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server_test.go
git commit -m "test(mcp): add cross-profile isolation and validation tests"
```

---

### Task 7: Update CLI wiring

**Files:**
- Modify: `cmd/digest/mcp.go`

- [ ] **Step 1: Update mcp.go to use new constructor**

```go
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
		server, err := mcp.NewServer(cfg, profileName)
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}
		defer server.Close()

		if err := server.ServeStdio(); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
```

Note: The `mcpCmd` no longer uses the global `store`, `opmlDoc`, or `opmlPath` variables — it creates its own via the config. The global `store` from `PersistentPreRunE` is still used by other commands but not by MCP.

- [ ] **Step 2: Verify the build compiles**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go build ./cmd/digest/`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... 2>&1 | tail -20`
Expected: All packages PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/digest/mcp.go
git commit -m "feat(mcp): wire profile-aware server into CLI mcp command"
```

---

### Task 8: Final verification and cleanup

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... -count=1`
Expected: All 13+ packages PASS.

- [ ] **Step 2: Run linter**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go vet ./...`
Expected: No issues.

- [ ] **Step 3: Verify MCP server starts**

Run: `cd /Users/harper/Public/src/personal/suite/digest && echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | go run ./cmd/digest mcp 2>/dev/null | head -1`
Expected: JSON response with server capabilities including all tools.

- [ ] **Step 4: Verify list_profiles tool in output**

Check that the initialize response includes `list_profiles` in the tools list.

- [ ] **Step 5: Commit any cleanup**

```bash
git add -A
git commit -m "chore(mcp): final verification of profile-aware MCP server"
```

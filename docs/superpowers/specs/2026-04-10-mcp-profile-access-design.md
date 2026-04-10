# MCP Profile Access Design

Per-tool profile parameter for MCP tools, allowing AI agents to target any profile without restarting the server.

## Problem

The MCP server takes a single `storage.Store` at startup, locked to whatever `--profile` was passed on the CLI. An MCP client cannot discover, switch, or target different profiles.

## Design

### Server struct

Replace the single store/OPML fields with a profile-aware cache:

```go
type profileContext struct {
    store    storage.Store
    opmlDoc  *opml.Document
    opmlPath string
    opmlMu   sync.RWMutex
}

type Server struct {
    mcpServer      *server.MCPServer
    cfg            *config.Config
    defaultProfile string
    profiles       map[string]*profileContext
    profilesMu     sync.Mutex
}
```

### NewServer signature

```go
func NewServer(cfg *config.Config, defaultProfile string) (*Server, error)
```

Eagerly loads the default profile context on construction so startup failures surface immediately. Returns error (currently returns bare `*Server`).

### Profile resolution

Private helper `getProfile(name string) (*profileContext, error)`:
- If name is empty, use `defaultProfile`
- Check cache; return if found
- Validate via `config.ValidateProfileName`
- Open store via `cfg.OpenProfileStorage(name)`
- Load or create OPML from `<profileDir>/feeds.opml`
- Cache and return

All tool/resource handlers call `getProfile` instead of accessing `s.store` / `s.opmlDoc` directly.

### Per-tool profile parameter

Every existing tool (`list_feeds`, `add_feed`, `remove_feed`, `move_feed`, `sync_feeds`, `list_entries`, `get_entry`, `mark_read`, `mark_unread`, `bulk_mark_read`) gains an optional `profile` string parameter:

```json
{
  "profile": {
    "type": "string",
    "description": "Target profile name. Defaults to the server's startup profile if omitted."
  }
}
```

Each handler extracts `profile` from the request arguments, passes it to `getProfile`, and operates on the returned context.

### New tool: list_profiles

```json
{
  "name": "list_profiles",
  "description": "List available profiles. Returns profile names and which is the current default."
}
```

Implementation: scan `cfg.GetDataDir()` for subdirectories that pass `ValidateProfileName`. Return list with a flag indicating which is the default.

### Resources

Resources (`digest://feeds`, `digest://entries/unread`, `digest://entries/today`, `digest://stats`) operate on the default profile. MCP resource URIs don't carry per-request parameters, so profile-scoped resources would require URI templates (`digest://profiles/{name}/feeds`). This is deferred — tools are sufficient for cross-profile access.

### cmd/digest/mcp.go

Change from:
```go
server := mcp.NewServer(store, opmlDoc, opmlPath)
```
To:
```go
server, err := mcp.NewServer(cfg, profileName)
```

The `PersistentPreRunE` in root.go still opens the default profile's store for other commands. The MCP command passes `cfg` and `profileName` through and lets the MCP server manage its own store lifecycle.

### Cleanup

`Server.Close()` iterates `s.profiles` and closes every cached store. Called from `mcp.go` after `ServeStdio` returns.

## Files changed

| File | Change |
|------|--------|
| `internal/mcp/server.go` | New struct, `NewServer` signature, `getProfile`, `Close` |
| `internal/mcp/tools.go` | Add `profile` param to all tools, extract in handlers |
| `internal/mcp/resources.go` | Use `getProfile(defaultProfile)` instead of `s.store` |
| `internal/mcp/prompts.go` | Same pattern if prompts use store |
| `internal/mcp/server_test.go` | Update for new constructor |
| `internal/mcp/helpers_test.go` | Update test helpers |
| `cmd/digest/mcp.go` | Pass `cfg` + `profileName`, call `Close` |

## Not in scope

- Profile CRUD via MCP (create/delete profiles) — use CLI for that
- Profile-scoped resource URIs — deferred
- Per-profile OPML customization beyond file location

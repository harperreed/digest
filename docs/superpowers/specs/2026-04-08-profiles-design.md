# Profiles: Isolated Feed Collections

## Problem

All feeds live in a single global collection. There's no way to separate work feeds from personal feeds from security feeds. Users want to run `digest --profile work list` and only see work-related entries.

## Decision

Profiles are **directory-per-profile** isolation. Each profile gets its own subdirectory under the data dir containing its own `digest.db` and `feeds.opml`. No shared state between profiles.

## Data Layout

```
~/.local/share/digest/
├── default/
│   ├── digest.db
│   └── feeds.opml
├── work/
│   ├── digest.db
│   └── feeds.opml
└── security/
    ├── digest.db
    └── feeds.opml
```

### Profile Discovery

A profile exists if its directory exists under the data dir. No registry file — the filesystem is the source of truth. `os.ReadDir` on the data dir yields the profile list.

### Auto-Create

When `--profile <name>` references a profile that doesn't exist yet, create the directory and let normal storage initialization proceed. No explicit `profile create` command.

### Migration (Old Layout → New Layout)

On startup, if `digest.db` or `feeds.opml` exist at the data dir root (flat/legacy layout), move them into a `default/` subdirectory. This is a one-time, idempotent migration using `os.Rename`.

Detection: check for `<data_dir>/digest.db` or `<data_dir>/feeds.opml` existing as files (not directories).

## CLI Interface

### Global Flag

```
digest --profile <name> <command> [flags]
```

`--profile` is a persistent flag on the root command. Omitting it is equivalent to `--profile default`.

### Examples

```bash
digest --profile work feed add https://example.com/feed.xml
digest --profile work fetch
digest --profile work list --unread
digest --profile security list --today
digest list                             # uses default profile
digest profile list                     # shows: default, work, security
digest profile delete work              # removes ~/.local/share/digest/work/ (with confirmation)
```

### Profile Management Commands

| Command | Behavior |
|---------|----------|
| `digest profile list` | List profile names (directories in data dir) |
| `digest profile delete <name>` | Remove profile directory after confirmation prompt. Refuses to delete `default`. |

No `profile create` (auto-created on first use). No `profile use` or "current profile" switching.

### MCP Server

`digest --profile work mcp` starts the MCP server scoped to the `work` profile. The MCP tools and resources operate on that profile's data only. No cross-profile MCP operations.

## Code Changes

### Files Modified

1. **`internal/config/config.go`**
   - Add `ProfileDataDir(profile string) string` returning `<data_dir>/<profile>/`
   - Existing `DataDir()` behavior remains for backwards compatibility

2. **`cmd/digest/root.go`**
   - Add `--profile` persistent flag (default: `"default"`)
   - In `PersistentPreRunE`: resolve profile name → profile data dir → pass to storage init
   - Migration logic: detect flat layout, move files into `default/`

### Files Added

3. **`cmd/digest/profile.go`**
   - `profile list`: read directories from data dir, print names
   - `profile delete <name>`: confirm and remove directory
   - Estimated ~80 lines

### Files NOT Modified

- `internal/storage/*` — receives a path, unchanged
- `internal/fetch/*` — unchanged
- `internal/parse/*` — unchanged
- `internal/mcp/*` — unchanged (profile resolved before server starts)
- All existing commands — unchanged (operate on whatever `Store` they receive)

## Testing

| Test | Type | What It Verifies |
|------|------|------------------|
| `ProfileDataDir` returns correct path | Unit | Path construction `<data_dir>/<profile>/` |
| Migration moves flat files into `default/` | Integration | `digest.db` at root → `default/digest.db` |
| Migration is idempotent | Integration | Running twice doesn't break anything |
| Auto-create on first use | Integration | `--profile new` creates directory and initializes storage |
| Profile isolation | E2E | Feed added to profile A is not visible in profile B |
| `profile list` shows all profiles | Integration | Reads directory names correctly |
| `profile delete` removes directory | Integration | Directory gone after confirmation |
| `profile delete default` is refused | Unit | Error returned |
| No `--profile` flag uses default | Integration | Commands without flag operate on `default/` |

## Out of Scope

- Cross-profile operations (merge, copy, move feeds between profiles)
- Profile-specific configuration (different backends per profile)
- "Current profile" switching state
- Profile aliases or renaming

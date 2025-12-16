# Vault Sync Integration Design

Integrate suite-sync vault library for E2E encrypted sync of feeds and read state across devices.

## Goals

- Multi-device subscriptions: Same feed list syncs across devices
- Shared reading lists: Multiple people can share a feed collection
- Backup & restore: Recovery phrase restores subscriptions on new machine
- Read state sync: Mark article read on one device, it's read everywhere

## Entities

### Feed Entity

| Field | Type | Description |
|-------|------|-------------|
| Entity ID | string | Feed URL (normalized) |
| `url` | string | Feed URL (primary identifier) |
| `title` | string | Display name |
| `folder` | string | Folder name (empty = root) |
| `created_at` | int64 | Unix timestamp |

### Read State Entity

| Field | Type | Description |
|-------|------|-------------|
| Entity ID | string | `{feed_url}:{guid}` |
| `feed_url` | string | Which feed |
| `guid` | string | Article identifier from RSS |
| `read` | bool | true = read, false = unread |
| `read_at` | int64 | When marked (for LWW resolution) |

## Architecture

### File Structure

```
internal/sync/
├── config.go     # Config struct, load/save, env overrides
└── sync.go       # Syncer with feed/read_state handlers

cmd/digest/
├── fetch.go      # RSS fetch (renamed from sync.go)
├── sync.go       # Vault sync commands (new)
├── feed.go       # Wire sync into feed add/remove
└── markread.go   # Wire sync into mark read/unread
```

### Storage

```
┌─────────────────┐     ┌─────────────────┐
│   digest.db     │     │   vault.db      │
│                 │     │                 │
│  feeds table    │────▶│  outbox table   │
│  entries table  │     │  applied table  │
│                 │     │  sync_state     │
└─────────────────┘     └─────────────────┘
```

- SQLite `feeds` table is source of truth
- OPML remains import/export format
- Config at `~/.config/digest/sync.json`
- Vault DB at `~/.config/digest/vault.db`

### Database Migration

```sql
ALTER TABLE feeds ADD COLUMN folder TEXT DEFAULT '';
```

## CLI Commands

### Renamed

```bash
digest fetch           # Fetch entries from all feeds (was: sync)
digest fetch <url>     # Fetch from specific feed
digest fetch --force   # Ignore cache headers
```

### New Sync Commands

```bash
digest sync init       # Generate device ID, create config
digest sync login      # Auth with server + recovery phrase
digest sync logout     # Clear credentials
digest sync status     # Show sync state (pending, last sync)
digest sync now        # Manual push/pull
digest sync wipe       # Emergency reset (clear server data)
```

## Data Flow

### Outgoing (local → server)

| User Action | Queue | Auto-sync |
|-------------|-------|-----------|
| `feed add` | QueueFeedChange(OpUpsert) | Sync() |
| `feed remove` | QueueFeedChange(OpDelete) | Sync() |
| `feed move` | QueueFeedChange(OpUpsert) | Sync() |
| `read <id>` | QueueReadState(read=true) | Sync() |
| `read -u <id>` | QueueReadState(read=false) | Sync() |

### Incoming (server → local)

- Feed upsert: INSERT OR REPLACE by URL, update OPML
- Feed delete: DELETE by URL, update OPML
- Read state: Find entry by (feed_url, guid), apply if timestamp newer

## Conflict Resolution

- **Feeds**: Last write wins (vault handles ordering)
- **Read state**: Compare `read_at` timestamps, newer wins
- **Missing article**: Skip read state (article appears on next RSS fetch)

## Error Handling

### Graceful Degradation

- Sync not configured → local operations work, no error
- Sync fails → log warning, local operation succeeds
- Server unreachable → queue changes, sync later

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| Read state for unknown article | Skip |
| Feed deleted, article read elsewhere | Read state orphaned (harmless) |
| Same feed added on two devices | Both succeed (same URL = same entity) |
| Folder renamed | Update each feed's folder field |
| Recovery phrase mismatch | Decrypt fails, offer wipe-and-resync |

## Implementation Reference

See `../position/internal/sync/` for working implementation of this pattern.

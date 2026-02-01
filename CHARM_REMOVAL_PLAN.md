# Digest - Charm Removal Plan

## Charmbracelet Dependencies

**REMOVE:**
- `github.com/charmbracelet/charm` (2389-research fork) - KV storage
- `github.com/charmbracelet/glamour v0.10.0` - Markdown rendering

## Usage Analysis

### Charm KV Usage

| File | Purpose |
|------|---------|
| `internal/charm/client.go` | ALL feed/entry CRUD operations |
| `cmd/digest/sync.go` | Sync commands: repair, reset, wipe, compact |
| `internal/charm/wal_test.go` | WAL tests |

### Glamour Usage

| File | Usage |
|------|-------|
| `cmd/digest/read.go` | `glamour.Render(markdown, "dark")` with fallback to plain text |

Only ONE file uses glamour, fallback already exists.

## Removal Strategy

### Phase 1: Replace Glamour

**Options:**
1. Plain text output (simplest - fallback already exists)
2. goldmark with ANSI renderer (already indirect dep)
3. Basic styling with fatih/color (already in project)

**Recommendation:** Plain text. The HTML→Markdown conversion already happens.

### Phase 2: Replace Charm KV with SQLite

**New:** `internal/storage/sqlite.go`

**Schema:**
```sql
CREATE TABLE feeds (
    id TEXT PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    folder TEXT DEFAULT '',
    etag TEXT,
    last_modified TEXT,
    last_fetched_at TIMESTAMP,
    last_error TEXT,
    error_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE entries (
    id TEXT PRIMARY KEY,
    feed_id TEXT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NOT NULL,
    title TEXT,
    link TEXT,
    author TEXT,
    published_at TIMESTAMP,
    content TEXT,
    read INTEGER DEFAULT 0,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    UNIQUE(feed_id, guid)
);

CREATE INDEX idx_entries_feed_id ON entries(feed_id);
CREATE INDEX idx_entries_read ON entries(read);
CREATE INDEX idx_entries_published_at ON entries(published_at);

-- FTS5 for content search
CREATE VIRTUAL TABLE entries_fts USING fts5(
    title,
    content,
    content=entries,
    content_rowid=rowid
);

-- Note: entries table needs rowid INTEGER PRIMARY KEY for FTS5
-- Add triggers to keep FTS in sync (see chronicle plan for pattern)
```

### Phase 3: Update Export Commands

**Existing:** OPML export already works.

**Add:**
```bash
digest export feeds --format yaml
digest export entries --format yaml
digest export entries --format markdown
digest export all --format yaml > digest-backup.yaml
```

**YAML Format:**
```yaml
version: "1.0"
exported_at: "2026-01-31T12:00:00Z"
tool: "digest"

feeds:
  - id: "abc123..."
    url: "https://example.com/feed.xml"
    title: "Example Blog"
    folder: "Tech"

entries:
  - id: "def456..."
    feed_id: "abc123..."
    title: "Article Title"
    content: |
      Article content...
```

**Markdown Format:**
```markdown
# Feed Entries Export

## Example Blog

### Article Title
- **Author:** John Doe
- **Published:** December 15, 2024
- **Link:** https://example.com/post/1

Article content...
```

### Phase 4: Remove Sync Commands

**Keep:** `sync compact` (reimplement for SQLite)

**Remove:** `sync status`, `sync link`, `sync unlink`, `sync repair`, `sync reset`, `sync wipe`

## Files to Modify

### DELETE:
- `internal/charm/client.go`
- `internal/charm/client_test.go`
- `internal/charm/wal_test.go`

### CREATE:
- `internal/storage/storage.go` - Interface
- `internal/storage/sqlite.go` - Implementation
- `internal/storage/sqlite_test.go`
- `internal/storage/migrations.go`

### MODIFY:
- `go.mod` - Remove charm/glamour
- `cmd/digest/root.go` - Use SQLite init
- `cmd/digest/sync.go` - Remove charm commands
- `cmd/digest/read.go` - Remove glamour, use plain text
- `cmd/digest/feed.go` - Update client type
- `cmd/digest/fetch.go` - Update client type
- `cmd/digest/list.go` - Update client type
- `cmd/digest/markread.go` - Update client type
- `cmd/digest/markunread.go` - Update client type
- `cmd/digest/open.go` - Update client type
- `cmd/digest/export.go` - Add YAML/Markdown formats
- `internal/mcp/server.go` - Update client type
- `internal/mcp/tools.go` - Update references
- `internal/mcp/resources.go` - Update references

## Data Path

| Current | New |
|---------|-----|
| `~/.charm/kv/digest/` | `~/.local/share/digest/digest.db` |

## Implementation Order

1. Create storage interface
2. Implement SQLite storage
3. Create migration tool (if users have Charm data)
4. Update root command to use SQLite
5. Update all commands
6. Remove glamour from read.go
7. Add YAML/Markdown export
8. Remove/update sync commands
9. Delete `internal/charm/`
10. Update go.mod
11. Update tests

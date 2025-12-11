# digest - RSS Feed Tracker Design

## Overview

**digest** is a CLI tool for tracking RSS/Atom feeds with MCP integration for AI agents. It serves dual purposes: personal feed reading and providing agents with access to current news/content.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐
│   feeds.opml    │────▶│   SQLite DB      │
│  (subscriptions │     │  (entries, state,│
│   + folders)    │     │   ETag cache)    │
└─────────────────┘     └──────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────────────────────────────┐
│              digest CLI                  │
│  (Cobra commands, Go, single binary)    │
└─────────────────────────────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌──────────────────┐
│   Human (CLI)   │     │   MCP Server     │
│   digest sync   │     │  (tools/resources│
│   digest read   │     │   for agents)    │
└─────────────────┘     └──────────────────┘
```

**Key principle:** OPML is the source of truth for *what* you're subscribed to. SQLite stores *content* that's been fetched, plus read state and HTTP caching metadata.

**File locations (XDG standard):**
- `~/.local/share/digest/feeds.opml` - subscriptions
- `~/.local/share/digest/digest.db` - SQLite database

## Data Model

### OPML Structure (feeds.opml)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>digest feeds</title>
  </head>
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss"/>
      <outline type="rss" text="Lobsters" xmlUrl="https://lobste.rs/rss"/>
    </outline>
    <outline text="Fun" title="Fun">
      <outline type="rss" text="XKCD" xmlUrl="https://xkcd.com/rss.xml"/>
    </outline>
    <outline type="rss" text="Uncategorized Feed" xmlUrl="https://example.com/feed"/>
  </body>
</opml>
```

### SQLite Schema

```sql
-- Tracks each feed's sync state (ETags, last fetch)
feeds (
  id TEXT PRIMARY KEY,        -- UUID
  url TEXT UNIQUE NOT NULL,   -- feed URL (matches OPML xmlUrl)
  title TEXT,                 -- cached title from feed
  etag TEXT,                  -- HTTP ETag for caching
  last_modified TEXT,         -- HTTP Last-Modified header
  last_fetched_at DATETIME,
  last_error TEXT,            -- last error message if any
  error_count INTEGER DEFAULT 0,
  created_at DATETIME
)

-- Individual articles/entries
entries (
  id TEXT PRIMARY KEY,        -- UUID
  feed_id TEXT NOT NULL,      -- FK to feeds
  guid TEXT NOT NULL,         -- entry's unique ID from feed
  title TEXT,
  link TEXT,
  author TEXT,
  published_at DATETIME,
  content TEXT,               -- full content or summary
  read BOOLEAN DEFAULT FALSE,
  read_at DATETIME,
  created_at DATETIME,
  UNIQUE(feed_id, guid)       -- no duplicate entries per feed
)

-- Tags that came from the feed itself (categories)
entry_tags (
  entry_id TEXT,
  tag TEXT,
  PRIMARY KEY (entry_id, tag)
)
```

## CLI Commands

```
digest (root)
├── feed                      # Feed management
│   ├── add <url> [--folder]  # Add feed, update OPML
│   ├── remove <url|prefix>   # Remove feed from OPML
│   ├── list / ls             # List all feeds (from OPML)
│   └── import <file.opml>    # Import/merge external OPML
│
├── folder                    # Folder management
│   ├── add <name>            # Create folder in OPML
│   ├── remove <name>         # Remove folder (moves feeds to root)
│   └── list / ls             # List folders
│
├── sync [url|prefix]         # Fetch feeds, respect caching
│   └── --force               # Ignore ETag/Last-Modified
│
├── list / ls                 # List entries (default: unread)
│   ├── --all / -a            # Include read entries
│   ├── --feed <url|prefix>   # Filter by feed
│   ├── --folder <name>       # Filter by folder
│   ├── --since <date>        # Filter by date
│   └── --limit <n>           # Limit results (default 20)
│
├── read <prefix>             # Mark entry as read
├── unread <prefix>           # Mark entry as unread
├── open <prefix>             # Open link in browser + mark read
│
├── export                    # Export OPML to stdout
├── mcp                       # Start MCP server (stdio)
└── --db / --opml             # Override default paths
```

**Aliases:** `digest ls` = `digest list`, `digest f` = `digest feed`

**UUID prefix matching:** Like toki - type 6+ chars of UUID instead of full ID.

## MCP Integration

### Tools (CRUD for agents)

```
# Feed management
add_feed(url, folder?)        # Add feed to OPML
remove_feed(url)              # Remove feed from OPML
list_feeds()                  # List all subscribed feeds

# Folder management
add_folder(name)              # Create folder
remove_folder(name)           # Remove folder

# Sync
sync_feeds(url?)              # Fetch all or specific feed

# Entry operations
list_entries(feed?, folder?, unread_only?, since?, limit?)
mark_read(entry_id)
mark_unread(entry_id)
search_entries(query, limit?) # Full-text search in titles/content

# Bulk operations
mark_all_read(feed?, folder?) # Mark multiple as read
```

### Resources (read-only views)

```
digest://feeds                 # All subscribed feeds
digest://feeds/{folder}        # Feeds in a folder
digest://entries/unread        # Unread entries
digest://entries/recent        # Last 24 hours
digest://entries/today         # Today's entries
digest://stats                 # Counts, last sync times
```

### Prompts (workflow templates)

```
daily-digest      # "What's new today? Summarize key articles"
research-topic    # "Find recent articles about {topic}"
catch-up          # "I've been away for {days}, what did I miss?"
curate-feeds      # "Review my feeds, suggest additions/removals"
```

## Sync Behavior & Caching

### HTTP Smart Caching

When fetching a feed:
1. Check `feeds` table for stored `etag` and `last_modified`
2. Send conditional request:
   ```
   GET /feed.xml
   If-None-Match: "abc123"
   If-Modified-Since: Tue, 10 Dec 2024 12:00:00 GMT
   ```
3. If server returns `304 Not Modified` → skip parsing, done
4. If `200 OK` → parse feed, store new ETag/Last-Modified, upsert entries

### Entry Deduplication

Each entry has a `guid` (from the feed's `<guid>` or `<id>` element). We use `UNIQUE(feed_id, guid)` to prevent duplicates. On sync:
- New GUID → insert entry
- Existing GUID → skip (or optionally update content if changed)

### Sync Output

```
$ digest sync
Syncing 12 feeds...
  ✓ Hacker News         +8 new
  ✓ Lobsters            +3 new
  - XKCD                (not modified)
  ✓ Some Blog           +1 new
  ✗ Dead Feed           (error: 404)

Synced 12 entries from 4 feeds (2 cached, 1 error)
```

### Error Handling

- Network errors → log warning, continue with other feeds
- Parse errors → log warning, skip feed
- Store `last_error` and `error_count` in feeds table for visibility
- `digest feed list` shows feeds with persistent errors

## Testing Strategy

Scenario-driven testing with real feeds (no mocks):

```
Scenario: Fresh sync of a new feed
  Given an empty database
  And a valid RSS feed URL (use a stable public feed like xkcd.com/rss.xml)
  When I run "digest feed add <url>"
  And I run "digest sync"
  Then entries appear in the database
  And "digest list" shows unread entries

Scenario: Cached sync respects ETag
  Given a feed was synced previously
  When I run "digest sync"
  Then the feed returns 304 Not Modified
  And no duplicate entries are created

Scenario: Mark entry as read
  Given unread entries exist
  When I run "digest read <prefix>"
  Then the entry is marked read
  And "digest list" (unread only) excludes it
  And "digest list --all" includes it

Scenario: OPML round-trip
  Given feeds in folders
  When I run "digest export > backup.opml"
  And I delete the OPML
  And I run "digest feed import backup.opml"
  Then all feeds and folders are restored
```

## Project Structure

```
digest/
├── cmd/digest/
│   ├── main.go              # Entry point
│   ├── root.go              # Root command, DB/OPML init
│   ├── feed.go              # feed add/remove/list/import
│   ├── folder.go            # folder add/remove/list
│   ├── sync.go              # sync command
│   ├── list.go              # list entries
│   ├── read.go              # read/unread/open commands
│   ├── export.go            # OPML export
│   └── mcp.go               # MCP server
├── internal/
│   ├── models/              # Feed, Entry, Folder structs
│   ├── db/                  # SQLite operations
│   ├── opml/                # OPML read/write/merge
│   ├── fetch/               # HTTP fetching with caching
│   ├── parse/               # RSS/Atom parsing
│   ├── ui/                  # Terminal formatting
│   └── mcp/                 # MCP tools/resources/prompts
├── test/                    # Integration tests
├── go.mod
└── Makefile
```

## Go Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/google/uuid` - UUID generation
- `modernc.org/sqlite` - Pure Go SQLite
- `github.com/mmcdole/gofeed` - RSS/Atom parser
- `github.com/mark3labs/mcp-go` - MCP SDK
- `github.com/fatih/color` - Terminal colors

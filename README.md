# Digest

A fast, lightweight RSS/Atom feed reader with both CLI and MCP (Model Context Protocol) server interfaces.

## Features

### Feed Management
- **Add feeds** with optional folder/category organization
- **Remove feeds** (cascades to delete all entries)
- **Move feeds** between folders for reorganization
- **Auto-discover** feed URLs from website URLs (built into `feed add`)
- **OPML import/export** for feed subscriptions

### Entry Tracking
- **Fetch feeds** with HTTP caching (ETag, Last-Modified) for efficiency
- **List entries** with filtering by feed, category, read status, and date
- **Smart date filters**: `today`, `yesterday`, `week`, `month`
- **Read articles** with HTML-to-markdown conversion
- **Mark as read/unread** - individual entries or bulk by date

### Storage Backends
- **SQLite** - fast, full-featured with FTS5 full-text search
- **Markdown** - human-readable file-based storage via mdstore
- Configurable via `digest setup` interactive wizard

### MCP Server
Full MCP integration for AI agents to manage feeds:

| Tool | Description |
|------|-------------|
| `list_feeds` | List all subscribed feeds with metadata |
| `add_feed` | Add a new feed with optional folder |
| `remove_feed` | Remove a feed and all its entries |
| `move_feed` | Move a feed to a different folder |
| `sync_feeds` | Fetch new entries from feeds |
| `list_entries` | List entries with date/read filters |
| `get_entry` | Get full article content as markdown |
| `mark_read` | Mark an entry as read |
| `mark_unread` | Mark an entry as unread |
| `bulk_mark_read` | Mark all entries before a date as read |

### MCP Resources
| Resource | Description |
|----------|-------------|
| `digest://feeds` | All subscribed feeds |
| `digest://entries/unread` | Unread entries |
| `digest://entries/today` | Today's entries |
| `digest://stats` | Feed statistics |

### MCP Prompts
Workflow templates for common RSS management tasks:

| Prompt | Description |
|--------|-------------|
| `daily-digest` | Morning routine to summarize today's entries, prioritize content, and generate a digest |
| `catch-up` | Efficiently process backlog after time away - triage, prioritize, declare bankruptcy on low-value feeds |
| `curate-feeds` | Quarterly review to remove low-value feeds, identify gaps, and optimize subscriptions |

**Example prompt usage:**
```
# Morning digest
Use the daily-digest prompt to catch up on today's news

# After vacation
Use the catch-up prompt with days=14 to process two weeks of entries

# Feed cleanup
Use the curate-feeds prompt to optimize my subscriptions
```

## Installation

```bash
# From source
go install github.com/harper/digest/cmd/digest@latest

# Or clone and build
git clone https://github.com/harperreed/digest.git
cd digest
make build
```

## CLI Usage

```bash
# First-time setup (choose storage backend and data directory)
digest setup

# Add a feed (auto-discovers feed URL from HTML pages)
digest feed add https://example.com/feed.xml
digest feed add https://example.com --folder "Tech"
digest feed add https://example.com --title "My Blog" --no-discover

# List feeds
digest feed list

# Move feed to category
digest feed move https://example.com/feed.xml "News"

# Remove a feed
digest feed remove https://example.com/feed.xml

# Manage folders
digest folder add "Tech"
digest folder list

# Fetch new entries from all feeds
digest fetch
digest fetch --force              # Ignore cache, force re-fetch
digest fetch https://example.com  # Fetch single feed

# List entries
digest list                    # Unread entries (default limit: 20)
digest list --all              # Include read entries
digest list --today            # Today's entries
digest list --yesterday        # Yesterday's entries
digest list --week             # This week's entries
digest list --category "Tech"  # Entries from Tech folder
digest list --feed <url>       # Entries from a specific feed

# Read an article (supports ID prefix matching)
digest read abc12345
digest read abc12345 --no-mark    # Read without marking as read

# Open article link in browser
digest open abc12345

# Mark as read
digest mark-read abc12345              # Single entry
digest mark-read --before yesterday    # Bulk mark

# Mark as unread
digest mark-unread abc12345

# Export data
digest export                      # OPML to stdout
digest export --format yaml        # Full YAML export
digest export --format markdown    # Markdown export

# Migrate between storage backends
digest migrate
```

## MCP Server Usage

### Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "digest": {
      "command": "/path/to/digest",
      "args": ["mcp"]
    }
  }
}
```

### Example Agent Workflows

```
# Get today's news
list_entries { "since": "today", "unread_only": true }

# Read an article
get_entry { "entry_id": "abc12345" }

# Organize feeds
move_feed { "url": "https://example.com/feed", "folder": "Tech Blogs" }

# Catch up on old articles
bulk_mark_read { "before": "week" }
```

## Data Storage

- **Config**: `~/.config/digest/config.json`
- **Data directory**: `~/.local/share/digest/` (respects `XDG_DATA_HOME`)
- **Subscriptions**: `~/.local/share/digest/feeds.opml` (OPML)

## Development

```bash
# Run tests
make test

# Build
make build

# Install locally
make install

# Run MCP server
./digest mcp
```

## License

MIT

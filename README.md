# Digest

A fast, lightweight RSS/Atom feed reader with both CLI and MCP (Model Context Protocol) server interfaces.

## Features

### Feed Management
- **Add feeds** with optional folder/category organization
- **Remove feeds** (cascades to delete all entries)
- **Move feeds** between folders for reorganization
- **Auto-discover** feed URLs from website URLs
- **OPML import/export** for feed subscriptions

### Entry Tracking
- **Sync feeds** with HTTP caching (ETag, Last-Modified) for efficiency
- **List entries** with filtering by feed, category, read status, and date
- **Smart date filters**: `today`, `yesterday`, `week`, `month`
- **Read articles** with beautiful markdown rendering (HTML auto-converted)
- **Mark as read/unread** - individual entries or bulk by date

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
# Add a feed
digest feed add https://example.com/feed.xml
digest feed add https://example.com/feed.xml --folder "Tech"

# Discover feed from website
digest feed discover https://example.com

# List feeds
digest feed list

# Move feed to category
digest feed move https://example.com/feed.xml "News"

# Sync all feeds
digest sync

# List entries
digest list                    # All unread entries
digest list --all              # Include read entries
digest list --today            # Today's entries
digest list --category "Tech"  # Entries from Tech folder

# Read an article (supports ID prefix)
digest read abc12345

# Mark as read
digest mark-read abc12345              # Single entry
digest mark-read --before yesterday    # Bulk mark
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

- **Database**: `~/.config/digest/digest.db` (SQLite)
- **Subscriptions**: `~/.config/digest/feeds.opml` (OPML)

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Build
make build

# Run MCP server
./digest mcp
```

## License

MIT

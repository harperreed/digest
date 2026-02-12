---
name: digest
description: RSS feed reader and article tracking - manage feeds, read articles, track reading progress. Use when the user discusses RSS feeds, articles, or reading lists.
---

# digest - Feed Reader

RSS/Atom feed aggregator with read tracking and HTML-to-markdown content rendering.

## When to use digest

- User mentions RSS feeds or subscriptions
- User wants to read or track articles
- User asks about their reading list
- User discusses news sources or blogs

## Available MCP tools

| Tool | Purpose |
|------|---------|
| `mcp__digest__list_feeds` | List all subscribed feeds with metadata |
| `mcp__digest__add_feed` | Subscribe to a feed (with optional folder) |
| `mcp__digest__remove_feed` | Unsubscribe from a feed |
| `mcp__digest__move_feed` | Move a feed to a different folder |
| `mcp__digest__sync_feeds` | Fetch new entries from feeds |
| `mcp__digest__list_entries` | List entries with date/read filters |
| `mcp__digest__get_entry` | Get full article content as markdown |
| `mcp__digest__mark_read` | Mark an entry as read |
| `mcp__digest__mark_unread` | Mark an entry as unread |
| `mcp__digest__bulk_mark_read` | Mark all entries before a date as read |

## Common patterns

### Subscribe to a feed
```
mcp__digest__add_feed(url="https://example.com/feed.xml", folder="Tech")
```

### Get unread entries
```
mcp__digest__list_entries(unread_only=true)
```

### Get today's entries
```
mcp__digest__list_entries(since="today")
```

### Read full article content
```
mcp__digest__get_entry(entry_id="abc12345")
```

### Mark as read
```
mcp__digest__mark_read(entry_id="abc12345-1234-1234-1234-123456789abc")
```

### Fetch new content from all feeds
```
mcp__digest__sync_feeds()
```

### Bulk mark old entries as read
```
mcp__digest__bulk_mark_read(before="week")
```

## CLI commands (if MCP unavailable)

```bash
digest feed add https://example.com/feed.xml         # Add a feed
digest feed add https://example.com --folder "Tech"   # Add with folder
digest feed list                                      # List feeds
digest feed remove https://example.com/feed.xml       # Remove a feed
digest feed move https://example.com/feed.xml "News"  # Move to folder
digest fetch                                          # Fetch new entries
digest fetch --force                                  # Force fetch (ignore cache)
digest list                                           # List unread entries
digest list --all                                     # Include read entries
digest list --today                                   # Today's entries only
digest list --category "Tech"                         # Filter by folder
digest read <entry-id>                                # Read article content
digest read <entry-id> --no-mark                      # Read without marking read
digest mark-read <entry-id>                           # Mark single entry read
digest mark-read --before yesterday                   # Bulk mark read
digest mark-unread <entry-id>                         # Mark entry unread
digest open <entry-id>                                # Open link in browser
digest export                                         # Export OPML
digest export --format yaml                           # Export as YAML
digest export --format markdown                       # Export as Markdown
```

## Data location

Config: `~/.config/digest/config.json`
Data: `~/.local/share/digest/` (respects XDG_DATA_HOME)

Supports SQLite and Markdown storage backends, configurable via `digest setup`.

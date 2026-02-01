---
name: digest
description: RSS feed reader and article tracking - manage feeds, read articles, track reading progress. Use when the user discusses RSS feeds, articles, or reading lists.
---

# digest - Feed Reader

RSS/Atom feed aggregator with read tracking and full-text search.

## When to use digest

- User mentions RSS feeds or subscriptions
- User wants to read or track articles
- User asks about their reading list
- User discusses news sources or blogs

## Available MCP tools

| Tool | Purpose |
|------|---------|
| `mcp__digest__add_feed` | Subscribe to a feed |
| `mcp__digest__list_feeds` | List subscriptions |
| `mcp__digest__remove_feed` | Unsubscribe |
| `mcp__digest__list_articles` | Get articles |
| `mcp__digest__mark_read` | Mark article as read |
| `mcp__digest__mark_unread` | Mark as unread |
| `mcp__digest__search_articles` | Full-text search |
| `mcp__digest__refresh_feeds` | Fetch new articles |

## Common patterns

### Subscribe to a feed
```
mcp__digest__add_feed(url="https://example.com/feed.xml", title="Example Blog")
```

### Get unread articles
```
mcp__digest__list_articles(unread_only=true)
```

### Search articles
```
mcp__digest__search_articles(query="kubernetes")
```

### Mark as read
```
mcp__digest__mark_read(article_id="uuid")
```

### Refresh all feeds
```
mcp__digest__refresh_feeds()
```

## CLI commands (if MCP unavailable)

```bash
digest add https://example.com/feed.xml
digest list                       # List feeds
digest articles                   # Unread articles
digest articles --all             # All articles
digest read <id>                  # Mark read
digest search "query"             # Search
digest refresh                    # Fetch new
digest export --format opml       # Export OPML
digest export --format markdown   # Export markdown
```

## Data location

`~/.local/share/digest/digest.db` (SQLite with FTS5, respects XDG_DATA_HOME)

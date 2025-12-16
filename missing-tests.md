Looking at this comprehensive RSS/Atom feed tracker codebase, I can identify several areas where test coverage could be improved. The code is generally well-tested, but there are some gaps and edge cases that should be addressed.

## Missing Test Cases and Issues

### 1. **Feed Discovery Edge Cases** (`internal/discover/`)
**Issue**: Limited error handling and edge case coverage
```markdown
**Missing Tests for `internal/discover/discover.go`:**

1. Test malformed HTML parsing edge cases:
   - HTML with broken `<link>` tags
   - Multiple feeds with same type but different priorities
   - Feeds with relative URLs that can't be resolved
   - HTML with encoding issues

2. Test network error scenarios:
   - Timeout during discovery
   - Redirect loops
   - SSL/TLS certificate errors
   - Connection refused scenarios

3. Test feed URL validation edge cases:
   - URLs with unusual ports
   - URLs with authentication credentials
   - IPv6 addresses
   - Internationalized domain names

4. Test probe path behavior:
   - Server returning 200 but non-feed content for common paths
   - Servers that return different content based on User-Agent
```

### 2. **Database Transaction Safety** (`internal/db/`)
**Issue**: No tests for concurrent access or transaction rollback scenarios
```markdown
**Missing Tests for `internal/db/entries.go` and `internal/db/feeds.go`:**

1. Test concurrent access patterns:
   - Multiple goroutines creating entries simultaneously
   - Feed updates while entries are being created
   - Bulk operations during individual operations

2. Test database constraint violations:
   - Foreign key constraint failures
   - Unique constraint violations with better error messages
   - Database connection failures during operations

3. Test edge cases in `ListEntries`:
   - Very large date ranges
   - Invalid time zone handling
   - SQL injection attempts in prefix matching
   - Empty result sets with various filter combinations

4. Test `MarkEntriesReadBefore` edge cases:
   - Marking entries when database is read-only
   - Very large batch operations
   - Entries with NULL published_at dates
```

### 3. **OPML Operations Edge Cases** (`internal/opml/`)
**Issue**: Missing validation and error recovery tests
```markdown
**Missing Tests for `internal/opml/opml.go`:**

1. Test malformed OPML parsing:
   - OPML files with invalid XML structure
   - OPML files missing required attributes
   - OPML files with circular folder references
   - OPML files with duplicate feed URLs

2. Test folder operations edge cases:
   - Moving feeds between non-existent folders
   - Removing folders that contain feeds
   - Folder names with special characters or very long names
   - Adding feeds to folders with conflicting names

3. Test write operations failures:
   - File system permission errors
   - Disk full scenarios
   - Concurrent writes to same OPML file
   - Invalid file paths or directories

4. Test round-trip integrity:
   - Preserve XML comments and processing instructions
   - Maintain attribute order consistency
   - Handle XML namespaces properly
```

### 4. **HTTP Client Error Handling** (`internal/fetch/`)
**Issue**: Limited error scenario coverage
```markdown
**Missing Tests for `internal/fetch/fetch.go`:**

1. Test HTTP edge cases:
   - Servers returning malformed ETag headers
   - Servers returning future Last-Modified dates
   - Very large response bodies (memory limits)
   - Chunked transfer encoding issues

2. Test timeout and retry scenarios:
   - Network timeouts during header reading
   - Network timeouts during body reading
   - Connection reset by peer
   - DNS resolution failures

3. Test HTTP status code handling:
   - 3xx redirects with caching headers
   - 4xx errors with meaningful error messages
   - 5xx errors with retry logic
   - Non-standard status codes

4. Test content validation:
   - Empty response bodies with 200 status
   - Binary content returned instead of XML
   - Very large headers that exceed limits
```

### 5. **Content Processing Edge Cases** (`internal/content/`)
**Issue**: HTML to Markdown conversion needs more thorough testing
```markdown
**Missing Tests for `internal/content/content.go`:**

1. Test complex HTML structures:
   - Nested lists and tables
   - HTML with embedded CSS and JavaScript
   - Malformed HTML with missing closing tags
   - HTML with unusual encoding (UTF-16, etc.)

2. Test edge cases in HTML detection:
   - Text containing angle brackets but not HTML
   - XML content that's not HTML
   - Base64 encoded content
   - Very large content blocks

3. Test markdown conversion quality:
   - Preserve link structures properly
   - Handle image alt text and titles
   - Convert code blocks and syntax highlighting
   - Maintain table formatting
```

### 6. **CLI Command Error Handling** (`cmd/digest/`)
**Issue**: Missing validation and error recovery tests
```markdown
**Missing Tests for CLI Commands:**

1. Test `cmd/digest/feed.go`:
   - Adding feeds with invalid URLs
   - Removing non-existent feeds
   - Moving feeds to invalid folder names
   - Network failures during feed addition

2. Test `cmd/digest/sync.go`:
   - Sync with database connection failures
   - Sync with partial network failures
   - Force sync with corrupted cache data
   - Concurrent sync operations

3. Test `cmd/digest/read.go`:
   - Reading entries with corrupted content
   - Markdown rendering failures
   - Terminal width edge cases for formatting
   - Reading entries with missing metadata

4. Test `cmd/digest/list.go`:
   - Listing with corrupted database indexes
   - Very large entry lists (pagination edge cases)
   - Date filter edge cases (leap years, DST changes)
   - Category filtering with special characters
```

### 7. **MCP Server Integration** (`internal/mcp/`)
**Issue**: Limited error handling and malformed input testing
```markdown
**Missing Tests for MCP Components:**

1. Test `internal/mcp/tools.go`:
   - Malformed JSON input to tools
   - Tools called with missing required parameters
   - Tools called during database maintenance
   - Very large tool responses exceeding limits

2. Test `internal/mcp/resources.go`:
   - Resource requests during database locks
   - Resource serialization failures
   - Very large datasets in resources
   - Resource requests with invalid URIs

3. Test `internal/mcp/server.go`:
   - MCP protocol version mismatches
   - Client disconnection during operations
   - Server shutdown during active requests
   - Memory pressure during large operations
```

### 8. **Time Utilities Edge Cases** (`internal/timeutil/`)
**Issue**: Missing timezone and boundary condition tests
```markdown
**Missing Tests for `internal/timeutil/timeutil.go`:**

1. Test timezone edge cases:
   - DST transitions (spring forward, fall back)
   - Operations across different timezones
   - Leap second handling
   - Invalid timezone configurations

2. Test boundary conditions:
   - Start/end of year calculations
   - Leap year February 29th handling
   - Week calculations crossing year boundaries
   - Month calculations with different month lengths

3. Test period parsing edge cases:
   - Case sensitivity in period names
   - Period names in different languages
   - Invalid period combinations
   - Very old or very future dates
```

### 9. **Integration Test Gaps** (`test/integration_test.go`)
**Issue**: Missing real-world failure scenarios
```markdown
**Missing Integration Tests:**

1. Test failure recovery:
   - Recovery from corrupted database files
   - Recovery from corrupted OPML files
   - Network failures during sync operations
   - Disk full scenarios during operations

2. Test performance edge cases:
   - Large numbers of feeds (1000+)
   - Large numbers of entries (100,000+)
   - Very frequent sync operations
   - Memory usage during large operations

3. Test real feed compatibility:
   - Test with major RSS platforms (WordPress, etc.)
   - Test with Atom feeds from major platforms
   - Test with feeds that change formats
   - Test with feeds that have irregular update schedules
```

### 10. **Configuration and Path Handling**
**Issue**: Missing XDG compliance and path validation tests
```markdown
**Missing Tests for Configuration:**

1. Test XDG path compliance:
   - Behavior with missing XDG environment variables
   - Behavior with invalid XDG paths
   - Permission issues in XDG directories
   - Path creation failures

2. Test cross-platform compatibility:
   - Windows path handling
   - macOS path handling
   - Path length limitations on different systems
   - Special characters in paths
```

These missing tests represent critical gaps in coverage that could lead to runtime failures, data corruption, or poor user experience. Priority should be given to database transaction safety, error recovery scenarios, and edge cases in core functionality like feed parsing and OPML handling.

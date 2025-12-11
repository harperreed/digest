# digest Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an RSS/Atom feed tracker CLI with MCP integration for AI agents.

**Architecture:** OPML file stores subscriptions (source of truth), SQLite stores fetched entries and HTTP cache state. Cobra CLI with MCP server mode. Single Go binary.

**Tech Stack:** Go 1.24+, Cobra, SQLite (modernc.org), gofeed (RSS/Atom parsing), MCP Go SDK, fatih/color

---

## Phase 1: Project Scaffolding

### Task 1.1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `cmd/digest/main.go`

**Step 1: Initialize Go module**

```bash
cd /Users/harper/Public/src/personal/rss-mcp
go mod init github.com/harper/digest
```

**Step 2: Create minimal main.go**

```go
// ABOUTME: Entry point for digest CLI
// ABOUTME: Initializes and executes root command

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Execute() error {
	return nil
}
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/digest`
Expected: No errors, produces `digest` binary

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "feat: initialize go module and main entry point"
```

---

### Task 1.2: Add Cobra Root Command

**Files:**
- Create: `cmd/digest/root.go`
- Modify: `cmd/digest/main.go`

**Step 1: Create root.go with basic structure**

```go
// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes database/OPML

package main

import (
	"github.com/spf13/cobra"
)

var (
	dbPath   string
	opmlPath string
)

var rootCmd = &cobra.Command{
	Use:   "digest",
	Short: "RSS/Atom feed tracker with MCP integration",
	Long: `
██████╗ ██╗ ██████╗ ███████╗███████╗████████╗
██╔══██╗██║██╔════╝ ██╔════╝██╔════╝╚══██╔══╝
██║  ██║██║██║  ███╗█████╗  ███████╗   ██║
██║  ██║██║██║   ██║██╔══╝  ╚════██║   ██║
██████╔╝██║╚██████╔╝███████╗███████║   ██║
╚═════╝ ╚═╝ ╚═════╝ ╚══════╝╚══════╝   ╚═╝

RSS/Atom feed tracker for humans and AI agents.

Track feeds, sync content, and expose via MCP for Claude.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database file path (default: ~/.local/share/digest/digest.db)")
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path (default: ~/.local/share/digest/feeds.opml)")
}
```

**Step 2: Update main.go to use Execute from root**

```go
// ABOUTME: Entry point for digest CLI
// ABOUTME: Initializes and executes root command

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Add Cobra dependency**

```bash
go get github.com/spf13/cobra@v1.10.1
```

**Step 4: Verify it runs**

Run: `go run ./cmd/digest --help`
Expected: Shows digest help with banner and flag descriptions

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add Cobra root command with global flags"
```

---

### Task 1.3: Add Core Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add all dependencies**

```bash
go get github.com/google/uuid@v1.6.0
go get modernc.org/sqlite@v1.40.1
go get github.com/mmcdole/gofeed@latest
go get github.com/fatih/color@v1.18.0
go get github.com/mark3labs/mcp-go@latest
```

**Step 2: Tidy dependencies**

```bash
go mod tidy
```

**Step 3: Verify go.mod has all deps**

Run: `cat go.mod`
Expected: Shows all five dependencies in require block

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add core dependencies"
```

---

## Phase 2: Models

### Task 2.1: Create Feed Model

**Files:**
- Create: `internal/models/feed.go`
- Create: `internal/models/feed_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for Feed model
// ABOUTME: Validates Feed creation and field handling

package models

import (
	"testing"
	"time"
)

func TestNewFeed(t *testing.T) {
	url := "https://example.com/feed.xml"
	feed := NewFeed(url)

	if feed.URL != url {
		t.Errorf("expected URL %s, got %s", url, feed.URL)
	}
	if feed.ID == "" {
		t.Error("expected ID to be set")
	}
	if feed.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestFeed_SetCacheHeaders(t *testing.T) {
	feed := NewFeed("https://example.com/feed.xml")
	etag := `"abc123"`
	lastMod := "Tue, 10 Dec 2024 12:00:00 GMT"

	feed.SetCacheHeaders(etag, lastMod)

	if feed.ETag == nil || *feed.ETag != etag {
		t.Errorf("expected ETag %s, got %v", etag, feed.ETag)
	}
	if feed.LastModified == nil || *feed.LastModified != lastMod {
		t.Errorf("expected LastModified %s, got %v", lastMod, feed.LastModified)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/models/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: Feed model representing an RSS/Atom subscription
// ABOUTME: Tracks URL, cache state, and sync metadata

package models

import (
	"time"

	"github.com/google/uuid"
)

type Feed struct {
	ID            string
	URL           string
	Title         *string
	ETag          *string
	LastModified  *string
	LastFetchedAt *time.Time
	LastError     *string
	ErrorCount    int
	CreatedAt     time.Time
}

func NewFeed(url string) *Feed {
	return &Feed{
		ID:        uuid.New().String(),
		URL:       url,
		CreatedAt: time.Now(),
	}
}

func (f *Feed) SetCacheHeaders(etag, lastModified string) {
	if etag != "" {
		f.ETag = &etag
	}
	if lastModified != "" {
		f.LastModified = &lastModified
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/models/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/
git commit -m "feat: add Feed model with cache header support"
```

---

### Task 2.2: Create Entry Model

**Files:**
- Create: `internal/models/entry.go`
- Modify: `internal/models/feed_test.go` (add entry tests)

**Step 1: Write the failing test**

Add to `internal/models/feed_test.go`:

```go
func TestNewEntry(t *testing.T) {
	feedID := uuid.New().String()
	guid := "https://example.com/post/123"
	title := "Test Post"

	entry := NewEntry(feedID, guid, title)

	if entry.FeedID != feedID {
		t.Errorf("expected FeedID %s, got %s", feedID, entry.FeedID)
	}
	if entry.GUID != guid {
		t.Errorf("expected GUID %s, got %s", guid, entry.GUID)
	}
	if entry.Title == nil || *entry.Title != title {
		t.Errorf("expected Title %s, got %v", title, entry.Title)
	}
	if entry.Read {
		t.Error("expected Read to be false")
	}
}

func TestEntry_MarkRead(t *testing.T) {
	entry := NewEntry("feed-id", "guid", "title")

	if entry.Read {
		t.Error("expected entry to be unread initially")
	}

	entry.MarkRead()

	if !entry.Read {
		t.Error("expected entry to be read after MarkRead")
	}
	if entry.ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}
}

func TestEntry_MarkUnread(t *testing.T) {
	entry := NewEntry("feed-id", "guid", "title")
	entry.MarkRead()

	entry.MarkUnread()

	if entry.Read {
		t.Error("expected entry to be unread after MarkUnread")
	}
	if entry.ReadAt != nil {
		t.Error("expected ReadAt to be nil")
	}
}
```

Add import for uuid at top of test file:
```go
import (
	"testing"

	"github.com/google/uuid"
)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/models/... -v`
Expected: FAIL - NewEntry undefined

**Step 3: Write minimal implementation**

Create `internal/models/entry.go`:

```go
// ABOUTME: Entry model representing a single article/post from a feed
// ABOUTME: Tracks content, read state, and metadata

package models

import (
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	ID          string
	FeedID      string
	GUID        string
	Title       *string
	Link        *string
	Author      *string
	PublishedAt *time.Time
	Content     *string
	Read        bool
	ReadAt      *time.Time
	CreatedAt   time.Time
}

func NewEntry(feedID, guid, title string) *Entry {
	return &Entry{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		GUID:      guid,
		Title:     &title,
		Read:      false,
		CreatedAt: time.Now(),
	}
}

func (e *Entry) MarkRead() {
	e.Read = true
	now := time.Now()
	e.ReadAt = &now
}

func (e *Entry) MarkUnread() {
	e.Read = false
	e.ReadAt = nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/models/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/models/
git commit -m "feat: add Entry model with read/unread state"
```

---

## Phase 3: Database Layer

### Task 3.1: Database Connection and Path Helpers

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for database connection and path helpers
// ABOUTME: Validates XDG path resolution and connection lifecycle

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultDBPath(t *testing.T) {
	path := GetDefaultDBPath()

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	if filepath.Base(path) != "digest.db" {
		t.Errorf("expected digest.db, got %s", filepath.Base(path))
	}
}

func TestGetDefaultOPMLPath(t *testing.T) {
	path := GetDefaultOPMLPath()

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	if filepath.Base(path) != "feeds.opml" {
		t.Errorf("expected feeds.opml, got %s", filepath.Base(path))
	}
}

func TestInitDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	conn, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer conn.Close()

	// Verify tables exist
	var count int
	err = conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='feeds'").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Error("feeds table not created")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/db/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: Database connection management and initialization
// ABOUTME: Handles SQLite connection, XDG paths, and migrations

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func GetDefaultDBPath() string {
	return filepath.Join(getDataDir(), "digest", "digest.db")
}

func GetDefaultOPMLPath() string {
	return filepath.Join(getDataDir(), "digest", "feeds.opml")
}

func getDataDir() string {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		return dataDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".local", "share")
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS feeds (
		id TEXT PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		etag TEXT,
		last_modified TEXT,
		last_fetched_at DATETIME,
		last_error TEXT,
		error_count INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS entries (
		id TEXT PRIMARY KEY,
		feed_id TEXT NOT NULL,
		guid TEXT NOT NULL,
		title TEXT,
		link TEXT,
		author TEXT,
		published_at DATETIME,
		content TEXT,
		read BOOLEAN DEFAULT FALSE,
		read_at DATETIME,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
		UNIQUE(feed_id, guid)
	);

	CREATE INDEX IF NOT EXISTS idx_entries_feed_id ON entries(feed_id);
	CREATE INDEX IF NOT EXISTS idx_entries_read ON entries(read);
	CREATE INDEX IF NOT EXISTS idx_entries_published_at ON entries(published_at);

	CREATE TABLE IF NOT EXISTS entry_tags (
		entry_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		PRIMARY KEY (entry_id, tag),
		FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE
	);
	`
	_, err := db.Exec(schema)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/db/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/
git commit -m "feat: add database initialization with migrations"
```

---

### Task 3.2: Feed CRUD Operations

**Files:**
- Create: `internal/db/feeds.go`
- Create: `internal/db/feeds_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for feed database operations
// ABOUTME: Validates CRUD operations for feeds table

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/models"
)

func TestCreateFeed(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	feed.Title = strPtr("Test Feed")

	err := CreateFeed(conn, feed)
	if err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}

	// Verify it exists
	got, err := GetFeedByURL(conn, feed.URL)
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %s, got %s", feed.ID, got.ID)
	}
}

func TestGetFeedByPrefix(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	// Use first 8 chars of UUID
	prefix := feed.ID[:8]
	got, err := GetFeedByPrefix(conn, prefix)
	if err != nil {
		t.Fatalf("GetFeedByPrefix failed: %v", err)
	}
	if got.ID != feed.ID {
		t.Errorf("expected ID %s, got %s", feed.ID, got.ID)
	}
}

func TestListFeeds(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed2 := models.NewFeed("https://example.com/feed2.xml")
	_ = CreateFeed(conn, feed1)
	_ = CreateFeed(conn, feed2)

	feeds, err := ListFeeds(conn)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}
}

func TestDeleteFeed(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	err := DeleteFeed(conn, feed.ID)
	if err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	_, err = GetFeedByURL(conn, feed.URL)
	if err == nil {
		t.Error("expected feed to be deleted")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	return conn
}

func strPtr(s string) *string {
	return &s
}
```

Add import at top:
```go
import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/models"
)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/db/... -v`
Expected: FAIL - CreateFeed undefined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Feed database operations
// ABOUTME: CRUD operations for the feeds table

package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/harper/digest/internal/models"
)

func CreateFeed(db *sql.DB, feed *models.Feed) error {
	_, err := db.Exec(`
		INSERT INTO feeds (id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feed.ID, feed.URL, feed.Title, feed.ETag, feed.LastModified,
		feed.LastFetchedAt, feed.LastError, feed.ErrorCount, feed.CreatedAt,
	)
	return err
}

func GetFeedByURL(db *sql.DB, url string) (*models.Feed, error) {
	feed := &models.Feed{}
	err := db.QueryRow(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE url = ?`, url,
	).Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
		&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func GetFeedByPrefix(db *sql.DB, prefix string) (*models.Feed, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	rows, err := db.Query(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id LIKE ?`, prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed := &models.Feed{}
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
			&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	if len(feeds) == 0 {
		return nil, fmt.Errorf("no feed found with prefix %s", prefix)
	}
	if len(feeds) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d feeds", prefix, len(feeds))
	}
	return feeds[0], nil
}

func GetFeedByID(db *sql.DB, id string) (*models.Feed, error) {
	feed := &models.Feed{}
	err := db.QueryRow(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id = ?`, id,
	).Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
		&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func ListFeeds(db *sql.DB) ([]*models.Feed, error) {
	rows, err := db.Query(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed := &models.Feed{}
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
			&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func UpdateFeed(db *sql.DB, feed *models.Feed) error {
	_, err := db.Exec(`
		UPDATE feeds SET
			title = ?, etag = ?, last_modified = ?, last_fetched_at = ?,
			last_error = ?, error_count = ?
		WHERE id = ?`,
		feed.Title, feed.ETag, feed.LastModified, feed.LastFetchedAt,
		feed.LastError, feed.ErrorCount, feed.ID,
	)
	return err
}

func DeleteFeed(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM feeds WHERE id = ?", id)
	return err
}

func UpdateFeedFetchState(db *sql.DB, feedID string, etag, lastModified *string, fetchedAt time.Time) error {
	_, err := db.Exec(`
		UPDATE feeds SET etag = ?, last_modified = ?, last_fetched_at = ?, last_error = NULL, error_count = 0
		WHERE id = ?`,
		etag, lastModified, fetchedAt, feedID,
	)
	return err
}

func UpdateFeedError(db *sql.DB, feedID string, errMsg string) error {
	_, err := db.Exec(`
		UPDATE feeds SET last_error = ?, error_count = error_count + 1
		WHERE id = ?`,
		errMsg, feedID,
	)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/db/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/
git commit -m "feat: add feed CRUD database operations"
```

---

### Task 3.3: Entry CRUD Operations

**Files:**
- Create: `internal/db/entries.go`
- Create: `internal/db/entries_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for entry database operations
// ABOUTME: Validates CRUD operations for entries table

package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

func TestCreateEntry(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	entry := models.NewEntry(feed.ID, "guid-123", "Test Entry")
	err := CreateEntry(conn, entry)
	if err != nil {
		t.Fatalf("CreateEntry failed: %v", err)
	}

	got, err := GetEntryByID(conn, entry.ID)
	if err != nil {
		t.Fatalf("GetEntryByID failed: %v", err)
	}
	if *got.Title != "Test Entry" {
		t.Errorf("expected title 'Test Entry', got %s", *got.Title)
	}
}

func TestCreateEntry_Duplicate(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	entry1 := models.NewEntry(feed.ID, "same-guid", "Entry 1")
	entry2 := models.NewEntry(feed.ID, "same-guid", "Entry 2")

	_ = CreateEntry(conn, entry1)
	err := CreateEntry(conn, entry2)
	if err == nil {
		t.Error("expected duplicate entry to fail")
	}
}

func TestListEntries_Unread(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	entry1 := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	entry2 := models.NewEntry(feed.ID, "guid-2", "Entry 2")
	entry2.MarkRead()

	_ = CreateEntry(conn, entry1)
	_ = CreateEntry(conn, entry2)

	unreadOnly := true
	entries, err := ListEntries(conn, nil, &unreadOnly, nil, 100)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 unread entry, got %d", len(entries))
	}
}

func TestMarkEntryRead(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	feed := models.NewFeed("https://example.com/feed.xml")
	_ = CreateFeed(conn, feed)

	entry := models.NewEntry(feed.ID, "guid-1", "Entry 1")
	_ = CreateEntry(conn, entry)

	err := MarkEntryRead(conn, entry.ID)
	if err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	got, _ := GetEntryByID(conn, entry.ID)
	if !got.Read {
		t.Error("expected entry to be marked read")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/db/... -v -run Entry`
Expected: FAIL - CreateEntry undefined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Entry database operations
// ABOUTME: CRUD operations for the entries table

package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/harper/digest/internal/models"
)

func CreateEntry(db *sql.DB, entry *models.Entry) error {
	_, err := db.Exec(`
		INSERT INTO entries (id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.FeedID, entry.GUID, entry.Title, entry.Link, entry.Author,
		entry.PublishedAt, entry.Content, entry.Read, entry.ReadAt, entry.CreatedAt,
	)
	return err
}

func GetEntryByID(db *sql.DB, id string) (*models.Entry, error) {
	entry := &models.Entry{}
	err := db.QueryRow(`
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries WHERE id = ?`, id,
	).Scan(&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link, &entry.Author,
		&entry.PublishedAt, &entry.Content, &entry.Read, &entry.ReadAt, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func GetEntryByPrefix(db *sql.DB, prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	rows, err := db.Query(`
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries WHERE id LIKE ?`, prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		entry := &models.Entry{}
		if err := rows.Scan(&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link, &entry.Author,
			&entry.PublishedAt, &entry.Content, &entry.Read, &entry.ReadAt, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entry found with prefix %s", prefix)
	}
	if len(entries) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d entries", prefix, len(entries))
	}
	return entries[0], nil
}

func ListEntries(db *sql.DB, feedID *string, unreadOnly *bool, since *time.Time, limit int) ([]*models.Entry, error) {
	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries WHERE 1=1`
	args := []interface{}{}

	if feedID != nil {
		query += " AND feed_id = ?"
		args = append(args, *feedID)
	}
	if unreadOnly != nil && *unreadOnly {
		query += " AND read = FALSE"
	}
	if since != nil {
		query += " AND published_at >= ?"
		args = append(args, *since)
	}

	query += " ORDER BY published_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		entry := &models.Entry{}
		if err := rows.Scan(&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link, &entry.Author,
			&entry.PublishedAt, &entry.Content, &entry.Read, &entry.ReadAt, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func MarkEntryRead(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE entries SET read = TRUE, read_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func MarkEntryUnread(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE entries SET read = FALSE, read_at = NULL WHERE id = ?", id)
	return err
}

func EntryExists(db *sql.DB, feedID, guid string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM entries WHERE feed_id = ? AND guid = ?", feedID, guid).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func CountUnreadEntries(db *sql.DB, feedID *string) (int, error) {
	query := "SELECT COUNT(*) FROM entries WHERE read = FALSE"
	args := []interface{}{}
	if feedID != nil {
		query += " AND feed_id = ?"
		args = append(args, *feedID)
	}
	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	return count, err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/db/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/
git commit -m "feat: add entry CRUD database operations"
```

---

## Phase 4: OPML Handling

### Task 4.1: OPML Parser and Writer

**Files:**
- Create: `internal/opml/opml.go`
- Create: `internal/opml/opml_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for OPML parsing and writing
// ABOUTME: Validates round-trip read/write of OPML files

package opml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOPML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test Feeds</title></head>
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="HN" xmlUrl="https://news.ycombinator.com/rss"/>
    </outline>
    <outline type="rss" text="XKCD" xmlUrl="https://xkcd.com/rss.xml"/>
  </body>
</opml>`

	doc, err := Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if doc.Title != "Test Feeds" {
		t.Errorf("expected title 'Test Feeds', got %s", doc.Title)
	}

	feeds := doc.AllFeeds()
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}

	folders := doc.Folders()
	if len(folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(folders))
	}
	if folders[0] != "Tech" {
		t.Errorf("expected folder 'Tech', got %s", folders[0])
	}
}

func TestOPML_AddFeed(t *testing.T) {
	doc := NewDocument("My Feeds")

	doc.AddFeed("https://example.com/feed.xml", "Example", "")

	feeds := doc.AllFeeds()
	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}
	if feeds[0].URL != "https://example.com/feed.xml" {
		t.Errorf("expected URL, got %s", feeds[0].URL)
	}
}

func TestOPML_AddFeedToFolder(t *testing.T) {
	doc := NewDocument("My Feeds")

	doc.AddFolder("Tech")
	doc.AddFeed("https://example.com/feed.xml", "Example", "Tech")

	feeds := doc.FeedsInFolder("Tech")
	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed in Tech folder, got %d", len(feeds))
	}
}

func TestOPML_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.opml")

	doc := NewDocument("Test Feeds")
	doc.AddFolder("News")
	doc.AddFeed("https://example.com/feed.xml", "Example", "News")
	doc.AddFeed("https://other.com/rss", "Other", "")

	err := doc.WriteFile(path)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	loaded, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(loaded.AllFeeds()) != 2 {
		t.Errorf("expected 2 feeds after round-trip, got %d", len(loaded.AllFeeds()))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/opml/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: OPML file parsing and writing
// ABOUTME: Handles RSS subscription lists with folder organization

package opml

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type Document struct {
	Title    string
	Outlines []*Outline
}

type Outline struct {
	Text     string
	Title    string
	Type     string
	XMLURL   string
	Children []*Outline
}

type Feed struct {
	URL    string
	Title  string
	Folder string
}

type opmlXML struct {
	XMLName xml.Name  `xml:"opml"`
	Version string    `xml:"version,attr"`
	Head    headXML   `xml:"head"`
	Body    bodyXML   `xml:"body"`
}

type headXML struct {
	Title string `xml:"title"`
}

type bodyXML struct {
	Outlines []outlineXML `xml:"outline"`
}

type outlineXML struct {
	Text     string       `xml:"text,attr"`
	Title    string       `xml:"title,attr,omitempty"`
	Type     string       `xml:"type,attr,omitempty"`
	XMLURL   string       `xml:"xmlUrl,attr,omitempty"`
	Children []outlineXML `xml:"outline,omitempty"`
}

func NewDocument(title string) *Document {
	return &Document{Title: title}
}

func Parse(r io.Reader) (*Document, error) {
	var raw opmlXML
	if err := xml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	doc := &Document{Title: raw.Head.Title}
	for _, o := range raw.Body.Outlines {
		doc.Outlines = append(doc.Outlines, convertOutline(o))
	}
	return doc, nil
}

func ParseFile(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func convertOutline(o outlineXML) *Outline {
	outline := &Outline{
		Text:   o.Text,
		Title:  o.Title,
		Type:   o.Type,
		XMLURL: o.XMLURL,
	}
	for _, child := range o.Children {
		outline.Children = append(outline.Children, convertOutline(child))
	}
	return outline
}

func (d *Document) AllFeeds() []Feed {
	var feeds []Feed
	for _, o := range d.Outlines {
		feeds = append(feeds, collectFeeds(o, "")...)
	}
	return feeds
}

func collectFeeds(o *Outline, folder string) []Feed {
	var feeds []Feed
	if o.XMLURL != "" {
		title := o.Text
		if title == "" {
			title = o.Title
		}
		feeds = append(feeds, Feed{URL: o.XMLURL, Title: title, Folder: folder})
	}
	folderName := folder
	if o.XMLURL == "" && len(o.Children) > 0 {
		folderName = o.Text
		if folderName == "" {
			folderName = o.Title
		}
	}
	for _, child := range o.Children {
		feeds = append(feeds, collectFeeds(child, folderName)...)
	}
	return feeds
}

func (d *Document) Folders() []string {
	var folders []string
	for _, o := range d.Outlines {
		if o.XMLURL == "" && len(o.Children) > 0 {
			name := o.Text
			if name == "" {
				name = o.Title
			}
			folders = append(folders, name)
		}
	}
	return folders
}

func (d *Document) FeedsInFolder(folder string) []Feed {
	var feeds []Feed
	for _, f := range d.AllFeeds() {
		if f.Folder == folder {
			feeds = append(feeds, f)
		}
	}
	return feeds
}

func (d *Document) AddFolder(name string) {
	for _, o := range d.Outlines {
		if (o.Text == name || o.Title == name) && o.XMLURL == "" {
			return
		}
	}
	d.Outlines = append(d.Outlines, &Outline{Text: name, Title: name})
}

func (d *Document) AddFeed(url, title, folder string) {
	feed := &Outline{Text: title, Title: title, Type: "rss", XMLURL: url}

	if folder == "" {
		d.Outlines = append(d.Outlines, feed)
		return
	}

	for _, o := range d.Outlines {
		if (o.Text == folder || o.Title == folder) && o.XMLURL == "" {
			o.Children = append(o.Children, feed)
			return
		}
	}

	folderOutline := &Outline{Text: folder, Title: folder, Children: []*Outline{feed}}
	d.Outlines = append(d.Outlines, folderOutline)
}

func (d *Document) RemoveFeed(url string) bool {
	return removeFeedFromOutlines(&d.Outlines, url)
}

func removeFeedFromOutlines(outlines *[]*Outline, url string) bool {
	for i, o := range *outlines {
		if o.XMLURL == url {
			*outlines = append((*outlines)[:i], (*outlines)[i+1:]...)
			return true
		}
		if removeFeedFromOutlines(&o.Children, url) {
			return true
		}
	}
	return false
}

func (d *Document) Write(w io.Writer) error {
	raw := opmlXML{
		Version: "2.0",
		Head:    headXML{Title: d.Title},
	}
	for _, o := range d.Outlines {
		raw.Body.Outlines = append(raw.Body.Outlines, convertToXML(o))
	}

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(raw)
}

func (d *Document) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.Write(f)
}

func convertToXML(o *Outline) outlineXML {
	out := outlineXML{
		Text:   o.Text,
		Title:  o.Title,
		Type:   o.Type,
		XMLURL: o.XMLURL,
	}
	for _, child := range o.Children {
		out.Children = append(out.Children, convertToXML(child))
	}
	return out
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/opml/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/opml/
git commit -m "feat: add OPML parsing and writing with folder support"
```

---

## Phase 5: Feed Fetching

### Task 5.1: HTTP Fetcher with Caching

**Files:**
- Create: `internal/fetch/fetch.go`
- Create: `internal/fetch/fetch_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for HTTP feed fetching with caching
// ABOUTME: Validates ETag/Last-Modified conditional requests

package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetch_Fresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", "Tue, 10 Dec 2024 12:00:00 GMT")
		w.Write([]byte("<rss><channel><title>Test</title></channel></rss>"))
	}))
	defer server.Close()

	result, err := Fetch(server.URL, nil, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if result.NotModified {
		t.Error("expected fresh content, got NotModified")
	}
	if len(result.Body) == 0 {
		t.Error("expected body content")
	}
	if result.ETag != `"abc123"` {
		t.Errorf("expected ETag, got %s", result.ETag)
	}
}

func TestFetch_Cached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"abc123"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write([]byte("content"))
	}))
	defer server.Close()

	etag := `"abc123"`
	result, err := Fetch(server.URL, &etag, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !result.NotModified {
		t.Error("expected NotModified for cached content")
	}
}

func TestFetch_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Fetch(server.URL, nil, nil)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/fetch/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: HTTP feed fetching with conditional request support
// ABOUTME: Handles ETag and Last-Modified headers for efficient caching

package fetch

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Result struct {
	Body         []byte
	ETag         string
	LastModified string
	NotModified  bool
}

var client = &http.Client{
	Timeout: 30 * time.Second,
}

func Fetch(url string, etag, lastModified *string) (*Result, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "digest/1.0 (RSS reader)")

	if etag != nil && *etag != "" {
		req.Header.Set("If-None-Match", *etag)
	}
	if lastModified != nil && *lastModified != "" {
		req.Header.Set("If-Modified-Since", *lastModified)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return &Result{NotModified: true}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return &Result{
		Body:         body,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		NotModified:  false,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/fetch/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/fetch/
git commit -m "feat: add HTTP fetcher with ETag/Last-Modified caching"
```

---

## Phase 6: Feed Parsing

### Task 6.1: RSS/Atom Parser Integration

**Files:**
- Create: `internal/parse/parse.go`
- Create: `internal/parse/parse_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for RSS/Atom feed parsing
// ABOUTME: Validates conversion from gofeed to internal models

package parse

import (
	"testing"
)

func TestParse_RSS(t *testing.T) {
	rss := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Test Entry</title>
      <link>https://example.com/post</link>
      <guid>https://example.com/post</guid>
      <pubDate>Tue, 10 Dec 2024 12:00:00 GMT</pubDate>
      <description>Test content</description>
    </item>
  </channel>
</rss>`

	result, err := Parse([]byte(rss))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Title != "Test Feed" {
		t.Errorf("expected title 'Test Feed', got %s", result.Title)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Title != "Test Entry" {
		t.Errorf("expected entry title 'Test Entry', got %s", result.Entries[0].Title)
	}
}

func TestParse_Atom(t *testing.T) {
	atom := `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <entry>
    <title>Atom Entry</title>
    <id>urn:uuid:123</id>
    <link href="https://example.com/atom"/>
    <updated>2024-12-10T12:00:00Z</updated>
    <content>Atom content</content>
  </entry>
</feed>`

	result, err := Parse([]byte(atom))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Title != "Test Atom Feed" {
		t.Errorf("expected title 'Test Atom Feed', got %s", result.Title)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/parse/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: RSS/Atom feed parsing using gofeed library
// ABOUTME: Converts external feed formats to internal ParsedFeed structure

package parse

import (
	"bytes"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type ParsedFeed struct {
	Title   string
	Entries []ParsedEntry
}

type ParsedEntry struct {
	GUID        string
	Title       string
	Link        string
	Author      string
	PublishedAt *time.Time
	Content     string
	Categories  []string
}

func Parse(data []byte) (*ParsedFeed, error) {
	fp := gofeed.NewParser()
	feed, err := fp.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	result := &ParsedFeed{
		Title: feed.Title,
	}

	for _, item := range feed.Items {
		entry := ParsedEntry{
			GUID:       item.GUID,
			Title:      item.Title,
			Link:       item.Link,
			Categories: item.Categories,
		}

		if entry.GUID == "" {
			entry.GUID = item.Link
		}

		if item.Author != nil {
			entry.Author = item.Author.Name
		}

		if item.PublishedParsed != nil {
			entry.PublishedAt = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			entry.PublishedAt = item.UpdatedParsed
		}

		if item.Content != "" {
			entry.Content = item.Content
		} else {
			entry.Content = item.Description
		}

		result.Entries = append(result.Entries, entry)
	}

	return result, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/parse/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/parse/
git commit -m "feat: add RSS/Atom parsing with gofeed"
```

---

## Phase 7: CLI Commands

### Task 7.1: Database/OPML Initialization in Root

**Files:**
- Modify: `cmd/digest/root.go`

**Step 1: Update root.go with initialization**

```go
// ABOUTME: Root Cobra command and global flags
// ABOUTME: Sets up CLI structure and initializes database/OPML

package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/opml"
	"github.com/spf13/cobra"
)

var (
	dbPath   string
	opmlPath string
	dbConn   *sql.DB
	opmlDoc  *opml.Document
)

var rootCmd = &cobra.Command{
	Use:   "digest",
	Short: "RSS/Atom feed tracker with MCP integration",
	Long: `
██████╗ ██╗ ██████╗ ███████╗███████╗████████╗
██╔══██╗██║██╔════╝ ██╔════╝██╔════╝╚══██╔══╝
██║  ██║██║██║  ███╗█████╗  ███████╗   ██║
██║  ██║██║██║   ██║██╔══╝  ╚════██║   ██║
██████╔╝██║╚██████╔╝███████╗███████║   ██║
╚═════╝ ╚═╝ ╚═════╝ ╚══════╝╚══════╝   ╚═╝

RSS/Atom feed tracker for humans and AI agents.

Track feeds, sync content, and expose via MCP for Claude.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if dbPath == "" {
			dbPath = db.GetDefaultDBPath()
		}
		if opmlPath == "" {
			opmlPath = db.GetDefaultOPMLPath()
		}

		var err error
		dbConn, err = db.InitDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		if _, err := os.Stat(opmlPath); os.IsNotExist(err) {
			opmlDoc = opml.NewDocument("digest feeds")
		} else {
			opmlDoc, err = opml.ParseFile(opmlPath)
			if err != nil {
				return fmt.Errorf("failed to load OPML: %w", err)
			}
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if dbConn != nil {
			return dbConn.Close()
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database file path")
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path")
}

func saveOPML() error {
	return opmlDoc.WriteFile(opmlPath)
}
```

**Step 2: Verify it compiles and runs**

Run: `go build ./cmd/digest && ./digest --help`
Expected: Shows help, no errors

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add database and OPML initialization to root command"
```

---

### Task 7.2: Feed Add Command

**Files:**
- Create: `cmd/digest/feed.go`

**Step 1: Create feed command with add subcommand**

```go
// ABOUTME: Feed management commands
// ABOUTME: Add, remove, list, and import RSS/Atom feeds

package main

import (
	"fmt"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/models"
	"github.com/spf13/cobra"
)

var feedCmd = &cobra.Command{
	Use:     "feed",
	Aliases: []string{"f"},
	Short:   "Manage RSS/Atom feeds",
}

var feedAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a new feed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		folder, _ := cmd.Flags().GetString("folder")
		title, _ := cmd.Flags().GetString("title")

		existing, err := db.GetFeedByURL(dbConn, url)
		if err == nil && existing != nil {
			return fmt.Errorf("feed already exists: %s", url)
		}

		feed := models.NewFeed(url)
		if title != "" {
			feed.Title = &title
		}

		if err := db.CreateFeed(dbConn, feed); err != nil {
			return fmt.Errorf("failed to create feed: %w", err)
		}

		displayTitle := url
		if title != "" {
			displayTitle = title
		}
		opmlDoc.AddFeed(url, displayTitle, folder)
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		fmt.Printf("Added feed: %s\n", url)
		if folder != "" {
			fmt.Printf("  Folder: %s\n", folder)
		}
		return nil
	},
}

var feedListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all feeds",
	RunE: func(cmd *cobra.Command, args []string) error {
		feeds := opmlDoc.AllFeeds()
		if len(feeds) == 0 {
			fmt.Println("No feeds configured. Use 'digest feed add <url>' to add one.")
			return nil
		}

		for _, f := range feeds {
			if f.Folder != "" {
				fmt.Printf("[%s] %s - %s\n", f.Folder, f.Title, f.URL)
			} else {
				fmt.Printf("%s - %s\n", f.Title, f.URL)
			}
		}
		return nil
	},
}

var feedRemoveCmd = &cobra.Command{
	Use:   "remove <url>",
	Short: "Remove a feed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		feed, err := db.GetFeedByURL(dbConn, url)
		if err != nil {
			return fmt.Errorf("feed not found: %s", url)
		}

		if err := db.DeleteFeed(dbConn, feed.ID); err != nil {
			return fmt.Errorf("failed to delete feed: %w", err)
		}

		opmlDoc.RemoveFeed(url)
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		fmt.Printf("Removed feed: %s\n", url)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(feedCmd)
	feedCmd.AddCommand(feedAddCmd)
	feedCmd.AddCommand(feedListCmd)
	feedCmd.AddCommand(feedRemoveCmd)

	feedAddCmd.Flags().StringP("folder", "f", "", "folder to add feed to")
	feedAddCmd.Flags().StringP("title", "t", "", "title for the feed")
}
```

**Step 2: Verify commands work**

Run: `go build ./cmd/digest && ./digest feed --help`
Expected: Shows feed subcommands

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add feed management commands (add, list, remove)"
```

---

### Task 7.3: Sync Command

**Files:**
- Create: `cmd/digest/sync.go`

**Step 1: Create sync command**

```go
// ABOUTME: Sync command for fetching feed content
// ABOUTME: Handles HTTP caching and entry deduplication

package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/parse"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [url]",
	Short: "Fetch new entries from feeds",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		feeds, err := db.ListFeeds(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list feeds: %w", err)
		}

		if len(feeds) == 0 {
			fmt.Println("No feeds to sync. Use 'digest feed add <url>' to add one.")
			return nil
		}

		if len(args) > 0 {
			feed, err := db.GetFeedByURL(dbConn, args[0])
			if err != nil {
				feed, err = db.GetFeedByPrefix(dbConn, args[0])
				if err != nil {
					return fmt.Errorf("feed not found: %s", args[0])
				}
			}
			feeds = []*models.Feed{feed}
		}

		fmt.Printf("Syncing %d feeds...\n", len(feeds))

		var totalNew, cached, errors int
		for _, feed := range feeds {
			newCount, wasCached, err := syncFeed(feed, force)
			if err != nil {
				color.Red("  ✗ %s (%v)\n", feedDisplayName(feed), err)
				errors++
				_ = db.UpdateFeedError(dbConn, feed.ID, err.Error())
				continue
			}

			if wasCached {
				color.HiBlack("  - %s (not modified)\n", feedDisplayName(feed))
				cached++
			} else if newCount > 0 {
				color.Green("  ✓ %s +%d new\n", feedDisplayName(feed), newCount)
				totalNew += newCount
			} else {
				color.Green("  ✓ %s (no new entries)\n", feedDisplayName(feed))
			}
		}

		fmt.Printf("\nSynced %d new entries from %d feeds (%d cached, %d errors)\n",
			totalNew, len(feeds)-errors, cached, errors)
		return nil
	},
}

func syncFeed(feed *models.Feed, force bool) (int, bool, error) {
	var etag, lastMod *string
	if !force {
		etag = feed.ETag
		lastMod = feed.LastModified
	}

	result, err := fetch.Fetch(feed.URL, etag, lastMod)
	if err != nil {
		return 0, false, err
	}

	if result.NotModified {
		return 0, true, nil
	}

	parsed, err := parse.Parse(result.Body)
	if err != nil {
		return 0, false, err
	}

	if feed.Title == nil || *feed.Title == "" {
		feed.Title = &parsed.Title
	}

	var newCount int
	for _, pe := range parsed.Entries {
		exists, _ := db.EntryExists(dbConn, feed.ID, pe.GUID)
		if exists {
			continue
		}

		entry := models.NewEntry(feed.ID, pe.GUID, pe.Title)
		entry.Link = &pe.Link
		entry.Author = &pe.Author
		entry.PublishedAt = pe.PublishedAt
		entry.Content = &pe.Content

		if err := db.CreateEntry(dbConn, entry); err != nil {
			continue
		}
		newCount++
	}

	now := time.Now()
	feed.LastFetchedAt = &now
	feed.SetCacheHeaders(result.ETag, result.LastModified)
	_ = db.UpdateFeed(dbConn, feed)

	return newCount, false, nil
}

func feedDisplayName(feed *models.Feed) string {
	if feed.Title != nil && *feed.Title != "" {
		return *feed.Title
	}
	return feed.URL
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().Bool("force", false, "ignore cache headers")
}
```

**Step 2: Verify command compiles**

Run: `go build ./cmd/digest && ./digest sync --help`
Expected: Shows sync help

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add sync command with HTTP caching support"
```

---

### Task 7.4: List Entries Command

**Files:**
- Create: `cmd/digest/list.go`

**Step 1: Create list command**

```go
// ABOUTME: List entries command
// ABOUTME: Shows feed entries with filtering options

package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/harper/digest/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List feed entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		feedFilter, _ := cmd.Flags().GetString("feed")
		limit, _ := cmd.Flags().GetInt("limit")

		var feedID *string
		if feedFilter != "" {
			feed, err := db.GetFeedByURL(dbConn, feedFilter)
			if err != nil {
				feed, err = db.GetFeedByPrefix(dbConn, feedFilter)
				if err != nil {
					return fmt.Errorf("feed not found: %s", feedFilter)
				}
			}
			feedID = &feed.ID
		}

		unreadOnly := !all
		entries, err := db.ListEntries(dbConn, feedID, &unreadOnly, nil, limit)
		if err != nil {
			return fmt.Errorf("failed to list entries: %w", err)
		}

		if len(entries) == 0 {
			if all {
				fmt.Println("No entries found.")
			} else {
				fmt.Println("No unread entries. Use --all to see read entries.")
			}
			return nil
		}

		faint := color.New(color.Faint)
		for _, entry := range entries {
			prefix := entry.ID[:8]
			title := "Untitled"
			if entry.Title != nil {
				title = *entry.Title
			}

			readMark := " "
			if entry.Read {
				readMark = "✓"
			}

			pubDate := ""
			if entry.PublishedAt != nil {
				pubDate = entry.PublishedAt.Format(time.RFC822)
			}

			faint.Printf("%s ", prefix)
			fmt.Printf("%s %s", readMark, title)
			if pubDate != "" {
				faint.Printf(" (%s)", pubDate)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("all", "a", false, "show all entries including read")
	listCmd.Flags().StringP("feed", "f", "", "filter by feed URL or ID prefix")
	listCmd.Flags().IntP("limit", "n", 20, "maximum entries to show")
}
```

**Step 2: Verify command compiles**

Run: `go build ./cmd/digest && ./digest list --help`
Expected: Shows list help with flags

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add list command for viewing entries"
```

---

### Task 7.5: Read/Unread Commands

**Files:**
- Create: `cmd/digest/read.go`

**Step 1: Create read and unread commands**

```go
// ABOUTME: Read state management commands
// ABOUTME: Mark entries as read or unread

package main

import (
	"fmt"

	"github.com/harper/digest/internal/db"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <entry-prefix>",
	Short: "Mark an entry as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		entry, err := db.GetEntryByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("entry not found: %s", prefix)
		}

		if err := db.MarkEntryRead(dbConn, entry.ID); err != nil {
			return fmt.Errorf("failed to mark read: %w", err)
		}

		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("Marked as read: %s\n", title)
		return nil
	},
}

var unreadCmd = &cobra.Command{
	Use:   "unread <entry-prefix>",
	Short: "Mark an entry as unread",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		entry, err := db.GetEntryByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("entry not found: %s", prefix)
		}

		if err := db.MarkEntryUnread(dbConn, entry.ID); err != nil {
			return fmt.Errorf("failed to mark unread: %w", err)
		}

		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("Marked as unread: %s\n", title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(unreadCmd)
}
```

**Step 2: Verify commands compile**

Run: `go build ./cmd/digest && ./digest read --help`
Expected: Shows read help

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add read/unread commands for entry state"
```

---

### Task 7.6: Open Command

**Files:**
- Create: `cmd/digest/open.go`

**Step 1: Create open command**

```go
// ABOUTME: Open entry link in browser
// ABOUTME: Opens URL and marks entry as read

package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/harper/digest/internal/db"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <entry-prefix>",
	Short: "Open entry link in browser and mark as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		entry, err := db.GetEntryByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("entry not found: %s", prefix)
		}

		if entry.Link == nil || *entry.Link == "" {
			return fmt.Errorf("entry has no link")
		}

		if err := openBrowser(*entry.Link); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
		}

		_ = db.MarkEntryRead(dbConn, entry.ID)

		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("Opened: %s\n", title)
		return nil
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(openCmd)
}
```

**Step 2: Verify command compiles**

Run: `go build ./cmd/digest && ./digest open --help`
Expected: Shows open help

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add open command to launch browser"
```

---

### Task 7.7: Folder Commands

**Files:**
- Create: `cmd/digest/folder.go`

**Step 1: Create folder commands**

```go
// ABOUTME: Folder management commands
// ABOUTME: Create and list OPML folders

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Manage feed folders",
}

var folderAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		opmlDoc.AddFolder(name)
		if err := saveOPML(); err != nil {
			return fmt.Errorf("failed to save OPML: %w", err)
		}

		fmt.Printf("Created folder: %s\n", name)
		return nil
	},
}

var folderListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		folders := opmlDoc.Folders()
		if len(folders) == 0 {
			fmt.Println("No folders. Use 'digest folder add <name>' to create one.")
			return nil
		}

		for _, f := range folders {
			feeds := opmlDoc.FeedsInFolder(f)
			fmt.Printf("%s (%d feeds)\n", f, len(feeds))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(folderCmd)
	folderCmd.AddCommand(folderAddCmd)
	folderCmd.AddCommand(folderListCmd)
}
```

**Step 2: Verify commands compile**

Run: `go build ./cmd/digest && ./digest folder --help`
Expected: Shows folder subcommands

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add folder management commands"
```

---

### Task 7.8: Export Command

**Files:**
- Create: `cmd/digest/export.go`

**Step 1: Create export command**

```go
// ABOUTME: Export OPML to stdout
// ABOUTME: Allows backing up or transferring subscriptions

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export OPML to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		return opmlDoc.Write(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
```

**Step 2: Verify command compiles**

Run: `go build ./cmd/digest && ./digest export --help`
Expected: Shows export help

**Step 3: Commit**

```bash
git add cmd/digest/
git commit -m "feat: add export command for OPML output"
```

---

## Phase 8: Integration Testing

### Task 8.1: End-to-End Workflow Test

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write integration test**

```go
// ABOUTME: Integration tests for digest CLI
// ABOUTME: Tests end-to-end workflows with real feeds

package test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/opml"
	"github.com/harper/digest/internal/parse"
)

func TestFullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	opmlPath := filepath.Join(tmpDir, "feeds.opml")

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer conn.Close()

	doc := opml.NewDocument("Test Feeds")

	url := "https://xkcd.com/rss.xml"
	feed := models.NewFeed(url)
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("CreateFeed failed: %v", err)
	}
	doc.AddFeed(url, "XKCD", "")
	doc.WriteFile(opmlPath)

	result, err := fetch.Fetch(url, nil, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	parsed, err := parse.Parse(result.Body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Entries) == 0 {
		t.Fatal("expected entries from XKCD feed")
	}

	for _, pe := range parsed.Entries {
		entry := models.NewEntry(feed.ID, pe.GUID, pe.Title)
		entry.Link = &pe.Link
		entry.Content = &pe.Content
		entry.PublishedAt = pe.PublishedAt
		_ = db.CreateEntry(conn, entry)
	}

	unreadOnly := true
	entries, err := db.ListEntries(conn, nil, &unreadOnly, nil, 100)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected unread entries")
	}

	if err := db.MarkEntryRead(conn, entries[0].ID); err != nil {
		t.Fatalf("MarkEntryRead failed: %v", err)
	}

	unreadAfter, _ := db.ListEntries(conn, nil, &unreadOnly, nil, 100)
	if len(unreadAfter) >= len(entries) {
		t.Error("expected fewer unread entries after marking read")
	}
}

func TestOPMLRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.opml")

	doc := opml.NewDocument("Test")
	doc.AddFolder("Tech")
	doc.AddFeed("https://example.com/feed1.xml", "Feed 1", "Tech")
	doc.AddFeed("https://example.com/feed2.xml", "Feed 2", "")

	if err := doc.WriteFile(path); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	loaded, err := opml.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	feeds := loaded.AllFeeds()
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}

	folders := loaded.Folders()
	if len(folders) != 1 || folders[0] != "Tech" {
		t.Errorf("expected Tech folder, got %v", folders)
	}
}

func TestCachedSync(t *testing.T) {
	url := "https://xkcd.com/rss.xml"

	result1, err := fetch.Fetch(url, nil, nil)
	if err != nil {
		t.Fatalf("First fetch failed: %v", err)
	}

	if result1.ETag == "" && result1.LastModified == "" {
		t.Skip("Feed doesn't support caching headers")
	}

	etag := result1.ETag
	lastMod := result1.LastModified

	result2, err := fetch.Fetch(url, &etag, &lastMod)
	if err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}

	if !result2.NotModified {
		t.Log("Feed was modified between requests (or doesn't honor cache headers)")
	}
}
```

**Step 2: Run integration tests**

Run: `go test ./test/... -v`
Expected: PASS (requires network)

**Step 3: Commit**

```bash
git add test/
git commit -m "test: add integration tests for full workflow"
```

---

## Phase 9: MCP Integration

### Task 9.1: MCP Server Setup

**Files:**
- Create: `internal/mcp/server.go`
- Create: `cmd/digest/mcp.go`

**Step 1: Create MCP server**

```go
// ABOUTME: MCP server initialization
// ABOUTME: Exposes digest functionality to AI agents

package mcp

import (
	"database/sql"

	"github.com/harper/digest/internal/opml"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer
	db        *sql.DB
	opmlDoc   *opml.Document
	opmlPath  string
}

func NewServer(db *sql.DB, opmlDoc *opml.Document, opmlPath string) *Server {
	s := &Server{
		db:       db,
		opmlDoc:  opmlDoc,
		opmlPath: opmlPath,
	}

	s.mcpServer = server.NewMCPServer(
		"digest",
		"1.0.0",
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
```

**Step 2: Create MCP command**

```go
// ABOUTME: MCP server command
// ABOUTME: Starts MCP server for AI agent integration

package main

import (
	"github.com/harper/digest/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(dbConn, opmlDoc, opmlPath)
		return server.ServeStdio()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
```

**Step 3: Commit stub**

```bash
git add internal/mcp/ cmd/digest/mcp.go
git commit -m "feat: add MCP server stub"
```

---

### Task 9.2: MCP Tools Implementation

**Files:**
- Create: `internal/mcp/tools.go`

**Step 1: Implement MCP tools**

```go
// ABOUTME: MCP tool implementations
// ABOUTME: CRUD operations for feeds and entries via MCP

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/parse"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) registerTools() {
	s.mcpServer.AddTool(mcp.NewTool("list_feeds",
		mcp.WithDescription("List all subscribed feeds"),
	), s.listFeedsHandler)

	s.mcpServer.AddTool(mcp.NewTool("add_feed",
		mcp.WithDescription("Add a new feed subscription"),
		mcp.WithString("url", mcp.Required(), mcp.Description("Feed URL")),
		mcp.WithString("title", mcp.Description("Optional title")),
		mcp.WithString("folder", mcp.Description("Optional folder name")),
	), s.addFeedHandler)

	s.mcpServer.AddTool(mcp.NewTool("remove_feed",
		mcp.WithDescription("Remove a feed subscription"),
		mcp.WithString("url", mcp.Required(), mcp.Description("Feed URL to remove")),
	), s.removeFeedHandler)

	s.mcpServer.AddTool(mcp.NewTool("sync_feeds",
		mcp.WithDescription("Fetch new entries from all or specific feed"),
		mcp.WithString("url", mcp.Description("Optional: specific feed URL to sync")),
	), s.syncFeedsHandler)

	s.mcpServer.AddTool(mcp.NewTool("list_entries",
		mcp.WithDescription("List feed entries with optional filters"),
		mcp.WithString("feed_id", mcp.Description("Filter by feed ID")),
		mcp.WithBoolean("unread_only", mcp.Description("Only show unread entries")),
		mcp.WithNumber("limit", mcp.Description("Max entries to return (default 20)")),
	), s.listEntriesHandler)

	s.mcpServer.AddTool(mcp.NewTool("mark_read",
		mcp.WithDescription("Mark an entry as read"),
		mcp.WithString("entry_id", mcp.Required(), mcp.Description("Entry ID or prefix")),
	), s.markReadHandler)

	s.mcpServer.AddTool(mcp.NewTool("mark_unread",
		mcp.WithDescription("Mark an entry as unread"),
		mcp.WithString("entry_id", mcp.Required(), mcp.Description("Entry ID or prefix")),
	), s.markUnreadHandler)
}

func (s *Server) listFeedsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	feeds := s.opmlDoc.AllFeeds()
	data, _ := json.MarshalIndent(feeds, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) addFeedHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := req.Params.Arguments["url"].(string)
	title, _ := req.Params.Arguments["title"].(string)
	folder, _ := req.Params.Arguments["folder"].(string)

	feed := models.NewFeed(url)
	if title != "" {
		feed.Title = &title
	}

	if err := db.CreateFeed(s.db, feed); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create feed: %v", err)), nil
	}

	displayTitle := url
	if title != "" {
		displayTitle = title
	}
	s.opmlDoc.AddFeed(url, displayTitle, folder)
	s.opmlDoc.WriteFile(s.opmlPath)

	return mcp.NewToolResultText(fmt.Sprintf("Added feed: %s", url)), nil
}

func (s *Server) removeFeedHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := req.Params.Arguments["url"].(string)

	feed, err := db.GetFeedByURL(s.db, url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("feed not found: %s", url)), nil
	}

	if err := db.DeleteFeed(s.db, feed.ID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete: %v", err)), nil
	}

	s.opmlDoc.RemoveFeed(url)
	s.opmlDoc.WriteFile(s.opmlPath)

	return mcp.NewToolResultText(fmt.Sprintf("Removed feed: %s", url)), nil
}

func (s *Server) syncFeedsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	feeds, _ := db.ListFeeds(s.db)

	if urlArg, ok := req.Params.Arguments["url"].(string); ok && urlArg != "" {
		feed, err := db.GetFeedByURL(s.db, urlArg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("feed not found: %s", urlArg)), nil
		}
		feeds = []*models.Feed{feed}
	}

	var totalNew int
	for _, feed := range feeds {
		result, err := fetch.Fetch(feed.URL, feed.ETag, feed.LastModified)
		if err != nil {
			continue
		}
		if result.NotModified {
			continue
		}

		parsed, err := parse.Parse(result.Body)
		if err != nil {
			continue
		}

		for _, pe := range parsed.Entries {
			exists, _ := db.EntryExists(s.db, feed.ID, pe.GUID)
			if exists {
				continue
			}

			entry := models.NewEntry(feed.ID, pe.GUID, pe.Title)
			entry.Link = &pe.Link
			entry.Content = &pe.Content
			entry.PublishedAt = pe.PublishedAt
			if err := db.CreateEntry(s.db, entry); err == nil {
				totalNew++
			}
		}

		now := time.Now()
		feed.LastFetchedAt = &now
		feed.SetCacheHeaders(result.ETag, result.LastModified)
		db.UpdateFeed(s.db, feed)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Synced %d new entries from %d feeds", totalNew, len(feeds))), nil
}

func (s *Server) listEntriesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var feedID *string
	if fid, ok := req.Params.Arguments["feed_id"].(string); ok && fid != "" {
		feedID = &fid
	}

	unreadOnly := true
	if uo, ok := req.Params.Arguments["unread_only"].(bool); ok {
		unreadOnly = uo
	}

	limit := 20
	if l, ok := req.Params.Arguments["limit"].(float64); ok {
		limit = int(l)
	}

	entries, err := db.ListEntries(s.db, feedID, &unreadOnly, nil, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list entries: %v", err)), nil
	}

	data, _ := json.MarshalIndent(entries, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) markReadHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entryID := req.Params.Arguments["entry_id"].(string)

	entry, err := db.GetEntryByPrefix(s.db, entryID)
	if err != nil {
		entry, err = db.GetEntryByID(s.db, entryID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("entry not found: %s", entryID)), nil
		}
	}

	if err := db.MarkEntryRead(s.db, entry.ID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to mark read: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Marked as read: %s", entry.ID)), nil
}

func (s *Server) markUnreadHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entryID := req.Params.Arguments["entry_id"].(string)

	entry, err := db.GetEntryByPrefix(s.db, entryID)
	if err != nil {
		entry, err = db.GetEntryByID(s.db, entryID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("entry not found: %s", entryID)), nil
		}
	}

	if err := db.MarkEntryUnread(s.db, entry.ID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to mark unread: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Marked as unread: %s", entry.ID)), nil
}
```

**Step 2: Commit tools**

```bash
git add internal/mcp/tools.go
git commit -m "feat: add MCP tools for feed and entry operations"
```

---

### Task 9.3: MCP Resources Implementation

**Files:**
- Create: `internal/mcp/resources.go`

**Step 1: Implement MCP resources**

```go
// ABOUTME: MCP resource implementations
// ABOUTME: Read-only views of feeds and entries for AI agents

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harper/digest/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerResources() {
	s.mcpServer.AddResource(mcp.NewResource(
		"digest://feeds",
		"All subscribed feeds",
		mcp.WithResourceMIMEType("application/json"),
	), s.feedsResourceHandler)

	s.mcpServer.AddResource(mcp.NewResource(
		"digest://entries/unread",
		"Unread entries",
		mcp.WithResourceMIMEType("application/json"),
	), s.unreadEntriesHandler)

	s.mcpServer.AddResource(mcp.NewResource(
		"digest://entries/today",
		"Today's entries",
		mcp.WithResourceMIMEType("application/json"),
	), s.todayEntriesHandler)

	s.mcpServer.AddResource(mcp.NewResource(
		"digest://stats",
		"Feed statistics",
		mcp.WithResourceMIMEType("application/json"),
	), s.statsHandler)
}

func (s *Server) feedsResourceHandler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	feeds := s.opmlDoc.AllFeeds()
	data, _ := json.MarshalIndent(feeds, "", "  ")
	return []mcp.ResourceContents{
		mcp.NewTextResourceContents(req.Params.URI, "application/json", string(data)),
	}, nil
}

func (s *Server) unreadEntriesHandler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	unreadOnly := true
	entries, err := db.ListEntries(s.db, nil, &unreadOnly, nil, 50)
	if err != nil {
		return nil, err
	}
	data, _ := json.MarshalIndent(entries, "", "  ")
	return []mcp.ResourceContents{
		mcp.NewTextResourceContents(req.Params.URI, "application/json", string(data)),
	}, nil
}

func (s *Server) todayEntriesHandler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	today := time.Now().Truncate(24 * time.Hour)
	entries, err := db.ListEntries(s.db, nil, nil, &today, 50)
	if err != nil {
		return nil, err
	}
	data, _ := json.MarshalIndent(entries, "", "  ")
	return []mcp.ResourceContents{
		mcp.NewTextResourceContents(req.Params.URI, "application/json", string(data)),
	}, nil
}

func (s *Server) statsHandler(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	feeds, _ := db.ListFeeds(s.db)
	unreadCount, _ := db.CountUnreadEntries(s.db, nil)

	stats := map[string]interface{}{
		"total_feeds":    len(feeds),
		"unread_entries": unreadCount,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	data, _ := json.MarshalIndent(stats, "", "  ")
	return []mcp.ResourceContents{
		mcp.NewTextResourceContents(req.Params.URI, "application/json", string(data)),
	}, nil
}
```

**Step 2: Commit resources**

```bash
git add internal/mcp/resources.go
git commit -m "feat: add MCP resources for feed data views"
```

---

### Task 9.4: MCP Prompts Implementation

**Files:**
- Create: `internal/mcp/prompts.go`

**Step 1: Implement MCP prompts**

```go
// ABOUTME: MCP prompt templates
// ABOUTME: Workflow templates for AI agent interactions

package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPrompts() {
	s.mcpServer.AddPrompt(mcp.NewPrompt("daily-digest",
		mcp.WithPromptDescription("Summarize today's feed entries"),
	), s.dailyDigestPrompt)

	s.mcpServer.AddPrompt(mcp.NewPrompt("catch-up",
		mcp.WithPromptDescription("Catch up on missed entries"),
		mcp.WithArgument("days", mcp.ArgumentDescription("Number of days to look back")),
	), s.catchUpPrompt)

	s.mcpServer.AddPrompt(mcp.NewPrompt("curate-feeds",
		mcp.WithPromptDescription("Review and suggest feed improvements"),
	), s.curateFeedsPrompt)
}

func (s *Server) dailyDigestPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Daily feed digest",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Review today's feed entries and provide a summary:

1. First, use the list_entries tool with unread_only=true to get unread entries
2. Group entries by topic or feed
3. Highlight the most important or interesting items
4. Suggest which entries I should read in full
5. Mark entries as read after summarizing them

Keep the summary concise but informative.`,
				},
			},
		},
	}, nil
}

func (s *Server) catchUpPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	days := "3"
	if d, ok := req.Params.Arguments["days"]; ok {
		days = d
	}

	return &mcp.GetPromptResult{
		Description: "Catch up on missed entries",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `I've been away for ` + days + ` days. Help me catch up:

1. Sync feeds to get latest entries
2. List all unread entries
3. Identify the most significant news or updates
4. Summarize key themes across all feeds
5. Recommend top 5 entries I should read in full

Prioritize important updates over routine posts.`,
				},
			},
		},
	}, nil
}

func (s *Server) curateFeedsPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Review and curate feeds",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Review my feed subscriptions and suggest improvements:

1. List all current feeds
2. Check for feeds that haven't had new content recently
3. Identify any feeds with frequent errors
4. Suggest categories/folders for better organization
5. Recommend any feeds I might want to add based on my interests

Provide actionable suggestions for improving my feed collection.`,
				},
			},
		},
	}, nil
}
```

**Step 2: Commit prompts**

```bash
git add internal/mcp/prompts.go
git commit -m "feat: add MCP prompts for agent workflows"
```

---

## Phase 10: Final Polish

### Task 10.1: Add Version Command

**Files:**
- Create: `cmd/digest/version.go`

**Step 1: Create version command**

```go
// ABOUTME: Version command
// ABOUTME: Displays build version information

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("digest %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/digest/version.go
git commit -m "feat: add version command"
```

---

### Task 10.2: Add Makefile

**Files:**
- Create: `Makefile`

**Step 1: Create Makefile**

```makefile
# ABOUTME: Build and development tasks
# ABOUTME: Provides common operations for digest CLI

.PHONY: build test clean install

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o digest ./cmd/digest

test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f digest coverage.out coverage.html

install:
	go install $(LDFLAGS) ./cmd/digest
```

**Step 2: Verify build**

Run: `make build && ./digest version`
Expected: Shows version info

**Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile for build tasks"
```

---

### Task 10.3: Run All Tests

**Step 1: Run full test suite**

Run: `make test`
Expected: All tests pass

**Step 2: Final commit if any fixes needed**

```bash
git add .
git commit -m "fix: address any test failures"
```

---

## Execution Checklist

After completing all tasks, verify:

- [ ] `make build` succeeds
- [ ] `make test` passes all tests
- [ ] `./digest --help` shows all commands
- [ ] `./digest feed add https://xkcd.com/rss.xml` adds a feed
- [ ] `./digest sync` fetches entries
- [ ] `./digest list` shows unread entries
- [ ] `./digest read <prefix>` marks entry read
- [ ] `./digest export` outputs valid OPML
- [ ] `./digest mcp` starts without error (Ctrl+C to exit)

---

**Plan complete and saved to `docs/plans/2025-12-10-digest-implementation.md`.**

**Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach, Doctor Biz?**

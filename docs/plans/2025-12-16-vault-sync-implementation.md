# Vault Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add E2E encrypted sync for feeds and read state across devices using suite-sync vault library.

**Architecture:** Dual storage pattern - app database is source of truth, vault outbox mirrors changes for sync. Config stored separately. CLI command renamed from `sync` to `fetch`, new `sync` command for vault operations.

**Tech Stack:** Go, suitesync v0.3.0, SQLite, ULID, cobra

---

## Task 1: Add suitesync Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

```bash
cd /Users/harper/Public/src/personal/suite/digest
```

Edit `go.mod` to add after the `require` block:

```go
require (
	// ... existing deps
	github.com/oklog/ulid/v2 v2.1.1
	golang.org/x/term v0.38.0
	suitesync v0.3.0
)

replace suitesync => github.com/harperreed/sweet v0.3.0
```

**Step 2: Tidy modules**

Run: `uv run --with go go mod tidy` or just `go mod tidy`

**Step 3: Verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add suitesync v0.3.0 for vault sync"
```

---

## Task 2: Add Folder Column to Feeds Table

**Files:**
- Modify: `internal/db/db.go`
- Modify: `internal/models/feed.go`
- Create: `internal/db/db_test.go` (add migration test)

**Step 1: Write failing test for folder field**

Add to `internal/db/db_test.go`:

```go
func TestFeedFolderColumn(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a feed with folder
	_, err := db.Exec(`INSERT INTO feeds (id, url, folder, created_at) VALUES (?, ?, ?, ?)`,
		"test-id", "https://example.com/feed.xml", "Tech", time.Now())
	require.NoError(t, err)

	// Read it back
	var folder string
	err = db.QueryRow(`SELECT folder FROM feeds WHERE id = ?`, "test-id").Scan(&folder)
	require.NoError(t, err)
	assert.Equal(t, "Tech", folder)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/db/... -run TestFeedFolderColumn -v`
Expected: FAIL (no such column: folder)

**Step 3: Add folder to schema**

In `internal/db/db.go`, update the feeds table schema:

```go
CREATE TABLE IF NOT EXISTS feeds (
	id TEXT PRIMARY KEY,
	url TEXT UNIQUE NOT NULL,
	title TEXT,
	folder TEXT DEFAULT '',
	etag TEXT,
	last_modified TEXT,
	last_fetched_at DATETIME,
	last_error TEXT,
	error_count INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL
);
```

**Step 4: Add folder to Feed model**

In `internal/models/feed.go`, add field to struct:

```go
type Feed struct {
	ID            string
	URL           string
	Title         *string
	Folder        string     // Folder for organization (empty = root)
	ETag          *string
	LastModified  *string
	LastFetchedAt *time.Time
	LastError     *string
	ErrorCount    int
	CreatedAt     time.Time
}
```

**Step 5: Update feed queries to include folder**

In `internal/db/feeds.go`, update `scanFeed` and queries to include folder column.

**Step 6: Run test to verify it passes**

Run: `go test ./internal/db/... -run TestFeedFolderColumn -v`
Expected: PASS

**Step 7: Run all tests**

Run: `go test ./...`
Expected: All pass

**Step 8: Commit**

```bash
git add internal/db/db.go internal/db/feeds.go internal/models/feed.go internal/db/db_test.go
git commit -m "feat: add folder column to feeds table"
```

---

## Task 3: Rename sync.go to fetch.go

**Files:**
- Rename: `cmd/digest/sync.go` → `cmd/digest/fetch.go`
- Modify: Update command name from `sync` to `fetch`

**Step 1: Rename the file**

```bash
git mv cmd/digest/sync.go cmd/digest/fetch.go
```

**Step 2: Update command in fetch.go**

Change:
- `syncCmd` → `fetchCmd`
- `Use: "sync [url]"` → `Use: "fetch [url]"`
- `Short: "Fetch new entries from feeds"` (keep)
- Update `init()` to use `fetchCmd`

**Step 3: Update ABOUTME comments**

```go
// ABOUTME: Fetch command to retrieve new entries from RSS/Atom feeds with HTTP caching support
// ABOUTME: Handles batch fetching of all feeds or individual feed fetch with colored progress output
```

**Step 4: Run tests**

Run: `go test ./...`
Expected: All pass

**Step 5: Verify CLI**

Run: `go run ./cmd/digest fetch --help`
Expected: Shows fetch help

**Step 6: Commit**

```bash
git add cmd/digest/fetch.go
git commit -m "refactor: rename sync command to fetch

The sync command now fetches RSS entries. This frees up 'sync' for vault operations."
```

---

## Task 4: Create Sync Config

**Files:**
- Create: `internal/sync/config.go`
- Create: `internal/sync/config_test.go`

**Step 1: Write failing test**

Create `internal/sync/config_test.go`:

```go
// ABOUTME: Tests for sync configuration management
// ABOUTME: Verifies config load/save, defaults, and env overrides

package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDefaults(t *testing.T) {
	// Use temp dir for test
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Empty(t, cfg.Server)
	assert.Empty(t, cfg.Token)
	assert.NotEmpty(t, cfg.VaultDB) // Should have default path
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := &Config{
		Server:     "https://api.example.com",
		UserID:     "user123",
		Token:      "token456",
		DeviceID:   "device789",
		DerivedKey: "abcdef",
		AutoSync:   true,
	}

	err := SaveConfig(cfg)
	require.NoError(t, err)

	loaded, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, cfg.Server, loaded.Server)
	assert.Equal(t, cfg.UserID, loaded.UserID)
	assert.Equal(t, cfg.Token, loaded.Token)
	assert.Equal(t, cfg.DeviceID, loaded.DeviceID)
	assert.Equal(t, cfg.DerivedKey, loaded.DerivedKey)
	assert.Equal(t, cfg.AutoSync, loaded.AutoSync)
}

func TestIsConfigured(t *testing.T) {
	cfg := &Config{}
	assert.False(t, cfg.IsConfigured())

	cfg.Server = "https://api.example.com"
	assert.False(t, cfg.IsConfigured())

	cfg.Token = "token"
	cfg.UserID = "user"
	cfg.DerivedKey = "key"
	assert.True(t, cfg.IsConfigured())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/... -v`
Expected: FAIL (package does not exist)

**Step 3: Create config.go**

Create `internal/sync/config.go`:

```go
// ABOUTME: Sync configuration management for vault integration
// ABOUTME: Handles loading, saving, and environment overrides for sync settings

package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Config represents the sync configuration.
type Config struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenExpires string `json:"token_expires,omitempty"`
	DerivedKey   string `json:"derived_key"`
	DeviceID     string `json:"device_id"`
	VaultDB      string `json:"vault_db"`
	AutoSync     bool   `json:"auto_sync"`
}

// ConfigPath returns the path to the sync config file.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".digest", "sync.json")
	}
	return filepath.Join(home, ".config", "digest", "sync.json")
}

// ConfigDir returns the directory containing the config file.
func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	dir := ConfigDir()
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			backup := dir + ".backup." + time.Now().Format("20060102-150405")
			if err := os.Rename(dir, backup); err != nil {
				return fmt.Errorf("config path %s is a file, failed to backup: %w", dir, err)
			}
		} else {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check config dir: %w", err)
	}
	return os.MkdirAll(dir, 0o750)
}

// LoadConfig loads config from file and applies environment variable overrides.
func LoadConfig() (*Config, error) {
	cfg := defaultConfig()

	configPath := ConfigPath()

	info, statErr := os.Stat(configPath)
	if statErr == nil && info.IsDir() {
		return nil, fmt.Errorf("config path %s is a directory, not a file", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, cfg); jsonErr != nil {
			backup := configPath + ".corrupt." + time.Now().Format("20060102-150405")
			if renameErr := os.Rename(configPath, backup); renameErr == nil {
				fmt.Fprintf(os.Stderr, "Warning: corrupted config backed up to %s\n", backup)
			}
			return nil, fmt.Errorf("config file corrupted: %w", jsonErr)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	applyEnvOverrides(cfg)

	if cfg.VaultDB == "" {
		cfg.VaultDB = filepath.Join(ConfigDir(), "vault.db")
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		VaultDB: filepath.Join(ConfigDir(), "vault.db"),
	}
}

func applyEnvOverrides(cfg *Config) {
	if server := os.Getenv("DIGEST_SERVER"); server != "" {
		cfg.Server = server
	}
	if token := os.Getenv("DIGEST_TOKEN"); token != "" {
		cfg.Token = token
	}
	if userID := os.Getenv("DIGEST_USER_ID"); userID != "" {
		cfg.UserID = userID
	}
	if vaultDB := os.Getenv("DIGEST_VAULT_DB"); vaultDB != "" {
		cfg.VaultDB = expandPath(vaultDB)
	}
	if deviceID := os.Getenv("DIGEST_DEVICE_ID"); deviceID != "" {
		cfg.DeviceID = deviceID
	}
	if autoSync := os.Getenv("DIGEST_AUTO_SYNC"); autoSync == "1" || autoSync == "true" {
		cfg.AutoSync = true
	}
}

// SaveConfig writes config to file.
func SaveConfig(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// InitConfig creates a new config with device ID.
func InitConfig() (*Config, error) {
	deviceID := ulid.Make().String()

	cfg := &Config{
		DeviceID: deviceID,
		VaultDB:  filepath.Join(ConfigDir(), "vault.db"),
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ConfigExists returns true if config file exists.
func ConfigExists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// IsConfigured returns true if sync is fully configured.
func (c *Config) IsConfigured() bool {
	return c.Server != "" && c.Token != "" && c.UserID != "" && c.DerivedKey != ""
}

func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
```

**Step 4: Run tests**

Run: `go test ./internal/sync/... -v`
Expected: All pass

**Step 5: Commit**

```bash
git add internal/sync/
git commit -m "feat: add sync config management

Config stored at ~/.config/digest/sync.json with env overrides (DIGEST_*)."
```

---

## Task 5: Create Syncer

**Files:**
- Create: `internal/sync/sync.go`
- Create: `internal/sync/sync_test.go`

**Step 1: Write failing test**

Create `internal/sync/sync_test.go`:

```go
// ABOUTME: Tests for vault sync operations
// ABOUTME: Verifies syncer creation, entity queueing, and change application

package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncerRequiresDerivedKey(t *testing.T) {
	cfg := &Config{
		Server:   "https://api.example.com",
		Token:    "token",
		DeviceID: "device",
		// DerivedKey intentionally empty
	}

	_, err := NewSyncer(cfg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "derived key")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/sync/... -run TestNewSyncerRequiresDerivedKey -v`
Expected: FAIL (NewSyncer undefined)

**Step 3: Create sync.go**

Create `internal/sync/sync.go`:

```go
// ABOUTME: Vault sync integration for digest
// ABOUTME: Handles change queueing, syncing, and applying remote changes

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"suitesync/vault"
)

const (
	EntityFeed      = "feed"
	EntityReadState = "read_state"
)

// Syncer manages vault sync for digest data.
type Syncer struct {
	config *Config
	store  *vault.Store
	keys   vault.Keys
	client *vault.Client
	appDB  *sql.DB
}

// NewSyncer creates a new syncer from config.
func NewSyncer(cfg *Config, appDB *sql.DB) (*Syncer, error) {
	if cfg.DerivedKey == "" {
		return nil, errors.New("derived key not configured - run 'digest sync login' first")
	}

	seed, err := vault.ParseSeedPhrase(cfg.DerivedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid derived key: %w", err)
	}

	keys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	if err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}

	store, err := vault.OpenStore(cfg.VaultDB)
	if err != nil {
		return nil, fmt.Errorf("open vault store: %w", err)
	}

	client := vault.NewClient(vault.SyncConfig{
		BaseURL:   cfg.Server,
		DeviceID:  cfg.DeviceID,
		AuthToken: cfg.Token,
	})

	return &Syncer{
		config: cfg,
		store:  store,
		keys:   keys,
		client: client,
		appDB:  appDB,
	}, nil
}

// Close releases syncer resources.
func (s *Syncer) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// QueueFeedChange queues a change for a feed.
func (s *Syncer) QueueFeedChange(ctx context.Context, url, title, folder string, createdAt time.Time, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"url":        url,
			"title":      title,
			"folder":     folder,
			"created_at": createdAt.UTC().Unix(),
		}
	}

	// Use URL as entity ID (natural key)
	return s.queueChange(ctx, EntityFeed, url, op, payload)
}

// QueueReadStateChange queues a read state change.
func (s *Syncer) QueueReadStateChange(ctx context.Context, feedURL, guid string, read bool, readAt time.Time) error {
	entityID := feedURL + ":" + guid
	op := vault.OpUpsert

	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     read,
		"read_at":  readAt.UTC().Unix(),
	}

	return s.queueChange(ctx, EntityReadState, entityID, op, payload)
}

func (s *Syncer) queueChange(ctx context.Context, entity, entityID string, op vault.Op, payload map[string]any) error {
	change, err := vault.NewChange(entity, entityID, op, payload)
	if err != nil {
		return fmt.Errorf("create change: %w", err)
	}
	if op == vault.OpDelete {
		change.Deleted = true
	}

	plain, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("marshal change: %w", err)
	}

	aad := change.AAD(s.config.UserID, s.config.DeviceID)
	env, err := vault.Encrypt(s.keys.EncKey, plain, aad)
	if err != nil {
		return fmt.Errorf("encrypt change: %w", err)
	}

	if err := s.store.EnqueueEncryptedChange(ctx, change, s.config.UserID, s.config.DeviceID, env); err != nil {
		return fmt.Errorf("enqueue change: %w", err)
	}

	if s.config.AutoSync && s.canSync() {
		return s.Sync(ctx)
	}

	return nil
}

func (s *Syncer) canSync() bool {
	return s.config.Server != "" && s.config.Token != "" && s.config.UserID != ""
}

// Sync pushes local changes and pulls remote changes.
func (s *Syncer) Sync(ctx context.Context) error {
	return s.SyncWithEvents(ctx, nil)
}

// SyncWithEvents pushes local changes and pulls remote changes with progress callbacks.
func (s *Syncer) SyncWithEvents(ctx context.Context, events *vault.SyncEvents) error {
	if !s.canSync() {
		return errors.New("sync not configured - run 'digest sync login' first")
	}

	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange, events)
}

// applyChange applies a remote change to the local database.
func (s *Syncer) applyChange(ctx context.Context, c vault.Change) error {
	switch c.Entity {
	case EntityFeed:
		return s.applyFeedChange(ctx, c)
	case EntityReadState:
		return s.applyReadStateChange(ctx, c)
	default:
		return nil
	}
}

func (s *Syncer) applyFeedChange(ctx context.Context, c vault.Change) error {
	if c.Op == vault.OpDelete || c.Deleted {
		_, err := s.appDB.ExecContext(ctx, `DELETE FROM feeds WHERE url = ?`, c.EntityID)
		return err
	}

	var payload struct {
		URL       string `json:"url"`
		Title     string `json:"title"`
		Folder    string `json:"folder"`
		CreatedAt int64  `json:"created_at"`
	}
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal feed payload: %w", err)
	}

	createdAt := time.Unix(payload.CreatedAt, 0)

	// Check if feed exists
	var existingID string
	err := s.appDB.QueryRowContext(ctx, `SELECT id FROM feeds WHERE url = ?`, payload.URL).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		// Insert new feed
		id := generateID()
		_, err = s.appDB.ExecContext(ctx, `
			INSERT INTO feeds (id, url, title, folder, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, id, payload.URL, payload.Title, payload.Folder, createdAt)
		return err
	} else if err != nil {
		return fmt.Errorf("check feed exists: %w", err)
	}

	// Update existing feed
	_, err = s.appDB.ExecContext(ctx, `
		UPDATE feeds SET title = ?, folder = ? WHERE url = ?
	`, payload.Title, payload.Folder, payload.URL)
	return err
}

func (s *Syncer) applyReadStateChange(ctx context.Context, c vault.Change) error {
	var payload struct {
		FeedURL string `json:"feed_url"`
		GUID    string `json:"guid"`
		Read    bool   `json:"read"`
		ReadAt  int64  `json:"read_at"`
	}
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal read_state payload: %w", err)
	}

	// Find the entry by feed URL and GUID
	var entryID string
	var currentReadAt sql.NullTime
	err := s.appDB.QueryRowContext(ctx, `
		SELECT e.id, e.read_at FROM entries e
		JOIN feeds f ON e.feed_id = f.id
		WHERE f.url = ? AND e.guid = ?
	`, payload.FeedURL, payload.GUID).Scan(&entryID, &currentReadAt)

	if errors.Is(err, sql.ErrNoRows) {
		// Entry doesn't exist locally, skip
		return nil
	} else if err != nil {
		return fmt.Errorf("find entry: %w", err)
	}

	// Last-writer-wins: only apply if incoming is newer
	incomingReadAt := time.Unix(payload.ReadAt, 0)
	if currentReadAt.Valid && currentReadAt.Time.After(incomingReadAt) {
		return nil // Local is newer, skip
	}

	// Apply the change
	if payload.Read {
		_, err = s.appDB.ExecContext(ctx, `
			UPDATE entries SET read = TRUE, read_at = ? WHERE id = ?
		`, incomingReadAt, entryID)
	} else {
		_, err = s.appDB.ExecContext(ctx, `
			UPDATE entries SET read = FALSE, read_at = NULL WHERE id = ?
		`, entryID)
	}
	return err
}

// PendingCount returns the number of changes waiting to be synced.
func (s *Syncer) PendingCount(ctx context.Context) (int, error) {
	batch, err := s.store.DequeueBatch(ctx, 1000)
	if err != nil {
		return 0, err
	}
	return len(batch), nil
}

// PendingItem represents a change waiting to be synced.
type PendingItem struct {
	ChangeID string
	Entity   string
	TS       time.Time
}

// PendingChanges returns details of changes waiting to be synced.
func (s *Syncer) PendingChanges(ctx context.Context) ([]PendingItem, error) {
	batch, err := s.store.DequeueBatch(ctx, 100)
	if err != nil {
		return nil, err
	}

	items := make([]PendingItem, len(batch))
	for i, b := range batch {
		items[i] = PendingItem{
			ChangeID: b.ChangeID,
			Entity:   b.Entity,
			TS:       time.Unix(b.TS, 0),
		}
	}
	return items, nil
}

// LastSyncedSeq returns the last pulled sequence number.
func (s *Syncer) LastSyncedSeq(ctx context.Context) (string, error) {
	return s.store.GetState(ctx, "last_pulled_seq", "0")
}

func generateID() string {
	return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000"), ".", "") + randomSuffix()
}

func randomSuffix() string {
	// Simple random suffix
	return fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
}
```

**Step 4: Run tests**

Run: `go test ./internal/sync/... -v`
Expected: All pass

**Step 5: Commit**

```bash
git add internal/sync/sync.go internal/sync/sync_test.go
git commit -m "feat: add syncer with feed and read_state handlers"
```

---

## Task 6: Create Sync CLI Commands

**Files:**
- Create: `cmd/digest/sync.go`

**Step 1: Create sync.go with all subcommands**

Create `cmd/digest/sync.go` - this is a large file, modeled on position's implementation. Include:
- `sync init` - Initialize config with device ID
- `sync login` - Auth with server + recovery phrase (use LoginWithDevice for v0.3)
- `sync status` - Show sync state
- `sync now` - Manual push/pull
- `sync logout` - Clear credentials
- `sync pending` - Show pending changes
- `sync wipe` - Emergency reset

**Step 2: Build and verify**

Run: `go build ./cmd/digest`
Expected: Build succeeds

**Step 3: Test CLI**

Run: `./digest sync --help`
Expected: Shows sync subcommands

**Step 4: Commit**

```bash
git add cmd/digest/sync.go
git commit -m "feat: add sync CLI commands (init, login, status, now, logout, wipe)"
```

---

## Task 7: Wire Sync into Feed Commands

**Files:**
- Modify: `cmd/digest/feed.go`
- Modify: `internal/db/feeds.go` (ensure folder is handled)

**Step 1: Update feed add to queue sync**

In the feed add command, after successfully adding a feed:

```go
// Queue for sync if configured
cfg, _ := sync.LoadConfig()
if cfg != nil && cfg.IsConfigured() {
	syncer, err := sync.NewSyncer(cfg, dbConn)
	if err == nil {
		defer syncer.Close()
		if err := syncer.QueueFeedChange(ctx, feed.URL, feedTitle, folder, time.Now(), vault.OpUpsert); err != nil {
			log.Printf("warning: failed to queue sync: %v", err)
		}
	}
}
```

**Step 2: Update feed remove to queue sync**

Similar pattern for feed removal with `vault.OpDelete`.

**Step 3: Update feed move to queue sync**

Similar pattern for feed move with `vault.OpUpsert`.

**Step 4: Run tests**

Run: `go test ./...`
Expected: All pass

**Step 5: Commit**

```bash
git add cmd/digest/feed.go internal/db/feeds.go
git commit -m "feat: wire sync into feed add/remove/move commands"
```

---

## Task 8: Wire Sync into Read Commands

**Files:**
- Modify: `cmd/digest/markread.go`

**Step 1: Update mark read to queue sync**

After marking an entry as read:

```go
// Queue read state sync
cfg, _ := sync.LoadConfig()
if cfg != nil && cfg.IsConfigured() {
	syncer, err := sync.NewSyncer(cfg, dbConn)
	if err == nil {
		defer syncer.Close()
		// Get feed URL for this entry
		feedURL := getFeedURLForEntry(entry.FeedID)
		if err := syncer.QueueReadStateChange(ctx, feedURL, entry.GUID, true, time.Now()); err != nil {
			log.Printf("warning: failed to queue read state sync: %v", err)
		}
	}
}
```

**Step 2: Update mark unread similarly**

With `read: false`.

**Step 3: Run tests**

Run: `go test ./...`
Expected: All pass

**Step 4: Commit**

```bash
git add cmd/digest/markread.go
git commit -m "feat: wire sync into mark read/unread commands"
```

---

## Task 9: Final Integration Test

**Files:**
- Modify: `test/integration_test.go` (add sync tests if applicable)

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All pass

**Step 2: Manual smoke test**

```bash
# Build
go build -o digest ./cmd/digest

# Test fetch (renamed from sync)
./digest fetch --help

# Test sync commands
./digest sync init
./digest sync status
```

**Step 3: Final commit if needed**

```bash
git add .
git commit -m "test: add integration tests for vault sync"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Add suitesync dependency | go.mod |
| 2 | Add folder column | db.go, feed.go |
| 3 | Rename sync→fetch | fetch.go |
| 4 | Create sync config | internal/sync/config.go |
| 5 | Create syncer | internal/sync/sync.go |
| 6 | Create sync CLI | cmd/digest/sync.go |
| 7 | Wire feed commands | cmd/digest/feed.go |
| 8 | Wire read commands | cmd/digest/markread.go |
| 9 | Integration test | test/ |

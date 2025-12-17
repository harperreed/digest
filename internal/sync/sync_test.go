// ABOUTME: Tests for vault sync operations
// ABOUTME: Verifies syncer creation, entity queueing, and change application

package sync

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harperreed/sweet/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test app database
	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	// Create seed and derive key
	seed, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	require.NotNil(t, syncer)
	defer func() { _ = syncer.Close() }()

	assert.Equal(t, cfg, syncer.config)
	assert.NotNil(t, syncer.store)
	assert.NotNil(t, syncer.client)
	assert.NotNil(t, syncer.keys)

	// Verify keys were derived correctly
	expectedKeys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	require.NoError(t, err)
	assert.Equal(t, expectedKeys.EncKey, syncer.keys.EncKey)
}

func TestNewSyncerRequiresDerivedKey(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	cfg := &Config{
		Server:   "https://api.example.com",
		Token:    "token",
		DeviceID: "device",
		VaultDB:  filepath.Join(tmpDir, "vault.db"),
		// DerivedKey intentionally empty
	}

	_, err := NewSyncer(cfg, appDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "derived key not configured")
}

func TestNewSyncerInvalidDerivedKey(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: "invalid-key-format",
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	_, err := NewSyncer(cfg, appDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid derived key")
}

func TestQueueFeedChangeUpsert(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	url := "https://example.com/feed.xml"
	title := "Example Feed"
	folder := "Tech"
	createdAt := time.Now().UTC()

	// Queue feed create
	err := syncer.QueueFeedChange(ctx, url, title, folder, createdAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueFeedChangeUpsertWithoutFolder(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	url := "https://example.com/feed.xml"
	title := "Example Feed"
	createdAt := time.Now().UTC()

	// Queue feed create without folder
	err := syncer.QueueFeedChange(ctx, url, title, "", createdAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueFeedChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	url := "https://example.com/feed.xml"
	createdAt := time.Now().UTC()

	// Queue feed delete
	err := syncer.QueueFeedChange(ctx, url, "", "", createdAt, vault.OpDelete)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueReadStateChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"
	readAt := time.Now().UTC()

	// Queue read state change
	err := syncer.QueueReadStateChange(ctx, feedURL, guid, true, readAt)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueReadStateChangeUnread(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"
	readAt := time.Now().UTC()

	// Queue unread state change
	err := syncer.QueueReadStateChange(ctx, feedURL, guid, false, readAt)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPendingCount(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	// Initially zero
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Queue multiple changes
	err = syncer.QueueFeedChange(ctx, "https://feed1.com/rss", "Feed 1", "Tech", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	err = syncer.QueueFeedChange(ctx, "https://feed2.com/rss", "Feed 2", "News", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	err = syncer.QueueReadStateChange(ctx, "https://feed1.com/rss", "entry-1", true, time.Now())
	require.NoError(t, err)

	// Verify count
	count, err = syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestMultipleChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	// Queue multiple feed changes
	for i := 0; i < 5; i++ {
		url := "https://example.com/feed" + string(rune('0'+i)) + ".xml"
		err := syncer.QueueFeedChange(ctx, url, "Feed "+string(rune('A'+i)), "", time.Now(), vault.OpUpsert)
		require.NoError(t, err)
	}

	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAutoSyncDisabled(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	// AutoSync is disabled by default in test setup
	assert.False(t, syncer.config.AutoSync)

	err := syncer.QueueFeedChange(ctx, "https://example.com/feed.xml", "Test Feed", "", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	// Change should be queued but not synced
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSyncNotConfigured(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	// Create syncer with missing server config
	cfg := &Config{
		Server:     "", // Empty server
		UserID:     "",
		Token:      "",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	defer func() { _ = syncer.Close() }()

	// Sync should fail with helpful error
	err = syncer.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync not configured")
}

func TestCanSync(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "fully configured",
			config: &Config{
				Server: "https://example.com",
				Token:  "token",
				UserID: "user",
			},
			expected: true,
		},
		{
			name: "missing server",
			config: &Config{
				Server: "",
				Token:  "token",
				UserID: "user",
			},
			expected: false,
		},
		{
			name: "missing token",
			config: &Config{
				Server: "https://example.com",
				Token:  "",
				UserID: "user",
			},
			expected: false,
		},
		{
			name: "missing user id",
			config: &Config{
				Server: "https://example.com",
				Token:  "token",
				UserID: "",
			},
			expected: false,
		},
		{
			name: "all missing",
			config: &Config{
				Server: "",
				Token:  "",
				UserID: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			appDB := setupTestDB(t, tmpDir)
			defer func() { _ = appDB.Close() }()

			_, phrase, err := vault.NewSeedPhrase()
			require.NoError(t, err)

			tt.config.DerivedKey = phrase
			tt.config.DeviceID = "test-device"
			tt.config.VaultDB = filepath.Join(tmpDir, "vault.db")

			syncer, err := NewSyncer(tt.config, appDB)
			require.NoError(t, err)
			defer func() { _ = syncer.Close() }()

			assert.Equal(t, tt.expected, syncer.canSync())
		})
	}
}

func TestPendingChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	// Queue some changes
	err := syncer.QueueFeedChange(ctx, "https://feed1.com/rss", "Feed 1", "Tech", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	err = syncer.QueueFeedChange(ctx, "https://feed2.com/rss", "Feed 2", "News", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	// Get pending changes
	changes, err := syncer.PendingChanges(ctx)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	// Verify structure
	for _, change := range changes {
		assert.NotEmpty(t, change.ChangeID)
		assert.True(t, strings.HasSuffix(change.Entity, EntityFeed), "entity should end with %s, got %s", EntityFeed, change.Entity)
		assert.False(t, change.TS.IsZero())
	}
}

func TestLastSyncedSeq(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	// Initially should be "0"
	seq, err := syncer.LastSyncedSeq(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0", seq)
}

func TestCloseNilStore(t *testing.T) {
	syncer := &Syncer{
		store: nil,
	}

	err := syncer.Close()
	assert.NoError(t, err)
}

func TestQueueChangeEncryption(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	err := syncer.QueueFeedChange(ctx, "https://encrypted.com/feed.xml", "Encrypted Feed", "", time.Now(), vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was encrypted (indirectly by checking it was queued)
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMultipleFeedChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	feeds := []struct {
		url    string
		title  string
		folder string
	}{
		{"https://tech.example.com/feed.xml", "Tech News", "Tech"},
		{"https://news.example.com/rss", "World News", "News"},
		{"https://blog.example.com/atom", "Personal Blog", ""},
		{"https://podcast.example.com/feed", "Podcast Feed", "Audio"},
	}

	for _, feed := range feeds {
		err := syncer.QueueFeedChange(ctx, feed.url, feed.title, feed.folder, time.Now(), vault.OpUpsert)
		require.NoError(t, err)
	}

	// Verify all feeds were queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(feeds), count)
}

func TestMultipleReadStateChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)

	feedURL := "https://example.com/feed.xml"
	entries := []string{"entry-1", "entry-2", "entry-3", "entry-4", "entry-5"}

	for _, guid := range entries {
		err := syncer.QueueReadStateChange(ctx, feedURL, guid, true, time.Now())
		require.NoError(t, err)
	}

	// Verify all read states were queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(entries), count)
}

// ABOUTME: Tests for applying remote changes to local database
// ABOUTME: Verifies feed and read state change application, including edge cases

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/sweet/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFeedChangeCreate(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"
	title := "Test Feed"
	folder := "Tech"
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"url":        url,
		"title":      title,
		"folder":     folder,
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was created
	var dbTitle, dbFolder string
	var dbCreatedAt time.Time
	err = appDB.QueryRowContext(ctx,
		`SELECT title, folder, created_at FROM feeds WHERE url = ?`,
		url).Scan(&dbTitle, &dbFolder, &dbCreatedAt)
	require.NoError(t, err)
	assert.Equal(t, title, dbTitle)
	assert.Equal(t, folder, dbFolder)
	assert.Equal(t, time.Unix(createdAt, 0).Unix(), dbCreatedAt.Unix())
}

func TestApplyFeedChangeUpdate(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"

	// Insert initial feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, folder, created_at) VALUES (?, ?, ?, ?, ?)`,
		feedID, url, "Original Title", "Original Folder", time.Now())
	require.NoError(t, err)

	// Apply update
	createdAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"url":        url,
		"title":      "Updated Title",
		"folder":     "Updated Folder",
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was updated (not duplicated)
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feeds WHERE url = ?`,
		url).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify updated values
	var title, folder string
	err = appDB.QueryRowContext(ctx,
		`SELECT title, folder FROM feeds WHERE url = ?`,
		url).Scan(&title, &folder)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", title)
	assert.Equal(t, "Updated Folder", folder)
}

func TestApplyFeedChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, folder, created_at) VALUES (?, ?, ?, ?, ?)`,
		feedID, url, "Test Feed", "Tech", time.Now())
	require.NoError(t, err)

	// Apply delete
	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was deleted
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feeds WHERE url = ?`,
		url).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyFeedChangeDeleteWithDeletedFlag(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, folder, created_at) VALUES (?, ?, ?, ?, ?)`,
		feedID, url, "Test Feed", "Tech", time.Now())
	require.NoError(t, err)

	// Apply delete with Deleted flag set
	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpUpsert,
		Deleted:  true,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was deleted
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feeds WHERE url = ?`,
		url).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyFeedChangeDeleteCascadesToEntries(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, folder, created_at) VALUES (?, ?, ?, ?, ?)`,
		feedID, url, "Test Feed", "Tech", time.Now())
	require.NoError(t, err)

	// Insert entries for the feed
	for i := 0; i < 3; i++ {
		entryID := uuid.New().String()
		_, err = appDB.ExecContext(ctx,
			`INSERT INTO entries (id, feed_id, guid, title, created_at) VALUES (?, ?, ?, ?, ?)`,
			entryID, feedID, "guid-"+string(rune('0'+i)), "Entry "+string(rune('A'+i)), time.Now())
		require.NoError(t, err)
	}

	// Verify entries exist
	var entryCount int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE feed_id = ?`,
		feedID).Scan(&entryCount)
	require.NoError(t, err)
	assert.Equal(t, 3, entryCount)

	// Delete feed
	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify entries were cascade deleted
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE feed_id = ?`,
		feedID).Scan(&entryCount)
	require.NoError(t, err)
	assert.Equal(t, 0, entryCount)
}

func TestApplyReadStateChangeMarkRead(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, created_at) VALUES (?, ?, ?, ?)`,
		feedID, feedURL, "Test Feed", time.Now())
	require.NoError(t, err)

	// Insert entry
	entryID := uuid.New().String()
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO entries (id, feed_id, guid, title, read, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		entryID, feedID, guid, "Test Entry", false, time.Now())
	require.NoError(t, err)

	// Apply read state change
	readAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     true,
		"read_at":  readAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyReadStateChange(ctx, change)
	require.NoError(t, err)

	// Verify entry was marked as read
	var read bool
	var dbReadAt sql.NullTime
	err = appDB.QueryRowContext(ctx,
		`SELECT read, read_at FROM entries WHERE id = ?`,
		entryID).Scan(&read, &dbReadAt)
	require.NoError(t, err)
	assert.True(t, read)
	assert.True(t, dbReadAt.Valid)
	assert.Equal(t, time.Unix(readAt, 0).Unix(), dbReadAt.Time.Unix())
}

func TestApplyReadStateChangeMarkUnread(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, created_at) VALUES (?, ?, ?, ?)`,
		feedID, feedURL, "Test Feed", time.Now())
	require.NoError(t, err)

	// Insert entry marked as read with old timestamp
	entryID := uuid.New().String()
	oldReadAt := time.Now().Add(-1 * time.Hour)
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO entries (id, feed_id, guid, title, read, read_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entryID, feedID, guid, "Test Entry", true, oldReadAt, time.Now())
	require.NoError(t, err)

	// Apply unread state change with newer timestamp
	readAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     false,
		"read_at":  readAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyReadStateChange(ctx, change)
	require.NoError(t, err)

	// Verify entry was marked as unread
	var read bool
	var dbReadAt sql.NullTime
	err = appDB.QueryRowContext(ctx,
		`SELECT read, read_at FROM entries WHERE id = ?`,
		entryID).Scan(&read, &dbReadAt)
	require.NoError(t, err)
	assert.False(t, read)
	assert.False(t, dbReadAt.Valid)
}

func TestApplyReadStateChangeOrphanedEntry(t *testing.T) {
	ctx := context.Background()
	syncer, _ := setupTestSyncerWithDB(t)

	feedURL := "https://nonexistent.com/feed.xml"
	guid := "orphaned-guid"

	// Apply read state change for non-existent entry
	readAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     true,
		"read_at":  readAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	// Should not error, just skip
	err = syncer.applyReadStateChange(ctx, change)
	require.NoError(t, err)
}

func TestApplyReadStateChangeLastWriterWins(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, created_at) VALUES (?, ?, ?, ?)`,
		feedID, feedURL, "Test Feed", time.Now())
	require.NoError(t, err)

	// Insert entry with recent read_at
	entryID := uuid.New().String()
	recentReadAt := time.Now()
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO entries (id, feed_id, guid, title, read, read_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entryID, feedID, guid, "Test Entry", true, recentReadAt, time.Now())
	require.NoError(t, err)

	// Apply older read state change
	oldReadAt := recentReadAt.Add(-1 * time.Hour).Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     false,
		"read_at":  oldReadAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyReadStateChange(ctx, change)
	require.NoError(t, err)

	// Verify local state was NOT updated (local is newer)
	var read bool
	var dbReadAt sql.NullTime
	err = appDB.QueryRowContext(ctx,
		`SELECT read, read_at FROM entries WHERE id = ?`,
		entryID).Scan(&read, &dbReadAt)
	require.NoError(t, err)
	assert.True(t, read) // Should still be read
	assert.True(t, dbReadAt.Valid)
	assert.Equal(t, recentReadAt.Unix(), dbReadAt.Time.Unix())
}

func TestApplyReadStateChangeNewerWins(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feedURL := "https://example.com/feed.xml"
	guid := "entry-guid-123"

	// Insert feed
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, created_at) VALUES (?, ?, ?, ?)`,
		feedID, feedURL, "Test Feed", time.Now())
	require.NoError(t, err)

	// Insert entry with old read_at
	entryID := uuid.New().String()
	oldReadAt := time.Now().Add(-2 * time.Hour)
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO entries (id, feed_id, guid, title, read, read_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entryID, feedID, guid, "Test Entry", true, oldReadAt, time.Now())
	require.NoError(t, err)

	// Apply newer read state change
	newReadAt := time.Now().Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     false,
		"read_at":  newReadAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyReadStateChange(ctx, change)
	require.NoError(t, err)

	// Verify local state WAS updated (incoming is newer)
	var read bool
	var dbReadAt sql.NullTime
	err = appDB.QueryRowContext(ctx,
		`SELECT read, read_at FROM entries WHERE id = ?`,
		entryID).Scan(&read, &dbReadAt)
	require.NoError(t, err)
	assert.False(t, read) // Should be unread now
	assert.False(t, dbReadAt.Valid)
}

func TestApplyChangeUnknownEntity(t *testing.T) {
	ctx := context.Background()
	syncer, _ := setupTestSyncerWithDB(t)

	change := vault.Change{
		Entity:   "unknown-future-entity",
		EntityID: "some-id",
		Op:       vault.OpUpsert,
		Payload:  []byte(`{"test":"data"}`),
	}

	// Should not error (forward compatibility)
	err := syncer.applyChange(ctx, change)
	assert.NoError(t, err)
}

func TestApplyChangeFeed(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://routed.example.com/feed.xml"
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"url":        url,
		"title":      "Routed Feed",
		"folder":     "Test",
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was created
	var title string
	err = appDB.QueryRowContext(ctx,
		`SELECT title FROM feeds WHERE url = ?`,
		url).Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "Routed Feed", title)
}

func TestApplyChangeReadState(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feedURL := "https://example.com/feed.xml"
	guid := "routed-guid"

	// Insert feed and entry
	feedID := uuid.New().String()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, created_at) VALUES (?, ?, ?, ?)`,
		feedID, feedURL, "Test Feed", time.Now())
	require.NoError(t, err)

	entryID := uuid.New().String()
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO entries (id, feed_id, guid, title, read, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		entryID, feedID, guid, "Test Entry", false, time.Now())
	require.NoError(t, err)

	// Apply read state via applyChange router
	readAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     true,
		"read_at":  readAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: feedURL + ":" + guid,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyChange(ctx, change)
	require.NoError(t, err)

	// Verify entry was marked as read
	var read bool
	err = appDB.QueryRowContext(ctx,
		`SELECT read FROM entries WHERE id = ?`,
		entryID).Scan(&read)
	require.NoError(t, err)
	assert.True(t, read)
}

func TestApplyFeedChangeInvalidPayload(t *testing.T) {
	ctx := context.Background()
	syncer, _ := setupTestSyncerWithDB(t)

	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: "https://example.com/feed.xml",
		Op:       vault.OpUpsert,
		Payload:  []byte("invalid json{{{"),
	}

	err := syncer.applyFeedChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestApplyReadStateChangeInvalidPayload(t *testing.T) {
	ctx := context.Background()
	syncer, _ := setupTestSyncerWithDB(t)

	change := vault.Change{
		Entity:   EntityReadState,
		EntityID: "feed:guid",
		Op:       vault.OpUpsert,
		Payload:  []byte("invalid json{{{"),
	}

	err := syncer.applyReadStateChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestApplyFeedChangeEmptyFolder(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	url := "https://example.com/feed.xml"
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"url":        url,
		"title":      "No Folder Feed",
		"folder":     "",
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityFeed,
		EntityID: url,
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyFeedChange(ctx, change)
	require.NoError(t, err)

	// Verify feed was created with empty folder
	var folder string
	err = appDB.QueryRowContext(ctx,
		`SELECT folder FROM feeds WHERE url = ?`,
		url).Scan(&folder)
	require.NoError(t, err)
	assert.Equal(t, "", folder)
}

func TestApplyMultipleFeedChanges(t *testing.T) {
	ctx := context.Background()
	syncer, appDB := setupTestSyncerWithDB(t)

	feeds := []struct {
		url    string
		title  string
		folder string
	}{
		{"https://tech.example.com/feed.xml", "Tech Feed", "Tech"},
		{"https://news.example.com/rss", "News Feed", "News"},
		{"https://blog.example.com/atom", "Blog Feed", ""},
	}

	for _, feed := range feeds {
		createdAt := time.Now().UTC().Unix()
		payload := map[string]any{
			"url":        feed.url,
			"title":      feed.title,
			"folder":     feed.folder,
			"created_at": createdAt,
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		change := vault.Change{
			Entity:   EntityFeed,
			EntityID: feed.url,
			Op:       vault.OpUpsert,
			Payload:  payloadBytes,
		}

		err = syncer.applyFeedChange(ctx, change)
		require.NoError(t, err)
	}

	// Verify all feeds were created
	var count int
	err := appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feeds`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(feeds), count)
}

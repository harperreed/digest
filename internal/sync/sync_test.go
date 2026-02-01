// ABOUTME: Tests for feed synchronization logic
// ABOUTME: Verifies SyncResult struct and basic sync behavior

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/storage"
)

func TestSyncResultFields(t *testing.T) {
	result := &SyncResult{
		NewEntries: 5,
		WasCached:  true,
	}

	if result.NewEntries != 5 {
		t.Errorf("expected NewEntries=5, got %d", result.NewEntries)
	}
	if !result.WasCached {
		t.Error("expected WasCached=true")
	}
}

func TestSyncFeed_Fresh(t *testing.T) {
	// Create test server with valid RSS feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"test123"`)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Test Article</title>
      <link>https://example.com/article1</link>
      <guid>guid-1</guid>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
      <description>Test content</description>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed(server.URL)
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync feed
	result, err := SyncFeed(context.Background(), store, feed, false)
	if err != nil {
		t.Fatalf("SyncFeed: %v", err)
	}

	if result.WasCached {
		t.Error("expected WasCached=false for fresh fetch")
	}
	if result.NewEntries != 1 {
		t.Errorf("expected 1 new entry, got %d", result.NewEntries)
	}

	// Verify entry was created
	entries, err := store.ListEntries(nil)
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry in store, got %d", len(entries))
	}
}

func TestSyncFeed_Cached(t *testing.T) {
	// Create test server that returns 304
	etag := `"cached123"`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<rss><channel><title>Test</title></channel></rss>`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	// Create feed with etag
	feed := models.NewFeed(server.URL)
	feed.ETag = &etag
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync feed (should be cached)
	result, err := SyncFeed(context.Background(), store, feed, false)
	if err != nil {
		t.Fatalf("SyncFeed: %v", err)
	}

	if !result.WasCached {
		t.Error("expected WasCached=true for 304 response")
	}
	if result.NewEntries != 0 {
		t.Errorf("expected 0 new entries for cached, got %d", result.NewEntries)
	}
}

func TestSyncFeed_Force(t *testing.T) {
	// Create test server with valid feed
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Should NOT have conditional headers when force=true
		if r.Header.Get("If-None-Match") != "" && requestCount == 2 {
			t.Error("expected no If-None-Match header when force=true")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test</title>
    <item>
      <title>Item 1</title>
      <guid>guid-1</guid>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	// Create feed with etag
	etag := `"force123"`
	feed := models.NewFeed(server.URL)
	feed.ETag = &etag
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync with force=true
	result, err := SyncFeed(context.Background(), store, feed, true)
	if err != nil {
		t.Fatalf("SyncFeed force: %v", err)
	}

	if result.WasCached {
		t.Error("expected WasCached=false when force=true")
	}
}

func TestSyncFeed_FetchError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	feed := models.NewFeed(server.URL)
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync should fail
	_, err := SyncFeed(context.Background(), store, feed, false)
	if err == nil {
		t.Error("expected error for failed fetch")
	}

	// Check error was recorded
	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if got.LastError == nil {
		t.Error("expected LastError to be set")
	}
	if got.ErrorCount != 1 {
		t.Errorf("expected ErrorCount=1, got %d", got.ErrorCount)
	}
}

func TestSyncFeed_ParseError(t *testing.T) {
	// Create test server with invalid feed content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid xml or feed content`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	feed := models.NewFeed(server.URL)
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync should fail
	_, err := SyncFeed(context.Background(), store, feed, false)
	if err == nil {
		t.Error("expected error for parse failure")
	}

	// Check error was recorded
	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if got.LastError == nil {
		t.Error("expected LastError to be set")
	}
}

func TestSyncFeed_TitleUpdate(t *testing.T) {
	// Create test server with feed that has a title
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>My Great Feed</title>
    <link>https://example.com</link>
  </channel>
</rss>`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	// Create feed without title
	feed := models.NewFeed(server.URL)
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Sync feed
	_, err := SyncFeed(context.Background(), store, feed, false)
	if err != nil {
		t.Fatalf("SyncFeed: %v", err)
	}

	// Check title was updated
	got, err := store.GetFeed(feed.ID)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if got.Title == nil || *got.Title != "My Great Feed" {
		t.Errorf("expected title 'My Great Feed', got %v", got.Title)
	}
}

func TestSyncFeed_ExistingEntries(t *testing.T) {
	// Create test server with entries
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test</title>
    <item>
      <title>Existing Article</title>
      <guid>existing-guid</guid>
    </item>
    <item>
      <title>New Article</title>
      <guid>new-guid</guid>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()

	store := newTestStore(t)
	defer store.Close()

	// Create feed
	feed := models.NewFeed(server.URL)
	if err := store.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Create existing entry
	entry := storage.NewEntry(feed.ID, "existing-guid", "Existing Article")
	if err := store.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Sync feed
	result, err := SyncFeed(context.Background(), store, feed, false)
	if err != nil {
		t.Fatalf("SyncFeed: %v", err)
	}

	// Should only add 1 new entry (the existing one is skipped)
	if result.NewEntries != 1 {
		t.Errorf("expected 1 new entry (existing skipped), got %d", result.NewEntries)
	}

	// Verify 2 total entries
	entries, err := store.ListEntries(nil)
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries total, got %d", len(entries))
	}
}

func newTestStore(t *testing.T) storage.Store {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "sync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store
}

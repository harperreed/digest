// ABOUTME: Tests for MCP server handlers
// ABOUTME: Uses real charm client and temp OPML files for isolated testing

//go:build !race

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/harper/digest/internal/charm"
	"github.com/harper/digest/internal/opml"
	"github.com/mark3labs/mcp-go/mcp"
)

// testServer creates a test MCP server with a temp OPML file and test charm client.
// Returns the server, client, and OPML path (path used internally for OPML writes).
func testServer(t *testing.T) (*Server, *charm.Client, string) { //nolint:unparam
	t.Helper()

	// Create temp OPML file
	tmpOPML, err := os.CreateTemp("", "test-*.opml")
	if err != nil {
		t.Fatalf("failed to create temp OPML: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(tmpOPML.Name())
	})

	// Write initial OPML content
	initialOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test Feeds</title></head>
  <body>
    <outline text="Tech" title="Tech">
      <outline text="Example Blog" title="Example Blog" type="rss" xmlUrl="https://example.com/feed.xml"/>
    </outline>
  </body>
</opml>`
	if _, err := tmpOPML.WriteString(initialOPML); err != nil {
		t.Fatalf("failed to write OPML: %v", err)
	}
	tmpOPML.Close()

	// Parse OPML
	opmlDoc, err := opml.ParseFile(tmpOPML.Name())
	if err != nil {
		t.Fatalf("failed to parse OPML: %v", err)
	}

	// Create test charm client
	client := newTestCharmClient(t)

	// Create server
	s := NewServer(client, opmlDoc, tmpOPML.Name())

	return s, client, tmpOPML.Name()
}

// newTestCharmClient creates a charm.Client for testing
func newTestCharmClient(t *testing.T) *charm.Client {
	t.Helper()

	dbName := "mcp-server-test-" + t.Name()
	tmpDir, err := os.MkdirTemp("", "mcp-charm-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	os.Setenv("CHARM_DATA_DIR", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("CHARM_DATA_DIR")
	})

	return charm.NewTestClientWithDBName(dbName, false)
}

func TestHandleListFeeds(t *testing.T) {
	s, client, _ := testServer(t)

	// Add a feed to the charm store
	feed := charm.NewFeed("https://example.com/feed.xml")
	title := "Example Blog"
	feed.Title = &title
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Call handler
	req := mcp.CallToolRequest{}
	result, err := s.handleListFeeds(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListFeeds: %v", err)
	}

	// Parse response
	var output ListFeedsOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.Count != 1 {
		t.Errorf("expected 1 feed, got %d", output.Count)
	}
	if len(output.Feeds) != 1 {
		t.Fatalf("expected 1 feed in list, got %d", len(output.Feeds))
	}
	if output.Feeds[0].URL != "https://example.com/feed.xml" {
		t.Errorf("expected URL 'https://example.com/feed.xml', got %q", output.Feeds[0].URL)
	}
}

func TestHandleListEntries(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and entries
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	now := time.Now()
	entry1 := charm.NewEntry(feed.ID, "guid-1", "Entry 1")
	entry1.PublishedAt = &now

	entry2 := charm.NewEntry(feed.ID, "guid-2", "Entry 2")
	entry2.PublishedAt = &now
	entry2.Read = true

	if err := client.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := client.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Test: all entries
	req := mcp.CallToolRequest{}
	result, err := s.handleListEntries(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListEntries: %v", err)
	}

	var output ListEntriesOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.Count != 2 {
		t.Errorf("expected 2 entries, got %d", output.Count)
	}

	// Test: unread only
	reqUnread := mcp.CallToolRequest{}
	reqUnread.Params.Arguments = map[string]interface{}{"unread_only": true}
	result, err = s.handleListEntries(context.Background(), reqUnread)
	if err != nil {
		t.Fatalf("handleListEntries (unread): %v", err)
	}

	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.Count != 1 {
		t.Errorf("expected 1 unread entry, got %d", output.Count)
	}
}

func TestHandleMarkRead(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and entry
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := charm.NewEntry(feed.ID, "guid-1", "Test Entry")
	if err := client.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Verify initially unread
	got, _ := client.GetEntry(entry.ID)
	if got.Read {
		t.Error("expected entry to be unread initially")
	}

	// Mark read
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"entry_id": entry.ID}
	result, err := s.handleMarkRead(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMarkRead: %v", err)
	}

	var output EntryOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if !output.Read {
		t.Error("expected entry to be marked as read")
	}
	if output.ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}

	// Verify in store
	got, _ = client.GetEntry(entry.ID)
	if !got.Read {
		t.Error("expected entry to be read in store")
	}
}

func TestHandleMarkUnread(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and read entry
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := charm.NewEntry(feed.ID, "guid-1", "Test Entry")
	entry.Read = true
	now := time.Now()
	entry.ReadAt = &now
	if err := client.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Mark unread
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"entry_id": entry.ID}
	result, err := s.handleMarkUnread(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMarkUnread: %v", err)
	}

	var output EntryOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.Read {
		t.Error("expected entry to be marked as unread")
	}
	if output.ReadAt != nil {
		t.Error("expected ReadAt to be nil")
	}
}

func TestHandleGetEntry(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and entry
	feed := charm.NewFeed("https://example.com/feed.xml")
	title := "Test Feed"
	feed.Title = &title
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := charm.NewEntry(feed.ID, "guid-1", "Test Entry")
	content := "<p>Hello <b>world</b></p>"
	entry.Content = &content
	link := "https://example.com/post/1"
	entry.Link = &link
	if err := client.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Get entry
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"entry_id": entry.ID}
	result, err := s.handleGetEntry(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetEntry: %v", err)
	}

	var output GetEntryOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.ID != entry.ID {
		t.Errorf("expected ID %q, got %q", entry.ID, output.ID)
	}
	if output.FeedTitle != "Test Feed" {
		t.Errorf("expected FeedTitle 'Test Feed', got %q", output.FeedTitle)
	}
	if output.Link == nil || *output.Link != link {
		t.Errorf("expected Link %q, got %v", link, output.Link)
	}
	// Content should be converted to markdown
	if output.Content == nil {
		t.Error("expected Content to be set")
	}
}

func TestHandleGetEntryByPrefix(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and entry
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	entry := charm.NewEntry(feed.ID, "guid-1", "Test Entry")
	if err := client.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Get by 8-char prefix
	prefix := entry.ID[:8]
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"entry_id": prefix}
	result, err := s.handleGetEntry(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetEntry (prefix): %v", err)
	}

	var output GetEntryOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if output.ID != entry.ID {
		t.Errorf("expected ID %q, got %q", entry.ID, output.ID)
	}
}

func TestHandleBulkMarkRead(t *testing.T) {
	s, client, _ := testServer(t)

	// Create feed and entries
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	entry1 := charm.NewEntry(feed.ID, "guid-1", "Today")
	entry1.PublishedAt = &now

	entry2 := charm.NewEntry(feed.ID, "guid-2", "Yesterday")
	entry2.PublishedAt = &yesterday

	entry3 := charm.NewEntry(feed.ID, "guid-3", "Two days ago")
	entry3.PublishedAt = &twoDaysAgo

	if err := client.CreateEntry(entry1); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := client.CreateEntry(entry2); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if err := client.CreateEntry(entry3); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Bulk mark entries before yesterday as read
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"before": "yesterday"}
	result, err := s.handleBulkMarkRead(context.Background(), req)
	if err != nil {
		t.Fatalf("handleBulkMarkRead: %v", err)
	}

	var output BulkMarkReadOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	// Should mark entries older than yesterday's start
	if output.Count < 1 {
		t.Errorf("expected at least 1 entry marked, got %d", output.Count)
	}
}

func TestHandleAddFeedValidation(t *testing.T) {
	s, _, _ := testServer(t)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Note: we use unique URLs that aren't in the initial OPML
		{"valid http", "http://newsite.com/feed.xml", false},
		{"valid https", "https://newsite2.com/feed.xml", false},
		{"missing scheme", "example.com/feed.xml", true},
		{"ftp scheme", "ftp://example.com/feed.xml", true},
		{"no host", "https:///feed.xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]interface{}{"url": tt.url}
			_, err := s.handleAddFeed(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAddFeed(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestHandleRemoveFeed(t *testing.T) {
	s, client, _ := testServer(t)

	// Add a feed first
	feed := charm.NewFeed("https://example.com/feed.xml")
	if err := client.CreateFeed(feed); err != nil {
		t.Fatalf("CreateFeed: %v", err)
	}

	// Add some entries
	entry := charm.NewEntry(feed.ID, "guid-1", "Test Entry")
	if err := client.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	// Remove feed
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"url": "https://example.com/feed.xml"}
	result, err := s.handleRemoveFeed(context.Background(), req)
	if err != nil {
		t.Fatalf("handleRemoveFeed: %v", err)
	}

	var output RemoveFeedOutput
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if !output.Success {
		t.Error("expected success to be true")
	}

	// Verify feed is gone
	_, err = client.GetFeed(feed.ID)
	if err == nil {
		t.Error("expected error getting deleted feed")
	}

	// Verify entries are gone (cascade delete)
	entries, _ := client.ListEntries(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after cascade delete, got %d", len(entries))
	}
}

func TestResourceData(t *testing.T) {
	// Test ResourceData and ResourceMetadata structs
	now := time.Now()
	rd := ResourceData{
		Metadata: ResourceMetadata{
			Timestamp:   now,
			Count:       5,
			ResourceURI: "digest://test",
			Filters:     map[string]any{"key": "value"},
		},
		Data:  []string{"a", "b", "c"},
		Links: map[string]string{"self": "digest://test"},
	}

	data, err := json.Marshal(rd)
	if err != nil {
		t.Fatalf("marshal ResourceData: %v", err)
	}

	var decoded ResourceData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal ResourceData: %v", err)
	}

	if decoded.Metadata.Count != 5 {
		t.Errorf("expected Count 5, got %d", decoded.Metadata.Count)
	}
	if decoded.Metadata.ResourceURI != "digest://test" {
		t.Errorf("expected ResourceURI 'digest://test', got %q", decoded.Metadata.ResourceURI)
	}
}

func TestStatsData(t *testing.T) {
	// Test stats types
	now := time.Now()
	stats := StatsData{
		Summary: StatsSummary{
			TotalFeeds:   5,
			TotalEntries: 100,
			UnreadCount:  25,
		},
		ByFeed: []FeedStats{
			{
				FeedID:      "feed-1",
				FeedTitle:   "Test Feed",
				FeedURL:     "https://example.com/feed.xml",
				EntryCount:  20,
				UnreadCount: 5,
				LastFetched: &now,
				ErrorCount:  0,
				HasErrors:   false,
			},
		},
		LastSync: &SyncInfo{
			LastFetchedAt: &now,
			FeedID:        "feed-1",
			FeedTitle:     "Test Feed",
		},
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("marshal StatsData: %v", err)
	}

	var decoded StatsData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal StatsData: %v", err)
	}

	if decoded.Summary.TotalFeeds != 5 {
		t.Errorf("expected TotalFeeds 5, got %d", decoded.Summary.TotalFeeds)
	}
	if len(decoded.ByFeed) != 1 {
		t.Fatalf("expected 1 FeedStats, got %d", len(decoded.ByFeed))
	}
	if decoded.ByFeed[0].FeedTitle != "Test Feed" {
		t.Errorf("expected FeedTitle 'Test Feed', got %q", decoded.ByFeed[0].FeedTitle)
	}
}

func TestInputOutputTypes(t *testing.T) {
	// Test that all input/output types marshal correctly
	now := time.Now()

	tests := []struct {
		name string
		v    interface{}
	}{
		{"ListFeedsInput", ListFeedsInput{}},
		{"FeedOutput", FeedOutput{ID: "test", URL: "http://test.com", CreatedAt: now}},
		{"ListFeedsOutput", ListFeedsOutput{Feeds: []FeedOutput{}, Count: 0, Folders: []string{}}},
		{"AddFeedInput", AddFeedInput{URL: "http://test.com"}},
		{"RemoveFeedInput", RemoveFeedInput{URL: "http://test.com"}},
		{"RemoveFeedOutput", RemoveFeedOutput{Success: true, Message: "ok", URL: "http://test.com"}},
		{"MoveFeedInput", MoveFeedInput{URL: "http://test.com", Folder: "News"}},
		{"MoveFeedOutput", MoveFeedOutput{Success: true, Message: "ok", URL: "http://test.com"}},
		{"SyncFeedsInput", SyncFeedsInput{}},
		{"SyncResult", SyncResult{FeedID: "test", FeedTitle: "Test", NewEntries: 5}},
		{"SyncFeedsOutput", SyncFeedsOutput{Results: []SyncResult{}, TotalFeeds: 0}},
		{"ListEntriesInput", ListEntriesInput{}},
		{"EntryOutput", EntryOutput{ID: "test", FeedID: "feed", Read: false, CreatedAt: now}},
		{"ListEntriesOutput", ListEntriesOutput{Entries: []EntryOutput{}, Count: 0}},
		{"MarkReadInput", MarkReadInput{EntryID: "test"}},
		{"MarkUnreadInput", MarkUnreadInput{EntryID: "test"}},
		{"BulkMarkReadInput", BulkMarkReadInput{Before: "yesterday"}},
		{"BulkMarkReadOutput", BulkMarkReadOutput{Count: 5, Before: now, Message: "ok"}},
		{"GetEntryInput", GetEntryInput{EntryID: "test"}},
		{"GetEntryOutput", GetEntryOutput{ID: "test", FeedID: "feed", CreatedAt: now}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.v)
			if err != nil {
				t.Errorf("marshal %s: %v", tt.name, err)
			}
			if len(data) == 0 {
				t.Errorf("empty JSON for %s", tt.name)
			}
		})
	}
}

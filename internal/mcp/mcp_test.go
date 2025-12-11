// ABOUTME: Tests for MCP server tools, resources, and input validation
// ABOUTME: Validates tool parameter handling, resource serialization, and edge cases

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/opml"
	"github.com/mark3labs/mcp-go/mcp"
)

// Test helpers

func setupTestServer(t *testing.T) (*Server, *sql.DB, string) {
	t.Helper()

	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}

	// Create temp OPML file
	opmlPath := filepath.Join(tmpDir, "test.opml")
	doc := opml.NewDocument("Test OPML")
	if err := doc.WriteFile(opmlPath); err != nil {
		t.Fatalf("failed to write test OPML: %v", err)
	}

	// Load OPML
	doc, err = opml.ParseFile(opmlPath)
	if err != nil {
		t.Fatalf("failed to load test OPML: %v", err)
	}

	server := NewServer(conn, doc, opmlPath)
	return server, conn, opmlPath
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// marshalToMap converts a struct to map[string]interface{} for test input
func marshalToMap(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	inputJSON, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}
	var inputMap map[string]interface{}
	if err := json.Unmarshal(inputJSON, &inputMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	return inputMap
}

// Tool Input Validation Tests

func TestHandleListEntries_NegativeOffset(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create test feed and entry
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create test feed: %v", err)
	}

	entry := models.NewEntry(feed.ID, "guid1", "Test Entry")
	if err := db.CreateEntry(conn, entry); err != nil {
		t.Fatalf("failed to create test entry: %v", err)
	}

	// Test negative offset
	negativeOffset := -10
	input := ListEntriesInput{
		Offset: intPtr(negativeOffset),
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleListEntries(context.Background(), req)

	if err == nil {
		t.Error("expected error for negative offset, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for negative offset, got %v", result)
	}
	if err.Error() != "offset must be non-negative, got -10" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHandleListEntries_NegativeLimit(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Test negative limit
	negativeLimit := -5
	input := ListEntriesInput{
		Limit: intPtr(negativeLimit),
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleListEntries(context.Background(), req)

	if err == nil {
		t.Error("expected error for negative limit, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for negative limit, got %v", result)
	}
	if err.Error() != "limit must be non-negative, got -5" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHandleListEntries_ValidOffset(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create test feed and multiple entries
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create test feed: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := models.NewEntry(feed.ID, "guid"+string(rune('0'+i)), "Entry "+string(rune('0'+i)))
		if err := db.CreateEntry(conn, entry); err != nil {
			t.Fatalf("failed to create test entry: %v", err)
		}
	}

	// Test with valid offset
	input := ListEntriesInput{
		Offset: intPtr(2),
		Limit:  intPtr(2),
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleListEntries(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListEntries failed: %v", err)
	}

	// Parse result
	var output ListEntriesOutput
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// Should have 2 entries (limit) starting from offset 2
	if output.Count != 2 {
		t.Errorf("expected 2 entries with offset 2 and limit 2, got %d", output.Count)
	}

	// Check filters are recorded
	if output.Filters["offset"].(float64) != 2 {
		t.Errorf("expected offset filter to be 2, got %v", output.Filters["offset"])
	}
	if output.Filters["limit"].(float64) != 2 {
		t.Errorf("expected limit filter to be 2, got %v", output.Filters["limit"])
	}
}

func TestHandleListEntries_InvalidSinceDate(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := ListEntriesInput{
		Since: strPtr("invalid-date-format"),
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleListEntries(context.Background(), req)

	if err == nil {
		t.Error("expected error for invalid since date, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for invalid date, got %v", result)
	}
}

func TestHandleMoveFeed_InvalidURL(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	tests := []struct {
		name        string
		url         string
		folder      string
		expectedErr string
	}{
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			folder:      "test",
			expectedErr: "feed URL must use http or https scheme",
		},
		{
			name:        "wrong scheme",
			url:         "ftp://example.com/feed.xml",
			folder:      "test",
			expectedErr: "feed URL must use http or https scheme, got: ftp",
		},
		{
			name:        "no host",
			url:         "https://",
			folder:      "test",
			expectedErr: "feed URL must have a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := MoveFeedInput{
				URL:    tt.url,
				Folder: tt.folder,
			}
			inputMap := marshalToMap(t, input)

			req := mcp.CallToolRequest{}
			req.Params.Arguments = inputMap

			result, err := server.handleMoveFeed(context.Background(), req)

			if err == nil {
				t.Error("expected error, got nil")
			}
			if result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
			if err != nil && err.Error() != tt.expectedErr && !contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestHandleMoveFeed_FeedNotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := MoveFeedInput{
		URL:    "https://example.com/nonexistent.xml",
		Folder: "test",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleMoveFeed(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent feed, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "feed not found") {
		t.Errorf("expected 'feed not found' error, got: %v", err)
	}
}

func TestHandleMoveFeed_SameFolder(t *testing.T) {
	server, conn, opmlPath := setupTestServer(t)
	defer conn.Close()

	// Create feed in database and OPML
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create feed: %v", err)
	}

	doc, _ := opml.ParseFile(opmlPath)
	if err := doc.AddFeed(feed.URL, "Test Feed", "testfolder"); err != nil {
		t.Fatalf("failed to add feed to OPML: %v", err)
	}
	if err := doc.WriteFile(opmlPath); err != nil {
		t.Fatalf("failed to write OPML: %v", err)
	}

	// Reload server with updated OPML
	doc, _ = opml.ParseFile(opmlPath)
	server.opmlDoc = doc

	// Try to move to same folder
	input := MoveFeedInput{
		URL:    feed.URL,
		Folder: "testfolder",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleMoveFeed(context.Background(), req)
	if err != nil {
		t.Fatalf("handleMoveFeed failed: %v", err)
	}

	// Parse result
	var output MoveFeedOutput
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if !output.Success {
		t.Error("expected success for same-folder move")
	}
	if !contains(output.Message, "already in") {
		t.Errorf("expected message about already being in folder, got: %s", output.Message)
	}
	if output.OldFolder != "testfolder" || output.NewFolder != "testfolder" {
		t.Errorf("expected both folders to be 'testfolder', got old=%s new=%s", output.OldFolder, output.NewFolder)
	}
}

func TestHandleAddFeed_InvalidURL(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	tests := []struct {
		name        string
		url         string
		expectedErr string
	}{
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectedErr: "feed URL must use http or https scheme",
		},
		{
			name:        "wrong scheme",
			url:         "ftp://example.com/feed.xml",
			expectedErr: "feed URL must use http or https scheme",
		},
		{
			name:        "no host",
			url:         "https://",
			expectedErr: "feed URL must have a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := AddFeedInput{
				URL: tt.url,
			}
			inputMap := marshalToMap(t, input)

			req := mcp.CallToolRequest{}
			req.Params.Arguments = inputMap

			result, err := server.handleAddFeed(context.Background(), req)

			if err == nil {
				t.Error("expected error, got nil")
			}
			if result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
			if !contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestHandleAddFeed_DuplicateFeed(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create feed first
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create feed: %v", err)
	}

	// Try to add same feed again
	input := AddFeedInput{
		URL: feed.URL,
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleAddFeed(context.Background(), req)

	if err == nil {
		t.Error("expected error for duplicate feed, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "feed already exists") {
		t.Errorf("expected 'feed already exists' error, got: %v", err)
	}
}

func TestHandleRemoveFeed_NotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := RemoveFeedInput{
		URL: "https://example.com/nonexistent.xml",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleRemoveFeed(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent feed, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "feed not found") {
		t.Errorf("expected 'feed not found' error, got: %v", err)
	}
}

func TestHandleGetEntry_NotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := GetEntryInput{
		EntryID: "nonexistent-id",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleGetEntry(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent entry, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "entry not found") {
		t.Errorf("expected 'entry not found' error, got: %v", err)
	}
}

func TestHandleMarkRead_NotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := MarkReadInput{
		EntryID: "nonexistent-id",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleMarkRead(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent entry, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "entry not found") {
		t.Errorf("expected 'entry not found' error, got: %v", err)
	}
}

func TestHandleMarkUnread_NotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := MarkUnreadInput{
		EntryID: "nonexistent-id",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleMarkUnread(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent entry, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "entry not found") {
		t.Errorf("expected 'entry not found' error, got: %v", err)
	}
}

func TestHandleBulkMarkRead_InvalidDate(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	input := BulkMarkReadInput{
		Before: "invalid-date",
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleBulkMarkRead(context.Background(), req)

	if err == nil {
		t.Error("expected error for invalid date, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "invalid before value") {
		t.Errorf("expected 'invalid before value' error, got: %v", err)
	}
}

func TestHandleSyncFeeds_NoFeeds(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = make(map[string]interface{})

	result, err := server.handleSyncFeeds(context.Background(), req)

	if err == nil {
		t.Error("expected error when no feeds exist, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "no feeds found") {
		t.Errorf("expected 'no feeds found' error, got: %v", err)
	}
}

func TestHandleSyncFeeds_SpecificFeedNotFound(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create a feed so we don't get "no feeds" error
	feed := models.NewFeed("https://example.com/feed.xml")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create feed: %v", err)
	}

	// Try to sync a different feed
	input := SyncFeedsInput{
		URL: strPtr("https://example.com/other.xml"),
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleSyncFeeds(context.Background(), req)

	if err == nil {
		t.Error("expected error for nonexistent feed, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !contains(err.Error(), "feed not found") {
		t.Errorf("expected 'feed not found' error, got: %v", err)
	}
}

// Resource Serialization Tests

func TestCalculateStats_EmptyDatabase(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	stats, err := server.calculateStats()
	if err != nil {
		t.Fatalf("calculateStats failed: %v", err)
	}

	if stats.Summary.TotalFeeds != 0 {
		t.Errorf("expected 0 feeds, got %d", stats.Summary.TotalFeeds)
	}
	if stats.Summary.TotalEntries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Summary.TotalEntries)
	}
	if stats.Summary.UnreadCount != 0 {
		t.Errorf("expected 0 unread, got %d", stats.Summary.UnreadCount)
	}
	if len(stats.ByFeed) != 0 {
		t.Errorf("expected 0 feed stats, got %d", len(stats.ByFeed))
	}
	if stats.LastSync != nil {
		t.Errorf("expected nil LastSync, got %v", stats.LastSync)
	}
}

func TestCalculateStats_WithData(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create feeds
	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed1.Title = strPtr("Feed 1")
	if err := db.CreateFeed(conn, feed1); err != nil {
		t.Fatalf("failed to create feed1: %v", err)
	}

	feed2 := models.NewFeed("https://example.com/feed2.xml")
	feed2.Title = strPtr("Feed 2")
	if err := db.CreateFeed(conn, feed2); err != nil {
		t.Fatalf("failed to create feed2: %v", err)
	}

	// Create entries
	for i := 0; i < 3; i++ {
		entry := models.NewEntry(feed1.ID, "guid1-"+string(rune('0'+i)), "Entry 1-"+string(rune('0'+i)))
		if err := db.CreateEntry(conn, entry); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		entry := models.NewEntry(feed2.ID, "guid2-"+string(rune('0'+i)), "Entry 2-"+string(rune('0'+i)))
		if err := db.CreateEntry(conn, entry); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
	}

	// Mark one entry as read
	entries, _ := db.ListEntries(conn, nil, nil, nil, nil, nil, intPtr(1), nil)
	if len(entries) > 0 {
		_ = db.MarkEntryRead(conn, entries[0].ID)
	}

	stats, err := server.calculateStats()
	if err != nil {
		t.Fatalf("calculateStats failed: %v", err)
	}

	if stats.Summary.TotalFeeds != 2 {
		t.Errorf("expected 2 feeds, got %d", stats.Summary.TotalFeeds)
	}
	if stats.Summary.TotalEntries != 5 {
		t.Errorf("expected 5 entries, got %d", stats.Summary.TotalEntries)
	}
	if stats.Summary.UnreadCount != 4 {
		t.Errorf("expected 4 unread (5 total - 1 marked read), got %d", stats.Summary.UnreadCount)
	}
	if len(stats.ByFeed) != 2 {
		t.Errorf("expected 2 feed stats, got %d", len(stats.ByFeed))
	}

	// Verify per-feed stats
	for _, feedStat := range stats.ByFeed {
		switch feedStat.FeedID {
		case feed1.ID:
			if feedStat.EntryCount != 3 {
				t.Errorf("expected 3 entries for feed1, got %d", feedStat.EntryCount)
			}
			if feedStat.FeedTitle != "Feed 1" {
				t.Errorf("expected 'Feed 1', got %s", feedStat.FeedTitle)
			}
		case feed2.ID:
			if feedStat.EntryCount != 2 {
				t.Errorf("expected 2 entries for feed2, got %d", feedStat.EntryCount)
			}
			if feedStat.FeedTitle != "Feed 2" {
				t.Errorf("expected 'Feed 2', got %s", feedStat.FeedTitle)
			}
		}
	}
}

func TestCalculateStats_JSONSerialization(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create feed with data
	feed := models.NewFeed("https://example.com/feed.xml")
	feed.Title = strPtr("Test Feed")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create feed: %v", err)
	}

	entry := models.NewEntry(feed.ID, "guid1", "Test Entry")
	if err := db.CreateEntry(conn, entry); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	stats, err := server.calculateStats()
	if err != nil {
		t.Fatalf("calculateStats failed: %v", err)
	}

	// Test JSON serialization
	jsonBytes, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal stats to JSON: %v", err)
	}

	// Verify it's valid JSON
	var decoded StatsData
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("failed to unmarshal stats JSON: %v", err)
	}

	// Verify decoded data matches
	if decoded.Summary.TotalFeeds != stats.Summary.TotalFeeds {
		t.Errorf("decoded TotalFeeds doesn't match: expected %d, got %d",
			stats.Summary.TotalFeeds, decoded.Summary.TotalFeeds)
	}
	if decoded.Summary.TotalEntries != stats.Summary.TotalEntries {
		t.Errorf("decoded TotalEntries doesn't match: expected %d, got %d",
			stats.Summary.TotalEntries, decoded.Summary.TotalEntries)
	}
}

func TestParseDateString_ValidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"period today", "today", true},
		{"period yesterday", "yesterday", true},
		{"period week", "week", true},
		{"period month", "month", true},
		{"ISO date", "2024-01-15", true},
		{"RFC3339", "2024-01-15T10:30:00Z", true},
		{"invalid", "not-a-date", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDateString(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("expected valid parse for '%s', got error: %v", tt.input, err)
				}
				if result.IsZero() {
					t.Errorf("expected non-zero time for '%s'", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("expected error for '%s', got nil", tt.input)
				}
			}
		})
	}
}

func TestFormatFolder(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "root level"},
		{"tech", "'tech'"},
		{"Tech Blogs", "'Tech Blogs'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatFolder(tt.input)
			if result != tt.expected {
				t.Errorf("formatFolder(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Integration Tests

func TestHandleListFeeds_Integration(t *testing.T) {
	server, conn, opmlPath := setupTestServer(t)
	defer conn.Close()

	// Create feeds in database and OPML
	feed1 := models.NewFeed("https://example.com/feed1.xml")
	feed1.Title = strPtr("Feed 1")
	if err := db.CreateFeed(conn, feed1); err != nil {
		t.Fatalf("failed to create feed1: %v", err)
	}

	feed2 := models.NewFeed("https://example.com/feed2.xml")
	feed2.Title = strPtr("Feed 2")
	if err := db.CreateFeed(conn, feed2); err != nil {
		t.Fatalf("failed to create feed2: %v", err)
	}

	doc, _ := opml.ParseFile(opmlPath)
	_ = doc.AddFeed(feed1.URL, "Feed 1", "tech")
	_ = doc.AddFeed(feed2.URL, "Feed 2", "news")
	_ = doc.WriteFile(opmlPath)

	// Reload server with updated OPML
	doc, _ = opml.ParseFile(opmlPath)
	server.opmlDoc = doc

	// Test list_feeds
	req := mcp.CallToolRequest{}
	result, err := server.handleListFeeds(context.Background(), req)
	if err != nil {
		t.Fatalf("handleListFeeds failed: %v", err)
	}

	// Parse result
	var output ListFeedsOutput
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output.Count != 2 {
		t.Errorf("expected 2 feeds, got %d", output.Count)
	}
	if len(output.Folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(output.Folders))
	}

	// Verify folders
	folderMap := make(map[string]bool)
	for _, folder := range output.Folders {
		folderMap[folder] = true
	}
	if !folderMap["tech"] || !folderMap["news"] {
		t.Errorf("expected folders 'tech' and 'news', got %v", output.Folders)
	}
}

func TestHandleGetEntry_WithPrefix(t *testing.T) {
	server, conn, _ := setupTestServer(t)
	defer conn.Close()

	// Create test feed and entry
	feed := models.NewFeed("https://example.com/feed.xml")
	feed.Title = strPtr("Test Feed")
	if err := db.CreateFeed(conn, feed); err != nil {
		t.Fatalf("failed to create feed: %v", err)
	}

	entry := models.NewEntry(feed.ID, "guid1", "Test Entry")
	entry.Content = strPtr("<p>Test content</p>")
	if err := db.CreateEntry(conn, entry); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Test with prefix (first 8 chars)
	prefix := entry.ID[:8]
	input := GetEntryInput{
		EntryID: prefix,
	}
	inputMap := marshalToMap(t, input)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = inputMap

	result, err := server.handleGetEntry(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetEntry failed with prefix: %v", err)
	}

	// Parse result
	var output GetEntryOutput
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output.ID != entry.ID {
		t.Errorf("expected entry ID %s, got %s", entry.ID, output.ID)
	}
	if output.FeedTitle != "Test Feed" {
		t.Errorf("expected feed title 'Test Feed', got %s", output.FeedTitle)
	}
	// Content should be converted to markdown
	if output.Content == nil {
		t.Error("expected content to be present")
	}
}

// Helper function

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

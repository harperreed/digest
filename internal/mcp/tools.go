// ABOUTME: MCP tool definitions and handlers for feed and entry operations
// ABOUTME: Provides tools for managing RSS feeds, syncing content, and tracking read/unread entries

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/harper/digest/internal/content"
	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/models"
	"github.com/harper/digest/internal/parse"
	"github.com/harper/digest/internal/timeutil"
	"github.com/mark3labs/mcp-go/mcp"
)

// Type definitions for input/output structures

type ListFeedsInput struct{}

type FeedOutput struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Title         *string    `json:"title,omitempty"`
	Folder        string     `json:"folder,omitempty"`
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	ErrorCount    int        `json:"error_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ListFeedsOutput struct {
	Feeds   []FeedOutput `json:"feeds"`
	Count   int          `json:"count"`
	Folders []string     `json:"folders"`
}

type AddFeedInput struct {
	URL    string  `json:"url"`
	Title  *string `json:"title,omitempty"`
	Folder *string `json:"folder,omitempty"`
}

type RemoveFeedInput struct {
	URL string `json:"url"`
}

type RemoveFeedOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

type MoveFeedInput struct {
	URL    string `json:"url"`
	Folder string `json:"folder"`
}

type MoveFeedOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	URL       string `json:"url"`
	OldFolder string `json:"old_folder"`
	NewFolder string `json:"new_folder"`
}

type SyncFeedsInput struct {
	URL   *string `json:"url,omitempty"`
	Force *bool   `json:"force,omitempty"`
}

type SyncResult struct {
	FeedID     string  `json:"feed_id"`
	FeedTitle  string  `json:"feed_title"`
	NewEntries int     `json:"new_entries"`
	WasCached  bool    `json:"was_cached"`
	Error      *string `json:"error,omitempty"`
}

type SyncFeedsOutput struct {
	Results     []SyncResult `json:"results"`
	TotalFeeds  int          `json:"total_feeds"`
	TotalNew    int          `json:"total_new"`
	TotalCached int          `json:"total_cached"`
	TotalErrors int          `json:"total_errors"`
}

type ListEntriesInput struct {
	FeedID     *string `json:"feed_id,omitempty"`
	UnreadOnly *bool   `json:"unread_only,omitempty"`
	Since      *string `json:"since,omitempty"`
	Until      *string `json:"until,omitempty"`
	Limit      *int    `json:"limit,omitempty"`
	Offset     *int    `json:"offset,omitempty"`
}

type EntryOutput struct {
	ID          string     `json:"id"`
	FeedID      string     `json:"feed_id"`
	Title       *string    `json:"title,omitempty"`
	Link        *string    `json:"link,omitempty"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Read        bool       `json:"read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ListEntriesOutput struct {
	Entries []EntryOutput  `json:"entries"`
	Count   int            `json:"count"`
	Filters map[string]any `json:"filters"`
}

type MarkReadInput struct {
	EntryID string `json:"entry_id"`
}

type MarkUnreadInput struct {
	EntryID string `json:"entry_id"`
}

type BulkMarkReadInput struct {
	Before string `json:"before"`
}

type BulkMarkReadOutput struct {
	Count   int64     `json:"count"`
	Before  time.Time `json:"before"`
	Message string    `json:"message"`
}

type GetEntryInput struct {
	EntryID string `json:"entry_id"`
}

type GetEntryOutput struct {
	ID          string     `json:"id"`
	FeedID      string     `json:"feed_id"`
	FeedTitle   string     `json:"feed_title,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Link        *string    `json:"link,omitempty"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Content     *string    `json:"content,omitempty"`
	Read        bool       `json:"read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Tool registration

func (s *Server) registerTools() {
	s.registerListFeedsTool()
	s.registerAddFeedTool()
	s.registerRemoveFeedTool()
	s.registerMoveFeedTool()
	s.registerSyncFeedsTool()
	s.registerListEntriesTool()
	s.registerGetEntryTool()
	s.registerMarkReadTool()
	s.registerMarkUnreadTool()
	s.registerBulkMarkReadTool()
}

func (s *Server) registerListFeedsTool() {
	tool := mcp.Tool{
		Name:        "list_feeds",
		Description: "Retrieve all RSS/Atom feeds from the OPML subscription list. Returns a complete list of feeds with their metadata including URLs, titles, folders, last fetch times, and error states. Use this to see all subscribed feeds before performing other operations.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
	s.mcpServer.AddTool(tool, s.handleListFeeds)
}

func (s *Server) registerAddFeedTool() {
	tool := mcp.Tool{
		Name:        "add_feed",
		Description: "Add a new RSS/Atom feed to the subscription list. The feed is added to both the database and the OPML file. Optionally specify a title and folder for organization. If no title is provided, it will be fetched from the feed on first sync. Returns the created feed with its unique ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The feed URL (RSS or Atom). Example: 'https://example.com/feed.xml'",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Optional feed title. If not provided, will be fetched from the feed metadata. Example: 'My Favorite Blog'",
				},
				"folder": map[string]interface{}{
					"type":        "string",
					"description": "Optional folder/category for organization in OPML. Example: 'Tech Blogs'",
				},
			},
			Required: []string{"url"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleAddFeed)
}

func (s *Server) registerRemoveFeedTool() {
	tool := mcp.Tool{
		Name:        "remove_feed",
		Description: "Remove a feed from the subscription list. This removes the feed from both the database and the OPML file. All entries associated with this feed will also be deleted due to CASCADE constraints. This action cannot be undone.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The feed URL to remove. Must match exactly. Example: 'https://example.com/feed.xml'",
				},
			},
			Required: []string{"url"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleRemoveFeed)
}

func (s *Server) registerMoveFeedTool() {
	tool := mcp.Tool{
		Name:        "move_feed",
		Description: "Move a feed to a different folder/category in the OPML file. Use this to reorganize feeds after they've been added. If the target folder doesn't exist, it will be created. Use an empty string for folder to move to the root level.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The feed URL to move. Must match exactly. Example: 'https://example.com/feed.xml'",
				},
				"folder": map[string]interface{}{
					"type":        "string",
					"description": "Target folder name. Use empty string '' to move to root level. Example: 'Tech Blogs'",
				},
			},
			Required: []string{"url", "folder"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleMoveFeed)
}

func (s *Server) registerSyncFeedsTool() {
	tool := mcp.Tool{
		Name:        "sync_feeds",
		Description: "Fetch new entries from RSS/Atom feeds. If url is provided, syncs only that specific feed. Otherwise, syncs all subscribed feeds. Uses HTTP caching headers (ETag, Last-Modified) to avoid unnecessary downloads. Set force=true to ignore cache and fetch unconditionally. Returns a summary of new entries, cached responses, and any errors.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "Optional feed URL to sync only that specific feed. If omitted, syncs all feeds. Example: 'https://example.com/feed.xml'",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, ignores HTTP cache headers and forces a fresh fetch. Default: false",
				},
			},
		},
	}
	s.mcpServer.AddTool(tool, s.handleSyncFeeds)
}

func (s *Server) registerListEntriesTool() {
	tool := mcp.Tool{
		Name:        "list_entries",
		Description: "Retrieve feed entries with optional filtering. Use 'since' with values like 'today', 'yesterday', 'week', 'month' to get recent entries (e.g., since='today' for today's entries). Filter by feed_id for a specific feed, unread_only for unread entries, and limit to control results. All filters are optional and can be combined. Returns entries sorted by published date (newest first). Use get_entry to read full article content.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"feed_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional feed ID to filter entries. Only entries from this feed will be returned. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
				"unread_only": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, returns only unread entries. If false or omitted, returns all entries. Example: true",
				},
				"since": map[string]interface{}{
					"type":        "string",
					"description": "Only return entries published on or after this date. Accepts shortcuts: 'today', 'yesterday', 'week', 'month', or ISO date (YYYY-MM-DD). Example: 'today' for today's entries",
				},
				"until": map[string]interface{}{
					"type":        "string",
					"description": "Only return entries published before this date. Accepts: 'today', 'yesterday', 'week', 'month', or ISO date (YYYY-MM-DD). Example: 'today' for yesterday and earlier",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entries to return. If omitted, returns all matching entries. Example: 50",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Number of entries to skip for pagination. Use with limit for paging through results. Example: 20 to skip first 20 entries",
				},
			},
		},
	}
	s.mcpServer.AddTool(tool, s.handleListEntries)
}

func (s *Server) registerGetEntryTool() {
	tool := mcp.Tool{
		Name:        "get_entry",
		Description: "Get the full details of a single entry including its content. Content is converted from HTML to Markdown for better readability. Use this after list_entries to read the full article. Supports both full entry IDs and ID prefixes (first 8 characters).",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"entry_id": map[string]interface{}{
					"type":        "string",
					"description": "The entry ID or ID prefix. Example: 'abc12345' (prefix) or 'abc12345-1234-1234-1234-123456789abc' (full)",
				},
			},
			Required: []string{"entry_id"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleGetEntry)
}

func (s *Server) registerMarkReadTool() {
	tool := mcp.Tool{
		Name:        "mark_read",
		Description: "Mark an entry as read by its ID. Sets the read flag to true and records the current timestamp. Returns the updated entry. Use list_entries to find entry IDs.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"entry_id": map[string]interface{}{
					"type":        "string",
					"description": "The entry ID to mark as read. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			Required: []string{"entry_id"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleMarkRead)
}

func (s *Server) registerMarkUnreadTool() {
	tool := mcp.Tool{
		Name:        "mark_unread",
		Description: "Mark an entry as unread by its ID. Sets the read flag to false and clears the read timestamp. Returns the updated entry. Use list_entries to find entry IDs.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"entry_id": map[string]interface{}{
					"type":        "string",
					"description": "The entry ID to mark as unread. Example: 'abc12345-1234-1234-1234-123456789abc'",
				},
			},
			Required: []string{"entry_id"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleMarkUnread)
}

func (s *Server) registerBulkMarkReadTool() {
	tool := mcp.Tool{
		Name:        "bulk_mark_read",
		Description: "Mark all entries older than a specified period as read. Use this to catch up on older content. Accepts period names (yesterday, week, month) or ISO 8601 dates (YYYY-MM-DD). Returns the count of entries marked as read.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"before": map[string]interface{}{
					"type":        "string",
					"description": "Mark entries published before this date/period as read. Accepts: 'yesterday', 'week', 'month', or YYYY-MM-DD. Example: 'yesterday' or '2024-01-15'",
				},
			},
			Required: []string{"before"},
		},
	}
	s.mcpServer.AddTool(tool, s.handleBulkMarkRead)
}

// Handler implementations

func (s *Server) handleListFeeds(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all feeds from OPML
	opmlFeeds := s.opmlDoc.AllFeeds()
	folders := s.opmlDoc.Folders()

	// Get all feeds from database
	dbFeeds, err := db.ListFeeds(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list feeds from database: %w", err)
	}

	// Create map for quick lookup
	dbFeedMap := make(map[string]*models.Feed)
	for _, feed := range dbFeeds {
		dbFeedMap[feed.URL] = feed
	}

	// Build output by combining OPML and DB data
	feedOutputs := make([]FeedOutput, 0, len(opmlFeeds))
	for _, opmlFeed := range opmlFeeds {
		output := FeedOutput{
			URL:    opmlFeed.URL,
			Folder: opmlFeed.Folder,
		}

		// Add database info if available
		if dbFeed, exists := dbFeedMap[opmlFeed.URL]; exists {
			output.ID = dbFeed.ID
			output.Title = dbFeed.Title
			output.LastFetchedAt = dbFeed.LastFetchedAt
			output.LastError = dbFeed.LastError
			output.ErrorCount = dbFeed.ErrorCount
			output.CreatedAt = dbFeed.CreatedAt
		} else {
			// Feed in OPML but not in DB
			title := opmlFeed.Title
			output.Title = &title
		}

		feedOutputs = append(feedOutputs, output)
	}

	result := ListFeedsOutput{
		Feeds:   feedOutputs,
		Count:   len(feedOutputs),
		Folders: folders,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleAddFeed(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input AddFeedInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate URL format
	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid feed URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("feed URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("feed URL must have a host")
	}

	// Check if feed already exists in database
	existingFeed, err := db.GetFeedByURL(s.db, input.URL)
	if err == nil && existingFeed != nil {
		return nil, fmt.Errorf("feed already exists: %s", input.URL)
	}

	// Create feed in database
	feed := models.NewFeed(input.URL)
	if input.Title != nil {
		feed.Title = input.Title
	}

	if err := db.CreateFeed(s.db, feed); err != nil {
		return nil, fmt.Errorf("failed to create feed in database: %w", err)
	}

	// Add to OPML
	title := input.URL
	if input.Title != nil {
		title = *input.Title
	}
	folder := ""
	if input.Folder != nil {
		folder = *input.Folder
	}

	if err := s.opmlDoc.AddFeed(input.URL, title, folder); err != nil {
		return nil, fmt.Errorf("failed to add feed to OPML: %w", err)
	}

	// Write OPML back to file
	if err := s.opmlDoc.WriteFile(s.opmlPath); err != nil {
		return nil, fmt.Errorf("failed to write OPML file: %w", err)
	}

	output := FeedOutput{
		ID:         feed.ID,
		URL:        feed.URL,
		Title:      feed.Title,
		Folder:     folder,
		ErrorCount: feed.ErrorCount,
		CreatedAt:  feed.CreatedAt,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleRemoveFeed(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input RemoveFeedInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Get feed from database to get ID
	feed, err := db.GetFeedByURL(s.db, input.URL)
	if err != nil {
		return nil, fmt.Errorf("feed not found: %s", input.URL)
	}

	// Delete from database (CASCADE will delete entries)
	if err := db.DeleteFeed(s.db, feed.ID); err != nil {
		return nil, fmt.Errorf("failed to delete feed from database: %w", err)
	}

	// Remove from OPML
	if err := s.opmlDoc.RemoveFeed(input.URL); err != nil {
		return nil, fmt.Errorf("failed to remove feed from OPML: %w", err)
	}

	// Write OPML back to file
	if err := s.opmlDoc.WriteFile(s.opmlPath); err != nil {
		return nil, fmt.Errorf("failed to write OPML file: %w", err)
	}

	output := RemoveFeedOutput{
		Success: true,
		Message: fmt.Sprintf("Feed '%s' and all its entries successfully removed", input.URL),
		URL:     input.URL,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleMoveFeed(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input MoveFeedInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate URL format (consistent with handleAddFeed)
	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid feed URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("feed URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("feed URL must have a host")
	}

	// Verify feed exists in database (consistent with handleRemoveFeed)
	if _, err := db.GetFeedByURL(s.db, input.URL); err != nil {
		return nil, fmt.Errorf("feed not found: %s", input.URL)
	}

	// Find current folder for the feed
	oldFolder := ""
	found := false
	for _, feed := range s.opmlDoc.AllFeeds() {
		if feed.URL == input.URL {
			oldFolder = feed.Folder
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("feed not found in OPML: %s", input.URL)
	}

	// Skip if already in target folder
	if oldFolder == input.Folder {
		output := MoveFeedOutput{
			Success:   true,
			Message:   fmt.Sprintf("Feed is already in %s", formatFolder(oldFolder)),
			URL:       input.URL,
			OldFolder: oldFolder,
			NewFolder: input.Folder,
		}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal output: %w", err)
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}

	// Move the feed
	if err := s.opmlDoc.MoveFeed(input.URL, input.Folder); err != nil {
		return nil, fmt.Errorf("failed to move feed: %w", err)
	}

	// Write OPML back to file
	if err := s.opmlDoc.WriteFile(s.opmlPath); err != nil {
		return nil, fmt.Errorf("failed to write OPML file: %w", err)
	}

	output := MoveFeedOutput{
		Success:   true,
		Message:   fmt.Sprintf("Feed moved from %s to %s", formatFolder(oldFolder), formatFolder(input.Folder)),
		URL:       input.URL,
		OldFolder: oldFolder,
		NewFolder: input.Folder,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleSyncFeeds(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input SyncFeedsInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	force := false
	if input.Force != nil {
		force = *input.Force
	}

	// Get feeds to sync
	feeds, err := db.ListFeeds(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		return nil, fmt.Errorf("no feeds found. Add a feed first using add_feed")
	}

	// Filter to specific URL if provided
	if input.URL != nil {
		filtered := []*models.Feed{}
		for _, feed := range feeds {
			if feed.URL == *input.URL {
				filtered = append(filtered, feed)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("feed not found: %s", *input.URL)
		}
		feeds = filtered
	}

	// Sync each feed
	results := make([]SyncResult, 0, len(feeds))
	totalNew := 0
	totalCached := 0
	totalErrors := 0

	for _, feed := range feeds {
		result := SyncResult{
			FeedID: feed.ID,
			FeedTitle: func() string {
				if feed.Title != nil {
					return *feed.Title
				}
				return feed.URL
			}(),
		}

		newCount, wasCached, err := s.syncFeed(feed, force)
		if err != nil {
			errMsg := err.Error()
			result.Error = &errMsg
			totalErrors++
		} else {
			result.NewEntries = newCount
			result.WasCached = wasCached
			totalNew += newCount
			if wasCached {
				totalCached++
			}
		}

		results = append(results, result)
	}

	output := SyncFeedsOutput{
		Results:     results,
		TotalFeeds:  len(feeds),
		TotalNew:    totalNew,
		TotalCached: totalCached,
		TotalErrors: totalErrors,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleListEntries(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input ListEntriesInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Parse since/until date strings
	var since, until *time.Time
	if input.Since != nil {
		t, err := parseDateString(*input.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since value: %w", err)
		}
		since = &t
	}
	if input.Until != nil {
		t, err := parseDateString(*input.Until)
		if err != nil {
			return nil, fmt.Errorf("invalid until value: %w", err)
		}
		until = &t
	}
	if input.Offset != nil && *input.Offset < 0 {
		return nil, fmt.Errorf("offset must be non-negative, got %d", *input.Offset)
	}
	if input.Limit != nil && *input.Limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative, got %d", *input.Limit)
	}

	// List entries with filters
	entries, err := db.ListEntries(s.db, input.FeedID, nil, input.UnreadOnly, since, until, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	// Build output
	entryOutputs := make([]EntryOutput, 0, len(entries))
	for _, entry := range entries {
		entryOutputs = append(entryOutputs, EntryOutput{
			ID:          entry.ID,
			FeedID:      entry.FeedID,
			Title:       entry.Title,
			Link:        entry.Link,
			Author:      entry.Author,
			PublishedAt: entry.PublishedAt,
			Read:        entry.Read,
			ReadAt:      entry.ReadAt,
			CreatedAt:   entry.CreatedAt,
		})
	}

	// Build applied filters
	filters := make(map[string]any)
	if input.FeedID != nil {
		filters["feed_id"] = *input.FeedID
	}
	if input.UnreadOnly != nil {
		filters["unread_only"] = *input.UnreadOnly
	}
	if since != nil {
		filters["since"] = *since
	}
	if until != nil {
		filters["until"] = *until
	}
	if input.Limit != nil {
		filters["limit"] = *input.Limit
	}
	if input.Offset != nil {
		filters["offset"] = *input.Offset
	}

	output := ListEntriesOutput{
		Entries: entryOutputs,
		Count:   len(entryOutputs),
		Filters: filters,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleGetEntry(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input GetEntryInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Get entry by ID or prefix
	entry, err := db.GetEntryByID(s.db, input.EntryID)
	if err != nil {
		// Try prefix match
		entry, err = db.GetEntryByPrefix(s.db, input.EntryID)
		if err != nil {
			return nil, fmt.Errorf("entry not found: %s", input.EntryID)
		}
	}

	// Get feed for context
	feed, err := db.GetFeedByID(s.db, entry.FeedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	feedTitle := feed.URL
	if feed.Title != nil {
		feedTitle = *feed.Title
	}

	// Convert content to markdown if HTML
	var contentPtr *string
	if entry.Content != nil && *entry.Content != "" {
		markdown := content.ToMarkdown(*entry.Content)
		contentPtr = &markdown
	}

	output := GetEntryOutput{
		ID:          entry.ID,
		FeedID:      entry.FeedID,
		FeedTitle:   feedTitle,
		Title:       entry.Title,
		Link:        entry.Link,
		Author:      entry.Author,
		PublishedAt: entry.PublishedAt,
		Content:     contentPtr,
		Read:        entry.Read,
		ReadAt:      entry.ReadAt,
		CreatedAt:   entry.CreatedAt,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleMarkRead(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input MarkReadInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Verify entry exists
	if _, err := db.GetEntryByID(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("entry not found: %s", input.EntryID)
	}

	// Mark as read
	if err := db.MarkEntryRead(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("failed to mark entry as read: %w", err)
	}

	// Reload entry to get updated read_at
	entry, err := db.GetEntryByID(s.db, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload entry: %w", err)
	}

	output := EntryOutput{
		ID:          entry.ID,
		FeedID:      entry.FeedID,
		Title:       entry.Title,
		Link:        entry.Link,
		Author:      entry.Author,
		PublishedAt: entry.PublishedAt,
		Read:        entry.Read,
		ReadAt:      entry.ReadAt,
		CreatedAt:   entry.CreatedAt,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleMarkUnread(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input MarkUnreadInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Verify entry exists
	if _, err := db.GetEntryByID(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("entry not found: %s", input.EntryID)
	}

	// Mark as unread
	if err := db.MarkEntryUnread(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("failed to mark entry as unread: %w", err)
	}

	// Reload entry to get updated state
	entry, err := db.GetEntryByID(s.db, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("failed to reload entry: %w", err)
	}

	output := EntryOutput{
		ID:          entry.ID,
		FeedID:      entry.FeedID,
		Title:       entry.Title,
		Link:        entry.Link,
		Author:      entry.Author,
		PublishedAt: entry.PublishedAt,
		Read:        entry.Read,
		ReadAt:      entry.ReadAt,
		CreatedAt:   entry.CreatedAt,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// syncFeed is a helper that fetches and processes a single feed
// Returns (newCount, wasCached, error)
func (s *Server) syncFeed(feed *models.Feed, force bool) (int, bool, error) {
	// Get cache headers from feed (skip if force)
	var etag, lastModified *string
	if !force {
		etag = feed.ETag
		lastModified = feed.LastModified
	}

	// Fetch the feed
	result, err := fetch.Fetch(feed.URL, etag, lastModified)
	if err != nil {
		// Update error state in database
		if updateErr := db.UpdateFeedError(s.db, feed.ID, err.Error()); updateErr != nil {
			return 0, false, fmt.Errorf("fetch failed (%v) and error update failed: %w", err, updateErr)
		}
		return 0, false, err
	}

	// Handle 304 Not Modified
	if result.NotModified {
		return 0, true, nil
	}

	// Parse the feed
	parsed, err := parse.Parse(result.Body)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse feed: %v", err)
		if updateErr := db.UpdateFeedError(s.db, feed.ID, errMsg); updateErr != nil {
			return 0, false, fmt.Errorf("parse failed (%v) and error update failed: %w", err, updateErr)
		}
		return 0, false, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Update feed title if empty and persist to database
	titleUpdated := false
	if feed.Title == nil || *feed.Title == "" {
		feed.Title = &parsed.Title
		titleUpdated = true
	}

	// Process entries
	newCount := 0
	for _, parsedEntry := range parsed.Entries {
		// Check if entry already exists
		exists, err := db.EntryExists(s.db, feed.ID, parsedEntry.GUID)
		if err != nil {
			return newCount, false, fmt.Errorf("failed to check entry existence: %w", err)
		}

		if exists {
			continue
		}

		// Create new entry
		entry := models.NewEntry(feed.ID, parsedEntry.GUID, parsedEntry.Title)
		entry.Link = &parsedEntry.Link
		entry.Author = &parsedEntry.Author
		entry.PublishedAt = parsedEntry.PublishedAt
		entry.Content = &parsedEntry.Content

		if err := db.CreateEntry(s.db, entry); err != nil {
			return newCount, false, fmt.Errorf("failed to create entry: %w", err)
		}

		newCount++
	}

	// Update feed fetch state
	fetchedAt := time.Now()
	if err := db.UpdateFeedFetchState(s.db, feed.ID, &result.ETag, &result.LastModified, fetchedAt); err != nil {
		return newCount, false, fmt.Errorf("failed to update feed state: %w", err)
	}

	// If title was updated, persist to database
	if titleUpdated {
		if err := db.UpdateFeed(s.db, feed); err != nil {
			return newCount, false, fmt.Errorf("failed to update feed title: %w", err)
		}
	}

	return newCount, false, nil
}

func (s *Server) handleBulkMarkRead(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input BulkMarkReadInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Parse the before date
	cutoff, err := parseDateString(input.Before)
	if err != nil {
		return nil, fmt.Errorf("invalid before value: %w", err)
	}

	// Mark entries as read
	count, err := db.MarkEntriesReadBefore(s.db, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to mark entries as read: %w", err)
	}

	output := BulkMarkReadOutput{
		Count:  count,
		Before: cutoff,
	}

	if count == 0 {
		output.Message = "No entries to mark as read"
	} else {
		output.Message = fmt.Sprintf("Marked %d entries as read", count)
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// parseDateString parses a date string that can be a period name or ISO date.
func parseDateString(s string) (time.Time, error) {
	// Try period name first
	if t, ok := timeutil.ParsePeriod(s); ok {
		return t, nil
	}

	// Try ISO date format
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date: use yesterday, week, month, today, or YYYY-MM-DD format")
}

// formatFolder returns a human-readable folder name for messages
func formatFolder(folder string) string {
	if folder == "" {
		return "root level"
	}
	return fmt.Sprintf("'%s'", folder)
}

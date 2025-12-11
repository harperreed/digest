// ABOUTME: MCP tool definitions and handlers for feed and entry operations
// ABOUTME: Provides tools for managing RSS feeds, syncing content, and tracking read/unread entries

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
	Limit      *int    `json:"limit,omitempty"`
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

// Tool registration

func (s *Server) registerTools() {
	s.registerListFeedsTool()
	s.registerAddFeedTool()
	s.registerRemoveFeedTool()
	s.registerSyncFeedsTool()
	s.registerListEntriesTool()
	s.registerMarkReadTool()
	s.registerMarkUnreadTool()
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
		Description: "Retrieve feed entries with optional filtering. Filter by feed_id to see entries from a specific feed, unread_only to see only unread entries, and limit to control the number of results. All filters are optional and can be combined. Returns entries sorted by published date (newest first).",
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
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entries to return. If omitted, returns all matching entries. Example: 50",
				},
			},
		},
	}
	s.mcpServer.AddTool(tool, s.handleListEntries)
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

	// List entries with filters
	entries, err := db.ListEntries(s.db, input.FeedID, input.UnreadOnly, nil, input.Limit)
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
	if input.Limit != nil {
		filters["limit"] = *input.Limit
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

func (s *Server) handleMarkRead(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input MarkReadInput
	if err := req.BindArguments(&input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Get entry to verify it exists
	entry, err := db.GetEntryByID(s.db, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("entry not found: %s", input.EntryID)
	}

	// Mark as read
	if err := db.MarkEntryRead(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("failed to mark entry as read: %w", err)
	}

	// Reload entry to get updated read_at
	entry, err = db.GetEntryByID(s.db, input.EntryID)
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

	// Get entry to verify it exists
	entry, err := db.GetEntryByID(s.db, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("entry not found: %s", input.EntryID)
	}

	// Mark as unread
	if err := db.MarkEntryUnread(s.db, input.EntryID); err != nil {
		return nil, fmt.Errorf("failed to mark entry as unread: %w", err)
	}

	// Reload entry to get updated state
	entry, err = db.GetEntryByID(s.db, input.EntryID)
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
			return 0, false, fmt.Errorf("fetch failed and error update failed: %w (original: %v)", updateErr, err)
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
			return 0, false, fmt.Errorf("parse failed and error update failed: %w (original: %v)", updateErr, err)
		}
		return 0, false, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Update feed title if empty
	if feed.Title == nil || *feed.Title == "" {
		feed.Title = &parsed.Title
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

	return newCount, false, nil
}

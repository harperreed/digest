// ABOUTME: MCP resource providers for digest
// ABOUTME: Exposes read-only views of feeds, entries, and statistics

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harper/digest/internal/charm"
	"github.com/harper/digest/internal/timeutil"
	"github.com/mark3labs/mcp-go/mcp"
)

// ResourceData is the standard response format for all resources.
type ResourceData struct {
	Metadata ResourceMetadata  `json:"metadata"`
	Data     interface{}       `json:"data"`
	Links    map[string]string `json:"links"`
}

// ResourceMetadata contains metadata about the resource response.
type ResourceMetadata struct {
	Timestamp   time.Time      `json:"timestamp"`
	Count       int            `json:"count"`
	ResourceURI string         `json:"resource_uri"`
	Filters     map[string]any `json:"filters,omitempty"`
}

func (s *Server) registerResources() {
	// Feed resources
	s.registerFeedsResource()

	// Entry resources
	s.registerEntriesUnreadResource()
	s.registerEntriesTodayResource()

	// Statistics resource
	s.registerStatsResource()
}

func (s *Server) registerFeedsResource() {
	s.mcpServer.AddResource(
		mcp.Resource{
			URI:         "digest://feeds",
			Name:        "All Feeds",
			Description: "List all subscribed RSS/Atom feeds with metadata including title, URL, last fetch time, and error status",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			feeds, err := s.client.ListFeeds()
			if err != nil {
				return nil, fmt.Errorf("failed to list feeds: %w", err)
			}

			// Convert to output format
			feedOutputs := make([]map[string]interface{}, 0, len(feeds))
			for _, feed := range feeds {
				output := map[string]interface{}{
					"id":          feed.ID,
					"url":         feed.URL,
					"created_at":  feed.CreatedAt,
					"error_count": feed.ErrorCount,
				}
				if feed.Title != nil {
					output["title"] = *feed.Title
				}
				if feed.ETag != nil {
					output["etag"] = *feed.ETag
				}
				if feed.LastModified != nil {
					output["last_modified"] = *feed.LastModified
				}
				if feed.LastFetchedAt != nil {
					output["last_fetched_at"] = *feed.LastFetchedAt
				}
				if feed.LastError != nil {
					output["last_error"] = *feed.LastError
				}
				feedOutputs = append(feedOutputs, output)
			}

			resourceData := ResourceData{
				Metadata: ResourceMetadata{
					Timestamp:   time.Now(),
					Count:       len(feedOutputs),
					ResourceURI: "digest://feeds",
				},
				Data: feedOutputs,
				Links: map[string]string{
					"unread_entries": "digest://entries/unread",
					"today_entries":  "digest://entries/today",
					"stats":          "digest://stats",
				},
			}

			jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource data: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			}, nil
		},
	)
}

func (s *Server) registerEntriesUnreadResource() {
	s.mcpServer.AddResource(
		mcp.Resource{
			URI:         "digest://entries/unread",
			Name:        "Unread Entries",
			Description: "List all unread feed entries across all subscribed feeds, sorted by published date (newest first)",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			unreadOnly := true
			filter := &charm.EntryFilter{UnreadOnly: &unreadOnly}
			entries, err := s.client.ListEntries(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to list unread entries: %w", err)
			}

			// Convert to output format
			entryOutputs := make([]map[string]interface{}, 0, len(entries))
			for _, entry := range entries {
				output := map[string]interface{}{
					"id":         entry.ID,
					"feed_id":    entry.FeedID,
					"guid":       entry.GUID,
					"read":       entry.Read,
					"created_at": entry.CreatedAt,
				}
				if entry.Title != nil {
					output["title"] = *entry.Title
				}
				if entry.Link != nil {
					output["link"] = *entry.Link
				}
				if entry.Author != nil {
					output["author"] = *entry.Author
				}
				if entry.PublishedAt != nil {
					output["published_at"] = *entry.PublishedAt
				}
				if entry.Content != nil {
					output["content"] = *entry.Content
				}
				if entry.ReadAt != nil {
					output["read_at"] = *entry.ReadAt
				}
				entryOutputs = append(entryOutputs, output)
			}

			resourceData := ResourceData{
				Metadata: ResourceMetadata{
					Timestamp:   time.Now(),
					Count:       len(entryOutputs),
					ResourceURI: "digest://entries/unread",
					Filters: map[string]any{
						"read": false,
					},
				},
				Data: entryOutputs,
				Links: map[string]string{
					"all_feeds":     "digest://feeds",
					"today_entries": "digest://entries/today",
					"stats":         "digest://stats",
				},
			}

			jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource data: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			}, nil
		},
	)
}

func (s *Server) registerEntriesTodayResource() {
	s.mcpServer.AddResource(
		mcp.Resource{
			URI:         "digest://entries/today",
			Name:        "Today's Entries",
			Description: "List all feed entries published today (since midnight local time), regardless of read status",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Calculate start of today (midnight local time) - consistent with CLI and timeutil
			startOfDay := timeutil.StartOfToday()

			filter := &charm.EntryFilter{Since: &startOfDay}
			entries, err := s.client.ListEntries(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to list today's entries: %w", err)
			}

			// Convert to output format
			entryOutputs := make([]map[string]interface{}, 0, len(entries))
			for _, entry := range entries {
				output := map[string]interface{}{
					"id":         entry.ID,
					"feed_id":    entry.FeedID,
					"guid":       entry.GUID,
					"read":       entry.Read,
					"created_at": entry.CreatedAt,
				}
				if entry.Title != nil {
					output["title"] = *entry.Title
				}
				if entry.Link != nil {
					output["link"] = *entry.Link
				}
				if entry.Author != nil {
					output["author"] = *entry.Author
				}
				if entry.PublishedAt != nil {
					output["published_at"] = *entry.PublishedAt
				}
				if entry.Content != nil {
					output["content"] = *entry.Content
				}
				if entry.ReadAt != nil {
					output["read_at"] = *entry.ReadAt
				}
				entryOutputs = append(entryOutputs, output)
			}

			resourceData := ResourceData{
				Metadata: ResourceMetadata{
					Timestamp:   time.Now(),
					Count:       len(entryOutputs),
					ResourceURI: "digest://entries/today",
					Filters: map[string]any{
						"published_since": startOfDay,
					},
				},
				Data: entryOutputs,
				Links: map[string]string{
					"all_feeds":      "digest://feeds",
					"unread_entries": "digest://entries/unread",
					"stats":          "digest://stats",
				},
			}

			jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource data: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			}, nil
		},
	)
}

func (s *Server) registerStatsResource() {
	s.mcpServer.AddResource(
		mcp.Resource{
			URI:         "digest://stats",
			Name:        "Feed Statistics",
			Description: "Overview statistics including feed counts, entry counts (total, unread), last sync times, and per-feed breakdowns",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			stats, err := s.calculateStats()
			if err != nil {
				return nil, fmt.Errorf("failed to calculate stats: %w", err)
			}

			resourceData := ResourceData{
				Metadata: ResourceMetadata{
					Timestamp:   time.Now(),
					Count:       0, // Stats don't have a count
					ResourceURI: "digest://stats",
				},
				Data: stats,
				Links: map[string]string{
					"all_feeds":      "digest://feeds",
					"unread_entries": "digest://entries/unread",
					"today_entries":  "digest://entries/today",
				},
			}

			jsonBytes, err := json.MarshalIndent(resourceData, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource data: %w", err)
			}

			return []mcp.ResourceContents{
				&mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonBytes),
				},
			}, nil
		},
	)
}

// StatsData represents the statistics summary.
type StatsData struct {
	Summary  StatsSummary `json:"summary"`
	ByFeed   []FeedStats  `json:"by_feed"`
	LastSync *SyncInfo    `json:"last_sync,omitempty"`
}

// StatsSummary contains overall counts.
type StatsSummary struct {
	TotalFeeds   int `json:"total_feeds"`
	TotalEntries int `json:"total_entries"`
	UnreadCount  int `json:"unread_count"`
}

// FeedStats contains per-feed statistics.
type FeedStats struct {
	FeedID      string     `json:"feed_id"`
	FeedTitle   string     `json:"feed_title"`
	FeedURL     string     `json:"feed_url"`
	EntryCount  int        `json:"entry_count"`
	UnreadCount int        `json:"unread_count"`
	LastFetched *time.Time `json:"last_fetched,omitempty"`
	ErrorCount  int        `json:"error_count"`
	HasErrors   bool       `json:"has_errors"`
}

// SyncInfo represents information about the last sync.
type SyncInfo struct {
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
	FeedID        string     `json:"feed_id"`
	FeedTitle     string     `json:"feed_title"`
}

func (s *Server) calculateStats() (*StatsData, error) {
	// Get overall stats
	overallStats, err := s.client.GetOverallStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get overall stats: %w", err)
	}

	summary := StatsSummary{
		TotalFeeds:   overallStats.TotalFeeds,
		TotalEntries: overallStats.TotalEntries,
		UnreadCount:  overallStats.UnreadCount,
	}

	// Get per-feed stats
	feedStats, err := s.client.GetFeedStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get feed stats: %w", err)
	}

	// Build per-feed stats and track most recent sync
	byFeed := make([]FeedStats, 0, len(feedStats))
	var lastSync *SyncInfo

	for _, stat := range feedStats {
		feedTitle := "Untitled Feed"
		if stat.FeedTitle != nil {
			feedTitle = *stat.FeedTitle
		}

		feedStat := FeedStats{
			FeedID:      stat.FeedID,
			FeedTitle:   feedTitle,
			FeedURL:     stat.FeedURL,
			EntryCount:  stat.EntryCount,
			UnreadCount: stat.UnreadCount,
			LastFetched: stat.LastFetchedAt,
			ErrorCount:  stat.ErrorCount,
			HasErrors:   stat.LastError != nil,
		}
		byFeed = append(byFeed, feedStat)

		// Track most recent sync
		if stat.LastFetchedAt != nil {
			if lastSync == nil || stat.LastFetchedAt.After(*lastSync.LastFetchedAt) {
				lastSync = &SyncInfo{
					LastFetchedAt: stat.LastFetchedAt,
					FeedID:        stat.FeedID,
					FeedTitle:     feedTitle,
				}
			}
		}
	}

	return &StatsData{
		Summary:  summary,
		ByFeed:   byFeed,
		LastSync: lastSync,
	}, nil
}

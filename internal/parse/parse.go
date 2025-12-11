// ABOUTME: RSS/Atom feed parsing using gofeed library
// ABOUTME: Converts gofeed.Feed to simplified ParsedFeed structure with normalized fields

package parse

import (
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// ParsedFeed represents a normalized feed structure
type ParsedFeed struct {
	Title   string
	Entries []ParsedEntry
}

// ParsedEntry represents a normalized feed entry
type ParsedEntry struct {
	GUID        string
	Title       string
	Link        string
	Author      string
	PublishedAt *time.Time
	Content     string
	Categories  []string
}

// Parse parses RSS or Atom feed data and returns a normalized ParsedFeed
func Parse(data []byte) (*ParsedFeed, error) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseString(string(data))
	if err != nil {
		return nil, err
	}

	parsed := &ParsedFeed{
		Title:   feed.Title,
		Entries: make([]ParsedEntry, 0, len(feed.Items)),
	}

	for _, item := range feed.Items {
		entry := ParsedEntry{
			GUID:       item.GUID,
			Title:      item.Title,
			Link:       item.Link,
			Categories: item.Categories,
		}

		// Fallback GUID to Link if empty
		if entry.GUID == "" {
			entry.GUID = item.Link
		}

		// Extract author name
		if item.Author != nil {
			entry.Author = item.Author.Name
		}

		// Use PublishedParsed or fallback to UpdatedParsed
		if item.PublishedParsed != nil {
			entry.PublishedAt = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			entry.PublishedAt = item.UpdatedParsed
		}

		// Prefer Content over Description
		if item.Content != "" {
			entry.Content = item.Content
		} else {
			entry.Content = item.Description
		}

		// Clean up content - remove HTML tags if needed
		entry.Content = strings.TrimSpace(entry.Content)

		parsed.Entries = append(parsed.Entries, entry)
	}

	return parsed, nil
}

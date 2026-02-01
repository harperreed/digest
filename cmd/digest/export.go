// ABOUTME: Export command for exporting data in various formats
// ABOUTME: Supports OPML, YAML, and Markdown export formats

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data in various formats",
	Long: `Export feeds and entries in various formats.

Formats:
  opml     - OPML feed list (default)
  yaml     - Full data export in YAML
  markdown - Human-readable Markdown

Examples:
  digest export              # OPML to stdout
  digest export --format yaml > backup.yaml
  digest export --format markdown > reading-list.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		switch format {
		case "opml", "":
			return opmlDoc.Write(os.Stdout)
		case "yaml":
			return exportYAML()
		case "markdown", "md":
			return exportMarkdown()
		default:
			return fmt.Errorf("unknown format: %s (use opml, yaml, or markdown)", format)
		}
	},
}

// YAMLExport represents the full export structure
type YAMLExport struct {
	Version    string      `yaml:"version"`
	ExportedAt string      `yaml:"exported_at"`
	Tool       string      `yaml:"tool"`
	Feeds      []YAMLFeed  `yaml:"feeds"`
	Entries    []YAMLEntry `yaml:"entries,omitempty"`
}

// YAMLFeed represents a feed in YAML export
type YAMLFeed struct {
	ID            string `yaml:"id"`
	URL           string `yaml:"url"`
	Title         string `yaml:"title,omitempty"`
	Folder        string `yaml:"folder,omitempty"`
	LastFetchedAt string `yaml:"last_fetched_at,omitempty"`
	ErrorCount    int    `yaml:"error_count,omitempty"`
	LastError     string `yaml:"last_error,omitempty"`
}

// YAMLEntry represents an entry in YAML export
type YAMLEntry struct {
	ID          string `yaml:"id"`
	FeedID      string `yaml:"feed_id"`
	GUID        string `yaml:"guid"`
	Title       string `yaml:"title,omitempty"`
	Link        string `yaml:"link,omitempty"`
	Author      string `yaml:"author,omitempty"`
	PublishedAt string `yaml:"published_at,omitempty"`
	Read        bool   `yaml:"read"`
	ReadAt      string `yaml:"read_at,omitempty"`
	Content     string `yaml:"content,omitempty"`
}

func exportYAML() error {
	feeds, err := store.ListFeeds()
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	entries, err := store.ListEntries(nil)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	export := YAMLExport{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Tool:       "digest",
		Feeds:      make([]YAMLFeed, 0, len(feeds)),
		Entries:    make([]YAMLEntry, 0, len(entries)),
	}

	for _, feed := range feeds {
		yf := YAMLFeed{
			ID:         feed.ID,
			URL:        feed.URL,
			Folder:     feed.Folder,
			ErrorCount: feed.ErrorCount,
		}
		if feed.Title != nil {
			yf.Title = *feed.Title
		}
		if feed.LastFetchedAt != nil {
			yf.LastFetchedAt = feed.LastFetchedAt.Format(time.RFC3339)
		}
		if feed.LastError != nil {
			yf.LastError = *feed.LastError
		}
		export.Feeds = append(export.Feeds, yf)
	}

	for _, entry := range entries {
		ye := YAMLEntry{
			ID:     entry.ID,
			FeedID: entry.FeedID,
			GUID:   entry.GUID,
			Read:   entry.Read,
		}
		if entry.Title != nil {
			ye.Title = *entry.Title
		}
		if entry.Link != nil {
			ye.Link = *entry.Link
		}
		if entry.Author != nil {
			ye.Author = *entry.Author
		}
		if entry.PublishedAt != nil {
			ye.PublishedAt = entry.PublishedAt.Format(time.RFC3339)
		}
		if entry.ReadAt != nil {
			ye.ReadAt = entry.ReadAt.Format(time.RFC3339)
		}
		if entry.Content != nil {
			ye.Content = *entry.Content
		}
		export.Entries = append(export.Entries, ye)
	}

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	return encoder.Encode(export)
}

func exportMarkdown() error {
	feeds, err := store.ListFeeds()
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	entries, err := store.ListEntries(nil)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Group entries by feed
	entriesByFeed := make(map[string][]*struct {
		Title       string
		Link        string
		Author      string
		PublishedAt string
		Read        bool
	})

	for _, entry := range entries {
		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		link := ""
		if entry.Link != nil {
			link = *entry.Link
		}
		author := ""
		if entry.Author != nil {
			author = *entry.Author
		}
		publishedAt := ""
		if entry.PublishedAt != nil {
			publishedAt = entry.PublishedAt.Format("January 2, 2006")
		}

		entriesByFeed[entry.FeedID] = append(entriesByFeed[entry.FeedID], &struct {
			Title       string
			Link        string
			Author      string
			PublishedAt string
			Read        bool
		}{
			Title:       title,
			Link:        link,
			Author:      author,
			PublishedAt: publishedAt,
			Read:        entry.Read,
		})
	}

	// Write header
	fmt.Printf("# Feed Entries Export - %s\n\n", time.Now().Format("January 2, 2006"))
	fmt.Printf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	// Write by feed
	for _, feed := range feeds {
		feedTitle := feed.URL
		if feed.Title != nil {
			feedTitle = *feed.Title
		}

		feedEntries := entriesByFeed[feed.ID]
		if len(feedEntries) == 0 {
			continue
		}

		fmt.Printf("## %s\n\n", feedTitle)

		for _, entry := range feedEntries {
			readStatus := ""
			if entry.Read {
				readStatus = " [read]"
			}
			fmt.Printf("### %s%s\n\n", entry.Title, readStatus)

			if entry.Author != "" {
				fmt.Printf("- **Author:** %s\n", entry.Author)
			}
			if entry.PublishedAt != "" {
				fmt.Printf("- **Published:** %s\n", entry.PublishedAt)
			}
			if entry.Link != "" {
				fmt.Printf("- **Link:** %s\n", entry.Link)
			}
			fmt.Println()
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("format", "f", "opml", "output format: opml, yaml, or markdown")
}

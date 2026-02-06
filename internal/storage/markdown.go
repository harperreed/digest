// ABOUTME: Core MarkdownStore struct and helpers for file-based digest storage
// ABOUTME: Provides constructor, YAML feed registry types, and entry frontmatter parsing

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harper/suite/mdstore"
	"gopkg.in/yaml.v3"

	"github.com/harper/digest/internal/models"
)

// MarkdownStore provides file-based storage for digest data using markdown files and YAML.
type MarkdownStore struct {
	dataDir string
}

// Compile-time check that MarkdownStore implements Store.
var _ Store = (*MarkdownStore)(nil)

// NewMarkdownStore creates a new markdown-backed store rooted at dataDir.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	if err := mdstore.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	return &MarkdownStore{dataDir: dataDir}, nil
}

// Close releases resources. For MarkdownStore this is a no-op.
func (s *MarkdownStore) Close() error {
	return nil
}

// feedsFilePath returns the path to the _feeds.yaml file.
func (s *MarkdownStore) feedsFilePath() string {
	return filepath.Join(s.dataDir, "_feeds.yaml")
}

// feedDirPath returns the directory path for a feed.
func (s *MarkdownStore) feedDirPath(feedSlug string) string {
	return filepath.Join(s.dataDir, feedSlug)
}

// feedEntry represents a single feed in the _feeds.yaml file.
type feedEntry struct {
	ID            string  `yaml:"id"`
	URL           string  `yaml:"url"`
	Title         *string `yaml:"title,omitempty"`
	Folder        string  `yaml:"folder,omitempty"`
	ETag          *string `yaml:"etag,omitempty"`
	LastModified  *string `yaml:"last_modified,omitempty"`
	LastFetchedAt *string `yaml:"last_fetched_at,omitempty"`
	LastError     *string `yaml:"last_error,omitempty"`
	ErrorCount    int     `yaml:"error_count,omitempty"`
	CreatedAt     string  `yaml:"created_at"`
	Slug          string  `yaml:"slug"`
}

// toModel converts a feedEntry to a models.Feed.
func (e *feedEntry) toModel() (*models.Feed, error) {
	createdAt, err := mdstore.ParseTime(e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse feed created_at %q: %w", e.CreatedAt, err)
	}

	feed := &models.Feed{
		ID:           e.ID,
		URL:          e.URL,
		Title:        e.Title,
		Folder:       e.Folder,
		ETag:         e.ETag,
		LastModified: e.LastModified,
		LastError:    e.LastError,
		ErrorCount:   e.ErrorCount,
		CreatedAt:    createdAt,
	}

	if e.LastFetchedAt != nil {
		t, err := mdstore.ParseTime(*e.LastFetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse feed last_fetched_at %q: %w", *e.LastFetchedAt, err)
		}
		feed.LastFetchedAt = &t
	}

	return feed, nil
}

// fromFeedModel converts a models.Feed to a feedEntry with a computed slug.
func fromFeedModel(f *models.Feed, slug string) feedEntry {
	entry := feedEntry{
		ID:           f.ID,
		URL:          f.URL,
		Title:        f.Title,
		Folder:       f.Folder,
		ETag:         f.ETag,
		LastModified: f.LastModified,
		LastError:    f.LastError,
		ErrorCount:   f.ErrorCount,
		CreatedAt:    mdstore.FormatTime(f.CreatedAt.UTC()),
		Slug:         slug,
	}

	if f.LastFetchedAt != nil {
		s := mdstore.FormatTime(f.LastFetchedAt.UTC())
		entry.LastFetchedAt = &s
	}

	return entry
}

// readFeeds reads the _feeds.yaml file.
func (s *MarkdownStore) readFeeds() ([]feedEntry, error) {
	var entries []feedEntry
	if err := mdstore.ReadYAML(s.feedsFilePath(), &entries); err != nil {
		return nil, fmt.Errorf("read feeds file: %w", err)
	}
	return entries, nil
}

// writeFeeds writes the _feeds.yaml file atomically.
func (s *MarkdownStore) writeFeeds(entries []feedEntry) error {
	return mdstore.WriteYAML(s.feedsFilePath(), entries)
}

// feedSlugForModel generates a unique directory slug for a feed.
func (s *MarkdownStore) feedSlugForModel(f *models.Feed) string {
	var base string
	if f.Title != nil && *f.Title != "" {
		base = *f.Title
	} else {
		// Use hostname from URL as fallback
		base = f.URL
		// Strip scheme
		if idx := strings.Index(base, "://"); idx != -1 {
			base = base[idx+3:]
		}
		// Strip path
		if idx := strings.Index(base, "/"); idx != -1 {
			base = base[:idx]
		}
	}

	return mdstore.UniqueSlug(base, func(candidate string) bool {
		dir := s.feedDirPath(candidate)
		_, err := os.Stat(dir)
		return err == nil
	})
}

// feedSlugByID finds the slug for a feed by its ID from the feed registry.
func (s *MarkdownStore) feedSlugByID(feedID string) (string, error) {
	entries, err := s.readFeeds()
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.ID == feedID {
			return e.Slug, nil
		}
	}
	return "", fmt.Errorf("feed not found: %s", feedID)
}

// entryFrontmatter holds the YAML frontmatter of an entry markdown file.
type entryFrontmatter struct {
	ID          string  `yaml:"id"`
	FeedID      string  `yaml:"feed_id"`
	GUID        string  `yaml:"guid"`
	Title       *string `yaml:"title,omitempty"`
	Link        *string `yaml:"link,omitempty"`
	Author      *string `yaml:"author,omitempty"`
	PublishedAt *string `yaml:"published_at,omitempty"`
	Read        bool    `yaml:"read"`
	ReadAt      *string `yaml:"read_at,omitempty"`
	CreatedAt   string  `yaml:"created_at"`
}

// toModel converts an entryFrontmatter (plus body content) to a models.Entry.
func (fm *entryFrontmatter) toModel(content string) (*models.Entry, error) {
	createdAt, err := mdstore.ParseTime(fm.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse entry created_at %q: %w", fm.CreatedAt, err)
	}

	entry := &models.Entry{
		ID:        fm.ID,
		FeedID:    fm.FeedID,
		GUID:      fm.GUID,
		Title:     fm.Title,
		Link:      fm.Link,
		Author:    fm.Author,
		Read:      fm.Read,
		CreatedAt: createdAt,
	}

	if content != "" {
		entry.Content = &content
	}

	if fm.PublishedAt != nil {
		t, err := mdstore.ParseTime(*fm.PublishedAt)
		if err != nil {
			return nil, fmt.Errorf("parse entry published_at %q: %w", *fm.PublishedAt, err)
		}
		entry.PublishedAt = &t
	}

	if fm.ReadAt != nil {
		t, err := mdstore.ParseTime(*fm.ReadAt)
		if err != nil {
			return nil, fmt.Errorf("parse entry read_at %q: %w", *fm.ReadAt, err)
		}
		entry.ReadAt = &t
	}

	return entry, nil
}

// fromEntryModel converts a models.Entry to an entryFrontmatter.
func fromEntryModel(e *models.Entry) entryFrontmatter {
	fm := entryFrontmatter{
		ID:        e.ID,
		FeedID:    e.FeedID,
		GUID:      e.GUID,
		Title:     e.Title,
		Link:      e.Link,
		Author:    e.Author,
		Read:      e.Read,
		CreatedAt: mdstore.FormatTime(e.CreatedAt.UTC()),
	}

	if e.PublishedAt != nil {
		s := mdstore.FormatTime(e.PublishedAt.UTC())
		fm.PublishedAt = &s
	}

	if e.ReadAt != nil {
		s := mdstore.FormatTime(e.ReadAt.UTC())
		fm.ReadAt = &s
	}

	return fm
}

// entryFileName generates a filename for an entry markdown file.
// Uses slugified title + first 8 chars of entry ID to guarantee uniqueness.
func entryFileName(e *models.Entry) string {
	titleStr := "untitled"
	if e.Title != nil && *e.Title != "" {
		titleStr = *e.Title
	}
	slug := mdstore.Slugify(titleStr)
	// Truncate slug to avoid overly long filenames
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return slug + "-" + e.ID[:8] + ".md"
}

// readEntryFile reads a single entry markdown file and returns the model.
func readEntryFile(path string) (*models.Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	yamlStr, body := mdstore.ParseFrontmatter(string(data))
	if yamlStr == "" {
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	var fm entryFrontmatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, fmt.Errorf("parse entry frontmatter in %s: %w", path, err)
	}

	content := strings.TrimSpace(body)
	return fm.toModel(content)
}

// writeEntryFile writes an entry to a markdown file with frontmatter.
func writeEntryFile(path string, e *models.Entry) error {
	fm := fromEntryModel(e)

	body := ""
	if e.Content != nil {
		body = "\n" + *e.Content + "\n"
	}

	content, err := mdstore.RenderFrontmatter(&fm, body)
	if err != nil {
		return fmt.Errorf("render entry frontmatter: %w", err)
	}

	return mdstore.AtomicWrite(path, []byte(content))
}

// findEntryFile locates the file for a specific entry ID within a feed directory.
func findEntryFile(feedDir string, entryID string) (string, error) {
	dirEntries, err := os.ReadDir(feedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("entry not found: %s", entryID)
		}
		return "", fmt.Errorf("read feed directory: %w", err)
	}

	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		fp := filepath.Join(feedDir, de.Name())
		entry, err := readEntryFile(fp)
		if err != nil {
			continue
		}
		if entry.ID == entryID {
			return fp, nil
		}
	}
	return "", fmt.Errorf("entry not found: %s", entryID)
}

// readAllEntries reads all entries from a feed directory.
func readAllEntries(feedDir string) ([]*models.Entry, error) {
	dirEntries, err := os.ReadDir(feedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read feed directory: %w", err)
	}

	var entries []*models.Entry
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		fp := filepath.Join(feedDir, de.Name())
		entry, err := readEntryFile(fp)
		if err != nil {
			// Skip malformed files
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// timePtr is a helper to convert *time.Time to a comparable value for filtering.
func timeAfterOrEqual(t time.Time, ref time.Time) bool {
	return !t.Before(ref)
}

func timeBefore(t time.Time, ref time.Time) bool {
	return t.Before(ref)
}

// ABOUTME: Tests for storage migration between backends
// ABOUTME: Covers sqlite-to-markdown, markdown-to-sqlite, data integrity, and round-trips

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
)

// seedDigestTestData populates a storage backend with a representative data set
// and returns the feeds and entries for verification.
func seedDigestTestData(t *testing.T, src Store) (feeds []*models.Feed, entries []*models.Entry) {
	t.Helper()

	// Create feeds
	feed1 := models.NewFeed("https://example1.com/feed.xml")
	title1 := "Tech Blog"
	feed1.Title = &title1
	feed1.Folder = "Tech"

	feed2 := models.NewFeed("https://example2.com/feed.xml")
	title2 := "Science Daily"
	feed2.Title = &title2
	feed2.Folder = "Science"

	mustNoErr(t, src.CreateFeed(feed1))
	mustNoErr(t, src.CreateFeed(feed2))
	feeds = append(feeds, feed1, feed2)

	// Create entries in feed1
	now := time.Now()
	entry1 := models.NewEntry(feed1.ID, "guid-1-1", "Go 1.22 Released")
	link1 := "https://example1.com/go-122"
	author1 := "Jane Doe"
	content1 := "Go 1.22 introduces range over integers and more."
	pub1 := now.Add(-1 * time.Hour)
	entry1.Link = &link1
	entry1.Author = &author1
	entry1.Content = &content1
	entry1.PublishedAt = &pub1
	mustNoErr(t, src.CreateEntry(entry1))

	entry2 := models.NewEntry(feed1.ID, "guid-1-2", "Understanding Channels")
	content2 := "Deep dive into Go channel patterns."
	pub2 := now.Add(-2 * time.Hour)
	entry2.Content = &content2
	entry2.PublishedAt = &pub2
	mustNoErr(t, src.CreateEntry(entry2))
	mustNoErr(t, src.MarkEntryRead(entry2.ID))
	// Re-read to get the read state
	entry2, _ = src.GetEntry(entry2.ID)

	// Create entries in feed2
	entry3 := models.NewEntry(feed2.ID, "guid-2-1", "Dark Matter Discovery")
	content3 := "Scientists announce breakthrough in dark matter research."
	pub3 := now.Add(-3 * time.Hour)
	entry3.Content = &content3
	entry3.PublishedAt = &pub3
	mustNoErr(t, src.CreateEntry(entry3))

	entries = append(entries, entry1, entry2, entry3)

	return
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// verifyMigratedDigestData checks that the destination storage contains all expected data.
func verifyMigratedDigestData(t *testing.T, dst Store, feeds []*models.Feed, entries []*models.Entry) {
	t.Helper()

	// Verify feeds
	for _, orig := range feeds {
		got, err := dst.GetFeed(orig.ID)
		if err != nil {
			t.Errorf("feed %s (%s) not found in destination: %v", orig.URL, orig.ID, err)
			continue
		}
		if got.URL != orig.URL {
			t.Errorf("feed URL mismatch: want %q, got %q", orig.URL, got.URL)
		}
		if (orig.Title == nil) != (got.Title == nil) {
			t.Errorf("feed title nil mismatch: orig=%v, got=%v", orig.Title, got.Title)
		}
		if orig.Title != nil && got.Title != nil && *got.Title != *orig.Title {
			t.Errorf("feed title mismatch: want %q, got %q", *orig.Title, *got.Title)
		}
		if got.Folder != orig.Folder {
			t.Errorf("feed folder mismatch: want %q, got %q", orig.Folder, got.Folder)
		}
	}

	// Verify entries
	for _, orig := range entries {
		got, err := dst.GetEntry(orig.ID)
		if err != nil {
			t.Errorf("entry %s not found in destination: %v", orig.ID, err)
			continue
		}
		if got.FeedID != orig.FeedID {
			t.Errorf("entry feedID mismatch: want %q, got %q", orig.FeedID, got.FeedID)
		}
		if got.GUID != orig.GUID {
			t.Errorf("entry GUID mismatch: want %q, got %q", orig.GUID, got.GUID)
		}
		if (orig.Title == nil) != (got.Title == nil) {
			t.Errorf("entry title nil mismatch: orig=%v, got=%v", orig.Title, got.Title)
		}
		if orig.Title != nil && got.Title != nil && *got.Title != *orig.Title {
			t.Errorf("entry title mismatch: want %q, got %q", *orig.Title, *got.Title)
		}
		if (orig.Content == nil) != (got.Content == nil) {
			t.Errorf("entry content nil mismatch: orig=%v, got=%v", orig.Content, got.Content)
		}
		if orig.Content != nil && got.Content != nil && *got.Content != *orig.Content {
			t.Errorf("entry content mismatch: want %q, got %q", *orig.Content, *got.Content)
		}
		if got.Read != orig.Read {
			t.Errorf("entry read mismatch: want %v, got %v", orig.Read, got.Read)
		}
	}
}

func TestMigrateData_SqliteToMarkdown(t *testing.T) {
	// Set up source (sqlite)
	srcDir := t.TempDir()
	src, err := NewSQLiteStore(filepath.Join(srcDir, "digest.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	feeds, entries := seedDigestTestData(t, src)

	// Set up destination (markdown)
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	// Run migration
	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify summary counts
	if summary.Feeds != len(feeds) {
		t.Errorf("summary feeds: want %d, got %d", len(feeds), summary.Feeds)
	}
	if summary.Entries != len(entries) {
		t.Errorf("summary entries: want %d, got %d", len(entries), summary.Entries)
	}

	// Verify all data was migrated correctly
	verifyMigratedDigestData(t, dst, feeds, entries)
}

func TestMigrateData_MarkdownToSqlite(t *testing.T) {
	// Set up source (markdown)
	srcDir := t.TempDir()
	src, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	feeds, entries := seedDigestTestData(t, src)

	// Set up destination (sqlite)
	dstDir := t.TempDir()
	dst, err := NewSQLiteStore(filepath.Join(dstDir, "digest.db"))
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	// Run migration
	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify summary counts
	if summary.Feeds != len(feeds) {
		t.Errorf("summary feeds: want %d, got %d", len(feeds), summary.Feeds)
	}
	if summary.Entries != len(entries) {
		t.Errorf("summary entries: want %d, got %d", len(entries), summary.Entries)
	}

	// Verify all data was migrated correctly
	verifyMigratedDigestData(t, dst, feeds, entries)
}

func TestMigrateData_EmptySource(t *testing.T) {
	// Set up empty source (sqlite)
	srcDir := t.TempDir()
	src, err := NewSQLiteStore(filepath.Join(srcDir, "digest.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	// Set up destination (markdown)
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Feeds != 0 || summary.Entries != 0 {
		t.Errorf("expected all zero counts for empty source, got feeds=%d entries=%d",
			summary.Feeds, summary.Entries)
	}
}

func TestMigrateData_SqliteToSqlite(t *testing.T) {
	// Test migrating between two sqlite instances (roundtrip sanity check)
	srcDir := t.TempDir()
	src, err := NewSQLiteStore(filepath.Join(srcDir, "digest.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	feeds, entries := seedDigestTestData(t, src)

	dstDir := t.TempDir()
	dst, err := NewSQLiteStore(filepath.Join(dstDir, "digest.db"))
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Feeds != len(feeds) {
		t.Errorf("summary feeds: want %d, got %d", len(feeds), summary.Feeds)
	}

	verifyMigratedDigestData(t, dst, feeds, entries)
}

func TestMigrateRoundTrip_SqliteToMarkdownToSqlite(t *testing.T) {
	// Phase 1: Create data in SQLite
	srcDir := t.TempDir()
	original, err := NewSQLiteStore(filepath.Join(srcDir, "original.db"))
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer original.Close()

	feeds, entries := seedDigestTestData(t, original)

	// Phase 2: Migrate SQLite -> Markdown
	mdDir := t.TempDir()
	mdStore, err := NewMarkdownStore(mdDir)
	if err != nil {
		t.Fatalf("create markdown store: %v", err)
	}
	defer mdStore.Close()

	summary1, err := MigrateData(original, mdStore)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}
	if summary1.Feeds != len(feeds) || summary1.Entries != len(entries) {
		t.Errorf("phase 1 summary mismatch: feeds=%d/%d entries=%d/%d",
			summary1.Feeds, len(feeds), summary1.Entries, len(entries))
	}

	// Phase 3: Migrate Markdown -> new SQLite
	dstDir := t.TempDir()
	final, err := NewSQLiteStore(filepath.Join(dstDir, "final.db"))
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer final.Close()

	summary2, err := MigrateData(mdStore, final)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}
	if summary2.Feeds != len(feeds) || summary2.Entries != len(entries) {
		t.Errorf("phase 2 summary mismatch: feeds=%d/%d entries=%d/%d",
			summary2.Feeds, len(feeds), summary2.Entries, len(entries))
	}

	// Phase 4: Field-by-field verification against original data
	verifyMigratedDigestData(t, final, feeds, entries)
}

func TestMigrateRoundTrip_MarkdownToSqliteToMarkdown(t *testing.T) {
	// Phase 1: Create data in Markdown
	srcDir := t.TempDir()
	original, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer original.Close()

	feeds, entries := seedDigestTestData(t, original)

	// Phase 2: Migrate Markdown -> SQLite
	sqlDir := t.TempDir()
	sqlStore, err := NewSQLiteStore(filepath.Join(sqlDir, "mid.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqlStore.Close()

	_, err = MigrateData(original, sqlStore)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}

	// Phase 3: Migrate SQLite -> new Markdown
	dstDir := t.TempDir()
	final, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer final.Close()

	_, err = MigrateData(sqlStore, final)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}

	// Phase 4: Verify all data
	verifyMigratedDigestData(t, final, feeds, entries)
}

func TestIsDirNonEmpty(t *testing.T) {
	// Empty directory
	emptyDir := t.TempDir()
	nonEmpty, err := IsDirNonEmpty(emptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on empty dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected empty dir to be reported as empty")
	}

	// Non-empty directory
	nonEmptyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	nonEmpty, err = IsDirNonEmpty(nonEmptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-empty dir: %v", err)
	}
	if !nonEmpty {
		t.Error("expected non-empty dir to be reported as non-empty")
	}

	// Non-existent directory
	nonEmpty, err = IsDirNonEmpty(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-existent dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected non-existent dir to be reported as empty")
	}
}

// ABOUTME: Test suite for OPML parsing, writing, and manipulation
// ABOUTME: Covers parsing XML, adding feeds, folder management, and round-trip integrity

package opml

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestParseOPML(t *testing.T) {
	opmlData := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>My Feeds</title>
  </head>
  <body>
    <outline text="Tech News">
      <outline type="rss" text="Hacker News" xmlUrl="https://hnrss.org/frontpage" />
      <outline type="rss" text="TechCrunch" xmlUrl="https://techcrunch.com/feed/" />
    </outline>
    <outline text="Blogs">
      <outline type="rss" text="Joel on Software" xmlUrl="https://www.joelonsoftware.com/feed/" />
    </outline>
    <outline type="rss" text="No Folder Feed" xmlUrl="https://example.com/feed" />
  </body>
</opml>`

	doc, err := Parse(bytes.NewBufferString(opmlData))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if doc.Title != "My Feeds" {
		t.Errorf("Title = %q, want %q", doc.Title, "My Feeds")
	}

	feeds := doc.AllFeeds()
	if len(feeds) != 4 {
		t.Fatalf("AllFeeds() returned %d feeds, want 4", len(feeds))
	}

	// Check Tech News folder
	techFeeds := doc.FeedsInFolder("Tech News")
	if len(techFeeds) != 2 {
		t.Errorf("FeedsInFolder('Tech News') = %d feeds, want 2", len(techFeeds))
	}

	// Check Blogs folder
	blogFeeds := doc.FeedsInFolder("Blogs")
	if len(blogFeeds) != 1 {
		t.Errorf("FeedsInFolder('Blogs') = %d feeds, want 1", len(blogFeeds))
	}

	// Check root level feed
	rootFeeds := doc.FeedsInFolder("")
	if len(rootFeeds) != 1 {
		t.Errorf("FeedsInFolder('') = %d feeds, want 1", len(rootFeeds))
	}

	// Check folders list
	folders := doc.Folders()
	expectedFolders := map[string]bool{"Tech News": true, "Blogs": true}
	if len(folders) != len(expectedFolders) {
		t.Errorf("Folders() returned %d folders, want %d", len(folders), len(expectedFolders))
	}
	for _, folder := range folders {
		if !expectedFolders[folder] {
			t.Errorf("Unexpected folder: %q", folder)
		}
	}
}

func TestOPML_AddFeed(t *testing.T) {
	doc := NewDocument("Test Document")

	err := doc.AddFeed("https://example.com/feed", "Example Feed", "")
	if err != nil {
		t.Fatalf("AddFeed() error = %v", err)
	}

	feeds := doc.AllFeeds()
	if len(feeds) != 1 {
		t.Fatalf("AllFeeds() = %d feeds, want 1", len(feeds))
	}

	feed := feeds[0]
	if feed.URL != "https://example.com/feed" {
		t.Errorf("feed.URL = %q, want %q", feed.URL, "https://example.com/feed")
	}
	if feed.Title != "Example Feed" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Example Feed")
	}
	if feed.Folder != "" {
		t.Errorf("feed.Folder = %q, want empty string", feed.Folder)
	}
}

func TestOPML_AddFeedToFolder(t *testing.T) {
	doc := NewDocument("Test Document")

	err := doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
	if err != nil {
		t.Fatalf("AddFeed() error = %v", err)
	}

	err = doc.AddFeed("https://example.com/feed2", "Feed 2", "Tech")
	if err != nil {
		t.Fatalf("AddFeed() error = %v", err)
	}

	err = doc.AddFeed("https://example.com/feed3", "Feed 3", "News")
	if err != nil {
		t.Fatalf("AddFeed() error = %v", err)
	}

	// Verify feeds in Tech folder
	techFeeds := doc.FeedsInFolder("Tech")
	if len(techFeeds) != 2 {
		t.Fatalf("FeedsInFolder('Tech') = %d feeds, want 2", len(techFeeds))
	}

	// Verify feeds in News folder
	newsFeeds := doc.FeedsInFolder("News")
	if len(newsFeeds) != 1 {
		t.Fatalf("FeedsInFolder('News') = %d feeds, want 1", len(newsFeeds))
	}

	// Verify total feeds
	allFeeds := doc.AllFeeds()
	if len(allFeeds) != 3 {
		t.Errorf("AllFeeds() = %d feeds, want 3", len(allFeeds))
	}

	// Verify folders
	folders := doc.Folders()
	if len(folders) != 2 {
		t.Fatalf("Folders() = %d folders, want 2", len(folders))
	}
}

func TestOPML_RoundTrip(t *testing.T) {
	// Create a document
	doc := NewDocument("Round Trip Test")
	doc.AddFeed("https://example.com/feed1", "Feed 1", "Folder A")
	doc.AddFeed("https://example.com/feed2", "Feed 2", "Folder A")
	doc.AddFeed("https://example.com/feed3", "Feed 3", "Folder B")
	doc.AddFeed("https://example.com/feed4", "Feed 4", "")

	// Write to file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.opml")

	err := doc.WriteFile(tmpFile)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Parse it back
	doc2, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	// Verify title
	if doc2.Title != doc.Title {
		t.Errorf("Title = %q, want %q", doc2.Title, doc.Title)
	}

	// Verify feeds
	feeds1 := doc.AllFeeds()
	feeds2 := doc2.AllFeeds()

	if len(feeds2) != len(feeds1) {
		t.Fatalf("AllFeeds() = %d feeds, want %d", len(feeds2), len(feeds1))
	}

	// Create maps for comparison
	feedMap1 := make(map[string]Feed)
	for _, f := range feeds1 {
		feedMap1[f.URL] = f
	}

	feedMap2 := make(map[string]Feed)
	for _, f := range feeds2 {
		feedMap2[f.URL] = f
	}

	// Compare feeds
	for url, f1 := range feedMap1 {
		f2, ok := feedMap2[url]
		if !ok {
			t.Errorf("Feed %q not found after round trip", url)
			continue
		}
		if f1.Title != f2.Title {
			t.Errorf("Feed %q: Title = %q, want %q", url, f2.Title, f1.Title)
		}
		if f1.Folder != f2.Folder {
			t.Errorf("Feed %q: Folder = %q, want %q", url, f2.Folder, f1.Folder)
		}
	}

	// Verify folders
	folders1 := doc.Folders()
	folders2 := doc2.Folders()
	if len(folders2) != len(folders1) {
		t.Errorf("Folders() = %d, want %d", len(folders2), len(folders1))
	}
}

func TestOPML_RemoveFeed(t *testing.T) {
	doc := NewDocument("Test Document")
	doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
	doc.AddFeed("https://example.com/feed2", "Feed 2", "Tech")
	doc.AddFeed("https://example.com/feed3", "Feed 3", "News")

	// Remove a feed
	err := doc.RemoveFeed("https://example.com/feed2")
	if err != nil {
		t.Fatalf("RemoveFeed() error = %v", err)
	}

	// Verify it's removed
	feeds := doc.AllFeeds()
	if len(feeds) != 2 {
		t.Fatalf("AllFeeds() = %d feeds, want 2", len(feeds))
	}

	// Verify the right feed was removed
	for _, feed := range feeds {
		if feed.URL == "https://example.com/feed2" {
			t.Error("Feed 2 should have been removed")
		}
	}

	// Verify Tech folder still has one feed
	techFeeds := doc.FeedsInFolder("Tech")
	if len(techFeeds) != 1 {
		t.Errorf("FeedsInFolder('Tech') = %d feeds, want 1", len(techFeeds))
	}
}

func TestOPML_AddFolder(t *testing.T) {
	doc := NewDocument("Test Document")

	// Add a folder
	err := doc.AddFolder("New Folder")
	if err != nil {
		t.Fatalf("AddFolder() error = %v", err)
	}

	// Verify folder exists
	folders := doc.Folders()
	found := false
	for _, f := range folders {
		if f == "New Folder" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Folder 'New Folder' not found")
	}

	// Add the same folder again (should be idempotent)
	err = doc.AddFolder("New Folder")
	if err != nil {
		t.Fatalf("AddFolder() second call error = %v", err)
	}

	// Verify still only one instance
	folders = doc.Folders()
	count := 0
	for _, f := range folders {
		if f == "New Folder" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Found %d instances of 'New Folder', want 1", count)
	}
}

func TestOPML_Write(t *testing.T) {
	doc := NewDocument("Write Test")
	doc.AddFeed("https://example.com/feed", "Example Feed", "Tech")

	var buf bytes.Buffer
	err := doc.Write(&buf)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Write() produced empty output")
	}

	// Basic validation that it looks like OPML
	if !bytes.Contains(buf.Bytes(), []byte("<?xml")) {
		t.Error("Output missing XML declaration")
	}
	if !bytes.Contains(buf.Bytes(), []byte("<opml")) {
		t.Error("Output missing <opml> tag")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Write Test")) {
		t.Error("Output missing title")
	}
	if !bytes.Contains(buf.Bytes(), []byte("https://example.com/feed")) {
		t.Error("Output missing feed URL")
	}
}

func TestOPML_MoveFeed(t *testing.T) {
	doc := NewDocument("Move Test")

	// Add feeds to different folders
	doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
	doc.AddFeed("https://example.com/feed2", "Feed 2", "News")
	doc.AddFeed("https://example.com/feed3", "Feed 3", "")

	// Verify initial state
	techFeeds := doc.FeedsInFolder("Tech")
	if len(techFeeds) != 1 {
		t.Fatalf("expected 1 feed in Tech, got %d", len(techFeeds))
	}

	newsFeeds := doc.FeedsInFolder("News")
	if len(newsFeeds) != 1 {
		t.Fatalf("expected 1 feed in News, got %d", len(newsFeeds))
	}

	// Move feed from Tech to News
	err := doc.MoveFeed("https://example.com/feed1", "News")
	if err != nil {
		t.Fatalf("MoveFeed() error = %v", err)
	}

	// Verify feed moved
	techFeeds = doc.FeedsInFolder("Tech")
	if len(techFeeds) != 0 {
		t.Errorf("expected 0 feeds in Tech after move, got %d", len(techFeeds))
	}

	newsFeeds = doc.FeedsInFolder("News")
	if len(newsFeeds) != 2 {
		t.Errorf("expected 2 feeds in News after move, got %d", len(newsFeeds))
	}

	// Move feed to root level
	err = doc.MoveFeed("https://example.com/feed2", "")
	if err != nil {
		t.Fatalf("MoveFeed() to root error = %v", err)
	}

	rootFeeds := doc.FeedsInFolder("")
	if len(rootFeeds) != 2 {
		t.Errorf("expected 2 feeds at root level, got %d", len(rootFeeds))
	}

	// Move feed to new folder (should create it)
	err = doc.MoveFeed("https://example.com/feed3", "Sports")
	if err != nil {
		t.Fatalf("MoveFeed() to new folder error = %v", err)
	}

	sportsFeeds := doc.FeedsInFolder("Sports")
	if len(sportsFeeds) != 1 {
		t.Errorf("expected 1 feed in Sports, got %d", len(sportsFeeds))
	}

	// Try to move non-existent feed
	err = doc.MoveFeed("https://example.com/nonexistent", "Tech")
	if err == nil {
		t.Error("expected error when moving non-existent feed")
	}
}

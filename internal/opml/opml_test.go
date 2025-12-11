// ABOUTME: Test suite for OPML parsing, writing, and manipulation
// ABOUTME: Covers parsing XML, adding feeds, folder management, and round-trip integrity

package opml

import (
	"bytes"
	"fmt"
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

func TestOPML_MalformedXML(t *testing.T) {
	tests := []struct {
		name     string
		opmlData string
		wantErr  bool
	}{
		{
			name:     "invalid XML syntax",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head><title>Test</title></head><body><outline text="Unclosed"`,
			wantErr:  true,
		},
		{
			name:     "missing closing tag",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head><title>Test</title></head><body><outline text="Test"></body></opml>`,
			wantErr:  true,
		},
		{
			name:     "completely invalid XML",
			opmlData: `this is not XML at all`,
			wantErr:  true,
		},
		{
			name:     "empty content",
			opmlData: ``,
			wantErr:  true,
		},
		{
			name:     "malformed attributes",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head><title>Test</title></head><body><outline text="Test" xmlUrl=invalid></outline></body></opml>`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(bytes.NewBufferString(tt.opmlData))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOPML_MissingAttributes(t *testing.T) {
	tests := []struct {
		name     string
		opmlData string
		wantErr  bool
		checkFn  func(*testing.T, *Document)
	}{
		{
			name: "feed missing xmlUrl",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" text="Feed Without URL" />
  </body>
</opml>`,
			wantErr: false,
			checkFn: func(t *testing.T, doc *Document) {
				feeds := doc.AllFeeds()
				// Should not include outlines without xmlUrl
				if len(feeds) != 0 {
					t.Errorf("expected 0 feeds (missing xmlUrl), got %d", len(feeds))
				}
			},
		},
		{
			name: "feed missing text attribute",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" xmlUrl="https://example.com/feed" />
  </body>
</opml>`,
			wantErr: false,
			checkFn: func(t *testing.T, doc *Document) {
				feeds := doc.AllFeeds()
				if len(feeds) != 1 {
					t.Fatalf("expected 1 feed, got %d", len(feeds))
				}
				// Text should be empty string
				if feeds[0].Title != "" {
					t.Errorf("expected empty title, got %q", feeds[0].Title)
				}
			},
		},
		{
			name: "missing head section",
			opmlData: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline type="rss" text="Test" xmlUrl="https://example.com/feed" />
  </body>
</opml>`,
			wantErr: false,
			checkFn: func(t *testing.T, doc *Document) {
				if doc.Title != "" {
					t.Errorf("expected empty title, got %q", doc.Title)
				}
				feeds := doc.AllFeeds()
				if len(feeds) != 1 {
					t.Errorf("expected 1 feed, got %d", len(feeds))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(bytes.NewBufferString(tt.opmlData))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, doc)
			}
		})
	}
}

func TestOPML_FolderEdgeCases(t *testing.T) {
	t.Run("special characters in folder names", func(t *testing.T) {
		doc := NewDocument("Test")
		specialNames := []string{
			"Tech & Science",
			"News/Politics",
			"Blogs (Personal)",
			"RSS<>Feeds",
			"Quotes\"Inside\"",
			"Apostrophe's Test",
			"Unicode: 日本語",
			"Emoji: 🚀📰",
		}

		for _, name := range specialNames {
			err := doc.AddFeed("https://example.com/"+name, "Feed", name)
			if err != nil {
				t.Errorf("AddFeed() with folder %q failed: %v", name, err)
			}
		}

		// Verify all folders were created
		folders := doc.Folders()
		if len(folders) != len(specialNames) {
			t.Errorf("expected %d folders, got %d", len(specialNames), len(folders))
		}

		// Verify feeds are in correct folders
		for _, name := range specialNames {
			feeds := doc.FeedsInFolder(name)
			if len(feeds) != 1 {
				t.Errorf("expected 1 feed in folder %q, got %d", name, len(feeds))
			}
		}
	})

	t.Run("very long folder name", func(t *testing.T) {
		doc := NewDocument("Test")
		longName := string(make([]byte, 10000))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		err := doc.AddFeed("https://example.com/feed", "Feed", longName)
		if err != nil {
			t.Errorf("AddFeed() with long folder name failed: %v", err)
		}

		feeds := doc.FeedsInFolder(longName)
		if len(feeds) != 1 {
			t.Errorf("expected 1 feed in long folder, got %d", len(feeds))
		}
	})

	t.Run("add feed to non-existent folder creates it", func(t *testing.T) {
		doc := NewDocument("Test")
		err := doc.AddFeed("https://example.com/feed", "Feed", "NewFolder")
		if err != nil {
			t.Fatalf("AddFeed() to non-existent folder failed: %v", err)
		}

		folders := doc.Folders()
		found := false
		for _, f := range folders {
			if f == "NewFolder" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Folder 'NewFolder' was not created")
		}
	})

	t.Run("duplicate folder names", func(t *testing.T) {
		doc := NewDocument("Test")

		// Add folder
		err := doc.AddFolder("Tech")
		if err != nil {
			t.Fatalf("AddFolder() failed: %v", err)
		}

		// Add same folder again
		err = doc.AddFolder("Tech")
		if err != nil {
			t.Fatalf("AddFolder() second call failed: %v", err)
		}

		// Add feed to folder
		err = doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
		if err != nil {
			t.Fatalf("AddFeed() failed: %v", err)
		}

		// Add another feed to same folder
		err = doc.AddFeed("https://example.com/feed2", "Feed 2", "Tech")
		if err != nil {
			t.Fatalf("AddFeed() second call failed: %v", err)
		}

		// Should only have one folder
		folders := doc.Folders()
		techCount := 0
		for _, f := range folders {
			if f == "Tech" {
				techCount++
			}
		}
		if techCount != 1 {
			t.Errorf("expected 1 'Tech' folder, got %d", techCount)
		}

		// Should have both feeds
		feeds := doc.FeedsInFolder("Tech")
		if len(feeds) != 2 {
			t.Errorf("expected 2 feeds in Tech, got %d", len(feeds))
		}
	})
}

func TestOPML_URLIndexOptimization(t *testing.T) {
	t.Run("URL index provides O(1) lookup", func(t *testing.T) {
		doc := NewDocument("Test")

		// Add multiple feeds
		for i := 0; i < 100; i++ {
			url := fmt.Sprintf("https://example.com/feed%d", i)
			doc.AddFeed(url, fmt.Sprintf("Feed %d", i), "")
		}

		// Try to add duplicate - should fail quickly via O(1) lookup
		err := doc.AddFeed("https://example.com/feed50", "Duplicate", "")
		if err == nil {
			t.Error("expected error when adding duplicate feed")
		}
	})

	t.Run("URL index stays in sync after AddFeed", func(t *testing.T) {
		doc := NewDocument("Test")

		url := "https://example.com/feed"
		err := doc.AddFeed(url, "Test Feed", "")
		if err != nil {
			t.Fatalf("AddFeed() failed: %v", err)
		}

		// Check that URL is in index
		if !doc.feedURLs[url] {
			t.Error("URL not in feedURLs map after AddFeed")
		}

		// Try to add duplicate
		err = doc.AddFeed(url, "Duplicate", "")
		if err == nil {
			t.Error("expected error when adding duplicate")
		}
	})

	t.Run("URL index stays in sync after RemoveFeed", func(t *testing.T) {
		doc := NewDocument("Test")

		url := "https://example.com/feed"
		doc.AddFeed(url, "Test Feed", "")

		err := doc.RemoveFeed(url)
		if err != nil {
			t.Fatalf("RemoveFeed() failed: %v", err)
		}

		// Check that URL is removed from index
		if doc.feedURLs[url] {
			t.Error("URL still in feedURLs map after RemoveFeed")
		}

		// Should be able to add it again
		err = doc.AddFeed(url, "Test Feed", "")
		if err != nil {
			t.Errorf("AddFeed() after remove failed: %v", err)
		}
	})

	t.Run("URL index stays in sync after MoveFeed", func(t *testing.T) {
		doc := NewDocument("Test")

		url := "https://example.com/feed"
		doc.AddFeed(url, "Test Feed", "Folder1")

		// URL should be in index
		if !doc.feedURLs[url] {
			t.Error("URL not in feedURLs map after AddFeed")
		}

		err := doc.MoveFeed(url, "Folder2")
		if err != nil {
			t.Fatalf("MoveFeed() failed: %v", err)
		}

		// URL should still be in index
		if !doc.feedURLs[url] {
			t.Error("URL not in feedURLs map after MoveFeed")
		}

		// Should not be able to add duplicate
		err = doc.AddFeed(url, "Duplicate", "")
		if err == nil {
			t.Error("expected error when adding duplicate after move")
		}
	})

	t.Run("ensureURLIndex initializes map when nil", func(t *testing.T) {
		doc := &Document{
			Title:    "Test",
			Outlines: []Outline{},
			feedURLs: nil, // Explicitly nil
		}

		// This should not panic
		doc.ensureURLIndex()

		if doc.feedURLs == nil {
			t.Error("ensureURLIndex() did not initialize feedURLs map")
		}

		// Should be able to add feeds now
		err := doc.AddFeed("https://example.com/feed", "Test", "")
		if err != nil {
			t.Errorf("AddFeed() after ensureURLIndex failed: %v", err)
		}
	})

	t.Run("Parse rebuilds URL index", func(t *testing.T) {
		opmlData := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" text="Feed 1" xmlUrl="https://example.com/feed1" />
    <outline type="rss" text="Feed 2" xmlUrl="https://example.com/feed2" />
    <outline type="rss" text="Feed 3" xmlUrl="https://example.com/feed3" />
  </body>
</opml>`

		doc, err := Parse(bytes.NewBufferString(opmlData))
		if err != nil {
			t.Fatalf("Parse() failed: %v", err)
		}

		// Check that all URLs are in index
		expectedURLs := []string{
			"https://example.com/feed1",
			"https://example.com/feed2",
			"https://example.com/feed3",
		}

		for _, url := range expectedURLs {
			if !doc.feedURLs[url] {
				t.Errorf("URL %q not in feedURLs map after Parse", url)
			}
		}

		// Try to add duplicate
		err = doc.AddFeed("https://example.com/feed2", "Duplicate", "")
		if err == nil {
			t.Error("expected error when adding duplicate after Parse")
		}
	})
}

func TestOPML_RoundTripIntegrity(t *testing.T) {
	t.Run("complex document preserves all data", func(t *testing.T) {
		doc := NewDocument("Complex Test")

		// Add diverse content
		doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
		doc.AddFeed("https://example.com/feed2", "Feed 2", "Tech")
		doc.AddFeed("https://example.com/feed3", "Feed & Special <Chars>", "News")
		doc.AddFeed("https://example.com/feed4", "Root Feed", "")
		doc.AddFolder("Empty Folder")

		// Write to bytes
		var buf bytes.Buffer
		err := doc.Write(&buf)
		if err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		// Parse back
		doc2, err := Parse(&buf)
		if err != nil {
			t.Fatalf("Parse() failed: %v", err)
		}

		// Compare titles
		if doc2.Title != doc.Title {
			t.Errorf("Title mismatch: got %q, want %q", doc2.Title, doc.Title)
		}

		// Compare feeds
		feeds1 := doc.AllFeeds()
		feeds2 := doc2.AllFeeds()

		if len(feeds2) != len(feeds1) {
			t.Fatalf("Feed count mismatch: got %d, want %d", len(feeds2), len(feeds1))
		}

		// Build maps for comparison
		feedMap1 := make(map[string]Feed)
		for _, f := range feeds1 {
			feedMap1[f.URL] = f
		}

		feedMap2 := make(map[string]Feed)
		for _, f := range feeds2 {
			feedMap2[f.URL] = f
		}

		// Compare each feed
		for url, f1 := range feedMap1 {
			f2, ok := feedMap2[url]
			if !ok {
				t.Errorf("Feed %q missing after round trip", url)
				continue
			}
			if f1.Title != f2.Title {
				t.Errorf("Feed %q title: got %q, want %q", url, f2.Title, f1.Title)
			}
			if f1.Folder != f2.Folder {
				t.Errorf("Feed %q folder: got %q, want %q", url, f2.Folder, f1.Folder)
			}
		}

		// Compare folders
		folders1 := doc.Folders()
		folders2 := doc2.Folders()

		if len(folders2) != len(folders1) {
			t.Errorf("Folder count mismatch: got %d, want %d", len(folders2), len(folders1))
		}

		folderMap1 := make(map[string]bool)
		for _, f := range folders1 {
			folderMap1[f] = true
		}

		for _, f := range folders2 {
			if !folderMap1[f] {
				t.Errorf("Unexpected folder after round trip: %q", f)
			}
		}
	})

	t.Run("multiple write/parse cycles preserve data", func(t *testing.T) {
		doc := NewDocument("Cycle Test")
		doc.AddFeed("https://example.com/feed1", "Feed 1", "Tech")
		doc.AddFeed("https://example.com/feed2", "Feed 2", "News")

		// Perform 5 write/parse cycles
		var currentDoc = doc
		for i := 0; i < 5; i++ {
			var buf bytes.Buffer
			err := currentDoc.Write(&buf)
			if err != nil {
				t.Fatalf("Write() cycle %d failed: %v", i, err)
			}

			parsedDoc, err := Parse(&buf)
			if err != nil {
				t.Fatalf("Parse() cycle %d failed: %v", i, err)
			}

			currentDoc = parsedDoc
		}

		// Verify final document matches original
		origFeeds := doc.AllFeeds()
		finalFeeds := currentDoc.AllFeeds()

		if len(finalFeeds) != len(origFeeds) {
			t.Fatalf("Feed count after cycles: got %d, want %d", len(finalFeeds), len(origFeeds))
		}

		if currentDoc.Title != doc.Title {
			t.Errorf("Title after cycles: got %q, want %q", currentDoc.Title, doc.Title)
		}
	})

	t.Run("special XML characters are escaped and preserved", func(t *testing.T) {
		doc := NewDocument("Special <>&\" Characters Test")
		doc.AddFeed("https://example.com/feed?a=1&b=2", "Feed <>&\"", "Folder <>&\"")

		var buf bytes.Buffer
		err := doc.Write(&buf)
		if err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		// Verify XML contains escaped characters
		if !bytes.Contains(buf.Bytes(), []byte("&lt;")) || !bytes.Contains(buf.Bytes(), []byte("&gt;")) {
			t.Error("Special characters not properly escaped in XML")
		}

		// Parse back
		doc2, err := Parse(&buf)
		if err != nil {
			t.Fatalf("Parse() failed: %v", err)
		}

		// Verify characters are unescaped correctly
		if doc2.Title != doc.Title {
			t.Errorf("Title: got %q, want %q", doc2.Title, doc.Title)
		}

		feeds := doc2.AllFeeds()
		if len(feeds) != 1 {
			t.Fatalf("expected 1 feed, got %d", len(feeds))
		}

		if feeds[0].URL != "https://example.com/feed?a=1&b=2" {
			t.Errorf("URL: got %q, want %q", feeds[0].URL, "https://example.com/feed?a=1&b=2")
		}

		if feeds[0].Title != "Feed <>&\"" {
			t.Errorf("Title: got %q, want %q", feeds[0].Title, "Feed <>&\"")
		}

		if feeds[0].Folder != "Folder <>&\"" {
			t.Errorf("Folder: got %q, want %q", feeds[0].Folder, "Folder <>&\"")
		}
	})
}

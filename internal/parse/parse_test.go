// ABOUTME: Test suite for RSS/Atom feed parsing functionality
// ABOUTME: Validates parsing of RSS 2.0 and Atom feeds using inline XML test data

package parse

import (
	"testing"
	"time"
)

const rss20XML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <link>https://example.com</link>
    <description>A test RSS feed</description>
    <item>
      <guid>https://example.com/post/1</guid>
      <title>First Post</title>
      <link>https://example.com/post/1</link>
      <author>john@example.com (John Doe)</author>
      <pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
      <description>First post description</description>
      <category>tech</category>
      <category>golang</category>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/post/2</link>
      <pubDate>Tue, 03 Jan 2006 15:04:05 MST</pubDate>
      <description>Second post description</description>
    </item>
  </channel>
</rss>`

const atomXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <updated>2006-01-02T15:04:05Z</updated>
  <entry>
    <id>https://example.com/entry/1</id>
    <title>First Entry</title>
    <link href="https://example.com/entry/1"/>
    <author>
      <name>Jane Smith</name>
    </author>
    <published>2006-01-02T15:04:05Z</published>
    <updated>2006-01-02T16:04:05Z</updated>
    <content type="html">First entry content</content>
    <summary>First entry summary</summary>
    <category term="science"/>
  </entry>
  <entry>
    <id>https://example.com/entry/2</id>
    <title>Second Entry</title>
    <link href="https://example.com/entry/2"/>
    <updated>2006-01-03T15:04:05Z</updated>
    <summary>Second entry summary</summary>
  </entry>
</feed>`

func TestParse_RSS(t *testing.T) {
	feed, err := Parse([]byte(rss20XML))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if feed.Title != "Test RSS Feed" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Test RSS Feed")
	}

	if len(feed.Entries) != 2 {
		t.Fatalf("len(feed.Entries) = %d, want 2", len(feed.Entries))
	}

	// Check first entry
	entry1 := feed.Entries[0]
	if entry1.GUID != "https://example.com/post/1" {
		t.Errorf("entry1.GUID = %q, want %q", entry1.GUID, "https://example.com/post/1")
	}
	if entry1.Title != "First Post" {
		t.Errorf("entry1.Title = %q, want %q", entry1.Title, "First Post")
	}
	if entry1.Link != "https://example.com/post/1" {
		t.Errorf("entry1.Link = %q, want %q", entry1.Link, "https://example.com/post/1")
	}
	if entry1.Author != "John Doe" {
		t.Errorf("entry1.Author = %q, want %q", entry1.Author, "John Doe")
	}
	if entry1.PublishedAt == nil {
		t.Error("entry1.PublishedAt is nil, want non-nil")
	}
	if entry1.Content != "First post description" {
		t.Errorf("entry1.Content = %q, want %q", entry1.Content, "First post description")
	}
	if len(entry1.Categories) != 2 {
		t.Errorf("len(entry1.Categories) = %d, want 2", len(entry1.Categories))
	}
	if len(entry1.Categories) >= 2 && (entry1.Categories[0] != "tech" || entry1.Categories[1] != "golang") {
		t.Errorf("entry1.Categories = %v, want [tech golang]", entry1.Categories)
	}

	// Check second entry (no GUID, should fallback to Link)
	entry2 := feed.Entries[1]
	if entry2.GUID != "https://example.com/post/2" {
		t.Errorf("entry2.GUID = %q, want %q (fallback to Link)", entry2.GUID, "https://example.com/post/2")
	}
	if entry2.Title != "Second Post" {
		t.Errorf("entry2.Title = %q, want %q", entry2.Title, "Second Post")
	}
	if entry2.Author != "" {
		t.Errorf("entry2.Author = %q, want empty string", entry2.Author)
	}
}

func TestParse_Atom(t *testing.T) {
	feed, err := Parse([]byte(atomXML))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if feed.Title != "Test Atom Feed" {
		t.Errorf("feed.Title = %q, want %q", feed.Title, "Test Atom Feed")
	}

	if len(feed.Entries) != 2 {
		t.Fatalf("len(feed.Entries) = %d, want 2", len(feed.Entries))
	}

	// Check first entry
	entry1 := feed.Entries[0]
	if entry1.GUID != "https://example.com/entry/1" {
		t.Errorf("entry1.GUID = %q, want %q", entry1.GUID, "https://example.com/entry/1")
	}
	if entry1.Title != "First Entry" {
		t.Errorf("entry1.Title = %q, want %q", entry1.Title, "First Entry")
	}
	if entry1.Link != "https://example.com/entry/1" {
		t.Errorf("entry1.Link = %q, want %q", entry1.Link, "https://example.com/entry/1")
	}
	if entry1.Author != "Jane Smith" {
		t.Errorf("entry1.Author = %q, want %q", entry1.Author, "Jane Smith")
	}
	if entry1.PublishedAt == nil {
		t.Error("entry1.PublishedAt is nil, want non-nil")
	} else {
		expected := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
		if !entry1.PublishedAt.Equal(expected) {
			t.Errorf("entry1.PublishedAt = %v, want %v", entry1.PublishedAt, expected)
		}
	}
	if entry1.Content != "First entry content" {
		t.Errorf("entry1.Content = %q, want %q", entry1.Content, "First entry content")
	}
	if len(entry1.Categories) != 1 || entry1.Categories[0] != "science" {
		t.Errorf("entry1.Categories = %v, want [science]", entry1.Categories)
	}

	// Check second entry (no published date, should use updated)
	entry2 := feed.Entries[1]
	if entry2.GUID != "https://example.com/entry/2" {
		t.Errorf("entry2.GUID = %q, want %q", entry2.GUID, "https://example.com/entry/2")
	}
	if entry2.Title != "Second Entry" {
		t.Errorf("entry2.Title = %q, want %q", entry2.Title, "Second Entry")
	}
	if entry2.PublishedAt == nil {
		t.Error("entry2.PublishedAt is nil, want non-nil (should fallback to updated)")
	} else {
		expected := time.Date(2006, 1, 3, 15, 4, 5, 0, time.UTC)
		if !entry2.PublishedAt.Equal(expected) {
			t.Errorf("entry2.PublishedAt = %v, want %v", entry2.PublishedAt, expected)
		}
	}
	if entry2.Content != "Second entry summary" {
		t.Errorf("entry2.Content = %q, want %q (fallback to summary)", entry2.Content, "Second entry summary")
	}
}

// ABOUTME: Unit tests for feed discovery package
// ABOUTME: Tests direct feed, HTML link extraction, and common path probing

package discover

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>A test feed</description>
    <item>
      <title>Test Entry</title>
      <link>https://example.com/entry1</link>
      <guid>entry-1</guid>
    </item>
  </channel>
</rss>`

const testAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <entry>
    <title>Test Entry</title>
    <link href="https://example.com/entry1"/>
    <id>entry-1</id>
  </entry>
</feed>`

const testHTMLWithFeedLink = `<!DOCTYPE html>
<html>
<head>
  <title>Test Site</title>
  <link rel="alternate" type="application/rss+xml" title="RSS Feed" href="/feed.xml">
  <link rel="alternate" type="application/atom+xml" title="Atom Feed" href="/atom.xml">
</head>
<body>
  <h1>Test Site</h1>
</body>
</html>`

const testHTMLWithRelativeFeedLink = `<!DOCTYPE html>
<html>
<head>
  <title>Test Site</title>
  <link rel="alternate" type="application/rss+xml" href="feed.xml">
</head>
<body></body>
</html>`

const testHTMLNoFeedLinks = `<!DOCTYPE html>
<html>
<head>
  <title>Test Site</title>
</head>
<body>
  <h1>No feeds here</h1>
</body>
</html>`

func TestDiscover_DirectFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer server.Close()

	feed, err := Discover(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	if feed.URL != server.URL {
		t.Errorf("expected URL %s, got %s", server.URL, feed.URL)
	}

	if feed.Title != "Test Feed" {
		t.Errorf("expected title 'Test Feed', got '%s'", feed.Title)
	}
}

func TestDiscover_DirectAtomFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(testAtomFeed))
	}))
	defer server.Close()

	feed, err := Discover(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	if feed.Title != "Test Atom Feed" {
		t.Errorf("expected title 'Test Atom Feed', got '%s'", feed.Title)
	}
}

func TestDiscover_HTMLWithFeedLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLWithFeedLink))
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		case "/atom.xml":
			w.Header().Set("Content-Type", "application/atom+xml")
			w.Write([]byte(testAtomFeed))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	feed, err := Discover(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	expectedURL := server.URL + "/feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}

	if feed.Title != "Test Feed" {
		t.Errorf("expected title 'Test Feed', got '%s'", feed.Title)
	}
}

func TestDiscover_HTMLWithRelativeFeedLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blog/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLWithRelativeFeedLink))
		case "/blog/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	feed, err := Discover(server.URL + "/blog/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	expectedURL := server.URL + "/blog/feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_ProbeCommonPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLNoFeedLinks))
		case "/rss.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	feed, err := Discover(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	expectedURL := server.URL + "/rss.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_NoFeedFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLNoFeedLinks))
	}))
	defer server.Close()

	feed, err := Discover(server.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if feed != nil {
		t.Errorf("expected nil feed, got: %+v", feed)
	}

	if err != ErrNoFeedFound {
		t.Errorf("expected ErrNoFeedFound, got: %v", err)
	}
}

func TestDiscover_InvalidURL(t *testing.T) {
	_, err := Discover("not-a-valid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDiscover_MissingScheme(t *testing.T) {
	_, err := Discover("example.com/feed")
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestExtractFeedLinks_MultipleFeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(testHTMLWithFeedLink))
	}))
	defer server.Close()

	// Fetch the HTML
	result, _, err := tryDirectFeed(server.URL)
	if result != nil {
		t.Fatal("expected HTML page, not a feed")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsFeedContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/rss+xml", true},
		{"application/atom+xml", true},
		{"application/xml", true},
		{"text/xml", true},
		{"text/html", false},
		{"application/json", false},
		{"", false},
	}

	for _, tc := range tests {
		result := isFeedContentType(tc.contentType)
		if result != tc.expected {
			t.Errorf("isFeedContentType(%q) = %v, expected %v", tc.contentType, result, tc.expected)
		}
	}
}

func TestDiscover_AbsoluteFeedLink(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer feedServer.Close()

	htmlWithAbsoluteLink := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" href="` + feedServer.URL + `/feed">
</head>
<body></body>
</html>`

	htmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlWithAbsoluteLink))
	}))
	defer htmlServer.Close()

	feed, err := Discover(htmlServer.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	expectedURL := feedServer.URL + "/feed"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

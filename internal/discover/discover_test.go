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

// Test malformed HTML parsing edge cases

func TestDiscover_BrokenLinkTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			// Malformed HTML with broken link tags
			html := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml">
  <link href="/feed.xml">
  <link rel="alternate" type="application/rss+xml" href="/valid-feed.xml">
</head>
<body></body>
</html>`
			w.Write([]byte(html))
		case "/valid-feed.xml":
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

	expectedURL := server.URL + "/valid-feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_MultipleFeedsSameType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			html := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" title="Feed 1" href="/feed1.xml">
  <link rel="alternate" type="application/rss+xml" title="Feed 2" href="/feed2.xml">
  <link rel="alternate" type="application/rss+xml" title="Feed 3" href="/feed3.xml">
</head>
<body></body>
</html>`
			w.Write([]byte(html))
		case "/feed1.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		case "/feed2.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		case "/feed3.xml":
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

	// Should return the first valid feed
	expectedURL := server.URL + "/feed1.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_RelativeURLWithDotDot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blog/posts/":
			w.Header().Set("Content-Type", "text/html")
			html := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" href="../feed.xml">
</head>
<body></body>
</html>`
			w.Write([]byte(html))
		case "/blog/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(testRSSFeed))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	feed, err := Discover(server.URL + "/blog/posts/")
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

func TestDiscover_MalformedHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			// Severely malformed HTML but with valid link tag
			html := `<html><head><link rel="alternate" type="application/rss+xml" href="/feed.xml"</head><body>broken`
			w.Write([]byte(html))
		case "/feed.xml":
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

	expectedURL := server.URL + "/feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

// Test feed URL validation edge cases

func TestDiscover_URLWithUnusualPort(t *testing.T) {
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
}

func TestDiscover_URLWithQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "format=rss&limit=10" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer server.Close()

	feedURL := server.URL + "?format=rss&limit=10"
	feed, err := Discover(feedURL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	if feed.URL != feedURL {
		t.Errorf("expected URL %s, got %s", feedURL, feed.URL)
	}
}

func TestDiscover_URLWithFragment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(testRSSFeed))
	}))
	defer server.Close()

	// Fragments should be preserved in the URL
	feedURL := server.URL + "#section"
	feed, err := Discover(feedURL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed == nil {
		t.Fatal("expected feed, got nil")
	}

	if feed.URL != feedURL {
		t.Errorf("expected URL %s, got %s", feedURL, feed.URL)
	}
}

func TestDiscover_InvalidURLMalformedHost(t *testing.T) {
	_, err := Discover("http://[invalid-host")
	if err == nil {
		t.Fatal("expected error for malformed host")
	}
}

func TestDiscover_URLWithSpaces(t *testing.T) {
	_, err := Discover("http://example.com/feed with spaces.xml")
	if err == nil {
		t.Fatal("expected error for URL with spaces")
	}
}

func TestDiscover_EmptyURL(t *testing.T) {
	_, err := Discover("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestDiscover_URLWithoutHost(t *testing.T) {
	_, err := Discover("http://")
	if err == nil {
		t.Fatal("expected error for URL without host")
	}
}

// Test probe path behavior edge cases

func TestDiscover_ProbeReturns200ButNotFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLNoFeedLinks))
		case "/feed.xml", "/rss.xml", "/atom.xml":
			// These paths return 200 but with non-feed content
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>Not a feed</body></html>"))
		case "/index.xml":
			// This one is a valid feed
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

	// Should find the valid feed at /index.xml
	expectedURL := server.URL + "/index.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_ProbeAllPathsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Return HTML at root so we continue to probe
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLNoFeedLinks))
			return
		}
		// Return 404 for all common paths
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := Discover(server.URL)
	if err != ErrNoFeedFound {
		t.Errorf("expected ErrNoFeedFound, got: %v", err)
	}
}

func TestDiscover_ProbeServerError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(testHTMLNoFeedLinks))
			return
		}
		// Return 500 for all probe paths
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	_, err := Discover(server.URL)
	if err != ErrNoFeedFound {
		t.Errorf("expected ErrNoFeedFound, got: %v", err)
	}

	// Should have attempted to probe common paths
	if requestCount < 2 {
		t.Errorf("expected multiple probe attempts, got %d requests", requestCount)
	}
}

func TestDiscover_FeedLinkPointsToInvalidURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			html := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" href="ht!tp://invalid">
  <link rel="alternate" type="application/rss+xml" href="/valid-feed.xml">
</head>
<body></body>
</html>`
			w.Write([]byte(html))
		case "/valid-feed.xml":
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

	// Should skip invalid link and find valid one
	expectedURL := server.URL + "/valid-feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

func TestDiscover_FeedLinkPointsTo404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			html := `<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" href="/missing.xml">
  <link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head>
<body></body>
</html>`
			w.Write([]byte(html))
		case "/feed.xml":
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

	// Should skip 404 link and find valid one
	expectedURL := server.URL + "/feed.xml"
	if feed.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, feed.URL)
	}
}

// ABOUTME: Tests for HTTP fetcher with ETag and Last-Modified caching support.
// ABOUTME: Uses httptest to simulate server responses including 304 Not Modified.

package fetch_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harper/digest/internal/fetch"
)

func TestFetch_Fresh(t *testing.T) {
	// Server returns fresh content with cache headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header
		if ua := r.Header.Get("User-Agent"); ua != "digest/1.0 (RSS reader)" {
			t.Errorf("expected User-Agent 'digest/1.0 (RSS reader)', got %q", ua)
		}

		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss>test content</rss>"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NotModified {
		t.Error("expected NotModified=false for fresh fetch")
	}

	if string(result.Body) != "<rss>test content</rss>" {
		t.Errorf("expected body '<rss>test content</rss>', got %q", string(result.Body))
	}

	if result.ETag != `"abc123"` {
		t.Errorf("expected ETag '\"abc123\"', got %q", result.ETag)
	}

	if result.LastModified != "Mon, 02 Jan 2006 15:04:05 GMT" {
		t.Errorf("expected LastModified 'Mon, 02 Jan 2006 15:04:05 GMT', got %q", result.LastModified)
	}
}

func TestFetch_Cached(t *testing.T) {
	// Server returns 304 Not Modified when If-None-Match matches
	etag := `"abc123"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify If-None-Match header was sent
		if inm := r.Header.Get("If-None-Match"); inm != etag {
			t.Errorf("expected If-None-Match %q, got %q", etag, inm)
		}

		// Return 304 Not Modified
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	result, err := fetch.Fetch(server.URL, &etag, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NotModified {
		t.Error("expected NotModified=true for 304 response")
	}

	if len(result.Body) != 0 {
		t.Errorf("expected empty body for 304 response, got %d bytes", len(result.Body))
	}
}

func TestFetch_Error(t *testing.T) {
	// Server returns 404, expect error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

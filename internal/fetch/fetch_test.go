// ABOUTME: Tests for HTTP fetcher with ETag and Last-Modified caching support.
// ABOUTME: Uses httptest to simulate server responses including 304 Not Modified.

package fetch_test

import (
	"context"
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

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
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

	result, err := fetch.Fetch(context.Background(), server.URL, &etag, nil)
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

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

// HTTP Edge Cases Tests

func TestFetch_MalformedETag(t *testing.T) {
	// Server returns malformed ETag header (missing quotes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "malformed-no-quotes")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss>test</rss>"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still accept and store the malformed ETag
	if result.ETag != "malformed-no-quotes" {
		t.Errorf("expected ETag 'malformed-no-quotes', got %q", result.ETag)
	}

	if result.NotModified {
		t.Error("expected NotModified=false for fresh fetch")
	}
}

func TestFetch_EmptyETag(t *testing.T) {
	// Server returns empty ETag header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss>test</rss>"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ETag != "" {
		t.Errorf("expected empty ETag, got %q", result.ETag)
	}

	if result.NotModified {
		t.Error("expected NotModified=false for fresh fetch")
	}
}

func TestFetch_EmptyLastModified(t *testing.T) {
	// Server returns empty Last-Modified header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss>test</rss>"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LastModified != "" {
		t.Errorf("expected empty LastModified, got %q", result.LastModified)
	}

	if result.NotModified {
		t.Error("expected NotModified=false for fresh fetch")
	}
}

func TestFetch_LargeResponseBody(t *testing.T) {
	// Server returns very large response body (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"large123"`)
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Body) != 1024*1024 {
		t.Errorf("expected body length 1048576, got %d", len(result.Body))
	}

	if result.ETag != `"large123"` {
		t.Errorf("expected ETag '\"large123\"', got %q", result.ETag)
	}
}

// HTTP Status Code Tests

func TestFetch_304WithCachingHeaders(t *testing.T) {
	// Server returns 304 Not Modified with caching headers
	etag := `"cache123"`
	lastModified := "Mon, 02 Jan 2006 15:04:05 GMT"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify both conditional headers were sent
		if inm := r.Header.Get("If-None-Match"); inm != etag {
			t.Errorf("expected If-None-Match %q, got %q", etag, inm)
		}
		if ims := r.Header.Get("If-Modified-Since"); ims != lastModified {
			t.Errorf("expected If-Modified-Since %q, got %q", lastModified, ims)
		}

		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, &etag, &lastModified)
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

func TestFetch_400BadRequest(t *testing.T) {
	// Server returns 400 Bad Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

func TestFetch_403Forbidden(t *testing.T) {
	// Server returns 403 Forbidden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

func TestFetch_500InternalServerError(t *testing.T) {
	// Server returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

func TestFetch_502BadGateway(t *testing.T) {
	// Server returns 502 Bad Gateway
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 502 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

func TestFetch_503ServiceUnavailable(t *testing.T) {
	// Server returns 503 Service Unavailable
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for 503 response, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %+v", result)
	}
}

// Content Validation Tests

func TestFetch_EmptyResponseBody(t *testing.T) {
	// Server returns 200 OK with empty body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"empty123"`)
		w.WriteHeader(http.StatusOK)
		// No body written
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Body) != 0 {
		t.Errorf("expected empty body, got %d bytes", len(result.Body))
	}

	if result.NotModified {
		t.Error("expected NotModified=false for 200 response")
	}

	if result.ETag != `"empty123"` {
		t.Errorf("expected ETag '\"empty123\"', got %q", result.ETag)
	}
}

func TestFetch_UnexpectedContentType(t *testing.T) {
	// Server returns 200 OK with unexpected content type (should still work)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"json123"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"not": "rss"}`))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result.Body) != `{"not": "rss"}` {
		t.Errorf("expected body '{\"not\": \"rss\"}', got %q", string(result.Body))
	}

	if result.NotModified {
		t.Error("expected NotModified=false for fresh fetch")
	}

	if result.ETag != `"json123"` {
		t.Errorf("expected ETag '\"json123\"', got %q", result.ETag)
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	// Invalid URL should return error
	_, err := fetch.Fetch(context.Background(), "://invalid-url", nil, nil)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFetch_EmptyEtagAndLastModified(t *testing.T) {
	// Test with empty string pointers (should not set headers)
	emptyStr := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no conditional headers were sent
		if inm := r.Header.Get("If-None-Match"); inm != "" {
			t.Errorf("expected no If-None-Match, got %q", inm)
		}
		if ims := r.Header.Get("If-Modified-Since"); ims != "" {
			t.Errorf("expected no If-Modified-Since, got %q", ims)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<rss>test</rss>"))
	}))
	defer server.Close()

	result, err := fetch.Fetch(context.Background(), server.URL, &emptyStr, &emptyStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NotModified {
		t.Error("expected NotModified=false")
	}
}

func TestResultStruct(t *testing.T) {
	// Test Result struct fields
	result := fetch.Result{
		Body:         []byte("test"),
		ETag:         "etag",
		LastModified: "modified",
		NotModified:  true,
	}

	if string(result.Body) != "test" {
		t.Errorf("expected Body 'test', got %q", string(result.Body))
	}
	if result.ETag != "etag" {
		t.Errorf("expected ETag 'etag', got %q", result.ETag)
	}
	if result.LastModified != "modified" {
		t.Errorf("expected LastModified 'modified', got %q", result.LastModified)
	}
	if !result.NotModified {
		t.Error("expected NotModified true")
	}
}

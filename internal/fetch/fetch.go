// ABOUTME: HTTP fetcher with support for conditional requests using ETag and Last-Modified headers.
// ABOUTME: Returns 304 Not Modified status when content hasn't changed, enabling efficient polling.

package fetch

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Result contains the response from an HTTP fetch operation.
type Result struct {
	Body         []byte
	ETag         string
	LastModified string
	NotModified  bool
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Fetch retrieves a URL with optional conditional request headers.
// If etag is provided, sets If-None-Match header.
// If lastModified is provided, sets If-Modified-Since header.
// Returns NotModified=true for 304 responses.
// Returns error for non-200/304 status codes.
func Fetch(url string, etag, lastModified *string) (*Result, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "digest/1.0 (RSS reader)")

	if etag != nil && *etag != "" {
		req.Header.Set("If-None-Match", *etag)
	}

	if lastModified != nil && *lastModified != "" {
		req.Header.Set("If-Modified-Since", *lastModified)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return &Result{
			NotModified: true,
		}, nil
	}

	// Handle non-200 status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Result{
		Body:         body,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		NotModified:  false,
	}, nil
}

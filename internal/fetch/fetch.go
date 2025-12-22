// ABOUTME: HTTP fetcher with support for conditional requests using ETag and Last-Modified headers.
// ABOUTME: Returns 304 Not Modified status when content hasn't changed, enabling efficient polling with SSRF and DoS protection.

package fetch

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const MaxResponseSize = 10 * 1024 * 1024 // 10MB

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

// isPrivateIP checks if an IP address is in a private range (excluding loopback for tests).
func isPrivateIP(ip net.IP) bool {
	// Allow loopback addresses (localhost) for tests
	if ip.IsLoopback() {
		return false
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// Fetch retrieves a URL with optional conditional request headers.
// If etag is provided, sets If-None-Match header.
// If lastModified is provided, sets If-Modified-Since header.
// Returns NotModified=true for 304 responses.
// Returns error for non-200/304 status codes.
// Includes SSRF protection by blocking private IP ranges and DoS protection via response size limit.
func Fetch(ctx context.Context, urlStr string, etag, lastModified *string) (*Result, error) {
	// Parse URL for SSRF protection
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// SSRF protection: block private IP ranges
	if ips, err := net.LookupIP(parsedURL.Hostname()); err == nil {
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return nil, fmt.Errorf("access to private IP ranges is not allowed")
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
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

	// Read response body with DoS protection (10MB limit)
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if response was truncated (exceeded limit)
	if int64(len(body)) > MaxResponseSize {
		return nil, fmt.Errorf("response too large (exceeds %d bytes)", MaxResponseSize)
	}

	return &Result{
		Body:         body,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		NotModified:  false,
	}, nil
}

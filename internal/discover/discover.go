// ABOUTME: Feed discovery package for finding RSS/Atom feeds from URLs
// ABOUTME: Supports direct feeds, HTML link headers, and common path probing

package discover

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/harper/digest/internal/fetch"
	"github.com/harper/digest/internal/parse"
	"golang.org/x/net/html"
)

// Common feed paths to probe when other discovery methods fail
var commonFeedPaths = []string{
	"/feed.xml",
	"/feed",
	"/rss.xml",
	"/rss",
	"/atom.xml",
	"/atom",
	"/index.xml",
	"/feed/rss",
	"/feed/atom",
	"/feeds/posts/default",
}

// Errors returned by discovery functions
var (
	ErrNoFeedFound = errors.New("no RSS/Atom feed found at URL")
	ErrInvalidURL  = errors.New("invalid URL")
)

// DiscoveredFeed represents a feed found during discovery
type DiscoveredFeed struct {
	URL   string // Absolute URL of the feed
	Title string // Feed title (from content or link element)
}

// Discover attempts to find an RSS/Atom feed from the given URL.
// It tries the following strategies in order:
//  1. Parse URL as a direct feed
//  2. Parse URL as HTML and extract <link rel="alternate"> headers
//  3. Probe common feed URL patterns
//
// Returns the discovered feed, or an error if none found.
func Discover(inputURL string) (*DiscoveredFeed, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("%w: missing scheme or host", ErrInvalidURL)
	}

	// Strategy 1: Try direct feed
	feed, body, err := tryDirectFeed(inputURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	if feed != nil {
		return feed, nil
	}

	// Strategy 2: Extract feed links from HTML
	feeds, err := extractFeedLinks(body, parsedURL)
	if err == nil && len(feeds) > 0 {
		// Verify the first discovered link is a valid feed
		for _, candidate := range feeds {
			verifiedFeed, _, verifyErr := tryDirectFeed(candidate.URL)
			if verifyErr == nil && verifiedFeed != nil {
				// Use title from HTML link if feed doesn't have one
				if verifiedFeed.Title == "" && candidate.Title != "" {
					verifiedFeed.Title = candidate.Title
				}
				return verifiedFeed, nil
			}
		}
	}

	// Strategy 3: Probe common paths
	feed, err = probeCommonPaths(parsedURL)
	if err == nil && feed != nil {
		return feed, nil
	}

	return nil, ErrNoFeedFound
}

// tryDirectFeed attempts to fetch and parse the URL as an RSS/Atom feed.
// Returns the feed if successful, or nil if the content is not a valid feed.
// Also returns the raw body for use in HTML parsing if it's not a feed.
func tryDirectFeed(feedURL string) (*DiscoveredFeed, []byte, error) {
	result, err := fetch.Fetch(feedURL, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	// Try to parse as a feed
	parsed, parseErr := parse.Parse(result.Body)
	if parseErr != nil {
		// Not a valid feed, return body for HTML parsing (not an error condition)
		return nil, result.Body, nil //nolint:nilerr // parseErr means not a feed, which is expected
	}

	return &DiscoveredFeed{
		URL:   feedURL,
		Title: parsed.Title,
	}, result.Body, nil
}

// extractFeedLinks parses HTML and returns feed URLs from <link rel="alternate"> elements
func extractFeedLinks(htmlBody []byte, baseURL *url.URL) ([]DiscoveredFeed, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBody)))
	if err != nil {
		return nil, err
	}

	var feeds []DiscoveredFeed
	var findLinks func(*html.Node)
	findLinks = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, linkType, href, title string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "type":
					linkType = attr.Val
				case "href":
					href = attr.Val
				case "title":
					title = attr.Val
				}
			}

			// Check if this is an alternate feed link
			if rel == "alternate" && isFeedContentType(linkType) && href != "" {
				resolvedURL, err := resolveURL(href, baseURL)
				if err == nil {
					feeds = append(feeds, DiscoveredFeed{
						URL:   resolvedURL,
						Title: title,
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLinks(c)
		}
	}

	findLinks(doc)
	return feeds, nil
}

// probeCommonPaths tries common feed URL patterns against the base URL
func probeCommonPaths(baseURL *url.URL) (*DiscoveredFeed, error) {
	// Build base URL without path
	probeBase := &url.URL{
		Scheme: baseURL.Scheme,
		Host:   baseURL.Host,
	}

	for _, path := range commonFeedPaths {
		probeURL := probeBase.String() + path
		feed, _, err := tryDirectFeed(probeURL)
		if err == nil && feed != nil {
			return feed, nil
		}
	}

	return nil, ErrNoFeedFound
}

// resolveURL resolves a potentially relative URL against a base URL
func resolveURL(href string, baseURL *url.URL) (string, error) {
	refURL, err := url.Parse(href)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}

// isFeedContentType checks if the content type indicates a feed
func isFeedContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "rss") ||
		strings.Contains(contentType, "atom") ||
		strings.Contains(contentType, "xml")
}

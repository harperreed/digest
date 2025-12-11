// ABOUTME: Content processing utilities for feed entries
// ABOUTME: Detects HTML and converts to Markdown for clean display

package content

import (
	"regexp"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// htmlTagPattern matches common HTML tags
var htmlTagPattern = regexp.MustCompile(`<\s*(p|div|span|a|br|img|h[1-6]|ul|ol|li|table|tr|td|th|strong|em|b|i|code|pre|blockquote)[^>]*>`)

// IsHTML checks if content appears to be HTML
func IsHTML(content string) bool {
	// Quick checks for obvious HTML markers
	if strings.Contains(content, "<!DOCTYPE") || strings.Contains(content, "<html") {
		return true
	}

	// Check for common HTML tags
	return htmlTagPattern.MatchString(content)
}

// ToMarkdown converts HTML content to Markdown
// If the content doesn't appear to be HTML, returns it unchanged
func ToMarkdown(content string) string {
	if content == "" {
		return content
	}

	if !IsHTML(content) {
		return content
	}

	markdown, err := htmltomarkdown.ConvertString(content)
	if err != nil {
		// If conversion fails, return original content
		return content
	}

	// Clean up excessive whitespace
	markdown = strings.TrimSpace(markdown)

	return markdown
}

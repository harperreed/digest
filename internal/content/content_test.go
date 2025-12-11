// ABOUTME: Tests for content processing utilities
// ABOUTME: Validates HTML detection and Markdown conversion

package content

import (
	"strings"
	"testing"
)

func TestIsHTML(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "plain text",
			content:  "This is just plain text without any HTML.",
			expected: false,
		},
		{
			name:     "paragraph tag",
			content:  "<p>This is a paragraph.</p>",
			expected: true,
		},
		{
			name:     "div tag",
			content:  "<div class=\"content\">Some content</div>",
			expected: true,
		},
		{
			name:     "link tag",
			content:  "Check out <a href=\"https://example.com\">this link</a>.",
			expected: true,
		},
		{
			name:     "DOCTYPE",
			content:  "<!DOCTYPE html><html><body>Test</body></html>",
			expected: true,
		},
		{
			name:     "br tag",
			content:  "Line one<br>Line two",
			expected: true,
		},
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "angle brackets but not HTML",
			content:  "5 < 10 and 10 > 5",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHTML(tt.content)
			if result != tt.expected {
				t.Errorf("IsHTML(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // strings that should be in the output
		excludes []string // strings that should NOT be in the output
	}{
		{
			name:     "plain text unchanged",
			input:    "Just plain text here.",
			contains: []string{"Just plain text here."},
		},
		{
			name:     "paragraph to text",
			input:    "<p>A paragraph of text.</p>",
			contains: []string{"A paragraph of text."},
			excludes: []string{"<p>", "</p>"},
		},
		{
			name:     "link to markdown",
			input:    "<a href=\"https://example.com\">Example</a>",
			contains: []string{"[Example]", "(https://example.com)"},
			excludes: []string{"<a", "</a>"},
		},
		{
			name:     "bold to markdown",
			input:    "<strong>Bold text</strong>",
			contains: []string{"**Bold text**"},
			excludes: []string{"<strong>"},
		},
		{
			name:     "italic to markdown",
			input:    "<em>Italic text</em>",
			contains: []string{"*Italic text*"},
			excludes: []string{"<em>"},
		},
		{
			name:     "list to markdown",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			contains: []string{"Item 1", "Item 2"},
			excludes: []string{"<ul>", "<li>"},
		},
		{
			name:     "empty string",
			input:    "",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToMarkdown(tt.input)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("ToMarkdown() result should contain %q, got %q", s, result)
				}
			}

			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("ToMarkdown() result should NOT contain %q, got %q", s, result)
				}
			}
		})
	}
}

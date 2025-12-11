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
		{
			name:     "math expressions with angle brackets",
			content:  "if x < y and y > z then x < z",
			expected: false,
		},
		{
			name:     "generic angle brackets",
			content:  "List<String> and Map<String, Integer>",
			expected: false,
		},
		{
			name:     "angle brackets in code comment",
			content:  "// Check if value < threshold",
			expected: false,
		},
		{
			name:     "tag with closing tag only",
			content:  "No opening tag </p> here",
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
		{
			name:     "nested lists",
			input:    "<ul><li>Level 1<ul><li>Level 2a</li><li>Level 2b</li></ul></li><li>Level 1 again</li></ul>",
			contains: []string{"Level 1", "Level 2a", "Level 2b", "Level 1 again"},
			excludes: []string{"<ul>", "<li>"},
		},
		{
			name:     "table structure",
			input:    "<table><tr><th>Header 1</th><th>Header 2</th></tr><tr><td>Cell 1</td><td>Cell 2</td></tr></table>",
			contains: []string{"Header 1", "Header 2", "Cell 1", "Cell 2"},
			excludes: []string{"<table>", "<tr>", "<td>", "<th>"},
		},
		{
			name:     "HTML with embedded CSS",
			input:    "<p style=\"color: red; font-size: 16px;\">Styled text</p><style>.test { color: blue; }</style>",
			contains: []string{"Styled text"},
			excludes: []string{"style=", "color: red", ".test", "<style>"},
		},
		{
			name:     "HTML with embedded JavaScript",
			input:    "<p>Content</p><script>alert('test');</script><p>More content</p>",
			contains: []string{"Content", "More content"},
			excludes: []string{"<script>", "alert", "test"},
		},
		{
			name:     "malformed HTML missing closing tags",
			input:    "<p>Paragraph without closing<div>Div without closing<span>Span text",
			contains: []string{"Paragraph without closing", "Div without closing", "Span text"},
			excludes: []string{"<p>", "<div>", "<span>"},
		},
		{
			name:     "complex nested structure",
			input:    "<div><p>Outer <strong>bold <em>and italic</em></strong> text</p></div>",
			contains: []string{"Outer", "bold", "and italic", "text"},
			excludes: []string{"<div>", "<p>"},
		},
		{
			name:     "image with alt text",
			input:    "<img src=\"https://example.com/image.jpg\" alt=\"A beautiful sunset\">",
			contains: []string{"beautiful sunset"},
			excludes: []string{"<img"},
		},
		{
			name:     "link with nested formatting",
			input:    "<a href=\"https://example.com\"><strong>Bold</strong> <em>link</em></a>",
			contains: []string{"Bold", "link", "https://example.com"},
			excludes: []string{"<a", "<strong>", "<em>"},
		},
		{
			name:     "code blocks",
			input:    "<pre><code>function test() { return true; }</code></pre>",
			contains: []string{"function test()", "return true"},
			excludes: []string{"<pre>", "<code>"},
		},
		{
			name:     "blockquote",
			input:    "<blockquote>This is a quoted text</blockquote>",
			contains: []string{"quoted text"},
			excludes: []string{"<blockquote>"},
		},
		{
			name:     "multiple links preserve structure",
			input:    "<p>Check <a href=\"https://one.com\">first</a> and <a href=\"https://two.com\">second</a> links.</p>",
			contains: []string{"[first]", "(https://one.com)", "[second]", "(https://two.com)"},
			excludes: []string{"<a", "</a>"},
		},
		{
			name:     "headings conversion",
			input:    "<h1>Main Title</h1><h2>Subtitle</h2><h3>Section</h3>",
			contains: []string{"Main Title", "Subtitle", "Section"},
			excludes: []string{"<h1>", "<h2>", "<h3>"},
		},
		{
			name:     "ordered list",
			input:    "<ol><li>First item</li><li>Second item</li><li>Third item</li></ol>",
			contains: []string{"First item", "Second item", "Third item"},
			excludes: []string{"<ol>", "<li>"},
		},
		{
			name:     "mixed inline formatting",
			input:    "<p>Text with <b>bold</b>, <i>italic</i>, and <code>code</code> elements.</p>",
			contains: []string{"bold", "italic", "code", "elements"},
			excludes: []string{"<b>", "<i>", "<code>"},
		},
		{
			name:     "HTML entities basic decoding",
			input:    "<p>Tom &amp; Jerry</p>",
			contains: []string{"Tom", "Jerry", "&"},
			excludes: []string{"<p>"},
		},
		{
			name: "complex document structure",
			input: `<article>
				<header><h1>Article Title</h1></header>
				<section>
					<p>First paragraph with <a href="https://example.com">a link</a>.</p>
					<ul><li>Point one</li><li>Point two</li></ul>
				</section>
				<footer>Copyright 2024</footer>
			</article>`,
			contains: []string{"Article Title", "First paragraph", "link", "Point one", "Point two", "Copyright 2024"},
			excludes: []string{"<article>", "<header>", "<section>", "<footer>"},
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

func TestToMarkdownEdgeCases(t *testing.T) {
	t.Run("handles deeply nested structures without crashing", func(t *testing.T) {
		// Build a deeply nested structure
		deeply := "<div>"
		for i := 0; i < 50; i++ {
			deeply += "<div>"
		}
		deeply += "Deep content"
		for i := 0; i < 50; i++ {
			deeply += "</div>"
		}
		deeply += "</div>"

		result := ToMarkdown(deeply)
		if !strings.Contains(result, "Deep content") {
			t.Errorf("Should handle deeply nested structures and extract content")
		}
	})

	t.Run("handles very long content", func(t *testing.T) {
		longContent := "<p>" + strings.Repeat("word ", 10000) + "</p>"
		result := ToMarkdown(longContent)
		if len(result) < 10000 {
			t.Errorf("Should handle long content without truncation")
		}
	})

	t.Run("handles HTML with only whitespace", func(t *testing.T) {
		result := ToMarkdown("<p>   </p>")
		if result != "" {
			t.Errorf("Whitespace-only HTML should result in empty string after trimming, got %q", result)
		}
	})

	t.Run("handles unclosed tags gracefully", func(t *testing.T) {
		input := "<div><p>Text without closing tags"
		result := ToMarkdown(input)
		if !strings.Contains(result, "Text without closing tags") {
			t.Errorf("Should extract text from unclosed tags, got %q", result)
		}
	})

	t.Run("preserves markdown when input is plain text", func(t *testing.T) {
		input := "# Already markdown\n\n**Bold** and *italic*"
		result := ToMarkdown(input)
		if result != input {
			t.Errorf("Plain markdown text should be unchanged, got %q", result)
		}
	})
}

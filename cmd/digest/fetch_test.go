// ABOUTME: Tests for fetch command helper functions
// ABOUTME: Verifies feed display name logic and sync behavior

package main

import (
	"testing"

	"github.com/harper/digest/internal/models"
)

func TestFeedDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		title    *string
		url      string
		expected string
	}{
		{
			name:     "nil title returns URL",
			title:    nil,
			url:      "https://example.com/feed.xml",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "empty title returns URL",
			title:    stringPtr(""),
			url:      "https://example.com/feed.xml",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "valid title returns title",
			title:    stringPtr("My Feed"),
			url:      "https://example.com/feed.xml",
			expected: "My Feed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := models.NewFeed(tt.url)
			feed.Title = tt.title

			got := feedDisplayName(feed)
			if got != tt.expected {
				t.Errorf("feedDisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

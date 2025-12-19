// ABOUTME: Tests for MCP helper functions
// ABOUTME: Covers date parsing and formatting utilities

package mcp

import (
	"testing"
	"time"
)

func TestParseDateString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t time.Time) bool
	}{
		{
			name:    "today",
			input:   "today",
			wantErr: false,
			check: func(parsed time.Time) bool {
				// Should be start of today
				now := time.Now()
				return parsed.Year() == now.Year() &&
					parsed.Month() == now.Month() &&
					parsed.Day() == now.Day()
			},
		},
		{
			name:    "yesterday",
			input:   "yesterday",
			wantErr: false,
			check: func(parsed time.Time) bool {
				yesterday := time.Now().Add(-24 * time.Hour)
				return parsed.Year() == yesterday.Year() &&
					parsed.Month() == yesterday.Month() &&
					parsed.Day() == yesterday.Day()
			},
		},
		{
			name:    "week",
			input:   "week",
			wantErr: false,
			check: func(parsed time.Time) bool {
				// Should be within the last 7 days
				weekAgo := time.Now().Add(-7 * 24 * time.Hour)
				return parsed.After(weekAgo.Add(-time.Hour)) &&
					parsed.Before(time.Now())
			},
		},
		{
			name:    "month",
			input:   "month",
			wantErr: false,
			check: func(parsed time.Time) bool {
				// Should be within the last 30 days
				monthAgo := time.Now().Add(-30 * 24 * time.Hour)
				return parsed.After(monthAgo.Add(-time.Hour)) &&
					parsed.Before(time.Now())
			},
		},
		{
			name:    "ISO date format",
			input:   "2024-12-15",
			wantErr: false,
			check: func(parsed time.Time) bool {
				return parsed.Year() == 2024 &&
					parsed.Month() == time.December &&
					parsed.Day() == 15
			},
		},
		{
			name:    "RFC3339 format",
			input:   "2024-12-15T10:30:00Z",
			wantErr: false,
			check: func(parsed time.Time) bool {
				return parsed.Year() == 2024 &&
					parsed.Month() == time.December &&
					parsed.Day() == 15 &&
					parsed.Hour() == 10 &&
					parsed.Minute() == 30
			},
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "partial date",
			input:   "2024-12",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDateString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				if !tt.check(result) {
					t.Errorf("parseDateString(%q) = %v, failed check", tt.input, result)
				}
			}
		})
	}
}

func TestFormatFolder(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "root level"},
		{"Tech", "'Tech'"},
		{"Tech Blogs", "'Tech Blogs'"},
		{"news", "'news'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatFolder(tt.input)
			if got != tt.want {
				t.Errorf("formatFolder(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

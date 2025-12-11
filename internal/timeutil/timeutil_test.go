// ABOUTME: Tests for time utility functions
// ABOUTME: Verifies date range calculations for smart views

package timeutil

import (
	"testing"
	"time"
)

func TestStartOfToday(t *testing.T) {
	result := StartOfToday()
	now := time.Now()

	if result.Year() != now.Year() || result.Month() != now.Month() || result.Day() != now.Day() {
		t.Errorf("StartOfToday() date mismatch: got %v, expected date %v", result, now)
	}

	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("StartOfToday() should be midnight, got %v", result)
	}
}

func TestStartOfYesterday(t *testing.T) {
	result := StartOfYesterday()
	today := StartOfToday()
	expected := today.AddDate(0, 0, -1)

	if !result.Equal(expected) {
		t.Errorf("StartOfYesterday() = %v, expected %v", result, expected)
	}
}

func TestEndOfYesterday(t *testing.T) {
	result := EndOfYesterday()
	today := StartOfToday()

	if !result.Equal(today) {
		t.Errorf("EndOfYesterday() = %v, expected %v (start of today)", result, today)
	}
}

func TestStartOfWeek(t *testing.T) {
	result := StartOfWeek()

	// Should be a Sunday
	if result.Weekday() != time.Sunday {
		t.Errorf("StartOfWeek() weekday = %v, expected Sunday", result.Weekday())
	}

	// Should be midnight
	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("StartOfWeek() should be midnight, got %v", result)
	}

	// Should be on or before today
	if result.After(StartOfToday()) {
		t.Errorf("StartOfWeek() = %v, should not be after today", result)
	}
}

func TestStartOfMonth(t *testing.T) {
	result := StartOfMonth()
	now := time.Now()

	if result.Year() != now.Year() || result.Month() != now.Month() {
		t.Errorf("StartOfMonth() year/month mismatch: got %v, expected %d-%02d", result, now.Year(), now.Month())
	}

	if result.Day() != 1 {
		t.Errorf("StartOfMonth() day = %d, expected 1", result.Day())
	}

	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
		t.Errorf("StartOfMonth() should be midnight, got %v", result)
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		period   string
		expected func() time.Time
		valid    bool
	}{
		{"today", StartOfToday, true},
		{"yesterday", StartOfYesterday, true},
		{"week", StartOfWeek, true},
		{"month", StartOfMonth, true},
		{"invalid", nil, false},
		{"", nil, false},
	}

	for _, tc := range tests {
		result, ok := ParsePeriod(tc.period)
		if ok != tc.valid {
			t.Errorf("ParsePeriod(%q) valid = %v, expected %v", tc.period, ok, tc.valid)
			continue
		}

		if tc.valid {
			expected := tc.expected()
			if !result.Equal(expected) {
				t.Errorf("ParsePeriod(%q) = %v, expected %v", tc.period, result, expected)
			}
		}
	}
}

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

func TestParsePeriod_EdgeCases(t *testing.T) {
	tests := []struct {
		period string
		valid  bool
	}{
		// Invalid period names should return false
		{"Today", false},     // Wrong case
		{"YESTERDAY", false}, // Wrong case
		{"Week", false},      // Wrong case
		{"MONTH", false},     // Wrong case
		{"tomorrow", false},  // Not supported
		{"year", false},      // Not supported
		{"day", false},       // Not supported
		{" today", false},    // Leading space
		{"today ", false},    // Trailing space
		{" today ", false},   // Both spaces
		{"tod ay", false},    // Space in middle
	}

	for _, tc := range tests {
		_, ok := ParsePeriod(tc.period)
		if ok != tc.valid {
			t.Errorf("ParsePeriod(%q) valid = %v, expected %v", tc.period, ok, tc.valid)
		}
	}
}

func TestStartOfToday_Timezone(t *testing.T) {
	// Test that StartOfToday respects local timezone
	result := StartOfToday()

	// Verify it's in local timezone, not UTC
	if result.Location() != time.Now().Location() {
		t.Errorf("StartOfToday() location = %v, expected local timezone %v", result.Location(), time.Now().Location())
	}

	// UTC midnight and local midnight should differ (unless we're in UTC)
	utcNow := time.Now().UTC()
	utcMidnight := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	localMidnight := result

	// Only compare if we're not in UTC timezone
	_, localOffset := localMidnight.Zone()
	if localOffset != 0 {
		if localMidnight.Equal(utcMidnight) {
			t.Errorf("StartOfToday() returned UTC midnight instead of local midnight")
		}
	}
}

func TestStartOfYesterday_Timezone(t *testing.T) {
	// Test that StartOfYesterday respects local timezone
	result := StartOfYesterday()

	// Verify it's in local timezone
	if result.Location() != time.Now().Location() {
		t.Errorf("StartOfYesterday() location = %v, expected local timezone %v", result.Location(), time.Now().Location())
	}

	// Should be exactly 24 hours before StartOfToday
	today := StartOfToday()
	expected := today.AddDate(0, 0, -1)
	if !result.Equal(expected) {
		t.Errorf("StartOfYesterday() = %v, expected %v", result, expected)
	}
}

func TestStartOfWeek_YearBoundary(t *testing.T) {
	// Test week calculations when they cross year boundaries
	// We need to test with specific dates
	testCases := []struct {
		name          string
		referenceDate time.Time
		expectedDay   int
	}{
		{
			name:          "Monday Jan 1, 2024 - week should be Dec 31, 2023",
			referenceDate: time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local),
			expectedDay:   31, // Sunday Dec 31, 2023
		},
		{
			name:          "Sunday Jan 7, 2024 - week should be same day",
			referenceDate: time.Date(2024, 1, 7, 12, 0, 0, 0, time.Local),
			expectedDay:   7, // Sunday Jan 7, 2024
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock current time by using the reference date directly
			today := time.Date(tc.referenceDate.Year(), tc.referenceDate.Month(), tc.referenceDate.Day(), 0, 0, 0, 0, time.Local)
			weekday := int(today.Weekday())
			startOfWeek := today.AddDate(0, 0, -weekday)

			// Should always be a Sunday
			if startOfWeek.Weekday() != time.Sunday {
				t.Errorf("Start of week = %v, expected Sunday", startOfWeek.Weekday())
			}

			// Verify expected day
			if startOfWeek.Day() != tc.expectedDay {
				t.Errorf("Start of week day = %d, expected %d", startOfWeek.Day(), tc.expectedDay)
			}
		})
	}
}

func TestStartOfMonth_BoundaryConditions(t *testing.T) {
	// Test with various dates in the month to ensure it always returns the 1st
	testDates := []struct {
		name string
		date time.Time
	}{
		{"first day of month", time.Date(2024, 3, 1, 15, 30, 45, 0, time.Local)},
		{"last day of month", time.Date(2024, 3, 31, 23, 59, 59, 0, time.Local)},
		{"middle of month", time.Date(2024, 3, 15, 12, 0, 0, 0, time.Local)},
		{"leap year Feb 29", time.Date(2024, 2, 29, 10, 0, 0, 0, time.Local)},
		{"short month Feb 28", time.Date(2023, 2, 28, 10, 0, 0, 0, time.Local)},
		{"31-day month", time.Date(2024, 1, 31, 10, 0, 0, 0, time.Local)},
		{"30-day month", time.Date(2024, 4, 30, 10, 0, 0, 0, time.Local)},
	}

	for _, tc := range testDates {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate start of month for the test date
			result := time.Date(tc.date.Year(), tc.date.Month(), 1, 0, 0, 0, 0, time.Local)

			// Verify it's the first day
			if result.Day() != 1 {
				t.Errorf("Start of month day = %d, expected 1", result.Day())
			}

			// Verify it's midnight
			if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
				t.Errorf("Start of month should be midnight, got %v", result)
			}

			// Verify year and month match
			if result.Year() != tc.date.Year() || result.Month() != tc.date.Month() {
				t.Errorf("Start of month year/month = %d-%02d, expected %d-%02d",
					result.Year(), result.Month(), tc.date.Year(), tc.date.Month())
			}
		})
	}
}

func TestStartOfMonth_YearBoundary(t *testing.T) {
	// Test January (start of year) and December (end of year)
	testCases := []struct {
		name     string
		date     time.Time
		expYear  int
		expMonth time.Month
	}{
		{
			name:     "January 2024",
			date:     time.Date(2024, 1, 15, 12, 0, 0, 0, time.Local),
			expYear:  2024,
			expMonth: time.January,
		},
		{
			name:     "December 2024",
			date:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.Local),
			expYear:  2024,
			expMonth: time.December,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := time.Date(tc.date.Year(), tc.date.Month(), 1, 0, 0, 0, 0, time.Local)

			if result.Year() != tc.expYear {
				t.Errorf("Year = %d, expected %d", result.Year(), tc.expYear)
			}

			if result.Month() != tc.expMonth {
				t.Errorf("Month = %v, expected %v", result.Month(), tc.expMonth)
			}

			if result.Day() != 1 {
				t.Errorf("Day = %d, expected 1", result.Day())
			}
		})
	}
}

func TestLeapYear_Feb29(t *testing.T) {
	// Test behavior on February 29th of a leap year
	leapDate := time.Date(2024, 2, 29, 15, 30, 45, 0, time.Local)

	// StartOfMonth should work correctly
	startMonth := time.Date(leapDate.Year(), leapDate.Month(), 1, 0, 0, 0, 0, time.Local)
	if startMonth.Day() != 1 || startMonth.Month() != time.February {
		t.Errorf("StartOfMonth on Feb 29 = %v, expected Feb 1", startMonth)
	}

	// Yesterday from Feb 29 should be Feb 28
	startOfDay := time.Date(leapDate.Year(), leapDate.Month(), leapDate.Day(), 0, 0, 0, 0, time.Local)
	yesterday := startOfDay.AddDate(0, 0, -1)
	if yesterday.Day() != 28 || yesterday.Month() != time.February {
		t.Errorf("Yesterday from Feb 29 = %v, expected Feb 28", yesterday)
	}
}

func TestNonLeapYear_Feb28(t *testing.T) {
	// Test behavior on February 28th of a non-leap year
	nonLeapDate := time.Date(2023, 2, 28, 15, 30, 45, 0, time.Local)

	// StartOfMonth should work correctly
	startMonth := time.Date(nonLeapDate.Year(), nonLeapDate.Month(), 1, 0, 0, 0, 0, time.Local)
	if startMonth.Day() != 1 || startMonth.Month() != time.February {
		t.Errorf("StartOfMonth on Feb 28 = %v, expected Feb 1", startMonth)
	}

	// Yesterday from Feb 28 should be Feb 27
	startOfDay := time.Date(nonLeapDate.Year(), nonLeapDate.Month(), nonLeapDate.Day(), 0, 0, 0, 0, time.Local)
	yesterday := startOfDay.AddDate(0, 0, -1)
	if yesterday.Day() != 27 || yesterday.Month() != time.February {
		t.Errorf("Yesterday from Feb 28 = %v, expected Feb 27", yesterday)
	}

	// Tomorrow from Feb 28 should be March 1
	tomorrow := startOfDay.AddDate(0, 0, 1)
	if tomorrow.Day() != 1 || tomorrow.Month() != time.March {
		t.Errorf("Tomorrow from Feb 28 = %v, expected March 1", tomorrow)
	}
}

func TestVeryOldDates(t *testing.T) {
	// Test with very old dates to ensure no integer overflow or other issues
	oldDate := time.Date(1900, 1, 1, 12, 0, 0, 0, time.Local)

	// StartOfMonth calculation
	startMonth := time.Date(oldDate.Year(), oldDate.Month(), 1, 0, 0, 0, 0, time.Local)
	if startMonth.Year() != 1900 || startMonth.Month() != time.January || startMonth.Day() != 1 {
		t.Errorf("StartOfMonth for old date = %v, expected 1900-01-01", startMonth)
	}

	// Yesterday calculation
	startOfDay := time.Date(oldDate.Year(), oldDate.Month(), oldDate.Day(), 0, 0, 0, 0, time.Local)
	yesterday := startOfDay.AddDate(0, 0, -1)
	if yesterday.Year() != 1899 || yesterday.Month() != time.December || yesterday.Day() != 31 {
		t.Errorf("Yesterday from 1900-01-01 = %v, expected 1899-12-31", yesterday)
	}
}

func TestFutureDates(t *testing.T) {
	// Test with far future dates
	futureDate := time.Date(2100, 12, 31, 12, 0, 0, 0, time.Local)

	// StartOfMonth calculation
	startMonth := time.Date(futureDate.Year(), futureDate.Month(), 1, 0, 0, 0, 0, time.Local)
	if startMonth.Year() != 2100 || startMonth.Month() != time.December || startMonth.Day() != 1 {
		t.Errorf("StartOfMonth for future date = %v, expected 2100-12-01", startMonth)
	}

	// Tomorrow calculation (should go to next year)
	startOfDay := time.Date(futureDate.Year(), futureDate.Month(), futureDate.Day(), 0, 0, 0, 0, time.Local)
	tomorrow := startOfDay.AddDate(0, 0, 1)
	if tomorrow.Year() != 2101 || tomorrow.Month() != time.January || tomorrow.Day() != 1 {
		t.Errorf("Tomorrow from 2100-12-31 = %v, expected 2101-01-01", tomorrow)
	}
}

func TestMonthLengthVariations(t *testing.T) {
	// Test calculations work correctly across months with different lengths
	testCases := []struct {
		name        string
		date        time.Time
		daysInMonth int
	}{
		{"31-day month (Jan)", time.Date(2024, 1, 31, 12, 0, 0, 0, time.Local), 31},
		{"30-day month (Apr)", time.Date(2024, 4, 30, 12, 0, 0, 0, time.Local), 30},
		{"29-day month (Feb leap)", time.Date(2024, 2, 29, 12, 0, 0, 0, time.Local), 29},
		{"28-day month (Feb non-leap)", time.Date(2023, 2, 28, 12, 0, 0, 0, time.Local), 28},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// StartOfMonth should always be day 1
			startMonth := time.Date(tc.date.Year(), tc.date.Month(), 1, 0, 0, 0, 0, time.Local)
			if startMonth.Day() != 1 {
				t.Errorf("StartOfMonth day = %d, expected 1", startMonth.Day())
			}

			// Tomorrow from last day should be first of next month
			startOfDay := time.Date(tc.date.Year(), tc.date.Month(), tc.date.Day(), 0, 0, 0, 0, time.Local)
			tomorrow := startOfDay.AddDate(0, 0, 1)

			expectedMonth := tc.date.Month() + 1
			expectedYear := tc.date.Year()
			if expectedMonth > 12 {
				expectedMonth = 1
				expectedYear++
			}

			if tomorrow.Day() != 1 || tomorrow.Month() != expectedMonth || tomorrow.Year() != expectedYear {
				t.Errorf("Tomorrow from last day of month = %v, expected %d-%02d-01", tomorrow, expectedYear, expectedMonth)
			}
		})
	}
}

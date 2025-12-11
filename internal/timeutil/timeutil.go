// ABOUTME: Time utility functions for date range calculations
// ABOUTME: Provides helpers for smart views like today, yesterday, this week

package timeutil

import "time"

// StartOfToday returns midnight (00:00:00) of the current day in local time
func StartOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// StartOfYesterday returns midnight (00:00:00) of yesterday in local time
func StartOfYesterday() time.Time {
	return StartOfToday().AddDate(0, 0, -1)
}

// EndOfYesterday returns the last moment of yesterday (start of today) in local time
func EndOfYesterday() time.Time {
	return StartOfToday()
}

// StartOfWeek returns midnight of the most recent Sunday in local time
// Note: Week starts on Sunday
func StartOfWeek() time.Time {
	today := StartOfToday()
	weekday := int(today.Weekday())
	return today.AddDate(0, 0, -weekday)
}

// StartOfMonth returns midnight of the first day of the current month in local time
func StartOfMonth() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// ParsePeriod converts a period string to a time.Time representing the cutoff
// Supported values: "today", "yesterday", "week", "month"
// Returns the start of that period (articles before this time would be marked)
func ParsePeriod(period string) (time.Time, bool) {
	switch period {
	case "today":
		return StartOfToday(), true
	case "yesterday":
		return StartOfYesterday(), true
	case "week":
		return StartOfWeek(), true
	case "month":
		return StartOfMonth(), true
	default:
		return time.Time{}, false
	}
}

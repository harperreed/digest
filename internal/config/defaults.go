// ABOUTME: Centralized configuration defaults for digest
// ABOUTME: Contains magic numbers and hardcoded values for display and storage

package config

import "time"

// HTTP settings
const (
	DefaultHTTPTimeout = 30 * time.Second
)

// Display settings
const (
	DefaultListLimit = 20
	DisplayIDLength  = 8
	SeparatorWidth   = 60
	DateFormatShort  = "02 Jan 06 15:04 MST"
	DateFormatLong   = "Mon, 02 Jan 2006 15:04 MST"
)

// Storage settings
const (
	MinPrefixLength       = 6
	ExpectedPrefixMatches = 2
	DefaultDirPerms       = 0755
)

// OPML settings
const (
	OPMLVersion = "2.0"
)

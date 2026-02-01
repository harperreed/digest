// ABOUTME: Tests for version command
// ABOUTME: Verifies version information display

package main

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Verify default values are set (these are set at build time but have defaults)
	if Version == "" {
		t.Error("expected Version to be set")
	}
	if Commit == "" {
		t.Error("expected Commit to be set")
	}
	if BuildDate == "" {
		t.Error("expected BuildDate to be set")
	}

	// Test default values
	if Version != "dev" {
		t.Logf("Version is %q (non-default, likely built with ldflags)", Version)
	}
}

func TestVersionCommandShort(t *testing.T) {
	if versionCmd.Short == "" {
		t.Error("expected version command to have a short description")
	}
}

func TestVersionCommandUse(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("expected Use to be 'version', got %q", versionCmd.Use)
	}
}

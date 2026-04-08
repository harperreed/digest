// ABOUTME: Tests for profile management commands
// ABOUTME: Verifies profile list and delete command structure and flags

package main

import (
	"testing"
)

func TestProfileCommand(t *testing.T) {
	if profileCmd.Use != "profile" {
		t.Errorf("expected Use to be 'profile', got %q", profileCmd.Use)
	}
}

func TestProfileListCommand(t *testing.T) {
	if profileListCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", profileListCmd.Use)
	}
	if len(profileListCmd.Aliases) == 0 {
		t.Error("expected profile list command to have aliases")
	}
}

func TestProfileDeleteCommand(t *testing.T) {
	if profileDeleteCmd.Use != "delete <name>" {
		t.Errorf("expected Use to be 'delete <name>', got %q", profileDeleteCmd.Use)
	}
}

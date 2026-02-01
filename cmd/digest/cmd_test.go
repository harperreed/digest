// ABOUTME: Tests for CLI commands
// ABOUTME: Tests command structure, flags, and subcommands

package main

import (
	"testing"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "digest" {
		t.Errorf("expected Use to be 'digest', got %q", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("expected root command to have a short description")
	}
}

func TestFeedCommand(t *testing.T) {
	if feedCmd.Use != "feed" {
		t.Errorf("expected Use to be 'feed', got %q", feedCmd.Use)
	}
	if len(feedCmd.Aliases) == 0 {
		t.Error("expected feed command to have aliases")
	}
}

func TestFeedAddCommand(t *testing.T) {
	if feedAddCmd.Use != "add <url>" {
		t.Errorf("expected Use to be 'add <url>', got %q", feedAddCmd.Use)
	}

	// Check flags exist
	if feedAddCmd.Flags().Lookup("folder") == nil {
		t.Error("expected --folder flag to exist")
	}
	if feedAddCmd.Flags().Lookup("title") == nil {
		t.Error("expected --title flag to exist")
	}
	if feedAddCmd.Flags().Lookup("no-discover") == nil {
		t.Error("expected --no-discover flag to exist")
	}
}

func TestFeedListCommand(t *testing.T) {
	if feedListCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", feedListCmd.Use)
	}
	if len(feedListCmd.Aliases) == 0 {
		t.Error("expected feed list command to have aliases")
	}
}

func TestFeedRemoveCommand(t *testing.T) {
	if feedRemoveCmd.Use != "remove <url>" {
		t.Errorf("expected Use to be 'remove <url>', got %q", feedRemoveCmd.Use)
	}
}

func TestFeedMoveCommand(t *testing.T) {
	if feedMoveCmd.Use != "move <url> <category>" {
		t.Errorf("expected Use to be 'move <url> <category>', got %q", feedMoveCmd.Use)
	}
}

func TestListCommand(t *testing.T) {
	if listCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", listCmd.Use)
	}
	if len(listCmd.Aliases) == 0 {
		t.Error("expected list command to have aliases")
	}

	// Check flags exist
	if listCmd.Flags().Lookup("all") == nil {
		t.Error("expected --all flag to exist")
	}
	if listCmd.Flags().Lookup("feed") == nil {
		t.Error("expected --feed flag to exist")
	}
	if listCmd.Flags().Lookup("category") == nil {
		t.Error("expected --category flag to exist")
	}
	if listCmd.Flags().Lookup("limit") == nil {
		t.Error("expected --limit flag to exist")
	}
	if listCmd.Flags().Lookup("offset") == nil {
		t.Error("expected --offset flag to exist")
	}
	if listCmd.Flags().Lookup("today") == nil {
		t.Error("expected --today flag to exist")
	}
	if listCmd.Flags().Lookup("yesterday") == nil {
		t.Error("expected --yesterday flag to exist")
	}
	if listCmd.Flags().Lookup("week") == nil {
		t.Error("expected --week flag to exist")
	}
}

func TestReadCommand(t *testing.T) {
	if readCmd.Use != "read <entry-id>" {
		t.Errorf("expected Use to be 'read <entry-id>', got %q", readCmd.Use)
	}

	// Check flags exist
	if readCmd.Flags().Lookup("no-mark") == nil {
		t.Error("expected --no-mark flag to exist")
	}
}

func TestMarkReadCommand(t *testing.T) {
	if markReadCmd.Use != "mark-read [entry-id]" {
		t.Errorf("expected Use to be 'mark-read [entry-id]', got %q", markReadCmd.Use)
	}

	// Check flags exist
	if markReadCmd.Flags().Lookup("before") == nil {
		t.Error("expected --before flag to exist")
	}
}

func TestMarkUnreadCommand(t *testing.T) {
	if markUnreadCmd.Use != "mark-unread <entry-id>" {
		t.Errorf("expected Use to be 'mark-unread <entry-id>', got %q", markUnreadCmd.Use)
	}
}

func TestFetchCommand(t *testing.T) {
	if fetchCmd.Use != "fetch [url]" {
		t.Errorf("expected Use to be 'fetch [url]', got %q", fetchCmd.Use)
	}

	// Check flags exist
	if fetchCmd.Flags().Lookup("force") == nil {
		t.Error("expected --force flag to exist")
	}
}

func TestFolderCommand(t *testing.T) {
	if folderCmd.Use != "folder" {
		t.Errorf("expected Use to be 'folder', got %q", folderCmd.Use)
	}
}

func TestFolderAddCommand(t *testing.T) {
	if folderAddCmd.Use != "add <name>" {
		t.Errorf("expected Use to be 'add <name>', got %q", folderAddCmd.Use)
	}
}

func TestFolderListCommand(t *testing.T) {
	if folderListCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", folderListCmd.Use)
	}
	if len(folderListCmd.Aliases) == 0 {
		t.Error("expected folder list command to have aliases")
	}
}

func TestInstallSkillCommand(t *testing.T) {
	if installSkillCmd.Use != "install-skill" {
		t.Errorf("expected Use to be 'install-skill', got %q", installSkillCmd.Use)
	}

	// Check flags exist
	if installSkillCmd.Flags().Lookup("yes") == nil {
		t.Error("expected --yes flag to exist")
	}
}

func TestRootPersistentFlags(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("opml") == nil {
		t.Error("expected --opml flag to exist")
	}
}

func TestCommandRegistration(t *testing.T) {
	// Check that subcommands are registered
	commands := rootCmd.Commands()

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{
		"feed",
		"list",
		"read",
		"mark-read",
		"mark-unread",
		"fetch",
		"folder",
		"version",
		"install-skill",
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected command %q to be registered", expected)
		}
	}
}

func TestFeedSubcommands(t *testing.T) {
	commands := feedCmd.Commands()

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{
		"add",
		"list",
		"remove",
		"move",
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected feed subcommand %q to be registered", expected)
		}
	}
}

func TestFolderSubcommands(t *testing.T) {
	commands := folderCmd.Commands()

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{
		"add",
		"list",
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected folder subcommand %q to be registered", expected)
		}
	}
}

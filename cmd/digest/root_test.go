// ABOUTME: Tests for root command and utility functions
// ABOUTME: Verifies default paths and OPML handling

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/opml"
)

func TestGetDefaultOPMLPath(t *testing.T) {
	path := GetDefaultOPMLPath()
	if path == "" {
		t.Error("expected non-empty default OPML path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
	if !containsString(path, "feeds.opml") {
		t.Errorf("expected path to contain 'feeds.opml', got %q", path)
	}
}

func TestGetDefaultOPMLPath_WithXDGDataHome(t *testing.T) {
	// Save original value
	original := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", original)

	// Set custom XDG_DATA_HOME
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	path := GetDefaultOPMLPath()
	if !containsString(path, tmpDir) {
		t.Errorf("expected path to use XDG_DATA_HOME %q, got %q", tmpDir, path)
	}
}

func TestSaveOPML(t *testing.T) {
	// Create a temp file for OPML
	tmpDir := t.TempDir()
	tmpOPMLPath := filepath.Join(tmpDir, "test.opml")

	// Setup global variables
	oldOpmlPath := opmlPath
	oldOpmlDoc := opmlDoc
	defer func() {
		opmlPath = oldOpmlPath
		opmlDoc = oldOpmlDoc
	}()

	opmlPath = tmpOPMLPath
	opmlDoc = opml.NewDocument("test")

	// Test saveOPML
	err := saveOPML()
	if err != nil {
		t.Fatalf("saveOPML: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpOPMLPath); err != nil {
		t.Errorf("expected OPML file to exist: %v", err)
	}
}

func TestSaveOPML_NilDoc(t *testing.T) {
	// Setup global variables
	oldOpmlDoc := opmlDoc
	defer func() {
		opmlDoc = oldOpmlDoc
	}()

	opmlDoc = nil

	err := saveOPML()
	if err == nil {
		t.Error("expected error when opmlDoc is nil")
	}
}

func TestGetDataDir(t *testing.T) {
	dir := getDataDir()
	if dir == "" {
		t.Error("expected non-empty data dir")
	}
}

func TestGetDataDir_WithXDGDataHome(t *testing.T) {
	// Save original value
	original := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", original)

	// Set custom XDG_DATA_HOME
	customDir := "/custom/data/dir"
	os.Setenv("XDG_DATA_HOME", customDir)

	dir := getDataDir()
	if dir != customDir {
		t.Errorf("expected %q, got %q", customDir, dir)
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

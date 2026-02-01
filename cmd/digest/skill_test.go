// ABOUTME: Tests for the install-skill command
// ABOUTME: Covers directory creation, file writing, and overwrite scenarios

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "digest")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Verify directory doesn't exist yet
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatal("skill directory should not exist before test")
	}

	// Install skill using helper
	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(skillDir)
	if err != nil {
		t.Fatalf("skill directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected skill directory to be a directory")
	}
}

func TestSkillWritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "digest", "SKILL.md")

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("skill file was not created: %v", err)
	}
	if info.IsDir() {
		t.Error("expected skill file to be a file, not a directory")
	}
	if info.Size() == 0 {
		t.Error("skill file should not be empty")
	}
}

func TestSkillFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "digest", "SKILL.md")

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}

	contentStr := string(content)

	// Verify required content sections
	expectedSections := []string{
		"name: digest",
		"# digest",
		"## When to use digest",
		"mcp__digest__",
		"CLI commands",
	}

	for _, section := range expectedSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("skill file missing expected section: %q", section)
		}
	}

	// Verify it starts with YAML front matter
	if !strings.HasPrefix(contentStr, "---") {
		t.Error("skill file should start with YAML front matter (---)")
	}
}

func TestSkillOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "digest")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Create directory and existing file
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	originalContent := "# Old skill file content\nThis should be overwritten."
	if err := os.WriteFile(skillPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	// Install skill (should overwrite)
	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	// Verify content was overwritten
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}

	if string(content) == originalContent {
		t.Error("skill file should have been overwritten")
	}

	if !strings.Contains(string(content), "name: digest") {
		t.Error("skill file should contain new content")
	}
}

func TestSkillFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "digest", "SKILL.md")

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("failed to stat skill file: %v", err)
	}

	// Check file is readable (at minimum)
	mode := info.Mode()
	if mode&0400 == 0 {
		t.Error("skill file should be readable by owner")
	}
}

func TestSkillDirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "digest")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	info, err := os.Stat(skillDir)
	if err != nil {
		t.Fatalf("failed to stat skill directory: %v", err)
	}

	// Check directory has expected permissions (0755)
	mode := info.Mode()
	if mode&0700 != 0700 {
		t.Error("skill directory should be readable/writable/executable by owner")
	}
}

func TestSkillCreatesNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a deep nested path that doesn't exist
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "digest", "SKILL.md")

	// Verify no intermediate directories exist
	claudeDir := filepath.Join(tmpDir, ".claude")
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Fatal(".claude directory should not exist before test")
	}

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	// Verify all directories were created
	dirs := []string{
		filepath.Join(tmpDir, ".claude"),
		filepath.Join(tmpDir, ".claude", "skills"),
		filepath.Join(tmpDir, ".claude", "skills", "digest"),
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %q was not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q should be a directory", dir)
		}
	}
}

func TestSkillContentMatchesEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, ".claude", "skills", "digest", "SKILL.md")

	err := installSkillToPath(skillPath)
	if err != nil {
		t.Fatalf("installSkillToPath failed: %v", err)
	}

	// Read installed content
	installedContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read installed skill file: %v", err)
	}

	// Read embedded content directly
	embeddedContent, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		t.Fatalf("failed to read embedded skill file: %v", err)
	}

	if string(installedContent) != string(embeddedContent) {
		t.Error("installed content does not match embedded content")
	}
}

// installSkillToPath is a testable version of installSkill that accepts a custom path
func installSkillToPath(skillPath string) error {
	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return err
	}

	// Create directory
	skillDir := filepath.Dir(skillPath)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	// Write skill file
	return os.WriteFile(skillPath, content, 0644)
}

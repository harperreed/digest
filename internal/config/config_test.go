// ABOUTME: Tests for config functionality
// ABOUTME: Verifies config load, save, path resolution, defaults, and backend factory

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("GetConfigPath returned non-absolute path: %s", path)
	}
}

func TestGetConfigPathWithXDGConfigHome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := GetConfigPath()
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("GetConfigPath should use XDG_CONFIG_HOME, got %s", path)
	}
	if !strings.HasSuffix(path, filepath.Join("digest", "config.json")) {
		t.Errorf("GetConfigPath should end with digest/config.json, got %s", path)
	}
}

func TestGetConfigPathWithoutXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath returned empty string")
	}
	// Should fall back to ~/.config
	if !strings.Contains(path, ".config") {
		t.Errorf("GetConfigPath should use .config fallback, got %s", path)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed on non-existent config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}
	if cfg.Backend != "markdown" {
		t.Errorf("expected default backend 'markdown' for new user, got %q", cfg.Backend)
	}

	// Verify the config file was auto-created
	configPath := GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to be auto-created on first run")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "digest")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte("invalid json {{{"), 0600); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load should fail on invalid JSON")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded == nil {
		t.Error("loaded config is nil")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	configDir := filepath.Join(tmpDir, "digest")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Errorf("Config directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Config path is not a directory")
	}
}

func TestDefaultBackend(t *testing.T) {
	cfg := &Config{}
	backend := cfg.GetBackend()
	if backend != "sqlite" {
		t.Errorf("expected default backend 'sqlite', got %q", backend)
	}
}

func TestExplicitBackend(t *testing.T) {
	cfg := &Config{Backend: "markdown"}
	backend := cfg.GetBackend()
	if backend != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", backend)
	}
}

func TestDefaultDataDir(t *testing.T) {
	cfg := &Config{}
	dataDir := cfg.GetDataDir()
	if dataDir == "" {
		t.Error("GetDataDir returned empty string")
	}
	if !filepath.IsAbs(dataDir) {
		t.Errorf("GetDataDir returned non-absolute path: %s", dataDir)
	}
	// Should end with "digest" directory
	if filepath.Base(dataDir) != "digest" {
		t.Errorf("GetDataDir should end with 'digest', got %s", dataDir)
	}
}

func TestExplicitDataDir(t *testing.T) {
	cfg := &Config{DataDir: "/custom/data/path"}
	dataDir := cfg.GetDataDir()
	if dataDir != "/custom/data/path" {
		t.Errorf("expected '/custom/data/path', got %q", dataDir)
	}
}

func TestDataDirTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home dir: %v", err)
	}

	cfg := &Config{DataDir: "~/my-digest-data"}
	dataDir := cfg.GetDataDir()
	expected := filepath.Join(home, "my-digest-data")
	if dataDir != expected {
		t.Errorf("expected %q, got %q", expected, dataDir)
	}
}

func TestDataDirTildeOnlyExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home dir: %v", err)
	}

	cfg := &Config{DataDir: "~"}
	dataDir := cfg.GetDataDir()
	if dataDir != home {
		t.Errorf("expected %q, got %q", home, dataDir)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home dir: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExpandPath(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSaveAndLoadWithBackendFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{
		Backend: "markdown",
		DataDir: "/custom/data",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Backend != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", loaded.Backend)
	}
	if loaded.DataDir != "/custom/data" {
		t.Errorf("expected data_dir '/custom/data', got %q", loaded.DataDir)
	}
}

func TestSaveAndLoadPreservesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{
		Backend: "sqlite",
		DataDir: "~/my-data",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw JSON: %v", err)
	}

	if raw["backend"] != "sqlite" {
		t.Errorf("expected JSON key 'backend' with value 'sqlite', got %v", raw["backend"])
	}
	if raw["data_dir"] != "~/my-data" {
		t.Errorf("expected JSON key 'data_dir' with value '~/my-data', got %v", raw["data_dir"])
	}
}

func TestOpenStorageSqliteBackend(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Backend: "sqlite",
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage failed for sqlite: %v", err)
	}
	defer store.Close()
}

func TestOpenStorageDefaultBackend(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage failed for default backend: %v", err)
	}
	defer store.Close()
}

func TestOpenStorageMarkdownBackend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: "markdown",
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage failed for markdown backend: %v", err)
	}
	defer store.Close()
}

func TestOpenStorageUnknownBackend(t *testing.T) {
	cfg := &Config{
		Backend: "redis",
		DataDir: "/tmp/digest-test",
	}

	_, err := cfg.OpenStorage()
	if err == nil {
		t.Fatal("expected error for unknown backend, got nil")
	}
	if !strings.Contains(err.Error(), "unknown backend") {
		t.Errorf("expected 'unknown backend' error, got: %v", err)
	}
}

func TestOpenStorageSqliteCreatesDBInDataDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Backend: "sqlite",
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage failed: %v", err)
	}
	defer store.Close()

	dbPath := filepath.Join(tmpDir, "digest.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file at %s", dbPath)
	}
}

func TestSaveToUnwritableDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/nonexistent/path/that/does/not/exist/12345")

	cfg := &Config{}
	err := cfg.Save()

	if err == nil {
		t.Error("Expected error when saving to unwritable directory")
	}
}

func TestLoadMissingConfig_ExistingSQLiteUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Create fake digest.db at the expected data directory
	dataDir := filepath.Join(tmpDir, "digest")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	dbPath := filepath.Join(dataDir, "digest.db")
	if err := os.WriteFile(dbPath, []byte("fake db"), 0600); err != nil {
		t.Fatalf("failed to create fake db: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("expected backend 'sqlite' for existing SQLite user, got %q", cfg.Backend)
	}
}

func TestLoadAutoCreatesValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Read the auto-created config file and validate its contents
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read auto-created config: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("auto-created config is not valid JSON: %v", err)
	}

	if raw["backend"] != "markdown" {
		t.Errorf("expected auto-created config backend 'markdown', got %v", raw["backend"])
	}
}

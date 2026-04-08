// ABOUTME: Tests for config functionality
// ABOUTME: Verifies config load, save, path resolution, defaults, and backend factory

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harper/digest/internal/models"
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

func TestProfileDataDir(t *testing.T) {
	cfg := &Config{DataDir: "/data/digest"}
	got, err := cfg.ProfileDataDir("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/data/digest/work"
	if got != expected {
		t.Errorf("ProfileDataDir(\"work\") = %q, want %q", got, expected)
	}
}

func TestProfileDataDirDefault(t *testing.T) {
	cfg := &Config{DataDir: "/data/digest"}
	got, err := cfg.ProfileDataDir("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/data/digest/default"
	if got != expected {
		t.Errorf("ProfileDataDir(\"default\") = %q, want %q", got, expected)
	}
}

func TestProfileDataDirTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home dir: %v", err)
	}
	cfg := &Config{DataDir: "~/digest-data"}
	got, err := cfg.ProfileDataDir("security")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(home, "digest-data", "security")
	if got != expected {
		t.Errorf("ProfileDataDir(\"security\") = %q, want %q", got, expected)
	}
}

func TestProfileDataDirDefaultDataDir(t *testing.T) {
	cfg := &Config{}
	got, err := cfg.ProfileDataDir("work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "work" {
		t.Errorf("expected path to end with 'work', got %q", got)
	}
}

func TestValidateProfileName_Valid(t *testing.T) {
	valid := []string{"default", "work", "my-feeds", "security_2024", "a", "A-B.c"}
	for _, name := range valid {
		if err := ValidateProfileName(name); err != nil {
			t.Errorf("ValidateProfileName(%q) should be valid, got: %v", name, err)
		}
	}
}

func TestValidateProfileName_Invalid(t *testing.T) {
	invalid := []struct {
		name   string
		reason string
	}{
		{"", "empty string"},
		{"..", "path traversal"},
		{".", "dot"},
		{"../../../etc", "path traversal with slashes"},
		{"foo/bar", "contains slash"},
		{"foo\\bar", "contains backslash"},
		{" work", "starts with space"},
		{"work ", "ends with space"},
		{"-work", "starts with hyphen"},
		{".hidden", "starts with dot"},
		{"a b", "contains space"},
		{"CON", "Windows reserved name"},
		{"nul", "Windows reserved name lowercase"},
		{"com1", "Windows reserved name lowercase"},
		{strings.Repeat("a", 65), "too long"},
	}
	for _, tc := range invalid {
		if err := ValidateProfileName(tc.name); err == nil {
			t.Errorf("ValidateProfileName(%q) should be invalid (%s)", tc.name, tc.reason)
		}
	}
}

func TestProfileDataDir_RejectsTraversal(t *testing.T) {
	cfg := &Config{DataDir: "/data/digest"}
	_, err := cfg.ProfileDataDir("../../../etc")
	if err == nil {
		t.Error("ProfileDataDir should reject path traversal")
	}
}

func TestOpenProfileStorageSqlite(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Backend: "sqlite", DataDir: tmpDir}

	store, err := cfg.OpenProfileStorage("work")
	if err != nil {
		t.Fatalf("OpenProfileStorage failed: %v", err)
	}
	defer store.Close()

	// Verify DB was created in profile subdirectory
	dbPath := filepath.Join(tmpDir, "work", "digest.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database at %s", dbPath)
	}
}

func TestOpenProfileStorageMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Backend: "markdown", DataDir: tmpDir}

	store, err := cfg.OpenProfileStorage("personal")
	if err != nil {
		t.Fatalf("OpenProfileStorage failed: %v", err)
	}
	defer store.Close()

	// Verify profile directory was created
	profileDir := filepath.Join(tmpDir, "personal")
	info, err := os.Stat(profileDir)
	if err != nil {
		t.Fatalf("expected profile directory at %s: %v", profileDir, err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory at %s", profileDir)
	}
}

func TestOpenProfileStorageAutoCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Backend: "sqlite", DataDir: tmpDir}

	profileDir := filepath.Join(tmpDir, "newprofile")
	if _, err := os.Stat(profileDir); !os.IsNotExist(err) {
		t.Fatal("profile dir should not exist yet")
	}

	store, err := cfg.OpenProfileStorage("newprofile")
	if err != nil {
		t.Fatalf("OpenProfileStorage failed: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		t.Error("expected profile directory to be auto-created")
	}
}

func TestMigrateToProfileLayout_MovesDB(t *testing.T) {
	tmpDir := t.TempDir()

	// Create flat layout files
	dbPath := filepath.Join(tmpDir, "digest.db")
	if err := os.WriteFile(dbPath, []byte("fake db"), 0600); err != nil {
		t.Fatal(err)
	}
	opmlPath := filepath.Join(tmpDir, "feeds.opml")
	if err := os.WriteFile(opmlPath, []byte("<opml/>"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	// Old files should be gone
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Error("expected old digest.db to be removed")
	}
	if _, err := os.Stat(opmlPath); !os.IsNotExist(err) {
		t.Error("expected old feeds.opml to be removed")
	}

	// New files should exist in default/
	newDB := filepath.Join(tmpDir, "default", "digest.db")
	if _, err := os.Stat(newDB); os.IsNotExist(err) {
		t.Error("expected digest.db in default/")
	}
	newOPML := filepath.Join(tmpDir, "default", "feeds.opml")
	if _, err := os.Stat(newOPML); os.IsNotExist(err) {
		t.Error("expected feeds.opml in default/")
	}
}

func TestMigrateToProfileLayout_NoOpWhenAlreadyMigrated(t *testing.T) {
	tmpDir := t.TempDir()

	// Create profile layout (default/ already exists)
	defaultDir := filepath.Join(tmpDir, "default")
	os.MkdirAll(defaultDir, 0750)
	os.WriteFile(filepath.Join(defaultDir, "digest.db"), []byte("db"), 0600)

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	// default/ should still have the db
	if _, err := os.Stat(filepath.Join(defaultDir, "digest.db")); os.IsNotExist(err) {
		t.Error("expected digest.db to still be in default/")
	}
}

func TestMigrateToProfileLayout_NoOpWhenEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	// default/ should not be created if there was nothing to migrate
	defaultDir := filepath.Join(tmpDir, "default")
	if _, err := os.Stat(defaultDir); !os.IsNotExist(err) {
		t.Error("expected no default/ directory when nothing to migrate")
	}
}

func TestMigrateToProfileLayout_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create flat layout
	os.WriteFile(filepath.Join(tmpDir, "digest.db"), []byte("fake db"), 0600)

	cfg := &Config{DataDir: tmpDir}

	// Run migration twice
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}

	// Data should still be in default/
	newDB := filepath.Join(tmpDir, "default", "digest.db")
	data, err := os.ReadFile(newDB)
	if err != nil {
		t.Fatalf("expected digest.db in default/: %v", err)
	}
	if string(data) != "fake db" {
		t.Errorf("unexpected db contents: %q", data)
	}
}

func TestProfileIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{Backend: "sqlite", DataDir: tmpDir}

	// Open two different profile stores
	workStore, err := cfg.OpenProfileStorage("work")
	if err != nil {
		t.Fatalf("failed to open work profile: %v", err)
	}
	defer workStore.Close()

	personalStore, err := cfg.OpenProfileStorage("personal")
	if err != nil {
		t.Fatalf("failed to open personal profile: %v", err)
	}
	defer personalStore.Close()

	// Verify separate directories
	workDir := filepath.Join(tmpDir, "work")
	personalDir := filepath.Join(tmpDir, "personal")

	if _, err := os.Stat(filepath.Join(workDir, "digest.db")); os.IsNotExist(err) {
		t.Error("expected work/digest.db")
	}
	if _, err := os.Stat(filepath.Join(personalDir, "digest.db")); os.IsNotExist(err) {
		t.Error("expected personal/digest.db")
	}

	// Create a feed in work profile
	feed := &models.Feed{
		ID:        "test-feed-1",
		URL:       "https://example.com/feed.xml",
		CreatedAt: time.Now(),
	}
	if err := workStore.CreateFeed(feed); err != nil {
		t.Fatalf("failed to create feed in work: %v", err)
	}

	// Verify feed exists in work but not in personal
	workFeeds, err := workStore.ListFeeds()
	if err != nil {
		t.Fatalf("failed to list work feeds: %v", err)
	}
	if len(workFeeds) != 1 {
		t.Errorf("expected 1 work feed, got %d", len(workFeeds))
	}

	personalFeeds, err := personalStore.ListFeeds()
	if err != nil {
		t.Fatalf("failed to list personal feeds: %v", err)
	}
	if len(personalFeeds) != 0 {
		t.Errorf("expected 0 personal feeds, got %d", len(personalFeeds))
	}
}

func TestMigrateToProfileLayout_MovesWALFiles(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "digest.db"), []byte("db"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "digest.db-wal"), []byte("wal"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "digest.db-shm"), []byte("shm"), 0600)

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	for _, name := range []string{"digest.db", "digest.db-wal", "digest.db-shm"} {
		path := filepath.Join(tmpDir, "default", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %s in default/", name)
		}
	}

	for _, name := range []string{"digest.db", "digest.db-wal", "digest.db-shm"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed from root", name)
		}
	}
}

func TestMigrateToProfileLayout_MovesMarkdownFiles(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "_feeds.yaml"), []byte("feeds"), 0600)
	feedDir := filepath.Join(tmpDir, "my-feed")
	os.MkdirAll(feedDir, 0750)
	os.WriteFile(filepath.Join(feedDir, "entry.md"), []byte("content"), 0600)

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "default", "_feeds.yaml")); os.IsNotExist(err) {
		t.Error("expected _feeds.yaml in default/")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "default", "my-feed", "entry.md")); os.IsNotExist(err) {
		t.Error("expected my-feed/entry.md in default/")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "_feeds.yaml")); !os.IsNotExist(err) {
		t.Error("expected _feeds.yaml to be removed from root")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "my-feed")); !os.IsNotExist(err) {
		t.Error("expected my-feed to be removed from root")
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

func TestDefaultFirstRunConfig_SQLiteInProfileLayout(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	dataDir := filepath.Join(tmpDir, "digest", "default")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "digest.db"), []byte("db"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("expected backend 'sqlite' for existing SQLite user in profile layout, got %q", cfg.Backend)
	}
}

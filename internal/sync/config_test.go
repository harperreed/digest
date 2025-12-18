// ABOUTME: Tests for sync configuration management
// ABOUTME: Verifies config load/save, defaults, and env overrides

package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv creates a temp directory and sets HOME and XDG_CONFIG_HOME to it.
// Returns a cleanup function that should be deferred.
func setupTestEnv(t *testing.T) func() {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	return func() {
		os.Setenv("HOME", origHome)
		if origXDG == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	defer setupTestEnv(t)()

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Empty(t, cfg.Server)
	assert.Empty(t, cfg.Token)
	assert.NotEmpty(t, cfg.VaultDB) // Should have default path
}

func TestConfigSaveLoad(t *testing.T) {
	defer setupTestEnv(t)()

	cfg := &Config{
		Server:     "https://api.example.com",
		UserID:     "user123",
		Token:      "token456",
		DeviceID:   "device789",
		DerivedKey: "abcdef",
	}

	err := SaveConfig(cfg)
	require.NoError(t, err)

	loaded, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, cfg.Server, loaded.Server)
	assert.Equal(t, cfg.UserID, loaded.UserID)
	assert.Equal(t, cfg.Token, loaded.Token)
	assert.Equal(t, cfg.DeviceID, loaded.DeviceID)
	assert.Equal(t, cfg.DerivedKey, loaded.DerivedKey)
}

func TestIsConfigured(t *testing.T) {
	cfg := &Config{}
	assert.False(t, cfg.IsConfigured())

	cfg.Server = "https://api.example.com"
	assert.False(t, cfg.IsConfigured())

	cfg.Token = "token"
	cfg.UserID = "user"
	cfg.DerivedKey = "key"
	assert.True(t, cfg.IsConfigured())
}

func TestConfigEnvOverrides(t *testing.T) {
	defer setupTestEnv(t)()

	// Save a base config
	cfg := &Config{
		Server:     "https://file.example.com",
		UserID:     "file-user",
		Token:      "file-token",
		DeviceID:   "file-device",
		DerivedKey: "file-key",
		VaultDB:    "/file/vault.db",
	}
	err := SaveConfig(cfg)
	require.NoError(t, err)

	// Set env overrides
	os.Setenv("DIGEST_SERVER", "https://env.example.com")
	os.Setenv("DIGEST_USER_ID", "env-user")
	os.Setenv("DIGEST_TOKEN", "env-token")
	os.Setenv("DIGEST_DEVICE_ID", "env-device")
	os.Setenv("DIGEST_VAULT_DB", "/env/vault.db")
	defer func() {
		os.Unsetenv("DIGEST_SERVER")
		os.Unsetenv("DIGEST_USER_ID")
		os.Unsetenv("DIGEST_TOKEN")
		os.Unsetenv("DIGEST_DEVICE_ID")
		os.Unsetenv("DIGEST_VAULT_DB")
	}()

	// Load config and verify env overrides
	loaded, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "https://env.example.com", loaded.Server)
	assert.Equal(t, "env-user", loaded.UserID)
	assert.Equal(t, "env-token", loaded.Token)
	assert.Equal(t, "env-device", loaded.DeviceID)
	assert.Equal(t, "/env/vault.db", loaded.VaultDB)
	// DerivedKey should come from file (not overridable)
	assert.Equal(t, "file-key", loaded.DerivedKey)
}

func TestConfigCorruptedFile(t *testing.T) {
	defer setupTestEnv(t)()

	// Create corrupted config file
	err := EnsureConfigDir()
	require.NoError(t, err)

	err = os.WriteFile(ConfigPath(), []byte("this is not valid json{{{"), 0o600)
	require.NoError(t, err)

	// Should return error and backup the file
	_, err = LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file corrupted")

	// Verify backup was created
	files, err := os.ReadDir(ConfigDir())
	require.NoError(t, err)

	var foundBackup bool
	for _, f := range files {
		if len(f.Name()) > 13 && f.Name()[:13] == "sync.json.cor" {
			foundBackup = true
			break
		}
	}
	assert.True(t, foundBackup, "Expected backup file to be created")
}

func TestConfigPathIsDirectory(t *testing.T) {
	defer setupTestEnv(t)()

	// Create config path as a directory instead of file
	err := os.MkdirAll(ConfigPath(), 0o750)
	require.NoError(t, err)

	// Should return error
	_, err = LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")
}

func TestInitConfig(t *testing.T) {
	defer setupTestEnv(t)()

	cfg, err := InitConfig()
	require.NoError(t, err)

	// Verify device ID was generated
	assert.NotEmpty(t, cfg.DeviceID)
	assert.Len(t, cfg.DeviceID, 26) // ULID length

	// Verify defaults
	assert.NotEmpty(t, cfg.VaultDB)

	// Verify it was saved
	loaded, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.DeviceID, loaded.DeviceID)
}

func TestConfigExists(t *testing.T) {
	defer setupTestEnv(t)()

	// Initially should not exist
	assert.False(t, ConfigExists())

	// Create config
	cfg := &Config{DeviceID: "test-device"}
	err := SaveConfig(cfg)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, ConfigExists())
}

func TestEnsureConfigDirBackupFile(t *testing.T) {
	defer setupTestEnv(t)()

	// Create config dir as a file
	err := os.MkdirAll(filepath.Dir(ConfigDir()), 0o750)
	require.NoError(t, err)

	err = os.WriteFile(ConfigDir(), []byte("test"), 0o600)
	require.NoError(t, err)

	// EnsureConfigDir should backup the file and create directory
	err = EnsureConfigDir()
	require.NoError(t, err)

	// Verify it's now a directory
	info, err := os.Stat(ConfigDir())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify backup was created
	files, err := os.ReadDir(filepath.Dir(ConfigDir()))
	require.NoError(t, err)

	var foundBackup bool
	for _, f := range files {
		if len(f.Name()) > 13 && f.Name()[:13] == "digest.backup" {
			foundBackup = true
			break
		}
	}
	assert.True(t, foundBackup, "Expected backup file to be created")
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "absolute path",
			input:    "/absolute/path/to/file",
			expected: "/absolute/path/to/file",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test tilde expansion
	t.Run("tilde expansion", func(t *testing.T) {
		result := expandPath("~/test/path")
		assert.NotContains(t, result, "~")
		assert.Contains(t, result, "test/path")
	})
}

func TestLoadConfigNoFile(t *testing.T) {
	defer setupTestEnv(t)()

	// Should return default config if file doesn't exist
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.VaultDB)
}

func TestConfigPathRespectsXDGConfigHome(t *testing.T) {
	tmpDir := t.TempDir()
	xdgDir := filepath.Join(tmpDir, "custom-config")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", xdgDir)
	defer func() {
		if origXDG == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
	}()

	expectedPath := filepath.Join(xdgDir, "digest", "sync.json")
	actualPath := ConfigPath()
	assert.Equal(t, expectedPath, actualPath)
}

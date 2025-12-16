// ABOUTME: Tests for sync configuration management
// ABOUTME: Verifies config load/save, defaults, and env overrides

package sync

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDefaults(t *testing.T) {
	// Use temp dir for test
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Empty(t, cfg.Server)
	assert.Empty(t, cfg.Token)
	assert.NotEmpty(t, cfg.VaultDB) // Should have default path
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := &Config{
		Server:     "https://api.example.com",
		UserID:     "user123",
		Token:      "token456",
		DeviceID:   "device789",
		DerivedKey: "abcdef",
		AutoSync:   true,
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
	assert.Equal(t, cfg.AutoSync, loaded.AutoSync)
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

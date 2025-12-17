// ABOUTME: Shared test helpers for sync package tests
// ABOUTME: Provides database setup and syncer creation utilities

package sync

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/harper/digest/internal/db"
	"github.com/harperreed/sweet/vault"
	"github.com/stretchr/testify/require"
)

// setupTestSyncer creates a test syncer with default test config.
func setupTestSyncer(t *testing.T) *Syncer {
	t.Helper()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	t.Cleanup(func() { _ = appDB.Close() })

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = syncer.Close() })

	return syncer
}

// setupTestSyncerWithDB creates a test syncer and returns both syncer and appDB.
func setupTestSyncerWithDB(t *testing.T) (*Syncer, *sql.DB) {
	t.Helper()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	t.Cleanup(func() { _ = appDB.Close() })

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   false,
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	t.Cleanup(func() { _ = syncer.Close() })

	return syncer, appDB
}

// setupTestDB creates a minimal test database with digest schema.
func setupTestDB(t *testing.T, tmpDir string) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.InitDB(dbPath)
	require.NoError(t, err)

	return database
}

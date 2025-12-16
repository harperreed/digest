// ABOUTME: Tests for vault sync operations
// ABOUTME: Verifies syncer creation, entity queueing, and change application

package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncerRequiresDerivedKey(t *testing.T) {
	cfg := &Config{
		Server:   "https://api.example.com",
		Token:    "token",
		DeviceID: "device",
		// DerivedKey intentionally empty
	}

	_, err := NewSyncer(cfg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "derived key")
}

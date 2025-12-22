// ABOUTME: Tests for configuration defaults
// ABOUTME: Verifies constants are properly defined

package config

import (
	"testing"
	"time"
)

func TestDefaultHTTPTimeout(t *testing.T) {
	if DefaultHTTPTimeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", DefaultHTTPTimeout)
	}
}

func TestDisplayConstants(t *testing.T) {
	if DefaultListLimit <= 0 {
		t.Error("DefaultListLimit should be positive")
	}
	if DisplayIDLength <= 0 {
		t.Error("DisplayIDLength should be positive")
	}
}

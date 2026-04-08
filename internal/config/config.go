// ABOUTME: Configuration management with storage backend selection
// ABOUTME: Handles settings, preferences, and storage backend factory function

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/harper/digest/internal/storage"
	"github.com/harperreed/mdstore"
)

// Config stores digest configuration.
type Config struct {
	// Backend selects the storage backend: "sqlite" (default) or "markdown".
	Backend string `json:"backend,omitempty"`

	// DataDir is the root directory for data storage.
	// SQLite puts digest.db here. Markdown puts _feeds.yaml and feed folders here.
	// Supports ~ expansion for home directory. Defaults to ~/.local/share/digest.
	DataDir string `json:"data_dir,omitempty"`
}

// defaultDBFilename is the SQLite database filename used for existing-user detection.
const defaultDBFilename = "digest.db"

// GetBackend returns the configured backend, defaulting to "sqlite".
func (c *Config) GetBackend() string {
	if c.Backend == "" {
		return "sqlite"
	}
	return c.Backend
}

// GetDataDir returns the configured data directory with ~ expanded,
// defaulting to the standard XDG data directory.
func (c *Config) GetDataDir() string {
	if c.DataDir == "" {
		return defaultDataDir()
	}
	return ExpandPath(c.DataDir)
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// ValidateProfileName checks that a profile name is safe to use as a directory name.
// Names must be 1-64 alphanumeric characters, hyphens, underscores, or dots,
// must start with an alphanumeric character, and must not be Windows reserved names.
func ValidateProfileName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must be 1-64 alphanumeric characters, hyphens, underscores, or dots (must start with alphanumeric)", name)
	}
	if windowsReserved[strings.ToLower(name)] {
		return fmt.Errorf("invalid profile name %q: reserved name", name)
	}
	return nil
}

// ProfileDataDir returns the data directory for a named profile.
// Each profile is a subdirectory under the main data directory.
// Returns an error if the profile name is invalid.
func (c *Config) ProfileDataDir(profile string) (string, error) {
	if err := ValidateProfileName(profile); err != nil {
		return "", err
	}
	return filepath.Join(c.GetDataDir(), profile), nil
}

// openStore creates a Store for the given backend and data directory.
func (c *Config) openStore(backend, dataDir string) (storage.Store, error) {
	switch backend {
	case "sqlite":
		dbPath := filepath.Join(dataDir, "digest.db")
		return storage.NewSQLiteStore(dbPath)
	case "markdown":
		return storage.NewMarkdownStore(dataDir)
	default:
		return nil, fmt.Errorf("unknown backend: %q", backend)
	}
}

// OpenStorage creates a Store implementation based on the configured backend.
func (c *Config) OpenStorage() (storage.Store, error) {
	return c.openStore(c.GetBackend(), c.GetDataDir())
}

// OpenProfileStorage creates a Store for the given profile.
// The profile's data directory is auto-created if it doesn't exist.
func (c *Config) OpenProfileStorage(profile string) (storage.Store, error) {
	profileDir, err := c.ProfileDataDir(profile)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}

	return c.openStore(c.GetBackend(), profileDir)
}

// MigrateToProfileLayout moves flat-layout data files (digest.db, feeds.opml)
// from the data dir root into a "default" profile subdirectory.
// This is a one-time, idempotent migration for upgrading from pre-profile layout.
func (c *Config) MigrateToProfileLayout() error {
	dataDir := c.GetDataDir()

	// Check for flat layout files at the root
	dbPath := filepath.Join(dataDir, "digest.db")
	opmlPath := filepath.Join(dataDir, "feeds.opml")

	dbExists := fileExists(dbPath)
	opmlExists := fileExists(opmlPath)

	// Nothing to migrate
	if !dbExists && !opmlExists {
		return nil
	}

	// Create default profile directory
	defaultDir := filepath.Join(dataDir, "default")
	if err := os.MkdirAll(defaultDir, 0750); err != nil {
		return fmt.Errorf("failed to create default profile directory: %w", err)
	}

	// Move files
	if dbExists {
		dst := filepath.Join(defaultDir, "digest.db")
		if err := os.Rename(dbPath, dst); err != nil {
			return fmt.Errorf("failed to move digest.db: %w", err)
		}
	}
	if opmlExists {
		dst := filepath.Join(defaultDir, "feeds.opml")
		if err := os.Rename(opmlPath, dst); err != nil {
			return fmt.Errorf("failed to move feeds.opml: %w", err)
		}
	}

	return nil
}

// fileExists checks if a path exists and is a regular file (not a directory).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetConfigPath returns the config file path.
func GetConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "digest", "config.json")
}

// Load reads config from disk.
func Load() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultFirstRunConfig()
			if saveErr := cfg.Save(); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save default config: %v\n", saveErr)
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes config to disk.
func (c *Config) Save() error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return mdstore.AtomicWrite(path, data)
}

// defaultDataDir returns the standard XDG data directory for digest.
func defaultDataDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "digest")
}

// defaultFirstRunConfig returns the appropriate default config for first-time runs.
// If an existing SQLite database is found, it preserves SQLite as the backend.
// Otherwise, it defaults to markdown for new users.
func defaultFirstRunConfig() *Config {
	dbPath := filepath.Join(defaultDataDir(), defaultDBFilename)
	_, err := os.Stat(dbPath)
	switch {
	case err == nil:
		return &Config{Backend: "sqlite"}
	case !os.IsNotExist(err):
		fmt.Fprintf(os.Stderr, "warning: could not check for existing database: %v\n", err)
	}
	return &Config{Backend: "markdown"}
}

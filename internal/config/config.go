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

// MigrateToProfileLayout moves flat-layout data files from the data dir root
// into a "default" profile subdirectory. Handles SQLite DB + WAL/SHM sidecars,
// OPML, the markdown feed registry (_feeds.yaml), and markdown feed directories.
// This is a one-time, idempotent migration for upgrading from pre-profile layout.
func (c *Config) MigrateToProfileLayout() error {
	dataDir := c.GetDataDir()

	// Detect flat layout by checking for known data files at root
	knownFiles := []string{"digest.db", "feeds.opml", "_feeds.yaml"}
	needsMigration := false
	for _, name := range knownFiles {
		if fileExists(filepath.Join(dataDir, name)) {
			needsMigration = true
			break
		}
	}

	if !needsMigration {
		return nil
	}

	// Create default profile directory
	defaultDir := filepath.Join(dataDir, "default")
	if err := os.MkdirAll(defaultDir, 0700); err != nil {
		return fmt.Errorf("failed to create default profile directory: %w", err)
	}

	// Move known files (SQLite DB + sidecars, OPML, markdown feed registry)
	filesToMove := []string{
		"digest.db", "digest.db-wal", "digest.db-shm",
		"feeds.opml",
		"_feeds.yaml",
	}
	for _, name := range filesToMove {
		src := filepath.Join(dataDir, name)
		if !fileExists(src) {
			continue
		}
		dst := filepath.Join(defaultDir, name)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s: %w", name, err)
		}
	}

	// Move remaining directories at root into default/ (markdown feed directories).
	// Skip "default" itself.
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "default" {
			continue
		}
		src := filepath.Join(dataDir, entry.Name())
		dst := filepath.Join(defaultDir, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move directory %s: %w", entry.Name(), err)
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
	dataDir := defaultDataDir()
	for _, path := range []string{
		filepath.Join(dataDir, defaultDBFilename),
		filepath.Join(dataDir, "default", defaultDBFilename),
	} {
		if _, err := os.Stat(path); err == nil {
			return &Config{Backend: "sqlite"}
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: could not check for existing database: %v\n", err)
		}
	}
	return &Config{Backend: "markdown"}
}

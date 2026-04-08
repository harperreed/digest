# Profiles Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix critical and important issues found during expert review of the profiles feature — path traversal, migration data safety, broken migrate command, and UX polish.

**Architecture:** Add `ValidateProfileName` as a gatekeeper for all profile operations. Rewrite migration to handle SQLite WAL/SHM files and markdown backend data. Fix migrate command to be profile-aware. Polish UX (verb consistency, help text, shorthand).

**Tech Stack:** Go 1.24, Cobra CLI, modernc.org/sqlite

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/config/config.go` | Modify | Add `ValidateProfileName`, fix migration, fix permissions, fix `defaultFirstRunConfig`, deduplicate storage factory |
| `internal/config/config_test.go` | Modify | Tests for validation, expanded migration tests |
| `cmd/digest/root.go` | Modify | Validate profile early, fix `--opml` help text, add `-p` shorthand |
| `cmd/digest/profile.go` | Modify | Rename `delete` to `remove`, case-insensitive default check, validate name in delete |
| `cmd/digest/profile_test.go` | Modify | Update tests for rename and validation |
| `cmd/digest/cmd_test.go` | Modify | Update command registration tests |
| `cmd/digest/migrate.go` | Modify | Use profile-aware storage |

---

### Task 1: ValidateProfileName — Input Validation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for ValidateProfileName**

Add to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run TestValidateProfileName -v`

Expected: compilation error — `ValidateProfileName` undefined.

- [ ] **Step 3: Implement ValidateProfileName**

Add to `internal/config/config.go`, after the imports. Add `"regexp"` to the import block:

```go
// profileNamePattern allows alphanumeric characters, hyphens, underscores, and dots.
// Must start with an alphanumeric character, max 64 characters.
var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// windowsReserved lists names that are reserved on Windows filesystems.
var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// ValidateProfileName checks that a profile name is safe for use as a directory name.
func ValidateProfileName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must be 1-64 alphanumeric characters, hyphens, underscores, or dots (must start with alphanumeric)", name)
	}
	if windowsReserved[strings.ToLower(name)] {
		return fmt.Errorf("invalid profile name %q: reserved name", name)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run TestValidateProfileName -v`

Expected: all tests PASS.

- [ ] **Step 5: Wire validation into ProfileDataDir and OpenProfileStorage**

Update `ProfileDataDir` in `internal/config/config.go`:

```go
// ProfileDataDir returns the data directory for a named profile.
// Each profile is a subdirectory under the main data directory.
// Returns an error if the profile name is invalid.
func (c *Config) ProfileDataDir(profile string) (string, error) {
	if err := ValidateProfileName(profile); err != nil {
		return "", err
	}
	return filepath.Join(c.GetDataDir(), profile), nil
}
```

Update `OpenProfileStorage` to handle the new error return:

```go
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
```

Note: also change `0750` to `0700` (finding #13).

- [ ] **Step 6: Fix all callers of ProfileDataDir to handle error**

In `cmd/digest/root.go`, update the OPML path line:

```go
	profileDir, err := cfg.ProfileDataDir(profileName)
	if err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}
	if opmlPath == "" {
		opmlPath = filepath.Join(profileDir, "feeds.opml")
	}

	store, err = cfg.OpenProfileStorage(profileName)
```

In `cmd/digest/profile.go` delete command, add validation:

```go
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
```

Update `GetDefaultOPMLPath` in `root.go`:

```go
func GetDefaultOPMLPath() string {
	cfg, err := config.Load()
	if err != nil {
		return filepath.Join(getDataDir(), "digest", "default", "feeds.opml")
	}
	profileDir, err := cfg.ProfileDataDir("default")
	if err != nil {
		return filepath.Join(getDataDir(), "digest", "default", "feeds.opml")
	}
	return filepath.Join(profileDir, "feeds.opml")
}
```

- [ ] **Step 7: Update existing ProfileDataDir tests**

All existing `TestProfileDataDir*` tests call `cfg.ProfileDataDir(name)` which now returns `(string, error)`. Update them:

```go
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
```

Repeat the same pattern for `TestProfileDataDirDefault`, `TestProfileDataDirTildeExpansion`, `TestProfileDataDirDefaultDataDir`. Also update `TestProfileIsolation` which calls `cfg.ProfileDataDir`.

- [ ] **Step 8: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... -count=1`

Expected: all tests PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go cmd/digest/root.go cmd/digest/profile.go
git commit -m "fix(security): add profile name validation to prevent path traversal"
```

---

### Task 2: Deduplicate Storage Factory

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Extract shared openStore helper**

Add a private helper to `internal/config/config.go`:

```go
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
```

- [ ] **Step 2: Refactor OpenStorage to use openStore**

```go
func (c *Config) OpenStorage() (storage.Store, error) {
	return c.openStore(c.GetBackend(), c.GetDataDir())
}
```

- [ ] **Step 3: Refactor OpenProfileStorage to use openStore**

(Already done in Task 1 Step 5 if following plan in order.)

- [ ] **Step 4: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... -count=1`

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor(config): deduplicate storage factory into shared openStore helper"
```

---

### Task 3: Fix Migration — WAL/SHM Files and Markdown Backend

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for WAL/SHM migration**

Add to `internal/config/config_test.go`:

```go
func TestMigrateToProfileLayout_MovesWALFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create SQLite files with WAL/SHM sidecars
	os.WriteFile(filepath.Join(tmpDir, "digest.db"), []byte("db"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "digest.db-wal"), []byte("wal"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "digest.db-shm"), []byte("shm"), 0600)

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	// All three files should be in default/
	for _, name := range []string{"digest.db", "digest.db-wal", "digest.db-shm"} {
		path := filepath.Join(tmpDir, "default", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %s in default/", name)
		}
	}

	// Old files should be gone
	for _, name := range []string{"digest.db", "digest.db-wal", "digest.db-shm"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed from root", name)
		}
	}
}

func TestMigrateToProfileLayout_MovesMarkdownFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create markdown backend files
	os.WriteFile(filepath.Join(tmpDir, "_feeds.yaml"), []byte("feeds"), 0600)
	feedDir := filepath.Join(tmpDir, "my-feed")
	os.MkdirAll(feedDir, 0750)
	os.WriteFile(filepath.Join(feedDir, "entry.md"), []byte("content"), 0600)

	cfg := &Config{DataDir: tmpDir}
	if err := cfg.MigrateToProfileLayout(); err != nil {
		t.Fatalf("MigrateToProfileLayout failed: %v", err)
	}

	// _feeds.yaml should be in default/
	if _, err := os.Stat(filepath.Join(tmpDir, "default", "_feeds.yaml")); os.IsNotExist(err) {
		t.Error("expected _feeds.yaml in default/")
	}
	// Feed directory should be in default/
	if _, err := os.Stat(filepath.Join(tmpDir, "default", "my-feed", "entry.md")); os.IsNotExist(err) {
		t.Error("expected my-feed/entry.md in default/")
	}

	// Old files should be gone
	if _, err := os.Stat(filepath.Join(tmpDir, "_feeds.yaml")); !os.IsNotExist(err) {
		t.Error("expected _feeds.yaml to be removed from root")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run "TestMigrateToProfileLayout_MovesWAL|TestMigrateToProfileLayout_MovesMarkdown" -v`

Expected: FAIL — WAL files and markdown files not moved.

- [ ] **Step 3: Rewrite MigrateToProfileLayout**

Replace the existing `MigrateToProfileLayout` in `internal/config/config.go`:

```go
// MigrateToProfileLayout moves flat-layout data files from the data dir root
// into a "default" profile subdirectory.
// Handles SQLite (digest.db + WAL/SHM sidecars), OPML (feeds.opml),
// and markdown backend (_feeds.yaml + feed directories).
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

	// Move feed directories (markdown backend stores feeds as subdirectories)
	// Any directory at the root that is NOT "default" (or another profile) is a feed dir
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
```

- [ ] **Step 4: Update MkdirAll permission in migration to 0700**

Already done in step 3 above (changed from `0750` to `0700`).

- [ ] **Step 5: Run migration tests**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run TestMigrateToProfileLayout -v`

Expected: all migration tests PASS (including old ones).

- [ ] **Step 6: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... -count=1`

Expected: all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "fix(migration): handle SQLite WAL/SHM files and markdown backend data"
```

---

### Task 4: Fix defaultFirstRunConfig for Profile Layout

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/config/config_test.go`:

```go
func TestDefaultFirstRunConfig_SQLiteInProfileLayout(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Create SQLite DB in profile layout (default/)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run TestDefaultFirstRunConfig_SQLiteInProfileLayout -v`

Expected: FAIL — returns "markdown" because it only checks root path.

- [ ] **Step 3: Fix defaultFirstRunConfig**

Update in `internal/config/config.go`:

```go
func defaultFirstRunConfig() *Config {
	dataDir := defaultDataDir()
	// Check for SQLite DB at root (flat layout) or in default profile
	for _, path := range []string{
		filepath.Join(dataDir, defaultDBFilename),
		filepath.Join(dataDir, "default", defaultDBFilename),
	} {
		if _, err := os.Stat(path); err == nil {
			return &Config{Backend: "sqlite"}
		}
	}
	return &Config{Backend: "markdown"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./internal/config/ -run TestDefaultFirstRunConfig -v`

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "fix(config): detect SQLite DB in profile layout for first-run config"
```

---

### Task 5: Fix migrate Command for Profile Layout

**Files:**
- Modify: `cmd/digest/migrate.go`

- [ ] **Step 1: Update migrate to use profile-aware storage**

In `cmd/digest/migrate.go`, update the `runMigrate` function. Replace lines 80-84:

```go
	// Open source storage (profile-aware)
	if err := cfg.MigrateToProfileLayout(); err != nil {
		return fmt.Errorf("migrate layout: %w", err)
	}
	src, err := cfg.OpenProfileStorage(profileName)
	if err != nil {
		return fmt.Errorf("open source storage (%s): %w", sourceBackend, err)
	}
	defer src.Close()
```

Also update the target data dir to be profile-aware when no explicit `--data-dir` is given. Replace line 66:

```go
	targetDataDir, err := cfg.ProfileDataDir(profileName)
	if err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}
	if migrateDataDir != "" {
		targetDataDir = config.ExpandPath(migrateDataDir)
	}
```

- [ ] **Step 2: Remove openMigrateStorage**

Delete the `openMigrateStorage` function (lines 122-133) since we're using `openStore` on config now. Replace its call on line 88 with:

```go
	dst, err := (&config.Config{Backend: targetBackend}).OpenStorage()
```

Wait — we can't do that cleanly since `openStore` is unexported. Keep `openMigrateStorage` for now since it's the target storage (not source). The key fix is line 81 using `OpenProfileStorage` for the source.

- [ ] **Step 3: Run tests**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./cmd/digest/ -v`

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/digest/migrate.go
git commit -m "fix(migrate): use profile-aware storage for source backend"
```

---

### Task 6: UX Polish — Rename delete to remove, Help Text, Shorthand

**Files:**
- Modify: `cmd/digest/profile.go`
- Modify: `cmd/digest/profile_test.go`
- Modify: `cmd/digest/cmd_test.go`
- Modify: `cmd/digest/root.go`

- [ ] **Step 1: Rename profile delete to profile remove**

In `cmd/digest/profile.go`, update `profileDeleteCmd`:

```go
var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile and all its data",
	Long:  "Remove a profile directory and all feeds, entries, and OPML data within it",
	Args:  cobra.ExactArgs(1),
```

Update all references: rename the variable from `profileDeleteCmd` to `profileRemoveCmd` throughout the file. In `init()`:

```go
	profileCmd.AddCommand(profileRemoveCmd)
	profileRemoveCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
```

- [ ] **Step 2: Fix case-insensitive default check**

In the `profileRemoveCmd` RunE, replace the default check:

```go
		if strings.EqualFold(name, "default") {
			return fmt.Errorf("cannot remove the default profile")
		}
```

Add `"strings"` to the import block.

- [ ] **Step 3: Add -p shorthand for --profile**

In `cmd/digest/root.go`, update the init function:

```go
	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "default", "profile name (e.g., work, personal). Profiles keep separate sets of feeds. Omit for default profile")
```

Also fix the stale `--opml` help text:

```go
	rootCmd.PersistentFlags().StringVar(&opmlPath, "opml", "", "OPML file path (default: <data-dir>/<profile>/feeds.opml)")
```

- [ ] **Step 4: Update profile_test.go**

Update `cmd/digest/profile_test.go` to match renamed command:

```go
func TestProfileRemoveCommand(t *testing.T) {
	if profileRemoveCmd.Use != "remove <name>" {
		t.Errorf("expected Use to be 'remove <name>', got %q", profileRemoveCmd.Use)
	}
}
```

- [ ] **Step 5: Update cmd_test.go**

In `TestProfileSubcommands`, change `"delete"` to `"remove"`:

```go
	expectedCommands := []string{
		"list",
		"remove",
	}
```

- [ ] **Step 6: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./cmd/digest/ -v`

Expected: all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/digest/profile.go cmd/digest/profile_test.go cmd/digest/cmd_test.go cmd/digest/root.go
git commit -m "fix(cli): rename profile delete to remove, add -p shorthand, improve help text"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go test ./... -count=1`

Expected: all tests PASS.

- [ ] **Step 2: Run linter**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go vet ./...`

Expected: no issues.

- [ ] **Step 3: Build and smoke test**

Run: `cd /Users/harper/Public/src/personal/suite/digest && go build -o /tmp/digest-test ./cmd/digest/`

Test path traversal is blocked:
```bash
/tmp/digest-test --profile "../../../tmp" feed list
```
Expected: error about invalid profile name.

Test shorthand works:
```bash
/tmp/digest-test -p work profile list
```
Expected: works (auto-creates work profile or shows no feeds).

Test remove command:
```bash
/tmp/digest-test profile remove Default
```
Expected: error "cannot remove the default profile" (case-insensitive).

- [ ] **Step 4: Commit any final fixes**

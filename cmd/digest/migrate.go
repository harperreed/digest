// ABOUTME: Migration command for converting digest data between storage backends
// ABOUTME: Supports sqlite-to-markdown and markdown-to-sqlite with safety checks

package main

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/config"
	"github.com/harper/digest/internal/storage"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data between storage backends",
	Long: `Migrate all digest data from the currently configured backend to a different backend.

Reads feeds and entries from the current backend and writes them to the target
backend. Does NOT update the config file; verify the migration was successful
then update config.json manually.

Examples:
  digest migrate --to markdown
  digest migrate --to sqlite --data-dir ~/digest-sqlite
  digest migrate --to markdown --force`,
	RunE: runMigrate,
}

var (
	migrateTo      string
	migrateDataDir string
	migrateForce   bool
)

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVar(&migrateTo, "to", "", "target backend (sqlite or markdown)")
	migrateCmd.Flags().StringVar(&migrateDataDir, "data-dir", "", "target data directory (defaults to current config data_dir)")
	migrateCmd.Flags().BoolVar(&migrateForce, "force", false, "allow writing into a non-empty target directory")
	_ = migrateCmd.MarkFlagRequired("to")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	// Load config and determine source backend
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sourceBackend := cfg.GetBackend()
	targetBackend := migrateTo

	// Validate target backend
	if targetBackend != "sqlite" && targetBackend != "markdown" {
		return fmt.Errorf("invalid target backend %q: must be \"sqlite\" or \"markdown\"", targetBackend)
	}
	if targetBackend == sourceBackend {
		return fmt.Errorf("target backend %q is the same as the current backend", targetBackend)
	}

	// Determine target data directory
	targetDataDir := cfg.GetDataDir()
	if migrateDataDir != "" {
		targetDataDir = config.ExpandPath(migrateDataDir)
	}

	// Check if target directory is non-empty
	nonEmpty, err := storage.IsDirNonEmpty(targetDataDir)
	if err != nil {
		return fmt.Errorf("check target directory: %w", err)
	}
	if nonEmpty && !migrateForce {
		return fmt.Errorf("target directory %q is not empty; use --force to overwrite", targetDataDir)
	}

	// Open source storage
	src, err := cfg.OpenStorage()
	if err != nil {
		return fmt.Errorf("open source storage (%s): %w", sourceBackend, err)
	}
	defer src.Close()

	// Open target storage
	dst, err := openMigrateStorage(targetBackend, targetDataDir)
	if err != nil {
		return fmt.Errorf("open target storage (%s): %w", targetBackend, err)
	}
	defer dst.Close()

	// Print plan
	color.Yellow("Migrating digest data:")
	fmt.Printf("  Source:  %s (%s)\n", sourceBackend, cfg.GetDataDir())
	fmt.Printf("  Target:  %s (%s)\n", targetBackend, targetDataDir)
	fmt.Println()

	// Run migration
	summary, err := storage.MigrateData(src, dst)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print summary
	color.Green("Migration complete!")
	fmt.Printf("  Feeds:   %d\n", summary.Feeds)
	fmt.Printf("  Entries: %d\n", summary.Entries)
	fmt.Println()
	color.Yellow("Note: config.json was NOT updated. To switch to the new backend, edit:")
	fmt.Printf("  %s\n", config.GetConfigPath())
	fmt.Printf("  Set \"backend\": %q", targetBackend)
	if migrateDataDir != "" {
		fmt.Printf(" and \"data_dir\": %q", migrateDataDir)
	}
	fmt.Println()

	return nil
}

// openMigrateStorage creates a Store implementation for the given backend and data directory.
func openMigrateStorage(backend, dataDir string) (storage.Store, error) {
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

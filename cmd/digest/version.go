// ABOUTME: Version command for digest CLI
// ABOUTME: Displays version, commit, and build date information

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information set via ldflags at build time
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version, commit hash, and build date of digest.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("digest %s\n", Version)
		fmt.Printf("  commit:  %s\n", Commit)
		fmt.Printf("  built:   %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

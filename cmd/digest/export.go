// ABOUTME: Export command for writing OPML document to stdout
// ABOUTME: Outputs the current feed list in OPML format for backup or import

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export OPML to stdout",
	Long:  "Export the current feed list in OPML format to standard output",
	RunE: func(cmd *cobra.Command, args []string) error {
		return opmlDoc.Write(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

// ABOUTME: Entry point for digest CLI
// ABOUTME: Initializes and executes root command

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

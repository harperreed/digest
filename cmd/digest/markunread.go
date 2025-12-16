// ABOUTME: Mark-unread command for marking entries as unread
// ABOUTME: Supports marking a single entry as unread by ID

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"

	"github.com/harper/digest/internal/db"
	"github.com/harper/digest/internal/sync"
)

var markUnreadCmd = &cobra.Command{
	Use:   "mark-unread <entry-id>",
	Short: "Mark an entry as unread",
	Long:  "Mark a single entry as unread by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entryRef := args[0]

		// Get entry by ID or prefix
		entry, err := db.GetEntryByID(dbConn, entryRef)
		if err != nil {
			// Only try prefix match if entry was not found (not for other DB errors)
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get entry: %w", err)
			}
			entry, err = db.GetEntryByPrefix(dbConn, entryRef)
			if err != nil {
				return fmt.Errorf("entry not found: %s", entryRef)
			}
		}

		if !entry.Read {
			fmt.Println("Entry is already marked as unread")
			return nil
		}

		if err := db.MarkEntryUnread(dbConn, entry.ID); err != nil {
			return fmt.Errorf("failed to mark entry as unread: %w", err)
		}

		// Queue read state sync if configured
		ctx := context.Background()
		cfg, _ := sync.LoadConfig()
		if cfg != nil && cfg.IsConfigured() {
			syncer, err := sync.NewSyncer(cfg, dbConn)
			if err == nil {
				defer syncer.Close()
				feedURL := getFeedURLForEntry(entry.FeedID)
				if feedURL != "" {
					if err := syncer.QueueReadStateChange(ctx, feedURL, entry.GUID, false, time.Now()); err != nil {
						log.Printf("warning: failed to queue read state sync: %v", err)
					}
				}
			}
		}

		title := "Untitled"
		if entry.Title != nil {
			title = *entry.Title
		}
		fmt.Printf("Marked as unread: %s\n", title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(markUnreadCmd)
}

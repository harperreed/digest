// ABOUTME: Entry database operations and CRUD functions
// ABOUTME: Handles creating, reading, updating entries with flexible filtering

package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/harper/digest/internal/models"
)

// CreateEntry inserts a new entry into the database
func CreateEntry(db *sql.DB, entry *models.Entry) error {
	query := `
		INSERT INTO entries (id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		entry.ID,
		entry.FeedID,
		entry.GUID,
		entry.Title,
		entry.Link,
		entry.Author,
		entry.PublishedAt,
		entry.Content,
		entry.Read,
		entry.ReadAt,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create entry: %w", err)
	}
	return nil
}

// GetEntryByID retrieves an entry by its ID
func GetEntryByID(db *sql.DB, id string) (*models.Entry, error) {
	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries
		WHERE id = ?
	`
	entry := &models.Entry{}
	err := db.QueryRow(query, id).Scan(
		&entry.ID,
		&entry.FeedID,
		&entry.GUID,
		&entry.Title,
		&entry.Link,
		&entry.Author,
		&entry.PublishedAt,
		&entry.Content,
		&entry.Read,
		&entry.ReadAt,
		&entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	return entry, nil
}

// GetEntryByPrefix finds an entry by ID prefix (minimum 6 characters)
// Returns an error if the prefix is ambiguous (matches multiple entries)
// UUIDs only contain hex characters (0-9, a-f) and hyphens, so no SQL wildcard escaping is needed
func GetEntryByPrefix(db *sql.DB, prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	rows, err := db.Query(`
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries
		WHERE id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries by prefix: %w", err)
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		entry := &models.Entry{}
		if err := rows.Scan(
			&entry.ID,
			&entry.FeedID,
			&entry.GUID,
			&entry.Title,
			&entry.Link,
			&entry.Author,
			&entry.PublishedAt,
			&entry.Content,
			&entry.Read,
			&entry.ReadAt,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entries: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entry found with prefix %s", prefix)
	}
	if len(entries) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d entries", prefix, len(entries))
	}
	return entries[0], nil
}

// ListEntries retrieves entries with flexible filtering
// All parameters are optional via pointers:
// - feedID: filter by specific feed (nil = all feeds)
// - feedIDs: filter by multiple feeds (nil = ignored, takes precedence over feedID)
// - unreadOnly: filter by read status (nil = all entries)
// - since: filter by published_at >= since (nil = no time filter)
// - until: filter by published_at < until (nil = no upper time bound)
// - limit: maximum number of results (nil = no limit)
func ListEntries(db *sql.DB, feedID *string, feedIDs []string, unreadOnly *bool, since *time.Time, until *time.Time, limit *int) ([]*models.Entry, error) {
	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries
		WHERE 1=1
	`
	args := []interface{}{}

	// feedIDs takes precedence over feedID
	if len(feedIDs) > 0 {
		placeholders := make([]string, len(feedIDs))
		for i, id := range feedIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += " AND feed_id IN (" + strings.Join(placeholders, ",") + ")"
	} else if feedID != nil {
		query += " AND feed_id = ?"
		args = append(args, *feedID)
	}

	if unreadOnly != nil && *unreadOnly {
		query += " AND read = FALSE"
	}

	if since != nil {
		query += " AND published_at >= ?"
		args = append(args, *since)
	}

	if until != nil {
		query += " AND published_at < ?"
		args = append(args, *until)
	}

	query += " ORDER BY published_at DESC"

	if limit != nil {
		query += " LIMIT ?"
		args = append(args, *limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	defer rows.Close()

	entries := []*models.Entry{}
	for rows.Next() {
		entry := &models.Entry{}
		err := rows.Scan(
			&entry.ID,
			&entry.FeedID,
			&entry.GUID,
			&entry.Title,
			&entry.Link,
			&entry.Author,
			&entry.PublishedAt,
			&entry.Content,
			&entry.Read,
			&entry.ReadAt,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entries: %w", err)
	}

	return entries, nil
}

// MarkEntryRead marks an entry as read and sets read_at to current time
func MarkEntryRead(db *sql.DB, id string) error {
	query := `
		UPDATE entries
		SET read = TRUE, read_at = ?
		WHERE id = ?
	`
	now := time.Now()
	_, err := db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark entry as read: %w", err)
	}
	return nil
}

// MarkEntryUnread marks an entry as unread and clears the read_at timestamp
func MarkEntryUnread(db *sql.DB, id string) error {
	query := `
		UPDATE entries
		SET read = FALSE, read_at = NULL
		WHERE id = ?
	`
	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to mark entry as unread: %w", err)
	}
	return nil
}

// MarkEntriesReadBefore marks all unread entries published before the given time as read
// Returns the number of entries that were marked as read
func MarkEntriesReadBefore(db *sql.DB, before time.Time) (int64, error) {
	query := `
		UPDATE entries
		SET read = TRUE, read_at = ?
		WHERE published_at < ? AND read = FALSE
	`
	now := time.Now()
	result, err := db.Exec(query, now, before)
	if err != nil {
		return 0, fmt.Errorf("failed to mark entries as read: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	return count, nil
}

// EntryExists checks if an entry exists with the given feed_id and guid
func EntryExists(db *sql.DB, feedID, guid string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM entries
		WHERE feed_id = ? AND guid = ?
	`
	var count int
	err := db.QueryRow(query, feedID, guid).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check entry existence: %w", err)
	}
	return count > 0, nil
}

// CountUnreadEntries counts unread entries, optionally filtered by feedID
// If feedID is nil, counts all unread entries across all feeds
func CountUnreadEntries(db *sql.DB, feedID *string) (int, error) {
	query := `
		SELECT COUNT(*) FROM entries
		WHERE read = FALSE
	`
	args := []interface{}{}

	if feedID != nil {
		query += " AND feed_id = ?"
		args = append(args, *feedID)
	}

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread entries: %w", err)
	}
	return count, nil
}

// ABOUTME: Feed database operations
// ABOUTME: CRUD operations for the feeds table

package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/harper/digest/internal/models"
)

func CreateFeed(db *sql.DB, feed *models.Feed) error {
	_, err := db.Exec(`
		INSERT INTO feeds (id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		feed.ID, feed.URL, feed.Title, feed.ETag, feed.LastModified,
		feed.LastFetchedAt, feed.LastError, feed.ErrorCount, feed.CreatedAt,
	)
	return err
}

func GetFeedByURL(db *sql.DB, url string) (*models.Feed, error) {
	feed := &models.Feed{}
	err := db.QueryRow(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE url = ?`, url,
	).Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
		&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func GetFeedByPrefix(db *sql.DB, prefix string) (*models.Feed, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	// Escape SQL wildcards in prefix
	escapedPrefix := strings.ReplaceAll(prefix, "%", "\\%")
	escapedPrefix = strings.ReplaceAll(escapedPrefix, "_", "\\_")

	rows, err := db.Query(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id LIKE ? ESCAPE '\'`, escapedPrefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed := &models.Feed{}
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
			&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	if len(feeds) == 0 {
		return nil, fmt.Errorf("no feed found with prefix %s", prefix)
	}
	if len(feeds) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d feeds", prefix, len(feeds))
	}
	return feeds[0], nil
}

func GetFeedByID(db *sql.DB, id string) (*models.Feed, error) {
	feed := &models.Feed{}
	err := db.QueryRow(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id = ?`, id,
	).Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
		&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func ListFeeds(db *sql.DB) ([]*models.Feed, error) {
	rows, err := db.Query(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed := &models.Feed{}
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified,
			&feed.LastFetchedAt, &feed.LastError, &feed.ErrorCount, &feed.CreatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}

func UpdateFeed(db *sql.DB, feed *models.Feed) error {
	_, err := db.Exec(`
		UPDATE feeds SET
			title = ?, etag = ?, last_modified = ?, last_fetched_at = ?,
			last_error = ?, error_count = ?
		WHERE id = ?`,
		feed.Title, feed.ETag, feed.LastModified, feed.LastFetchedAt,
		feed.LastError, feed.ErrorCount, feed.ID,
	)
	return err
}

func DeleteFeed(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM feeds WHERE id = ?", id)
	return err
}

func UpdateFeedFetchState(db *sql.DB, feedID string, etag, lastModified *string, fetchedAt time.Time) error {
	_, err := db.Exec(`
		UPDATE feeds SET etag = ?, last_modified = ?, last_fetched_at = ?, last_error = NULL, error_count = 0
		WHERE id = ?`,
		etag, lastModified, fetchedAt, feedID,
	)
	return err
}

func UpdateFeedError(db *sql.DB, feedID string, errMsg string) error {
	_, err := db.Exec(`
		UPDATE feeds SET last_error = ?, error_count = error_count + 1
		WHERE id = ?`,
		errMsg, feedID,
	)
	return err
}

// FeedStatsRow represents statistics for a single feed.
type FeedStatsRow struct {
	FeedID        string
	FeedURL       string
	FeedTitle     *string
	LastFetchedAt *time.Time
	ErrorCount    int
	LastError     *string
	EntryCount    int
	UnreadCount   int
}

// GetFeedStats retrieves statistics for all feeds in a single query using JOINs.
func GetFeedStats(db *sql.DB) ([]FeedStatsRow, error) {
	rows, err := db.Query(`
		SELECT
			f.id,
			f.url,
			f.title,
			f.last_fetched_at,
			f.error_count,
			f.last_error,
			COALESCE(COUNT(e.id), 0) as entry_count,
			COALESCE(SUM(CASE WHEN e.read = 0 THEN 1 ELSE 0 END), 0) as unread_count
		FROM feeds f
		LEFT JOIN entries e ON f.id = e.feed_id
		GROUP BY f.id, f.url, f.title, f.last_fetched_at, f.error_count, f.last_error
		ORDER BY f.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed stats: %w", err)
	}
	defer rows.Close()

	var stats []FeedStatsRow
	for rows.Next() {
		var stat FeedStatsRow
		if err := rows.Scan(
			&stat.FeedID,
			&stat.FeedURL,
			&stat.FeedTitle,
			&stat.LastFetchedAt,
			&stat.ErrorCount,
			&stat.LastError,
			&stat.EntryCount,
			&stat.UnreadCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan feed stat: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed stats: %w", err)
	}

	return stats, nil
}

// OverallStats represents overall database statistics.
type OverallStats struct {
	TotalFeeds   int
	TotalEntries int
	UnreadCount  int
}

// GetOverallStats retrieves overall statistics in a single query.
func GetOverallStats(db *sql.DB) (*OverallStats, error) {
	var stats OverallStats
	err := db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM feeds) as total_feeds,
			(SELECT COUNT(*) FROM entries) as total_entries,
			(SELECT COUNT(*) FROM entries WHERE read = 0) as unread_count
	`).Scan(&stats.TotalFeeds, &stats.TotalEntries, &stats.UnreadCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query overall stats: %w", err)
	}
	return &stats, nil
}

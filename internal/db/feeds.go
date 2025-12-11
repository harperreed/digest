// ABOUTME: Feed database operations
// ABOUTME: CRUD operations for the feeds table

package db

import (
	"database/sql"
	"fmt"
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

	rows, err := db.Query(`
		SELECT id, url, title, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id LIKE ?`, prefix+"%",
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

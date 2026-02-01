// ABOUTME: SQLite storage implementation using modernc.org/sqlite (pure Go)
// ABOUTME: Provides feed and entry persistence with FTS5 full-text search

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/digest/internal/models"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite storage instance.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database tables if they don't exist.
func (s *SQLiteStore) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS feeds (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			url TEXT UNIQUE NOT NULL,
			title TEXT,
			folder TEXT DEFAULT '',
			etag TEXT,
			last_modified TEXT,
			last_fetched_at TIMESTAMP,
			last_error TEXT,
			error_count INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_feeds_url ON feeds(url);
		CREATE INDEX IF NOT EXISTS idx_feeds_id ON feeds(id);

		CREATE TABLE IF NOT EXISTS entries (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			feed_id TEXT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
			guid TEXT NOT NULL,
			title TEXT,
			link TEXT,
			author TEXT,
			published_at TIMESTAMP,
			content TEXT,
			read INTEGER DEFAULT 0,
			read_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			UNIQUE(feed_id, guid)
		);

		CREATE INDEX IF NOT EXISTS idx_entries_feed_id ON entries(feed_id);
		CREATE INDEX IF NOT EXISTS idx_entries_read ON entries(read);
		CREATE INDEX IF NOT EXISTS idx_entries_published_at ON entries(published_at);
		CREATE INDEX IF NOT EXISTS idx_entries_id ON entries(id);

		-- FTS5 for content search
		CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
			title,
			content,
			content=entries,
			content_rowid=rowid
		);

		-- Triggers to keep FTS in sync
		CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
			INSERT INTO entries_fts(rowid, title, content)
			VALUES (new.rowid, new.title, new.content);
		END;

		CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, content)
			VALUES ('delete', old.rowid, old.title, old.content);
		END;

		CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, content)
			VALUES ('delete', old.rowid, old.title, old.content);
			INSERT INTO entries_fts(rowid, title, content)
			VALUES (new.rowid, new.title, new.content);
		END;
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Feed Operations

// CreateFeed stores a new feed.
func (s *SQLiteStore) CreateFeed(feed *models.Feed) error {
	query := `
		INSERT INTO feeds (id, url, title, folder, etag, last_modified, last_fetched_at, last_error, error_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		feed.ID, feed.URL, feed.Title, feed.Folder,
		feed.ETag, feed.LastModified, timeToSQL(feed.LastFetchedAt),
		feed.LastError, feed.ErrorCount, feed.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert feed: %w", err)
	}
	return nil
}

// GetFeed retrieves a feed by ID.
func (s *SQLiteStore) GetFeed(id string) (*models.Feed, error) {
	query := `
		SELECT id, url, title, folder, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id = ?
	`
	return s.scanFeed(s.db.QueryRow(query, id))
}

// GetFeedByURL finds a feed by its URL.
func (s *SQLiteStore) GetFeedByURL(url string) (*models.Feed, error) {
	query := `
		SELECT id, url, title, folder, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE url = ?
	`
	return s.scanFeed(s.db.QueryRow(query, url))
}

// GetFeedByPrefix finds a feed by ID prefix (min 6 chars).
func (s *SQLiteStore) GetFeedByPrefix(prefix string) (*models.Feed, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	query := `
		SELECT id, url, title, folder, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds WHERE id LIKE ?
	`
	rows, err := s.db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var matches []*models.Feed
	for rows.Next() {
		feed, err := s.scanFeedFromRows(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, feed)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no feed found with prefix %s", prefix)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d feeds", prefix, len(matches))
	}
	return matches[0], nil
}

// ListFeeds returns all feeds, sorted by creation date (newest first).
func (s *SQLiteStore) ListFeeds() ([]*models.Feed, error) {
	query := `
		SELECT id, url, title, folder, etag, last_modified, last_fetched_at, last_error, error_count, created_at
		FROM feeds ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		feed, err := s.scanFeedFromRows(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

// UpdateFeed updates an existing feed.
func (s *SQLiteStore) UpdateFeed(feed *models.Feed) error {
	query := `
		UPDATE feeds SET
			url = ?, title = ?, folder = ?, etag = ?, last_modified = ?,
			last_fetched_at = ?, last_error = ?, error_count = ?
		WHERE id = ?
	`
	result, err := s.db.Exec(query,
		feed.URL, feed.Title, feed.Folder, feed.ETag, feed.LastModified,
		timeToSQL(feed.LastFetchedAt), feed.LastError, feed.ErrorCount,
		feed.ID,
	)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed not found: %s", feed.ID)
	}
	return nil
}

// DeleteFeed removes a feed and all its entries (cascade).
func (s *SQLiteStore) DeleteFeed(id string) error {
	result, err := s.db.Exec("DELETE FROM feeds WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed not found: %s", id)
	}
	return nil
}

// UpdateFeedFetchState updates feed caching headers and clears errors.
func (s *SQLiteStore) UpdateFeedFetchState(feedID string, etag, lastModified *string, fetchedAt time.Time) error {
	query := `
		UPDATE feeds SET
			etag = ?, last_modified = ?, last_fetched_at = ?,
			last_error = NULL, error_count = 0
		WHERE id = ?
	`
	result, err := s.db.Exec(query, etag, lastModified, fetchedAt, feedID)
	if err != nil {
		return fmt.Errorf("update feed fetch state: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed not found: %s", feedID)
	}
	return nil
}

// UpdateFeedError records a fetch error for a feed.
func (s *SQLiteStore) UpdateFeedError(feedID string, errMsg string) error {
	query := `UPDATE feeds SET last_error = ?, error_count = error_count + 1 WHERE id = ?`
	result, err := s.db.Exec(query, errMsg, feedID)
	if err != nil {
		return fmt.Errorf("update feed error: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed not found: %s", feedID)
	}
	return nil
}

// Entry Operations

// CreateEntry stores a new entry.
func (s *SQLiteStore) CreateEntry(entry *models.Entry) error {
	query := `
		INSERT INTO entries (id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		entry.ID, entry.FeedID, entry.GUID, entry.Title, entry.Link, entry.Author,
		timeToSQL(entry.PublishedAt), entry.Content, boolToInt(entry.Read),
		timeToSQL(entry.ReadAt), entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert entry: %w", err)
	}
	return nil
}

// GetEntry retrieves an entry by ID.
func (s *SQLiteStore) GetEntry(id string) (*models.Entry, error) {
	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries WHERE id = ?
	`
	return s.scanEntry(s.db.QueryRow(query, id))
}

// GetEntryByPrefix finds an entry by ID prefix (min 6 chars).
func (s *SQLiteStore) GetEntryByPrefix(prefix string) (*models.Entry, error) {
	if len(prefix) < 6 {
		return nil, fmt.Errorf("prefix must be at least 6 characters")
	}

	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries WHERE id LIKE ?
	`
	rows, err := s.db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var matches []*models.Entry
	for rows.Next() {
		entry, err := s.scanEntryFromRows(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, entry)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no entry found with prefix %s", prefix)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %s matches %d entries", prefix, len(matches))
	}
	return matches[0], nil
}

// ListEntries returns entries matching the filter, sorted by published date.
func (s *SQLiteStore) ListEntries(filter *EntryFilter) ([]*models.Entry, error) {
	query := `
		SELECT id, feed_id, guid, title, link, author, published_at, content, read, read_at, created_at
		FROM entries
	`

	var conditions []string
	var args []interface{}

	if filter != nil {
		// FeedIDs takes precedence over FeedID
		if len(filter.FeedIDs) > 0 {
			placeholders := make([]string, len(filter.FeedIDs))
			for i, id := range filter.FeedIDs {
				placeholders[i] = "?"
				args = append(args, id)
			}
			conditions = append(conditions, "feed_id IN ("+strings.Join(placeholders, ",")+")")
		} else if filter.FeedID != nil {
			conditions = append(conditions, "feed_id = ?")
			args = append(args, *filter.FeedID)
		}

		if filter.UnreadOnly != nil && *filter.UnreadOnly {
			conditions = append(conditions, "read = 0")
		}

		if filter.Since != nil {
			conditions = append(conditions, "published_at >= ?")
			args = append(args, *filter.Since)
		}

		if filter.Until != nil {
			conditions = append(conditions, "published_at < ?")
			args = append(args, *filter.Until)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY published_at DESC"

	if filter != nil {
		if filter.Limit != nil {
			query += fmt.Sprintf(" LIMIT %d", *filter.Limit)
		}
		if filter.Offset != nil {
			if filter.Limit == nil {
				query += " LIMIT -1"
			}
			query += fmt.Sprintf(" OFFSET %d", *filter.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		entry, err := s.scanEntryFromRows(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// UpdateEntry updates an existing entry.
func (s *SQLiteStore) UpdateEntry(entry *models.Entry) error {
	query := `
		UPDATE entries SET
			title = ?, link = ?, author = ?, published_at = ?,
			content = ?, read = ?, read_at = ?
		WHERE id = ?
	`
	result, err := s.db.Exec(query,
		entry.Title, entry.Link, entry.Author, timeToSQL(entry.PublishedAt),
		entry.Content, boolToInt(entry.Read), timeToSQL(entry.ReadAt),
		entry.ID,
	)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found: %s", entry.ID)
	}
	return nil
}

// DeleteEntry removes an entry.
func (s *SQLiteStore) DeleteEntry(id string) error {
	result, err := s.db.Exec("DELETE FROM entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found: %s", id)
	}
	return nil
}

// MarkEntryRead marks an entry as read.
func (s *SQLiteStore) MarkEntryRead(id string) error {
	now := time.Now()
	query := `UPDATE entries SET read = 1, read_at = ? WHERE id = ?`
	result, err := s.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("mark entry read: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found: %s", id)
	}
	return nil
}

// MarkEntryUnread marks an entry as unread.
func (s *SQLiteStore) MarkEntryUnread(id string) error {
	query := `UPDATE entries SET read = 0, read_at = NULL WHERE id = ?`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("mark entry unread: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found: %s", id)
	}
	return nil
}

// MarkEntriesReadBefore marks all unread entries before the given time as read.
func (s *SQLiteStore) MarkEntriesReadBefore(before time.Time) (int64, error) {
	now := time.Now()
	query := `UPDATE entries SET read = 1, read_at = ? WHERE read = 0 AND published_at < ?`
	result, err := s.db.Exec(query, now, before)
	if err != nil {
		return 0, fmt.Errorf("mark entries read before: %w", err)
	}
	return result.RowsAffected()
}

// EntryExists checks if an entry exists with the given feed_id and guid.
func (s *SQLiteStore) EntryExists(feedID, guid string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM entries WHERE feed_id = ? AND guid = ?`
	if err := s.db.QueryRow(query, feedID, guid).Scan(&count); err != nil {
		return false, fmt.Errorf("check entry exists: %w", err)
	}
	return count > 0, nil
}

// CountUnreadEntries counts unread entries, optionally filtered by feedID.
func (s *SQLiteStore) CountUnreadEntries(feedID *string) (int, error) {
	var count int
	var query string
	var args []interface{}

	if feedID != nil {
		query = `SELECT COUNT(*) FROM entries WHERE read = 0 AND feed_id = ?`
		args = append(args, *feedID)
	} else {
		query = `SELECT COUNT(*) FROM entries WHERE read = 0`
	}

	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread entries: %w", err)
	}
	return count, nil
}

// Statistics

// GetFeedStats retrieves statistics for all feeds.
func (s *SQLiteStore) GetFeedStats() ([]FeedStatsRow, error) {
	query := `
		SELECT f.id, f.url, f.title, f.last_fetched_at, f.error_count, f.last_error,
			   COUNT(e.id) as entry_count,
			   SUM(CASE WHEN e.read = 0 THEN 1 ELSE 0 END) as unread_count
		FROM feeds f
		LEFT JOIN entries e ON f.id = e.feed_id
		GROUP BY f.id
		ORDER BY f.created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query feed stats: %w", err)
	}
	defer rows.Close()

	var stats []FeedStatsRow
	for rows.Next() {
		var row FeedStatsRow
		var lastFetched sql.NullTime
		var unreadCount sql.NullInt64
		if err := rows.Scan(
			&row.FeedID, &row.FeedURL, &row.FeedTitle, &lastFetched,
			&row.ErrorCount, &row.LastError, &row.EntryCount, &unreadCount,
		); err != nil {
			return nil, fmt.Errorf("scan feed stats: %w", err)
		}
		if lastFetched.Valid {
			row.LastFetchedAt = &lastFetched.Time
		}
		if unreadCount.Valid {
			row.UnreadCount = int(unreadCount.Int64)
		}
		stats = append(stats, row)
	}
	return stats, nil
}

// GetOverallStats retrieves overall statistics.
func (s *SQLiteStore) GetOverallStats() (*OverallStats, error) {
	var stats OverallStats

	// Total feeds
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM feeds`).Scan(&stats.TotalFeeds); err != nil {
		return nil, fmt.Errorf("count feeds: %w", err)
	}

	// Total entries
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&stats.TotalEntries); err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}

	// Unread count
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM entries WHERE read = 0`).Scan(&stats.UnreadCount); err != nil {
		return nil, fmt.Errorf("count unread: %w", err)
	}

	return &stats, nil
}

// Retrieval helpers

// GetEntryByIDOrPrefix tries to get an entry by exact ID first,
// then falls back to prefix matching if not found.
func (s *SQLiteStore) GetEntryByIDOrPrefix(ref string) (*models.Entry, error) {
	entry, err := s.GetEntry(ref)
	if err == nil {
		return entry, nil
	}

	// Try prefix match
	entry, err = s.GetEntryByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("entry not found: %s", ref)
	}
	return entry, nil
}

// GetFeedByURLOrPrefix tries to get a feed by exact URL first,
// then falls back to prefix matching if not found.
func (s *SQLiteStore) GetFeedByURLOrPrefix(ref string) (*models.Feed, error) {
	feed, err := s.GetFeedByURL(ref)
	if err == nil {
		return feed, nil
	}

	// Try prefix match
	feed, err = s.GetFeedByPrefix(ref)
	if err != nil {
		return nil, fmt.Errorf("feed not found: %s", ref)
	}
	return feed, nil
}

// Maintenance

// Compact performs database maintenance (VACUUM).
func (s *SQLiteStore) Compact() error {
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	return nil
}

// Search performs full-text search on entries.
func (s *SQLiteStore) Search(query string, limit int) ([]*models.Entry, error) {
	sqlQuery := `
		SELECT e.id, e.feed_id, e.guid, e.title, e.link, e.author, e.published_at, e.content, e.read, e.read_at, e.created_at
		FROM entries e
		INNER JOIN entries_fts fts ON e.rowid = fts.rowid
		WHERE entries_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := s.db.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search entries: %w", err)
	}
	defer rows.Close()

	var entries []*models.Entry
	for rows.Next() {
		entry, err := s.scanEntryFromRows(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Helper functions

func (s *SQLiteStore) scanFeed(row *sql.Row) (*models.Feed, error) {
	var feed models.Feed
	var lastFetched sql.NullTime
	if err := row.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Folder,
		&feed.ETag, &feed.LastModified, &lastFetched,
		&feed.LastError, &feed.ErrorCount, &feed.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found")
		}
		return nil, fmt.Errorf("scan feed: %w", err)
	}
	if lastFetched.Valid {
		feed.LastFetchedAt = &lastFetched.Time
	}
	return &feed, nil
}

func (s *SQLiteStore) scanFeedFromRows(rows *sql.Rows) (*models.Feed, error) {
	var feed models.Feed
	var lastFetched sql.NullTime
	if err := rows.Scan(
		&feed.ID, &feed.URL, &feed.Title, &feed.Folder,
		&feed.ETag, &feed.LastModified, &lastFetched,
		&feed.LastError, &feed.ErrorCount, &feed.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}
	if lastFetched.Valid {
		feed.LastFetchedAt = &lastFetched.Time
	}
	return &feed, nil
}

func (s *SQLiteStore) scanEntry(row *sql.Row) (*models.Entry, error) {
	var entry models.Entry
	var publishedAt, readAt sql.NullTime
	var readInt int
	if err := row.Scan(
		&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link,
		&entry.Author, &publishedAt, &entry.Content, &readInt, &readAt,
		&entry.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry not found")
		}
		return nil, fmt.Errorf("scan entry: %w", err)
	}
	if publishedAt.Valid {
		entry.PublishedAt = &publishedAt.Time
	}
	if readAt.Valid {
		entry.ReadAt = &readAt.Time
	}
	entry.Read = readInt == 1
	return &entry, nil
}

func (s *SQLiteStore) scanEntryFromRows(rows *sql.Rows) (*models.Entry, error) {
	var entry models.Entry
	var publishedAt, readAt sql.NullTime
	var readInt int
	if err := rows.Scan(
		&entry.ID, &entry.FeedID, &entry.GUID, &entry.Title, &entry.Link,
		&entry.Author, &publishedAt, &entry.Content, &readInt, &readAt,
		&entry.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan entry: %w", err)
	}
	if publishedAt.Valid {
		entry.PublishedAt = &publishedAt.Time
	}
	if readAt.Valid {
		entry.ReadAt = &readAt.Time
	}
	entry.Read = readInt == 1
	return &entry, nil
}

func timeToSQL(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// NewFeed creates a new feed with generated ID.
func NewFeed(url string) *models.Feed {
	return &models.Feed{
		ID:        uuid.New().String(),
		URL:       url,
		CreatedAt: time.Now(),
	}
}

// NewEntry creates a new entry with generated ID.
func NewEntry(feedID, guid, title string) *models.Entry {
	now := time.Now()
	return &models.Entry{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		GUID:      guid,
		Title:     &title,
		Read:      false,
		CreatedAt: now,
	}
}

// GetDefaultDBPath returns the default database path.
func GetDefaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "./digest.db"
		}
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	return filepath.Join(dataDir, "digest", "digest.db")
}

// Ensure SQLiteStore implements Store interface (but we need to add missing FeedIDs support)
var _ Store = (*SQLiteStore)(nil)

// SortFeeds is a helper to sort feeds for consistency with charm client
func SortFeeds(feeds []*models.Feed) {
	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].CreatedAt.After(feeds[j].CreatedAt)
	})
}

// ABOUTME: Vault sync integration for digest
// ABOUTME: Handles change queueing, syncing, and applying remote changes

package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"suitesync/vault"

	"github.com/google/uuid"
)

const (
	EntityFeed      = "feed"
	EntityReadState = "read_state"
)

// Syncer manages vault sync for digest data.
type Syncer struct {
	config *Config
	store  *vault.Store
	keys   vault.Keys
	client *vault.Client
	appDB  *sql.DB
}

// NewSyncer creates a new syncer from config.
func NewSyncer(cfg *Config, appDB *sql.DB) (*Syncer, error) {
	if cfg.DerivedKey == "" {
		return nil, errors.New("derived key not configured - run 'digest sync login' first")
	}

	seed, err := vault.ParseSeedPhrase(cfg.DerivedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid derived key: %w", err)
	}

	keys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	if err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}

	store, err := vault.OpenStore(cfg.VaultDB)
	if err != nil {
		return nil, fmt.Errorf("open vault store: %w", err)
	}

	client := vault.NewClient(vault.SyncConfig{
		BaseURL:   cfg.Server,
		DeviceID:  cfg.DeviceID,
		AuthToken: cfg.Token,
	})

	return &Syncer{
		config: cfg,
		store:  store,
		keys:   keys,
		client: client,
		appDB:  appDB,
	}, nil
}

// Close releases syncer resources.
func (s *Syncer) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// QueueFeedChange queues a change for a feed.
func (s *Syncer) QueueFeedChange(ctx context.Context, url, title, folder string, createdAt time.Time, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"url":        url,
			"title":      title,
			"folder":     folder,
			"created_at": createdAt.UTC().Unix(),
		}
	}

	// Use URL as entity ID (natural key)
	return s.queueChange(ctx, EntityFeed, url, op, payload)
}

// QueueReadStateChange queues a read state change.
func (s *Syncer) QueueReadStateChange(ctx context.Context, feedURL, guid string, read bool, readAt time.Time) error {
	entityID := feedURL + ":" + guid
	op := vault.OpUpsert

	payload := map[string]any{
		"feed_url": feedURL,
		"guid":     guid,
		"read":     read,
		"read_at":  readAt.UTC().Unix(),
	}

	return s.queueChange(ctx, EntityReadState, entityID, op, payload)
}

func (s *Syncer) queueChange(ctx context.Context, entity, entityID string, op vault.Op, payload map[string]any) error {
	change, err := vault.NewChange(entity, entityID, op, payload)
	if err != nil {
		return fmt.Errorf("create change: %w", err)
	}
	if op == vault.OpDelete {
		change.Deleted = true
	}

	plain, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("marshal change: %w", err)
	}

	aad := change.AAD(s.config.UserID, s.config.DeviceID)
	env, err := vault.Encrypt(s.keys.EncKey, plain, aad)
	if err != nil {
		return fmt.Errorf("encrypt change: %w", err)
	}

	if err := s.store.EnqueueEncryptedChange(ctx, change, s.config.UserID, s.config.DeviceID, env); err != nil {
		return fmt.Errorf("enqueue change: %w", err)
	}

	if s.config.AutoSync && s.canSync() {
		return s.Sync(ctx)
	}

	return nil
}

func (s *Syncer) canSync() bool {
	return s.config.Server != "" && s.config.Token != "" && s.config.UserID != ""
}

// Sync pushes local changes and pulls remote changes.
func (s *Syncer) Sync(ctx context.Context) error {
	return s.SyncWithEvents(ctx, nil)
}

// SyncWithEvents pushes local changes and pulls remote changes with progress callbacks.
func (s *Syncer) SyncWithEvents(ctx context.Context, events *vault.SyncEvents) error {
	if !s.canSync() {
		return errors.New("sync not configured - run 'digest sync login' first")
	}

	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange, events)
}

// applyChange applies a remote change to the local database.
func (s *Syncer) applyChange(ctx context.Context, c vault.Change) error {
	switch c.Entity {
	case EntityFeed:
		return s.applyFeedChange(ctx, c)
	case EntityReadState:
		return s.applyReadStateChange(ctx, c)
	default:
		return nil
	}
}

func (s *Syncer) applyFeedChange(ctx context.Context, c vault.Change) error {
	if c.Op == vault.OpDelete || c.Deleted {
		_, err := s.appDB.ExecContext(ctx, `DELETE FROM feeds WHERE url = ?`, c.EntityID)
		return err
	}

	var payload struct {
		URL       string `json:"url"`
		Title     string `json:"title"`
		Folder    string `json:"folder"`
		CreatedAt int64  `json:"created_at"`
	}
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal feed payload: %w", err)
	}

	createdAt := time.Unix(payload.CreatedAt, 0)

	// Check if feed exists
	var existingID string
	err := s.appDB.QueryRowContext(ctx, `SELECT id FROM feeds WHERE url = ?`, payload.URL).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		// Insert new feed - use UUID like the rest of digest
		id := uuid.New().String()
		_, err = s.appDB.ExecContext(ctx, `
			INSERT INTO feeds (id, url, title, folder, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, id, payload.URL, payload.Title, payload.Folder, createdAt)
		return err
	} else if err != nil {
		return fmt.Errorf("check feed exists: %w", err)
	}

	// Update existing feed
	_, err = s.appDB.ExecContext(ctx, `
		UPDATE feeds SET title = ?, folder = ? WHERE url = ?
	`, payload.Title, payload.Folder, payload.URL)
	return err
}

func (s *Syncer) applyReadStateChange(ctx context.Context, c vault.Change) error {
	var payload struct {
		FeedURL string `json:"feed_url"`
		GUID    string `json:"guid"`
		Read    bool   `json:"read"`
		ReadAt  int64  `json:"read_at"`
	}
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal read_state payload: %w", err)
	}

	// Find the entry by feed URL and GUID
	var entryID string
	var currentReadAt sql.NullTime
	err := s.appDB.QueryRowContext(ctx, `
		SELECT e.id, e.read_at FROM entries e
		JOIN feeds f ON e.feed_id = f.id
		WHERE f.url = ? AND e.guid = ?
	`, payload.FeedURL, payload.GUID).Scan(&entryID, &currentReadAt)

	if errors.Is(err, sql.ErrNoRows) {
		// Entry doesn't exist locally, skip
		return nil
	} else if err != nil {
		return fmt.Errorf("find entry: %w", err)
	}

	// Last-writer-wins: only apply if incoming is newer
	incomingReadAt := time.Unix(payload.ReadAt, 0)
	if currentReadAt.Valid && currentReadAt.Time.After(incomingReadAt) {
		return nil // Local is newer, skip
	}

	// Apply the change
	if payload.Read {
		_, err = s.appDB.ExecContext(ctx, `
			UPDATE entries SET read = TRUE, read_at = ? WHERE id = ?
		`, incomingReadAt, entryID)
	} else {
		_, err = s.appDB.ExecContext(ctx, `
			UPDATE entries SET read = FALSE, read_at = NULL WHERE id = ?
		`, entryID)
	}
	return err
}

// PendingCount returns the number of changes waiting to be synced.
func (s *Syncer) PendingCount(ctx context.Context) (int, error) {
	batch, err := s.store.DequeueBatch(ctx, 1000)
	if err != nil {
		return 0, err
	}
	return len(batch), nil
}

// PendingItem represents a change waiting to be synced.
type PendingItem struct {
	ChangeID string
	Entity   string
	TS       time.Time
}

// PendingChanges returns details of changes waiting to be synced.
func (s *Syncer) PendingChanges(ctx context.Context) ([]PendingItem, error) {
	batch, err := s.store.DequeueBatch(ctx, 100)
	if err != nil {
		return nil, err
	}

	items := make([]PendingItem, len(batch))
	for i, b := range batch {
		items[i] = PendingItem{
			ChangeID: b.ChangeID,
			Entity:   b.Entity,
			TS:       time.Unix(b.TS, 0),
		}
	}
	return items, nil
}

// LastSyncedSeq returns the last pulled sequence number.
func (s *Syncer) LastSyncedSeq(ctx context.Context) (string, error) {
	return s.store.GetState(ctx, "last_pulled_seq", "0")
}

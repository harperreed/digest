// ABOUTME: Entry model representing a single feed entry with read/unread state
// ABOUTME: Provides methods to mark entries as read or unread with timestamps

package models

import (
	"time"

	"github.com/google/uuid"
)

// Entry represents a single entry (article/item) in an RSS/Atom feed
type Entry struct {
	ID          string
	FeedID      string
	GUID        string
	Title       *string
	Link        *string
	Author      *string
	PublishedAt *time.Time
	Content     *string
	Read        bool
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// NewEntry creates a new Entry with the given feedID, guid, and title
// Sets ID to a new UUID, CreatedAt to current time, and Read to false
func NewEntry(feedID, guid, title string) *Entry {
	now := time.Now()
	return &Entry{
		ID:        uuid.New().String(),
		FeedID:    feedID,
		GUID:      guid,
		Title:     &title,
		Read:      false,
		ReadAt:    nil,
		CreatedAt: now,
	}
}

// MarkRead marks the entry as read and sets ReadAt to the current time
func (e *Entry) MarkRead() {
	now := time.Now()
	e.Read = true
	e.ReadAt = &now
}

// MarkUnread marks the entry as unread and clears the ReadAt timestamp
func (e *Entry) MarkUnread() {
	e.Read = false
	e.ReadAt = nil
}

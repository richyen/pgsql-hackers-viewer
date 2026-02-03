package models

import (
	"time"
)

// Thread represents a mailing list thread
type Thread struct {
	ID               string     `json:"id"`
	Subject          string     `json:"subject"`
	FirstMessageID   string     `json:"first_message_id"`
	FirstAuthor      string     `json:"first_author"`
	FirstAuthorEmail string     `json:"first_author_email"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastMessageAt    *time.Time `json:"last_message_at,omitempty"`
	MessageCount     int        `json:"message_count"`
	UniqueAuthors    int        `json:"unique_authors"`
	Status           string     `json:"status"` // in-progress, has-patch, stalled-patch, discussion, stalled, abandoned
}

// Message represents an email message in a thread
type Message struct {
	ID           string    `json:"id"`
	ThreadID     string    `json:"thread_id"`
	MessageID    string    `json:"message_id"`
	InReplyTo    string    `json:"in_reply_to,omitempty"`
	RefersTo     string    `json:"refers_to,omitempty"`
	Subject      string    `json:"subject"`
	Author       string    `json:"author"`
	AuthorEmail  string    `json:"author_email"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"created_at"`
	HasPatch     bool      `json:"has_patch"`
	PatchStatus  string    `json:"patch_status,omitempty"` // empty, "proposed", "accepted", "committed", "rejected"
	CommitFestID string    `json:"commitfest_id,omitempty"`
}

// ThreadActivity tracks activity metrics for a thread
type ThreadActivity struct {
	ID                   string    `json:"id"`
	ThreadID             string    `json:"thread_id"`
	MessageCount         int       `json:"message_count"`
	UniqueAuthors        int       `json:"unique_authors"`
	HasPatch             bool      `json:"has_patch"`
	HasReview            bool      `json:"has_review"`
	IsResolved           bool      `json:"is_resolved"`
	DaysSinceLastMessage int       `json:"days_since_last_message"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PatchInfo contains information about patches in the thread
type PatchInfo struct {
	MessageID    string    `json:"message_id"`
	PatchNumber  int       `json:"patch_number"`
	Status       string    `json:"status"`
	CommitFestID string    `json:"commitfest_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SyncProgress tracks the progress of mailing list synchronization
type SyncProgress struct {
	MonthsSynced      int        `json:"months_synced"`
	TotalMonths       int        `json:"total_months"`
	LatestMessageDate *time.Time `json:"latest_message_date,omitempty"`
	CurrentMonth      string     `json:"current_month"`
	IsSyncing         bool       `json:"is_syncing"`
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty"`
}

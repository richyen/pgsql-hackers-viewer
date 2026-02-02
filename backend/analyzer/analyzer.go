package analyzer

import (
	"database/sql"
	"log"
	"strings"
	"time"
)

type ThreadAnalyzer struct {
	db *sql.DB
}

func NewThreadAnalyzer(db *sql.DB) *ThreadAnalyzer {
	return &ThreadAnalyzer{db: db}
}

// ClassifyThread determines the status of a thread based on activity metrics
func (ta *ThreadAnalyzer) ClassifyThread(threadID string) (string, error) {
	var lastMessageAt sql.NullTime
	var messageCount int
	var uniqueAuthors int

	err := ta.db.QueryRow(`
		SELECT 
			COALESCE(last_message_at, created_at),
			message_count,
			unique_authors
		FROM threads
		WHERE id = $1
	`, threadID).Scan(&lastMessageAt, &messageCount, &uniqueAuthors)

	if err != nil {
		log.Printf("Error querying thread: %v", err)
		return "unknown", err
	}

	// Check for patch-related keywords
	hasPatch, hasReview := ta.checkForPatchKeywords(threadID)

	// Calculate days since last message
	daysSince := time.Since(lastMessageAt.Time).Hours() / 24

	// Classification logic
	if hasPatch && (hasReview || messageCount > 3) {
		return "in-progress", nil
	}

	if daysSince > 30 && messageCount < 5 {
		return "abandoned", nil
	}

	if daysSince > 7 {
		return "stalled", nil
	}

	return "discussion", nil
}

func (ta *ThreadAnalyzer) checkForPatchKeywords(threadID string) (bool, bool) {
	rows, err := ta.db.Query(`
		SELECT body FROM messages WHERE thread_id = $1
	`, threadID)
	if err != nil {
		return false, false
	}
	defer rows.Close()

	hasPatch := false
	hasReview := false
	patchKeywords := []string{"patch", "diff", "commit", "PR"}
	reviewKeywords := []string{"review", "LGTM", "approved", "looks good", "ACK"}

	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			continue
		}

		bodyLower := strings.ToLower(body)
		for _, keyword := range patchKeywords {
			if strings.Contains(bodyLower, strings.ToLower(keyword)) {
				hasPatch = true
				break
			}
		}
		for _, keyword := range reviewKeywords {
			if strings.Contains(bodyLower, strings.ToLower(keyword)) {
				hasReview = true
				break
			}
		}
	}

	return hasPatch, hasReview
}

// UpdateThreadActivity updates the activity metrics for a thread
func (ta *ThreadAnalyzer) UpdateThreadActivity(threadID string) error {
	var messageCount int
	var uniqueAuthors int
	var lastMessageAt time.Time

	// Get message count and unique authors
	err := ta.db.QueryRow(`
		SELECT 
			COUNT(*),
			COUNT(DISTINCT author_email),
			MAX(created_at)
		FROM messages
		WHERE thread_id = $1
	`, threadID).Scan(&messageCount, &uniqueAuthors, &lastMessageAt)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Check for patch and review keywords
	hasPatch, hasReview := ta.checkForPatchKeywords(threadID)

	// Calculate days since last message
	daysSince := int(time.Since(lastMessageAt).Hours() / 24)

	// Update thread record
	_, err = ta.db.Exec(`
		UPDATE threads
		SET 
			message_count = $1,
			unique_authors = $2,
			last_message_at = $3,
			updated_at = NOW()
		WHERE id = $4
	`, messageCount, uniqueAuthors, lastMessageAt, threadID)

	if err != nil {
		return err
	}

	// Upsert activity record
	_, err = ta.db.Exec(`
		INSERT INTO thread_activities 
			(id, thread_id, message_count, unique_authors, has_patch, has_review, days_since_last_message, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (thread_id) DO UPDATE SET
			message_count = $3,
			unique_authors = $4,
			has_patch = $5,
			has_review = $6,
			days_since_last_message = $7,
			updated_at = NOW()
	`, threadID, threadID, messageCount, uniqueAuthors, hasPatch, hasReview, daysSince)

	return err
}

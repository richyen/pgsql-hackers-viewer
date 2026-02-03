package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pgsql-analyzer/backend/analyzer"
	"github.com/pgsql-analyzer/backend/config"
	"github.com/pgsql-analyzer/backend/fetcher"
	"github.com/pgsql-analyzer/backend/models"
	"github.com/pgsql-analyzer/backend/parser"
)

func RegisterRoutes(router *mux.Router, db *sql.DB, cfg *config.Config) {
	// Health check
	router.HandleFunc("/api/health", healthHandler).Methods("GET")

	// Thread endpoints
	router.HandleFunc("/api/threads", getThreadsHandler(db)).Methods("GET")
	router.HandleFunc("/api/threads/{id}", getThreadHandler(db)).Methods("GET")
	router.HandleFunc("/api/threads/{id}/messages", getThreadMessagesHandler(db)).Methods("GET")

	// Message endpoints
	router.HandleFunc("/api/messages/{id}", getMessageHandler(db)).Methods("GET")

	// Stats endpoint
	router.HandleFunc("/api/stats", getStatsHandler(db)).Methods("GET")

	// Sync endpoints
	router.HandleFunc("/api/sync/progress", getSyncProgressHandler).Methods("GET")
	router.HandleFunc("/api/sync/mbox", uploadMboxHandler(db, cfg)).Methods("POST")
	router.HandleFunc("/api/sync/mbox/all", syncMboxHandler(db, cfg)).Methods("POST")

	// Reset: clear all threads/messages so next sync re-downloads from scratch
	router.HandleFunc("/api/reset", resetHandler(db)).Methods("POST")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func resetHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Truncate in FK order: activities and messages reference threads
		_, err := db.Exec(`
			TRUNCATE thread_activities CASCADE;
			TRUNCATE messages CASCADE;
			TRUNCATE threads CASCADE;
		`)
		if err != nil {
			log.Printf("Error resetting database: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to reset database"})
			return
		}
		log.Println("Database reset: threads, messages, and thread_activities cleared")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "Database cleared. Run Sync mbox files to re-download and re-import.",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func getThreadsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := r.URL.Query().Get("status")
		limit := r.URL.Query().Get("limit")
		if limit == "" {
			limit = "50"
		}

		query := `
			SELECT 
				id, subject, first_message_id, first_author, first_author_email,
				created_at, updated_at, last_message_at, message_count, unique_authors, status
			FROM threads
			WHERE 1=1
		`

		args := []interface{}{}
		argCount := 1

		if status != "" {
			query += " AND status = $" + fmt.Sprintf("%d", argCount)
			args = append(args, status)
			argCount++
		}

		query += " ORDER BY last_message_at DESC LIMIT $" + fmt.Sprintf("%d", argCount)
		args = append(args, limit)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying threads: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch threads"})
			return
		}
		defer rows.Close()

		var threads []*models.Thread
		for rows.Next() {
			thread := &models.Thread{}
			var lastMsgAt sql.NullTime
			if err := rows.Scan(
				&thread.ID, &thread.Subject, &thread.FirstMessageID, &thread.FirstAuthor,
				&thread.FirstAuthorEmail, &thread.CreatedAt, &thread.UpdatedAt, &lastMsgAt,
				&thread.MessageCount, &thread.UniqueAuthors, &thread.Status,
			); err != nil {
				log.Printf("Error scanning thread: %v", err)
				continue
			}
			if lastMsgAt.Valid {
				thread.LastMessageAt = &lastMsgAt.Time
			}
			threads = append(threads, thread)
		}

		json.NewEncoder(w).Encode(threads)
	}
}

func getThreadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		threadID := vars["id"]

		thread := &models.Thread{}
		var lastMsgAt sql.NullTime
		err := db.QueryRow(`
			SELECT 
				id, subject, first_message_id, first_author, first_author_email,
				created_at, updated_at, last_message_at, message_count, unique_authors, status
			FROM threads
			WHERE id = $1
		`, threadID).Scan(
			&thread.ID, &thread.Subject, &thread.FirstMessageID, &thread.FirstAuthor,
			&thread.FirstAuthorEmail, &thread.CreatedAt, &thread.UpdatedAt, &lastMsgAt,
			&thread.MessageCount, &thread.UniqueAuthors, &thread.Status,
		)
		if err == nil && lastMsgAt.Valid {
			thread.LastMessageAt = &lastMsgAt.Time
		}

		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "Thread not found"})
				return
			}
			log.Printf("Error querying thread: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch thread"})
			return
		}

		json.NewEncoder(w).Encode(thread)
	}
}

func getThreadMessagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		threadID := vars["id"]

		rows, err := db.Query(`
			SELECT id, thread_id, message_id, subject, author, author_email, body, created_at,
			       has_patch, patch_status, commitfest_id
			FROM messages
			WHERE thread_id = $1
			ORDER BY created_at ASC
		`, threadID)

		if err != nil {
			log.Printf("Error querying messages: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch messages"})
			return
		}
		defer rows.Close()

		var messages []*models.Message
		for rows.Next() {
			msg := &models.Message{}
			if err := rows.Scan(
				&msg.ID, &msg.ThreadID, &msg.MessageID, &msg.Subject,
				&msg.Author, &msg.AuthorEmail, &msg.Body, &msg.CreatedAt,
				&msg.HasPatch, &msg.PatchStatus, &msg.CommitFestID,
			); err != nil {
				log.Printf("Error scanning message: %v", err)
				continue
			}
			messages = append(messages, msg)
		}

		json.NewEncoder(w).Encode(messages)
	}
}

func getMessageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		messageID := vars["id"]

		msg := &models.Message{}
		err := db.QueryRow(`
			SELECT id, thread_id, message_id, subject, author, author_email, body, created_at,
			       has_patch, patch_status, commitfest_id
			FROM messages
			WHERE id = $1
		`, messageID).Scan(
			&msg.ID, &msg.ThreadID, &msg.MessageID, &msg.Subject,
			&msg.Author, &msg.AuthorEmail, &msg.Body, &msg.CreatedAt,
			&msg.HasPatch, &msg.PatchStatus, &msg.CommitFestID,
		)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"})
			return
		} else if err != nil {
			log.Printf("Error querying message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch message"})
			return
		}

		json.NewEncoder(w).Encode(msg)
	}
}

func getStatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		stats := map[string]interface{}{}

		// Total threads
		var totalThreads int
		db.QueryRow("SELECT COUNT(*) FROM threads").Scan(&totalThreads)
		stats["total_threads"] = totalThreads

		// Threads by status
		statuses := []string{"in-progress", "discussion", "stalled", "abandoned"}
		statusCounts := make(map[string]int)
		for _, status := range statuses {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM threads WHERE status = $1", status).Scan(&count)
			statusCounts[status] = count
		}
		stats["by_status"] = statusCounts

		// Total messages
		var totalMessages int
		db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)
		stats["total_messages"] = totalMessages

		// Last sync time
		var lastSync sql.NullTime
		db.QueryRow(`
			SELECT MAX(updated_at) FROM threads
		`).Scan(&lastSync)
		if lastSync.Valid {
			stats["last_sync"] = lastSync.Time
		}

		json.NewEncoder(w).Encode(stats)
	}
}

func getSyncProgressHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	progress := GlobalSyncState.Get()
	json.NewEncoder(w).Encode(progress)
}

func groupByThread(messages []*models.Message) map[string][]*models.Message {
	threads := make(map[string][]*models.Message)
	for _, msg := range messages {
		threads[msg.Subject] = append(threads[msg.Subject], msg)
	}
	return threads
}

func uploadMboxHandler(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse multipart form
		err := r.ParseMultipartForm(100 << 20) // 100MB max
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse upload"})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing file"})
			return
		}
		defer file.Close()

		// Read file content
		buf := make([]byte, header.Size)
		_, err = file.Read(buf)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read file"})
			return
		}

		// Save mbox file
		mboxParser := parser.NewMboxParser(cfg.DataDir)
		filePath, err := mboxParser.SaveMboxFile(header.Filename, buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
			return
		}

		// Parse and store messages
		go processMboxFile(db, cfg, filePath)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "Mbox file uploaded and queued for processing",
			"filename":  header.Filename,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func syncMboxHandler(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		go performMboxSync(db, cfg)

		json.NewEncoder(w).Encode(map[string]string{
			"status":    "Mbox sync started",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func processMboxFile(db *sql.DB, cfg *config.Config, filePath string) {
	log.Printf("Processing mbox file: %s", filePath)

	mboxParser := parser.NewMboxParser(cfg.DataDir)
	messages, err := mboxParser.ParseMboxFile(filePath)
	if err != nil {
		log.Printf("Error parsing mbox file: %v", err)
		return
	}

	storeMessagesInDB(db, messages)
	log.Printf("Completed processing %d messages from %s", len(messages), filePath)
}

func performMboxSync(db *sql.DB, cfg *config.Config) {
	log.Println("Starting mbox sync from PostgreSQL.org archives...")
	GlobalSyncState.SetSyncing(true)
	defer GlobalSyncState.SetSyncing(false)

	// Catch any panics and log them
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in performMboxSync: %v", r)
		}
	}()

	// Determine range: from last recorded message (or 365 days ago) to present
	const initialSyncDays = 365
	var lastMessageAt sql.NullTime
	err := db.QueryRow("SELECT MAX(created_at) FROM messages").Scan(&lastMessageAt)
	if err != nil {
		log.Printf("Error getting last message date: %v", err)
		return
	}

	now := time.Now()
	var start time.Time
	if lastMessageAt.Valid && !lastMessageAt.Time.IsZero() {
		// Incremental: sync from the month after last message through current month
		start = time.Date(lastMessageAt.Time.Year(), lastMessageAt.Time.Month(), 1, 0, 0, 0, 0, time.UTC)
		// Include the month we have so we can re-download and catch any late-arriving messages
		start = start.AddDate(0, 0, 0)
	} else {
		// Initial: last 365 days
		start = now.AddDate(0, 0, -initialSyncDays)
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	months := monthsBetween(start, end)
	if len(months) == 0 {
		log.Println("No new months to sync")
		return
	}

	totalMonths := len(months)
	log.Printf("Syncing %d month(s) from %s to %s", totalMonths, start.Format("2006-01"), end.Format("2006-01"))
	GlobalSyncState.Update(0, totalMonths, "")

	// Convert yearMonth to fetcher.MonthDownload
	downloads := make([]fetcher.MonthDownload, len(months))
	for i, ym := range months {
		downloads[i] = fetcher.MonthDownload{Year: ym.year, Month: ym.month}
	}

	// Download all months in parallel (3-4 workers)
	const concurrentDownloads = 4
	log.Printf("Starting parallel download with %d workers", concurrentDownloads)

	// In dev mode, skip download if file exists; in production, always download fresh
	skipIfExists := cfg.ENV == "development"
	if skipIfExists {
		log.Println("Dev mode: Using cached mbox files if available")
	} else {
		log.Println("Production mode: Downloading fresh mbox files")
	}

	downloadResults := fetcher.DownloadMonthsConcurrent(cfg.DataDir, cfg.ArchiveUsername, cfg.ArchivePassword, downloads, concurrentDownloads, skipIfExists)

	// Process downloads and parse mbox files
	log.Printf("Received %d download results", len(downloadResults))
	mboxParser := parser.NewMboxParser(cfg.DataDir)
	var totalStored int
	processedCount := 0

	for _, result := range downloadResults {
		processedCount++
		currentMonth := fmt.Sprintf("%04d-%02d", result.Year, result.Month)
		GlobalSyncState.Update(processedCount, totalMonths, currentMonth)

		if result.Error != nil {
			log.Printf("Skip month %04d-%02d: %v", result.Year, result.Month, result.Error)
			continue
		}

		log.Printf("Processing %04d-%02d from %s (took %v)", result.Year, result.Month, result.Path, result.Duration)

		messages, err := mboxParser.ParseMboxFile(result.Path)
		if err != nil {
			log.Printf("Error parsing %s: %v", result.Path, err)
			continue
		}
		log.Printf("Parsed %d messages from %s", len(messages), result.Path)
		if len(messages) == 0 {
			log.Printf("No messages in %s, skipping", result.Path)
			continue
		}
		log.Printf("Storing %d messages in database", len(messages))
		n := storeMessagesInDB(db, messages)
		totalStored += n
		log.Printf("Stored %d new messages (total so far: %d)", n, totalStored)

		// In production mode, cleanup (delete) mbox file after successful ingestion
		if cfg.CleanupMboxFiles {
			if err := os.Remove(result.Path); err != nil {
				log.Printf("Warning: Failed to cleanup mbox file %s: %v", result.Path, err)
			} else {
				log.Printf("Cleaned up mbox file: %s", result.Path)
			}
		}

		// Update latest message date
		if len(messages) > 0 {
			latestMsg := messages[len(messages)-1]
			GlobalSyncState.SetLatestMessageDate(latestMsg.CreatedAt)
		}
	}

	GlobalSyncState.Update(totalMonths, totalMonths, "")
	log.Printf("Mbox sync completed: %d new messages stored", totalStored)
}

// yearMonth is a (year, month) pair for sync range.
type yearMonth struct{ year, month int }

// monthsBetween returns (year, month) from start through end inclusive, month-by-month.
func monthsBetween(start, end time.Time) []yearMonth {
	var out []yearMonth
	for y, m := start.Year(), int(start.Month()); !(y > end.Year() || (y == end.Year() && m > int(end.Month()))); {
		out = append(out, yearMonth{year: y, month: m})
		m++
		if m > 12 {
			m = 1
			y++
		}
	}
	return out
}

// sanitizeUTF8 removes invalid UTF-8 sequences and replaces them with replacement character
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Build a new string with only valid UTF-8
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte, skip it
			i++
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

// storeMessagesInDB groups messages by subject, merges into existing threads by subject when present,
// and returns the number of messages newly inserted (excluding conflicts).
func storeMessagesInDB(db *sql.DB, messages []*models.Message) int {
	threads := groupByThread(messages)
	threadAnalyzer := analyzer.NewThreadAnalyzer(db)
	var inserted int
	for subject, msgs := range threads {
		firstMsg := msgs[0]
		var threadID string
		// Reuse existing thread with same subject (for incremental sync)
		err := db.QueryRow(`
			SELECT id FROM threads WHERE subject = $1 ORDER BY created_at ASC LIMIT 1
		`, subject).Scan(&threadID)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Error looking up thread by subject: %v", err)
				continue
			}
			threadID = uuid.New().String()
			// Sanitize thread fields
			sanitizedSubject := sanitizeUTF8(subject)
			sanitizedMessageID := sanitizeUTF8(firstMsg.MessageID)
			sanitizedAuthor := sanitizeUTF8(firstMsg.Author)
			sanitizedAuthorEmail := sanitizeUTF8(firstMsg.AuthorEmail)

			_, err = db.Exec(`
				INSERT INTO threads (id, subject, first_message_id, first_author, first_author_email, created_at, last_message_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (id) DO NOTHING
			`, threadID, sanitizedSubject, sanitizedMessageID, sanitizedAuthor, sanitizedAuthorEmail, firstMsg.CreatedAt, firstMsg.CreatedAt)
			if err != nil {
				log.Printf("Error inserting thread: %v", err)
				continue
			}
		}

		for _, msg := range msgs {
			msg.ID = uuid.New().String()
			msg.ThreadID = threadID

			// Sanitize all text fields to ensure valid UTF-8
			msg.Subject = sanitizeUTF8(msg.Subject)
			msg.Author = sanitizeUTF8(msg.Author)
			msg.AuthorEmail = sanitizeUTF8(msg.AuthorEmail)
			msg.Body = sanitizeUTF8(msg.Body)
			msg.MessageID = sanitizeUTF8(msg.MessageID)

			result, err := db.Exec(`
				INSERT INTO messages (id, thread_id, message_id, subject, author, author_email, body, created_at, has_patch, patch_status, commitfest_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
				ON CONFLICT (message_id) DO UPDATE SET thread_id = EXCLUDED.thread_id, has_patch = EXCLUDED.has_patch, patch_status = EXCLUDED.patch_status, commitfest_id = EXCLUDED.commitfest_id
			`, msg.ID, msg.ThreadID, msg.MessageID, msg.Subject, msg.Author, msg.AuthorEmail, msg.Body, msg.CreatedAt, msg.HasPatch, msg.PatchStatus, msg.CommitFestID)
			if err != nil {
				log.Printf("Error inserting message: %v", err)
				continue
			}
			rows, _ := result.RowsAffected()
			inserted += int(rows)
		}

		if err := threadAnalyzer.UpdateThreadActivity(threadID); err != nil {
			log.Printf("Error updating thread activity: %v", err)
		}
		status, err := threadAnalyzer.ClassifyThread(threadID)
		if err == nil {
			db.Exec("UPDATE threads SET status = $1 WHERE id = $2", status, threadID)
		}
	}

	// Refresh all thread stats from messages so every thread has correct counts
	// (fixes duplicates and any thread that lost messages to the canonical one)
	_, _ = db.Exec(`
		UPDATE threads t SET
			message_count = (SELECT COUNT(*) FROM messages m WHERE m.thread_id = t.id),
			unique_authors = (SELECT COUNT(DISTINCT author_email) FROM messages m WHERE m.thread_id = t.id),
			last_message_at = (SELECT MAX(created_at) FROM messages m WHERE m.thread_id = t.id),
			updated_at = NOW()
	`)

	// Delete threads with no messages (orphaned threads)
	_, _ = db.Exec(`DELETE FROM threads WHERE message_count = 0`)

	// Reclassify all threads so status (in-progress, stalled, etc.) matches updated counts
	rows, err := db.Query("SELECT id FROM threads")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			if status, err := threadAnalyzer.ClassifyThread(id); err == nil {
				db.Exec("UPDATE threads SET status = $1 WHERE id = $2", status, id)
			}
		}
	}
	return inserted
}

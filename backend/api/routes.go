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
		search := r.URL.Query().Get("search")
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")
		if limit == "" {
			limit = "50"
		}
		if offset == "" {
			offset = "0"
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

		if search != "" {
			// Search by message_id first (exact match), then by subject (substring match)
			// Message-ID exact match takes priority
			query += " AND (id IN (SELECT DISTINCT thread_id FROM messages WHERE message_id = $" + fmt.Sprintf("%d", argCount) + ") OR LOWER(subject) LIKE LOWER($" + fmt.Sprintf("%d", argCount+1) + "))"
			args = append(args, search)
			args = append(args, "%"+search+"%")
			argCount += 2
		}

		query += " ORDER BY last_message_at DESC LIMIT $" + fmt.Sprintf("%d", argCount)
		args = append(args, limit)
		argCount++

		query += " OFFSET $" + fmt.Sprintf("%d", argCount)
		args = append(args, offset)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying threads: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch threads"})
			return
		}
		defer rows.Close()

		threads := make([]*models.Thread, 0)
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
		statuses := []string{"in-progress", "has-patch", "stalled-patch", "discussion", "stalled", "abandoned"}
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
	// RFC 5256 THREAD=REFERENCES implementation
	// Build a map of message-id to message for quick lookups
	messageMap := make(map[string]*models.Message)
	for _, msg := range messages {
		messageMap[msg.MessageID] = msg
	}

	// messageToRoot maps each message-id to its thread root
	messageToRoot := make(map[string]string)

	// First, build the thread structure using References and In-Reply-To
	for _, msg := range messages {
		root := findThreadRootRFC5256(msg, messageMap, messageToRoot)
		messageToRoot[msg.MessageID] = root
	}

	// Group messages by their root
	threadMap := make(map[string][]*models.Message)
	for _, msg := range messages {
		root := messageToRoot[msg.MessageID]
		threadMap[root] = append(threadMap[root], msg)
	}

	return threadMap
}

// findThreadRootRFC5256 implements RFC 5256 threading algorithm
func findThreadRootRFC5256(msg *models.Message, messageMap map[string]*models.Message, messageToRoot map[string]string) string {
	// Extract all references from the References header
	refs := parseReferences(msg.RefersTo)

	// Add In-Reply-To to the reference chain if it exists
	if msg.InReplyTo != "" {
		// In-Reply-To should be the last reference
		refs = append(refs, msg.InReplyTo)
	}

	// If no references, this message is a root
	if len(refs) == 0 {
		return msg.MessageID
	}

	// The first reference in the chain is the real root (or the oldest missing message).
	// Even if that message doesn't exist in our dataset, we should use it as the thread root
	// to ensure all messages referencing it get grouped together.
	// This is important for handling threads with missing intermediate messages.

	// Traverse the reference chain to find the root (first/oldest reference)
	currentRefID := ""
	for _, refID := range refs {
		// Clean up the reference ID
		refID = strings.Trim(strings.TrimSpace(refID), "<>")
		if refID == "" {
			continue
		}
		currentRefID = refID
		break // The first valid reference is our candidate root
	}

	if currentRefID == "" {
		// No valid references found, this message is a root
		return msg.MessageID
	}

	// Check if we already know the root for the first reference
	if root, exists := messageToRoot[currentRefID]; exists {
		return root
	}

	// Check if the first reference exists in our message set
	if refMsg, exists := messageMap[currentRefID]; exists {
		// Recursively find the root of this reference
		root := findThreadRootRFC5256(refMsg, messageMap, messageToRoot)
		messageToRoot[currentRefID] = root
		return root
	}

	// First reference doesn't exist in our dataset, but we use it as the thread root anyway
	// This ensures all messages that reference it get grouped together,
	// even if the message itself is missing from our archives
	return currentRefID
}

// parseReferences extracts individual message IDs from a References header
// References can contain multiple message IDs separated by whitespace
func parseReferences(references string) []string {
	if references == "" {
		return nil
	}

	var refs []string
	// References can be space-separated or on multiple lines
	// Message IDs are typically in angle brackets: <id@domain>

	// Find all message IDs in angle brackets
	inBracket := false
	var currentRef strings.Builder

	for _, ch := range references {
		if ch == '<' {
			inBracket = true
			currentRef.Reset()
		} else if ch == '>' && inBracket {
			inBracket = false
			if currentRef.Len() > 0 {
				refs = append(refs, currentRef.String())
			}
		} else if inBracket {
			currentRef.WriteRune(ch)
		}
	}

	// If no angle brackets found, try splitting by whitespace
	if len(refs) == 0 && references != "" {
		parts := strings.Fields(references)
		for _, part := range parts {
			part = strings.Trim(part, "<>")
			if part != "" && strings.Contains(part, "@") {
				refs = append(refs, part)
			}
		}
	}

	return refs
}

// sortMessagesByTime sorts messages by creation time (earliest first)
func sortMessagesByTime(msgs []*models.Message) {
	for i := 0; i < len(msgs)-1; i++ {
		for j := i + 1; j < len(msgs); j++ {
			if msgs[i].CreatedAt.After(msgs[j].CreatedAt) {
				msgs[i], msgs[j] = msgs[j], msgs[i]
			}
		}
	}
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

// storeMessagesInDB groups messages by thread using In-Reply-To/References headers,
// merges into existing threads by message-id when present, and returns the number of messages newly inserted.
func storeMessagesInDB(db *sql.DB, messages []*models.Message) int {
	threads := groupByThread(messages)
	threadAnalyzer := analyzer.NewThreadAnalyzer(db)
	var inserted int

	for rootMessageID, msgs := range threads {
		if len(msgs) == 0 {
			continue
		}

		// Sort messages by creation time to ensure root message is first
		sortMessagesByTime(msgs)
		firstMsg := msgs[0]

		var threadID string

		// Try to find existing thread by the root message-id
		err := db.QueryRow(`
			SELECT id FROM threads WHERE first_message_id = $1 LIMIT 1
		`, rootMessageID).Scan(&threadID)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error looking up thread by message-id: %v", err)
			continue
		}

		// If thread doesn't exist by root message-id, check if any message in this thread
		// already exists and get its thread_id (handles missing intermediate messages)
		if err == sql.ErrNoRows {
			for _, msg := range msgs {
				err = db.QueryRow(`
					SELECT thread_id FROM messages WHERE message_id = $1 LIMIT 1
				`, msg.MessageID).Scan(&threadID)
				if err == nil {
					// Found an existing message, use its thread
					break
				}
			}
		}

		// If still no thread found, create a new one
		if threadID == "" {
			threadID = uuid.New().String()
			sanitizedSubject := sanitizeUTF8(firstMsg.Subject)
			sanitizedMessageID := sanitizeUTF8(rootMessageID)
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
			msg.InReplyTo = sanitizeUTF8(msg.InReplyTo)
			msg.RefersTo = sanitizeUTF8(msg.RefersTo)

			result, err := db.Exec(`
				INSERT INTO messages (id, thread_id, message_id, in_reply_to, refers_to, subject, author, author_email, body, created_at, has_patch, patch_status, commitfest_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				ON CONFLICT (message_id) DO UPDATE SET thread_id = EXCLUDED.thread_id, in_reply_to = EXCLUDED.in_reply_to, refers_to = EXCLUDED.refers_to, has_patch = EXCLUDED.has_patch, patch_status = EXCLUDED.patch_status, commitfest_id = EXCLUDED.commitfest_id
			`, msg.ID, msg.ThreadID, msg.MessageID, msg.InReplyTo, msg.RefersTo, msg.Subject, msg.Author, msg.AuthorEmail, msg.Body, msg.CreatedAt, msg.HasPatch, msg.PatchStatus, msg.CommitFestID)
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

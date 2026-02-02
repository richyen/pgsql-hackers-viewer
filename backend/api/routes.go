package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pgsql-analyzer/backend/analyzer"
	"github.com/pgsql-analyzer/backend/config"
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

	// Stats endpoint
	router.HandleFunc("/api/stats", getStatsHandler(db)).Methods("GET")

	// Sync endpoints
	router.HandleFunc("/api/sync", syncHandler(db, cfg)).Methods("POST")
	router.HandleFunc("/api/sync/mbox", uploadMboxHandler(db, cfg)).Methods("POST")
	router.HandleFunc("/api/sync/mbox/all", syncMboxHandler(db, cfg)).Methods("POST")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
			if err := rows.Scan(
				&thread.ID, &thread.Subject, &thread.FirstMessageID, &thread.FirstAuthor,
				&thread.FirstAuthorEmail, &thread.CreatedAt, &thread.UpdatedAt, &thread.LastMessageAt,
				&thread.MessageCount, &thread.UniqueAuthors, &thread.Status,
			); err != nil {
				log.Printf("Error scanning thread: %v", err)
				continue
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
		err := db.QueryRow(`
			SELECT 
				id, subject, first_message_id, first_author, first_author_email,
				created_at, updated_at, last_message_at, message_count, unique_authors, status
			FROM threads
			WHERE id = $1
		`, threadID).Scan(
			&thread.ID, &thread.Subject, &thread.FirstMessageID, &thread.FirstAuthor,
			&thread.FirstAuthorEmail, &thread.CreatedAt, &thread.UpdatedAt, &thread.LastMessageAt,
			&thread.MessageCount, &thread.UniqueAuthors, &thread.Status,
		)

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
			SELECT id, thread_id, message_id, subject, author, author_email, created_at
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
				&msg.Author, &msg.AuthorEmail, &msg.CreatedAt,
			); err != nil {
				log.Printf("Error scanning message: %v", err)
				continue
			}
			messages = append(messages, msg)
		}

		json.NewEncoder(w).Encode(messages)
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

func syncHandler(db *sql.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// For now, just return success - in production, this would async queue
		go performSync(db, cfg)

		json.NewEncoder(w).Encode(map[string]string{
			"status": "Sync started",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func performSync(db *sql.DB, cfg *config.Config) {
	log.Println("Starting mail sync...")

	// Create parser
	mp := parser.NewMailParser(cfg.MailIMAPHost, cfg.MailIMAPPort, cfg.MailUsername, cfg.MailPassword)

	// Fetch messages from past year
	since := time.Now().AddDate(-1, 0, 0)
	messages, err := mp.FetchMessages(cfg.MailingListEmail, since)
	if err != nil {
		log.Printf("Error fetching messages: %v", err)
		return
	}

	if len(messages) == 0 {
		log.Println("No new messages to sync")
		return
	}

	// Group messages by subject (thread)
	threads := groupByThread(messages)

	// Store in database
	threadAnalyzer := analyzer.NewThreadAnalyzer(db)
	for subject, msgs := range threads {
		threadID := uuid.New().String()

		// Insert thread
		firstMsg := msgs[0]
		_, err := db.Exec(`
			INSERT INTO threads (id, subject, first_message_id, first_author, first_author_email, created_at, last_message_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`, threadID, subject, firstMsg.MessageID, firstMsg.Author, firstMsg.AuthorEmail, firstMsg.CreatedAt, firstMsg.CreatedAt)

		if err != nil {
			log.Printf("Error inserting thread: %v", err)
			continue
		}

		// Insert messages
		for _, msg := range msgs {
			msg.ID = uuid.New().String()
			msg.ThreadID = threadID
			_, err := db.Exec(`
				INSERT INTO messages (id, thread_id, message_id, subject, author, author_email, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (message_id) DO NOTHING
			`, msg.ID, msg.ThreadID, msg.MessageID, msg.Subject, msg.Author, msg.AuthorEmail, msg.CreatedAt)

			if err != nil {
				log.Printf("Error inserting message: %v", err)
			}
		}

		// Analyze thread
		if err := threadAnalyzer.UpdateThreadActivity(threadID); err != nil {
			log.Printf("Error updating thread activity: %v", err)
		}

		// Classify thread
		status, err := threadAnalyzer.ClassifyThread(threadID)
		if err == nil {
			db.Exec("UPDATE threads SET status = $1 WHERE id = $2", status, threadID)
		}
	}

	log.Println("Mail sync completed")
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
			"status":   "Mbox file uploaded and queued for processing",
			"filename": header.Filename,
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
	log.Println("Starting mbox sync...")

	mboxParser := parser.NewMboxParser(cfg.DataDir)
	messages, err := mboxParser.ParseAllMboxFiles()
	if err != nil {
		log.Printf("Error parsing mbox files: %v", err)
		return
	}

	if len(messages) == 0 {
		log.Println("No mbox files found")
		return
	}

	storeMessagesInDB(db, messages)
	log.Printf("Mbox sync completed with %d messages", len(messages))
}

func storeMessagesInDB(db *sql.DB, messages []*models.Message) {
	// Group messages by subject (thread)
	threads := groupByThread(messages)

	threadAnalyzer := analyzer.NewThreadAnalyzer(db)
	for subject, msgs := range threads {
		threadID := uuid.New().String()

		// Insert thread
		firstMsg := msgs[0]
		_, err := db.Exec(`
			INSERT INTO threads (id, subject, first_message_id, first_author, first_author_email, created_at, last_message_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT DO NOTHING
		`, threadID, subject, firstMsg.MessageID, firstMsg.Author, firstMsg.AuthorEmail, firstMsg.CreatedAt, firstMsg.CreatedAt)

		if err != nil {
			log.Printf("Error inserting thread: %v", err)
			continue
		}

		// Insert messages
		for _, msg := range msgs {
			msg.ID = uuid.New().String()
			msg.ThreadID = threadID
			_, err := db.Exec(`
				INSERT INTO messages (id, thread_id, message_id, subject, author, author_email, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (message_id) DO NOTHING
			`, msg.ID, msg.ThreadID, msg.MessageID, msg.Subject, msg.Author, msg.AuthorEmail, msg.CreatedAt)

			if err != nil {
				log.Printf("Error inserting message: %v", err)
			}
		}

		// Analyze thread
		if err := threadAnalyzer.UpdateThreadActivity(threadID); err != nil {
			log.Printf("Error updating thread activity: %v", err)
		}

		// Classify thread
		status, err := threadAnalyzer.ClassifyThread(threadID)
		if err == nil {
			db.Exec("UPDATE threads SET status = $1 WHERE id = $2", status, threadID)
		}
	}
}

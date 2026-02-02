package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pgsql-analyzer/backend/config"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	var connStr string
	if cfg.DatabaseURL != "" {
		connStr = cfg.DatabaseURL
	} else {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS threads (
		id VARCHAR(255) PRIMARY KEY,
		subject TEXT NOT NULL,
		first_message_id VARCHAR(255) NOT NULL,
		first_author VARCHAR(255) NOT NULL,
		first_author_email VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_message_at TIMESTAMP,
		message_count INT DEFAULT 0,
		unique_authors INT DEFAULT 0,
		status VARCHAR(50) DEFAULT 'discussion'
	);

	CREATE TABLE IF NOT EXISTS messages (
		id VARCHAR(255) PRIMARY KEY,
		thread_id VARCHAR(255) NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
		message_id VARCHAR(255) NOT NULL UNIQUE,
		subject TEXT NOT NULL,
		author VARCHAR(255) NOT NULL,
		author_email VARCHAR(255) NOT NULL,
		body TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (thread_id) REFERENCES threads(id)
	);

	CREATE TABLE IF NOT EXISTS thread_activities (
		id VARCHAR(255) PRIMARY KEY,
		thread_id VARCHAR(255) NOT NULL UNIQUE REFERENCES threads(id) ON DELETE CASCADE,
		message_count INT DEFAULT 0,
		unique_authors INT DEFAULT 0,
		has_patch BOOLEAN DEFAULT FALSE,
		has_review BOOLEAN DEFAULT FALSE,
		is_resolved BOOLEAN DEFAULT FALSE,
		days_since_last_message INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
	CREATE INDEX IF NOT EXISTS idx_threads_status ON threads(status);
	CREATE INDEX IF NOT EXISTS idx_threads_last_message ON threads(last_message_at);
	`

	_, err := db.Exec(schema)
	return err
}

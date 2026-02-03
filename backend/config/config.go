package config

import (
	"os"
)

type Config struct {
	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBName      string
	DBUser      string
	DBPassword  string

	// API
	APIPort string
	APIHost string

	// Mail
	MailIMAPHost string
	MailIMAPPort string
	MailUsername string
	MailPassword string

	// Mailing list to sync
	MailingListEmail string

	// File storage
	DataDir string

	// PostgreSQL.org mbox archive (HTTP Basic Auth; required for raw mbox download)
	ArchiveUsername string
	ArchivePassword string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBName:           getEnv("DB_NAME", "pgsql_analyzer"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		APIPort:          getEnv("API_PORT", "8080"),
		APIHost:          getEnv("API_HOST", "0.0.0.0"),
		MailIMAPHost:     getEnv("MAIL_IMAP_HOST", "imap.gmail.com"),
		MailIMAPPort:     getEnv("MAIL_IMAP_PORT", "993"),
		MailUsername:     getEnv("MAIL_USERNAME", ""),
		MailPassword:     getEnv("MAIL_PASSWORD", ""),
		MailingListEmail: getEnv("MAILING_LIST_EMAIL", "pgsql-hackers@postgresql.org"),
		DataDir:          getEnv("DATA_DIR", "./data"),
		ArchiveUsername:  getEnv("ARCHIVE_USERNAME", "archives"),
		ArchivePassword:  getEnv("ARCHIVE_PASSWORD", "antispam"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

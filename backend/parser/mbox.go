package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pgsql-analyzer/backend/models"
)

// MboxParser handles parsing mbox format files
type MboxParser struct {
	dataDir string
}

// NewMboxParser creates a new mbox parser
func NewMboxParser(dataDir string) *MboxParser {
	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)
	return &MboxParser{
		dataDir: dataDir,
	}
}

// ParseMboxFile parses a single mbox file
func (mp *MboxParser) ParseMboxFile(filePath string) ([]*models.Message, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mbox file: %w", err)
	}
	defer file.Close()

	var messages []*models.Message
	var currentMessage *models.Message
	var messageBody strings.Builder
	inBody := false // Track if we've finished headers and are in body

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for start of new message (mbox format: "From " at line start)
		if strings.HasPrefix(line, "From ") {
			// Save previous message if it exists
			if currentMessage != nil {
				currentMessage.Body = strings.TrimSpace(messageBody.String())
				messages = append(messages, currentMessage)
			}

			// Start new message
			currentMessage = &models.Message{}
			messageBody.Reset()
			inBody = false
			continue
		}

		if currentMessage == nil {
			continue
		}

		// Blank line separates headers from body
		if !inBody && strings.TrimSpace(line) == "" {
			inBody = true
			continue
		}

		// Parse email headers (before blank line)
		if !inBody && strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				header := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch strings.ToLower(header) {
				case "message-id":
					// Clean up message-id by removing angle brackets
					currentMessage.MessageID = strings.Trim(value, "<>")
				case "subject":
					currentMessage.Subject = normalizeSubject(value)
				case "from":
					currentMessage.Author, currentMessage.AuthorEmail = parseFromHeader(value)
				case "date":
					currentMessage.CreatedAt = parseDate(value)
				}
			}
		} else if inBody {
			// Body content (after blank line)
			messageBody.WriteString(line)
			messageBody.WriteString("\n")
		}
	}

	// Save last message
	if currentMessage != nil {
		currentMessage.Body = strings.TrimSpace(messageBody.String())
		messages = append(messages, currentMessage)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading mbox file: %w", err)
	}

	return messages, nil
}

// SaveMboxFile saves an mbox file to the data directory
func (mp *MboxParser) SaveMboxFile(fileName string, content []byte) (string, error) {
	// Sanitize filename
	fileName = filepath.Base(fileName)
	filePath := filepath.Join(mp.dataDir, fileName)

	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save mbox file: %w", err)
	}

	return filePath, nil
}

// ListMboxFiles returns all mbox files in the data directory
func (mp *MboxParser) ListMboxFiles() ([]string, error) {
	entries, err := os.ReadDir(mp.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mbox") {
			files = append(files, filepath.Join(mp.dataDir, entry.Name()))
		}
	}

	return files, nil
}

// ParseAllMboxFiles parses all mbox files in the data directory
func (mp *MboxParser) ParseAllMboxFiles() ([]*models.Message, error) {
	files, err := mp.ListMboxFiles()
	if err != nil {
		return nil, err
	}

	var allMessages []*models.Message
	for _, filePath := range files {
		messages, err := mp.ParseMboxFile(filePath)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Error parsing %s: %v\n", filePath, err)
			continue
		}
		allMessages = append(allMessages, messages...)
	}

	return allMessages, nil
}

// normalizeSubject removes Re:, Fwd: prefixes from subject
func normalizeSubject(subject string) string {
	subject = strings.TrimSpace(subject)
	for {
		original := subject
		for _, prefix := range []string{"Re:", "RE:", "Fwd:", "FWD:", "Fw:"} {
			if strings.HasPrefix(subject, prefix) {
				subject = strings.TrimSpace(strings.TrimPrefix(subject, prefix))
			}
		}
		if subject == original {
			break
		}
	}
	return subject
}

// parseFromHeader extracts name and email from "From" header
func parseFromHeader(from string) (string, string) {
	// Handle "Name <email@example.com>" format
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		parts := strings.Split(from, "<")
		name := strings.TrimSpace(parts[0])
		email := strings.TrimRight(strings.TrimLeft(parts[1], "<"), ">")
		if name == "" {
			name = email
		}
		return name, email
	}
	// Handle "email@example.com" format
	return from, from
}

// parseDate parses RFC2822 date format
func parseDate(dateStr string) time.Time {
	// Try common formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Default to now if parsing fails
	return time.Now()
}

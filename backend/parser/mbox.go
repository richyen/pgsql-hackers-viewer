package parser

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
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
	var contentTransferEncoding string
	inBody := false // Track if we've finished headers and are in body

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for start of new message (mbox format: "From " at line start)
		if strings.HasPrefix(line, "From ") {
			// Save previous message if it exists
			if currentMessage != nil {
				currentMessage.Body = decodeMessageBody(messageBody.String(), contentTransferEncoding)
				messages = append(messages, currentMessage)
			}

			// Start new message
			currentMessage = &models.Message{}
			messageBody.Reset()
			contentTransferEncoding = ""
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
				case "content-transfer-encoding":
					contentTransferEncoding = strings.ToLower(strings.TrimSpace(value))
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
		currentMessage.Body = decodeMessageBody(messageBody.String(), contentTransferEncoding)
		// Detect patches in message body
		currentMessage.HasPatch = detectPatch(currentMessage.Body, currentMessage.Subject)
		if currentMessage.HasPatch {
			currentMessage.PatchStatus = detectPatchStatus(currentMessage.Body, currentMessage.Subject)
		}
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

// decodeMessageBody decodes the message body based on Content-Transfer-Encoding
// Also handles MIME multipart messages by extracting and decoding each part
func decodeMessageBody(body, encoding string) string {
	body = strings.TrimSpace(body)

	// Check if this is a multipart MIME message
	if strings.Contains(body, "Content-Type:") && strings.Contains(body, "------") {
		return decodeMimeMultipart(body)
	}

	switch encoding {
	case "base64":
		// Decode base64 content
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			// Try with padding if it fails
			body = strings.ReplaceAll(body, "\n", "")
			body = strings.ReplaceAll(body, "\r", "")
			decoded, err = base64.StdEncoding.DecodeString(body)
			if err != nil {
				// Return original if decoding fails
				return body
			}
		}
		return string(decoded)

	case "quoted-printable":
		// Decode quoted-printable content
		reader := quotedprintable.NewReader(strings.NewReader(body))
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return body
		}
		return string(decoded)

	case "7bit", "8bit", "binary", "":
		// No decoding needed
		return body

	default:
		// Unknown encoding, return as-is
		return body
	}
}

// decodeMimeMultipart extracts and decodes text parts from a MIME multipart message
func decodeMimeMultipart(body string) string {
	var result strings.Builder
	lines := strings.Split(body, "\n")

	var inPart bool
	var partEncoding string
	var partContentType string
	var partBody strings.Builder

	for _, line := range lines {
		// Check for boundary markers
		if strings.HasPrefix(line, "------") {
			// Save previous part if it was text
			if inPart && strings.Contains(partContentType, "text/") {
				decoded := decodePartBody(partBody.String(), partEncoding)
				if result.Len() > 0 {
					result.WriteString("\n\n---\n\n")
				}
				result.WriteString(decoded)
			}

			// Reset for new part
			inPart = true
			partEncoding = ""
			partContentType = ""
			partBody.Reset()
			continue
		}

		if !inPart {
			continue
		}

		// Parse part headers
		if strings.HasPrefix(line, "Content-Type:") {
			partContentType = strings.ToLower(line)
		} else if strings.HasPrefix(line, "Content-Transfer-Encoding:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				partEncoding = strings.ToLower(strings.TrimSpace(parts[1]))
			}
		} else if strings.TrimSpace(line) == "" && partContentType != "" {
			// Empty line after headers marks start of body
			continue
		} else if partContentType != "" {
			// Part body content
			partBody.WriteString(line)
			partBody.WriteString("\n")
		}
	}

	// Save last part
	if inPart && strings.Contains(partContentType, "text/") {
		decoded := decodePartBody(partBody.String(), partEncoding)
		if result.Len() > 0 {
			result.WriteString("\n\n---\n\n")
		}
		result.WriteString(decoded)
	}

	if result.Len() > 0 {
		return strings.TrimSpace(result.String())
	}

	// If no text parts found, return original
	return body
}

// decodePartBody decodes a MIME part body based on its encoding
func decodePartBody(body, encoding string) string {
	body = strings.TrimSpace(body)

	switch encoding {
	case "base64":
		// Remove newlines for base64 decoding
		body = strings.ReplaceAll(body, "\n", "")
		body = strings.ReplaceAll(body, "\r", "")
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return body
		}
		return string(decoded)

	case "quoted-printable":
		reader := quotedprintable.NewReader(strings.NewReader(body))
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return body
		}
		return string(decoded)

	default:
		return body
	}
}

// detectPatch checks if a message contains a patch
func detectPatch(body, subject string) bool {
	bodyLower := strings.ToLower(body)
	subjectLower := strings.ToLower(subject)

	// Check for patch indicators in subject
	if strings.Contains(subjectLower, "[patch") ||
		strings.Contains(subjectLower, "patch v") ||
		strings.Contains(subjectLower, "v1 patch") ||
		strings.Contains(subjectLower, "v2 patch") {
		return true
	}

	// Check for diff/patch content in body
	// Look for unified diff format markers
	if strings.Contains(body, "diff --git") ||
		strings.Contains(body, "--- a/") ||
		strings.Contains(body, "+++ b/") {
		return true
	}

	// Look for context diff markers
	if strings.Contains(body, "*** ") && strings.Contains(body, "--- ") {
		return true
	}

	// Look for patch attachment indicators
	if strings.Contains(bodyLower, "attached patch") ||
		strings.Contains(bodyLower, "patch attached") ||
		strings.Contains(bodyLower, ".patch") ||
		strings.Contains(bodyLower, "content-disposition: attachment") {
		return true
	}

	return false
}

// detectPatchStatus analyzes the message to determine patch status
func detectPatchStatus(body, subject string) string {
	bodyLower := strings.ToLower(body)
	subjectLower := strings.ToLower(subject)

	// Check for committed/applied indicators
	if strings.Contains(bodyLower, "committed") ||
		strings.Contains(bodyLower, "pushed") ||
		strings.Contains(bodyLower, "applied") ||
		strings.Contains(subjectLower, "committed") {
		return "committed"
	}

	// Check for accepted/ready for committer indicators
	if strings.Contains(bodyLower, "ready for committer") ||
		strings.Contains(bodyLower, "marked as ready") ||
		strings.Contains(bodyLower, "moved to ready for committer") {
		return "accepted"
	}

	// Check for rejected indicators
	if strings.Contains(bodyLower, "rejected") ||
		strings.Contains(bodyLower, "not applying") ||
		strings.Contains(bodyLower, "returned with feedback") {
		return "rejected"
	}

	// Check for commitfest references
	if strings.Contains(bodyLower, "commitfest") ||
		strings.Contains(bodyLower, "cf entry") ||
		strings.Contains(subjectLower, "commitfest") {
		// Extract commitfest ID if possible - for now just mark as proposed
		return "proposed"
	}

	// Default status for patches
	return "proposed"
}

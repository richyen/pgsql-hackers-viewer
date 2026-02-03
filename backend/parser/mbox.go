package parser

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime/quotedprintable"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgsql-analyzer/backend/models"
)

// ParseStats tracks statistics from parsing mbox files
type ParseStats struct {
	Total              int `json:"total"`
	Parsed             int `json:"parsed"`
	Skipped            int `json:"skipped"`
	InvalidMessageID   int `json:"invalid_message_id"`
	InvalidDate        int `json:"invalid_date"`
	InvalidFrom        int `json:"invalid_from"`
	MalformedMessageID int `json:"malformed_message_id"`
}

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

// cleanMessageID validates and cleans a Message-ID header value
// Returns cleaned Message-ID and error if invalid
func cleanMessageID(msgid string) (string, error) {
	// Remove leading/trailing whitespace and angle brackets
	msgid = strings.Trim(strings.TrimSpace(msgid), "<>")

	// Remove internal spaces (common issue in malformed Message-IDs)
	msgid = strings.ReplaceAll(msgid, " ", "")
	msgid = strings.ReplaceAll(msgid, "\t", "")

	// Validate format
	if msgid == "" {
		return "", fmt.Errorf("empty message-id")
	}

	// Basic validation: should contain @ symbol
	if !strings.Contains(msgid, "@") {
		return "", fmt.Errorf("invalid message-id format (no @): %s", msgid)
	}

	return msgid, nil
}

// generateFallbackMessageID creates a unique Message-ID for messages with missing/broken IDs
func generateFallbackMessageID() string {
	return fmt.Sprintf("generated-%s@pgsql-analyzer.local", uuid.New().String())
}

// processHeader applies a parsed header to the message
func processHeader(msg *models.Message, header, value string, contentTransferEncoding *string, contentType *string, stats *ParseStats) {
	switch header {
	case "message-id":
		// Clean and validate message-id
		cleaned, err := cleanMessageID(value)
		if err != nil {
			// Generate fallback Message-ID for broken headers
			cleaned = generateFallbackMessageID()
			log.Printf("WARNING: Generated Message-ID for malformed header '%s': %v", value, err)
			if stats != nil {
				stats.MalformedMessageID++
			}
		}
		msg.MessageID = cleaned
	case "in-reply-to":
		// Clean up in-reply-to by removing angle brackets and whitespace
		cleaned, _ := cleanMessageID(value)
		if cleaned != "" {
			msg.InReplyTo = cleaned
		}
	case "references":
		// Store references as-is (will be parsed by parseReferences in threading code)
		msg.RefersTo = value
	case "subject":
		msg.Subject = normalizeSubject(value)
	case "from":
		msg.Author, msg.AuthorEmail = parseFromHeader(value)
	case "date":
		msg.CreatedAt = parseDate(value)
	case "content-transfer-encoding":
		*contentTransferEncoding = strings.ToLower(strings.TrimSpace(value))
	case "content-type":
		*contentType = value
	}
}

// ParseMboxFile parses a single mbox file and returns messages with statistics
func (mp *MboxParser) ParseMboxFile(filePath string) ([]*models.Message, *ParseStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open mbox file: %w", err)
	}
	defer file.Close()

	stats := &ParseStats{}
	var messages []*models.Message
	var currentMessage *models.Message
	var messageBody strings.Builder
	var contentTransferEncoding string
	var contentType string
	inBody := false // Track if we've finished headers and are in body
	var lastHeader string
	var lastValue string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for start of new message (mbox format: "From " at line start)
		if strings.HasPrefix(line, "From ") {
			stats.Total++

			// Save any pending header
			if lastHeader != "" && currentMessage != nil {
				processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding, &contentType, stats)
			}

			// Save previous message if it exists and passes validation
			if currentMessage != nil {
				currentMessage.Body = decodeMessageBody(messageBody.String(), contentTransferEncoding, contentType)
				// Detect patches in message body
				currentMessage.HasPatch = detectPatch(currentMessage.Body, currentMessage.Subject)
				if currentMessage.HasPatch {
					currentMessage.PatchStatus = detectPatchStatus(currentMessage.Body, currentMessage.Subject)
				}

				// MANDATORY FIELD VALIDATION
				if currentMessage.MessageID == "" {
					log.Printf("SKIPPED: Message missing Message-ID (Subject: %s)", currentMessage.Subject)
					stats.Skipped++
					stats.InvalidMessageID++
				} else if currentMessage.Author == "" && currentMessage.AuthorEmail == "" {
					log.Printf("SKIPPED: Message %s missing From header", currentMessage.MessageID)
					stats.Skipped++
					stats.InvalidFrom++
				} else if currentMessage.CreatedAt.IsZero() || currentMessage.CreatedAt.Year() < 1990 {
					log.Printf("SKIPPED: Message %s has invalid date: %v", currentMessage.MessageID, currentMessage.CreatedAt)
					stats.Skipped++
					stats.InvalidDate++
				} else {
					// All validations passed
					messages = append(messages, currentMessage)
					stats.Parsed++
				}
			}

			// Start new message
			currentMessage = &models.Message{}
			messageBody.Reset()
			contentTransferEncoding = ""
			contentType = ""
			inBody = false
			lastHeader = ""
			lastValue = ""
			continue
		}

		if currentMessage == nil {
			continue
		}

		// Blank line separates headers from body
		if !inBody && strings.TrimSpace(line) == "" {
			// Process any pending header before switching to body
			if lastHeader != "" {
				processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding, &contentType, stats)
				lastHeader = ""
				lastValue = ""
			}
			inBody = true
			continue
		}

		// Parse email headers (before blank line)
		if !inBody {
			// Check if this is a header continuation line (starts with whitespace)
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				// Continuation of previous header
				lastValue += " " + strings.TrimSpace(line)
			} else if strings.Contains(line, ": ") {
				// New header - process previous one first
				if lastHeader != "" {
					processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding, &contentType, stats)
				}

				// Parse new header
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) == 2 {
					lastHeader = strings.ToLower(strings.TrimSpace(parts[0]))
					lastValue = strings.TrimSpace(parts[1])
				}
			}
		} else if inBody {
			// Body content (after blank line)
			messageBody.WriteString(line)
			messageBody.WriteString("\n")
		}
	}

	// Save last message with validation
	if currentMessage != nil {
		currentMessage.Body = decodeMessageBody(messageBody.String(), contentTransferEncoding, contentType)
		// Detect patches in message body
		currentMessage.HasPatch = detectPatch(currentMessage.Body, currentMessage.Subject)
		if currentMessage.HasPatch {
			currentMessage.PatchStatus = detectPatchStatus(currentMessage.Body, currentMessage.Subject)
		}

		// MANDATORY FIELD VALIDATION
		if currentMessage.MessageID == "" {
			log.Printf("SKIPPED: Last message missing Message-ID (Subject: %s)", currentMessage.Subject)
			stats.Skipped++
			stats.InvalidMessageID++
		} else if currentMessage.Author == "" && currentMessage.AuthorEmail == "" {
			log.Printf("SKIPPED: Message %s missing From header", currentMessage.MessageID)
			stats.Skipped++
			stats.InvalidFrom++
		} else if currentMessage.CreatedAt.IsZero() || currentMessage.CreatedAt.Year() < 1990 {
			log.Printf("SKIPPED: Message %s has invalid date: %v", currentMessage.MessageID, currentMessage.CreatedAt)
			stats.Skipped++
			stats.InvalidDate++
		} else {
			messages = append(messages, currentMessage)
			stats.Parsed++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, fmt.Errorf("error reading mbox file: %w", err)
	}

	log.Printf("Parse complete: %d total, %d parsed, %d skipped (MessageID: %d, Date: %d, From: %d, Malformed: %d)",
		stats.Total, stats.Parsed, stats.Skipped, stats.InvalidMessageID, stats.InvalidDate, stats.InvalidFrom, stats.MalformedMessageID)

	return messages, stats, nil
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
		name := entry.Name()
		// Match files ending in .mbox or starting with pgsql-hackers
		if !entry.IsDir() && (strings.HasSuffix(name, ".mbox") || strings.HasPrefix(name, "pgsql-hackers")) {
			files = append(files, filepath.Join(mp.dataDir, name))
		}
	}

	return files, nil
}

// ParseAllMboxFiles parses all mbox files in the data directory
func (mp *MboxParser) ParseAllMboxFiles() ([]*models.Message, *ParseStats, error) {
	files, err := mp.ListMboxFiles()
	if err != nil {
		return nil, nil, err
	}

	totalStats := &ParseStats{}
	var allMessages []*models.Message
	for _, filePath := range files {
		log.Printf("Parsing file: %s", filePath)
		messages, stats, err := mp.ParseMboxFile(filePath)
		if err != nil {
			// Log error but continue with other files
			log.Printf("Error parsing %s: %v", filePath, err)
			continue
		}
		// Aggregate stats
		if stats != nil {
			totalStats.Total += stats.Total
			totalStats.Parsed += stats.Parsed
			totalStats.Skipped += stats.Skipped
			totalStats.InvalidMessageID += stats.InvalidMessageID
			totalStats.InvalidDate += stats.InvalidDate
			totalStats.InvalidFrom += stats.InvalidFrom
			totalStats.MalformedMessageID += stats.MalformedMessageID
		}
		allMessages = append(allMessages, messages...)
	}

	log.Printf("\n=== TOTAL PARSING STATS ===")
	log.Printf("Total messages found: %d", totalStats.Total)
	log.Printf("Successfully parsed: %d (%.1f%%)", totalStats.Parsed, float64(totalStats.Parsed)/float64(totalStats.Total)*100)
	log.Printf("Skipped: %d (%.1f%%)", totalStats.Skipped, float64(totalStats.Skipped)/float64(totalStats.Total)*100)
	log.Printf("  - Missing/Invalid Message-ID: %d", totalStats.InvalidMessageID)
	log.Printf("  - Malformed Message-ID (fixed): %d", totalStats.MalformedMessageID)
	log.Printf("  - Invalid Date: %d", totalStats.InvalidDate)
	log.Printf("  - Missing From: %d", totalStats.InvalidFrom)

	return allMessages, totalStats, nil
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
		"Mon, 2 Jan 2006 15:04:05 -0700", // Single-digit day
		"02 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700", // Single-digit day without day name
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
func decodeMessageBody(body, encoding, contentType string) string {
	body = strings.TrimSpace(body)

	// Check if this is a multipart MIME message
	if strings.Contains(strings.ToLower(contentType), "multipart") && strings.Contains(contentType, "boundary=") {
		return decodeMimeMultipart(body, contentType)
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
// This function only extracts text/plain and text/html parts, skipping attachments
func decodeMimeMultipart(body, contentType string) string {
	// Extract boundary from Content-Type header
	boundary := extractBoundary(contentType)
	if boundary == "" {
		// No valid boundary found, return original
		return body
	}

	var result strings.Builder
	lines := strings.Split(body, "\n")

	// Track whether we're inside a part
	var inPart bool
	var partEncoding string
	var partContentType string
	var isAttachment bool
	var partBody strings.Builder
	var headersDone bool

	for _, line := range lines {
		// Check if this is a boundary marker
		if strings.HasPrefix(line, "--"+boundary) {
			// Save previous part only if it was text and not an attachment
			if inPart && strings.Contains(partContentType, "text/") && !isAttachment {
				decoded := decodePartBody(partBody.String(), partEncoding)
				if result.Len() > 0 && len(decoded) > 0 {
					result.WriteString("\n\n---\n\n")
				}
				if len(decoded) > 0 {
					result.WriteString(decoded)
				}
			}

			// Reset for new part
			inPart = true
			partEncoding = ""
			partContentType = ""
			isAttachment = false
			partBody.Reset()
			headersDone = false
			continue
		}

		if !inPart {
			continue
		}

		// Check if we've reached end of headers (empty line)
		if !headersDone && strings.TrimSpace(line) == "" {
			headersDone = true
			continue
		}

		// Parse part headers (before empty line)
		if !headersDone {
			lineLower := strings.ToLower(line)
			if strings.HasPrefix(lineLower, "content-type:") {
				partContentType = lineLower
			} else if strings.HasPrefix(lineLower, "content-transfer-encoding:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					partEncoding = strings.ToLower(strings.TrimSpace(parts[1]))
				}
			} else if strings.HasPrefix(lineLower, "content-disposition:") && strings.Contains(lineLower, "attachment") {
				// Mark this part as an attachment to skip it
				isAttachment = true
			}
		} else if headersDone && strings.Contains(partContentType, "text/") && !isAttachment {
			// Only collect body content for text parts that are not attachments
			partBody.WriteString(line)
			partBody.WriteString("\n")
		}
		// Skip collecting body for non-text parts and attachments
	}

	// Save last part only if it was text and not an attachment
	if inPart && strings.Contains(partContentType, "text/") && !isAttachment {
		decoded := decodePartBody(partBody.String(), partEncoding)
		if result.Len() > 0 && len(decoded) > 0 {
			result.WriteString("\n\n---\n\n")
		}
		if len(decoded) > 0 {
			result.WriteString(decoded)
		}
	}

	if result.Len() > 0 {
		return strings.TrimSpace(result.String())
	}

	// If no text parts found, return original
	return body
}

// extractBoundary extracts the MIME boundary from Content-Type header
func extractBoundary(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "boundary=") {
			// Extract boundary value
			// Format: boundary="value" or boundary=value
			idx := strings.Index(lineLower, "boundary=")
			if idx >= 0 {
				value := line[idx+9:] // Skip "boundary="
				value = strings.TrimSpace(value)

				// Remove quotes if present
				value = strings.Trim(value, "\"")
				value = strings.Trim(value, "'")

				// Take only up to next semicolon or newline
				if idx := strings.IndexAny(value, ";\n\r"); idx >= 0 {
					value = value[:idx]
				}

				value = strings.TrimSpace(value)
				if len(value) > 0 {
					return value
				}
			}
		}
	}
	return ""
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

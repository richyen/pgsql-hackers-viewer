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

// processHeader applies a parsed header to the message
func processHeader(msg *models.Message, header, value string, contentTransferEncoding *string) {
	switch header {
	case "message-id":
		// Clean up message-id by removing angle brackets and whitespace
		msg.MessageID = strings.Trim(strings.TrimSpace(value), "<>")
	case "in-reply-to":
		// Clean up in-reply-to by removing angle brackets and whitespace
		msg.InReplyTo = strings.Trim(strings.TrimSpace(value), "<>")
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
	var lastHeader string
	var lastValue string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for start of new message (mbox format: "From " at line start)
		if strings.HasPrefix(line, "From ") {
			// Save any pending header
			if lastHeader != "" && currentMessage != nil {
				processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding)
			}

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
				processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding)
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
					processHeader(currentMessage, lastHeader, lastValue, &contentTransferEncoding)
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
func decodeMessageBody(body, encoding string) string {
	body = strings.TrimSpace(body)

	// Check if this is a multipart MIME message
	if strings.Contains(body, "Content-Type:") && strings.Contains(body, "boundary=") {
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
// This function only extracts text/plain and text/html parts, skipping attachments
func decodeMimeMultipart(body string) string {
	// Extract boundary from message
	boundary := extractBoundary(body)
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

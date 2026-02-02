package parser

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/pgsql-analyzer/backend/models"
)

type MailParser struct {
	host     string
	port     string
	username string
	password string
}

func NewMailParser(host, port, username, password string) *MailParser {
	return &MailParser{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
}

func (mp *MailParser) FetchMessages(mailingListEmail string, since time.Time) ([]*models.Message, error) {
	addr := fmt.Sprintf("%s:%s", mp.host, mp.port)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		log.Printf("Error connecting to IMAP server: %v", err)
		return nil, err
	}
	defer c.Logout()

	if err := c.Login(mp.username, mp.password); err != nil {
		log.Printf("Error logging in: %v", err)
		return nil, err
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Printf("Error selecting inbox: %v", err)
		return nil, err
	}

	if mbox.Messages == 0 {
		log.Println("No messages in mailbox")
		return nil, nil
	}

	// Build search criteria
	criteria := &imap.SearchCriteria{
		Since: since,
	}

	// Search for messages
	ids, err := c.Search(criteria)
	if err != nil {
		log.Printf("Error searching messages: %v", err)
		return nil, err
	}

	if len(ids) == 0 {
		log.Println("No messages found matching criteria")
		return nil, nil
	}

	// Fetch messages
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}
		done <- c.Fetch(seqset, items, messages)
	}()

	var parsedMessages []*models.Message
	for msg := range messages {
		parsedMsg := parseIMAPMessage(msg)
		if parsedMsg != nil {
			parsedMessages = append(parsedMessages, parsedMsg)
		}
	}

	if err := <-done; err != nil {
		log.Printf("Error fetching messages: %v", err)
		return nil, err
	}

	return parsedMessages, nil
}

func parseIMAPMessage(msg *imap.Message) *models.Message {
	if msg == nil || msg.Envelope == nil {
		return nil
	}

	env := msg.Envelope
	var author, authorEmail string

	if len(env.From) > 0 {
		author = env.From[0].PersonalName
		if author == "" {
			author = env.From[0].MailboxName
		}
		authorEmail = env.From[0].Address()
	}

	messageID := env.MessageId
	if messageID == "" {
		messageID = generateMessageID(env)
	}

	subject := env.Subject
	// Normalize subject (remove Re:, Fwd:, etc.)
	subject = strings.TrimSpace(subject)
	for _, prefix := range []string{"Re:", "Fwd:", "RE:", "FWD:"} {
		subject = strings.TrimPrefix(subject, prefix)
		subject = strings.TrimSpace(subject)
	}

	date := env.Date
	if date.IsZero() {
		date = time.Now()
	}

	return &models.Message{
		MessageID:   messageID,
		Subject:     subject,
		Author:      author,
		AuthorEmail: authorEmail,
		CreatedAt:   date,
	}
}

func generateMessageID(env *imap.Envelope) string {
	return fmt.Sprintf("<%s-%d>", env.MessageId, time.Now().UnixNano())
}

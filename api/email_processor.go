package api

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// EmailMetadata stores information about emails
type EmailMetadata struct {
	ID           string    `json:"id"`
	ThreadID     string    `json:"threadId"`
	From         string    `json:"from"`
	To           []string  `json:"to"`
	Subject      string    `json:"subject"`
	Date         time.Time `json:"date"`
	Snippet      string    `json:"snippet"`
	LabelIDs     []string  `json:"labelIds"`
	SizeEstimate int64     `json:"sizeEstimate"`
}

// EmailStats tracks statistics about email communications
type EmailStats struct {
	// Maps sender email to number of emails received
	FromCount map[string]int `json:"fromCount"`
	// Maps recipient email to number of emails sent
	ToCount map[string]int `json:"toCount"`
	// Maps sender to total size of emails received
	FromSize map[string]int64 `json:"fromSize"`
	// Maps date to number of emails
	DateCount map[string]int `json:"dateCount"`
	// Total emails processed
	TotalEmails int `json:"totalEmails"`
	// Lock for concurrent map access
	mu sync.RWMutex
}

// NewEmailStats creates a new EmailStats instance
func NewEmailStats() *EmailStats {
	return &EmailStats{
		FromCount: make(map[string]int),
		ToCount:   make(map[string]int),
		FromSize:  make(map[string]int64),
		DateCount: make(map[string]int),
	}
}

// InboxProcessor manages the process of downloading and analyzing inbox data
type InboxProcessor struct {
	token        *oauth2.Token
	service      *gmail.Service
	emails       []EmailMetadata
	stats        *EmailStats
	pageToken    string
	isProcessing bool
	mu           sync.RWMutex
}

// NewInboxProcessor creates a new InboxProcessor
func NewInboxProcessor(token *oauth2.Token) (*InboxProcessor, error) {
	client := oauthConfig.Client(context.Background(), token)
	service, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &InboxProcessor{
		token:        token,
		service:      service,
		emails:       make([]EmailMetadata, 0),
		stats:        NewEmailStats(),
		isProcessing: false,
	}, nil
}

// StartProcessing begins downloading and processing emails in the background
func (p *InboxProcessor) StartProcessing() error {
	p.mu.Lock()
	if p.isProcessing {
		p.mu.Unlock()
		return fmt.Errorf("processing already in progress")
	}
	p.isProcessing = true
	p.mu.Unlock()

	go p.processInbox()
	return nil
}

// GetStats returns current email statistics
func (p *InboxProcessor) GetStats() *EmailStats {
	return p.stats
}

// GetProgress returns the current progress
func (p *InboxProcessor) GetProgress() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"totalEmails":  p.stats.TotalEmails,
		"isProcessing": p.isProcessing,
	}
}

// extractEmailAddress extracts the email address from the value field of a header
func extractEmailAddress(header string) string {
	// This is a simple extraction - you might want to use a regex for more accurate parsing
	// Example: "John Doe <john@example.com>" -> "john@example.com"

	start := 0
	end := len(header)

	// Find the start of the email (after '<' if present)
	for i := 0; i < len(header); i++ {
		if header[i] == '<' {
			start = i + 1
			break
		}
	}

	// Find the end of the email (before '>' if present)
	for i := len(header) - 1; i >= 0; i-- {
		if header[i] == '>' {
			end = i
			break
		}
	}

	if start < end {
		return header[start:end]
	}

	return header
}

// processInbox handles downloading all emails from the inbox
func (p *InboxProcessor) processInbox() {
	user := "me" // special value for the authenticated user
	pageToken := ""
	pageSize := int64(100) // Number of messages to fetch per API call

	for {
		req := p.service.Users.Messages.List(user).MaxResults(pageSize)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, err := req.Do()
		if err != nil {
			log.Printf("Failed to fetch messages: %v", err)
			break
		}

		// Process each message
		var wg sync.WaitGroup
		for _, msg := range resp.Messages {
			wg.Add(1)
			go func(messageID string) {
				defer wg.Done()
				p.processMessage(user, messageID)
			}(msg.Id)
		}
		wg.Wait()

		// Update total count
		p.stats.mu.Lock()
		p.stats.TotalEmails += len(resp.Messages)
		p.stats.mu.Unlock()

		// Check if there are more pages
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	p.mu.Lock()
	p.isProcessing = false
	p.mu.Unlock()

	log.Printf("Email processing complete. Total emails processed: %d", p.stats.TotalEmails)
}

// processMessage fetches and processes a single email message
func (p *InboxProcessor) processMessage(user, messageID string) {
	// Get the full message details
	msg, err := p.service.Users.Messages.Get(user, messageID).Format("full").Do()
	if err != nil {
		log.Printf("Failed to fetch message %s: %v", messageID, err)
		return
	}

	// Initialize metadata
	metadata := EmailMetadata{
		ID:           msg.Id,
		ThreadID:     msg.ThreadId,
		LabelIDs:     msg.LabelIds,
		Snippet:      msg.Snippet,
		SizeEstimate: msg.SizeEstimate,
		To:           make([]string, 0),
	}

	// Extract headers
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From":
			metadata.From = extractEmailAddress(header.Value)
		case "To":
			// Note: To might contain multiple addresses, this is a simplified version
			metadata.To = append(metadata.To, extractEmailAddress(header.Value))
		case "Subject":
			metadata.Subject = header.Value
		case "Date":
			// Parse the date, with error handling
			t, err := time.Parse(time.RFC1123Z, header.Value)
			if err == nil {
				metadata.Date = t
			} else {
				// Try alternative formats if the standard format fails
				formats := []string{
					time.RFC1123Z,
					time.RFC1123,
					"Mon, 2 Jan 2006 15:04:05 -0700",
					"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
				}

				for _, format := range formats {
					if t, err := time.Parse(format, header.Value); err == nil {
						metadata.Date = t
						break
					}
				}
			}
		}
	}

	// Add to emails list
	p.mu.Lock()
	p.emails = append(p.emails, metadata)
	p.mu.Unlock()

	// Update statistics
	p.stats.mu.Lock()

	// Update from counts
	p.stats.FromCount[metadata.From]++

	// Update from size
	p.stats.FromSize[metadata.From] += int64(metadata.SizeEstimate)

	// Update to counts for each recipient
	for _, to := range metadata.To {
		p.stats.ToCount[to]++
	}

	// Update date counts
	if !metadata.Date.IsZero() {
		dateStr := metadata.Date.Format("2006-01-02")
		p.stats.DateCount[dateStr]++
	}

	p.stats.mu.Unlock()
}

// GetTopSenders returns the top N senders by email count
func (p *InboxProcessor) GetTopSenders(n int) []map[string]interface{} {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	// Convert map to slice for sorting
	type emailCount struct {
		Email string
		Count int
		Size  int64
	}

	senders := make([]emailCount, 0, len(p.stats.FromCount))
	for email, count := range p.stats.FromCount {
		size := p.stats.FromSize[email]
		senders = append(senders, emailCount{Email: email, Count: count, Size: size})
	}

	// Sort by count (descending)
	// Note: A more efficient implementation would use a heap for top-N
	for i := 0; i < len(senders); i++ {
		for j := i + 1; j < len(senders); j++ {
			if senders[i].Count < senders[j].Count {
				senders[i], senders[j] = senders[j], senders[i]
			}
		}
	}

	// Take top N
	if n > len(senders) {
		n = len(senders)
	}
	senders = senders[:n]

	// Convert to map for JSON response
	result := make([]map[string]interface{}, n)
	for i, sender := range senders {
		result[i] = map[string]interface{}{
			"email": sender.Email,
			"count": sender.Count,
			"size":  sender.Size,
		}
	}

	return result
}

// ProcessorRegistry manages active inbox processors
type ProcessorRegistry struct {
	processors map[string]*InboxProcessor
	mu         sync.RWMutex
}

var (
	// Global registry for inbox processors
	Registry = &ProcessorRegistry{
		processors: make(map[string]*InboxProcessor),
	}
)

// Register adds a new processor to the registry
func (r *ProcessorRegistry) Register(userID string, processor *InboxProcessor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processors[userID] = processor
}

// Get retrieves a processor from the registry
func (r *ProcessorRegistry) Get(userID string) (*InboxProcessor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	proc, ok := r.processors[userID]
	return proc, ok
}

// Remove deletes a processor from the registry
func (r *ProcessorRegistry) Remove(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.processors, userID)
}

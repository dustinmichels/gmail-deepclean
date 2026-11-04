package api

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// EmailMetadata stores parsed information about an email message
type EmailMetadata struct {
	ID               string    `json:"id"`
	ThreadID         string    `json:"threadId"`
	From             string    `json:"from"`
	FromName         string    `json:"fromName"`
	To               []string  `json:"to"`
	Subject          string    `json:"subject"`
	Date             time.Time `json:"date"`
	Snippet          string    `json:"snippet"`
	LabelIDs         []string  `json:"labelIds"`
	SizeEstimate     int64     `json:"sizeEstimate"`
	IsSent           bool      `json:"isSent"`
	SenderEmailCount int       `json:"senderEmailCount,omitempty"`
}

// ProgressInfo represents the current scanning progress
type ProgressInfo struct {
	UserID         string     `json:"userId"`
	UserEmail      string     `json:"userEmail"`
	IsProcessing   bool       `json:"isProcessing"`
	TotalEmails    int        `json:"totalEmails"`
	EstimatedTotal int64      `json:"estimatedTotal"`
	StatusMessage  string     `json:"statusMessage"`
	LastScannedAt  *time.Time `json:"lastScannedAt,omitempty"`
}

// InboxProcessor coordinates quota-safe background downloading and processing of Gmail messages
type InboxProcessor struct {
	userID         string
	userEmail      string
	oauthToken     *oauth2.Token
	service        *gmail.Service
	scanState      *ScanState
	cancelFunc     context.CancelFunc
	mu             sync.RWMutex
	concurrency    int
	rateLimiter    *time.Ticker
	estimatedTotal int64
	scanWg         sync.WaitGroup
}

// Global registry of active processors per user
type ProcessorRegistry struct {
	processors map[string]*InboxProcessor
	mu         sync.RWMutex
}

var Registry = &ProcessorRegistry{
	processors: make(map[string]*InboxProcessor),
}

func (r *ProcessorRegistry) GetOrCreate(userID, userEmail string, token *oauth2.Token) (*InboxProcessor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if proc, exists := r.processors[userID]; exists {
		proc.mu.Lock()
		proc.oauthToken = token
		proc.mu.Unlock()
		return proc, nil
	}

	proc, err := NewInboxProcessor(userID, userEmail, token)
	if err != nil {
		return nil, err
	}
	r.processors[userID] = proc
	return proc, nil
}

func (r *ProcessorRegistry) Get(userID string) (*InboxProcessor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	proc, exists := r.processors[userID]
	return proc, exists
}

// PauseAll signals cancellation to all active processors and waits for their background workers to exit.
func (r *ProcessorRegistry) PauseAll(ctx context.Context) error {
	r.mu.RLock()
	procs := make([]*InboxProcessor, 0, len(r.processors))
	for _, proc := range r.processors {
		procs = append(procs, proc)
	}
	r.mu.RUnlock()

	for _, proc := range procs {
		_ = proc.PauseProcessing()
	}

	var wg sync.WaitGroup
	for _, proc := range procs {
		wg.Add(1)
		go func(p *InboxProcessor) {
			defer wg.Done()
			p.scanWg.Wait()
		}(proc)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewInboxProcessor initializes a processor with DB-backed state and rate limiting
func NewInboxProcessor(userID, userEmail string, token *oauth2.Token) (*InboxProcessor, error) {
	ctx := context.Background()
	client := oauthConfig.Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	state, err := DB.GetScanState(userID)
	if err != nil {
		state = &ScanState{
			UserID:        userID,
			IsProcessing:  false,
			TotalEmails:   0,
			StatusMessage: "Ready to scan",
			UpdatedAt:     time.Now(),
		}
	}
	// On server restart, ensure isProcessing is false until started
	state.IsProcessing = false

	return &InboxProcessor{
		userID:         userID,
		userEmail:      strings.ToLower(strings.TrimSpace(userEmail)),
		oauthToken:     token,
		service:        service,
		scanState:      state,
		concurrency:    5,                                      // Bounded worker pool: 5 concurrent workers
		rateLimiter:    time.NewTicker(200 * time.Millisecond), // 5 requests/sec = 100 quota units/sec (safe under 6,000 units/min per-user limit)
		estimatedTotal: state.EstimatedTotal,
	}, nil
}

// StartProcessing initiates background scanning with worker pool and rate limits
func (p *InboxProcessor) StartProcessing(reset bool) error {
	p.mu.Lock()
	if p.scanState.IsProcessing {
		p.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}

	if reset {
		p.scanState.NextPageToken = ""
		p.scanState.TotalEmails = 0
		p.scanState.StatusMessage = "Starting new scan..."
	} else {
		p.scanState.StatusMessage = "Resuming scan..."
	}

	p.scanState.IsProcessing = true
	now := time.Now()
	p.scanState.LastScannedAt = &now
	_ = DB.SaveScanState(p.scanState)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFunc = cancel
	p.mu.Unlock()

	p.scanWg.Add(1)
	go func() {
		defer p.scanWg.Done()
		p.runScan(ctx)
	}()
	return nil
}

// PauseProcessing stops the background scan gracefully
func (p *InboxProcessor) PauseProcessing() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.scanState.IsProcessing {
		return nil
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	p.scanState.IsProcessing = false
	p.scanState.StatusMessage = "Scan paused"
	now := time.Now()
	p.scanState.LastScannedAt = &now
	return DB.SaveScanState(p.scanState)
}

// GetProgress returns the current status and metrics
func (p *InboxProcessor) GetProgress() ProgressInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return ProgressInfo{
		UserID:         p.userID,
		UserEmail:      p.userEmail,
		IsProcessing:   p.scanState.IsProcessing,
		TotalEmails:    p.scanState.TotalEmails,
		EstimatedTotal: p.estimatedTotal,
		StatusMessage:  p.scanState.StatusMessage,
		LastScannedAt:  p.scanState.LastScannedAt,
	}
}

// runScan executes the paginated scanning loop using a bounded worker pool
func (p *InboxProcessor) runScan(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		p.scanState.IsProcessing = false
		p.cancelFunc = nil
		now := time.Now()
		p.scanState.LastScannedAt = &now
		_ = DB.SaveScanState(p.scanState)
		p.mu.Unlock()
	}()

	// Attempt fast IMAP XOAUTH2 scan first
	imapErr := p.runImapScan(ctx)
	if imapErr == nil {
		log.Printf("[%s] IMAP scan completed successfully for %s", p.userID, p.userEmail)
		return
	}

	// If cancelled or paused by context, do not continue
	if ctx.Err() != nil {
		log.Printf("[%s] Scan paused or cancelled by context", p.userID)
		return
	}

	log.Printf("[%s] IMAP scan unavailable (%v). Falling back to Gmail REST API.", p.userID, imapErr)
	errMsg := "IMAP unavailable, falling back to Gmail REST API..."
	if strings.Contains(imapErr.Error(), "XOAUTH2") || strings.Contains(imapErr.Error(), "Invalid credentials") {
		errMsg = "IMAP requires re-login (scope upgrade). Falling back to REST API..."
	}
	p.updateStatus(errMsg)
	user := "me"
	pageSize := int64(500)
	if profile, err := p.service.Users.GetProfile(user).Context(ctx).Do(); err == nil && profile != nil {
		p.mu.Lock()
		p.estimatedTotal = profile.MessagesTotal
		p.scanState.EstimatedTotal = profile.MessagesTotal
		_ = DB.SaveScanState(p.scanState)
		p.mu.Unlock()
	}
	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Scan cancelled by context", p.userID)
			p.updateStatus("Scan paused")
			return
		default:
		}

		p.mu.RLock()
		pageToken := p.scanState.NextPageToken
		p.mu.RUnlock()

		req := p.service.Users.Messages.List(user).MaxResults(pageSize)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		// Fetch page with retry and backoff
		var resp *gmail.ListMessagesResponse
		var err error
		for attempt := range 4 {
			<-p.rateLimiter.C
			resp, err = req.Context(ctx).Do()
			if err == nil {
				break
			}

			if isQuotaOrRateLimitErr(err) {
				backoff := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(500))*time.Millisecond
				log.Printf("[%s] Rate limited fetching message list (attempt %d). Backing off %v", p.userID, attempt+1, backoff)
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return
				}
			}

			log.Printf("[%s] Error fetching message list: %v", p.userID, err)
			break
		}

		if err != nil {
			p.updateStatus(fmt.Sprintf("Error fetching messages: %v", err))
			return
		}

		if resp == nil || len(resp.Messages) == 0 {
			p.updateStatus("Scan complete. No more messages.")
			p.mu.Lock()
			p.scanState.NextPageToken = ""
			p.mu.Unlock()
			return
		}

		// Process messages with bounded worker pool
		results := p.processMessageBatch(ctx, resp.Messages)

		select {
		case <-ctx.Done():
			return
		default:
		}

		// Persist batch into SQLite
		if len(results) > 0 {
			if err := DB.SaveEmailsBatch(p.userID, results); err != nil {
				log.Printf("[%s] Error saving emails batch: %v", p.userID, err)
				p.updateStatus(fmt.Sprintf("Error saving emails: %v", err))
				return
			}
		}

		p.mu.Lock()
		p.scanState.TotalEmails += len(results)
		p.scanState.NextPageToken = resp.NextPageToken
		if p.estimatedTotal > 0 {
			pct := int(float64(p.scanState.TotalEmails) / float64(p.estimatedTotal) * 100)
			if pct > 100 {
				pct = 100
			}
			p.scanState.StatusMessage = fmt.Sprintf("Scanned %d of ~%d emails (%d%%)", p.scanState.TotalEmails, p.estimatedTotal, pct)
		} else {
			p.scanState.StatusMessage = fmt.Sprintf("Scanned %d emails...", p.scanState.TotalEmails)
		}
		_ = DB.SaveScanState(p.scanState)
		p.mu.Unlock()
		log.Printf("[%s] Batch of %d messages processed. Total: %d", p.userID, len(results), p.scanState.TotalEmails)

		if resp.NextPageToken == "" {
			p.updateStatus(fmt.Sprintf("Scan completed! Processed %d emails.", p.scanState.TotalEmails))
			break
		}
	}
}

// processMessageBatch coordinates worker goroutines to fetch message details safely
func (p *InboxProcessor) processMessageBatch(ctx context.Context, messages []*gmail.Message) []EmailMetadata {
	total := len(messages)
	jobs := make(chan *gmail.Message, total)
	resultsChan := make(chan EmailMetadata, total)

	for _, msg := range messages {
		jobs <- msg
	}
	close(jobs)

	var wg sync.WaitGroup
	workers := p.concurrency
	if workers <= 0 {
		workers = 5
	}

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-jobs:
					if !ok {
						return
					}
					meta, err := p.fetchMessageDetails(ctx, msg.Id)
					if err != nil {
						log.Printf("[%s] Warning: failed fetching message %s: %v", p.userID, msg.Id, err)
						continue
					}
					if meta != nil {
						resultsChan <- *meta
					}
				}
			}
		}()
	}

	wg.Wait()
	close(resultsChan)

	var results []EmailMetadata
	for res := range resultsChan {
		results = append(results, res)
	}

	return results
}

// fetchMessageDetails fetches a single message with exponential backoff on 429
func (p *InboxProcessor) fetchMessageDetails(ctx context.Context, messageID string) (*EmailMetadata, error) {
	user := "me"
	var msg *gmail.Message
	var err error

	for attempt := range 4 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.rateLimiter.C:
		}

		msg, err = p.service.Users.Messages.Get(user, messageID).
			Format("metadata").
			MetadataHeaders("From", "To", "Cc", "Subject", "Date").
			Context(ctx).
			Do()
		if err == nil {
			break
		}

		if isQuotaOrRateLimitErr(err) {
			backoff := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(400))*time.Millisecond
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		return nil, err
	}

	if msg == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("message %s not found", messageID)
	}

	metadata := EmailMetadata{
		ID:           msg.Id,
		ThreadID:     msg.ThreadId,
		LabelIDs:     msg.LabelIds,
		Snippet:      msg.Snippet,
		SizeEstimate: msg.SizeEstimate,
		To:           make([]string, 0),
	}

	// Parse Headers
	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			switch strings.ToLower(header.Name) {
			case "from":
				addr, name := parseSingleEmail(header.Value)
				metadata.From = addr
				metadata.FromName = name
			case "to":
				recipients := parseMultiEmail(header.Value)
				metadata.To = append(metadata.To, recipients...)
			case "cc":
				ccRecipients := parseMultiEmail(header.Value)
				metadata.To = append(metadata.To, ccRecipients...)
			case "subject":
				metadata.Subject = header.Value
			case "date":
				metadata.Date = parseEmailDate(header.Value)
			}
		}
	}

	// Fallback to internal date if Date header is missing or unparseable
	if metadata.Date.IsZero() && msg.InternalDate > 0 {
		metadata.Date = time.UnixMilli(msg.InternalDate)
	}

	// Check if this is a draft; unsent drafts must never count as sent emails or contacts
	for _, label := range metadata.LabelIDs {
		if strings.EqualFold(label, "DRAFT") {
			return nil, nil
		}
	}

	// Determine if sent by user
	isSent := false
	if strings.EqualFold(metadata.From, p.userEmail) {
		isSent = true
	} else {
		for _, label := range metadata.LabelIDs {
			if strings.EqualFold(label, "SENT") {
				isSent = true
				break
			}
		}
	}
	metadata.IsSent = isSent

	return &metadata, nil
}

func (p *InboxProcessor) updateStatus(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scanState.StatusMessage = msg
	now := time.Now()
	p.scanState.LastScannedAt = &now
	_ = DB.SaveScanState(p.scanState)
}

// Helper: parse single email into clean address and display name
func parseSingleEmail(header string) (string, string) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return "", ""
	}

	addr, err := mail.ParseAddress(trimmed)
	if err == nil && addr != nil {
		return strings.ToLower(strings.TrimSpace(addr.Address)), strings.TrimSpace(addr.Name)
	}

	// Fallback regex-free extractor
	start := strings.Index(trimmed, "<")
	end := strings.Index(trimmed, ">")
	if start != -1 && end != -1 && start < end {
		cleanAddr := strings.ToLower(strings.TrimSpace(trimmed[start+1 : end]))
		name := strings.TrimSpace(trimmed[:start])
		name = strings.Trim(name, "\"'")
		return cleanAddr, name
	}

	return strings.ToLower(trimmed), ""
}

// Helper: parse multi-recipient header
func parseMultiEmail(header string) []string {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return nil
	}

	addresses, err := mail.ParseAddressList(trimmed)
	if err == nil && len(addresses) > 0 {
		result := make([]string, 0, len(addresses))
		for _, a := range addresses {
			clean := strings.ToLower(strings.TrimSpace(a.Address))
			if clean != "" {
				result = append(result, clean)
			}
		}
		return result
	}

	// Fallback: split by comma
	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		clean, _ := parseSingleEmail(part)
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

// Helper: parse flexible email dates
func parseEmailDate(val string) time.Time {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Time{}
	}

	// Common RFC formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 +0000",
		"2006-01-02 15:04:05 -0700",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, val); err == nil {
			return t
		}
	}

	return time.Time{}
}

// Helper: check if error is Google quota or rate limit error
func isQuotaOrRateLimitErr(err error) bool {
	if err == nil {
		return false
	}
	if gErr, ok := err.(*googleapi.Error); ok {
		if gErr.Code == 429 || gErr.Code == 403 || gErr.Code >= 500 {
			return true
		}
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "quota")
}

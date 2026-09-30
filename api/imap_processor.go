package api

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
)

// xoauth2Client implements sasl.Client for the XOAUTH2 mechanism
type xoauth2Client struct {
	Username string
	Token    string
}

// NewXOAuth2Client creates an XOAUTH2 SASL client
func NewXOAuth2Client(username, token string) sasl.Client {
	return &xoauth2Client{
		Username: username,
		Token:    token,
	}
}

func (a *xoauth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.Token))
	return mech, ir, nil
}

func (a *xoauth2Client) Next(challenge []byte) ([]byte, error) {
	// In the XOAUTH2 protocol, if the server issues a challenge (which contains an error JSON),
	// the client must send an empty response so the server completes the rejection.
	return []byte(""), nil
}

// connectIMAP dials Gmail IMAP over TLS and authenticates with XOAUTH2
func (p *InboxProcessor) connectIMAP(ctx context.Context) (*client.Client, error) {
	// Obtain a valid (refreshed if needed) OAuth token
	tokenSource := oauthConfig.TokenSource(ctx, p.oauthToken)
	tok, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	p.mu.Lock()
	p.oauthToken = tok
	p.mu.Unlock()

	// Dial with timeout
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	c, err := client.DialWithDialerTLS(dialer, "imap.gmail.com:993", nil)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to imap.gmail.com:993: %w", err)
	}

	c.Timeout = 30 * time.Second

	// Authenticate via XOAUTH2
	saslAuth := NewXOAuth2Client(p.userEmail, tok.AccessToken)
	if err := c.Authenticate(saslAuth); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("XOAUTH2 authentication failed: %w", err)
	}

	return c, nil
}

// findGmailAllMailFolder discovers the All Mail folder using IMAP special-use attributes
func findGmailAllMailFolder(c *client.Client) (string, error) {
	mailboxes := make(chan *imap.MailboxInfo, 50)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var allMailFolder string
	var fallbackAllMail string

	for mbox := range mailboxes {
		for _, attr := range mbox.Attributes {
			if attr == `\All` || strings.EqualFold(attr, `\AllMail`) {
				allMailFolder = mbox.Name
			}
		}
		nameLower := strings.ToLower(mbox.Name)
		if strings.Contains(nameLower, "all mail") || strings.Contains(nameLower, "tous les messages") {
			fallbackAllMail = mbox.Name
		}
	}

	if err := <-done; err != nil {
		return "", fmt.Errorf("failed listing IMAP folders: %w", err)
	}

	if allMailFolder != "" {
		return allMailFolder, nil
	}
	if fallbackAllMail != "" {
		return fallbackAllMail, nil
	}
	return "INBOX", nil
}

// runImapScan scans the user's Gmail mailbox using IMAP with XOAUTH2
func (p *InboxProcessor) runImapScan(ctx context.Context) error {
	p.updateStatus("Connecting to Gmail IMAP via XOAUTH2...")

	c, err := p.connectIMAP(ctx)
	if err != nil {
		return err
	}
	defer c.Logout()

	// Find the comprehensive "All Mail" folder
	folderName, err := findGmailAllMailFolder(c)
	if err != nil {
		return err
	}
	log.Printf("[%s] Selected IMAP folder: %s", p.userID, folderName)

	// STRICT SAFETY: Open mailbox in read-only mode (IMAP EXAMINE command)
	// This guarantees that no flags or messages can be altered or deleted.
	if _, err := c.Select(folderName, true); err != nil {
		return fmt.Errorf("failed selecting folder %s in read-only mode: %w", folderName, err)
	}

	// Search for all message UIDs in the folder
	criteria := imap.NewSearchCriteria()
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return fmt.Errorf("failed searching message UIDs: %w", err)
	}

	totalUIDs := len(uids)
	log.Printf("[%s] Found %d messages in IMAP folder %s", p.userID, totalUIDs, folderName)

	p.mu.Lock()
	p.estimatedTotal = int64(totalUIDs)
	p.scanState.EstimatedTotal = int64(totalUIDs)
	_ = DB.SaveScanState(p.scanState)
	p.mu.Unlock()

	if totalUIDs == 0 {
		p.updateStatus("Scan complete. Mailbox is empty.")
		return nil
	}

	// Check for resumption offset
	startIdx := 0
	p.mu.RLock()
	pageToken := p.scanState.NextPageToken
	p.mu.RUnlock()

	if strings.HasPrefix(pageToken, "imap:") {
		if offset, err := strconv.Atoi(strings.TrimPrefix(pageToken, "imap:")); err == nil && offset >= 0 && offset < totalUIDs {
			startIdx = offset
			log.Printf("[%s] Resuming IMAP scan from offset %d of %d", p.userID, startIdx, totalUIDs)
		}
	}

	batchSize := 250 // Fetch 250 envelopes per batch for rapid, bounded network throughput
	fetchItems := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchUid,
		imap.FetchRFC822Size,
		imap.FetchInternalDate,
		imap.FetchFlags,
		imap.FetchItem("X-GM-MSGID"),
		imap.FetchItem("X-GM-THRID"),
		imap.FetchItem("X-GM-LABELS"),
	}

	for startIdx < totalUIDs {
		select {
		case <-ctx.Done():
			log.Printf("[%s] IMAP scan paused by context at index %d", p.userID, startIdx)
			p.updateStatus("Scan paused")
			return ctx.Err()
		default:
		}

		endIdx := startIdx + batchSize
		if endIdx > totalUIDs {
			endIdx = totalUIDs
		}

		batchUIDs := uids[startIdx:endIdx]
		seqset := new(imap.SeqSet)
		seqset.AddNum(batchUIDs...)

		messagesChan := make(chan *imap.Message, len(batchUIDs))
		doneFetch := make(chan error, 1)

		go func() {
			doneFetch <- c.UidFetch(seqset, fetchItems, messagesChan)
		}()

		var batchEmails []EmailMetadata
		for msg := range messagesChan {
			if msg == nil {
				continue
			}

			meta := p.convertImapMessageToMetadata(msg)
			if meta != nil {
				batchEmails = append(batchEmails, *meta)
			}
		}

		if err := <-doneFetch; err != nil {
			log.Printf("[%s] Error fetching IMAP batch at offset %d: %v", p.userID, startIdx, err)
			return fmt.Errorf("UID FETCH failed at offset %d: %w", startIdx, err)
		}

		// Persist batch into SQLite
		if len(batchEmails) > 0 {
			if err := DB.SaveEmailsBatch(p.userID, batchEmails); err != nil {
				log.Printf("[%s] Error saving IMAP emails batch: %v", p.userID, err)
				p.updateStatus(fmt.Sprintf("Error saving emails: %v", err))
				return err
			}
		}

		startIdx = endIdx

		p.mu.Lock()
		p.scanState.TotalEmails += len(batchEmails)
		p.scanState.NextPageToken = fmt.Sprintf("imap:%d", startIdx)
		if p.estimatedTotal > 0 {
			pct := int(float64(startIdx) / float64(p.estimatedTotal) * 100)
			if pct > 100 {
				pct = 100
			}
			p.scanState.StatusMessage = fmt.Sprintf("Scanned %d of %d emails (%d%%) via IMAP", startIdx, p.estimatedTotal, pct)
		} else {
			p.scanState.StatusMessage = fmt.Sprintf("Scanned %d emails via IMAP...", p.scanState.TotalEmails)
		}
		_ = DB.SaveScanState(p.scanState)
		p.mu.Unlock()

		log.Printf("[%s] IMAP batch processed (%d/%d messages)", p.userID, startIdx, totalUIDs)
	}

	p.mu.Lock()
	p.scanState.NextPageToken = ""
	p.scanState.StatusMessage = fmt.Sprintf("Scan completed! Processed %d emails via IMAP.", p.scanState.TotalEmails)
	_ = DB.SaveScanState(p.scanState)
	p.mu.Unlock()

	return nil
}

// convertImapMessageToMetadata extracts EmailMetadata from an IMAP message
func (p *InboxProcessor) convertImapMessageToMetadata(msg *imap.Message) *EmailMetadata {
	if msg == nil {
		return nil
	}

	// Extract flags and labels
	var labels []string
	if rawLabels, ok := msg.Items[imap.FetchItem("X-GM-LABELS")]; ok {
		labels = extractXGmLabels(rawLabels)
	}

	// Skip drafts
	if hasDraftFlag(msg.Flags, labels) {
		return nil
	}

	// Gmail REST-compatible message and thread IDs
	var msgID string
	if rawMsgID, ok := msg.Items[imap.FetchItem("X-GM-MSGID")]; ok {
		msgID = extractXGmID(rawMsgID)
	}
	if msgID == "" {
		msgID = fmt.Sprintf("imap-%d", msg.Uid)
	}

	var threadID string
	if rawThrid, ok := msg.Items[imap.FetchItem("X-GM-THRID")]; ok {
		threadID = extractXGmID(rawThrid)
	}
	if threadID == "" {
		threadID = msgID
	}

	var fromAddr, fromName, subject string
	var toAddrs []string
	var date time.Time

	if env := msg.Envelope; env != nil {
		subject = decodeMIMEHeader(env.Subject)
		date = env.Date

		if len(env.From) > 0 && env.From[0] != nil {
			fromAddr = strings.ToLower(strings.TrimSpace(env.From[0].Address()))
			fromName = decodeMIMEHeader(env.From[0].PersonalName)
		}

		for _, addr := range env.To {
			if addr != nil {
				clean := strings.ToLower(strings.TrimSpace(addr.Address()))
				if clean != "" {
					toAddrs = append(toAddrs, clean)
				}
			}
		}

		for _, addr := range env.Cc {
			if addr != nil {
				clean := strings.ToLower(strings.TrimSpace(addr.Address()))
				if clean != "" {
					toAddrs = append(toAddrs, clean)
				}
			}
		}
	}

	if date.IsZero() && !msg.InternalDate.IsZero() {
		date = msg.InternalDate
	}

	// Determine if sent by user
	isSent := false
	if strings.EqualFold(fromAddr, p.userEmail) {
		isSent = true
	} else {
		for _, lbl := range labels {
			if strings.EqualFold(lbl, `\Sent`) || strings.EqualFold(lbl, "Sent") {
				isSent = true
				break
			}
		}
	}

	return &EmailMetadata{
		ID:           msgID,
		ThreadID:     threadID,
		From:         fromAddr,
		FromName:     fromName,
		To:           toAddrs,
		Subject:      subject,
		Date:         date,
		SizeEstimate: int64(msg.Size),
		LabelIDs:     labels,
		IsSent:       isSent,
	}
}

// decodeMIMEHeader handles RFC 2047 encoded words
func decodeMIMEHeader(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(val)
	if err == nil {
		return strings.TrimSpace(decoded)
	}
	return val
}

// extractXGmID converts Gmail 64-bit uint identifiers into standard hex strings
func extractXGmID(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case uint64:
		return fmt.Sprintf("%x", v)
	case int64:
		return fmt.Sprintf("%x", uint64(v))
	case uint32:
		return fmt.Sprintf("%x", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// extractXGmLabels extracts string slice from IMAP X-GM-LABELS response
func extractXGmLabels(val interface{}) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []interface{}:
		var labels []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				labels = append(labels, s)
			}
		}
		return labels
	case []string:
		return v
	default:
		return nil
	}
}

// hasDraftFlag checks if a message is flagged or labeled as a draft
func hasDraftFlag(flags []string, labels []string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, imap.DraftFlag) || strings.EqualFold(f, `\Draft`) {
			return true
		}
	}
	for _, l := range labels {
		if strings.EqualFold(l, `\Draft`) || strings.EqualFold(l, "Draft") || strings.EqualFold(l, "DRAFT") {
			return true
		}
	}
	return false
}

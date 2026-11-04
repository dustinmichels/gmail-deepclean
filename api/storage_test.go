package api

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func setupTestDB(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_deepclean.db")
	storage, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
	t.Cleanup(func() {
		_ = storage.Close()
	})
	return storage
}

func TestCSRFStateLifecycle(t *testing.T) {
	storage := setupTestDB(t)

	state := "test-csrf-random-state"
	if err := storage.SaveCSRFState(state, 5*time.Minute); err != nil {
		t.Fatalf("SaveCSRFState failed: %v", err)
	}

	// Validate and consume once
	valid, err := storage.ValidateAndConsumeCSRFState(state)
	if err != nil {
		t.Fatalf("ValidateAndConsumeCSRFState failed: %v", err)
	}
	if !valid {
		t.Errorf("Expected state to be valid")
	}

	// Second attempt should fail (single-use token)
	validSecond, err := storage.ValidateAndConsumeCSRFState(state)
	if err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}
	if validSecond {
		t.Errorf("Expected consumed state to be invalid on second attempt")
	}
}

func TestUserAndSessionManagement(t *testing.T) {
	storage := setupTestDB(t)

	userID := "user@gmail.com"
	userEmail := "user@gmail.com"

	if err := storage.UpsertUser(userID, userEmail); err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	token := &oauth2.Token{
		AccessToken:  "access-12345",
		TokenType:    "Bearer",
		RefreshToken: "refresh-67890",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	sessionToken := "session-token-abc"
	if err := storage.CreateSession(sessionToken, userID, userEmail, token, 24*time.Hour); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	sess, err := storage.GetSession(sessionToken)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess == nil {
		t.Fatalf("Expected session, got nil")
	}
	if sess.UserEmail != userEmail || sess.UserID != userID {
		t.Errorf("Session mismatch: got email %s, id %s", sess.UserEmail, sess.UserID)
	}
	if sess.OAuthToken.AccessToken != "access-12345" || sess.OAuthToken.RefreshToken != "refresh-67890" {
		t.Errorf("OAuth token mismatch: %+v", sess.OAuthToken)
	}

	// Update token
	newToken := &oauth2.Token{
		AccessToken:  "access-new",
		TokenType:    "Bearer",
		RefreshToken: "refresh-67890",
		Expiry:       time.Now().Add(2 * time.Hour),
	}
	if err := storage.UpdateSessionOAuthToken(sessionToken, newToken); err != nil {
		t.Fatalf("UpdateSessionOAuthToken failed: %v", err)
	}

	sessUpdated, _ := storage.GetSession(sessionToken)
	if sessUpdated.OAuthToken.AccessToken != "access-new" {
		t.Errorf("Expected updated access token 'access-new', got '%s'", sessUpdated.OAuthToken.AccessToken)
	}

	// Delete session
	if err := storage.DeleteSession(sessionToken); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	sessDeleted, _ := storage.GetSession(sessionToken)
	if sessDeleted != nil {
		t.Errorf("Expected nil session after delete, got %+v", sessDeleted)
	}
}

func TestEmailBatchAndContactStats(t *testing.T) {
	storage := setupTestDB(t)

	userID := "me@test.com"
	now := time.Now()

	emails := []EmailMetadata{
		// 1. Received from unknown newsletter (never replied)
		{
			ID:           "msg-1",
			ThreadID:     "th-1",
			From:         "newsletter@marketing.com",
			FromName:     "Weekly Deals",
			To:           []string{"me@test.com"},
			Subject:      "Huge Sales This Weekend!",
			Snippet:      "Save up to 50% on all items...",
			Date:         now.Add(-48 * time.Hour),
			SizeEstimate: 15000,
			IsSent:       false,
		},
		// 2. Received from another stranger (never replied)
		{
			ID:           "msg-2",
			ThreadID:     "th-2",
			From:         "stranger@coldreach.io",
			FromName:     "Alex Sales",
			To:           []string{"me@test.com"},
			Subject:      "Quick question about your tech stack",
			Snippet:      "Hi there, wanted to follow up...",
			Date:         now.Add(-24 * time.Hour),
			SizeEstimate: 4500,
			IsSent:       false,
		},
		// 3. Received from friend Alice
		{
			ID:           "msg-3",
			ThreadID:     "th-3",
			From:         "alice@friend.org",
			FromName:     "Alice Smith",
			To:           []string{"me@test.com"},
			Subject:      "Coffee next week?",
			Snippet:      "Hey! Let's grab lunch or coffee...",
			Date:         now.Add(-10 * time.Hour),
			SizeEstimate: 3200,
			IsSent:       false,
		},
		// 4. Sent reply to friend Alice
		{
			ID:           "msg-4",
			ThreadID:     "th-3",
			From:         "me@test.com",
			FromName:     "Me",
			To:           []string{"alice@friend.org"},
			Subject:      "Re: Coffee next week?",
			Snippet:      "Sounds great, Tuesday works...",
			Date:         now.Add(-5 * time.Hour),
			SizeEstimate: 2100,
			IsSent:       true,
		},
		// 5. Sent email to Bob
		{
			ID:           "msg-5",
			ThreadID:     "th-4",
			From:         "me@test.com",
			FromName:     "Me",
			To:           []string{"bob@work.com"},
			Subject:      "Q3 Report Attached",
			Snippet:      "Please review the attached spreadsheet...",
			Date:         now.Add(-2 * time.Hour),
			SizeEstimate: 8000,
			IsSent:       true,
		},
	}

	if err := storage.SaveEmailsBatch(userID, emails); err != nil {
		t.Fatalf("SaveEmailsBatch failed: %v", err)
	}

	// Verify Overall Stats
	overall, err := storage.GetOverallStats(userID)
	if err != nil {
		t.Fatalf("GetOverallStats failed: %v", err)
	}
	if overall.TotalEmailsProcessed != 5 {
		t.Errorf("Expected 5 total emails, got %d", overall.TotalEmailsProcessed)
	}
	if overall.TotalReceived != 3 {
		t.Errorf("Expected 3 received emails, got %d", overall.TotalReceived)
	}
	if overall.TotalSent != 2 {
		t.Errorf("Expected 2 sent emails, got %d", overall.TotalSent)
	}
	if overall.UniqueSenders != 3 {
		t.Errorf("Expected 3 unique senders, got %d", overall.UniqueSenders)
	}
	if overall.TotalSizeBytes != 32800 {
		t.Errorf("Expected 32800 total bytes across all mail, got %d", overall.TotalSizeBytes)
	}
	// Alice and Bob were emailed; newsletter and stranger were NEVER emailed
	if overall.NeverEmailedSenders != 2 {
		t.Errorf("Expected 2 never-emailed senders, got %d", overall.NeverEmailedSenders)
	}
	if overall.NeverEmailedEmails != 2 {
		t.Errorf("Expected 2 emails from never-emailed senders, got %d", overall.NeverEmailedEmails)
	}
	if overall.NeverEmailedSizeBytes != 19500 {
		t.Errorf("Expected 19500 bytes from never-emailed senders, got %d", overall.NeverEmailedSizeBytes)
	}
	if overall.LocalDbSizeBytes <= 0 {
		t.Errorf("Expected positive LocalDbSizeBytes, got %d", overall.LocalDbSizeBytes)
	}

	// Contacts list only ever contains never-emailed senders
	contacts, count, err := storage.GetContactStats(userID, "", "received_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("GetContactStats failed: %v", err)
	}
	if count != 2 || len(contacts) != 2 {
		t.Fatalf("Expected 2 never-emailed contacts, got count %d, len %d", count, len(contacts))
	}
	for _, c := range contacts {
		if c.Email == "alice@friend.org" || c.Email == "bob@work.com" {
			t.Errorf("Emailed contact %s must not be listed", c.Email)
		}
	}
	if _, aliceCount, err := storage.GetContactStats(userID, "alice", "received_count", "desc", 10, 0); err != nil || aliceCount != 0 {
		t.Errorf("Expected search for emailed contact to return 0, got %d (err %v)", aliceCount, err)
	}

	// Test GetEmails filter by sender
	senderEmails, total, err := storage.GetEmails(userID, "newsletter@marketing.com", "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails failed: %v", err)
	}
	if total != 1 || len(senderEmails) != 1 {
		t.Fatalf("Expected 1 email from newsletter, got %d", total)
	}
	if senderEmails[0].Subject != "Huge Sales This Weekend!" {
		t.Errorf("Unexpected subject: %s", senderEmails[0].Subject)
	}
	if senderEmails[0].SenderEmailCount != 1 {
		t.Errorf("Expected SenderEmailCount to be 1, got %d", senderEmails[0].SenderEmailCount)
	}

	// GetEmails never returns sent mail or mail from emailed contacts
	allEmails, allTotal, err := storage.GetEmails(userID, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails failed: %v", err)
	}
	if allTotal != 2 || len(allEmails) != 2 {
		t.Errorf("Expected 2 emails from never-emailed senders, got total=%d len=%d", allTotal, len(allEmails))
	}
	if _, aliceTotal, err := storage.GetEmails(userID, "alice@friend.org", "", "", 10, 0); err != nil || aliceTotal != 0 {
		t.Errorf("Expected 0 emails from emailed contact, got %d (err %v)", aliceTotal, err)
	}

	// Test GetEmailByID
	msg1, err := storage.GetEmailByID(userID, "msg-1")
	if err != nil {
		t.Fatalf("GetEmailByID failed: %v", err)
	}
	if msg1 == nil || msg1.ID != "msg-1" {
		t.Errorf("Expected msg-1, got %+v", msg1)
	}
	for _, hiddenID := range []string{"msg-3", "msg-4"} {
		hidden, err := storage.GetEmailByID(userID, hiddenID)
		if err != nil {
			t.Fatalf("GetEmailByID %s failed: %v", hiddenID, err)
		}
		if hidden != nil {
			t.Errorf("Expected %s (contact or sent mail) to be hidden, got %+v", hiddenID, hidden)
		}
	}
}

func TestScanStatePersistence(t *testing.T) {
	storage := setupTestDB(t)
	userID := "user@gmail.com"

	// Initial state when empty
	initial, err := storage.GetScanState(userID)
	if err != nil {
		t.Fatalf("GetScanState failed: %v", err)
	}
	if initial.TotalEmails != 0 || initial.IsProcessing {
		t.Errorf("Expected default empty scan state, got %+v", initial)
	}

	// Update state
	now := time.Now()
	state := &ScanState{
		UserID:        userID,
		IsProcessing:  true,
		TotalEmails:   150,
		NextPageToken: "token-page-2",
		StatusMessage: "Scanned 150 emails...",
		LastScannedAt: &now,
	}
	if err := storage.SaveScanState(state); err != nil {
		t.Fatalf("SaveScanState failed: %v", err)
	}

	// Retrieve updated state
	loaded, err := storage.GetScanState(userID)
	if err != nil {
		t.Fatalf("GetScanState failed: %v", err)
	}
	if loaded.TotalEmails != 150 || !loaded.IsProcessing || loaded.NextPageToken != "token-page-2" {
		t.Errorf("Loaded state mismatch: %+v", loaded)
	}
}

func TestSaveEmailsBatchIdempotency(t *testing.T) {
	storage := setupTestDB(t)
	userID := "user@example.com"
	now := time.Now()

	batch := []EmailMetadata{
		{
			ID:           "idempotent-1",
			ThreadID:     "th-1",
			From:         "sender@promo.com",
			FromName:     "Promo",
			To:           []string{"user@example.com"},
			Subject:      "Discount Inside",
			Date:         now.Add(-2 * time.Hour),
			SizeEstimate: 1000,
			IsSent:       false,
		},
		{
			ID:           "idempotent-2",
			ThreadID:     "th-2",
			From:         "user@example.com",
			FromName:     "User",
			To:           []string{"colleague@work.com"},
			Subject:      "Project Plan",
			Date:         now.Add(-1 * time.Hour),
			SizeEstimate: 2500,
			IsSent:       true,
		},
		{
			ID:           "idempotent-draft",
			ThreadID:     "th-3",
			From:         "user@example.com",
			FromName:     "User",
			To:           []string{"never-sent@target.com"},
			Subject:      "Unsent Draft",
			Date:         now,
			SizeEstimate: 500,
			LabelIDs:     []string{"DRAFT"},
			IsSent:       true,
		},
	}

	// First insert
	if err := storage.SaveEmailsBatch(userID, batch); err != nil {
		t.Fatalf("First SaveEmailsBatch failed: %v", err)
	}

	overallFirst, err := storage.GetOverallStats(userID)
	if err != nil {
		t.Fatalf("GetOverallStats failed: %v", err)
	}
	contactsFirst, totalContactsFirst, err := storage.GetContactStats(userID, "", "received_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("GetContactStats failed: %v", err)
	}
	_, emailCountFirst, err := storage.GetEmails(userID, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails failed: %v", err)
	}

	// Verify draft was ignored
	if overallFirst.TotalEmailsProcessed != 2 {
		t.Errorf("Expected 2 emails processed (draft ignored), got %d", overallFirst.TotalEmailsProcessed)
	}

	// Second insert with the identical batch (simulating Rescan All or rerun)
	if err := storage.SaveEmailsBatch(userID, batch); err != nil {
		t.Fatalf("Second SaveEmailsBatch failed: %v", err)
	}

	overallSecond, err := storage.GetOverallStats(userID)
	if err != nil {
		t.Fatalf("Second GetOverallStats failed: %v", err)
	}
	contactsSecond, totalContactsSecond, err := storage.GetContactStats(userID, "", "received_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("Second GetContactStats failed: %v", err)
	}
	_, emailCountSecond, err := storage.GetEmails(userID, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("Second GetEmails failed: %v", err)
	}

	// Assert overall stats did not change
	if overallFirst.TotalEmailsProcessed != overallSecond.TotalEmailsProcessed {
		t.Errorf("TotalEmailsProcessed inflated: first=%d, second=%d", overallFirst.TotalEmailsProcessed, overallSecond.TotalEmailsProcessed)
	}
	if overallFirst.TotalReceived != overallSecond.TotalReceived {
		t.Errorf("TotalReceived inflated: first=%d, second=%d", overallFirst.TotalReceived, overallSecond.TotalReceived)
	}
	if overallFirst.TotalSent != overallSecond.TotalSent {
		t.Errorf("TotalSent inflated: first=%d, second=%d", overallFirst.TotalSent, overallSecond.TotalSent)
	}
	if overallFirst.NeverEmailedSenders != overallSecond.NeverEmailedSenders {
		t.Errorf("NeverEmailedSenders changed: first=%d, second=%d", overallFirst.NeverEmailedSenders, overallSecond.NeverEmailedSenders)
	}
	if overallFirst.NeverEmailedEmails != overallSecond.NeverEmailedEmails {
		t.Errorf("NeverEmailedEmails inflated: first=%d, second=%d", overallFirst.NeverEmailedEmails, overallSecond.NeverEmailedEmails)
	}
	if emailCountFirst != emailCountSecond {
		t.Errorf("Email count inflated: first=%d, second=%d", emailCountFirst, emailCountSecond)
	}
	if totalContactsFirst != totalContactsSecond {
		t.Errorf("Total contacts inflated: first=%d, second=%d", totalContactsFirst, totalContactsSecond)
	}

	// Assert individual contact stats did not change
	if len(contactsFirst) != len(contactsSecond) {
		t.Fatalf("Contact list length changed: first=%d, second=%d", len(contactsFirst), len(contactsSecond))
	}
	for i := range contactsFirst {
		c1 := contactsFirst[i]
		c2 := contactsSecond[i]
		if c1.Email != c2.Email {
			t.Errorf("Contact[%d] email mismatch: %s vs %s", i, c1.Email, c2.Email)
		}
		if c1.ReceivedCount != c2.ReceivedCount {
			t.Errorf("Contact %s ReceivedCount inflated: first=%d, second=%d", c1.Email, c1.ReceivedCount, c2.ReceivedCount)
		}
		if c1.TotalSize != c2.TotalSize {
			t.Errorf("Contact %s TotalSize inflated: first=%d, second=%d", c1.Email, c1.TotalSize, c2.TotalSize)
		}
	}
}

func TestGetContactStatsLegacyTimestamps(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	userID := "test-legacy-user"

	// Insert contact_stats with varied legacy timestamp strings (e.g. Go time.String() with repeated offsets or timezones)
	_, err := storage.db.Exec(`
		INSERT INTO contact_stats (
			user_id, email, name, received_count, sent_count, total_size,
			first_received, last_received
		) VALUES (
			?, 'legacy@example.com', 'Legacy Contact', 10, 0, 2048,
			'2024-06-19 10:52:18 -0700 -0700',
			'2026-09-12 11:16:37 -0400 EDT'
		)
	`, userID)
	if err != nil {
		t.Fatalf("Failed inserting legacy contact stats: %v", err)
	}

	contacts, total, err := storage.GetContactStats(userID, "", "received_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("GetContactStats failed on legacy timestamps: %v", err)
	}
	if total != 1 || len(contacts) != 1 {
		t.Fatalf("Expected 1 contact, got total=%d, len=%d", total, len(contacts))
	}

	c := contacts[0]
	if c.Email != "legacy@example.com" {
		t.Errorf("Expected email legacy@example.com, got %s", c.Email)
	}
	expectedFirstReceivedUTC := time.Date(2024, 6, 19, 17, 52, 18, 0, time.UTC)
	if c.FirstReceived == nil || !c.FirstReceived.UTC().Equal(expectedFirstReceivedUTC) {
		t.Errorf("Unexpected FirstReceived UTC: got %v, expected %v", c.FirstReceived, expectedFirstReceivedUTC)
	}
	if c.LastReceived == nil || c.LastReceived.Year() != 2026 || c.LastReceived.Month() != 9 || c.LastReceived.Day() != 12 {
		t.Errorf("Unexpected LastReceived: %v", c.LastReceived)
	}
}

func TestGetContactStatsCorruptTimestamp(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	userID := "test-corrupt-user"
	_, err := storage.db.Exec(`
		INSERT INTO contact_stats (
			user_id, email, name, received_count, sent_count, total_size, first_received
		) VALUES (
			?, 'badtime@example.com', 'Bad Time', 5, 0, 1024, 'completely-invalid-date'
		)
	`, userID)
	if err != nil {
		t.Fatalf("Failed inserting corrupt contact stat: %v", err)
	}

	_, _, err = storage.GetContactStats(userID, "", "received_count", "desc", 10, 0)
	if err == nil {
		t.Fatal("Expected error on corrupt timestamp, got nil")
	}
	if !strings.Contains(err.Error(), "badtime@example.com") || !strings.Contains(err.Error(), "first_received") {
		t.Errorf("Expected error to contain contact email and field name, got: %v", err)
	}
}
func TestGetEmailsLegacyTimestamps(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	userID := "test-emails-legacy-user"

	_, err := storage.db.Exec(`
		INSERT INTO emails (
			id, user_id, thread_id, from_address, from_name, to_addresses,
			subject, snippet, date, size_estimate, label_ids, is_sent, created_at
		) VALUES (
			'msg-legacy-1', ?, 'th-1', 'notification+pvl16c6f@facebookmail.com', 'Facebook', '["me@example.com"]',
			'Facebook notification', 'You have a notification', '2009-08-01 16:01:23 -0700 -0700', 4096, '["INBOX"]', 0, '2026-09-29 12:00:00'
		), (
			'msg-legacy-2', ?, 'th-2', 'notification+pvl16c6f@facebookmail.com', 'Facebook', '["me@example.com"]',
			'Another notification', 'Another update', '2026-09-12 11:16:37 -0400 EDT', 2048, '["INBOX"]', 0, '2026-09-29 12:00:00'
		)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed inserting emails with legacy timestamps: %v", err)
	}
	_, err = storage.db.Exec(`
		INSERT INTO contact_stats (user_id, email, name, received_count, sent_count, total_size)
		VALUES (?, 'notification+pvl16c6f@facebookmail.com', 'Facebook', 2, 0, 6144)
	`, userID)
	if err != nil {
		t.Fatalf("Failed inserting contact stats: %v", err)
	}

	// 1. Test GetEmails by sender
	emails, total, err := storage.GetEmails(userID, "notification+pvl16c6f@facebookmail.com", "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails failed with legacy timestamps: %v", err)
	}
	if total != 2 || len(emails) != 2 {
		t.Fatalf("Expected 2 emails, got total=%d, len=%d", total, len(emails))
	}

	// Check dates are parsed
	for _, em := range emails {
		if em.Date.IsZero() {
			t.Errorf("Expected non-zero Date for email %s", em.ID)
		}
	}

	// 2. Test GetEmailByID
	em1, err := storage.GetEmailByID(userID, "msg-legacy-1")
	if err != nil {
		t.Fatalf("GetEmailByID failed: %v", err)
	}
	if em1 == nil {
		t.Fatal("Expected email msg-legacy-1, got nil")
	}
	expectedDate1 := time.Date(2009, 8, 1, 16, 1, 23, 0, time.FixedZone("", -7*3600))
	if !em1.Date.Equal(expectedDate1) {
		t.Errorf("Expected Date %v, got %v", expectedDate1, em1.Date)
	}
}

func TestCandidateEmailsAndSenders(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	userID := "candidate-test-user"
	now := time.Now()

	emails := []EmailMetadata{
		// 1. Candidate sender: never replied, older than 1 month (45 days)
		{
			ID:           "cand-1",
			ThreadID:     "th-c1",
			From:         "old-promo@marketing.com",
			FromName:     "Promo Blast",
			To:           []string{"me@test.com"},
			Subject:      "Summer Discounts",
			Snippet:      "Save 30% today only...",
			Date:         now.AddDate(0, 0, -45),
			SizeEstimate: 1000,
			IsSent:       false,
		},
		// 2. Candidate sender: second email older than 1 month (60 days)
		{
			ID:           "cand-2",
			ThreadID:     "th-c2",
			From:         "old-promo@marketing.com",
			FromName:     "Promo Blast",
			To:           []string{"me@test.com"},
			Subject:      "Spring Clearance",
			Snippet:      "Clearance sale starts...",
			Date:         now.AddDate(0, 0, -60),
			SizeEstimate: 2000,
			IsSent:       false,
		},
		// 3. Recent sender: never replied, but only 5 days old (< 1 month) -> NOT a candidate
		{
			ID:           "recent-1",
			ThreadID:     "th-r1",
			From:         "recent-spammer@spam.io",
			FromName:     "Spam Team",
			To:           []string{"me@test.com"},
			Subject:      "Just signed up?",
			Snippet:      "Welcome to our service...",
			Date:         now.AddDate(0, 0, -5),
			SizeEstimate: 1500,
			IsSent:       false,
		},
		// 4. Emailed back sender: received 60 days ago, but user replied 50 days ago -> NOT candidate
		{
			ID:           "replied-recv",
			ThreadID:     "th-rep",
			From:         "colleague@work.com",
			FromName:     "Colleague",
			To:           []string{"me@test.com"},
			Subject:      "Old Project Question",
			Snippet:      "Can you send the docs?",
			Date:         now.AddDate(0, 0, -60),
			SizeEstimate: 3000,
			IsSent:       false,
		},
		{
			ID:           "replied-sent",
			ThreadID:     "th-rep",
			From:         "me@test.com",
			FromName:     "Me",
			To:           []string{"colleague@work.com"},
			Subject:      "Re: Old Project Question",
			Snippet:      "Attached docs here.",
			Date:         now.AddDate(0, 0, -50),
			SizeEstimate: 4000,
			IsSent:       true,
		},
	}

	if err := storage.SaveEmailsBatch(userID, emails); err != nil {
		t.Fatalf("SaveEmailsBatch failed: %v", err)
	}

	// 1. Verify OverallStats candidate metrics
	stats, err := storage.GetOverallStats(userID)
	if err != nil {
		t.Fatalf("GetOverallStats failed: %v", err)
	}
	if stats.CandidateEmailsCount != 2 {
		t.Errorf("Expected 2 candidate emails, got %d", stats.CandidateEmailsCount)
	}
	if stats.CandidateSizeBytes != 3000 {
		t.Errorf("Expected 3000 candidate size bytes, got %d", stats.CandidateSizeBytes)
	}
	if stats.CandidateSendersCount != 1 {
		t.Errorf("Expected 1 candidate sender, got %d", stats.CandidateSendersCount)
	}

	// 2. Verify GetCandidateSenders
	senders, err := storage.GetCandidateSenders(userID)
	if err != nil {
		t.Fatalf("GetCandidateSenders failed: %v", err)
	}
	if len(senders) != 1 {
		t.Fatalf("Expected 1 candidate sender, got %d", len(senders))
	}
	if senders[0].Email != "old-promo@marketing.com" {
		t.Errorf("Expected old-promo@marketing.com, got %s", senders[0].Email)
	}
	if senders[0].CandidateCount != 2 {
		t.Errorf("Expected 2 candidate count for sender, got %d", senders[0].CandidateCount)
	}
	if senders[0].CandidateSize != 3000 {
		t.Errorf("Expected 3000 candidate size for sender, got %d", senders[0].CandidateSize)
	}

	// 3. Verify GetEmails with filter="candidates"
	candEmails, totalCand, err := storage.GetEmails(userID, "", "", "candidates", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails candidates failed: %v", err)
	}
	if totalCand != 2 || len(candEmails) != 2 {
		t.Errorf("Expected 2 candidate emails, got total=%d len=%d", totalCand, len(candEmails))
	}

	// 4. Verify GetEmails with filter="candidates" and sender filter
	senderEmails, totalSenderCand, err := storage.GetEmails(userID, "old-promo@marketing.com", "", "candidates", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails with sender filter failed: %v", err)
	}
	if totalSenderCand != 2 || len(senderEmails) != 2 {
		t.Errorf("Expected 2 emails from old-promo, got total=%d len=%d", totalSenderCand, len(senderEmails))
	}

	// Non-candidate sender should return 0 with filter="candidates"
	recentEmails, totalRecentCand, err := storage.GetEmails(userID, "recent-spammer@spam.io", "", "candidates", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails for recent sender failed: %v", err)
	}
	if totalRecentCand != 0 || len(recentEmails) != 0 {
		t.Errorf("Expected 0 candidate emails for recent sender, got total=%d len=%d", totalRecentCand, len(recentEmails))
	}
}

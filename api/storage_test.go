package api

import (
	"path/filepath"
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
	// Alice was replied to; newsletter and stranger were NEVER emailed back!
	if overall.NeverEmailedCount != 2 {
		t.Errorf("Expected 2 never-emailed senders, got %d", overall.NeverEmailedCount)
	}

	// Verify "never_emailed" filter
	neverEmailedList, count, err := storage.GetContactStats(userID, "never_emailed", "", "received_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("GetContactStats never_emailed failed: %v", err)
	}
	if count != 2 || len(neverEmailedList) != 2 {
		t.Fatalf("Expected 2 never_emailed contacts, got count %d, len %d", count, len(neverEmailedList))
	}
	for _, c := range neverEmailedList {
		if !c.NeverEmailed {
			t.Errorf("Expected c.NeverEmailed == true for %s", c.Email)
		}
		if c.SentCount != 0 {
			t.Errorf("Expected 0 sent count for %s, got %d", c.Email, c.SentCount)
		}
	}

	// Verify "emailed" filter
	emailedList, countEmailed, err := storage.GetContactStats(userID, "emailed", "", "sent_count", "desc", 10, 0)
	if err != nil {
		t.Fatalf("GetContactStats emailed failed: %v", err)
	}
	if countEmailed != 2 || len(emailedList) != 2 {
		t.Fatalf("Expected 2 emailed contacts (alice, bob), got %d", countEmailed)
	}

	// Test GetEmails filter by sender
	senderEmails, total, err := storage.GetEmails(userID, "newsletter@marketing.com", "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails failed: %v", err)
	}
	if total != 1 || len(senderEmails) != 1 {
		t.Errorf("Expected 1 email from newsletter, got %d", total)
	}
	if senderEmails[0].Subject != "Huge Sales This Weekend!" {
		t.Errorf("Unexpected subject: %s", senderEmails[0].Subject)
	}
	if !senderEmails[0].SenderNeverEmailed {
		t.Errorf("Expected newsletter to be marked as SenderNeverEmailed")
	}
	if senderEmails[0].SenderEmailCount != 1 {
		t.Errorf("Expected SenderEmailCount to be 1, got %d", senderEmails[0].SenderEmailCount)
	}

	// Test GetEmails filter="never_emailed"
	neverEmails, neverTotal, err := storage.GetEmails(userID, "", "", "never_emailed", 10, 0)
	if err != nil {
		t.Fatalf("GetEmails never_emailed failed: %v", err)
	}
	if neverTotal != 2 || len(neverEmails) != 2 {
		t.Errorf("Expected 2 never_emailed emails, got total=%d len=%d", neverTotal, len(neverEmails))
	}
	// Test GetEmailByID
	msg1, err := storage.GetEmailByID(userID, "msg-1")
	if err != nil {
		t.Fatalf("GetEmailByID failed: %v", err)
	}
	if msg1 == nil || msg1.ID != "msg-1" {
		t.Errorf("Expected msg-1, got %+v", msg1)
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
	contactsFirst, totalContactsFirst, err := storage.GetContactStats(userID, "all", "", "received_count", "desc", 10, 0)
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
	contactsSecond, totalContactsSecond, err := storage.GetContactStats(userID, "all", "", "received_count", "desc", 10, 0)
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
	if overallFirst.NeverEmailedCount != overallSecond.NeverEmailedCount {
		t.Errorf("NeverEmailedCount changed: first=%d, second=%d", overallFirst.NeverEmailedCount, overallSecond.NeverEmailedCount)
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
		if c1.SentCount != c2.SentCount {
			t.Errorf("Contact %s SentCount inflated: first=%d, second=%d", c1.Email, c1.SentCount, c2.SentCount)
		}
		if c1.TotalSize != c2.TotalSize {
			t.Errorf("Contact %s TotalSize inflated: first=%d, second=%d", c1.Email, c1.TotalSize, c2.TotalSize)
		}
	}
}

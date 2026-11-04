package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestAuthAndHandlerEndpoints(t *testing.T) {
	storage := setupTestDB(t)
	DB = storage

	userID := "user@gmail.com"
	userEmail := "user@gmail.com"
	sessionToken := "valid-session-token"

	_ = DB.UpsertUser(userID, userEmail)
	_ = DB.CreateSession(sessionToken, userID, userEmail, &oauth2.Token{
		AccessToken: "mock-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}, 24*time.Hour)
	router := NewRouter("")

	// Populate some sample emails
	_ = DB.SaveEmailsBatch(userID, []EmailMetadata{
		{
			ID:           "test-msg-1",
			ThreadID:     "th-1",
			From:         "sender@example.com",
			FromName:     "Example Sender",
			To:           []string{userEmail},
			Subject:      "Important Update",
			Snippet:      "Here is the weekly status...",
			Date:         time.Now(),
			SizeEstimate: 1200,
			IsSent:       false,
		},
	})

	// 1. Test /api/auth/me without token -> 401
	{
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized without auth header, got %d", rec.Code)
		}
	}

	// 2. Test /api/auth/me with valid Bearer token -> 200
	{
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if resp["userEmail"] != userEmail {
			t.Errorf("Expected email %s, got %v", userEmail, resp["userEmail"])
		}
	}

	// 3. Test /api/emails
	{
		req := httptest.NewRequest("GET", "/api/emails?limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		emailsList, ok := resp["emails"].([]interface{})
		if !ok || len(emailsList) != 1 {
			t.Errorf("Expected 1 email returned, got %v", resp)
		}
	}

	// 4. Test /api/emails/{id}
	{
		req := httptest.NewRequest("GET", "/api/emails/test-msg-1", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var email EmailMetadata
		_ = json.Unmarshal(rec.Body.Bytes(), &email)
		if email.ID != "test-msg-1" || email.Subject != "Important Update" {
			t.Errorf("Unexpected email details: %+v", email)
		}
	}
	// 4b. Test method not allowed via ServeMux method routing
	{
		req := httptest.NewRequest("POST", "/api/emails", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 Method Not Allowed, got %d", rec.Code)
		}
	}

	// 5. Test /api/inbox/stats
	{
		req := httptest.NewRequest("GET", "/api/inbox/stats", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var stats OverallStats
		_ = json.Unmarshal(rec.Body.Bytes(), &stats)
		if stats.TotalEmailsProcessed != 1 || stats.NeverEmailedSenders != 1 || stats.NeverEmailedEmails != 1 {
			t.Errorf("Expected 1 email processed from 1 never-emailed sender, got %+v", stats)
		}
	}

	// 6. Test /api/inbox/contacts
	{
		req := httptest.NewRequest("GET", "/api/inbox/contacts", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		contactsList, ok := resp["contacts"].([]interface{})
		if !ok || len(contactsList) != 1 {
			t.Errorf("Expected 1 contact, got %v", resp)
		}
	}

	// 6b. Test /api/inbox/candidate-senders
	{
		req := httptest.NewRequest("GET", "/api/inbox/candidate-senders", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK from /api/inbox/candidate-senders, got %d", rec.Code)
		}

		var senders []map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &senders)
		// test-msg-1 is from today, so 0 candidates older than 1 month
		if len(senders) != 0 {
			t.Errorf("Expected 0 candidate senders for today's email, got %d", len(senders))
		}
	}

	// 7. Test /api/auth/logout
	{
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK on logout, got %d", rec.Code)
		}

		// Subsequent /api/auth/me should now fail with 401
		reqMe := httptest.NewRequest("GET", "/api/auth/me", nil)
		reqMe.Header.Set("Authorization", "Bearer "+sessionToken)
		recMe := httptest.NewRecorder()
		router.ServeHTTP(recMe, reqMe)

		if recMe.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 after logout, got %d", recMe.Code)
		}
	}
}

func TestSPAFallback(t *testing.T) {
	tmpDir := t.TempDir()
	indexContent := "<html><body>SPA App</body></html>"
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	router := NewRouter(tmpDir)

	// Unknown client-side route should fall back to index.html
	req := httptest.NewRequest("GET", "/dashboard/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for SPA fallback, got %d", rec.Code)
	}
	if rec.Body.String() != indexContent {
		t.Errorf("Expected index.html content, got %s", rec.Body.String())
	}

	// Unknown /api/* route should return 404, not index.html
	{
		req := httptest.NewRequest("GET", "/api/unknown-endpoint", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for unknown API route, got %d", rec.Code)
		}
	}

	// Unknown /auth/* route should return 404, not index.html
	{
		req := httptest.NewRequest("GET", "/auth/unknown-endpoint", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for unknown auth route, got %d", rec.Code)
		}
	}
}
func TestGetEmailsWithPlusAddressAndLegacyTimestamp(t *testing.T) {
	storage := setupTestDB(t)
	DB = storage

	userID := "user-plus-test@example.com"
	sessionToken := "token-plus-test"

	_ = DB.UpsertUser(userID, userID)
	_ = DB.CreateSession(sessionToken, userID, userID, &oauth2.Token{
		AccessToken: "mock-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}, 24*time.Hour)

	router := NewRouter("")

	// Insert email with + in address and legacy timestamp string directly into database
	msgID := "msg-fb-plus-1"
	senderAddr := "notification+pvl16c6f@facebookmail.com"
	_, err := DB.db.Exec(`
		INSERT INTO emails (
			id, user_id, thread_id, from_address, from_name, to_addresses,
			subject, snippet, date, size_estimate, label_ids, is_sent, created_at
		) VALUES (
			?, ?, 'th-fb-1', ?, 'Facebook', '["user-plus-test@example.com"]',
			'Facebook Notification', 'You have a new alert',
			'2009-08-01 16:01:23 -0700 -0700', 4096, '["INBOX"]', 0, '2026-09-29 12:00:00'
		)
	`, msgID, userID, senderAddr)
	if err != nil {
		t.Fatalf("Failed inserting test email: %v", err)
	}
	_, err = DB.db.Exec(`
		INSERT INTO contact_stats (user_id, email, name, received_count, sent_count, total_size)
		VALUES (?, ?, 'Facebook', 1, 0, 4096)
	`, userID, senderAddr)
	if err != nil {
		t.Fatalf("Failed inserting contact stats: %v", err)
	}

	// 1. Test GET /api/emails?sender=notification%2Bpvl16c6f%40facebookmail.com
	{
		req := httptest.NewRequest("GET", "/api/emails?sender=notification%2Bpvl16c6f%40facebookmail.com", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for sender with plus, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		total, _ := resp["total"].(float64)
		if total != 1 {
			t.Errorf("Expected total=1 email from plus sender, got %v", total)
		}

		emailsList, ok := resp["emails"].([]interface{})
		if !ok || len(emailsList) != 1 {
			t.Fatalf("Expected 1 email in list, got %v", emailsList)
		}

		emMap := emailsList[0].(map[string]interface{})
		if emMap["from"] != senderAddr {
			t.Errorf("Expected from %q, got %v", senderAddr, emMap["from"])
		}
		if emMap["date"] == nil || emMap["date"] == "" {
			t.Errorf("Expected non-empty date, got %v", emMap["date"])
		}
	}

	// 2. Test GET /api/emails/{id} detail route
	{
		req := httptest.NewRequest("GET", "/api/emails/"+msgID, nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for single email, got %d: %s", rec.Code, rec.Body.String())
		}

		var em EmailMetadata
		if err := json.Unmarshal(rec.Body.Bytes(), &em); err != nil {
			t.Fatalf("Failed to parse single email JSON: %v", err)
		}
		if em.ID != msgID {
			t.Errorf("Expected email ID %q, got %q", msgID, em.ID)
		}
		if em.Date.IsZero() {
			t.Errorf("Expected non-zero parsed date on single email")
		}
	}
}

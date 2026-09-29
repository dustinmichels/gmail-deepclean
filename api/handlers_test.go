package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
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
		HandleGetMe(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized without auth header, got %d", rec.Code)
		}
	}

	// 2. Test /api/auth/me with valid Bearer token -> 200
	{
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		HandleGetMe(rec, req)

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
		HandleGetEmails(rec, req)

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
		req = mux.SetURLVars(req, map[string]string{"id": "test-msg-1"})
		rec := httptest.NewRecorder()
		HandleGetEmailByID(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var email EmailMetadata
		_ = json.Unmarshal(rec.Body.Bytes(), &email)
		if email.ID != "test-msg-1" || email.Subject != "Important Update" {
			t.Errorf("Unexpected email details: %+v", email)
		}
	}

	// 5. Test /api/inbox/stats
	{
		req := httptest.NewRequest("GET", "/api/inbox/stats", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		HandleGetEmailStats(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var stats OverallStats
		_ = json.Unmarshal(rec.Body.Bytes(), &stats)
		if stats.TotalEmailsProcessed != 1 || stats.NeverEmailedCount != 1 {
			t.Errorf("Expected 1 email processed and 1 never-emailed sender, got %+v", stats)
		}
	}

	// 6. Test /api/inbox/contacts
	{
		req := httptest.NewRequest("GET", "/api/inbox/contacts?filter=never_emailed", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		HandleGetContacts(rec, req)

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

	// 7. Test /api/auth/logout
	{
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+sessionToken)
		rec := httptest.NewRecorder()
		HandleLogout(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK on logout, got %d", rec.Code)
		}

		// Subsequent /api/auth/me should now fail with 401
		reqMe := httptest.NewRequest("GET", "/api/auth/me", nil)
		reqMe.Header.Set("Authorization", "Bearer "+sessionToken)
		recMe := httptest.NewRecorder()
		HandleGetMe(recMe, reqMe)

		if recMe.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 after logout, got %d", recMe.Code)
		}
	}
}

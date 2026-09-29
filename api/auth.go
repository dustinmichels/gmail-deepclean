package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// generateSecureToken creates a cryptographically secure random hex string
func generateSecureToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HandleGmailAuth initiates the OAuth flow with a cryptographically secure state
func HandleGmailAuth(w http.ResponseWriter, r *http.Request) {
	state, err := generateSecureToken(32)
	if err != nil {
		http.Error(w, "Failed to generate CSRF token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Persist state with 15-minute TTL
	if err := DB.SaveCSRFState(state, 15*time.Minute); err != nil {
		http.Error(w, "Failed to persist CSRF state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Also set in an HTTP-only SameSite cookie for defense-in-depth
	http.SetCookie(w, &http.Cookie{
		Name:     "gmail_oauth_csrf",
		Value:    state,
		Path:     "/auth/gmail/callback",
		Expires:  time.Now().Add(15 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to Google OAuth URL (offline access for refresh token, prompt consent)
	authURL := oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleGmailCallback processes the OAuth callback
func HandleGmailCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state to prevent CSRF
	state := r.FormValue("state")
	if state == "" {
		http.Error(w, "Missing OAuth state parameter", http.StatusBadRequest)
		return
	}

	valid, err := DB.ValidateAndConsumeCSRFState(state)
	if err != nil || !valid {
		http.Error(w, "Invalid or expired OAuth CSRF state", http.StatusBadRequest)
		return
	}

	// Check for auth error from Google
	if authErr := r.FormValue("error"); authErr != "" {
		http.Error(w, "Google OAuth error: "+authErr, http.StatusBadRequest)
		return
	}

	// Exchange auth code for token
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch authenticated user profile using Gmail API
	client := oauthConfig.Client(ctx, token)
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Failed to initialize Gmail service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	profile, err := gmailService.Users.GetProfile("me").Do()
	if err != nil {
		http.Error(w, "Failed to fetch user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userEmail := strings.ToLower(strings.TrimSpace(profile.EmailAddress))
	userID := userEmail

	// Upsert user in database
	if err := DB.UpsertUser(userID, userEmail); err != nil {
		log.Printf("Failed to upsert user %s: %v", userID, err)
	}

	// Generate secure session token (30-day session)
	sessionToken, err := generateSecureToken(32)
	if err != nil {
		http.Error(w, "Failed to generate session token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := DB.CreateSession(sessionToken, userID, userEmail, token, 30*24*time.Hour); err != nil {
		http.Error(w, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare payload for frontend postMessage
	payload := map[string]interface{}{
		"sessionToken": sessionToken,
		"userEmail":    userEmail,
		"userId":       userID,
	}
	payloadJSON, _ := json.Marshal(payload)

	// Set content type and write HTML response to pass session to window.opener
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; text-align: center; padding: 40px; }
        .success { color: #1e7e34; }
    </style>
</head>
<body>
    <h3 class="success">Authentication Successful!</h3>
    <p>Connected as <strong>%s</strong></p>
    <p>Closing window...</p>
    <script>
        try {
            const data = %s;
            if (window.opener) {
                window.opener.postMessage({ type: 'GMAIL_AUTH_SUCCESS', session: data }, window.location.origin);
                setTimeout(function() {
                    window.close();
                }, 800);
            } else {
                document.body.innerHTML += "<p>Please return to the main application window.</p>";
            }
        } catch (e) {
            document.body.innerHTML += "<p>Error: " + e.message + "</p>";
        }
    </script>
</body>
</html>`, userEmail, string(payloadJSON))

	w.Write([]byte(html))
}

// ParseSession extracts and validates the session token from the Authorization header
func ParseSession(r *http.Request) (*Session, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header not provided")
	}

	var sessionToken string
	if strings.HasPrefix(authHeader, "Bearer ") {
		sessionToken = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		// Fallback if raw token passed
		sessionToken = authHeader
	}
	sessionToken = strings.TrimSpace(sessionToken)

	sess, err := DB.GetSession(sessionToken)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if sess == nil {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Check if OAuth access token is expired, and refresh if needed
	if sess.OAuthToken != nil && sess.OAuthToken.Expiry.Before(time.Now().Add(2*time.Minute)) {
		tokenSource := oauthConfig.TokenSource(r.Context(), sess.OAuthToken)
		newToken, err := tokenSource.Token()
		if err == nil && newToken.AccessToken != sess.OAuthToken.AccessToken {
			sess.OAuthToken = newToken
			_ = DB.UpdateSessionOAuthToken(sess.Token, newToken)
		}
	}

	return sess, nil
}

// HandleGetMe returns details of the currently authenticated user
func HandleGetMe(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"userId":        sess.UserID,
		"userEmail":     sess.UserEmail,
		"createdAt":     sess.CreatedAt,
	})
}

// HandleLogout invalidates the current session
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err == nil && sess != nil {
		_ = DB.DeleteSession(sess.Token)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

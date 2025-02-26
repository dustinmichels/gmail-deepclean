package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// handleGmailAuth initiates the OAuth flow
func HandleGmailAuth(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleGmailCallback processes the OAuth callback
func HandleGmailCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state to prevent CSRF
	state := r.FormValue("state")
	if state != oauthStateString {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	// Exchange auth code for token
	code := r.FormValue("code")
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert token to a map for easier JSON handling
	tokenMap := map[string]interface{}{
		"access_token":  token.AccessToken,
		"token_type":    token.TokenType,
		"refresh_token": token.RefreshToken,
		"expiry":        token.Expiry.Format(time.RFC3339),
	}

	// Convert to JSON
	tokenJSON, err := json.Marshal(tokenMap)
	if err != nil {
		http.Error(w, "Failed to marshal token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a base64 encoded version of the token JSON to avoid any escaping issues
	tokenBase64 := base64.StdEncoding.EncodeToString(tokenJSON)

	// Set content type and write the HTML response
	w.Header().Set("Content-Type", "text/html")
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
</head>
<body>
    <h3>Authentication Successful</h3>
    <p>You can close this window now.</p>
    <script>
        try {
            // Decode the base64 encoded token
            const tokenBase64 = "%s";
            const tokenJSON = atob(tokenBase64);
            const token = JSON.parse(tokenJSON);
            
            if (window.opener) {
                window.opener.postMessage({token: token}, "*");
                console.log("Token sent to main window");
                setTimeout(function() {
                    window.close();
                }, 1000);
            } else {
                document.body.innerHTML += "<p>Error: Could not communicate with the main application window.</p>";
            }
        } catch (e) {
            document.body.innerHTML += "<p>Error during authentication: " + e.message + "</p>";
            console.error("Auth error:", e);
        }
    </script>
</body>
</html>`, tokenBase64)

	w.Write([]byte(html))
}

// ParseToken extracts and validates the OAuth token from the Authorization header
func ParseToken(r *http.Request) (*oauth2.Token, error) {
	// Get token from Authorization header
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		return nil, fmt.Errorf("authorization header not provided")
	}

	// Remove "Bearer " prefix if present
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	// Parse token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenStr), &token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	return &token, nil
}

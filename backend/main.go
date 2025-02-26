package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Configuration struct for OAuth
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

var (
	config           Config
	oauthConfig      *oauth2.Config
	oauthStateString = "random-state-string" // Replace with a secure random string in production
)

func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Initialize configuration
	config = Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
	}

	// Set up OAuth2 configuration
	oauthConfig = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			gmail.GmailReadonlyScope, // For reading emails
			gmail.GmailModifyScope,   // For modifying/deleting emails
		},
		Endpoint: google.Endpoint,
	}
}

func main() {
	router := mux.NewRouter()

	// API Routes
	router.HandleFunc("/auth/gmail", handleGmailAuth).Methods("GET")
	router.HandleFunc("/auth/gmail/callback", handleGmailCallback).Methods("GET")
	router.HandleFunc("/api/emails", handleGetEmails).Methods("GET")
	router.HandleFunc("/api/emails/{id}", handleDeleteEmail).Methods("DELETE")

	// Serve Svelte frontend from dist directory
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/dist")))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

// handleGmailAuth initiates the OAuth flow
func handleGmailAuth(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleGmailCallback processes the OAuth callback
func handleGmailCallback(w http.ResponseWriter, r *http.Request) {
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

// handleGetEmails retrieves emails using the Gmail API
func handleGetEmails(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	// Parse token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenStr), &token); err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Create Gmail service
	client := oauthConfig.Client(context.Background(), &token)
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Failed to create Gmail service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get emails (example: list 10 messages from inbox)
	user := "me" // special value for the authenticated user
	messages, err := gmailService.Users.Messages.List(user).MaxResults(10).Q("in:inbox").Do()
	if err != nil {
		http.Error(w, "Failed to fetch emails: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return messages as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleDeleteEmail deletes an email using the Gmail API
func handleDeleteEmail(w http.ResponseWriter, r *http.Request) {
	// Get message ID from URL
	vars := mux.Vars(r)
	messageID := vars["id"]

	// Get token from Authorization header
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	// Parse token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenStr), &token); err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Create Gmail service
	client := oauthConfig.Client(context.Background(), &token)
	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Failed to create Gmail service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete message (using trash)
	user := "me" // special value for the authenticated user
	_, err = gmailService.Users.Messages.Trash(user, messageID).Do()
	if err != nil {
		http.Error(w, "Failed to delete email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Email moved to trash"})
}

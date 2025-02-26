package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// Store token (in a real app, you would persist this securely)
	// For now, just send it to the frontend
	w.Header().Set("Content-Type", "application/json")
	tokenJSON, _ := json.Marshal(token)

	// In a production app, you would set a secure HTTP-only cookie or use a session
	// and NOT return the token directly to the frontend
	// This is just for demonstration purposes
	script := fmt.Sprintf(`
		<script>
			window.opener.postMessage({"token": %s}, "*");
			window.close();
		</script>
	`, string(tokenJSON))

	w.Write([]byte(script))
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

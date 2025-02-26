package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// HandleGetEmails retrieves emails using the Gmail API
func HandleGetEmails(w http.ResponseWriter, r *http.Request) {
	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Create Gmail service
	client := oauthConfig.Client(context.Background(), token)
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

// HandleDeleteEmail deletes an email using the Gmail API
func HandleDeleteEmail(w http.ResponseWriter, r *http.Request) {
	// Get message ID from URL
	vars := mux.Vars(r)
	messageID := vars["id"]

	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Create Gmail service
	client := oauthConfig.Client(context.Background(), token)
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

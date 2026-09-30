package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// HandleGetEmails retrieves emails with rich metadata, filtering, and pagination from persistent storage
func HandleGetEmails(w http.ResponseWriter, r *http.Request, sess *Session) {

	query := r.URL.Query()
	sender := query.Get("sender")
	search := query.Get("search")
	filter := query.Get("filter")

	limit := 25
	if l := query.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	emails, total, err := DB.GetEmails(sess.UserID, sender, search, filter, limit, offset)
	if err != nil {
		http.Error(w, "Failed to retrieve emails: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"emails": emails,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// HandleGetEmailByID retrieves details for a single email message
func HandleGetEmailByID(w http.ResponseWriter, r *http.Request, sess *Session) {

	messageID := r.PathValue("id")
	if messageID == "" {
		http.Error(w, "Missing email ID", http.StatusBadRequest)
		return
	}

	email, err := DB.GetEmailByID(sess.UserID, messageID)
	if err != nil {
		http.Error(w, "Failed to retrieve email: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if email == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(email)
}

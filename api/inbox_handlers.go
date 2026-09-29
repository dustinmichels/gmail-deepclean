package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// HandleStartProcessingInbox initiates or resumes inbox scanning
func HandleStartProcessingInbox(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	reset := r.URL.Query().Get("reset") == "true"

	processor, err := Registry.GetOrCreate(sess.UserID, sess.UserEmail, sess.OAuthToken)
	if err != nil {
		http.Error(w, "Failed to initialize processor: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := processor.StartProcessing(reset); err != nil {
		http.Error(w, "Failed to start scan: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processor.GetProgress())
}

// HandlePauseProcessingInbox stops the inbox scanning
func HandlePauseProcessingInbox(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	processor, exists := Registry.Get(sess.UserID)
	if exists {
		_ = processor.PauseProcessing()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(processor.GetProgress())
		return
	}

	state, err := DB.GetScanState(sess.UserID)
	if err != nil {
		http.Error(w, "Failed to retrieve scan state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	state.IsProcessing = false
	state.StatusMessage = "Scan paused"
	_ = DB.SaveScanState(state)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProgressInfo{
		UserID:        sess.UserID,
		UserEmail:     sess.UserEmail,
		IsProcessing:  false,
		TotalEmails:   state.TotalEmails,
		StatusMessage: state.StatusMessage,
		LastScannedAt: state.LastScannedAt,
	})
}

// HandleGetInboxStatus returns the current processing status
func HandleGetInboxStatus(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	processor, exists := Registry.Get(sess.UserID)
	if exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(processor.GetProgress())
		return
	}

	// Fallback to SQLite scan state
	state, err := DB.GetScanState(sess.UserID)
	if err != nil {
		http.Error(w, "Failed to load scan state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProgressInfo{
		UserID:        sess.UserID,
		UserEmail:     sess.UserEmail,
		IsProcessing:  state.IsProcessing,
		TotalEmails:   state.TotalEmails,
		StatusMessage: state.StatusMessage,
		LastScannedAt: state.LastScannedAt,
	})
}

// HandleGetContacts returns contact statistics with filtering, searching, and sorting
func HandleGetContacts(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()
	filter := query.Get("filter") // "all", "never_emailed", "emailed"
	search := query.Get("search")
	sortField := query.Get("sort")
	sortOrder := query.Get("order")

	limit := 50
	if l := query.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	contacts, total, err := DB.GetContactStats(sess.UserID, filter, search, sortField, sortOrder, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch contact statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contacts": contacts,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// HandleGetTopSenders returns top senders (convenience endpoint)
func HandleGetTopSenders(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	contacts, _, err := DB.GetContactStats(sess.UserID, "all", "", "received_count", "desc", 20, 0)
	if err != nil {
		http.Error(w, "Failed to fetch top senders: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

// HandleGetEmailStats returns aggregated metrics
func HandleGetEmailStats(w http.ResponseWriter, r *http.Request) {
	sess, err := ParseSession(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	stats, err := DB.GetOverallStats(sess.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch overall statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

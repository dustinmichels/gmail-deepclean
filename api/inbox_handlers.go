package api

import (
	"encoding/json"
	"net/http"
)

// HandleStartProcessingInbox initiates the inbox processing
func HandleStartProcessingInbox(w http.ResponseWriter, r *http.Request) {
	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Use token hash as user ID (simplified, use a better ID method in production)
	userID := token.AccessToken[:10]

	// Check if already processing
	if processor, exists := Registry.Get(userID); exists {
		// Return current status
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(processor.GetProgress())
		return
	}

	// Create new processor
	processor, err := NewInboxProcessor(token)
	if err != nil {
		http.Error(w, "Failed to create inbox processor: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Register processor
	Registry.Register(userID, processor)

	// Start processing
	if err := processor.StartProcessing(); err != nil {
		http.Error(w, "Failed to start processing: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return initial status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processor.GetProgress())
}

// HandleGetInboxStatus returns the current processing status
func HandleGetInboxStatus(w http.ResponseWriter, r *http.Request) {
	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Use token hash as user ID (simplified, use a better ID method in production)
	userID := token.AccessToken[:10]

	// Get processor
	processor, exists := Registry.Get(userID)
	if !exists {
		http.Error(w, "No processing found for this user", http.StatusNotFound)
		return
	}

	// Return current progress
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processor.GetProgress())
}

// HandleGetTopSenders returns the top email senders
func HandleGetTopSenders(w http.ResponseWriter, r *http.Request) {
	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Use token hash as user ID (simplified, use a better ID method in production)
	userID := token.AccessToken[:10]

	// Get processor
	processor, exists := Registry.Get(userID)
	if !exists {
		http.Error(w, "No processing found for this user", http.StatusNotFound)
		return
	}

	// Get the top 20 senders
	topSenders := processor.GetTopSenders(20)

	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topSenders)
}

// HandleGetEmailStats returns the email statistics
func HandleGetEmailStats(w http.ResponseWriter, r *http.Request) {
	// Parse token from Authorization header
	token, err := ParseToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Use token hash as user ID (simplified, use a better ID method in production)
	userID := token.AccessToken[:10]

	// Get processor
	processor, exists := Registry.Get(userID)
	if !exists {
		http.Error(w, "No processing found for this user", http.StatusNotFound)
		return
	}

	// Return statistics
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processor.GetStats())
}

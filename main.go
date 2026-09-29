package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/dustinmichels/gmail-deepclean/api"
)

func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Note: .env file not loaded: %v", err)
	}
}

func main() {
	router := mux.NewRouter()

	// Initialize API and persistent database
	api.Init()

	// Auth routes
	router.HandleFunc("/auth/gmail", api.HandleGmailAuth).Methods("GET")
	router.HandleFunc("/auth/gmail/callback", api.HandleGmailCallback).Methods("GET")
	router.HandleFunc("/api/auth/me", api.HandleGetMe).Methods("GET")
	router.HandleFunc("/api/auth/logout", api.HandleLogout).Methods("POST")

	// Email routes - strictly read-only
	router.HandleFunc("/api/emails", api.HandleGetEmails).Methods("GET")
	router.HandleFunc("/api/emails/{id}", api.HandleGetEmailByID).Methods("GET")

	// Inbox processing & analysis routes
	router.HandleFunc("/api/inbox/process", api.HandleStartProcessingInbox).Methods("POST")
	router.HandleFunc("/api/inbox/pause", api.HandlePauseProcessingInbox).Methods("POST")
	router.HandleFunc("/api/inbox/status", api.HandleGetInboxStatus).Methods("GET")
	router.HandleFunc("/api/inbox/stats", api.HandleGetEmailStats).Methods("GET")
	router.HandleFunc("/api/inbox/contacts", api.HandleGetContacts).Methods("GET")
	router.HandleFunc("/api/inbox/top-senders", api.HandleGetTopSenders).Methods("GET")

	// Serve Vue frontend from dist directory
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

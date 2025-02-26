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
		log.Printf("Error loading .env file: %v", err)
	}
}

func main() {
	router := mux.NewRouter()

	// Initialize API
	api.Init()

	// API Routes
	router.HandleFunc("/auth/gmail", api.HandleGmailAuth).Methods("GET")
	router.HandleFunc("/auth/gmail/callback", api.HandleGmailCallback).Methods("GET")
	router.HandleFunc("/api/emails", api.HandleGetEmails).Methods("GET")
	router.HandleFunc("/api/emails/{id}", api.HandleDeleteEmail).Methods("DELETE")

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

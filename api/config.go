package api

import (
	"log"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Configuration struct for OAuth and App settings
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	DBPath       string
}

var (
	config      Config
	oauthConfig *oauth2.Config
)

// Init initializes the API configuration and persistent storage
func Init() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/deepclean.db"
	}

	config = Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		DBPath:       dbPath,
	}

	// Set up OAuth2 configuration
	// Note: gmail.MailGoogleComScope is required for IMAP XOAUTH2 access.
	// All application processing remains strictly read-only.
	oauthConfig = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			gmail.MailGoogleComScope,
			gmail.GmailReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	// Initialize SQLite database
	var err error
	DB, err = InitDB(config.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database at %s: %v", config.DBPath, err)
	}
	log.Printf("Initialized persistent storage at %s", config.DBPath)
}

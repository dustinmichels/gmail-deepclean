package api

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
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

// Init initializes the API configuration
func Init() {
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

package api

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// NewRouter configures and returns the application HTTP router.
func NewRouter(staticDir string) http.Handler {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /auth/gmail", HandleGmailAuth)
	mux.HandleFunc("GET /auth/gmail/callback", HandleGmailCallback)
	mux.HandleFunc("GET /api/auth/me", RequireAuth(HandleGetMe))
	mux.HandleFunc("POST /api/auth/logout", HandleLogout)

	// Email routes - strictly read-only
	mux.HandleFunc("GET /api/emails", RequireAuth(HandleGetEmails))
	mux.HandleFunc("GET /api/emails/{id}", RequireAuth(HandleGetEmailByID))

	// Inbox processing & analysis routes
	mux.HandleFunc("POST /api/inbox/process", RequireAuth(HandleStartProcessingInbox))
	mux.HandleFunc("POST /api/inbox/pause", RequireAuth(HandlePauseProcessingInbox))
	mux.HandleFunc("GET /api/inbox/status", RequireAuth(HandleGetInboxStatus))
	mux.HandleFunc("GET /api/inbox/stats", RequireAuth(HandleGetEmailStats))
	mux.HandleFunc("GET /api/inbox/contacts", RequireAuth(HandleGetContacts))
	mux.HandleFunc("GET /api/inbox/top-senders", RequireAuth(HandleGetTopSenders))
	mux.HandleFunc("GET /api/inbox/candidate-senders", RequireAuth(HandleGetCandidateSenders))

	// Serve static frontend with SPA fallback
	if staticDir != "" {
		mux.Handle("/", spaHandler(staticDir))
	}

	return mux
}

// spaHandler serves static assets from dir, falling back to index.html for SPA routing.
// Unmatched API and auth routes are rejected with 404 rather than serving index.html.
func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)
	fileServer := http.FileServer(fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/auth/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := path.Clean(r.URL.Path)
		f, err := fs.Open(cleanPath)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}

		if stat.IsDir() {
			if idxFile, err := fs.Open(path.Join(cleanPath, "index.html")); err == nil {
				_ = idxFile.Close()
			} else {
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

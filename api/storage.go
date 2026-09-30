package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
)

// Session represents an authenticated user session
type Session struct {
	Token      string        `json:"token"`
	UserID     string        `json:"userId"`
	UserEmail  string        `json:"userEmail"`
	OAuthToken *oauth2.Token `json:"-"`
	CreatedAt  time.Time     `json:"createdAt"`
	ExpiresAt  time.Time     `json:"expiresAt"`
}

// ContactStat stores aggregated communication metrics with an email contact
type ContactStat struct {
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	ReceivedCount int        `json:"receivedCount"`
	SentCount     int        `json:"sentCount"`
	TotalSize     int64      `json:"totalSize"`
	FirstReceived *time.Time `json:"firstReceived,omitempty"`
	LastReceived  *time.Time `json:"lastReceived,omitempty"`
	FirstSent     *time.Time `json:"firstSent,omitempty"`
	LastSent      *time.Time `json:"lastSent,omitempty"`
	NeverEmailed  bool       `json:"neverEmailed"`
}

// OverallStats contains high-level metrics for the user's inbox
type OverallStats struct {
	TotalEmailsProcessed int   `json:"totalEmailsProcessed"`
	TotalReceived        int   `json:"totalReceived"`
	TotalSent            int   `json:"totalSent"`
	UniqueSenders        int   `json:"uniqueSenders"`
	NeverEmailedCount    int   `json:"neverEmailedCount"`
	TotalSizeBytes       int64 `json:"totalSizeBytes"`
}

// ScanState tracks background scanning state across restarts
type ScanState struct {
	UserID         string     `json:"userId"`
	IsProcessing   bool       `json:"isProcessing"`
	TotalEmails    int        `json:"totalEmails"`
	EstimatedTotal int64      `json:"estimatedTotal"`
	NextPageToken  string     `json:"nextPageToken"`
	StatusMessage  string     `json:"statusMessage"`
	LastScannedAt  *time.Time `json:"lastScannedAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// Storage manages persistent SQLite storage
type Storage struct {
	db *sql.DB
	mu sync.Mutex
}

var DB *Storage

// InitDB initializes the SQLite database at dbPath and runs migrations
func InitDB(dbPath string) (*Storage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Restrict open connections to avoid SQLite concurrency locks on writes
	db.SetMaxOpenConns(1)

	// Ensure WAL mode and busy timeout are applied
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed setting sqlite pragmas: %w", err)
	}
	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	DB = s
	return s, nil
}

func (s *Storage) migrate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			user_email TEXT NOT NULL,
			access_token TEXT NOT NULL,
			token_type TEXT NOT NULL,
			refresh_token TEXT,
			expiry DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS csrf_states (
			state TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			thread_id TEXT NOT NULL,
			from_address TEXT NOT NULL,
			from_name TEXT,
			to_addresses TEXT NOT NULL,
			subject TEXT,
			snippet TEXT,
			date DATETIME NOT NULL,
			size_estimate INTEGER NOT NULL,
			label_ids TEXT NOT NULL,
			is_sent INTEGER NOT NULL,
			created_at DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_emails_user_from ON emails(user_id, from_address);`,
		`CREATE INDEX IF NOT EXISTS idx_emails_user_sent ON emails(user_id, is_sent);`,
		`CREATE INDEX IF NOT EXISTS idx_emails_user_date ON emails(user_id, date DESC);`,

		`CREATE TABLE IF NOT EXISTS contact_stats (
			user_id TEXT NOT NULL,
			email TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			received_count INTEGER NOT NULL DEFAULT 0,
			sent_count INTEGER NOT NULL DEFAULT 0,
			total_size INTEGER NOT NULL DEFAULT 0,
			first_received DATETIME,
			last_received DATETIME,
			first_sent DATETIME,
			last_sent DATETIME,
			PRIMARY KEY (user_id, email)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_contact_stats_user_counts ON contact_stats(user_id, received_count, sent_count);`,

		`CREATE TABLE IF NOT EXISTS scan_state (
			user_id TEXT PRIMARY KEY,
			is_processing INTEGER NOT NULL DEFAULT 0,
			total_emails INTEGER NOT NULL DEFAULT 0,
			estimated_total INTEGER NOT NULL DEFAULT 0,
			next_page_token TEXT NOT NULL DEFAULT '',
			status_message TEXT NOT NULL DEFAULT '',
			last_scanned_at DATETIME,
			updated_at DATETIME NOT NULL
		);`,
	}

	// Ensure backwards compatibility for existing databases without estimated_total column
	_, _ = s.db.Exec(`ALTER TABLE scan_state ADD COLUMN estimated_total INTEGER NOT NULL DEFAULT 0;`)

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed executing migration: %s: %w", q, err)
		}
	}
	return nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CSRF State Management
func (s *Storage) SaveCSRFState(state string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(ttl)

	// Clean expired states periodically
	_, _ = s.db.Exec(`DELETE FROM csrf_states WHERE expires_at < ?`, now)

	_, err := s.db.Exec(
		`INSERT INTO csrf_states (state, created_at, expires_at) VALUES (?, ?, ?)`,
		state, now, expiresAt,
	)
	return err
}

func (s *Storage) ValidateAndConsumeCSRFState(state string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM csrf_states WHERE state = ? AND expires_at > ?`,
		state, now,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		// Consume once
		_, _ = s.db.Exec(`DELETE FROM csrf_states WHERE state = ?`, state)
		return true, nil
	}

	return false, nil
}

// User & Session Management
func (s *Storage) UpsertUser(id, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO users (id, email, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			updated_at = excluded.updated_at
	`, id, email, now, now)
	return err
}

func (s *Storage) CreateSession(token, userID, userEmail string, oauthToken *oauth2.Token, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(ttl)

	_, err := s.db.Exec(`
		INSERT INTO sessions (
			token, user_id, user_email, access_token, token_type,
			refresh_token, expiry, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, token, userID, userEmail, oauthToken.AccessToken, oauthToken.TokenType,
		oauthToken.RefreshToken, oauthToken.Expiry, now, expiresAt)
	return err
}

func (s *Storage) GetSession(token string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var (
		sess      Session
		accToken  string
		tokenType string
		refToken  sql.NullString
		expiry    time.Time
	)

	err := s.db.QueryRow(`
		SELECT token, user_id, user_email, access_token, token_type, refresh_token, expiry, created_at, expires_at
		FROM sessions
		WHERE token = ? AND expires_at > ?
	`, token, now).Scan(
		&sess.Token,
		&sess.UserID,
		&sess.UserEmail,
		&accToken,
		&tokenType,
		&refToken,
		&expiry,
		&sess.CreatedAt,
		&sess.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	sess.OAuthToken = &oauth2.Token{
		AccessToken: accToken,
		TokenType:   tokenType,
		Expiry:      expiry,
	}
	if refToken.Valid {
		sess.OAuthToken.RefreshToken = refToken.String
	}

	return &sess, nil
}

func (s *Storage) UpdateSessionOAuthToken(sessionToken string, oauthToken *oauth2.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE sessions
		SET access_token = ?, token_type = ?, refresh_token = COALESCE(NULLIF(?, ''), refresh_token), expiry = ?
		WHERE token = ?
	`, oauthToken.AccessToken, oauthToken.TokenType, oauthToken.RefreshToken, oauthToken.Expiry, sessionToken)
	return err
}

func (s *Storage) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// Scan State Persistence
func (s *Storage) GetScanState(userID string) (*ScanState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		state         ScanState
		isProcInt     int
		lastScannedAt sql.NullTime
	)

	err := s.db.QueryRow(`
		SELECT user_id, is_processing, total_emails, estimated_total, next_page_token, status_message, last_scanned_at, updated_at
		FROM scan_state
		WHERE user_id = ?
	`, userID).Scan(
		&state.UserID,
		&isProcInt,
		&state.TotalEmails,
		&state.EstimatedTotal,
		&state.NextPageToken,
		&state.StatusMessage,
		&lastScannedAt,
		&state.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &ScanState{
				UserID:        userID,
				IsProcessing:  false,
				TotalEmails:   0,
				StatusMessage: "Ready to scan",
				UpdatedAt:     time.Now(),
			}, nil
		}
		return nil, err
	}

	state.IsProcessing = isProcInt == 1
	if lastScannedAt.Valid {
		state.LastScannedAt = &lastScannedAt.Time
	}

	return &state, nil
}

func (s *Storage) SaveScanState(state *ScanState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	isProcInt := 0
	if state.IsProcessing {
		isProcInt = 1
	}

	now := time.Now()
	state.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO scan_state (
			user_id, is_processing, total_emails, estimated_total, next_page_token, status_message, last_scanned_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			is_processing = excluded.is_processing,
			total_emails = excluded.total_emails,
			estimated_total = excluded.estimated_total,
			next_page_token = excluded.next_page_token,
			status_message = excluded.status_message,
			last_scanned_at = excluded.last_scanned_at,
			updated_at = excluded.updated_at
	`, state.UserID, isProcInt, state.TotalEmails, state.EstimatedTotal, state.NextPageToken, state.StatusMessage, state.LastScannedAt, now)
	return err
}

// Emails and Contact Stats Batch Storage
func (s *Storage) SaveEmailsBatch(userID string, emails []EmailMetadata) error {
	if len(emails) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	emailStmt, err := tx.Prepare(`
		INSERT INTO emails (
			id, user_id, thread_id, from_address, from_name, to_addresses,
			subject, snippet, date, size_estimate, label_ids, is_sent, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer emailStmt.Close()

	now := time.Now()

	for _, em := range emails {
		// Ignore drafts from affecting contact stats or counts
		isDraft := false
		for _, label := range em.LabelIDs {
			if strings.EqualFold(label, "DRAFT") {
				isDraft = true
				break
			}
		}
		if isDraft {
			continue
		}

		toAddressesJSON, _ := json.Marshal(em.To)
		labelIDsJSON, _ := json.Marshal(em.LabelIDs)

		isSentInt := 0
		if em.IsSent {
			isSentInt = 1
		}

		res, err := emailStmt.Exec(
			em.ID,
			userID,
			em.ThreadID,
			strings.ToLower(strings.TrimSpace(em.From)),
			em.FromName,
			string(toAddressesJSON),
			em.Subject,
			em.Snippet,
			em.Date,
			em.SizeEstimate,
			string(labelIDsJSON),
			isSentInt,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed inserting email %s: %w", em.ID, err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			// Email already inserted previously; skip contact_stats update to prevent count inflation
			continue
		}
		// Update Contact Stats
		if em.IsSent {
			// User sent this email to em.To recipients
			for _, to := range em.To {
				cleanTo := strings.ToLower(strings.TrimSpace(to))
				if cleanTo == "" {
					continue
				}

				_, err := tx.Exec(`
					INSERT INTO contact_stats (user_id, email, name, sent_count, first_sent, last_sent)
					VALUES (?, ?, '', 1, ?, ?)
					ON CONFLICT(user_id, email) DO UPDATE SET
						sent_count = contact_stats.sent_count + 1,
						first_sent = CASE WHEN contact_stats.first_sent IS NULL OR excluded.first_sent < contact_stats.first_sent THEN excluded.first_sent ELSE contact_stats.first_sent END,
						last_sent = CASE WHEN contact_stats.last_sent IS NULL OR excluded.last_sent > contact_stats.last_sent THEN excluded.last_sent ELSE contact_stats.last_sent END
				`, userID, cleanTo, em.Date, em.Date)
				if err != nil {
					return fmt.Errorf("failed updating contact stats for to %s: %w", cleanTo, err)
				}
			}
		} else {
			// Received email from em.From
			cleanFrom := strings.ToLower(strings.TrimSpace(em.From))
			if cleanFrom != "" {
				_, err := tx.Exec(`
					INSERT INTO contact_stats (
						user_id, email, name, received_count, total_size, first_received, last_received
					)
					VALUES (?, ?, ?, 1, ?, ?, ?)
					ON CONFLICT(user_id, email) DO UPDATE SET
						received_count = contact_stats.received_count + 1,
						total_size = contact_stats.total_size + excluded.total_size,
						name = CASE WHEN contact_stats.name = '' AND excluded.name != '' THEN excluded.name ELSE contact_stats.name END,
						first_received = CASE WHEN contact_stats.first_received IS NULL OR excluded.first_received < contact_stats.first_received THEN excluded.first_received ELSE contact_stats.first_received END,
						last_received = CASE WHEN contact_stats.last_received IS NULL OR excluded.last_received > contact_stats.last_received THEN excluded.last_received ELSE contact_stats.last_received END
				`, userID, cleanFrom, em.FromName, em.SizeEstimate, em.Date, em.Date)
				if err != nil {
					return fmt.Errorf("failed updating contact stats for from %s: %w", cleanFrom, err)
				}
			}
		}
	}

	return tx.Commit()
}

// GetContactStats returns contact statistics with filtering, searching, sorting, and pagination
func (s *Storage) GetContactStats(
	userID string,
	filter string, // "all", "never_emailed", "emailed"
	search string,
	sortField string, // "received_count", "sent_count", "total_size", "last_received", "email"
	sortOrder string, // "asc", "desc"
	limit int,
	offset int,
) ([]ContactStat, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	whereClauses := []string{"user_id = ?"}
	args := []interface{}{userID}

	switch filter {
	case "never_emailed":
		whereClauses = append(whereClauses, "received_count > 0", "sent_count = 0")
	case "emailed":
		whereClauses = append(whereClauses, "sent_count > 0")
	default: // "all"
		whereClauses = append(whereClauses, "(received_count > 0 OR sent_count > 0)")
	}

	if search != "" {
		whereClauses = append(whereClauses, "(email LIKE ? OR name LIKE ?)")
		searchTerm := "%" + strings.ToLower(search) + "%"
		args = append(args, searchTerm, searchTerm)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total matching
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM contact_stats WHERE %s", whereSQL)
	if err := s.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Allowed sort fields
	validSorts := map[string]string{
		"received_count": "received_count",
		"sent_count":     "sent_count",
		"total_size":     "total_size",
		"last_received":  "last_received",
		"email":          "email",
	}
	col, ok := validSorts[sortField]
	if !ok {
		col = "received_count"
	}

	order := "DESC"
	if strings.ToLower(sortOrder) == "asc" {
		order = "ASC"
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT email, name, received_count, sent_count, total_size,
		       first_received, last_received, first_sent, last_sent
		FROM contact_stats
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereSQL, col, order)

	queryArgs := append(args, limit, offset)
	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []ContactStat
	for rows.Next() {
		var (
			cs            ContactStat
			firstReceived sql.NullTime
			lastReceived  sql.NullTime
			firstSent     sql.NullTime
			lastSent      sql.NullTime
		)

		err := rows.Scan(
			&cs.Email,
			&cs.Name,
			&cs.ReceivedCount,
			&cs.SentCount,
			&cs.TotalSize,
			&firstReceived,
			&lastReceived,
			&firstSent,
			&lastSent,
		)
		if err != nil {
			return nil, 0, err
		}

		if firstReceived.Valid {
			cs.FirstReceived = &firstReceived.Time
		}
		if lastReceived.Valid {
			cs.LastReceived = &lastReceived.Time
		}
		if firstSent.Valid {
			cs.FirstSent = &firstSent.Time
		}
		if lastSent.Valid {
			cs.LastSent = &lastSent.Time
		}

		cs.NeverEmailed = cs.ReceivedCount > 0 && cs.SentCount == 0
		results = append(results, cs)
	}

	return results, totalCount, nil
}

// GetOverallStats aggregates high-level metrics for a user
func (s *Storage) GetOverallStats(userID string) (*OverallStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stats OverallStats

	// Count emails
	row := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_sent = 0 THEN 1 ELSE 0 END), 0) as received,
			COALESCE(SUM(CASE WHEN is_sent = 1 THEN 1 ELSE 0 END), 0) as sent,
			COALESCE(SUM(size_estimate), 0) as total_size
		FROM emails
		WHERE user_id = ?
	`, userID)
	if err := row.Scan(&stats.TotalEmailsProcessed, &stats.TotalReceived, &stats.TotalSent, &stats.TotalSizeBytes); err != nil {
		return nil, err
	}

	// Count contacts
	row = s.db.QueryRow(`
		SELECT
			COUNT(*) as unique_senders,
			COALESCE(SUM(CASE WHEN received_count > 0 AND sent_count = 0 THEN 1 ELSE 0 END), 0) as never_emailed
		FROM contact_stats
		WHERE user_id = ? AND received_count > 0
	`, userID)
	if err := row.Scan(&stats.UniqueSenders, &stats.NeverEmailedCount); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetEmails retrieves emails for a user with optional sender filter, search, filter mode ("never_emailed"), and pagination
func (s *Storage) GetEmails(userID, senderFilter, search, filter string, limit, offset int) ([]EmailMetadata, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	whereClauses := []string{"e.user_id = ?"}
	args := []interface{}{userID}

	if senderFilter != "" {
		whereClauses = append(whereClauses, "e.from_address = ?")
		args = append(args, strings.ToLower(strings.TrimSpace(senderFilter)))
	}

	if search != "" {
		whereClauses = append(whereClauses, "(e.subject LIKE ? OR e.snippet LIKE ? OR e.from_address LIKE ?)")
		term := "%" + strings.ToLower(search) + "%"
		args = append(args, term, term, term)
	}

	if filter == "never_emailed" {
		whereClauses = append(whereClauses, "e.is_sent = 0", "COALESCE(cs.sent_count, 0) = 0", "COALESCE(cs.received_count, 0) > 0")
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM emails e
		LEFT JOIN contact_stats cs ON cs.user_id = e.user_id AND cs.email = e.from_address
		WHERE %s
	`, whereSQL)
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT e.id, e.thread_id, e.from_address, e.from_name, e.to_addresses,
		       e.subject, e.snippet, e.date, e.size_estimate, e.label_ids, e.is_sent,
		       COALESCE(cs.sent_count, 0) as cs_sent,
		       COALESCE(cs.received_count, 0) as cs_recv
		FROM emails e
		LEFT JOIN contact_stats cs ON cs.user_id = e.user_id AND cs.email = e.from_address
		WHERE %s
		ORDER BY e.date DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var emails []EmailMetadata
	for rows.Next() {
		var (
			em              EmailMetadata
			toAddressesJSON string
			labelIDsJSON    string
			isSentInt       int
			fromName        sql.NullString
			subject         sql.NullString
			snippet         sql.NullString
			csSent          int
			csRecv          int
		)

		err := rows.Scan(
			&em.ID,
			&em.ThreadID,
			&em.From,
			&fromName,
			&toAddressesJSON,
			&subject,
			&snippet,
			&em.Date,
			&em.SizeEstimate,
			&labelIDsJSON,
			&isSentInt,
			&csSent,
			&csRecv,
		)
		if err != nil {
			return nil, 0, err
		}

		em.FromName = fromName.String
		em.Subject = subject.String
		em.Snippet = snippet.String
		em.IsSent = isSentInt == 1
		em.SenderNeverEmailed = !em.IsSent && csRecv > 0 && csSent == 0
		em.SenderEmailCount = csRecv

		_ = json.Unmarshal([]byte(toAddressesJSON), &em.To)
		_ = json.Unmarshal([]byte(labelIDsJSON), &em.LabelIDs)

		emails = append(emails, em)
	}

	return emails, total, nil
}

// GetEmailByID retrieves a single email
func (s *Storage) GetEmailByID(userID, emailID string) (*EmailMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		em              EmailMetadata
		toAddressesJSON string
		labelIDsJSON    string
		isSentInt       int
		fromName        sql.NullString
		subject         sql.NullString
		snippet         sql.NullString
		csSent          int
		csRecv          int
	)
	err := s.db.QueryRow(`
		SELECT e.id, e.thread_id, e.from_address, e.from_name, e.to_addresses,
		       e.subject, e.snippet, e.date, e.size_estimate, e.label_ids, e.is_sent,
		       COALESCE(cs.sent_count, 0) as cs_sent,
		       COALESCE(cs.received_count, 0) as cs_recv
		FROM emails e
		LEFT JOIN contact_stats cs ON cs.user_id = e.user_id AND cs.email = e.from_address
		WHERE e.user_id = ? AND e.id = ?
	`, userID, emailID).Scan(
		&em.ID,
		&em.ThreadID,
		&em.From,
		&fromName,
		&toAddressesJSON,
		&subject,
		&snippet,
		&em.Date,
		&em.SizeEstimate,
		&labelIDsJSON,
		&isSentInt,
		&csSent,
		&csRecv,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	em.FromName = fromName.String
	em.Subject = subject.String
	em.Snippet = snippet.String
	em.IsSent = isSentInt == 1
	em.SenderNeverEmailed = !em.IsSent && csRecv > 0 && csSent == 0
	em.SenderEmailCount = csRecv
	_ = json.Unmarshal([]byte(toAddressesJSON), &em.To)
	_ = json.Unmarshal([]byte(labelIDsJSON), &em.LabelIDs)

	return &em, nil
}

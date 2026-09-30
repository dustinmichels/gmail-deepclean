package api

import (
	"errors"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

func TestParseSingleEmail(t *testing.T) {
	tests := []struct {
		input    string
		wantAddr string
		wantName string
	}{
		{"John Doe <john@example.com>", "john@example.com", "John Doe"},
		{"<simple@test.org>", "simple@test.org", ""},
		{"plain@example.com", "plain@example.com", ""},
		{"\"Quotes, Complex\" <quotes@domain.com>", "quotes@domain.com", "Quotes, Complex"},
		{"", "", ""},
		{"   spaced@addr.com   ", "spaced@addr.com", ""},
	}

	for _, tt := range tests {
		gotAddr, gotName := parseSingleEmail(tt.input)
		if gotAddr != tt.wantAddr {
			t.Errorf("parseSingleEmail(%q) gotAddr = %q, want %q", tt.input, gotAddr, tt.wantAddr)
		}
		if gotName != tt.wantName {
			t.Errorf("parseSingleEmail(%q) gotName = %q, want %q", tt.input, gotName, tt.wantName)
		}
	}
}

func TestParseMultiEmail(t *testing.T) {
	input := "Alice <alice@test.com>, bob@test.com, \"Charlie\" <charlie@test.com>"
	results := parseMultiEmail(input)

	if len(results) != 3 {
		t.Fatalf("Expected 3 emails, got %d: %v", len(results), results)
	}

	expected := []string{"alice@test.com", "bob@test.com", "charlie@test.com"}
	for i, exp := range expected {
		if results[i] != exp {
			t.Errorf("Result[%d] = %s, want %s", i, results[i], exp)
		}
	}
}

func TestParseEmailDate(t *testing.T) {
	rfc1123 := "Mon, 02 Jan 2006 15:04:05 -0700"
	d1 := parseEmailDate(rfc1123)
	if d1.IsZero() {
		t.Errorf("Failed to parse RFC1123 date: %s", rfc1123)
	}

	rfc3339 := "2026-09-29T12:00:00Z"
	d2 := parseEmailDate(rfc3339)
	if d2.IsZero() || d2.Year() != 2026 {
		t.Errorf("Failed to parse RFC3339 date: %s", rfc3339)
	}

	empty := parseEmailDate("")
	if !empty.IsZero() {
		t.Errorf("Expected zero time for empty string, got %v", empty)
	}
}

func TestIsQuotaOrRateLimitErr(t *testing.T) {
	if isQuotaOrRateLimitErr(nil) {
		t.Errorf("nil should not be rate limit error")
	}

	err429 := &googleapi.Error{Code: 429, Message: "Too Many Requests"}
	if !isQuotaOrRateLimitErr(err429) {
		t.Errorf("HTTP 429 should be detected as rate limit error")
	}

	errQuotaStr := errors.New("User Rate Limit Exceeded")
	if !isQuotaOrRateLimitErr(errQuotaStr) {
		t.Errorf("Error string with 'Rate Limit' should be detected")
	}

	unrelatedErr := errors.New("unrelated connection error")
	if isQuotaOrRateLimitErr(unrelatedErr) {
		t.Errorf("unrelated error should not be detected as quota error")
	}
}

func TestNewInboxProcessorConfiguration(t *testing.T) {
	DB = setupTestDB(t)
	oauthConfig = &oauth2.Config{}

	token := &oauth2.Token{
		AccessToken: "mock-access-token",
	}

	proc, err := NewInboxProcessor("user123", "test@example.com", token)
	if err != nil {
		t.Fatalf("NewInboxProcessor failed: %v", err)
	}

	if proc.concurrency != 20 {
		t.Errorf("Expected concurrency=20, got %d", proc.concurrency)
	}
	if proc.rateLimiter == nil {
		t.Errorf("Expected non-nil rateLimiter")
	}
}

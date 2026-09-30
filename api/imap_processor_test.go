package api

import (
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestXOAuth2Client(t *testing.T) {
	client := NewXOAuth2Client("user@example.com", "ya29.secret-token")

	mech, ir, err := client.Start()
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	if mech != "XOAUTH2" {
		t.Errorf("Expected mechanism 'XOAUTH2', got '%s'", mech)
	}

	expectedIR := "user=user@example.com\x01auth=Bearer ya29.secret-token\x01\x01"
	if string(ir) != expectedIR {
		t.Errorf("Expected IR '%s', got '%s'", expectedIR, string(ir))
	}

	resp, err := client.Next([]byte(`{"status":"401"}`))
	if err != nil {
		t.Errorf("Next() returned error: %v", err)
	}
	if string(resp) != "" {
		t.Errorf("Expected empty response to challenge, got '%s'", string(resp))
	}
}

func TestDecodeMIMEHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Hello World", "Hello World"},
		{"=?UTF-8?B?4pyoIFlvdXIgd2Vla2x5IHVwZGF0ZQ==?=", "✨ Your weekly update"},
		{"=?ISO-8859-1?Q?Keld_J=F8rn_Simonsen?=", "Keld Jørn Simonsen"},
	}

	for _, tt := range tests {
		got := decodeMIMEHeader(tt.input)
		if got != tt.expected {
			t.Errorf("decodeMIMEHeader(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractXGmID(t *testing.T) {
	if got := extractXGmID(nil); got != "" {
		t.Errorf("extractXGmID(nil) = %s, expected empty", got)
	}

	// 16-hex characters test
	val := uint64(0x18b7625143a50d24)
	got := extractXGmID(val)
	if got != "18b7625143a50d24" {
		t.Errorf("extractXGmID(0x18b7625143a50d24) = %s, expected 18b7625143a50d24", got)
	}

	gotStr := extractXGmID("custom-id-123")
	if gotStr != "custom-id-123" {
		t.Errorf("extractXGmID string failed, got %s", gotStr)
	}
}

func TestHasDraftFlag(t *testing.T) {
	if !hasDraftFlag([]string{imap.DraftFlag}, nil) {
		t.Errorf("Expected true for imap.DraftFlag")
	}
	if !hasDraftFlag([]string{`\Draft`}, nil) {
		t.Errorf("Expected true for \\Draft")
	}
	if !hasDraftFlag(nil, []string{"DRAFT"}) {
		t.Errorf("Expected true for DRAFT label")
	}
	if hasDraftFlag([]string{imap.SeenFlag}, []string{"INBOX", "UNREAD"}) {
		t.Errorf("Expected false for non-draft")
	}
}

func TestConvertImapMessageToMetadata(t *testing.T) {
	proc := &InboxProcessor{
		userID:    "user@example.com",
		userEmail: "user@example.com",
	}

	now := time.Now().Truncate(time.Second)

	// Test 1: Received normal email
	msg := imap.NewMessage(1, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid})
	msg.Uid = 101
	msg.Size = 4096
	msg.Envelope = &imap.Envelope{
		Date:    now,
		Subject: "Hello there",
		From: []*imap.Address{
			{PersonalName: "Alice Smith", MailboxName: "alice", HostName: "example.org"},
		},
		To: []*imap.Address{
			{PersonalName: "User", MailboxName: "user", HostName: "example.com"},
		},
	}
	msg.Items = map[imap.FetchItem]interface{}{
		imap.FetchItem("X-GM-MSGID"):  uint64(0x123456789abcdef0),
		imap.FetchItem("X-GM-THRID"):  uint64(0xabcdef0123456789),
		imap.FetchItem("X-GM-LABELS"): []interface{}{"INBOX", "IMPORTANT"},
	}

	meta := proc.convertImapMessageToMetadata(msg)
	if meta == nil {
		t.Fatalf("Expected non-nil metadata")
	}

	if meta.ID != "123456789abcdef0" {
		t.Errorf("Expected ID 123456789abcdef0, got %s", meta.ID)
	}
	if meta.ThreadID != "abcdef0123456789" {
		t.Errorf("Expected ThreadID abcdef0123456789, got %s", meta.ThreadID)
	}
	if meta.From != "alice@example.org" {
		t.Errorf("Expected From alice@example.org, got %s", meta.From)
	}
	if meta.FromName != "Alice Smith" {
		t.Errorf("Expected FromName Alice Smith, got %s", meta.FromName)
	}
	if len(meta.To) != 1 || meta.To[0] != "user@example.com" {
		t.Errorf("Expected To user@example.com, got %+v", meta.To)
	}
	if meta.IsSent {
		t.Errorf("Expected IsSent=false for incoming email")
	}

	// Test 2: Sent email by user
	msgSent := imap.NewMessage(2, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid})
	msgSent.Uid = 102
	msgSent.Envelope = &imap.Envelope{
		Date:    now,
		Subject: "My response",
		From: []*imap.Address{
			{PersonalName: "User", MailboxName: "USER", HostName: "Example.COM"},
		},
		To: []*imap.Address{
			{PersonalName: "Alice", MailboxName: "alice", HostName: "example.org"},
		},
	}
	metaSent := proc.convertImapMessageToMetadata(msgSent)
	if metaSent == nil {
		t.Fatalf("Expected non-nil metadata")
	}
	if !metaSent.IsSent {
		t.Errorf("Expected IsSent=true for user-authored email")
	}

	// Test 3: Draft should be filtered out
	msgDraft := imap.NewMessage(3, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags})
	msgDraft.Flags = []string{imap.DraftFlag}
	metaDraft := proc.convertImapMessageToMetadata(msgDraft)
	if metaDraft != nil {
		t.Errorf("Expected draft to be nil, got %+v", metaDraft)
	}
}

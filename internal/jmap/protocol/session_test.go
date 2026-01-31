package protocol

import (
	"encoding/json"
	"testing"
)

func TestDiscoveryURL(t *testing.T) {
	tests := []struct {
		hostname string
		expected string
	}{
		{"example.com", "https://example.com/.well-known/jmap"},
		{"https://example.com", "https://example.com/.well-known/jmap"},
		{"http://example.com", "http://example.com/.well-known/jmap"},
		{"example.com/", "https://example.com/.well-known/jmap"},
		{"https://example.com/", "https://example.com/.well-known/jmap"},
	}

	for _, tt := range tests {
		result := DiscoveryURL(tt.hostname)
		if result != tt.expected {
			t.Errorf("DiscoveryURL(%q) = %q, want %q", tt.hostname, result, tt.expected)
		}
	}
}

func TestParseSession(t *testing.T) {
	sessionJSON := `{
		"capabilities": {
			"urn:ietf:params:jmap:core": {},
			"urn:ietf:params:jmap:mail": {}
		},
		"accounts": {
			"A123": {
				"name": "user@example.com",
				"isPersonal": true,
				"isReadOnly": false,
				"accountCapabilities": {}
			}
		},
		"primaryAccounts": {
			"urn:ietf:params:jmap:mail": "A123"
		},
		"username": "user@example.com",
		"apiUrl": "https://jmap.example.com/api/",
		"downloadUrl": "https://jmap.example.com/download/",
		"uploadUrl": "https://jmap.example.com/upload/",
		"eventSourceUrl": "https://jmap.example.com/events/",
		"state": "abc123"
	}`

	session, err := ParseSession([]byte(sessionJSON))
	if err != nil {
		t.Fatalf("ParseSession() error: %v", err)
	}

	if session.Username != "user@example.com" {
		t.Errorf("Username = %q, want %q", session.Username, "user@example.com")
	}

	if session.APIURL != "https://jmap.example.com/api/" {
		t.Errorf("APIURL = %q, want %q", session.APIURL, "https://jmap.example.com/api/")
	}

	if len(session.Capabilities) != 2 {
		t.Errorf("Capabilities count = %d, want 2", len(session.Capabilities))
	}

	if len(session.Accounts) != 1 {
		t.Errorf("Accounts count = %d, want 1", len(session.Accounts))
	}
}

func TestParseSession_Invalid(t *testing.T) {
	_, err := ParseSession([]byte("invalid json"))
	if err == nil {
		t.Error("ParseSession() expected error for invalid JSON")
	}
}

func TestSession_GetCapabilityNames(t *testing.T) {
	session := &Session{
		Capabilities: map[string]json.RawMessage{
			CoreCapability: []byte("{}"),
			MailCapability: []byte("{}"),
		},
	}

	names := session.GetCapabilityNames()
	if len(names) != 2 {
		t.Errorf("GetCapabilityNames() returned %d names, want 2", len(names))
	}
}

func TestSession_HasCapability(t *testing.T) {
	session := &Session{
		Capabilities: map[string]json.RawMessage{
			CoreCapability: []byte("{}"),
			MailCapability: []byte("{}"),
		},
	}

	tests := []struct {
		cap      string
		expected bool
	}{
		{CoreCapability, true},
		{MailCapability, true},
		{SubmissionCapability, false},
		{"unknown", false},
	}

	for _, tt := range tests {
		result := session.HasCapability(tt.cap)
		if result != tt.expected {
			t.Errorf("HasCapability(%q) = %v, want %v", tt.cap, result, tt.expected)
		}
	}
}

func TestSession_HasMailCapability(t *testing.T) {
	tests := []struct {
		name     string
		caps     map[string]json.RawMessage
		expected bool
	}{
		{
			name:     "has mail",
			caps:     map[string]json.RawMessage{MailCapability: []byte("{}")},
			expected: true,
		},
		{
			name:     "no mail",
			caps:     map[string]json.RawMessage{CoreCapability: []byte("{}")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{Capabilities: tt.caps}
			if session.HasMailCapability() != tt.expected {
				t.Errorf("HasMailCapability() = %v, want %v", session.HasMailCapability(), tt.expected)
			}
		})
	}
}

func TestSession_HasSubmissionCapability(t *testing.T) {
	tests := []struct {
		name     string
		caps     map[string]json.RawMessage
		expected bool
	}{
		{
			name:     "has submission",
			caps:     map[string]json.RawMessage{SubmissionCapability: []byte("{}")},
			expected: true,
		},
		{
			name:     "no submission",
			caps:     map[string]json.RawMessage{CoreCapability: []byte("{}")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{Capabilities: tt.caps}
			if session.HasSubmissionCapability() != tt.expected {
				t.Errorf("HasSubmissionCapability() = %v, want %v", session.HasSubmissionCapability(), tt.expected)
			}
		})
	}
}

func TestSession_GetPrimaryMailAccountId(t *testing.T) {
	tests := []struct {
		name        string
		primary     map[string]Id
		expectedId  Id
		expectedOK  bool
	}{
		{
			name:       "has primary mail",
			primary:    map[string]Id{MailCapability: "A123"},
			expectedId: "A123",
			expectedOK: true,
		},
		{
			name:       "no primary mail",
			primary:    map[string]Id{},
			expectedId: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{PrimaryAccounts: tt.primary}
			id, ok := session.GetPrimaryMailAccountId()
			if ok != tt.expectedOK {
				t.Errorf("GetPrimaryMailAccountId() ok = %v, want %v", ok, tt.expectedOK)
			}
			if id != tt.expectedId {
				t.Errorf("GetPrimaryMailAccountId() id = %q, want %q", id, tt.expectedId)
			}
		})
	}
}

func TestSession_GetAccountCount(t *testing.T) {
	tests := []struct {
		name     string
		accounts map[Id]Account
		expected int
	}{
		{
			name:     "no accounts",
			accounts: map[Id]Account{},
			expected: 0,
		},
		{
			name:     "one account",
			accounts: map[Id]Account{"A1": {Name: "test"}},
			expected: 1,
		},
		{
			name:     "multiple accounts",
			accounts: map[Id]Account{"A1": {Name: "test1"}, "A2": {Name: "test2"}},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{Accounts: tt.accounts}
			if session.GetAccountCount() != tt.expected {
				t.Errorf("GetAccountCount() = %d, want %d", session.GetAccountCount(), tt.expected)
			}
		})
	}
}

func TestSession_GetAccountNames(t *testing.T) {
	session := &Session{
		Accounts: map[Id]Account{
			"A1": {Name: "account1"},
			"A2": {Name: "account2"},
		},
	}

	names := session.GetAccountNames()
	if len(names) != 2 {
		t.Errorf("GetAccountNames() returned %d names, want 2", len(names))
	}
}

func TestSession_Validate(t *testing.T) {
	tests := []struct {
		name    string
		session *Session
		wantErr bool
	}{
		{
			name: "valid session",
			session: &Session{
				APIURL: "https://api.example.com",
				Capabilities: map[string]json.RawMessage{
					CoreCapability: []byte("{}"),
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiUrl",
			session: &Session{
				Capabilities: map[string]json.RawMessage{
					CoreCapability: []byte("{}"),
				},
			},
			wantErr: true,
		},
		{
			name: "missing capabilities",
			session: &Session{
				APIURL:       "https://api.example.com",
				Capabilities: map[string]json.RawMessage{},
			},
			wantErr: true,
		},
		{
			name: "missing core capability",
			session: &Session{
				APIURL: "https://api.example.com",
				Capabilities: map[string]json.RawMessage{
					MailCapability: []byte("{}"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSession_Summary(t *testing.T) {
	session := &Session{
		Username: "user@example.com",
		APIURL:   "https://api.example.com",
		Accounts: map[Id]Account{
			"A1": {Name: "test"},
		},
		Capabilities: map[string]json.RawMessage{
			CoreCapability: []byte("{}"),
		},
	}

	summary := session.Summary()
	if summary == "" {
		t.Error("Summary() returned empty string")
	}

	// Check that key info is present
	if !contains(summary, "user@example.com") {
		t.Error("Summary() should contain username")
	}
	if !contains(summary, "api.example.com") {
		t.Error("Summary() should contain API URL")
	}
}

func TestSession_GetCoreCapability(t *testing.T) {
	coreJSON := `{"maxSizeUpload": 50000000, "maxConcurrentUpload": 4, "maxSizeRequest": 10000000, "maxConcurrentRequests": 10, "maxCallsInRequest": 64, "maxObjectsInGet": 1000, "maxObjectsInSet": 500, "collationAlgorithms": ["i;ascii-casemap", "i;unicode-casemap"]}`

	session := &Session{
		Capabilities: map[string]json.RawMessage{
			CoreCapability: []byte(coreJSON),
		},
	}

	info, err := session.GetCoreCapability()
	if err != nil {
		t.Fatalf("GetCoreCapability() error: %v", err)
	}

	if info.MaxSizeUpload != 50000000 {
		t.Errorf("MaxSizeUpload = %d, want 50000000", info.MaxSizeUpload)
	}
	if info.MaxConcurrentUpload != 4 {
		t.Errorf("MaxConcurrentUpload = %d, want 4", info.MaxConcurrentUpload)
	}
	if info.MaxCallsInRequest != 64 {
		t.Errorf("MaxCallsInRequest = %d, want 64", info.MaxCallsInRequest)
	}
}

func TestSession_GetCoreCapability_Missing(t *testing.T) {
	session := &Session{
		Capabilities: map[string]json.RawMessage{},
	}

	_, err := session.GetCoreCapability()
	if err == nil {
		t.Error("GetCoreCapability() expected error when core capability missing")
	}
}

func TestSession_GetMailCapability(t *testing.T) {
	mailJSON := `{"maxMailboxesPerEmail": null, "maxMailboxDepth": null, "maxSizeMailboxName": 490, "maxSizeAttachmentsPerEmail": 50000000, "emailQuerySortOptions": ["receivedAt", "sentAt", "size", "from", "to", "subject"], "mayCreateTopLevelMailbox": true}`

	session := &Session{
		Capabilities: map[string]json.RawMessage{
			MailCapability: []byte(mailJSON),
		},
	}

	info, err := session.GetMailCapability()
	if err != nil {
		t.Fatalf("GetMailCapability() error: %v", err)
	}

	if info.MaxSizeMailboxName != 490 {
		t.Errorf("MaxSizeMailboxName = %d, want 490", info.MaxSizeMailboxName)
	}
	if info.MaxSizeAttachmentsPerEmail != 50000000 {
		t.Errorf("MaxSizeAttachmentsPerEmail = %d, want 50000000", info.MaxSizeAttachmentsPerEmail)
	}
	if !info.MayCreateTopLevelMailbox {
		t.Error("MayCreateTopLevelMailbox should be true")
	}
}

func TestSession_GetMailCapability_Missing(t *testing.T) {
	session := &Session{
		Capabilities: map[string]json.RawMessage{},
	}

	_, err := session.GetMailCapability()
	if err == nil {
		t.Error("GetMailCapability() expected error when mail capability missing")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

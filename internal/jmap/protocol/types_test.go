package protocol

import (
	"encoding/json"
	"testing"
)

func TestMailbox_JSON(t *testing.T) {
	mailboxJSON := `{
		"id": "mb1",
		"name": "Inbox",
		"parentId": null,
		"role": "inbox",
		"sortOrder": 1,
		"totalEmails": 100,
		"unreadEmails": 5,
		"totalThreads": 80,
		"unreadThreads": 3,
		"isSubscribed": true
	}`

	var mb Mailbox
	if err := json.Unmarshal([]byte(mailboxJSON), &mb); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if mb.Id != "mb1" {
		t.Errorf("Id = %q, want %q", mb.Id, "mb1")
	}
	if mb.Name != "Inbox" {
		t.Errorf("Name = %q, want %q", mb.Name, "Inbox")
	}
	if mb.ParentId != nil {
		t.Errorf("ParentId = %v, want nil", mb.ParentId)
	}
	if mb.Role == nil || *mb.Role != "inbox" {
		t.Errorf("Role = %v, want 'inbox'", mb.Role)
	}
	if mb.SortOrder != 1 {
		t.Errorf("SortOrder = %d, want 1", mb.SortOrder)
	}
	if mb.TotalEmails != 100 {
		t.Errorf("TotalEmails = %d, want 100", mb.TotalEmails)
	}
	if mb.UnreadEmails != 5 {
		t.Errorf("UnreadEmails = %d, want 5", mb.UnreadEmails)
	}
	if mb.TotalThreads != 80 {
		t.Errorf("TotalThreads = %d, want 80", mb.TotalThreads)
	}
	if mb.UnreadThreads != 3 {
		t.Errorf("UnreadThreads = %d, want 3", mb.UnreadThreads)
	}
	if !mb.IsSubscribed {
		t.Error("IsSubscribed should be true")
	}
}

func TestMailbox_WithParentId(t *testing.T) {
	mailboxJSON := `{
		"id": "mb2",
		"name": "Subfolder",
		"parentId": "mb1",
		"role": null,
		"sortOrder": 2,
		"totalEmails": 10,
		"unreadEmails": 0
	}`

	var mb Mailbox
	if err := json.Unmarshal([]byte(mailboxJSON), &mb); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if mb.ParentId == nil {
		t.Fatal("ParentId should not be nil")
	}
	if *mb.ParentId != "mb1" {
		t.Errorf("ParentId = %q, want %q", *mb.ParentId, "mb1")
	}
	if mb.Role != nil {
		t.Errorf("Role should be nil, got %v", *mb.Role)
	}
}

func TestMailboxRights_JSON(t *testing.T) {
	rightsJSON := `{
		"mayReadItems": true,
		"mayAddItems": true,
		"mayRemoveItems": true,
		"maySetSeen": true,
		"maySetKeywords": true,
		"mayCreateChild": false,
		"mayRename": false,
		"mayDelete": false,
		"maySubmit": true
	}`

	var rights MailboxRights
	if err := json.Unmarshal([]byte(rightsJSON), &rights); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !rights.MayReadItems {
		t.Error("MayReadItems should be true")
	}
	if !rights.MayAddItems {
		t.Error("MayAddItems should be true")
	}
	if !rights.MayRemoveItems {
		t.Error("MayRemoveItems should be true")
	}
	if !rights.MaySetSeen {
		t.Error("MaySetSeen should be true")
	}
	if !rights.MaySetKeywords {
		t.Error("MaySetKeywords should be true")
	}
	if rights.MayCreateChild {
		t.Error("MayCreateChild should be false")
	}
	if rights.MayRename {
		t.Error("MayRename should be false")
	}
	if rights.MayDelete {
		t.Error("MayDelete should be false")
	}
	if !rights.MaySubmit {
		t.Error("MaySubmit should be true")
	}
}

func TestEmail_JSON(t *testing.T) {
	emailJSON := `{
		"id": "email1",
		"blobId": "blob1",
		"threadId": "thread1",
		"mailboxIds": {"mb1": true},
		"keywords": {"$seen": true, "$flagged": false},
		"size": 12345,
		"receivedAt": "2024-01-15T10:30:00Z",
		"messageId": ["<msg123@example.com>"],
		"subject": "Test Subject",
		"from": [{"name": "Sender", "email": "sender@example.com"}],
		"to": [{"name": "Recipient", "email": "recipient@example.com"}],
		"preview": "This is a preview...",
		"hasAttachment": true
	}`

	var email Email
	if err := json.Unmarshal([]byte(emailJSON), &email); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if email.Id != "email1" {
		t.Errorf("Id = %q, want %q", email.Id, "email1")
	}
	if email.BlobId != "blob1" {
		t.Errorf("BlobId = %q, want %q", email.BlobId, "blob1")
	}
	if email.ThreadId != "thread1" {
		t.Errorf("ThreadId = %q, want %q", email.ThreadId, "thread1")
	}
	if len(email.MailboxIds) != 1 {
		t.Errorf("MailboxIds length = %d, want 1", len(email.MailboxIds))
	}
	if !email.MailboxIds["mb1"] {
		t.Error("MailboxIds[mb1] should be true")
	}
	if email.Size != 12345 {
		t.Errorf("Size = %d, want 12345", email.Size)
	}
	if email.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", email.Subject, "Test Subject")
	}
	if len(email.From) != 1 {
		t.Errorf("From length = %d, want 1", len(email.From))
	}
	if email.From[0].Email != "sender@example.com" {
		t.Errorf("From[0].Email = %q, want %q", email.From[0].Email, "sender@example.com")
	}
	if !email.HasAttachment {
		t.Error("HasAttachment should be true")
	}
}

func TestEmailAddress_JSON(t *testing.T) {
	addrJSON := `{"name": "John Doe", "email": "john@example.com"}`

	var addr EmailAddress
	if err := json.Unmarshal([]byte(addrJSON), &addr); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if addr.Name != "John Doe" {
		t.Errorf("Name = %q, want %q", addr.Name, "John Doe")
	}
	if addr.Email != "john@example.com" {
		t.Errorf("Email = %q, want %q", addr.Email, "john@example.com")
	}
}

func TestAccount_JSON(t *testing.T) {
	accountJSON := `{
		"name": "user@example.com",
		"isPersonal": true,
		"isReadOnly": false,
		"accountCapabilities": {
			"urn:ietf:params:jmap:mail": {}
		}
	}`

	var account Account
	if err := json.Unmarshal([]byte(accountJSON), &account); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if account.Name != "user@example.com" {
		t.Errorf("Name = %q, want %q", account.Name, "user@example.com")
	}
	if !account.IsPersonal {
		t.Error("IsPersonal should be true")
	}
	if account.IsReadOnly {
		t.Error("IsReadOnly should be false")
	}
	if len(account.AccountCapabilities) != 1 {
		t.Errorf("AccountCapabilities length = %d, want 1", len(account.AccountCapabilities))
	}
}

func TestId_Type(t *testing.T) {
	var id Id = "test-id-123"

	// Test string conversion
	if string(id) != "test-id-123" {
		t.Errorf("string(Id) = %q, want %q", string(id), "test-id-123")
	}

	// Test JSON marshaling
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if string(data) != `"test-id-123"` {
		t.Errorf("JSON = %s, want %q", string(data), `"test-id-123"`)
	}

	// Test JSON unmarshaling
	var id2 Id
	if err := json.Unmarshal([]byte(`"another-id"`), &id2); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if id2 != "another-id" {
		t.Errorf("Id = %q, want %q", id2, "another-id")
	}
}

func TestGetMailboxesResponse(t *testing.T) {
	respJSON := `{
		"accountId": "A123",
		"state": "state456",
		"list": [
			{"id": "mb1", "name": "Inbox", "totalEmails": 100, "unreadEmails": 5}
		],
		"notFound": ["missing1"]
	}`

	var resp GetMailboxesResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if resp.AccountId != "A123" {
		t.Errorf("AccountId = %q, want %q", resp.AccountId, "A123")
	}
	if resp.State != "state456" {
		t.Errorf("State = %q, want %q", resp.State, "state456")
	}
	if len(resp.List) != 1 {
		t.Errorf("List length = %d, want 1", len(resp.List))
	}
	if len(resp.NotFound) != 1 {
		t.Errorf("NotFound length = %d, want 1", len(resp.NotFound))
	}
	if resp.NotFound[0] != "missing1" {
		t.Errorf("NotFound[0] = %q, want %q", resp.NotFound[0], "missing1")
	}
}

func TestQueryEmailsResponse(t *testing.T) {
	respJSON := `{
		"accountId": "A123",
		"queryState": "qstate789",
		"canCalculateChanges": true,
		"position": 0,
		"total": 250,
		"ids": ["e1", "e2", "e3"]
	}`

	var resp QueryEmailsResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if resp.AccountId != "A123" {
		t.Errorf("AccountId = %q, want %q", resp.AccountId, "A123")
	}
	if resp.QueryState != "qstate789" {
		t.Errorf("QueryState = %q, want %q", resp.QueryState, "qstate789")
	}
	if !resp.CanCalculateChanges {
		t.Error("CanCalculateChanges should be true")
	}
	if resp.Position != 0 {
		t.Errorf("Position = %d, want 0", resp.Position)
	}
	if resp.Total != 250 {
		t.Errorf("Total = %d, want 250", resp.Total)
	}
	if len(resp.Ids) != 3 {
		t.Errorf("Ids length = %d, want 3", len(resp.Ids))
	}
}

func TestGetEmailsResponse(t *testing.T) {
	respJSON := `{
		"accountId": "A123",
		"state": "estate123",
		"list": [
			{"id": "e1", "subject": "Test Email", "size": 1000}
		],
		"notFound": []
	}`

	var resp GetEmailsResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if resp.AccountId != "A123" {
		t.Errorf("AccountId = %q, want %q", resp.AccountId, "A123")
	}
	if resp.State != "estate123" {
		t.Errorf("State = %q, want %q", resp.State, "estate123")
	}
	if len(resp.List) != 1 {
		t.Errorf("List length = %d, want 1", len(resp.List))
	}
	if resp.List[0].Subject != "Test Email" {
		t.Errorf("List[0].Subject = %q, want %q", resp.List[0].Subject, "Test Email")
	}
}

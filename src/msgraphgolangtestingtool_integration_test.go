//go:build integration
// +build integration

// Integration tests for Microsoft Graph EXO Mails/Calendar Golang Testing Tool
// These tests make real API calls to Microsoft Graph and require valid credentials.
//
// Usage:
//   Set environment variables:
//     MSGRAPHTENANTID, MSGRAPHCLIENTID, MSGRAPHSECRET, MSGRAPHMAILBOX
//   Run tests:
//     go test -tags=integration -v ./src
//
// IMPORTANT: These tests will:
//   - Make real API calls to Microsoft Graph
//   - Send actual emails and create calendar events
//   - Consume API quota
//   - Require a test mailbox with proper permissions
//
// Only run these tests in a dedicated test environment!

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestIntegration_Prerequisites checks that all required environment variables are set
func TestIntegration_Prerequisites(t *testing.T) {
	requiredEnvVars := []string{
		"MSGRAPHTENANTID",
		"MSGRAPHCLIENTID",
		"MSGRAPHSECRET",
		"MSGRAPHMAILBOX",
	}

	missing := []string{}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		t.Skipf("Skipping integration tests - missing required environment variables: %v", missing)
	}
}

// TestIntegration_GraphClientCreation tests that we can create a Graph API client with the provided credentials
func TestIntegration_GraphClientCreation(t *testing.T) {
	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	if client == nil {
		t.Fatal("Graph client is nil")
	}

	t.Log("✅ Graph client created successfully")
}

// TestIntegration_ListEvents tests retrieving calendar events
func TestIntegration_ListEvents(t *testing.T) {
	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	t.Logf("Retrieving %d upcoming calendar events from %s", config.Count, config.Mailbox)

	err = listEvents(ctx, client, config.Mailbox, config.Count, config, nil)
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}

	t.Log("✅ Successfully retrieved calendar events")
}

// TestIntegration_ListInbox tests retrieving inbox messages
func TestIntegration_ListInbox(t *testing.T) {
	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	t.Logf("Retrieving %d newest inbox messages from %s", config.Count, config.Mailbox)

	err = listInbox(ctx, client, config.Mailbox, config.Count, config, nil)
	if err != nil {
		t.Fatalf("Failed to list inbox: %v", err)
	}

	t.Log("✅ Successfully retrieved inbox messages")
}

// TestIntegration_CheckAvailability tests checking recipient availability
func TestIntegration_CheckAvailability(t *testing.T) {
	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	// Use the mailbox as the recipient (check own availability)
	recipient := config.Mailbox
	t.Logf("Checking availability for %s (next working day at 12:00 UTC)", recipient)

	// Set the To field for validation
	config.To = []string{recipient}

	err = checkAvailability(ctx, client, config.Mailbox, recipient, config, nil)
	if err != nil {
		t.Fatalf("Failed to check availability: %v", err)
	}

	t.Log("✅ Successfully checked availability")
}

// TestIntegration_SendEmail tests sending an email (sends to self)
func TestIntegration_SendEmail(t *testing.T) {
	if os.Getenv("MSGRAPH_INTEGRATION_WRITE") != "true" {
		t.Skip("Skipping write operation test - set MSGRAPH_INTEGRATION_WRITE=true to enable")
	}

	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	subject := fmt.Sprintf("Integration Test Email - %s", time.Now().Format(time.RFC3339))
	body := "This is an automated integration test email from the Microsoft Graph EXO Mails/Calendar Golang Testing Tool. Safe to delete."
	to := []string{config.Mailbox} // Send to self

	t.Logf("Sending test email to %s", config.Mailbox)
	t.Logf("  Subject: %s", subject)

	sendEmail(ctx, client, config.Mailbox, to, nil, nil, subject, body, "", nil, config, nil)

	t.Log("✅ Email sent successfully")
	t.Log("  Check your inbox to verify delivery")
	t.Log("  Note: Email delivery may take a few seconds")
}

// TestIntegration_CreateCalendarEvent tests creating a calendar invite
func TestIntegration_CreateCalendarEvent(t *testing.T) {
	if os.Getenv("MSGRAPH_INTEGRATION_WRITE") != "true" {
		t.Skip("Skipping write operation test - set MSGRAPH_INTEGRATION_WRITE=true to enable")
	}

	config := loadTestConfig(t)
	ctx := context.Background()

	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	subject := fmt.Sprintf("Integration Test Event - %s", time.Now().Format("2006-01-02 15:04"))
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)          // Tomorrow
	endTime := time.Now().Add(24*time.Hour + 1*time.Hour).Format(time.RFC3339) // Tomorrow + 1 hour

	t.Log("Creating test calendar event...")
	t.Logf("  Subject: %s", subject)
	t.Logf("  Start: %s", startTime)
	t.Logf("  End: %s", endTime)

	createInvite(ctx, client, config.Mailbox, subject, startTime, endTime, config, nil)

	t.Log("✅ Calendar event created successfully")
	t.Log("  Check your calendar to verify the event")
}

// TestIntegration_ValidateConfiguration tests the configuration validation logic
func TestIntegration_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name: "valid configuration",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "test-secret",
				Action:   ActionGetInbox,
			},
			wantError: false,
		},
		{
			name: "invalid tenant ID",
			config: &Config{
				TenantID: "invalid",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Secret:   "test-secret",
				Action:   ActionGetInbox,
			},
			wantError: true,
		},
		{
			name: "invalid email",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "invalid-email",
				Secret:   "test-secret",
				Action:   ActionGetInbox,
			},
			wantError: true,
		},
		{
			name: "no authentication method",
			config: &Config{
				TenantID: "12345678-1234-1234-1234-123456789012",
				ClientID: "abcdefgh-5678-9012-abcd-ef1234567890",
				Mailbox:  "user@example.com",
				Action:   ActionGetInbox,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("validateConfiguration() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// loadTestConfig loads configuration from environment variables for testing
func loadTestConfig(t *testing.T) *Config {
	t.Helper()

	tenantID := os.Getenv("MSGRAPHTENANTID")
	clientID := os.Getenv("MSGRAPHCLIENTID")
	secret := os.Getenv("MSGRAPHSECRET")
	mailbox := os.Getenv("MSGRAPHMAILBOX")

	if tenantID == "" || clientID == "" || secret == "" || mailbox == "" {
		t.Skip("Skipping integration test - required environment variables not set")
	}

	config := &Config{
		TenantID:    tenantID,
		ClientID:    clientID,
		Secret:      secret,
		Mailbox:     mailbox,
		VerboseMode: false,
		Count:       3,
		MaxRetries:  3,
		RetryDelay:  2000 * time.Millisecond,
	}

	return config
}

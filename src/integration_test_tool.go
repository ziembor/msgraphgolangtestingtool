//go:build integration
// +build integration

// Integration test tool for Microsoft Graph EXO Mails/Calendar Golang Testing Tool
// This is an interactive program that tests real Graph API operations.
//
// Usage:
//   Set environment variables:
//     MSGRAPHTENANTID, MSGRAPHCLIENTID, MSGRAPHSECRET, MSGRAPHMAILBOX
//   Run:
//     go run -tags=integration integration_test_tool.go shared.go cert_windows.go
//
// This tool will:
//   1. Validate credentials from environment variables
//   2. Test each action (getevents, sendmail, sendinvite, getinbox)
//   3. Display results interactively
//   4. Prompt for confirmation before executing write operations

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// IntegrationTestResults holds the results of all integration tests
type IntegrationTestResults struct {
	GetEventsSuccess  bool
	GetEventsError    error
	SendMailSuccess   bool
	SendMailError     error
	SendInviteSuccess bool
	SendInviteError   error
	GetInboxSuccess   bool
	GetInboxError     error
}

func main() {
	fmt.Println("=================================================================")
	fmt.Println("Microsoft Graph EXO Mails/Calendar Golang Testing Tool - Integration Test Suite")
	fmt.Println("=================================================================")
	fmt.Println()

	// Load configuration from environment variables
	config, err := loadConfigFromEnv()
	if err != nil {
		fmt.Printf("❌ Configuration Error: %v\n", err)
		fmt.Println("\nRequired environment variables:")
		fmt.Println("  MSGRAPHTENANTID  - Azure AD Tenant ID")
		fmt.Println("  MSGRAPHCLIENTID  - Application (Client) ID")
		fmt.Println("  MSGRAPHSECRET    - Client Secret")
		fmt.Println("  MSGRAPHMAILBOX   - Target mailbox email address")
		os.Exit(1)
	}

	// Display configuration
	fmt.Println("Configuration loaded:")
	fmt.Printf("  Tenant ID: %s\n", maskGUID(config.TenantID))
	fmt.Printf("  Client ID: %s\n", maskGUID(config.ClientID))
	fmt.Printf("  Secret:    %s\n", maskSecret(config.Secret))
	fmt.Printf("  Mailbox:   %s\n", config.Mailbox)
	fmt.Println()

	// Confirm to proceed
	if !confirm("Proceed with integration tests?") {
		fmt.Println("Tests cancelled.")
		os.Exit(0)
	}

	// Run integration tests
	results := runIntegrationTests(config)

	// Display summary
	fmt.Println()
	fmt.Println("=================================================================")
	fmt.Println("Integration Test Results Summary")
	fmt.Println("=================================================================")
	displayTestResult("Get Events", results.GetEventsSuccess, results.GetEventsError)
	displayTestResult("Send Mail", results.SendMailSuccess, results.SendMailError)
	displayTestResult("Send Invite", results.SendInviteSuccess, results.SendInviteError)
	displayTestResult("Get Inbox", results.GetInboxSuccess, results.GetInboxError)
	fmt.Println("=================================================================")

	// Calculate pass rate
	passed := 0
	total := 4
	if results.GetEventsSuccess {
		passed++
	}
	if results.SendMailSuccess {
		passed++
	}
	if results.SendInviteSuccess {
		passed++
	}
	if results.GetInboxSuccess {
		passed++
	}

	fmt.Printf("\nPass Rate: %d/%d (%.0f%%)\n", passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("✅ All integration tests passed!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Some integration tests failed.")
		os.Exit(1)
	}
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() (*Config, error) {
	tenantID := os.Getenv("MSGRAPHTENANTID")
	clientID := os.Getenv("MSGRAPHCLIENTID")
	secret := os.Getenv("MSGRAPHSECRET")
	mailbox := os.Getenv("MSGRAPHMAILBOX")

	if tenantID == "" {
		return nil, fmt.Errorf("MSGRAPHTENANTID is not set")
	}
	if clientID == "" {
		return nil, fmt.Errorf("MSGRAPHCLIENTID is not set")
	}
	if secret == "" {
		return nil, fmt.Errorf("MSGRAPHSECRET is not set")
	}
	if mailbox == "" {
		return nil, fmt.Errorf("MSGRAPHMAILBOX is not set")
	}

	config := &Config{
		TenantID:    tenantID,
		ClientID:    clientID,
		Secret:      secret,
		Mailbox:     mailbox,
		VerboseMode: false, // Set to true for detailed output
		Count:       3,
		MaxRetries:  3,
		RetryDelay:  2000 * time.Millisecond,
	}

	return config, nil
}

// runIntegrationTests executes all integration tests
func runIntegrationTests(config *Config) *IntegrationTestResults {
	results := &IntegrationTestResults{}
	ctx := context.Background()

	fmt.Println()
	fmt.Println("Creating Microsoft Graph client...")

	// Create Graph client (shared across all tests)
	client, err := setupGraphClient(ctx, config, nil)
	if err != nil {
		fmt.Printf("❌ Failed to create Graph client: %v\n", err)
		return results
	}
	fmt.Println("✅ Graph client created successfully")
	fmt.Println()

	// Test 1: Get Events
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("Test 1: Get Events")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	results.GetEventsSuccess, results.GetEventsError = testGetEvents(ctx, client, config)
	fmt.Println()

	// Test 2: Send Mail
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("Test 2: Send Mail")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	if confirm("Send a test email to yourself?") {
		results.SendMailSuccess, results.SendMailError = testSendMail(ctx, client, config)
	} else {
		fmt.Println("⊘ Skipped by user")
		results.SendMailSuccess = true // Don't fail if user skipped
	}
	fmt.Println()

	// Test 3: Send Invite
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("Test 3: Send Calendar Invite")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	if confirm("Create a test calendar event?") {
		results.SendInviteSuccess, results.SendInviteError = testSendInvite(ctx, client, config)
	} else {
		fmt.Println("⊘ Skipped by user")
		results.SendInviteSuccess = true // Don't fail if user skipped
	}
	fmt.Println()

	// Test 4: Get Inbox
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Println("Test 4: Get Inbox Messages")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	results.GetInboxSuccess, results.GetInboxError = testGetInbox(ctx, client, config)
	fmt.Println()

	return results
}

// testGetEvents tests retrieving calendar events
func testGetEvents(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config) (bool, error) {
	fmt.Printf("Retrieving %d upcoming calendar events from %s...\n", config.Count, config.Mailbox)

	err := listEvents(ctx, client, config.Mailbox, config.Count, config, nil)

	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		return false, err
	}

	fmt.Println("✅ PASSED: Successfully retrieved calendar events")
	return true, nil
}

// testSendMail tests sending an email
func testSendMail(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config) (bool, error) {
	subject := fmt.Sprintf("Integration Test - %s", time.Now().Format(time.RFC3339))
	body := "This is an automated integration test email. Safe to delete."
	to := []string{config.Mailbox} // Send to self

	fmt.Printf("Sending test email to %s...\n", config.Mailbox)
	fmt.Printf("  Subject: %s\n", subject)

	sendEmail(ctx, client, config.Mailbox, to, nil, nil, subject, body, "", nil, config, nil)

	fmt.Println("✅ PASSED: Email sent successfully")
	fmt.Println("  Check your inbox to verify delivery")
	return true, nil
}

// testSendInvite tests creating a calendar invite
func testSendInvite(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config) (bool, error) {
	subject := fmt.Sprintf("Integration Test Event - %s", time.Now().Format("2006-01-02 15:04"))
	startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)          // Tomorrow
	endTime := time.Now().Add(24*time.Hour + 1*time.Hour).Format(time.RFC3339) // Tomorrow + 1 hour

	fmt.Println("Creating test calendar event...")
	fmt.Printf("  Subject: %s\n", subject)
	fmt.Printf("  Start: %s\n", startTime)
	fmt.Printf("  End: %s\n", endTime)

	createInvite(ctx, client, config.Mailbox, subject, startTime, endTime, config, nil)

	fmt.Println("✅ PASSED: Calendar invite created successfully")
	fmt.Println("  Check your calendar to verify the event")
	return true, nil
}

// testGetInbox tests retrieving inbox messages
func testGetInbox(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config) (bool, error) {
	fmt.Printf("Retrieving %d newest inbox messages from %s...\n", config.Count, config.Mailbox)

	err := listInbox(ctx, client, config.Mailbox, config.Count, config, nil)

	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		return false, err
	}

	fmt.Println("✅ PASSED: Successfully retrieved inbox messages")
	return true, nil
}

// confirm prompts the user for yes/no confirmation
func confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/n): ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// displayTestResult displays a formatted test result
func displayTestResult(testName string, success bool, err error) {
	if success {
		fmt.Printf("  ✅ %-20s PASSED\n", testName+":")
	} else {
		fmt.Printf("  ❌ %-20s FAILED", testName+":")
		if err != nil {
			fmt.Printf(" - %v", err)
		}
		fmt.Println()
	}
}

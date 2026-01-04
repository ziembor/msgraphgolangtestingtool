# Integration Testing Guide

This guide explains how to run interactive integration tests for the Microsoft Graph GoLang Testing Tool.

## Overview

The integration test tool (`integration_test_tool.go`) is a standalone program that tests real Microsoft Graph API operations interactively. Unlike unit tests, these tests:

- ✅ Make real API calls to Microsoft Graph
- ✅ Use actual Azure AD authentication
- ✅ Send real emails and create real calendar events
- ✅ Provide interactive confirmation before write operations
- ✅ Display results in real-time

## Prerequisites

1. **Azure AD App Registration** with the following permissions:
   - `Mail.Send` (for sending emails)
   - `Mail.Read` (for reading inbox)
   - `Calendars.ReadWrite` (for calendar operations)
   - **Admin Consent** must be granted

2. **Client Secret** (certificate authentication not used in integration tests)

3. **Test Mailbox** - a mailbox you have access to for testing

## Setup

### 1. Set Environment Variables

**PowerShell:**
```powershell
$env:MSGRAPHTENANTID = "12345678-1234-1234-1234-123456789012"
$env:MSGRAPHCLIENTID = "abcdefgh-5678-9012-abcd-ef1234567890"
$env:MSGRAPHSECRET = "your-client-secret-here"
$env:MSGRAPHMAILBOX = "test-user@example.com"
```

**Command Prompt:**
```cmd
set MSGRAPHTENANTID=12345678-1234-1234-1234-123456789012
set MSGRAPHCLIENTID=abcdefgh-5678-9012-abcd-ef1234567890
set MSGRAPHSECRET=your-client-secret-here
set MSGRAPHMAILBOX=test-user@example.com
```

**Linux/macOS:**
```bash
export MSGRAPHTENANTID="12345678-1234-1234-1234-123456789012"
export MSGRAPHCLIENTID="abcdefgh-5678-9012-abcd-ef1234567890"
export MSGRAPHSECRET="your-client-secret-here"
export MSGRAPHMAILBOX="test-user@example.com"
```

### 2. Verify Environment Variables

**PowerShell:**
```powershell
# Check that all required variables are set
if ($env:MSGRAPHTENANTID -and $env:MSGRAPHCLIENTID -and $env:MSGRAPHSECRET -and $env:MSGRAPHMAILBOX) {
    Write-Host "✅ All environment variables are set" -ForegroundColor Green
} else {
    Write-Host "❌ Missing environment variables" -ForegroundColor Red
}
```

## Running Integration Tests

### Method 1: Using PowerShell Script (Recommended)

```powershell
.\run-integration-tests.ps1
```

### Method 2: Direct Execution

```powershell
cd src
go run integration_test_tool.go
```

### Method 3: Build and Run

```powershell
# Build the integration test tool (requires -tags integration)
cd src
go build -tags integration -o ../integration_test_tool.exe integration_test_tool.go msgraphgolangtestingtool_lib.go cert_windows.go

# Run it
cd ..
.\integration_test_tool.exe
```

## What Gets Tested

The integration test suite executes the following tests:

### Test 1: Get Events (Read-Only) ✅ Auto-runs
- **Action:** Retrieves calendar events from the test mailbox
- **Verifies:** Authentication works, API connectivity, calendar read permissions
- **Side Effects:** None (read-only operation)

### Test 2: Send Mail (Write Operation) ⚠️ Requires Confirmation
- **Action:** Sends a test email to the test mailbox (to self)
- **Subject:** "Integration Test - [timestamp]"
- **Body:** "This is an automated integration test email. Safe to delete."
- **Verifies:** Email sending permissions, SMTP routing
- **Side Effects:** Creates an email in your inbox

### Test 3: Send Calendar Invite (Write Operation) ⚠️ Requires Confirmation
- **Action:** Creates a calendar event for tomorrow
- **Subject:** "Integration Test Event - [timestamp]"
- **Start:** Tomorrow at current time
- **End:** Tomorrow + 1 hour
- **Verifies:** Calendar write permissions, event creation
- **Side Effects:** Creates a calendar event

### Test 4: Get Inbox (Read-Only) ✅ Auto-runs
- **Action:** Retrieves newest inbox messages
- **Verifies:** Mail read permissions, inbox access
- **Side Effects:** None (read-only operation)

## Example Output

```
=================================================================
Microsoft Graph GoLang Testing Tool - Integration Test Suite
=================================================================

Configuration loaded:
  Tenant ID: 1234****-****-****-****9012
  Client ID: abcd****-****-****-****7890
  Secret:    your********cret
  Mailbox:   test-user@example.com

Proceed with integration tests? (y/n): y

Creating Microsoft Graph client...
✅ Graph client created successfully

─────────────────────────────────────────────────────────────────
Test 1: Get Events
─────────────────────────────────────────────────────────────────
Retrieving 3 upcoming calendar events from test-user@example.com...

Event 1:
  Subject: Team Meeting
  Start: 2026-01-05T10:00:00Z
  End: 2026-01-05T11:00:00Z

Event 2:
  Subject: Project Review
  Start: 2026-01-06T14:00:00Z
  End: 2026-01-06T15:00:00Z

✅ PASSED: Successfully retrieved calendar events

─────────────────────────────────────────────────────────────────
Test 2: Send Mail
─────────────────────────────────────────────────────────────────
Send a test email to yourself? (y/n): y
Sending test email to test-user@example.com...
  Subject: Integration Test - 2026-01-04T14:30:00Z
✅ PASSED: Email sent successfully
  Check your inbox to verify delivery

─────────────────────────────────────────────────────────────────
Test 3: Send Calendar Invite
─────────────────────────────────────────────────────────────────
Create a test calendar event? (y/n): y
Creating test calendar event...
  Subject: Integration Test Event - 2026-01-04 14:30
  Start: 2026-01-05T14:30:00Z
  End: 2026-01-05T15:30:00Z
✅ PASSED: Calendar invite created successfully
  Check your calendar to verify the event

─────────────────────────────────────────────────────────────────
Test 4: Get Inbox Messages
─────────────────────────────────────────────────────────────────
Retrieving 3 newest inbox messages from test-user@example.com...

Message 1:
  From: sender@example.com
  To: test-user@example.com
  Subject: Weekly Report
  Received: 2026-01-04T10:15:30Z

Message 2:
  From: notifications@service.com
  To: test-user@example.com
  Subject: System Alert
  Received: 2026-01-04T09:45:00Z

✅ PASSED: Successfully retrieved inbox messages

=================================================================
Integration Test Results Summary
=================================================================
  ✅ Get Events:          PASSED
  ✅ Send Mail:           PASSED
  ✅ Send Invite:         PASSED
  ✅ Get Inbox:           PASSED
=================================================================

Pass Rate: 4/4 (100%)
✅ All integration tests passed!
```

## Interactive Confirmations

The tool prompts for confirmation before:
1. **Starting the test suite** - "Proceed with integration tests?"
2. **Sending email** - "Send a test email to yourself?"
3. **Creating calendar event** - "Create a test calendar event?"

You can skip write operations by answering "n" (no). Skipped tests are not counted as failures.

## Troubleshooting

### Authentication Errors

**Error:** "Failed to create Graph client: authentication setup failed"

**Solutions:**
- Verify all environment variables are set correctly
- Check that the client secret hasn't expired
- Ensure the App Registration has the correct permissions
- Verify admin consent has been granted

### Permission Errors

**Error:** "Insufficient privileges to complete the operation"

**Solutions:**
- Check API permissions in Azure AD App Registration:
  - Mail.Send
  - Mail.Read
  - Calendars.ReadWrite
- Grant Admin Consent for these permissions
- Wait 5-10 minutes for permissions to propagate

### Mailbox Access Denied

**Error:** "Access is denied. Check credentials and try again."

**Solutions:**
- Verify the mailbox address is correct
- Ensure the mailbox exists and is licensed
- Check if Application Access Policies restrict this app
- Verify the service principal has access to the mailbox

### Network Errors

**Error:** "dial tcp: i/o timeout" or "connection timeout"

**Solutions:**
- Check network connectivity to graph.microsoft.com
- If behind a proxy, set MSGRAPHPROXY environment variable:
  ```powershell
  $env:MSGRAPHPROXY = "http://proxy.company.com:8080"
  ```

## Cleanup

After testing, you may want to:

1. **Delete test emails** from your inbox
2. **Delete test calendar events** from your calendar
3. **Clear environment variables** (for security):

```powershell
Remove-Item Env:\MSGRAPHSECRET
Remove-Item Env:\MSGRAPHTENANTID
Remove-Item Env:\MSGRAPHCLIENTID
Remove-Item Env:\MSGRAPHMAILBOX
```

## Security Best Practices

⚠️ **Important Security Notes:**

1. **Never commit credentials** to source control
2. **Use environment variables** for all sensitive data
3. **Clear secrets after testing:**
   ```powershell
   Remove-Item Env:\MSGRAPHSECRET
   ```
4. **Use dedicated test mailbox** - don't use production mailboxes
5. **Rotate secrets regularly** - generate new client secrets every 90 days
6. **Monitor API usage** - check Azure AD sign-in logs for test activity

## CI/CD Integration

To run integration tests in CI/CD pipelines:

```yaml
# Example: GitHub Actions
steps:
  - name: Run Integration Tests
    env:
      MSGRAPHTENANTID: ${{ secrets.MSGRAPH_TENANT_ID }}
      MSGRAPHCLIENTID: ${{ secrets.MSGRAPH_CLIENT_ID }}
      MSGRAPHSECRET: ${{ secrets.MSGRAPH_SECRET }}
      MSGRAPHMAILBOX: ${{ secrets.MSGRAPH_TEST_MAILBOX }}
    run: |
      cd src
      go run integration_test_tool.go
```

**Note:** For CI/CD, you may want to modify the tool to skip interactive confirmations:
- Set an environment variable like `MSGRAPH_AUTO_CONFIRM=true`
- Update the `confirm()` function to auto-return `true` when this is set

## Limitations

- **Only tests client secret authentication** - certificate auth not tested
- **Requires manual verification** - check inbox/calendar to confirm operations
- **No automatic cleanup** - test emails and events must be deleted manually
- **Network dependent** - requires internet access to graph.microsoft.com
- **API quota consumption** - each test run consumes API calls against your quota

## Support

For issues or questions:
- See main [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common errors
- Check [SECURITY_PRACTICES.md](SECURITY_PRACTICES.md) for security guidance
- Review [README.md](README.md) for general usage information

---

*Integration Testing Guide - Version 1.15.2 - 2026-01-04*

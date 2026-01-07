# Example Usage - msgraphgolangtestingtool.exe

This document provides comprehensive examples of how to use `msgraphgolangtestingtool.exe` for various Microsoft Graph API operations.

**Prerequisites:** Set authentication environment variables:

```powershell
$env:MSGRAPHTENANTID = "your-tenant-id"
$env:MSGRAPHCLIENTID = "your-client-id"
$env:MSGRAPHSECRET = "your-secret"  # or use -pfx/-thumbprint
$env:MSGRAPHMAILBOX = "user@example.com"
```

---

## 1. Get Calendar Events

```powershell
# Get default 3 upcoming events
./msgraphgolangtestingtool.exe -action getevents

# Get 10 upcoming events
./msgraphgolangtestingtool.exe -action getevents -count 10

# Get 5 events with verbose output
./msgraphgolangtestingtool.exe -action getevents -count 5 -verbose
```

---

## 2. Send Email - Basic

```powershell
# Send to self (default behavior when no recipients specified)
./msgraphgolangtestingtool.exe -action sendmail

# Send to specific recipient
./msgraphgolangtestingtool.exe -action sendmail -to "recipient@example.com"

# Send with custom subject and body
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "Test Email" \
    -body "This is a test message"

# Send to multiple recipients (comma-separated)
./msgraphgolangtestingtool.exe -action sendmail \
    -to "user1@example.com,user2@example.com" \
    -subject "Team Update"
```

---

## 3. Send Email - With CC/BCC

```powershell
# Send with CC recipients
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -cc "cc1@example.com,cc2@example.com" \
    -subject "Meeting Notes"

# Send with BCC recipients
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -bcc "bcc@example.com" \
    -subject "Confidential Update"

# Send with To, CC, and BCC
./msgraphgolangtestingtool.exe -action sendmail \
    -to "primary@example.com" \
    -cc "cc1@example.com,cc2@example.com" \
    -bcc "bcc@example.com" \
    -subject "Quarterly Report" \
    -body "Please review the attached report."
```

---

## 4. Send Email - HTML Content

```powershell
# Send HTML email
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "HTML Email Test" \
    -bodyHTML "<h1>Hello</h1><p>This is an <strong>HTML</strong> email.</p>"

# Send both text and HTML (multipart MIME)
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "Multipart Email" \
    -body "This is the plain text version" \
    -bodyHTML "<h1>HTML Version</h1><p>This is the <em>HTML</em> version</p>"
```

---

## 5. Send Email - With Attachments

```powershell
# Send with single attachment
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "Document Attached" \
    -attachments "C:\Reports\report.pdf"

# Send with multiple attachments (comma-separated)
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "Multiple Files" \
    -attachments "C:\Files\doc1.pdf,C:\Files\spreadsheet.xlsx,C:\Files\image.png"

# Send HTML email with attachments
./msgraphgolangtestingtool.exe -action sendmail \
    -to "recipient@example.com" \
    -subject "Report with Charts" \
    -bodyHTML "<h1>Monthly Report</h1><p>See attached files.</p>" \
    -attachments "C:\Reports\report.pdf,C:\Charts\chart.png"
```

---

## 6. Create Calendar Invites

```powershell
# Create invite with default subject and time (now + 1 hour)
./msgraphgolangtestingtool.exe -action sendinvite

# Create invite with custom subject
./msgraphgolangtestingtool.exe -action sendinvite -invite-subject "Team Meeting"

# Create invite with specific start time
./msgraphgolangtestingtool.exe -action sendinvite \
    -invite-subject "Project Review" \
    -start "2026-01-15T14:00:00Z"

# Create invite with start and end times
./msgraphgolangtestingtool.exe -action sendinvite \
    -invite-subject "Weekly Standup" \
    -start "2026-01-15T10:00:00Z" \
    -end "2026-01-15T10:30:00Z"

# Create all-day event (midnight to midnight next day)
./msgraphgolangtestingtool.exe -action sendinvite \
    -invite-subject "Conference Day" \
    -start "2026-02-01T00:00:00Z" \
    -end "2026-02-02T00:00:00Z"
```

---

## 7. Get Inbox Messages

```powershell
# Get default 3 newest messages
./msgraphgolangtestingtool.exe -action getinbox

# Get 10 newest messages
./msgraphgolangtestingtool.exe -action getinbox -count 10

# Get 20 newest messages with verbose output
./msgraphgolangtestingtool.exe -action getinbox -count 20 -verbose
```

---

## 8. Using Proxy

```powershell
# Use proxy for all network requests
./msgraphgolangtestingtool.exe -action sendmail \
    -to "user@example.com" \
    -proxy "http://proxy.company.com:8080"

# Proxy with environment variable
$env:MSGRAPHPROXY = "http://proxy.company.com:8080"
./msgraphgolangtestingtool.exe -action getevents
```

---

## 9. Verbose Output

```powershell
# Show detailed configuration, authentication, and API call information
./msgraphgolangtestingtool.exe -action sendmail -to "user@example.com" -verbose

# Verbose output shows:
# - Environment variables (MSGRAPH*)
# - Configuration after env vars + flags
# - Authentication method
# - Token information (truncated for security)
# - API endpoints being called
# - Response details
```

---

## 10. Complex Examples

### Send Formatted HTML Report with Multiple Attachments

```powershell
./msgraphgolangtestingtool.exe -action sendmail \
    -to "team-lead@example.com,manager@example.com" \
    -cc "team@example.com" \
    -subject "Q1 2026 Performance Report" \
    -bodyHTML "<h1>Q1 Performance Report</h1><p>Dear Team,</p><p>Please find attached the Q1 performance metrics and analysis.</p><ul><li>Revenue: Up 15%</li><li>Customer Satisfaction: 94%</li></ul><p>Best regards,<br>Analytics Team</p>" \
    -attachments "C:\Reports\Q1-Metrics.xlsx,C:\Reports\Q1-Analysis.pdf,C:\Charts\revenue-chart.png" \
    -verbose
```

### Create Weekly Meeting Series

```powershell
# Week 1
./msgraphgolangtestingtool.exe -action sendinvite \
    -invite-subject "Weekly Team Sync - Week 1" \
    -start "2026-01-06T15:00:00Z" \
    -end "2026-01-06T15:30:00Z"

# Week 2
./msgraphgolangtestingtool.exe -action sendinvite \
    -invite-subject "Weekly Team Sync - Week 2" \
    -start "2026-01-13T15:00:00Z" \
    -end "2026-01-13T15:30:00Z"
```

### Automated Monitoring Script

```powershell
# Log inbox and calendar to files
./msgraphgolangtestingtool.exe -action getinbox -count 50 | Out-File -Append "C:\Logs\inbox-monitor.log"
./msgraphgolangtestingtool.exe -action getevents -count 20 | Out-File -Append "C:\Logs\calendar-monitor.log"
```

---

## 11. Using Different Authentication Methods

```powershell
# Client Secret (via environment variable)
$env:MSGRAPHSECRET = "your-secret"
./msgraphgolangtestingtool.exe -action getevents

# PFX Certificate File
./msgraphgolangtestingtool.exe -action getevents \
    -pfx "C:\Certs\app-cert.pfx" \
    -pfxpass "MyP@ssw0rd"

# Windows Certificate Store (Thumbprint)
./msgraphgolangtestingtool.exe -action getevents \
    -thumbprint "CD817B3329802E692CF30D8DDF896FE811B048AB"
```

---

## 12. Mixed Environment Variables and Flags

```powershell
# Set defaults via environment variables
$env:MSGRAPHTENANTID = "tenant-id"
$env:MSGRAPHCLIENTID = "client-id"
$env:MSGRAPHSECRET = "secret"
$env:MSGRAPHMAILBOX = "user@example.com"
$env:MSGRAPHACTION = "sendmail"  # Default action

# Override specific parameters via flags (flags take precedence)
./msgraphgolangtestingtool.exe -to "override@example.com" -subject "Override Test"
```

---

## 13. Graceful Shutdown

```powershell
# Press Ctrl+C during long-running operations to gracefully shutdown
./msgraphgolangtestingtool.exe -action getinbox -count 100

# Output on interrupt:
#
# Received interrupt signal. Shutting down gracefully...
# (CSV logger closes properly, no data loss)
```

---

## 14. Error Handling Examples

```powershell
# Test with invalid action (will show error)
./msgraphgolangtestingtool.exe -action invalid

# Test missing required parameters (will show error)
./msgraphgolangtestingtool.exe -action sendmail
# Error: Missing required parameters (tenantid, clientid, mailbox).

# Test with verbose to debug issues
./msgraphgolangtestingtool.exe -action sendmail -to "user@example.com" -verbose
```

---

## 15. CSV Log Files

All operations automatically log to action-specific CSV files:

```powershell
# Location: %TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv

# Examples:
C:\Users\<Username>\AppData\Local\Temp\_msgraphgolangtestingtool_sendmail_2026-01-04.csv
C:\Users\<Username>\AppData\Local\Temp\_msgraphgolangtestingtool_getevents_2026-01-04.csv
C:\Users\<Username>\AppData\Local\Temp\_msgraphgolangtestingtool_sendinvite_2026-01-04.csv
C:\Users\<Username>\AppData\Local\Temp\_msgraphgolangtestingtool_getinbox_2026-01-04.csv
```

Each action type has its own schema:

- **getevents**: Timestamp, Action, Status, Mailbox, Event Subject, Event ID
- **sendmail**: Timestamp, Action, Status, Mailbox, To, CC, BCC, Subject, Body Type, Attachments
- **sendinvite**: Timestamp, Action, Status, Mailbox, Subject, Start Time, End Time, Event ID
- **getinbox**: Timestamp, Action, Status, Mailbox, Subject, From, To, Received DateTime

---

## 16. Environment Variables Reference

All flags can be set via environment variables with the `MSGRAPH` prefix:

| Flag | Environment Variable | Example |
|------|---------------------|---------|
| `-tenantid` | `MSGRAPHTENANTID` | `"tenant-id"` |
| `-clientid` | `MSGRAPHCLIENTID` | `"client-id"` |
| `-secret` | `MSGRAPHSECRET` | `"secret"` |
| `-pfx` | `MSGRAPHPFX` | `"C:\cert.pfx"` |
| `-pfxpass` | `MSGRAPHPFXPASS` | `"password"` |
| `-thumbprint` | `MSGRAPHTHUMBPRINT` | `"CD817..."` |
| `-mailbox` | `MSGRAPHMAILBOX` | `"user@example.com"` |
| `-to` | `MSGRAPHTO` | `"user1@example.com,user2@example.com"` |
| `-cc` | `MSGRAPHCC` | `"cc@example.com"` |
| `-bcc` | `MSGRAPHBCC` | `"bcc@example.com"` |
| `-subject` | `MSGRAPHSUBJECT` | `"Email Subject"` |
| `-body` | `MSGRAPHBODY` | `"Email body"` |
| `-bodyHTML` | `MSGRAPHBODYHTML` | `"<h1>HTML</h1>"` |
| `-attachments` | `MSGRAPHATTACHMENTS` | `"file1.pdf,file2.xlsx"` |
| `-invite-subject` | `MSGRAPHINVITESUBJECT` | `"Meeting"` |
| `-start` | `MSGRAPHSTART` | `"2026-01-15T14:00:00Z"` |
| `-end` | `MSGRAPHEND` | `"2026-01-15T15:00:00Z"` |
| `-action` | `MSGRAPHACTION` | `"sendmail"` |
| `-proxy` | `MSGRAPHPROXY` | `"http://proxy:8080"` |
| `-count` | `MSGRAPHCOUNT` | `"10"` |

---

## Tips and Best Practices

1. **Security**: Use environment variables for sensitive data (secrets, passwords) to avoid exposing them in command history
2. **Verbose Mode**: Use `-verbose` when troubleshooting authentication or API issues
3. **CSV Logs**: Check the CSV log files for historical records of all operations
4. **Graceful Shutdown**: Press Ctrl+C to interrupt long-running operations safely
5. **Flag Precedence**: Command-line flags override environment variables
6. **Comma Separation**: Lists (to, cc, bcc, attachments) use comma separation without spaces (or with spaces - they're trimmed)
7. **Time Format**: Calendar times use RFC3339 format (e.g., `2026-01-15T14:00:00Z`)
8. **HTML Emails**: Use `-bodyHTML` for rich formatting, optionally with `-body` for plain text fallback

---

## Quick Reference

```powershell
# Check version
./msgraphgolangtestingtool.exe -version

# Get help (shows all flags)
./msgraphgolangtestingtool.exe -h

# Test authentication
./msgraphgolangtestingtool.exe -action getevents -verbose

# Send quick test email
./msgraphgolangtestingtool.exe -action sendmail

# View recent inbox
./msgraphgolangtestingtool.exe -action getinbox -count 10
```

---

## 17. Retry Configuration (v1.16.0+)

Configure network resilience with automatic retry on transient failures:

```powershell
# Use custom retry settings
./msgraphgolangtestingtool.exe -action getevents \
    -maxretries 5 \
    -retrydelay 1000  # 1 second base delay

# Disable retries (set to 0)
./msgraphgolangtestingtool.exe -action sendmail \
    -to "user@example.com" \
    -maxretries 0

# Use aggressive retry for unreliable networks
./msgraphgolangtestingtool.exe -action getinbox \
    -maxretries 10 \
    -retrydelay 3000  # 3 second base delay

# Set via environment variables
$env:MSGRAPHMAXRETRIES = "5"
$env:MSGRAPHRETRYDELAY = "2500"
./msgraphgolangtestingtool.exe -action getevents
```

**Retry Behavior:**
- **Default**: 3 retries with 2-second base delay
- **Exponential backoff**: Delay pattern 2s → 4s → 8s → 16s → 30s (capped at 30 seconds)
- **Automatic retry on**:
  - HTTP 429 (Too Many Requests / Graph API throttling)
  - HTTP 503 (Service Unavailable)
  - HTTP 504 (Gateway Timeout)
  - Network timeouts and connection errors
- **Never retries**: Authentication failures, bad requests (400), not found (404)

**Example with verbose output:**
```powershell
./msgraphgolangtestingtool.exe -action getinbox \
    -maxretries 3 \
    -retrydelay 1000 \
    -verbose

# Output will show retry attempts:
# Retryable error encountered (attempt 1/3): timeout. Retrying in 1s...
# Retryable error encountered (attempt 2/3): timeout. Retrying in 2s...
# Operation succeeded after 2 retries
```

---

## 18. Environment Variables Reference (Updated v1.16.0)

All flags can be set via environment variables with the `MSGRAPH` prefix:

| Flag | Environment Variable | Example |
|------|---------------------|---------|
| `-tenantid` | `MSGRAPHTENANTID` | `"tenant-id"` |
| `-clientid` | `MSGRAPHCLIENTID` | `"client-id"` |
| `-secret` | `MSGRAPHSECRET` | `"secret"` |
| `-pfx` | `MSGRAPHPFX` | `"C:\\cert.pfx"` |
| `-pfxpass` | `MSGRAPHPFXPASS` | `"password"` |
| `-thumbprint` | `MSGRAPHTHUMBPRINT` | `"CD817..."` |
| `-mailbox` | `MSGRAPHMAILBOX` | `"user@example.com"` |
| `-to` | `MSGRAPHTO` | `"user1@example.com,user2@example.com"` |
| `-cc` | `MSGRAPHCC` | `"cc@example.com"` |
| `-bcc` | `MSGRAPHBCC` | `"bcc@example.com"` |
| `-subject` | `MSGRAPHSUBJECT` | `"Email Subject"` |
| `-body` | `MSGRAPHBODY` | `"Email body"` |
| `-bodyHTML` | `MSGRAPHBODYHTML` | `"<h1>HTML</h1>"` |
| `-attachments` | `MSGRAPHATTACHMENTS` | `"file1.pdf,file2.xlsx"` |
| `-invite-subject` | `MSGRAPHINVITESUBJECT` | `"Meeting"` |
| `-start` | `MSGRAPHSTART` | `"2026-01-15T14:00:00Z"` |
| `-end` | `MSGRAPHEND` | `"2026-01-15T15:00:00Z"` |
| `-action` | `MSGRAPHACTION` | `"sendmail"` |
| `-proxy` | `MSGRAPHPROXY` | `"http://proxy:8080"` |
| `-count` | `MSGRAPHCOUNT` | `"10"` |
| `-maxretries` | `MSGRAPHMAXRETRIES` | `"5"` |
| `-retrydelay` | `MSGRAPHRETRYDELAY` | `"2000"` |

---

NOTE: *Generated for msgraphgolangtestingtool v1.16.0*

                          ..ooOO END OOoo..


# Microsoft Graph EXO Mails/Calendar Golang Testing Tool

A portable, single-binary CLI tool for interacting with Microsoft Graph API to manage Exchange Online emails and calendar events.

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Purpose

This tool provides a lightweight, standalone executable for testing and managing Microsoft Graph API operations on Exchange Online mailboxes without requiring additional runtimes or dependencies. Cross-platform support for Windows, Linux, and macOS with multiple authentication methods and automatic CSV logging of all operations.

## Key Features

### Authentication Methods

- **Client Secret**: Standard Entra ID Application Registration secret
- **PFX Certificate**: Local certificate file with password protection
- **Windows Certificate Store**: Direct certificate access via thumbprint (no file management required)

### Operations

#### 1. Get Events (`-action getevents`)

Retrieves upcoming calendar events from a user's mailbox.

**Example:**

```powershell
.\msgraphgolangtestingtool.exe -tenantid"TENANT_ID" -clientid"CLIENT_ID" -secret "SECRET" -mailbox "user@example.com" -action getevents
```

#### 2. Send Mail (`-action sendmail`)

Sends emails with support for multiple recipients (To/CC/BCC) and custom subject/body.

**Example:**
```powershell
.\msgraphgolangtestingtool.exe -tenantid"TENANT_ID" -clientid"CLIENT_ID" -secret "SECRET" -mailbox "sender@example.com" -action sendmail -to "recipient@example.com" -subject "Test Email" -body "This is a test"
```

**Features:**

- Defaults to sending to self if no recipients specified
- Supports multiple To, CC, and BCC recipients (comma-separated)
- Text-based email bodies (HTML support coming soon)

#### 3. Send Invite (`-action sendinvite`)

Creates and sends calendar meeting invitations.

**Example:**

```powershell
.\msgraphgolangtestingtool.exe -tenantid"TENANT_ID" -clientid"CLIENT_ID" -secret "SECRET" -mailbox "user@example.com" -action sendinvite
```

#### 4. Get Inbox (`-action getinbox`)

Lists the newest 10 messages from a user's inbox with sender, recipients, subject, and received date.

**Example:**

```powershell
.\msgraphgolangtestingtool.exe -tenantid"TENANT_ID" -clientid"CLIENT_ID" -secret "SECRET" -mailbox "user@example.com" -action getinbox
```

#### 5. Check Availability (`-action getschedule`)

Checks recipient availability for the next working day at 12:00 UTC.

**Features:**
- Automatically calculates next working day (Monday-Friday, skips weekends)
- Checks 1-hour availability window (12:00-13:00 UTC)
- Returns Free/Busy status
- Single recipient only

**Required Permissions:** Exchange Online RBAC **Application Calendars.ReadWrite**

**Example:**

```powershell
.\msgraphgolangtestingtool.exe -tenantid"TENANT_ID" -clientid"CLIENT_ID" -secret "SECRET" -mailbox "organizer@example.com" -action getschedule -to "recipient@example.com"
```

**Output:**
```
Availability Check Results:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Organizer:     organizer@example.com
Recipient:     recipient@example.com
Check Date:    2026-01-06
Check Time:    12:00-13:00 UTC
Status:        Free
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Possible Status Values:**
- Free
- Tentative
- Busy
- Out of Office
- Working Elsewhere

### CSV Logging

All operations are automatically logged to action-specific CSV files in the Windows temp directory:
- **Location**: `%TEMP%\_msgraphgolangtestingtool_{action}_YYYY-MM-DD.csv`
- **Examples**: `sendmail_2026-01-03.csv`, `getevents_2026-01-03.csv`, `sendinvite_2026-01-03.csv`, `getinbox_2026-01-03.csv`, `getschedule_2026-01-05.csv`
- **Content**: Timestamps, action details, results, and status
- **Mode**: Append (multiple runs of the same action on the same day add to the same file)
- **Schema**: Each action type has its own consistent schema to prevent column conflicts

## Quick Start

### Prerequisites

1. **Go 1.25+** (for building from source)
2. **Entra ID Application Registration** with Exchange Online RBAC permissions:
   - **Application Mail.ReadWrite** (for sendmail, getinbox actions)
   - **Application Calendars.ReadWrite** (for getevents, sendinvite, getschedule actions)

   **Important**: These are Exchange Online RBAC permissions, NOT Entra ID API permissions.

   - **Documentation**: [Exchange Online Application RBAC](https://learn.microsoft.com/en-us/exchange/permissions-exo/application-rbac)
   - **Recommended Role**: Exchange Administrator (from PIM) to assign these permissions
   - **Note**: While Global Administrator can assign these permissions, Exchange Administrator is recommended following the Principle of Least Privilege

### Build

```powershell
# From project root
go build -C src -o msgraphgolangtestingtool.exe

# Or from src directory
cd src
go build -o ../msgraphgolangtestingtool.exe
```

See [BUILD.md](BUILD.md) for detailed build instructions.

### Usage
```powershell
.\msgraphgolangtestingtool.exe -tenantid "<TENANT_ID>" -clientid "<CLIENT_ID>" -secret "<SECRET>" -mailbox "<EMAIL>" -action <ACTION>
```

### Required Flags

| Flag | Description |
|------|-------------|
| `-tenantid` | Azure Tenant ID |
| `-clientid` | Application (Client) ID |
| `-mailbox` | Target user email address |
| **One of:** | |
| `-secret` | Client Secret |
| `-pfx` + `-pfxpass` | Path to PFX file and password |
| `-thumbprint` | Certificate thumbprint from Windows store |

### Optional Flags (for sendmail)

| Flag | Description |
|------|-------------|
| `-to` | Comma-separated To recipients |
| `-cc` | Comma-separated CC recipients |
| `-bcc` | Comma-separated BCC recipients |
| `-subject` | Email subject (default: "Automated Tool Notification") |
| `-body` | Email body text (default: "It's a test message, please ignore") |

### Optional Flags (for getevents and getinbox)

| Flag | Description |
|------|-------------|
| `-count` | Number of items to retrieve (default: 3) |

### Other Optional Flags

| Flag | Description |
|------|-------------|
| `-verbose` | Enable detailed diagnostic output |
| `-proxy` | HTTP/HTTPS proxy URL |

### Environment Variables

All flags can be set using environment variables with the `MSGRAPH` prefix (no underscores for easier PowerShell usage). Command-line flags take precedence over environment variables.

| Environment Variable | Equivalent Flag |
|---------------------|-----------------|
| `MSGRAPHTENANTID` | `-tenantid` |
| `MSGRAPHCLIENTID` | `-clientid` |
| `MSGRAPHSECRET` | `-secret` |
| `MSGRAPHPFX` | `-pfx` |
| `MSGRAPHPFXPASS` | `-pfxpass` |
| `MSGRAPHTHUMBPRINT` | `-thumbprint` |
| `MSGRAPHMAILBOX` | `-mailbox` |
| `MSGRAPHTO` | `-to` |
| `MSGRAPHCC` | `-cc` |
| `MSGRAPHBCC` | `-bcc` |
| `MSGRAPHSUBJECT` | `-subject` |
| `MSGRAPHBODY` | `-body` |
| `MSGRAPHINVITESUBJECT` | `-invite-subject` |
| `MSGRAPHSTART` | `-start` |
| `MSGRAPHEND` | `-end` |
| `MSGRAPHACTION` | `-action` |
| `MSGRAPHPROXY` | `-proxy` |
| `MSGRAPHCOUNT` | `-count` |

**Example:**

```powershell
# Set environment variables
$env:MSGRAPHTENANTID = "your-tenant-id"
$env:MSGRAPHCLIENTID = "your-client-id"
$env:MSGRAPHSECRET = "your-secret"
$env:MSGRAPHMAILBOX = "user@example.com"

# Run without repeating credentials

.\msgraphgolangtestingtool.exe                                               # Runs default action (getinbox)
.\msgraphgolangtestingtool.exe -action getevents                            # List calendar events
.\msgraphgolangtestingtool.exe -action sendmail -to "someone@example.com"   # Send email
```

### Verbose Mode

Enable detailed diagnostic output with the `-verbose` flag:

```powershell
.\msgraphgolangtestingtool.exe -verbose -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com"
```

**Verbose output includes:**
- **Environment Variables Section**: Lists all MSGRAPH* environment variables currently set
- **Final Configuration Section**: Shows resolved parameter values (after env vars + command-line flags)
- Authentication method and details
- JWT token information (expiration, validity period, truncated token)
- Graph API endpoints being called
- Request parameters and response details

**Use verbose mode for:**
- Verifying which environment variables are set and active
- Understanding parameter precedence (env vars vs command-line flags)
- Troubleshooting authentication issues
- Debugging API call failures
- Verifying configuration is correct
- Understanding token expiration

**Security note:** Verbose mode masks sensitive data (MSGRAPHSECRET and MSGRAPHPFXPASS show only first/last 4 characters, tokens are truncated).

### Proxy Support

Use the `-proxy` flag or `MSGRAPHPROXY` environment variable to route traffic through an HTTP/HTTPS proxy:

```powershell
# Using command-line flag
.\msgraphgolangtestingtool.exe -proxy "http://proxy.example.com:8080" -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com"

# Using environment variable
$env:MSGRAPHPROXY = "http://proxy.example.com:8080"
.\msgraphgolangtestingtool.exe -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com"

# Combine with other environment variables (runs default action: getinbox)
$env:MSGRAPHTENANTID = "xxx"
$env:MSGRAPHCLIENTID = "yyy"
$env:MSGRAPHSECRET = "zzz"
$env:MSGRAPHMAILBOX = "user@example.com"
$env:MSGRAPHPROXY = "http://proxy.example.com:8080"
.\msgraphgolangtestingtool.exe
```

## Authentication Examples

### Using Client Secret

```powershell
# Default action (getinbox)
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -secret "xxx" -mailbox "user@example.com"

# Specific action
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -secret "xxx" -mailbox "user@example.com" -action getevents
```

### Using PFX Certificate

```powershell
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -pfx ".\cert.pfx" -pfxpass "password" -mailbox "user@example.com" -action sendmail -to "recipient@example.com"
```

### Using Windows Certificate Store

```powershell
# Default action (getinbox)
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -thumbprint "CD817B3329802E692CF30D8DDF896FE811B048AB" -mailbox "user@example.com"
```

## Certificate Setup

For testing purposes, use the included PowerShell script to generate a self-signed certificate:

```powershell
.\selfsignedcert.ps1
```

This creates:

- A 2048-bit RSA certificate with SHA256 hash
- PFX file (private key) for authentication
- CER file (public key) to upload to Entra ID Application Registration

**For production**: Use CA-signed certificates instead of self-signed certificates.

## Platform Requirements

**Cross-Platform** - The tool can be built for Windows, Linux, and macOS. However, the `-thumbprint` authentication method is **Windows-specific** as it utilizes the native Windows Certificate Store (via CryptoAPI).

## Documentation

- **[BUILD.md](BUILD.md)**: Detailed build instructions
- **[RELEASE.md](RELEASE.md)**: Interactive release script documentation
- **[CLAUDE.md](CLAUDE.md)**: Architecture and code structure for AI assistants
- **[GEMINI.md](GEMINI.md)**: Comprehensive usage guide
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Error troubleshooting and solutions
- **[SECURITY_PRACTICES.md](SECURITY_PRACTICES.md)**: Security best practices and guidelines

## Release Management

Use the interactive release script to create new releases:

```powershell
.\release.ps1
```

The script handles version updates, changelog creation, git operations, and triggers GitHub Actions to build binaries. See **[RELEASE.md](RELEASE.md)** for complete documentation.

## Output

All operations display results on screen and simultaneously log to action-specific CSV files:

```powershell
C:\Users\<Username>\AppData\Local\Temp\_msgraphgolangtestingtool_{action}_YYYY-MM-DD.csv
```

Examples:
- `_msgraphgolangtestingtool_sendmail_2026-01-03.csv`
- `_msgraphgolangtestingtool_getevents_2026-01-03.csv`
- `_msgraphgolangtestingtool_sendinvite_2026-01-03.csv`
- `_msgraphgolangtestingtool_getinbox_2026-01-03.csv`

The CSV file path is displayed at the start of each operation.

## License

This tool is provided as-is for testing and automation purposes.

## Support

For issues, questions, or feature requests, please refer to the documentation files or contact your system administrator.

# Microsoft Graph GoLang Testing Tool

A portable, single-binary CLI tool for interacting with Microsoft Graph API to manage Exchange Online emails and calendar events.

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Purpose

This tool provides a lightweight, standalone executable for testing and managing Microsoft Graph API operations on Exchange Online mailboxes without requiring additional runtimes or dependencies. Designed for Windows environments, it supports multiple authentication methods and automatically logs all operations to CSV files.

## Key Features

### Authentication Methods

- **Client Secret**: Standard Azure AD App Registration secret
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

### CSV Logging

All operations are automatically logged to action-specific CSV files in the Windows temp directory:
- **Location**: `%TEMP%\_msgraphgolangtestingtool_{action}_YYYY-MM-DD.csv`
- **Examples**: `sendmail_2026-01-03.csv`, `getevents_2026-01-03.csv`, `sendinvite_2026-01-03.csv`, `getinbox_2026-01-03.csv`
- **Content**: Timestamps, action details, results, and status
- **Mode**: Append (multiple runs of the same action on the same day add to the same file)
- **Schema**: Each action type has its own consistent schema to prevent column conflicts

## Quick Start

### Prerequisites

1. **Go 1.25+** (for building from source)
2. **Azure AD App Registration** with required permissions:
   - `Mail.Send`
   - `Mail.Read`
   - `Calendars.ReadWrite`
3. **Admin Consent** granted for the application

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

.\msgraphgolangtestingtool.exe -action getevents
.\msgraphgolangtestingtool.exe -action getinbox
.\msgraphgolangtestingtool.exe -action sendmail -to "someone@example.com"
```

### Verbose Mode

Enable detailed diagnostic output with the `-verbose` flag:

```powershell
.\msgraphgolangtestingtool.exe -verbose -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com" -action getevents
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
.\msgraphgolangtestingtool.exe -proxy "http://proxy.example.com:8080" -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com" -action getevents

# Using environment variable
$env:MSGRAPHPROXY = "http://proxy.example.com:8080"
.\msgraphgolangtestingtool.exe -tenantid "xxx" -clientid "yyy" -secret "zzz" -mailbox "user@example.com" -action getevents

# Combine with other environment variables
$env:MSGRAPHTENANTID = "xxx"
$env:MSGRAPHCLIENTID = "yyy"
$env:MSGRAPHSECRET = "zzz"
$env:MSGRAPHMAILBOX = "user@example.com"
$env:MSGRAPHPROXY = "http://proxy.example.com:8080"
.\msgraphgolangtestingtool.exe -action getevents
```

## Authentication Examples

### Using Client Secret

```powershell
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -secret "xxx" -mailbox "user@example.com" -action getevents
```

### Using PFX Certificate

```powershell
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -pfx ".\cert.pfx" -pfxpass "password" -mailbox "user@example.com" -action sendmail -to "recipient@example.com"
```

### Using Windows Certificate Store

```powershell
.\msgraphgolangtestingtool.exe -tenantid"xxx" -clientid"xxx" -thumbprint "CD817B3329802E692CF30D8DDF896FE811B048AB" -mailbox "user@example.com" -action getinbox
```

## Certificate Setup

For testing purposes, use the included PowerShell script to generate a self-signed certificate:

```powershell
.\selfsignedcert.ps1
```

This creates:

- A 2048-bit RSA certificate with SHA256 hash
- PFX file (private key) for authentication
- CER file (public key) to upload to Azure AD App Registration

**For production**: Use CA-signed certificates instead of self-signed certificates.

## Platform Requirements

**Cross-Platform** - The tool can be built for Windows, Linux, and macOS. However, the `-thumbprint` authentication method is **Windows-specific** as it utilizes the native Windows Certificate Store (via CryptoAPI).

## Documentation

- **[BUILD.md](BUILD.md)**: Detailed build instructions
- **[CLAUDE.md](CLAUDE.md)**: Architecture and code structure for AI assistants
- **[GEMINI.md](GEMINI.md)**: Comprehensive usage guide
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Error troubleshooting and solutions
- **[SECURITY_PRACTICES.md](SECURITY_PRACTICES.md)**: Security best practices and guidelines

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

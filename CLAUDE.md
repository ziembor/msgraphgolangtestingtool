# Portable Microsoft Graph GoLang CLI Tool

## Overview

This is a lightweight, portable command-line interface (CLI) tool written in **Go (Golang)** designed for Windows (but cross-compatible). It allows interactions with the **Microsoft Graph API** to manage emails and calendar events on Exchange Online (EXO) mailboxes.

The tool is designed for **minimal external dependencies** — it compiles into a single static binary (`.exe`) that does not require installing runtimes or libraries on the target machine.

## Features

* **Authentication Modes:**
  * **Client Secret:** Standard App Registration secret.
  * **Certificate (PFX):** Secure, password-protected PFX file support.
  * **Windows Certificate Store:** Use certificates directly from the Current User's Personal ("My") store via Thumbprint (requires no physical file management).
 **Graph Operations:**
  * **Send Mail:** Send Text emails via the Graph API with support for:
    * Custom Subject and Body.
    * Multiple To, CC, and BCC recipients.
    * Defaults to sending to self if no recipient is provided.
  * **List Events:** Retrieve upcoming calendar events for a specific user.
  * **Create Invite:** Create and send calendar meeting invitations.
  * **List Inbox:** Retrieve the newest 10 messages from inbox with sender, recipients, subject, and received date.
 **Network:**
  * **Proxy Support:** Route traffic through HTTP/HTTPS proxies via flag or environment variable.
* **CSV Logging:**
  * All operations are automatically logged to `%TEMP%\_msgraphgolangtestingtool_{date}.csv`
  * Includes timestamps and action-specific details
  * Output shown on screen and written to CSV simultaneously

## Versioning

The project follows Semantic Versioning, but the **major version is locked at 1** and cannot be upgraded. All breaking changes or major features will be released as minor version increments within the 1.x.y branch.

**Version Management:**
- Current version is stored in the `VERSION` file at the project root
- The version constant in `src/msgraphgolangtestingtool.go` must match the VERSION file
- When updating the version, update BOTH files to maintain consistency
- **IMPORTANT**: Future AI assistants should always read and update the VERSION file when making version changes

## Prerequisites

* **Microsoft Entra ID (Azure AD):** App Registration.
* **Permissions (Application Type):**
  * `Mail.Send`
  * `Mail.Read`
  * `Calendars.ReadWrite`
  * *Grant Admin Consent* must be applied.

## Usage

### Command Line Flags

| Flag | Description | Required |
| :--- | :--- | :--- |
| `-tenantid` | The Azure Directory (Tenant) ID. | **Yes** |
| `-clientid` | The Application (Client) ID. | **Yes** |
| `-mailbox` | The target user email address to act upon (sender). | **Yes** |
| `-action` | Operation to perform: `getevents`, `sendmail`, `sendinvite`, or `getinbox`. | No (default: `getevents`) |
| **Authentication** | | |
| `-secret` | The Client Secret. | Use one Auth method |
| `-pfx` | Path to a local `.pfx` certificate file. | Use one Auth method |
| `-pfxpass` | Password for the `.pfx` file. | If PFX is protected |
| `-thumbprint` | SHA1 Thumbprint of a cert in `CurrentUser\My` store. | Use one Auth method |
| **Mail Options** | (Used with `-action sendmail`) | |
| `-to` | Comma-separated list of TO recipients. | No (defaults to self) |
| `-cc` | Comma-separated list of CC recipients. | No |
| `-bcc` | Comma-separated list of BCC recipients. | No |
| `-subject` | Email subject line. | No (default: "Automated...") |
| `-body` | Email body content (Text). | No (default: "It's test...") |
| **Calendar Options** | (Used with `-action sendinvite`) | |
| `-invite-subject` | Subject of the calendar invite. | No (default: "System Sync") |
| `-start` | Start time (RFC3339, e.g., 2026-01-15T14:00:00Z). | No (default: Now) |
| `-end` | End time (RFC3339). | No (default: 1h after start) |
| **Network** | | |
| `-proxy` | HTTP/HTTPS proxy URL (e.g., http://proxy.example.com:8080). | No |
| **Other** | | |
| `-version` | Show version information. | No |
| `-verbose` | Enable verbose output (shows configuration, tokens, API details). | No |

### Verbose Mode

Enable verbose output with the `-verbose` flag to see detailed diagnostic information:

```powershell
.\msgraphgolangtestingtool.exe -verbose -tenantid "..." -clientid "..." -secret "..." -mailbox "..." -action getevents
```

Verbose mode displays:
- **Environment Variables**: All MSGRAPH* environment variables currently set (with sensitive values masked)
- **Final Configuration**: All parameters and their final values (after environment variable processing and command-line flags)
- **Authentication Details**: Method used, certificate info, masked secrets
- **Token Information**: Expiration time, validity period, truncated token for verification
- **API Call Details**: Endpoints being called, request parameters
- **Response Information**: Number of items retrieved, operation results

This is useful for:
- Debugging configuration issues
- Verifying which environment variables are set and being used
- Troubleshooting authentication issues
- Understanding the tool's behavior and parameter precedence

### Environment Variables

All flags can be set via environment variables (Command Line flags take precedence).
Prefix: `MSGRAPH` (e.g., `MSGRAPHTENANTID`, `MSGRAPHCLIENTID`, `MSGRAPHPROXY`).

### Examples

#### 1. List Calendar Events (Client Secret)

```powershell
.\msgraphgolangtestingtool.exe -tenantid "1111-2222-3333" `
                 -clientid "aaaa-bbbb-cccc" `
                 -secret "MySecretValue" `
                 -mailbox "user@example.com" `
                 -action getevents
```

#### 2. Send Email using Local Certificate File (PFX)

```powershell
.\msgraphgolangtestingtool.exe -tenantid "1111-2222-3333" `
                 -clientid "aaaa-bbbb-cccc" `
                 -mailbox "sender@example.com" `
                 -pfx ".\cert.pfx" -pfxpass "Pass123" `
                 -action sendmail `
                 -to "recipient@example.com" `
                 -subject "Weekly Report" `
                 -body "Here is the update."
```

#### 3. Send Email using Windows Certificate Store

No need to manage PFX files. The tool extracts the public/private key pair directly from the user's certificate store into memory using the Thumbprint and native Windows APIs.

```powershell
.\msgraphgolangtestingtool.exe -tenantid "1111-2222-3333" `
                 -clientid "aaaa-bbbb-cccc" `
                 -mailbox "sender@example.com" `
                 -thumbprint "CD817B3329802E692CF30D8DDF896FE811B048AB" `
                 -action sendmail `
                 -to "boss@example.com" -cc "team@example.com"
```

#### 4. List Newest 10 Inbox Messages

```powershell
.\msgraphgolangtestingtool.exe -tenantid "1111-2222-3333" `
                 -clientid "aaaa-bbbb-cccc" `
                 -secret "MySecretValue" `
                 -mailbox "user@example.com" `
                 -action getinbox
```

#### 5. Use Proxy (Flag or Env Var)

```powershell
# Using flag
.\msgraphgolangtestingtool.exe -proxy "http://10.0.0.1:8080" ...

# Using Environment Variable
$env:MSGRAPHPROXY = "http://10.0.0.1:8080"
.\msgraphgolangtestingtool.exe ...
```

## Project Overview

This is a portable, single-binary Go CLI tool for interacting with Microsoft Graph API to manage Exchange Online (EXO) emails and calendar events. The tool compiles to a standalone executable with minimal external dependencies.

**Platform**: Cross-platform (Windows, Linux, macOS), but `-thumbprint` auth is Windows-only.
**Module name**: `msgraphgolangtestingtool` (defined in go.mod)
**Go version**: 1.25+
**Current Version**: See `VERSION` file in project root

**Versioning Policy**: The major version of this project is locked at 1 and must not be upgraded. All changes, including breaking ones, must be released within the 1.x.y version range. The version is maintained in two places: the `VERSION` file and `src/msgraphgolangtestingtool.go` (const version).

### CSV Logging

All operations are automatically logged to a CSV file in the Windows temp directory (`%TEMP%\_msgraphgolangtestingtool_{date}.csv`). The log includes timestamps and action-specific data. Output is shown on screen and written to the CSV file simultaneously.

## Build and Run Commands

```powershell
# Build the executable from project root
cd src
go build -o ../msgraphgolangtestingtool.exe

# Or build from project root using -C flag
go build -C src -o msgraphgolangtestingtool.exe

# Run with Go (development) from src directory
cd src
go run . [flags]

# Example: List calendar events using client secret
.\msgraphgolangtestingtool.exe -tenantid "YOUR_TENANT_ID" -clientid "YOUR_CLIENT_ID" -secret "YOUR_SECRET" -mailbox "user@example.com" -action getevents

# Example: Send email using PFX certificate
.\msgraphgolangtestingtool.exe -tenantid"YOUR_TENANT_ID" -clientid "YOUR_CLIENT_ID" -pfx ".\cert.pfx" -pfxpass "password" -mailbox "sender@example.com" -action sendmail -to "recipient@example.com" -subject "Test" -body "Test message"

# Example: Send email using Windows Certificate Store (thumbprint)
.\msgraphgolangtestingtool.exe -tenantid"YOUR_TENANT_ID" -clientid "YOUR_CLIENT_ID" -thumbprint "CERT_THUMBPRINT" -mailbox "sender@example.com" -action sendmail

# Example: List newest 10 inbox messages
.\msgraphgolangtestingtool.exe -tenantid"YOUR_TENANT_ID" -clientid "YOUR_CLIENT_ID" -secret "YOUR_SECRET" -mailbox "user@example.com" -action getinbox

# Example: Using environment variables (PowerShell)
$env:MSGRAPHTENANTID = "YOUR_TENANT_ID"
$env:MSGRAPHCLIENTID = "YOUR_CLIENT_ID"
$env:MSGRAPHSECRET = "YOUR_SECRET"
$env:MSGRAPHMAILBOX = "user@example.com"
.\msgraphgolangtestingtool.exe -action getevents

# Example: Mix of environment variables and command-line flags (flags take precedence)
$env:MSGRAPHTENANTID = "YOUR_TENANT_ID"
$env:MSGRAPHCLIENTID = "YOUR_CLIENT_ID"
$env:MSGRAPHSECRET = "YOUR_SECRET"
.\msgraphgolangtestingtool.exe -mailbox "user@example.com" -action sendmail -to "recipient@example.com"
```

## Architecture

### Multi-File Design

The application is structured into multiple files under the `src/` directory to support platform-specific authentication:

* `src/msgraphgolangtestingtool.go`: Main logic and Graph API interaction.
* `src/cert_windows.go`: Native Windows CryptoAPI implementation for certificate store access.
* `src/cert_stub.go`: Stub implementation for non-Windows platforms.

This structure allows the tool to be cross-compiled while maintaining native integration on Windows.

### Environment Variable Support

The tool supports configuration via environment variables with the `MSGRAPH` prefix (no underscores for easier PowerShell usage) (src/msgraphgolangtestingtool.go:35-81):

* All command-line flags have corresponding environment variables
* Command-line flags **take precedence** over environment variables
* Environment variables are only used if the corresponding flag is not provided
* Mapping: `-tenantid` → `MSGRAPHTENANTID`, `-clientid` → `MSGRAPHCLIENTID`, `-mailbox` → `MSGRAPHMAILBOX`, `-proxy` → `MSGRAPHPROXY`, etc.
* Useful for CI/CD pipelines, containerized environments, and reducing repetitive typing
* All 17 configuration parameters support environment variables (see CHANGELOG for complete list)

### Proxy Support

The tool supports HTTP/HTTPS proxy configuration (src/msgraphgolangtestingtool.go:156-162):

* Configure via `-proxy` flag or `MSGRAPHPROXY` environment variable
* Supports standard proxy URL format: `http://proxy.example.com:8080`
* Automatically sets `HTTP_PROXY` and `HTTPS_PROXY` system environment variables
* All Microsoft Graph API requests will route through the specified proxy
* Useful for corporate networks, testing environments, and traffic monitoring

### Authentication Flow

The application supports three mutually exclusive authentication methods (src/msgraphgolangtestingtool.go:107-132):

1. **Client Secret** (`-secret`): Standard App Registration secret authentication
2. **PFX File** (`-pfx` + `-pfxpass`): Certificate-based authentication using a local .pfx file
3. **Windows Certificate Store** (`-thumbprint`): Extracts certificate from `CurrentUser\My` store via native Windows CryptoAPI (crypt32.dll), exports it to a memory buffer as PFX, then authenticates.

The Windows Certificate Store authentication (`src/cert_windows.go`) uses native Windows syscalls to:

* Open the `CurrentUser\My` certificate store.
* Find the certificate by its SHA1 thumbprint.
* Export the certificate and its private key directly to a memory buffer (PFX format) using a temporary random password.
* Perform all operations in memory without creating temporary files on disk.

### Action Dispatch

The main function routes to four action handlers based on the `-action` flag (src/msgraphgolangtestingtool.go:71-89):

* `getevents`: Lists upcoming calendar events for the specified mailbox
* `sendmail`: Sends an email with support for To/CC/BCC recipients, custom subject/body
* `sendinvite`: Creates a calendar meeting invitation with customizable subject, start time, and end time (defaults: now and +1 hour)
* `getinbox`: Lists the newest 10 messages from inbox (shows sender, recipient, received date, subject)

### Graph API Integration

Uses Microsoft Graph SDK for Go (`github.com/microsoftgraph/msgraph-sdk-go`) with application permissions requiring:

* `Mail.Send`
* `Mail.Read`
* `Calendars.ReadWrite`

Admin consent must be granted in Azure AD.

### Recipient Handling

Email recipients (src/msgraphgolangtestingtool.go:75-82):

* Parses comma-separated lists for To/CC/BCC
* Defaults to sending to self (the mailbox owner) if no recipients specified
* Helper function `createRecipients` (src/msgraphgolangtestingtool.go:256-268) converts email strings to Graph API Recipient objects

### Email Content

Currently supports TEXT-only email bodies (src/msgraphgolangtestingtool.go:245). HTML support will be added in future updates.

### CSV Logging System

The tool implements automatic CSV logging (src/msgraphgolangtestingtool.go:357-425):

* Creates a daily CSV file in `%TEMP%` directory with format `_msgraphgolangtestingtool_YYYY-MM-DD.csv`
* File is opened in append mode (multiple runs on the same day append to the same file)
* Headers are written only when creating a new file
* Each action has a custom CSV schema (Status is always the 3rd column for consistency):
  * `getevents`: Timestamp, Action, Status, Mailbox, Event Subject, Event ID
  * `sendmail`: Timestamp, Action, Status, Mailbox, To, CC, BCC, Subject
  * `sendinvite`: Timestamp, Action, Status, Mailbox, Subject, Start Time, End Time, Event ID
  * `getinbox`: Timestamp, Action, Status, Mailbox, Subject, From, To, Received DateTime
* All outputs are flushed immediately to ensure data is written even if the program terminates unexpectedly

## Key Dependencies

* `github.com/Azure/azure-sdk-for-go/sdk/azidentity`: Azure authentication
* `github.com/microsoftgraph/msgraph-sdk-go`: Microsoft Graph SDK
* `golang.org/x/crypto/pkcs12`: PFX certificate decoding

## Certificate Management

The `selfsignedcert.ps1` PowerShell script generates a self-signed certificate for **testing and development**:

* Creates a 2048-bit RSA certificate with SHA256 hash
* Stores in CurrentUser\My certificate store
* Exports both PFX (private key) and CER (public key) files
* Certificate valid for 2 years
* Includes comprehensive error handling and validation
* Provides clear instructions for Azure AD configuration

For production use, upload the public certificate (.cer) to your Azure AD App Registration and use a CA-signed certificate.

## Required Flags

All executions require:

* `-tenantid`: Azure Tenant ID
* `-clientid`: Application (Client) ID
* `-mailbox`: Target user email address

Plus one authentication method (`-secret`, `-pfx`, or `-thumbprint`).

..ooOO End OOoo..

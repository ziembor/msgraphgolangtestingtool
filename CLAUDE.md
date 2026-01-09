# Microsoft Graph & SMTP Testing Tools

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Overview

This repository contains two complementary, lightweight, portable command-line interface (CLI) tools written in **Go (Golang)** with cross-platform support for **Windows, Linux, and macOS**:

- **msgraphgolangtestingtool**: Interacts with the **Microsoft Graph API** to manage emails and calendar events on Exchange Online (EXO) mailboxes.
- **smtptool**: Tests SMTP connectivity with comprehensive TLS diagnostics for on-premises Exchange servers and generic SMTP servers.

Both tools are designed for **minimal external dependencies** — they compile into single static binaries that do not require installing runtimes or libraries on the target machine.

## Features

### Microsoft Graph Tool (msgraphgolangtestingtool)

* **Authentication Modes:**
  * **Client Secret:** Standard App Registration secret.
  * **Certificate (PFX):** Secure, password-protected PFX file support.
  * **Windows Certificate Store:** Use certificates directly from the Current User's Personal ("My") store via Thumbprint (requires no physical file management).
* **Graph Operations:**
  * **Send Mail:** Send Text emails via the Graph API with support for:
    * Custom Subject and Body.
    * Multiple To, CC, and BCC recipients.
    * Defaults to sending to self if no recipient is provided.
  * **List Events:** Retrieve upcoming calendar events for a specific user.
  * **Create Invite:** Create and send calendar meeting invitations.
  * **List Inbox:** Retrieve the newest 10 messages from inbox with sender, recipients, subject, and received date.
  * **Check Availability:** Check recipient availability for next working day at 12:00 UTC (returns Free/Busy status).
  * **Export Inbox:** Export inbox messages to individual JSON files in date-stamped directories (`%TEMP%\export\{date}`).
  * **Search and Export:** Find and export specific email by Internet Message ID to JSON file.
* **Network:**
  * **Proxy Support:** Route traffic through HTTP/HTTPS proxies via flag or environment variable.

### SMTP Tool (smtptool)

* **SMTP Operations:**
  * **Test Connect:** Basic SMTP connectivity with capability detection and Exchange server detection.
  * **Test STARTTLS:** Comprehensive TLS diagnostics including:
    * SSL/TLS handshake analysis
    * Certificate chain validation
    * Cipher suite and strength assessment
    * Protocol version detection
    * Hostname verification
    * Expiry warnings
  * **Test Auth:** SMTP authentication testing with mechanism auto-selection (PLAIN, LOGIN, CRAM-MD5).
  * **Send Mail:** End-to-end email sending with STARTTLS and authentication support.
* **Diagnostics:**
  * Exchange version detection (2003-2019)
  * TLS warnings for deprecated protocols and weak ciphers
  * Certificate validation with detailed error reporting

### Both Tools

* **CSV Logging:**
  * All operations are automatically logged to `%TEMP%\_{toolname}_{action}_{date}.csv`
  * Each action type creates its own log file
  * Includes timestamps and action-specific details
  * Output shown on screen and written to CSV simultaneously

## Versioning

The project follows Semantic Versioning (x.y.z):
- **Major (x):** Breaking changes, major architectural shifts, new tools/executables
- **Minor (y):** New features, significant enhancements
- **Patch (z):** Bug fixes, documentation updates

**Version Management:**
- Current version is stored in `src/VERSION`
- The version is automatically embedded into the Go binary at compile time using `//go:embed VERSION`
- **Only update `src/VERSION`**
- **Refer to `RELEASE.md` for the full policy.**

## Prerequisites

* **Microsoft Entra ID (Azure AD):** App Registration.
* **Exchange Online RBAC Permissions:**
  * **Application Mail.ReadWrite** (for sendmail, getinbox, exportinbox, searchandexport actions)
  * **Application Calendars.ReadWrite** (for getevents, sendinvite, getschedule actions)
  * **Important**: These are Exchange Online RBAC permissions, NOT Entra ID API permissions.

## Usage

See **[EXAMPLES.md](EXAMPLES.md)** for detailed usage scenarios.

## Project Overview

**Platform**: Cross-platform (Windows, Linux, macOS), but `-thumbprint` auth is Windows-only.
**Module name**: `msgraphgolangtestingtool`
**Go version**: 1.25+

### Project Structure

The repository uses a modular structure with shared internal packages:

```
msgraphgolangtestingtool/
├── cmd/
│   ├── msgraphtool/              # Microsoft Graph tool source
│   │   ├── main.go
│   │   ├── config.go
│   │   ├── handlers.go
│   │   ├── auth.go
│   │   └── completions.go
│   └── smtptool/                 # SMTP tool source
│       ├── main.go
│       ├── config.go
│       ├── handlers.go
│       ├── smtp_client.go
│       ├── smtp_connect.go
│       ├── smtp_starttls.go
│       ├── smtp_auth.go
│       ├── smtp_sendmail.go
│       └── completions.go
├── internal/
│   ├── common/                   # Shared packages (70-80% code reuse)
│   │   ├── logger/               # CSV and structured logging
│   │   ├── retry/                # Retry with exponential backoff
│   │   ├── version/              # Version embedding
│   │   └── validation/           # Input validators
│   ├── msgraph/                  # Graph-specific code
│   └── smtp/                     # SMTP-specific code
│       ├── protocol/             # SMTP command builders and response parsing
│       ├── tls/                  # TLS handshake and certificate analysis
│       └── exchange/             # Exchange detection
├── src/
│   └── VERSION                   # Version file (embedded at compile time)
├── build-all.ps1                 # Build script for both tools
├── run-integration-tests.ps1     # Release automation script
├── selfsignedcert.ps1            # Certificate generation script
├── Changelog/                    # Version changelogs
├── CLAUDE.md                     # Project documentation (this file)
├── BUILD.md                      # Build instructions
├── SMTP_TOOL_README.md           # SMTP tool documentation
└── README.md                     # Public documentation
```

**Key Points:**
- Root `go.mod` and `go.sum` in project root
- VERSION file remains in `src/` for backward compatibility
- Build both tools: `.\build-all.ps1`
- Build individually: `go build -C cmd/msgraphtool` or `go build -C cmd/smtptool`

### CSV Logging

All operations are automatically logged to action-specific CSV files in the temp directory:
- Microsoft Graph Tool: `%TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv`
- SMTP Tool: `%TEMP%\_smtptool_{action}_{date}.csv`

## Build and Run Commands

```powershell
# Build both tools at once (recommended)
.\build-all.ps1

# Build Microsoft Graph tool individually
go build -C cmd/msgraphtool -o msgraphgolangtestingtool.exe

# Build SMTP tool individually
go build -C cmd/smtptool -o smtptool.exe

# Run with Go (development)
cd cmd/msgraphtool
go run . [flags]

cd cmd/smtptool
go run . [flags]
```

See **[BUILD.md](BUILD.md)** for more details.

## Release Process

**IMPORTANT: Use the interactive release script for all releases.**

```powershell
# From project root
.\run-integration-tests.ps1
```

See **[RELEASE.md](RELEASE.md)** for the complete release guide.

## Documentation Reference

- **[RELEASE.md](RELEASE.md)** - Release process and versioning policy.
- **[BUILD.md](BUILD.md)** - Build instructions for both tools.
- **[README.md](README.md)** - User-facing documentation.
- **[SMTP_TOOL_README.md](SMTP_TOOL_README.md)** - Complete SMTP tool documentation.
- **[EXAMPLES.md](EXAMPLES.md)** - Microsoft Graph tool usage examples.
- **[SECURITY.md](SECURITY.md)** - Security policy and best practices.

                          ..ooOO END OOoo..



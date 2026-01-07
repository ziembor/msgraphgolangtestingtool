# Portable Microsoft Graph EXO Mails/Calendar Golang CLI Tool

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Overview

This is a lightweight, portable command-line interface (CLI) tool written in **Go (Golang)** with cross-platform support for **Windows, Linux, and macOS**. It allows interactions with the **Microsoft Graph API** to manage emails and calendar events on Exchange Online (EXO) mailboxes.

The tool is designed for **minimal external dependencies** — it compiles into a single static binary that does not require installing runtimes or libraries on the target machine.

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
  * **Check Availability:** Check recipient availability for next working day at 12:00 UTC (returns Free/Busy status).
 **Network:**
  * **Proxy Support:** Route traffic through HTTP/HTTPS proxies via flag or environment variable.
* **CSV Logging:**
  * All operations are automatically logged to `%TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv`
  * Each action type creates its own log file (e.g., `sendmail_2026-01-03.csv`, `getevents_2026-01-03.csv`)
  * Includes timestamps and action-specific details
  * Output shown on screen and written to CSV simultaneously

## Versioning

The project follows Semantic Versioning, but the **major version is locked at 1**. All breaking changes or major features will be released as minor version increments within the 1.x.y branch.

**Version Management:**
- Current version is stored in `src/VERSION`
- The version is automatically embedded into the Go binary at compile time using `//go:embed VERSION`
- **Only update `src/VERSION`**
- **Refer to `RELEASE.md` for the full policy.**

## Prerequisites

* **Microsoft Entra ID (Azure AD):** App Registration.
* **Exchange Online RBAC Permissions:**
  * **Application Mail.ReadWrite** (for sendmail, getinbox actions)
  * **Application Calendars.ReadWrite** (for getevents, sendinvite, getschedule actions)
  * **Important**: These are Exchange Online RBAC permissions, NOT Entra ID API permissions.

## Usage

See **[EXAMPLES.md](EXAMPLES.md)** for detailed usage scenarios.

## Project Overview

**Platform**: Cross-platform (Windows, Linux, macOS), but `-thumbprint` auth is Windows-only.
**Module name**: `msgraphgolangtestingtool`
**Go version**: 1.25+

### Project Structure

**IMPORTANT:** All Go and PowerShell source code must be kept in the `src/` directory.

```
msgraphgolangtestingtool/
├── src/                          # All source code
│   ├── *.go                      # Go source files
│   ├── go.mod                    # Go module definition
│   ├── go.sum                    # Go dependencies
│   ├── VERSION                   # Version file (embedded at compile time)
│   └── *.ps1                     # PowerShell scripts (if any)
├── release.ps1                   # Release automation script (project root)
├── selfsignedcert.ps1            # Certificate generation script (project root)
├── Changelog/                    # Version changelogs
├── CLAUDE.md                     # Project documentation (this file)
└── README.md                     # Public documentation
```

**Key Points:**
- `go.mod` and `go.sum` are in `src/`
- The `VERSION` file is in `src/`
- Build from project root: `go build -C src -o msgraphgolangtestingtool.exe`

### CSV Logging

All operations are automatically logged to action-specific CSV files in the Windows temp directory (`%TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv`).

## Build and Run Commands

```powershell
# Build the executable from project root
go build -C src -o msgraphgolangtestingtool.exe

# Run with Go (development) from src directory
cd src
go run . [flags]
```

See **[BUILD.md](BUILD.md)** for more details.

## Release Process

**IMPORTANT: Use the interactive release script for all releases.**

```powershell
# From project root
.\release.ps1
```

See **[RELEASE.md](RELEASE.md)** for the complete release guide.

## Documentation Reference

- **[RELEASE.md](RELEASE.md)** - Release process and versioning policy.
- **[BUILD.md](BUILD.md)** - Build instructions.
- **[README.md](README.md)** - User-facing documentation.
- **[EXAMPLES.md](EXAMPLES.md)** - Usage examples.
- **[SECURITY.md](SECURITY.md)** - Security policy and best practices.
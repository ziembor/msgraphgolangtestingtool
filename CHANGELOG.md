# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.12.0] - 2026-01-03

### Added
- Native Windows CryptoAPI (`crypt32.dll`) integration for certificate store authentication
- Support for cross-compilation (Windows-specific code isolated via build tags)

### Changed
- Removed PowerShell dependency for `-thumbprint` authentication
- Certificate export from store now happens entirely in memory (no temporary PFX files on disk)
- Project structure changed from single-file to multi-file (`cert_windows.go`, `cert_stub.go`)

## [1.11.0] - 2026-01-03

### Changed
- **BREAKING**: Project renamed from `msgraph-testing-tool` to `msgraphgolangtestingtool`
- Executable name changed to `msgraphgolangtestingtool.exe`
- Module name updated to `msgraphgolangtestingtool` in `go.mod`
- Log file name format updated to `_msgraphgolangtestingtool_{date}.csv`
- All documentation updated to reflect the new project name

## [1.10.0] - 2026-01-03

### Added
- Proxy support via `-proxy` flag and `MSGRAPHPROXY` environment variable
- Support for HTTP and HTTPS proxies (e.g., `http://proxy.example.com:8080`)
- Automatic configuration of system proxy settings when specified

### Usage
```powershell
# Using command-line flag
.\msgraphgolangtestingtool.exe -proxy "http://proxy.example.com:8080" -tenantid "xxx" -clientid "yyy" -mailbox "user@example.com"

# Using environment variable
$env:MSGRAPHPROXY = "http://proxy.example.com:8080"
.\msgraphgolangtestingtool.exe -tenantid "xxx" -clientid "yyy" -mailbox "user@example.com"
```

## [1.9.0] - 2026-01-03

### Changed
- **BREAKING**: Environment variable names simplified - removed underscores for easier PowerShell usage
- All environment variables changed from `MSGRAPH_NAME` format to `MSGRAPHNAME` format
- Command-line flags remain unchanged

### Migration Guide
Update environment variable names by removing underscores:

| Old (v2.x) | New (v1.9) |
|------------|-----------|
| `MSGRAPH_TENANT` | `MSGRAPHTENANT` |
| `MSGRAPH_CLIENTID` | `MSGRAPHCLIENTID` |
| `MSGRAPH_SECRET` | `MSGRAPHSECRET` |
| `MSGRAPH_PFX` | `MSGRAPHPFX` |
| `MSGRAPH_PFXPASS` | `MSGRAPHPFXPASS` |
| `MSGRAPH_THUMBPRINT` | `MSGRAPHTHUMBPRINT` |
| `MSGRAPH_MAILBOX` | `MSGRAPHMAILBOX` |
| `MSGRAPH_TO` | `MSGRAPHTO` |
| `MSGRAPH_CC` | `MSGRAPHCC` |
| `MSGRAPH_BCC` | `MSGRAPHBCC` |
| `MSGRAPH_SUBJECT` | `MSGRAPHSUBJECT` |
| `MSGRAPH_BODY` | `MSGRAPHBODY` |
| `MSGRAPH_INVITE_SUBJECT` | `MSGRAPHINVITESUBJECT` |
| `MSGRAPH_START` | `MSGRAPHSTART` |
| `MSGRAPH_END` | `MSGRAPHEND` |
| `MSGRAPH_ACTION` | `MSGRAPHACTION` |

Example:
```powershell
# Old (v2.x)
$env:MSGRAPH_TENANT = "xxx"
$env:MSGRAPH_CLIENTID = "yyy"

# New (v1.9)
$env:MSGRAPHTENANT = "xxx"
$env:MSGRAPHCLIENTID = "yyy"
```

## [1.8.0] - 2026-01-03

### Changed
- **BREAKING**: Renamed command-line flag `-tenant` to `-tenantid`
- **BREAKING**: Renamed command-line flag `-client` to `-clientid`
- Environment variable names remain unchanged (`MSGRAPH_TENANT` and `MSGRAPH_CLIENTID`)

### Migration Guide
Replace in your scripts:
- `-tenant` → `-tenantid`
- `-client` → `-clientid`

Example:
```powershell
# Old (v1.x)
.\msgraphgolangtestingtool.exe -tenant "xxx" -client "yyy" -mailbox "user@example.com"

# New (v1.8)
.\msgraphgolangtestingtool.exe -tenantid "xxx" -clientid "yyy" -mailbox "user@example.com"
```

## [1.3.0] - 2026-01-03

### Added
- Environment variable support for all configuration parameters with `MSGRAPH_` prefix
- Command-line flags take precedence over environment variables
- Supported environment variables:
  - `MSGRAPH_TENANT` - Azure Tenant ID
  - `MSGRAPH_CLIENTID` - Application (Client) ID
  - `MSGRAPH_SECRET` - Client Secret
  - `MSGRAPH_PFX` - Path to PFX certificate file
  - `MSGRAPH_PFXPASS` - PFX file password
  - `MSGRAPH_THUMBPRINT` - Certificate thumbprint
  - `MSGRAPH_MAILBOX` - Target mailbox email address
  - `MSGRAPH_TO` - To recipients (comma-separated)
  - `MSGRAPH_CC` - CC recipients (comma-separated)
  - `MSGRAPH_BCC` - BCC recipients (comma-separated)
  - `MSGRAPH_SUBJECT` - Email subject
  - `MSGRAPH_BODY` - Email body
  - `MSGRAPH_INVITE_SUBJECT` - Calendar invite subject
  - `MSGRAPH_START` - Calendar invite start time
  - `MSGRAPH_END` - Calendar invite end time
  - `MSGRAPH_ACTION` - Action to perform

## [1.2.1] - 2026-01-03

### Fixed
- `getevents` now always logs at least one CSV entry, even when no events are found (previously logged nothing if calendar was empty)
- `getinbox` now always logs at least one CSV entry, even when no messages are found (previously logged nothing if inbox was empty)
- Both actions now log a summary entry showing total count of items retrieved

### Added
- User-friendly console output when no events/messages are found
- Summary count displayed after retrieving events or messages

## [1.2.0] - 2026-01-03

### Changed
- **BREAKING**: Reorganized CSV column order - Status now appears immediately after Action (3rd column) for better readability
- All CSV logs now include Status column for consistency (previously missing from `getevents` and `getinbox`)
- New CSV column order: Timestamp, Action, Status, [other parameters]

### Added
- Status tracking for `getevents` and `getinbox` operations (always "Success" if operation completes)

## [1.1.3] - 2026-01-03

### Changed
- Removed redundant file cleanup code in certificate export function (defer handles cleanup)

## [1.1.2] - 2026-01-03

### Added
- PowerShell certificate script now exports public key (CER file) for Azure AD upload
- Comprehensive error handling in certificate generation script
- Informative colored output showing certificate details and next steps

### Changed
- PowerShell script now validates password is not empty
- Improved user guidance with clear instructions on how to use generated certificates

## [1.1.1] - 2026-01-03

### Fixed
- Corrected typo in default email body message: "It's test message" → "It's a test message"

## [1.1.0] - 2026-01-03

### Added
- Calendar invite now supports customizable start and end times via `-start` and `-end` flags (RFC3339 format)
- Calendar invite subject can be customized via `-invite-subject` flag
- Intelligent defaults: start time defaults to current time, end time defaults to 1 hour after start
- Enhanced CSV logging for calendar invites now includes subject, start time, end time, and event ID

### Fixed
- Calendar invites now properly set required start and end times, preventing invalid event creation

## [1.0.3] - 2026-01-03

### Fixed
- Added nil checks before dereferencing event subject and ID pointers in `listEvents()` to prevent panics when API returns nil values

## [1.0.2] - 2026-01-03

### Security
- Temporary certificate files now use random UUIDs instead of exposing certificate thumbprints in filenames

### Changed
- Renamed temp file prefix from "gemini_cert" to "msgraph_cert" for consistency

## [1.0.1] - 2026-01-03

### Added
- Version flag (`-version`) to display current tool version

### Security
- Fixed unchecked cryptographic error in random password generation that could have resulted in weak temporary passwords

## [1.0.0] - 2026-01-03

### Added
- Initial release of Microsoft Graph Testing Tool
- Support for three authentication methods:
  - Client Secret authentication
  - PFX certificate file authentication
  - Windows Certificate Store authentication (via thumbprint)
- Four core operations:
  - `getevents`: List calendar events from user mailbox
  - `sendmail`: Send emails with To/CC/BCC support
  - `sendinvite`: Create calendar meeting invitations
  - `getinbox`: List newest 10 inbox messages
- Automatic CSV logging to Windows temp directory
- Single-file Go implementation for portability
- PowerShell script for self-signed certificate generation
- Command-line flag-based configuration

### Known Issues
- Email body supports TEXT format only (HTML support planned)
- Calendar invite creation lacks start/end time configuration
- Windows-only platform support (PowerShell dependency)

[1.0.0]: https://github.com/yourorg/msgraphgolangtestingtool/releases/tag/v1.0.0

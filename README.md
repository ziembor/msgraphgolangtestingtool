# Microsoft Graph & SMTP Testing Tools

Portable, single-binary CLI tools for testing and managing email infrastructure - both cloud (Exchange Online via Microsoft Graph) and on-premises (SMTP servers).

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Purpose

This repository contains two complementary tools for comprehensive email infrastructure testing:

- **msgraphgolangtestingtool**: Microsoft Graph API client for Exchange Online mailbox operations (send mail, calendar events, inbox management).
- **smtptool**: SMTP connectivity testing tool with comprehensive TLS diagnostics for on-premises Exchange servers and generic SMTP servers.

Both tools are lightweight, standalone executables requiring no additional runtimes or dependencies. Cross-platform support for Windows, Linux, and macOS with automatic CSV logging.

## Key Features

### Microsoft Graph Tool (msgraphgolangtestingtool)
- **Authentication**: Client Secret, PFX Certificate, Windows Certificate Store (Thumbprint).
- **Operations**: Get Events, Send Mail, Send Invite, Get Inbox, Get Schedule, Export Inbox, Search and Export.
- **Target**: Exchange Online (cloud-based) mailboxes.

### SMTP Tool (smtptool)
- **Operations**: Test Connect, Test STARTTLS (comprehensive TLS diagnostics), Test Auth, Send Mail.
- **Diagnostics**: SSL/TLS handshake analysis, certificate validation, cipher strength assessment, Exchange detection.
- **Target**: On-premises Exchange servers and generic SMTP servers.

### Both Tools
- **Logging**: Automatic CSV logging of all operations to `%TEMP%`.
- **Portable**: Single binary, no dependencies.

## Documentation

- **[BUILD.md](BUILD.md)**: Build instructions for both tools.
- **[SMTP_TOOL_README.md](SMTP_TOOL_README.md)**: Complete SMTP tool documentation and usage guide.
- **[EXAMPLES.md](EXAMPLES.md)**: Microsoft Graph tool usage examples.
- **[RELEASE.md](RELEASE.md)**: Release process and versioning policy.
- **[SECURITY.md](SECURITY.md)**: Security policy, threat model, and best practices ⚠️ **Read this before production use**.
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Common errors and solutions.

## Quick Start

### Build Both Tools
```powershell
# Build both tools at once
.\build-all.ps1

# Or build individually
go build -C cmd/msgraphtool -o msgraphgolangtestingtool.exe
go build -C cmd/smtptool -o smtptool.exe
```
See [BUILD.md](BUILD.md) for cross-platform builds and additional options.

### Usage Examples

**Microsoft Graph Tool:**
```powershell
# Get calendar events
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action getevents

# Send email
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action sendmail
```
See [EXAMPLES.md](EXAMPLES.md) for comprehensive scenarios.

**SMTP Tool:**
```powershell
# Test SMTP connectivity
.\smtptool.exe -action testconnect -host smtp.example.com -port 25

# Test STARTTLS with comprehensive TLS diagnostics
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587

# Send test email
.\smtptool.exe -action sendmail -host smtp.example.com -port 587 -username "user@example.com" -password "..." -from "sender@example.com" -to "recipient@example.com"
```
See [SMTP_TOOL_README.md](SMTP_TOOL_README.md) for complete documentation.

### Environment Variables
- **Microsoft Graph Tool**: `MSGRAPH` prefix (e.g., `MSGRAPHTENANTID`, `MSGRAPHSECRET`)
- **SMTP Tool**: `SMTP` prefix (e.g., `SMTPHOST`, `SMTPPORT`, `SMTPUSERNAME`)

## Security Considerations

⚠️ **Important**: These are diagnostic CLI tools designed for authorized personnel (system administrators, IT staff).

- **CLI flags and environment variables are trusted input** from authorized users
- **Not designed for untrusted web/API input** or public-facing services
- **Defense-in-depth measures** implemented in v2.0.2+ (CRLF sanitization, v2.1.0+: password masking)
- **Comprehensive security testing** in v2.1.0 with 100% coverage on critical functions
- **See [SECURITY.md](SECURITY.md)** for complete threat model and deployment guidelines

**Before production use:**
1. Review [SECURITY.md](SECURITY.md) for security assumptions
2. Follow credential management best practices
3. Restrict tool execution to authorized personnel
4. Monitor CSV logs for unauthorized usage

## License
This tool is provided as-is for testing and automation purposes.

                          ..ooOO END OOoo..



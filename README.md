# Microsoft Graph EXO Mails/Calendar Golang Testing Tool

A portable, single-binary CLI tool for interacting with Microsoft Graph API to manage Exchange Online emails and calendar events.

**Repository:** [https://github.com/ziembor/msgraphgolangtestingtool](https://github.com/ziembor/msgraphgolangtestingtool)

## Purpose

This tool provides a lightweight, standalone executable for testing and managing Microsoft Graph API operations on Exchange Online mailboxes without requiring additional runtimes or dependencies. Cross-platform support for Windows, Linux, and macOS with multiple authentication methods and automatic CSV logging.

## Key Features

- **Authentication**: Client Secret, PFX Certificate, Windows Certificate Store (Thumbprint).
- **Operations**: Get Events, Send Mail, Send Invite, Get Inbox, Get Schedule, Export Inbox, Search and Export.
- **Logging**: Automatic CSV logging of all operations to `%TEMP%`.
- **Portable**: Single binary, no dependencies.

## Documentation

- **[BUILD.md](BUILD.md)**: Build instructions.
- **[RELEASE.md](RELEASE.md)**: Release process and versioning policy.
- **[EXAMPLES.md](EXAMPLES.md)**: Comprehensive usage examples.
- **[SECURITY.md](SECURITY.md)**: Security policy and best practices.
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Common errors and solutions.

## Quick Start

### Build
```powershell
go build -C src -o msgraphgolangtestingtool.exe
```
See [BUILD.md](BUILD.md) for details.

### Usage
```powershell
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action getevents
```
See [EXAMPLES.md](EXAMPLES.md) for more scenarios.

### Environment Variables
All flags can be set via `MSGRAPH` prefix (e.g., `MSGRAPHTENANTID`, `MSGRAPHSECRET`).

## License
This tool is provided as-is for testing and automation purposes.

                          ..ooOO END OOoo..



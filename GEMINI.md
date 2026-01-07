# Gemini Context: msgraphgolangtestingtool

This document provides context for Gemini to effectively assist with the `msgraphgolangtestingtool` project.

## Project Overview

**Name:** `msgraphgolangtestingtool`
**Type:** Go CLI Tool (Single Binary)
**Purpose:** Interact with Microsoft Graph API (Exchange Online) for testing and automation.
**Key Functionality:**
-   Sending emails (Text body).
-   Listing calendar events.
-   Creating calendar invites.
-   Listing inbox messages.
-   **Authentication:** Client Secret, PFX Certificate, Windows Certificate Store (Thumbprint).
-   **Logging:** Automatic CSV logging of all operations to `%TEMP%`.
**Platform:** Cross-platform (Windows, Linux, macOS), but optimized for Windows (Certificate Store support).

## Directory Structure

*   `src/`: Go source code and module definition (`go.mod`).
*   `src/msgraphgolangtestingtool.go`: Main entry point and application logic.
*   `src/cert_windows.go`: Windows-specific certificate store implementation.
*   `src/cert_stub.go`: Stub for non-Windows builds.
*   `src/VERSION`: Current version string (e.g., `1.12.6`).
*   `.github/workflows/`: GitHub Actions for CI/CD.
*   `tests/`: Integration tests.
*   `Changelog/`: Directory containing changelog files (e.g., `1.12.6.md`).

## Development Workflow

### 1. Build

The Go code is located in `src/`. You must run build commands from the root pointing to `src` or inside `src`.

**Command:**
```powershell
go build -C src -o msgraphgolangtestingtool.exe
```

**Optimized Build (Smaller Binary):**
```powershell
go build -C src -ldflags="-s -w" -o msgraphgolangtestingtool.exe
```

### 2. Versioning

*   **Major Version Locked:** Always `1.x.y`.
*   **Single Source of Truth:** `src/VERSION`.
*   **Mechanism:** Go code uses `//go:embed VERSION` to compile the version string. **Do not** edit Go source for version updates.
*   **Process:** Update `src/VERSION` and create `Changelog/{version}.md`.

### 3. Release Process

Releases are automated via GitHub Actions when a tag is pushed.

1.  **Recommended:** Use `.\release.ps1` for interactive release.
2.  **Manual:**
    *   Update `src/VERSION`.
    *   Create `Changelog/{version}.md`.
    *   Commit and Tag (`git tag v1.x.y`).
    *   Push tag (`git push origin v1.x.y`).

This triggers `.github/workflows/build.yml` which builds the binary and attaches it to a GitHub Release.

## Usage & Testing

### Common Commands

*   **Get Events:**
    ```powershell
    .\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action getevents
    ```
*   **Send Mail:**
    ```powershell
    .\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -secret "..." -mailbox "user@example.com" -action sendmail -to "recipient@example.com"
    ```

### Authentication Methods

1.  **Client Secret:** `-secret "VALUE"`
2.  **PFX File:** `-pfx "file.pfx" -pfxpass "PASSWORD"`
3.  **Windows Store:** `-thumbprint "HEX_HASH"` (Windows only)

### Environment Variables

All flags can be set via `MSGRAPH` prefix (e.g., `MSGRAPHTENANTID`, `MSGRAPHCLIENTID`). Flags take precedence over environment variables.

### Logging

CSV logs are written to `%TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv`.

## Key Conventions

*   **Code Style:** Standard Go fmt.
*   **Dependencies:** Minimal. `msgraph-sdk-go`, `azidentity`, `pkcs12`.
*   **Safety:** Never commit secrets. The tool masks secrets in verbose output.
*   **Architecture:** `main` function dispatches to specific action functions (`performAction...`).

## Reference Files

*   `CLAUDE.md`: Detailed architecture and AI context (primary reference).
*   `BUILD.md`: Detailed build instructions.
*   `RELEASE.md`: Release process and versioning policy.
*   `README.md`: User-facing documentation.
*   `SECURITY.md`: Security policy and best practices.
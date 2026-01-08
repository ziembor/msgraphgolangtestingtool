# Architecture Diagram - Microsoft Graph EXO Mails/Calendar Golang Testing Tool

## File Structure and Dependencies

```bash
msgraphgolangtestingtool/
├── src/
│   ├── msgraphgolangtestingtool.go  (Main CLI entry point)
│   ├── shared.go                     (Business logic & utilities)
│   ├── shared_test.go                (Unit tests)
│   ├── integration_test.go           (Integration tests)
│   ├── cert_windows.go               (Windows cert store - +build windows)
│   ├── cert_stub.go                  (Cross-platform stub - +build !windows)
│   ├── go.mod                        (Go module definition)
│   └── VERSION                       (Version file - embedded at compile time)
├── run-integration-tests.ps1                       (Release automation script)
└── selfsignedcert.ps1               (Certificate generation utility)
```

## Main Application Flow

```bash
┌─────────────────────────────────────────────────────────────────┐
│                    msgraphgolangtestingtool.go                   │
│                        (Main Entry Point)                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ├─► main()
                           │   ├─► Parse flags & environment variables
                           │   ├─► validateConfiguration()
                           │   ├─► setupLogger()
                           │   └─► Route to action handlers
                           │
                           ├─► Action Handlers (dispatch based on -action flag)
                           │   ├─► listEvents()      (-action getevents)
                           │   ├─► sendEmail()       (-action sendmail)
                           │   ├─► createInvite()    (-action sendinvite)
                           │   ├─► listInbox()       (-action getinbox)
                           │   ├─► exportInbox()     (-action exportinbox)
                           │   └─► searchAndExport() (-action searchandexport)
                           │
                           └─► Utility Functions
                               ├─► showVersion()
                               ├─► generateBashCompletion()
                               └─► generatePowerShellCompletion()
```

## Shared Business Logic (shared.go)

### Authentication & Client Setup

```bash
┌────────────────────────────────────────────────────────────────┐
│                    Authentication Layer                         │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          ├─► setupGraphClient()
                          │   └─► getCredential()
                          │       ├─► azidentity.NewClientSecretCredential()
                          │       │   (uses -secret flag)
                          │       │
                          │       ├─► azidentity.NewClientCertificateCredential()
                          │       │   ├─► From PFX file (-pfx + -pfxpass)
                          │       │   │   └─► pkcs12.DecodeChain()
                          │       │   │
                          │       │   └─► From Windows Cert Store (-thumbprint)
                          │       │       └─► getCertFromStore_windows.go
                          │       │           └─► Windows CryptoAPI (crypt32.dll)
                          │       │
                          │       └─► Returns: azcore.TokenCredential
                          │
                          └─► retryWithBackoff()
                              ├─► isRetryableError()
                              └─► Exponential backoff (50ms → 10s cap)
```

### Core Graph API Operations

```bash
┌────────────────────────────────────────────────────────────────┐
│                    Microsoft Graph API Layer                    │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          ├─► listEvents()
                          │   └─► client.Users().ByUserId().Events().Get()
                          │       └─► Returns: []models.Event
                          │
                          ├─► sendEmail()
                          │   ├─► createRecipients()
                          │   ├─► createFileAttachments()
                          │   │   └─► getAttachmentContentBase64()
                          │   └─► client.Users().ByUserId().SendMail().Post()
                          │
                          ├─► createInvite()
                          │   ├─► parseFlexibleTime()
                          │   ├─► createRecipients()
                          │   └─► client.Users().ByUserId().Events().Post()
                          │
                          ├─► listInbox()
                          │   └─► client.Users().ByUserId().Messages().Get()
                          │       └─► Returns: []models.Message
                          │
                          ├─► exportInbox()
                          │   ├─► client.Users().ByUserId().Messages().Get()
                          │   ├─► Create date-stamped directory (%TEMP%\export\{date})
                          │   └─► Export each message to individual JSON file
                          │
                          └─► searchAndExport()
                              ├─► client.Users().ByUserId().Messages().Get()
                              │   └─► Filter by InternetMessageId
                              └─► Export matching message to JSON file
```

### Validation & Helper Functions

```bash
┌────────────────────────────────────────────────────────────────┐
│                    Validation & Utilities                       │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          ├─► Validation Functions
                          │   ├─► validateConfiguration()    [87.8% coverage]
                          │   ├─► validateEmail()            [100% coverage]
                          │   ├─► validateEmails()           [100% coverage]
                          │   ├─► validateGUID()             [100% coverage]
                          │   ├─► validateFilePath()         [77.3% coverage]
                          │   ├─► validateRFC3339Time()      [100% coverage]
                          │   └─► parseFlexibleTime()        [100% coverage]
                          │
                          ├─► Data Transformation
                          │   ├─► createRecipients()         [100% coverage]
                          │   ├─► createFileAttachments()    [95.2% coverage]
                          │   └─► getAttachmentContentBase64()[100% coverage]
                          │
                          ├─► Security & Masking
                          │   ├─► maskSecret()               [100% coverage]
                          │   └─► maskGUID()                 [100% coverage]
                          │
                          ├─► Logging & Diagnostics
                          │   ├─► setupLogger()
                          │   ├─► parseLogLevel()            [100% coverage]
                          │   ├─► logVerbose()               [100% coverage]
                          │   └─► printTokenInfo()           [0% - requires auth]
                          │
                          └─► Helper Functions
                              ├─► Int32Ptr()                 [100% coverage]
                              ├─► enrichGraphAPIError()
                              └─► CSV logging functions
```

## Test Suite Architecture

```bash
┌────────────────────────────────────────────────────────────────┐
│                        Test Structure                           │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          ├─► shared_test.go (Unit Tests)
                          │   ├─► //go:build !integration
                          │   │
                          │   ├─► Data Transformation Tests
                          │   │   ├─► TestCreateFileAttachments
                          │   │   ├─► TestGetAttachmentContentBase64
                          │   │   └─► TestCreateRecipients
                          │   │
                          │   ├─► Validation Tests
                          │   │   ├─► TestValidateEmail
                          │   │   ├─► TestValidateGUID
                          │   │   ├─► TestValidateFilePath
                          │   │   ├─► TestValidateConfiguration
                          │   │   └─► TestParseFlexibleTime
                          │   │
                          │   ├─► Retry Logic Tests
                          │   │   ├─► TestIsRetryableError
                          │   │   └─► TestRetryWithBackoff
                          │   │
                          │   ├─► Security Tests
                          │   │   ├─► TestMaskSecret
                          │   │   └─► TestMaskGUID
                          │   │
                          │   ├─► Completion Script Tests
                          │   │   ├─► TestGenerateBashCompletion
                          │   │   └─► TestGeneratePowerShellCompletion
                          │   │
                          │   └─► Helper Function Tests
                          │       ├─► TestInt32Ptr
                          │       ├─► TestLogVerbose
                          │       ├─► TestParseLogLevel
                          │       └─► TestStringSlice*
                          │
                          └─► integration_test.go (Integration Tests)
                              ├─► //go:build integration
                              └─► Requires live Azure AD & Graph API
```

## Certificate Authentication Flow (Windows)

```bash
┌────────────────────────────────────────────────────────────────┐
│              Windows Certificate Store Integration              │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          └─► cert_windows.go (Windows only)
                              │
                              ├─► getCertFromStore()
                              │   ├─► syscall.LoadDLL("crypt32.dll")
                              │   ├─► CertOpenStore(CERT_SYSTEM_STORE_CURRENT_USER)
                              │   ├─► CertFindCertificateInStore(by thumbprint)
                              │   ├─► PFXExportCertStoreEx() → memory buffer
                              │   ├─► pkcs12.DecodeChain()
                              │   └─► Returns: crypto.PrivateKey + x509.Certificate
                              │
                              └─► Uses Windows CryptoAPI
                                  ├─► No temporary files created
                                  ├─► Certificate extracted to memory only
                                  └─► Automatic cleanup via defer
```

## Release Automation (run-integration-tests.ps1)

```bash
┌────────────────────────────────────────────────────────────────┐
│                    Release Script Workflow                      │
└─────────────────────────┬──────────────────────────────────────┘
                          │
                          ├─► Step 1: Git Status Check
                          │   └─► Ensures working tree is clean
                          │
                          ├─► Step 2: Security Scan for Secrets
                          │   ├─► Scan patterns:
                          │   │   ├─► Azure AD Client Secrets ([a-zA-Z0-9~_-]{34,})
                          │   │   ├─► GUIDs/UUIDs (standard format)
                          │   │   ├─► Email addresses (non-example domains)
                          │   │   └─► API Keys (access_token, secret_key, etc.)
                          │   │
                          │   ├─► Scanned files:
                          │   │   ├─► test-results/*.md
                          │   │   ├─► ChangeLog/*.md
                          │   │   ├─► *.md (root level)
                          │   │   └─► src/*.go
                          │   │
                          │   ├─► Smart filtering:
                          │   │   ├─► Skip EXAMPLES, README, CLAUDE, IMPROVEMENTS
                          │   │   ├─► Skip placeholders (xxx, yyy, example.com)
                          │   │   └─► Skip known safe patterns
                          │   │
                          │   └─► Blocks release if secrets detected
                          │
                          ├─► Step 3: Version Management
                          │   ├─► Read src/VERSION
                          │   ├─► Validate 1.x.y format (major locked at 1)
                          │   ├─► Prompt for new version
                          │   └─► Update src/VERSION
                          │
                          ├─► Step 4: Changelog Creation
                          │   ├─► Interactive prompts for:
                          │   │   ├─► Added features
                          │   │   ├─► Changed features
                          │   │   ├─► Fixed bugs
                          │   │   └─► Security updates
                          │   └─► Create ChangeLog/{version}.md
                          │
                          ├─► Steps 5-7: Git Operations
                          │   ├─► git commit (formatted message)
                          │   ├─► git push origin {branch}
                          │   └─► git tag v{version} && git push origin --tags
                          │
                          └─► Steps 8-12: Optional GitHub Integration
                              ├─► Create Pull Request (via gh CLI)
                              ├─► Monitor GitHub Actions workflow
                              └─► Open releases page
```

## Data Flow Example: Send Email with Attachments

```bash
User Command:
  msgraphgolangtestingtool.exe -action sendmail -to "user@example.com"
    -subject "Test" -body "Hello" -attachment "file.pdf"
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. main() - Parse flags & validate configuration                │
│    └─► validateConfiguration() checks all required fields       │
└─────────────────────┬───────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. setupGraphClient() - Authenticate                            │
│    ├─► getCredential() → azcore.TokenCredential                 │
│    └─► msgraphsdk.NewGraphServiceClientWithCredentials()        │
└─────────────────────┬───────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. sendEmail() - Build and send message                         │
│    ├─► createRecipients(["user@example.com"]) → []Recipient    │
│    ├─► createFileAttachments(["file.pdf"])                      │
│    │   └─► getAttachmentContentBase64() → base64 string         │
│    ├─► Build models.Message object                              │
│    └─► client.Users().ByUserId().SendMail().Post()              │
└─────────────────────┬───────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. retryWithBackoff() - Handle transient failures               │
│    ├─► isRetryableError() checks status codes (429, 503, 504)  │
│    └─► Exponential backoff: 50ms → 100ms → 200ms → ... → 10s   │
└─────────────────────┬───────────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. CSV Logging - Record operation result                        │
│    └─► %TEMP%\_msgraphgolangtestingtool_sendmail_2026-01-05.csv│
│        Timestamp, Action, Status, Mailbox, To, CC, BCC, Subject │
└─────────────────────────────────────────────────────────────────┘
```

## Test Coverage by Function Category

```bash
┌──────────────────────────────────────────────────────────────────┐
│                    Coverage Summary (24.6%)                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ✅ 100% Coverage (15 functions):                                │
│     • Int32Ptr(), logVerbose()                                   │
│     • maskGUID(), maskSecret()                                   │
│     • getAttachmentContentBase64(), createRecipients()           │
│     • validateEmail(), validateEmails(), validateGUID()          │
│     • parseFlexibleTime(), validateRFC3339Time()                 │
│     • parseLogLevel(), isRetryableError()                        │
│     • generateBashCompletion(), generatePowerShellCompletion()   │
│                                                                  │
│  ⚠️  High Coverage (2 functions):                                │
│     • createFileAttachments() - 95.2%                            │
│     • validateConfiguration() - 87.8%                            │
│     • retryWithBackoff() - 88.9%                                 │
│     • validateFilePath() - 77.3%                                 │
│                                                                  │
│  ❌ No Coverage - Integration Only (6 functions):                │
│     • setupGraphClient(), getCredential()                        │
│     • sendEmail(), listEvents(), createInvite(), listInbox()     │
│     • printTokenInfo()                                           │
│     (Require live Azure AD + Graph API access)                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Key Design Patterns

### 1. Table-Driven Tests

All unit tests use the table-driven pattern for maintainability:

```go
tests := []struct {
    name     string
    input    string
    expected string
    wantErr  bool
}{ /* test cases */ }
```

### 2. Config Struct Pattern

Centralized configuration simplifies function signatures:

```go
type Config struct { /* all configuration */ }
func sendEmail(ctx context.Context, client *msgraphsdk.GraphServiceClient,
               cfg *Config) error
```

### 3. Retry with Exponential Backoff

Handles transient failures gracefully:

```go
retryWithBackoff(ctx, maxRetries, baseDelay, operation func() error)
```

### 4. Platform-Specific Builds

Build tags enable Windows-specific features while maintaining cross-platform support for **Windows, Linux, and macOS**:

```go
// cert_windows.go - //go:build windows (Windows Certificate Store access)
// cert_stub.go    - //go:build !windows (Linux/macOS stub)
```

**GitHub Actions Workflow** builds binaries for all three platforms:

- `msgraphgolangtestingtool-windows.zip` - Windows binary (.exe)
- `msgraphgolangtestingtool-linux.zip` - Linux binary (ELF)
- `msgraphgolangtestingtool-macos.zip` - macOS binary (Mach-O)

**Note:** The `-thumbprint` authentication method (Windows Certificate Store) is only available on Windows. Linux and macOS users should use `-secret` or `-pfx` authentication.

### 5. CSV Logging Pattern

Action-specific CSV files prevent schema conflicts:

```bash
_msgraphgolangtestingtool_sendmail_2026-01-05.csv
_msgraphgolangtestingtool_getevents_2026-01-05.csv
_msgraphgolangtestingtool_getinbox_2026-01-05.csv
_msgraphgolangtestingtool_exportinbox_2026-01-07.csv
_msgraphgolangtestingtool_searchandexport_2026-01-07.csv
```

### 6. JSON Export Pattern (v1.21.0+)

Export actions create date-stamped directories with individual JSON files:

```bash
%TEMP%\export\2026-01-07\
├── message_1_2026-01-07T10-30-45.json
├── message_2_2026-01-07T10-25-12.json
├── message_3_2026-01-07T09-58-03.json
└── message_search_2026-01-07T11-15-30.json (from searchandexport)
```

---

**Version:** 1.21.0
**Last Updated:** 2026-01-07
**Total Lines of Code:** 2,949 (862 test + 1,462 source + 625 automation)
**Test Coverage:** 24.6% (46 passing tests)

                          ..ooOO END OOoo..

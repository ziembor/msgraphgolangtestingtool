# Unit Tests Documentation

## Overview

This document describes the Go unit tests for the Microsoft Graph EXO Mails/Calendar Golang Testing Tool. The test suite provides comprehensive coverage of helper functions, validation logic, data transformation, and utility functions.

**Test Coverage:** 24.6% of statements (46 passing tests)

**Test Files:**
- `src/shared_test.go` - Tests for shared business logic
- `src/msgraphgolangtestingtool_test.go` - Tests for main program logic

## Running Tests

### Run All Unit Tests

```bash
cd src
go test -v
```

### Run Specific Test

```bash
cd src
go test -v -run TestCreateFileAttachments
```

### Generate Coverage Report

```bash
cd src
go test -coverprofile=coverage.out
go tool cover -html=coverage.out  # Open in browser
go tool cover -func=coverage.out  # Text report
```

### Run Tests Without Integration Tests

By default, unit tests exclude integration tests using build tags:

```bash
cd src
go test -v  # Excludes integration tests
go test -v -tags=integration  # Includes integration tests
```

## Test Categories

### 1. Data Transformation Tests

#### TestCreateFileAttachments
**Location:** `src/shared_test.go:444-541`
**Coverage:** 95.2% of `createFileAttachments()`

Tests file attachment creation for email messages.

**Test Cases:**
- Single file attachment
- Multiple file attachments
- Empty file list
- Nonexistent files (graceful skip)
- All files nonexistent (error)

**Example:**
```go
t.Run("single file attachment", func(t *testing.T) {
    attachments, err := createFileAttachments([]string{tmpFile1.Name()}, &Config{VerboseMode: false})
    // Expects: 1 attachment, no error
})
```

#### TestGetAttachmentContentBase64
**Location:** `src/shared_test.go:543-580`
**Coverage:** 100% of `getAttachmentContentBase64()`

Tests base64 encoding of file content.

**Test Cases:**
- Empty data
- Simple text ("Hello World" → "SGVsbG8gV29ybGQ=")
- Binary data (0x00, 0xFF, 0xAA, 0x55)
- Newline characters

#### TestCreateRecipients
**Location:** `src/shared_test.go` (lines vary)
**Coverage:** 100% of `createRecipients()`

Tests email recipient object creation.

**Test Cases:**
- Empty list
- Single recipient
- Multiple recipients
- Three recipients

### 2. Validation Tests

#### TestValidateEmail
**Coverage:** 100% of `validateEmail()`

Tests email address validation.

**Test Cases:**
- Valid email formats
- Email with subdomain
- Email with dots
- Missing @ symbol
- Empty string
- No domain part
- Multiple @ symbols

**Example:**
```go
validateEmail("user@example.com")  // Valid
validateEmail("invalid")           // Error: no @
validateEmail("")                  // Error: empty
```

#### TestValidateGUID
**Coverage:** 100% of `validateGUID()`

Tests GUID/UUID validation.

**Test Cases:**
- Valid GUID format (12345678-1234-1234-1234-123456789012)
- Lowercase GUID
- Too short
- Too long
- No dashes
- Wrong dash positions

#### TestValidateFilePath
**Coverage:** 77.3% of `validateFilePath()`

Tests file path validation with security checks.

**Test Cases:**
- Empty path (allowed)
- Valid absolute path
- Path traversal with ".."
- Windows-style path traversal
- Nonexistent file
- Directory instead of file

**Security Features:**
- Detects path traversal attempts (../)
- Validates file existence
- Ensures path is a file, not directory

#### TestValidateConfiguration
**Coverage:** 87.8% of `validateConfiguration()`

Tests complete configuration validation.

**Test Cases:**
- Valid configuration with secret
- Valid configuration with thumbprint
- Missing tenant ID
- Missing client ID
- Missing mailbox
- No authentication method
- Multiple authentication methods (error)

#### TestParseFlexibleTime
**Coverage:** 100% of `parseFlexibleTime()`

Tests flexible time parsing for calendar events.

**Supported Formats:**
- RFC3339: `2026-01-15T14:00:00Z`
- RFC3339 with offset: `2026-01-15T14:00:00+01:00`
- PowerShell sortable: `2026-01-15T14:00:00`

**Test Cases:**
- Valid RFC3339 formats
- PowerShell datetime formats
- Empty string
- Invalid formats
- Invalid dates

### 3. Retry Logic Tests

#### TestIsRetryableError
**Coverage:** 100% of `isRetryableError()`

Tests error classification for retry logic.

**Retryable Errors:**
- HTTP 429 (Too Many Requests)
- HTTP 503 (Service Unavailable)
- HTTP 504 (Gateway Timeout)
- Timeout errors
- Connection reset
- Network unreachable

**Non-Retryable Errors:**
- HTTP 400 (Bad Request)
- HTTP 404 (Not Found)
- Authentication errors
- Generic errors

**Example:**
```go
err := &azcore.ResponseError{StatusCode: 429}
isRetryableError(err)  // true - rate limited

err := &azcore.ResponseError{StatusCode: 404}
isRetryableError(err)  // false - not found
```

#### TestRetryWithBackoff
**Coverage:** 88.9% of `retryWithBackoff()`

Tests exponential backoff retry mechanism.

**Test Cases:**
- Success on first try (no retries)
- Success after retries
- Max retries exceeded
- Non-retryable error (immediate fail)
- Context cancellation
- Exponential backoff timing
- Delay cap at 10 seconds

**Backoff Strategy:**
- Initial delay: 50ms
- Multiplier: 2x per retry
- Max delay: 10 seconds
- Jitter: ±25%

### 4. Security and Masking Tests

#### TestMaskSecret
**Coverage:** 100% of `maskSecret()`

Tests secret masking for secure display.

**Format:** `<first4>********<last4>`

**Test Cases:**
- Empty string → "********"
- Short string (≤8 chars) → "********"
- Long string → "abcd********xyz1"

#### TestMaskGUID
**Coverage:** 100% of `maskGUID()`

Tests GUID masking for secure display.

**Format:** `<first4>****-****-****-****<last4>`

**Test Cases:**
- Standard GUID → "1234****-****-****-****9012"
- GUID without dashes → "1234****-****-****-****9012"
- Short string → "****"
- Empty string → "****"

### 5. Completion Script Tests

#### TestGenerateBashCompletion
**Coverage:** 100% of `generateBashCompletion()`

Tests bash completion script generation.

**Validates:**
- Script structure (COMPREPLY, COMP_WORDS, COMP_CWORD)
- All flags present (-action, -tenantid, -clientid, etc.)
- Action completions (getevents, sendmail, sendinvite, getinbox)
- Log level completions (DEBUG, INFO, WARN, ERROR)
- Installation instructions

**Example Output:**
```bash
_msgraphgolangtestingtool_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    case "${prev}" in
        -action)
            COMPREPLY=( $(compgen -W "getevents sendmail sendinvite getinbox" -- ${cur}) )
            ;;
    esac
}
```

#### TestGeneratePowerShellCompletion
**Coverage:** 100% of `generatePowerShellCompletion()`

Tests PowerShell completion script generation.

**Validates:**
- Script structure (Register-ArgumentCompleter, CompletionResult)
- All flags present
- Log levels (DEBUG, INFO, WARN, ERROR)
- Shell types (bash, powershell)
- Success message

### 6. Helper Function Tests

#### TestInt32Ptr
**Coverage:** 100% of `Int32Ptr()`

Tests int32 pointer creation helper.

**Test Cases:**
- Zero value (0)
- Positive value (42)
- Negative value (-100)
- Max int32 (2,147,483,647)
- Min int32 (-2,147,483,648)

**Validates:**
- Pointer is not nil
- Dereferenced value matches input
- New memory allocation created

#### TestLogVerbose
**Coverage:** 100% of `logVerbose()`

Tests verbose logging helper.

**Test Cases:**
- Verbose mode enabled (no args)
- Verbose mode enabled (with args)
- Verbose mode disabled (no output)
- Empty format string
- Multiple placeholders

**Example:**
```go
logVerbose(true, "Test message with %s and %d", "string", 42)
// Output: [VERBOSE] Test message with string and 42

logVerbose(false, "This won't print", "arg")
// No output
```

### 7. Logging Tests

#### TestParseLogLevel
**Coverage:** 100% of `parseLogLevel()`

Tests log level parsing from string.

**Supported Levels:**
- DEBUG (case-insensitive)
- INFO (default)
- WARN/WARNING
- ERROR

**Test Cases:**
- Lowercase and uppercase variants
- Invalid level (defaults to INFO)
- Empty string (defaults to INFO)

#### TestSetupLogger
**Coverage:** Tests logger configuration

Tests structured logging setup.

**Test Cases:**
- Verbose mode enables DEBUG level
- Debug level enables DEBUG
- INFO level disables DEBUG
- ERROR level disables DEBUG

### 8. String Slice Tests

#### TestStringSliceSet
**Tests custom flag type for comma-separated values**

Tests string slice parsing from command-line flags.

**Test Cases:**
- Empty string
- Single value
- Multiple values
- Values with spaces (trimmed)
- Trailing commas
- Extra spaces
- Leading commas

**Example:**
```go
var recipients stringSlice
recipients.Set("user1@example.com, user2@example.com")
// Result: []string{"user1@example.com", "user2@example.com"}
```

#### TestStringSliceString
**Tests string representation**

Tests string slice conversion back to string.

**Test Cases:**
- Nil slice → "[]"
- Empty slice → "[]"
- Single item → "[item]"
- Multiple items → "[item1, item2]"

### 9. Certificate Tests

#### TestCreateCertCredential_ModernPFX
**Tests modern PFX certificate handling**

Tests certificate credential creation with modern PFX format.

#### TestCreateCertCredential_LegacyPFX
**Tests legacy PFX certificate handling**

Tests certificate credential creation with legacy PFX format.

#### TestCreateCertCredential_WrongPassword
**Tests password validation**

Ensures wrong password is detected and rejected.

#### TestCreateCertCredential_EmptyPassword
**Tests empty password handling**

Tests certificate with no password protection.

#### TestCreateCertCredential_MalformedPFX
**Tests error handling**

Ensures malformed PFX files are rejected gracefully.

## Coverage Summary

### Functions at 100% Coverage

**Helper Functions:**
- `Int32Ptr()` - Int32 pointer creation
- `logVerbose()` - Verbose logging
- `maskGUID()` - GUID masking
- `maskSecret()` - Secret masking
- `getAttachmentContentBase64()` - Base64 encoding
- `generateBashCompletion()` - Bash completion script
- `generatePowerShellCompletion()` - PowerShell completion script

**Validation:**
- `validateEmail()` - Email validation
- `validateEmails()` - Multiple email validation
- `validateGUID()` - GUID validation
- `parseFlexibleTime()` - Flexible time parsing
- `validateRFC3339Time()` - RFC3339 time validation

**Data Transformation:**
- `createRecipients()` - Recipient creation
- `getAttachmentContentBase64()` - Base64 encoding

**Retry Logic:**
- `isRetryableError()` - Error classification

### Functions Not Tested (Integration Only)

The following functions require live Microsoft Graph API access and are tested via integration tests:

- `setupGraphClient()` - Requires Azure credentials
- `getCredential()` - Requires certificate/secret authentication
- `sendEmail()` - Requires Graph API client
- `listEvents()` - Requires Graph API client
- `createInvite()` - Requires Graph API client
- `listInbox()` - Requires Graph API client
- `printTokenInfo()` - Requires Azure token

See `INTEGRATION_TESTS.md` for integration test documentation.

## Test Organization

### Build Tags

Tests use Go build tags to separate unit and integration tests:

```go
//go:build !integration
// +build !integration
```

**Unit tests:** Exclude integration tag (default)
**Integration tests:** Require integration tag

### Test File Structure

```
src/
├── shared.go                    # Business logic
├── shared_test.go               # Unit tests (46 tests)
├── msgraphgolangtestingtool.go  # Main program
├── msgraphgolangtestingtool_test.go
└── integration_test.go          # Integration tests (requires -tags=integration)
```

## Writing New Tests

### Table-Driven Test Pattern

Use table-driven tests for multiple scenarios:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if result != tt.expected {
                t.Errorf("MyFunction() = %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Test Helpers

**Temporary Files:**
```go
tmpFile, err := os.CreateTemp("", "test-*.txt")
if err != nil {
    t.Fatalf("Failed to create temp file: %v", err)
}
defer os.Remove(tmpFile.Name())
```

**Panic Recovery:**
```go
defer func() {
    if r := recover(); r != nil {
        t.Errorf("Function panicked: %v", r)
    }
}()
```

## Best Practices

1. **Use descriptive test names** - Name test cases clearly (e.g., "valid_email", "empty_input")
2. **Test edge cases** - Empty strings, nil values, boundary conditions
3. **Table-driven tests** - Multiple scenarios in a single test function
4. **Independent tests** - Tests should not depend on each other
5. **Clean up resources** - Use defer to clean up temp files
6. **Error messages** - Provide clear failure messages with expected/actual values

## Continuous Integration

Tests run automatically on:
- Every git push
- Pull request creation
- GitHub Actions workflow

**Required:** All tests must pass before merging.

## Related Documentation

- `INTEGRATION_TESTS.md` - Integration test documentation
- `IMPROVEMENTS.md` - Planned improvements and coverage goals
- `test-results/README.md` - Test results archive
- `CLAUDE.md` - Project documentation

## Version History

**v1.16.11** (2026-01-05)
- Added 7 new unit tests (medium and low priority)
- Improved coverage from 20.9% to 24.6%
- Achieved 100% coverage on 15+ helper functions
- Total: 46 passing tests

---

**Last Updated:** 2026-01-05
**Total Tests:** 46
**Overall Coverage:** 24.6%
--- End of content ---
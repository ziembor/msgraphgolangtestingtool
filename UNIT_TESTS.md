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

---

## Troubleshooting

This section covers common issues you may encounter when running unit tests and how to resolve them.

### Test Failures on Windows vs Linux

**Issue:** Path separator differences causing test failures

**Symptoms:**
- Tests pass on Windows but fail on Linux/macOS (or vice versa)
- Path-related assertions fail with backslash vs forward slash issues
- Error messages like: `got "C:\temp\file.txt", want "/tmp/file.txt"`

**Root Cause:**
Windows uses backslash (`\`) as path separator, while Linux/macOS use forward slash (`/`).

**Solution:**
Use `filepath.Join()` instead of string concatenation for paths:

```go
// ❌ Bad: Hard-coded path separator
path := "src" + "/" + "file.go"

// ✅ Good: Cross-platform path construction
path := filepath.Join("src", "file.go")
```

**Example Fix:**
```go
// Before (platform-specific)
expected := "test-results/output.txt"

// After (cross-platform)
expected := filepath.Join("test-results", "output.txt")
```

---

### Temporary File Location Differences

**Issue:** Tests fail because temporary file paths differ between platforms

**Platforms:**
- **Windows:** `%TEMP%` (typically `C:\Users\<user>\AppData\Local\Temp`)
- **Linux/macOS:** `/tmp`

**Solution:**
Use `os.TempDir()` or `os.CreateTemp()` for portable temporary file handling:

```go
// ❌ Bad: Hard-coded temp directory
tmpFile := "/tmp/test-file.txt"

// ✅ Good: Cross-platform temp directory
tmpDir := os.TempDir()
tmpFile := filepath.Join(tmpDir, "test-file.txt")

// ✅ Best: Use os.CreateTemp() with automatic cleanup
tmpFile, err := os.CreateTemp("", "test-*.txt")
if err != nil {
    t.Fatalf("Failed to create temp file: %v", err)
}
defer os.Remove(tmpFile.Name())
```

---

### Coverage Report Not Generating

**Issue:** Coverage report fails to generate or `coverage.out` file is missing

**Possible Causes:**
1. `go tool cover` not installed or not in PATH
2. Write permissions issue in `src/` directory
3. No tests executed (all skipped)

**Solutions:**

**1. Verify go tool cover is available:**
```bash
go tool cover -h
```

If the command fails, reinstall Go or check PATH configuration.

**2. Check write permissions:**
```bash
# Linux/macOS
ls -la src/

# Windows PowerShell
Get-Acl src/
```

Ensure you have write permissions in the `src/` directory.

**3. Verify tests are running:**
```bash
cd src
go test -v  # Should show test execution

# If all tests are skipped:
go test -v -count=1  # Disable test caching
```

**4. Specify full path for coverage file:**
```bash
cd src
go test -coverprofile=$(pwd)/coverage.out
go tool cover -html=$(pwd)/coverage.out
```

---

### Tests Timing Out

**Issue:** Tests exceed default timeout and are killed

**Symptoms:**
```
panic: test timed out after 10m0s
```

**Cause:**
Default Go test timeout is 10 minutes. Tests with network calls, large file processing, or slow operations may exceed this.

**Solution:**
Increase timeout with `-timeout` flag:

```bash
# Increase to 20 minutes
go test -v -timeout 20m

# Increase to 1 hour (for very slow tests)
go test -v -timeout 1h

# Disable timeout (not recommended)
go test -v -timeout 0
```

**Best Practice:**
Optimize slow tests instead of increasing timeout:
- Use smaller test files
- Mock external dependencies
- Parallelize independent tests with `t.Parallel()`

---

### Build Cache Issues

**Issue:** Tests pass/fail inconsistently due to cached results

**Symptoms:**
- Test results don't change after code modifications
- Old test outputs appear despite code changes
- `PASS (cached)` appears in test output

**Solution:**
Disable test caching:

```bash
# Disable cache for single run
go test -count=1

# Clear entire build cache
go clean -testcache

# Clear all caches (build + module)
go clean -cache -testcache -modcache
```

---

### Import Errors After Refactoring

**Issue:** Tests fail with "undefined" or "not found" errors after moving functions

**Symptoms:**
```
undefined: myFunction
```

**Cause:**
Function moved to different file but tests not updated.

**Solution:**
1. Verify function is exported (starts with capital letter)
2. Check function location (use `grep` or IDE "Go to Definition")
3. Rebuild test binary:

```bash
go clean -testcache
go test -v
```

---

### Race Condition Failures

**Issue:** Tests fail intermittently with race detector enabled

**Symptoms:**
```
WARNING: DATA RACE
```

**Enable race detector:**
```bash
go test -race
```

**Common Causes:**
- Concurrent map access without locks
- Shared variables accessed by goroutines
- Missing synchronization primitives

**Solution:**
Add proper synchronization:

```go
// ❌ Bad: Concurrent map access
var cache = make(map[string]string)
go func() { cache["key"] = "value" }()

// ✅ Good: Use mutex or sync.Map
var mu sync.Mutex
var cache = make(map[string]string)
go func() {
    mu.Lock()
    cache["key"] = "value"
    mu.Unlock()
}()
```

---

## CI/CD Integration

This section provides ready-to-use examples for integrating unit tests into your CI/CD pipelines.

### GitHub Actions

Add to `.github/workflows/test.yml`:

```yaml
name: Unit Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    name: Run Unit Tests
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'  # Use your Go version
          cache: true

      - name: Download dependencies
        run: |
          cd src
          go mod download

      - name: Run unit tests
        run: |
          cd src
          go test -v -coverprofile=coverage.out

      - name: Generate coverage report
        run: |
          cd src
          go tool cover -func=coverage.out
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./src/coverage.out
          flags: unittests
          name: codecov-umbrella
          fail_ci_if_error: true

      - name: Upload coverage artifact
        uses: actions/upload-artifact@v4
        with:
          name: coverage-report
          path: src/coverage.html
```

**Features:**
- ✅ Runs on every push and pull request
- ✅ Caches Go modules for faster builds
- ✅ Generates coverage reports
- ✅ Uploads coverage to Codecov
- ✅ Stores HTML coverage report as artifact

---

### GitLab CI/CD

Add to `.gitlab-ci.yml`:

```yaml
image: golang:1.23

stages:
  - test
  - coverage

variables:
  GO_VERSION: "1.23"

before_script:
  - cd src
  - go mod download

unit-tests:
  stage: test
  script:
    - go test -v -coverprofile=coverage.out
    - go tool cover -func=coverage.out
  coverage: '/total:.*\s(\d+\.\d+)%/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: src/coverage.out
    paths:
      - src/coverage.out
    expire_in: 1 week

coverage-report:
  stage: coverage
  dependencies:
    - unit-tests
  script:
    - cd src
    - go tool cover -html=coverage.out -o coverage.html
  artifacts:
    paths:
      - src/coverage.html
    expire_in: 1 month
  only:
    - main
    - develop
```

**Features:**
- ✅ Runs on every commit
- ✅ Extracts coverage percentage for GitLab UI
- ✅ Generates Cobertura coverage report
- ✅ Stores coverage artifacts
- ✅ Separate coverage HTML generation for main branches

---

### Azure DevOps

Add to `azure-pipelines.yml`:

```yaml
trigger:
  branches:
    include:
      - main
      - develop

pool:
  vmImage: 'ubuntu-latest'

variables:
  GOVERSION: '1.23'
  GOPATH: '$(Agent.WorkFolder)/go'
  GOBIN: '$(GOPATH)/bin'

steps:
  - task: GoTool@0
    displayName: 'Install Go $(GOVERSION)'
    inputs:
      version: '$(GOVERSION)'

  - script: |
      cd src
      go mod download
    displayName: 'Download Go dependencies'

  - script: |
      cd src
      go test -v -coverprofile=coverage.out
    displayName: 'Run unit tests'

  - script: |
      cd src
      go tool cover -func=coverage.out
      go tool cover -html=coverage.out -o coverage.html
    displayName: 'Generate coverage report'

  - task: PublishCodeCoverageResults@2
    inputs:
      codeCoverageTool: 'Cobertura'
      summaryFileLocation: 'src/coverage.out'
    displayName: 'Publish coverage results'

  - task: PublishBuildArtifacts@1
    inputs:
      PathtoPublish: 'src/coverage.html'
      ArtifactName: 'coverage-report'
    displayName: 'Publish coverage HTML'
```

---

### Jenkins

Add to `Jenkinsfile`:

```groovy
pipeline {
    agent any

    tools {
        go 'go-1.23'
    }

    environment {
        GO111MODULE = 'on'
        GOPATH = "${WORKSPACE}/go"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Dependencies') {
            steps {
                dir('src') {
                    sh 'go mod download'
                }
            }
        }

        stage('Unit Tests') {
            steps {
                dir('src') {
                    sh 'go test -v -coverprofile=coverage.out'
                }
            }
        }

        stage('Coverage Report') {
            steps {
                dir('src') {
                    sh 'go tool cover -func=coverage.out'
                    sh 'go tool cover -html=coverage.out -o coverage.html'
                }
            }
        }

        stage('Publish Results') {
            steps {
                publishHTML([
                    reportDir: 'src',
                    reportFiles: 'coverage.html',
                    reportName: 'Coverage Report'
                ])
            }
        }
    }

    post {
        always {
            cleanWs()
        }
    }
}
```

---

### Local CI Simulation

Test your CI/CD configuration locally before pushing:

#### Using Act (GitHub Actions locally)

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run GitHub Actions locally
act -j test
```

#### Using Docker

```bash
# Create a test container
docker run --rm -v $(pwd):/workspace -w /workspace/src golang:1.23 bash -c "
  go mod download &&
  go test -v -coverprofile=coverage.out &&
  go tool cover -func=coverage.out
"
```

---

### Coverage Thresholds

Enforce minimum coverage requirements in CI/CD:

#### GitHub Actions

```yaml
- name: Check coverage threshold
  run: |
    cd src
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    threshold=24.0
    if (( $(echo "$coverage < $threshold" | bc -l) )); then
      echo "Coverage $coverage% is below threshold $threshold%"
      exit 1
    fi
    echo "Coverage $coverage% meets threshold $threshold%"
```

#### GitLab CI

```yaml
coverage-check:
  stage: test
  script:
    - cd src
    - go test -coverprofile=coverage.out
    - |
      coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
      if [ $(echo "$coverage < 24.0" | bc) -eq 1 ]; then
        echo "Coverage $coverage% is below threshold 24.0%"
        exit 1
      fi
```

---

### Pre-commit Hooks

Run tests automatically before every commit:

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Pre-commit hook to run unit tests

echo "Running unit tests..."
cd src
go test -v

if [ $? -ne 0 ]; then
    echo "❌ Unit tests failed. Commit aborted."
    exit 1
fi

echo "✅ All unit tests passed!"
exit 0
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Version History

**v1.22.3** (2026-01-08)
- Added comprehensive Troubleshooting section (7 common issues)
- Added CI/CD Integration section with ready-to-use examples
- Documented GitHub Actions, GitLab CI, Azure DevOps, and Jenkins configurations
- Added coverage threshold enforcement examples
- Added pre-commit hook template
- Added local CI simulation examples (Act and Docker)

**v1.16.11** (2026-01-05)
- Added 7 new unit tests (medium and low priority)
- Improved coverage from 20.9% to 24.6%
- Achieved 100% coverage on 15+ helper functions
- Total: 46 passing tests

---

**Last Updated:** 2026-01-08
**Total Tests:** 46
**Overall Coverage:** 24.6%
**Documentation Sections:** 18 (including Troubleshooting and CI/CD Integration)

                          ..ooOO END OOoo..



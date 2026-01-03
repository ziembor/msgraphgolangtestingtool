# Code Review & Improvement Suggestions

## 1. Security Improvements

### Prevent Command Injection
In `exportCertFromStore`, the `thumbprint` variable is directly interpolated into the PowerShell command string. While a thumbprint is typically a SHA1 hash, malicious input could theoretically execute arbitrary PowerShell commands.
**Recommendation:** Validate that the `thumbprint` string contains only hexadecimal characters before using it.

### Ensure Temporary File Cleanup
In `exportCertFromStore`, the temporary PFX file is explicitly removed only if `os.ReadFile` succeeds. If reading fails, the file remains on disk.
**Recommendation:** Use `defer os.Remove(tempFile)` immediately after the file path is defined or after the command execution to ensure it is always deleted, even on error.

### Secure String Handling
The code generates a random password for the PFX export. Ensure that this password and the PFX data are handled as securely as possible (e.g., minimizing their lifespan in memory, though Go's GC makes this hard to strictly enforce).

## 2. Reliability & Correctness

### Fix `log.Fatalf` preventing `defer` execution
The `main` function uses `log.Fatalf` for authentication or client initialization errors. `log.Fatalf` calls `os.Exit(1)`, which **skips** any deferred functions. This means `closeCSVLog()` will not be called, potentially leaving the CSV file buffer unflushed or the file handle open (though the OS will eventually close it, data might be lost).
**Recommendation:** Instead of `log.Fatalf`, use `log.Printf` followed by `return` or a strict error handling flow in `main` so that `defer` statements run.

### Fix CSV Logging Logic
The tool logs to `_msgraphgolangtestingtool_{date}.csv` in the temp directory.
1.  **Mixed Schema Issue:** If a user runs `-action getevents` and then `-action sendmail` on the same day, the tool appends rows with different column counts/meanings to the *same file*. This results in a corrupted/unreadable CSV.
    *   **Recommendation:** Include the action name in the filename (e.g., `_msgraphgolangtestingtool_{action}_{date}.csv`) OR use a generic header (Timestamp, Action, Details...) where "Details" is a JSON object or formatted string.
2.  **Concurrency/State:** `csvWriter` and `csvFile` are global variables. While this is a CLI tool, avoiding globals makes the code more testable and robust.

### Fix Typo in Error Message
In `getCredential`: `fmt.Errorf("no valid authentication method provided (use -secret, -pfx, or -thumbprint")` is missing a closing parenthesis.

## 3. Code Quality & Maintainability

### Refactor `main` function
The `main` function is becoming a "God function," handling flag parsing, validation, auth setup, and dispatching actions.
**Recommendation:**
*   Move action handlers (e.g., `handleSendMail`, `handleListEvents`) into separate functions that accept the `client` and specific flags.
*   Create a struct for the Configuration/Flags to pass them around cleanly.

### Context Cancellation
The tool uses `context.Background()` but does not handle OS signals (like Ctrl+C).
**Recommendation:** Use `signal.Notify` with `context.WithCancel` to gracefully handle interruptions, allowing ongoing network requests to potentially clean up or log "Cancelled".

### Remove Magic Strings
Strings like `"getevents"`, `"sendmail"`, `"Success"` are hardcoded in multiple places.
**Recommendation:** Define constants for these values (e.g., `const ActionSendMail = "sendmail"`).

### Improve Flag Parsing for Lists
Parsing comma-separated lists (`parseList`) is manual.
**Recommendation:** Define a custom `flag.Value` type (e.g., `stringSlice`) so `flag.Parse()` handles it automatically.

## 4. Example Refactoring (Snippet)

**Safer Temporary File Cleanup:**

```go
func exportCertFromStore(thumbprint string) ([]byte, string, error) {
    // ... validation ...
    
    tempDir := os.TempDir()
    tempFile := filepath.Join(tempDir, fmt.Sprintf("gemini_cert_%s.pfx", thumbprint))
    
    // Ensure cleanup happens no matter what
    defer os.Remove(tempFile) 

    // ... execute command ...
}
```

**Graceful Exit:**

```go
func main() {
    // ... setup ...
    initCSVLog(*action)
    defer closeCSVLog()

    if err := run(); err != nil {
        log.Println(err) // Log error
        // Defer will run now
        os.Exit(1)
    }
}

func run() error {
    // Logic here, returning errors instead of log.Fatalf
}
```

# Code Review & Improvement Plan: msgraphtool
**Date:** 2026-01-31
**Version Reviewed:** 1.25.0+ (Current Codebase)
**Reviewer:** Gemini CLI

## 1. Executive Summary

Since the last review on 2026-01-07, `msgraphtool` has evolved significantly. The **Modularization (Phase 1)** is fully complete and stable. The tool has transitioned from a basic testing utility to a feature-rich CLI with advanced capabilities like Availability Checking (`getschedule`), Email Search & Export (`searchandexport`), and detailed JSON output.

**Highlights:**
*   **Architecture:** The file-based modularity (`config.go`, `handlers.go`, `auth.go`) works perfectly. It strikes the right balance between structure and simplicity for a CLI tool.
*   **Security:** The implementation of `validateMessageID` to prevent OData injection is exemplary. Path traversal checks are robust.
*   **Features:** The addition of `getschedule` and `exportinbox` greatly expands the tool's utility for troubleshooting and data extraction.

**Primary Concern:**
*   **Testing Technical Debt:** While code logic was split, tests were not cleanly refactored. We now have two overlapping test files (`shared_test.go` and `msgraphgolangtestingtool_test.go`) containing duplicate logic and lacking clear ownership. This is the main focus for the next cleanup phase.

---

## 2. Code Quality & Architecture Review

### 2.1 Modularization Status
The separation of concerns is excellent:
*   **`msgraphtool.go`**: Clean entry point, handles signals and high-level flow.
*   **`config.go`**: Centralized configuration and validation.
*   **`auth.go`**: Isolated authentication logic.
*   **`handlers.go`**: Business logic. *Note: This file is growing large (approx. 500 lines) and covers disparate domains (Mail, Calendar, Search).*
*   **`utils.go`**: Generic helpers.

### 2.2 Security Review
*   **OData Injection:** The `validateMessageID` function in `utils.go` (or `msgraphgolangtestingtool_test.go` context) is a great defense-in-depth measure. It explicitly blocks quotes and OData operators.
*   **Path Traversal:** `validateFilePath` correctly uses `filepath.Clean` and checks for `..` to prevent writing exports outside the temp directory.
*   **Secrets:** Secrets are masked in verbose logs.

### 2.3 New Features Analysis
*   **`getschedule`**: Correctly calculates "next working day" logic (`addWorkingDays`). The 12:00-13:00 UTC check is a sensible default for testing availability.
*   **`exportinbox` / `searchandexport`**: The manual map construction in `exportMessageToJSON` ensures the output JSON is clean and predictable, avoiding the massive nesting often found in raw Graph SDK serializations.
*   **Templates**: `-body-template` implementation is simple and effective.

---

## 3. Testing Technical Debt

This is the only area requiring immediate attention.

**Current State:**
*   **`src/shared_test.go`**: A legacy file name containing tests for `config.go`, `utils.go`, and `handlers.go`.
*   **`src/msgraphgolangtestingtool_test.go`**: A massive (1000+ line) catch-all file that *also* tests `config.go`, `utils.go`, and helpers.

**Issues:**
1.  **Duplication:** Both files test `validateConfiguration` and `validateFilePath`.
2.  **Discoverability:** It is unclear where to add a new test.
3.  **Maintenance:** `shared_test.go` refers to a file that no longer exists (`shared.go`).

**Recommendation:**
Split these two files into domain-specific test files matching the source:
*   `src/config_test.go`: Move configuration tests here.
*   `src/utils_test.go`: Move `validateFilePath`, `maskGUID`, `retryWithBackoff` tests here.
*   `src/handlers_test.go`: Move `createFileAttachments` and logic tests here.
*   `src/auth_test.go`: Move PFX/Cert tests here.

---

## 4. Improvement Plan

### 4.1 Phase 1: Test Refactoring (High Priority)
*   **Goal:** Eliminate `shared_test.go` and `msgraphgolangtestingtool_test.go`.
*   **Action:** Create `config_test.go`, `utils_test.go`, `handlers_test.go`, `auth_test.go` and redistribute existing tests. Remove duplicates.

### 4.2 Phase 2: Logic Split (Medium Priority)
*   **Goal:** Prevent `handlers.go` from becoming the new "god object".
*   **Action:** Split `handlers.go` into:
    *   `handler_mail.go` (SendMail, GetInbox, Export)
    *   `handler_calendar.go` (GetEvents, SendInvite, GetSchedule)
    *   `handler_search.go` (SearchAndExport)

### 4.3 Phase 3: Usability (Low Priority)
*   **Interactive Wizard:** If no args are provided, guide the user.
*   **Dry Run (`-whatif`):** For `sendmail` and `sendinvite`, print what *would* happen without sending the request.

---

## 5. Actionable Next Steps (Commands)

1.  **Refactor Tests:** Run a series of moves to split the test files.
2.  **Verify:** Run `go test ./...` to ensure no coverage is lost.

```powershell
# Proposed File Structure after Refactoring
src/
├── msgraphtool.go
├── config.go
├── config_test.go       <-- New (from shared_test.go & msgraph...test.go)
├── auth.go
├── auth_test.go         <-- New
├── handlers.go
├── handlers_test.go     <-- New
├── utils.go
├── utils_test.go        <-- New
└── ...
```

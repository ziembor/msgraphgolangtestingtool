# Release & Versioning Guide

This document is the **definitive guide** for versioning and releasing the `msgraphgolangtestingtool` project.

## 1. Versioning Policy

### Single Source of Truth
The version is stored in **one place only**:
- File: `internal/common/version/version.go`
- Format: Go const string (e.g., `const Version = "2.0.2"`)

To update the version, edit the `Version` constant in `internal/common/version/version.go`. No external VERSION files are needed.

### Version Numbering
- **Format:** `x.y.z` (Semantic Versioning)
  - `x` (Major): Breaking changes, major architectural shifts, new tools/executables.
  - `y` (Minor): New features, significant enhancements.
  - `z` (Patch): Bug fixes, documentation updates.

### Changelog Format
Changelogs are stored as individual files in the `Changelog/` directory:
- **Location:** `Changelog/{version}.md` (e.g., `Changelog/1.16.2.md`)
- **Format:** [Keep a Changelog](https://keepachangelog.com) style.

## 2. Automated Release (Recommended)

The `run-integration-tests.ps1` script is the standard way to perform releases. It handles validation, file updates, and git operations automatically.

### Quick Start
```powershell
# From project root
.\run-integration-tests.ps1
```

### What the Script Does
1. **Safety Checks:** Validates git status and scans for potential secrets.
2. **Version Bump:** Prompts for new version and updates `internal/common/version/version.go`.
3. **Changelog:** Interactively creates `Changelog/{version}.md`.
4. **Commit:** Stages and commits changes with a standardized message.
5. **Tag & Push:** Creates a git tag (e.g., `v2.0.2`) and pushes it to trigger GitHub Actions.

## 3. Manual Release Process

If you cannot use the automation script, follow these steps to release manually.

### Step 1: Update Version
Update the version constant with the new number (e.g., `2.1.0`).

Edit `internal/common/version/version.go`:
```go
const Version = "2.1.0"  // Change this line
```

### Step 2: Create Changelog
Create a new file `Changelog/2.1.0.md`:
```markdown
## [2.1.0] - 2026-01-05

### Added
- New feature X

### Fixed
- Bug fix Y
```

### Step 3: Verify Build (Optional)
```powershell
.\build-all.ps1
.\msgraphgolangtestingtool.exe -version
# Should output: ... Version 2.1.0
.\smtptool.exe -version
# Should output: ... Version 2.1.0
```

### Step 4: Commit Changes
```powershell
git add internal/common/version/version.go Changelog/2.1.0.md
git commit -m "Release v2.1.0"
git push origin main
```

### Step 5: Create and Push Tag
**This is the trigger for the automated build pipeline.**
```powershell
git tag v2.1.0
git push origin v2.1.0
```

## 4. GitHub Actions Workflow

When a tag matching `v*` is pushed (e.g., `v2.1.0`), the `.github/workflows/build.yml` workflow triggers automatically.

**The Workflow:**
1. **Builds** the application for Windows, Linux, and macOS.
2. **Creates** a GitHub Release with the tag name.
3. **Packages** the binary and documentation (README, LICENSE) into ZIP files.
4. **Uploads** the ZIP files to the Release.

**Artifacts Produced:**
- `msgraphgolangtestingtool-windows.zip`
- `msgraphgolangtestingtool-linux.zip`
- `msgraphgolangtestingtool-macos.zip`

## 5. Troubleshooting & Rollback

### Build Fails in GitHub Actions
- Check the workflow logs: `gh run view --log-failed`
- Ensure `src/go.mod` matches the Go version in `build.yml` (currently 1.25+).

### Emergency Rollback
If a bad release is pushed:

1. **Delete Tag:**
   ```powershell
   git tag -d v1.16.2
   git push origin :refs/tags/v1.16.2
   ```
2. **Delete Release:**
   ```powershell
   gh release delete v1.16.2 --yes
   ```
3. **Revert Commit:**
   ```powershell
   git revert HEAD
   git push origin main
   ```

## For Future AI Assistants

To perform a release:
1. **Prefer `.\run-integration-tests.ps1`** if interactive mode is possible.
2. **If manual:**
   - Read `internal/common/version/version.go` to get current version.
   - Increment according to Semantic Versioning (major.minor.patch).
   - Update the `Version` const in `internal/common/version/version.go`.
   - Write `Changelog/{version}.md`.
   - Commit & Tag.
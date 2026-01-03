# Build Instructions

This document provides step-by-step instructions for building the `msgraphgolangtestingtool` executable.

## Prerequisites

### 1. Install Go

- Download and install Go 1.25 or later from [https://golang.org/dl/](https://golang.org/dl/)
- Verify installation:
  
  ```powershell
  go version
  ```
  
  Expected output: `go version go1.25.x windows/amd64`

### 2. Verify Git (Optional)

If you're cloning from a repository:

```powershell
git --version
```

## Build Steps

### 1. Navigate to Project Directory

```powershell
cd C:\Workspace\msgraphgolangtestingtool
```

### 2. Download Dependencies

Go will automatically download required dependencies during the build, but you can explicitly fetch them:

```powershell
cd src
go mod download
```

This downloads:

- `github.com/Azure/azure-sdk-for-go/sdk/azidentity`
- `github.com/microsoftgraph/msgraph-sdk-go`
- `golang.org/x/crypto/pkcs12`
- And all transitive dependencies

### 3. Build the Executable

**Note:** All Go source files are located in the `src/` subdirectory.

#### Standard Build (From Project Root)

Creates `msgraphgolangtestingtool.exe` in the project root:

```powershell
# Option 1: Build from project root using -C flag
go build -C src -o msgraphgolangtestingtool.exe

# Option 2: Build from src directory
cd src
go build -o ../msgraphgolangtestingtool.exe
```

#### Optimized Build (Smaller Binary)

Removes debug information and symbol table for a smaller executable:

```powershell
# From project root
go build -C src -ldflags="-s -w" -o msgraphgolangtestingtool.exe

# From src directory
cd src
go build -ldflags="-s -w" -o ../msgraphgolangtestingtool.exe
```

#### Build with Version Information

The version is now read from the `VERSION` file and hardcoded in `src/msgraphgolangtestingtool.go`:

```powershell
# From project root
go build -C src -ldflags="-s -w" -o msgraphgolangtestingtool.exe

# From src directory
cd src
go build -ldflags="-s -w" -o ../msgraphgolangtestingtool.exe
```

### 4. Verify the Build

Check that the executable was created:

```powershell
dir msgraphgolangtestingtool.exe
```

Test the executable:

```powershell
.\msgraphgolangtestingtool.exe -h
```

## Build Output

- **Output file**: `msgraphgolangtestingtool.exe`
- **Typical size**: 15-25 MB (standard build), 10-18 MB (optimized)
- **Architecture**: Windows AMD64 (64-bit)

## Cross-Platform Builds (Advanced)

The tool is cross-platform and can be built for Windows, Linux, and macOS. However, the Windows Certificate Store authentication (`-thumbprint`) is only available on Windows.

### Build for Linux

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -C src -o msgraphgolangtestingtool
```

**Note**: Windows Certificate Store authentication (`-thumbprint`) will not work on Linux. The tool will only support `-secret` and `-pfx` authentication methods.

### Build for macOS

```powershell
$env:GOOS="darwin"
$env:GOARCH="amd64"
go build -C src -o msgraphgolangtestingtool
```

**Note**: Windows Certificate Store authentication (`-thumbprint`) will not work on macOS. The tool will only support `-secret` and `-pfx` authentication methods.

### Reset Environment Variables

After cross-compilation, reset the environment:

```powershell
Remove-Item Env:\GOOS
Remove-Item Env:\GOARCH
```

## Troubleshooting

### "go: command not found"

- Go is not installed or not in your PATH
- Reinstall Go and ensure it's added to system PATH

### "package X is not in GOROOT"

- Run `go mod tidy` to clean up dependencies
- Run `go mod download` to re-download dependencies

### Build Takes Too Long

- First build downloads all dependencies (can take 1-3 minutes)
- Subsequent builds are much faster (cached dependencies)

### "Access Denied" When Building

- Close any running instances of `msgraphgolangtestingtool.exe`
- Run PowerShell as Administrator

## Development Builds

For development and testing, you can run without building:

```powershell
# From src directory
cd src
go run . -tenantid "xxx" -clientid "xxx" -secret "xxx" -mailbox "user@example.com" -action getevents
```

## Clean Build

To remove build artifacts and start fresh:

```powershell
# Remove the executable
Remove-Item msgraphgolangtestingtool.exe -ErrorAction SilentlyContinue

# Clean Go build cache
cd src
go clean -cache

# Re-download dependencies
go mod download

# Rebuild
go build -o ../msgraphgolangtestingtool.exe
```

## Automated Build Script

Create a `build.ps1` file for automated builds:

```powershell
# build.ps1
Write-Host "Building msgraphgolangtestingtool..." -ForegroundColor Green

# Clean previous build
Remove-Item msgraphgolangtestingtool.exe -ErrorAction SilentlyContinue

# Build with optimization from src directory
go build -C src -ldflags="-s -w" -o msgraphgolangtestingtool.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    $size = (Get-Item msgraphgolangtestingtool.exe).Length / 1MB
    Write-Host ("Binary size: {0:N2} MB" -f $size) -ForegroundColor Cyan

    # Display version
    $version = Get-Content VERSION
    Write-Host ("Version: {0}" -f $version) -ForegroundColor Cyan
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
```

Run the script:

```powershell
.\build.ps1
```

## Next Steps

After building, see the README.md and CLAUDE.md files for usage instructions.

---

## Release Process

When you're ready to submit changes to the main branch and create a new release with GitHub Actions:

### Step 1: Verify All Version Files Match

```powershell
# Check that all version references are consistent
cat VERSION
grep "const version" src/msgraphgolangtestingtool.go
head -20 CHANGELOG.md
```

All three should show the same version number (e.g., `1.12.6`).

### Step 2: Check Current Status

```powershell
git status
```

### Step 3: Stage All Changes

```powershell
git add .
```

### Step 4: Commit Changes

```powershell
git commit -m "$(cat <<'EOF'
Release v1.12.6 - Critical security fixes and code improvements

### Fixed
- CRITICAL: Fixed log.Fatalf preventing deferred cleanup (CSV data loss)
- Refactored main() to use run() pattern

### Security
- CRITICAL: Added thumbprint validation (40 hex chars)
- Added isHexString() helper function

### Changed
- Replaced magic strings with constants
- ActionGetEvents, ActionSendMail, ActionSendInvite, ActionGetInbox
- StatusSuccess, StatusError

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Step 5: Push Current Branch to Remote

```powershell
# Replace 'current-branch-name' with your actual branch name
git push origin v1.12.2
```

### Step 6: Merge to Main

**Option A: Direct merge (if you have permissions)**

```powershell
git checkout main
git merge v1.12.2
git push origin main
```

**Option B: Create Pull Request**

```powershell
gh pr create --title "Release v1.12.6" --body "$(cat <<'EOF'
## Summary
Critical security fixes and code quality improvements for v1.12.6

## Changes
- **CRITICAL**: Fixed log.Fatalf preventing deferred cleanup
- **CRITICAL**: Added thumbprint validation for security
- Replaced magic strings with constants throughout codebase

## Test plan
- [x] Code builds successfully
- [x] Thumbprint validation tested with valid/invalid inputs
- [x] All improvements documented in CHANGELOG.md

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### Step 7: Create and Push Git Tag (This triggers GitHub Actions!)

```powershell
# Create tag matching the VERSION file
git tag v1.12.6
git push origin v1.12.6
```

**IMPORTANT:** Pushing the tag triggers `.github/workflows/build.yml` which will:
- Build the Windows executable from the `src/` directory
- Create a GitHub Release
- Attach the compiled binary (`msgraphgolangtestingtool.exe`) to the release
- Generate release notes automatically from CHANGELOG.md

### Step 8: Verify GitHub Actions Workflow

```powershell
# List recent workflow runs
gh run list --limit 5

# Watch the current run in real-time
gh run watch
```

You can also view the workflow in your browser:
```powershell
# Open the GitHub Actions page
gh browse --repo <your-repo> actions
```

### What Happens During the Automated Build

The GitHub Actions workflow (`.github/workflows/build.yml`) performs these steps:

1. **Checkout code** - Pulls the tagged version
2. **Setup Go** - Installs Go 1.25+
3. **Download dependencies** - Runs `go mod download` in `src/`
4. **Build executable** - Runs `go build -C src -o msgraphgolangtestingtool.exe`
5. **Verify build** - Checks that the executable was created and displays version
6. **Create release** - Creates a GitHub release with tag name
7. **Upload artifact** - Attaches `msgraphgolangtestingtool.exe` to the release

### Key Points

- ✅ **Tag triggers build**: The GitHub Actions workflow is triggered by pushing a tag matching `v*` pattern
- ✅ **Version consistency**: VERSION file, source code constant (`const version`), CHANGELOG.md, and git tag must all match
- ✅ **Workflow permissions**: The workflow has `contents: write` permission to create releases
- ✅ **Automatic release**: The build artifact will be automatically attached to the GitHub release
- ✅ **Build from src**: The workflow uses `go build -C src -o msgraphgolangtestingtool.exe`

### Troubleshooting Release Issues

**Workflow not triggering?**
- Verify you pushed the tag: `git push origin v1.12.6` (not just `git push`)
- Check tag format matches `v*` pattern
- Verify `.github/workflows/build.yml` exists and has `on: push: tags: ['v*']`

**Build fails in GitHub Actions?**
- Check Go version compatibility (requires Go 1.25+)
- Verify all dependencies are in `src/go.mod`
- Review workflow logs: `gh run view --log-failed`

**Release created but no binary attached?**
- Check that the build step succeeded
- Verify the upload step has correct path to `msgraphgolangtestingtool.exe`
- Ensure `contents: write` permission is set in workflow

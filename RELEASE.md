# Release Process Documentation

This document explains how to create a new release of the Microsoft Graph EXO Mails/Calendar Golang Testing Tool using the interactive release script.

## Quick Start

```powershell
# From project root
.\release.ps1
```

The script will guide you through the entire release process interactively.

## Prerequisites

### Required
- **Git** - For version control operations
- **PowerShell 5.1+** - Script execution environment
- **Write access** to the repository

### Optional
- **GitHub CLI (`gh`)** - For PR creation and workflow monitoring
  - Install: https://cli.github.com/
  - Not required, but enables additional features

## Release Script Overview

The `release.ps1` script is an **interactive PowerShell tool** that automates the release process while giving you full control at each step.

### What It Does

1. ✅ Validates current git status
2. ✅ **Scans for non-sanitized secrets (NEW!)**
3. ✅ Prompts for new version (enforces 1.x.y format)
4. ✅ Updates `src/VERSION` file
5. ✅ Creates/updates changelog entry
6. ✅ Commits changes with formatted message
7. ✅ Pushes to remote branch
8. ✅ Creates and pushes git tag (triggers GitHub Actions)
9. ✅ Optionally creates Pull Request
10. ✅ Optionally monitors GitHub Actions workflow
11. ✅ Bumps version for next development cycle

### Version Requirements

**IMPORTANT:** The major version is **locked at 1**. All releases must follow the `1.x.y` format.

- ✅ **Valid**: `1.16.2`, `1.17.0`, `1.20.5`
- ❌ **Invalid**: `2.0.0`, `0.1.0`, `1.16`

The script enforces this and will reject invalid version formats.

## Step-by-Step Process

### Step 1: Check Git Status

The script checks for uncommitted changes and warns you if any exist.

```
========================================
Step 1: Checking Git Status
========================================

⚠ You have uncommitted changes:
 M IMPROVEMENTS.md
?? GEMINI.md

Do you want to continue anyway? (y/N):
```

**Best Practice:** Commit or stash changes before running the release script.

### Step 2: Security Scan for Secrets

**NEW!** The script automatically scans for non-sanitized secrets before proceeding with the release.

```

Shows current version from `src/VERSION` and suggests next versions.

```
========================================
Step 2: Version Information
========================================

Current version: 1.16.1

Suggested versions:
  1. Patch release (bug fixes):    1.16.2
  2. Minor release (new features): 1.17.0

Enter new version (must be 1.x.y format, or press Enter for patch: 1.16.2):
```

**Options:**
- Press **Enter** for suggested patch version
- Type `1.17.0` for minor version
- Type any valid `1.x.y` version

**Version Types:**
- **Patch (1.x.Y)**: Bug fixes, documentation updates, minor improvements
- **Minor (1.X.0)**: New features, significant changes

### Step 4: Update Version File

Updates `src/VERSION` with the new version.

```
========================================
Step 3: Updating Version File
========================================

Updating src\VERSION...
✓ Updated src\VERSION to 1.16.2

Note: Version is automatically embedded into the binary at compile time via go:embed directive
      (No need to update source code - it reads from VERSION file)
```

**How It Works:**
- The Go source code uses `//go:embed VERSION` directive
- Version is read from `src/VERSION` at compile time
- No manual code changes needed

### Step 5: Changelog Entry

Interactive changelog creation with prompts for each section.

```
========================================
Step 4: Changelog Entry
========================================

Creating new changelog entry at Changelog\1.16.2.md

Enter changelog entries (press Enter on empty line to finish each section)

[Added] New features:
  - New getinbox action for listing inbox messages
  -

[Changed] Changes to existing functionality:
  - Changed default action from getevents to getinbox
  - Updated all documentation
  -

[Fixed] Bug fixes:
  -

[Security] Security improvements:
  -
```

**Changelog Sections:**
- **Added**: New features
- **Changed**: Changes to existing functionality
- **Fixed**: Bug fixes
- **Security**: Security improvements

**Tips:**
- Press **Enter** on empty line to skip a section
- Entries are automatically prefixed with `-`
- If changelog exists, you can edit it in Notepad

**Changelog Format:**
```markdown
## [1.16.2] 2026-01-04

### Changed

- Changed default action from getevents to getinbox
- Updated all documentation

### Fixed

- Fixed CSV logging issue
```

### Step 6: Review Changes

Shows summary of all changes before committing.

```
========================================
Step 5: Review Changes
========================================

The following changes will be made:
  • src\VERSION: 1.16.1 → 1.16.2 (embedded via go:embed)
  • Changelog\1.16.2.md: Created

Commit message:
------------------------
Release v1.16.2 - Patch Release

## [1.16.2] 2026-01-04

### Changed

- Changed default action from getevents to getinbox
- Updated all documentation

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
------------------------

Do you want to proceed with these changes? (y/N):
```

**Review Carefully:**
- Verify version number is correct
- Check changelog entries are accurate
- Confirm commit message looks good

### Step 7: Commit Changes

Creates a git commit with formatted message.

```
========================================
Step 6: Committing Changes
========================================

Staging changes...

Files to be committed:
 M src\VERSION
 A Changelog\1.16.2.md

Creating commit...
✓ Committed changes
Current branch: main
```

**What Gets Committed:**
- `src/VERSION` - Updated version
- `Changelog/{version}.md` - Changelog entry

### Step 8: Push to Remote

Optional push to remote repository.

```
========================================
Step 7: Push to Remote
========================================

Push commit to remote branch 'main'? (Y/n): y

Pushing to origin/main...
✓ Pushed to origin/main
```

**Options:**
- Press **Enter** or **y** to push
- Type **n** to skip (you can push manually later)

### Step 9: Create Git Tag (Triggers GitHub Actions)

**⚡ IMPORTANT: This step triggers the automated build!**

```
========================================
Step 8: Create Git Tag (Triggers GitHub Actions)
========================================

⚠ Creating and pushing tag 'v1.16.2' will trigger GitHub Actions workflow!
This will:
  • Build Windows and Linux binaries
  • Create GitHub Release
  • Attach ZIP files with binaries to the release

Create and push tag 'v1.16.2'? (y/N): y

Creating tag v1.16.2...
Pushing tag v1.16.2...
✓ Tag pushed successfully
✓ GitHub Actions workflow should start building now!
```

**What Happens:**
1. Tag `v1.16.2` is created locally
2. Tag is pushed to `origin`
3. GitHub Actions detects the tag push
4. Workflow `.github/workflows/build.yml` triggers
5. Binaries are built for Windows and Linux
6. GitHub Release is created with tag name
7. ZIP files are uploaded to the release

**GitHub Actions Workflow:**
- **Builds**: Windows (`msgraphgolangtestingtool.exe`), Linux (`msgraphgolangtestingtool`)
- **Creates**: ZIP files with binaries + EXAMPLES.md + LICENSE + README.md
- **Publishes**: GitHub Release with automatic release notes

### Step 10: Pull Request (Optional)

If you're on a feature branch, optionally create a PR.

```
========================================
Step 9: Pull Request (Optional)
========================================

Create Pull Request to merge 'feature-branch' into 'main'? (y/N): y

Creating Pull Request...
✓ Pull Request created

Merge PR immediately? (y/N): n
```

**When to Use:**
- If releasing from a feature branch
- Skip if already on `main` branch

**Requires:**
- GitHub CLI (`gh`) installed and authenticated

### Step 11: Monitor GitHub Actions

Optional workflow monitoring.

```
========================================
Step 10: Monitor GitHub Actions (Optional)
========================================

Monitor GitHub Actions workflow? (y/N): y

Recent workflow runs:
  ✓ Build and Release  v1.16.2  main  2m ago

To watch the current run:
  gh run watch
```

**Monitoring Options:**
- **List runs**: `gh run list --limit 5`
- **Watch live**: `gh run watch`
- **View in browser**: GitHub Actions tab

## Manual Tag Creation (If Needed)

If you skipped Step 8, you can create the tag manually:

```powershell
# Create tag locally
git tag v1.16.2

# Push tag to trigger GitHub Actions
git push origin v1.16.2
```

## Verification Steps

After the release script completes:

### 1. Verify Local Changes

```powershell
# Check version file
cat src\VERSION
# Output: 1.16.2

# Check changelog
cat Changelog\1.16.2.md

# Check git log
git log -1 --oneline
# Output: abc1234 Release v1.16.2 - Patch Release

# Check tags
git tag -l "v1.16*"
# Output: v1.16.1, v1.16.2
```

### 2. Verify GitHub Actions

```powershell
# List recent workflow runs
gh run list --limit 5

# Watch the current run (blocks until complete)
gh run watch

# View run in browser
gh browse --repo ziembor/msgraphgolangtestingtool actions
```

### 3. Verify GitHub Release

```powershell
# View latest release
gh release view

# Open releases page in browser
gh browse --repo ziembor/msgraphgolangtestingtool releases
```

**Expected Release Contents:**
- Release title: `v1.16.2`
- Release notes with changelog
- Attached files:
  - `msgraphgolangtestingtool-windows.zip` (Windows binary + docs)
  - `msgraphgolangtestingtool-linux.zip` (Linux binary + docs)

### 4. Test Downloaded Binary

```powershell
# Download Windows ZIP from release
Invoke-WebRequest -Uri "https://github.com/ziembor/msgraphgolangtestingtool/releases/download/v1.16.2/msgraphgolangtestingtool-windows.zip" -OutFile "release-test.zip"

# Extract
Expand-Archive -Path release-test.zip -DestinationPath release-test

# Verify version
.\release-test\msgraphgolangtestingtool.exe -version
# Output: Microsoft Graph EXO Mails/Calendar Golang Testing Tool - Version 1.16.2
```

## Troubleshooting

### Issue: "Git commit failed"

**Cause:** Nothing to commit (files already staged and committed)

**Solution:**
```powershell
# Check status
git status

# If files already committed, just create tag
git tag v1.16.2
git push origin v1.16.2
```

### Issue: "Failed to push tag"

**Cause:** Network issues or authentication problems

**Solution:**
```powershell
# Try manual push
git push origin v1.16.2

# If authentication fails, check credentials
gh auth status

# Re-authenticate if needed
gh auth login
```

### Issue: "GitHub Actions not triggering"

**Cause:** Tag format doesn't match workflow trigger pattern

**Solution:**
```powershell
# Check tag format (must be v*.*.*)
git tag -l
# Should show: v1.16.2

# Check workflow file
cat .github\workflows\build.yml
# Should have: on: push: tags: ['v*']

# Verify tag was pushed to remote
git ls-remote --tags origin
# Should show: refs/tags/v1.16.2
```

### Issue: "Build fails in GitHub Actions"

**Cause:** Build dependencies or Go version mismatch

**Solution:**
```powershell
# View workflow logs
gh run view --log-failed

# Check Go version requirement
cat .github\workflows\build.yml
# go-version: '1.21'

# Test build locally first
cd src
go build -o ../msgraphgolangtestingtool.exe
```

### Issue: "Changelog already exists"

**Cause:** Version was already released or changelog created manually

**Solution:**
- Script will prompt to edit existing changelog in Notepad
- Review and update as needed
- If version should be incremented, cancel and choose a higher version

### Issue: "PR creation failed"

**Cause:** GitHub CLI not installed or not authenticated

**Solution:**
```powershell
# Install GitHub CLI
winget install GitHub.cli

# Authenticate
gh auth login

# Or create PR manually
gh pr create --title "Release v1.16.2" --body "Release notes..."
```

## Best Practices

### Before Running Script

1. **Review Changes:**
   ```powershell
   git status
   git diff
   ```

2. **Update Documentation:**
   - Update README.md if features changed
   - Update CLAUDE.md if architecture changed
   - Update EXAMPLES.md if usage changed

3. **Test Locally:**
   ```powershell
   cd src
   go build -o ../msgraphgolangtestingtool.exe
   ..\msgraphgolangtestingtool.exe -version
   ```

### During Script Execution

1. **Choose Appropriate Version:**
   - Patch (1.x.Y): Bug fixes, docs, small improvements
   - Minor (1.X.0): New features, breaking changes

2. **Write Clear Changelog:**
   - Be specific about changes
   - Include user-facing changes
   - Mention breaking changes clearly

3. **Review Before Confirming:**
   - Double-check version number
   - Verify changelog entries
   - Check commit message

### After Release

1. **Monitor Build:**
   ```powershell
   gh run watch
   ```

2. **Test Release:**
   - Download ZIP from GitHub Release
   - Extract and test binary
   - Verify version: `.\msgraphgolangtestingtool.exe -version`

3. **Update Branch:**
   ```powershell
   # If you created PR, merge and update local
   git checkout main
   git pull origin main
   ```

## Version History Format

The project maintains version history in two places:

1. **`src/VERSION`** - Single line with version number
   ```
   1.16.2
   ```

2. **`Changelog/{version}.md`** - Detailed changelog
   ```markdown
   ## [1.16.2] 2026-01-04

   ### Changed
   - Changed default action from getevents to getinbox
   - Updated documentation

   ### Fixed
   - Fixed CSV logging bug
   ```

## GitHub Actions Workflow

The workflow (`.github/workflows/build.yml`) is triggered by tags matching `v*`:

```yaml
on:
  push:
    tags:
      - 'v*'
```

**Build Matrix:**
- **Windows**: `msgraphgolangtestingtool.exe`
- **Linux**: `msgraphgolangtestingtool`

**Output Files:**
- `msgraphgolangtestingtool-windows.zip`
- `msgraphgolangtestingtool-linux.zip`

Each ZIP contains:
- Binary executable
- EXAMPLES.md
- LICENSE
- README.md

## Emergency Rollback

If a release has issues:

### 1. Delete Tag Locally and Remotely

```powershell
# Delete local tag
git tag -d v1.16.2

# Delete remote tag
git push origin :refs/tags/v1.16.2
```

### 2. Delete GitHub Release

```powershell
# Delete release
gh release delete v1.16.2 --yes

# Or delete via web UI
gh browse --repo ziembor/msgraphgolangtestingtool releases
```

### 3. Revert Version File

```powershell
# Revert to previous version
echo "1.16.1" | Out-File -NoNewline src\VERSION

# Commit
git add src\VERSION
git commit -m "Revert version to 1.16.1"
git push origin main
```

### 4. Create New Release

Run `.\release.ps1` again with corrected version.

## Script Features

### Safety Features
- ✅ **Secret detection** (scans for non-sanitized credentials)
- ✅ Version format validation (enforces 1.x.y)
- ✅ Version increment validation (must be greater than current)
- ✅ Git status check (warns about uncommitted changes)
- ✅ Confirmation prompts at each major step
- ✅ Rollback on cancel (restores VERSION file)

### User Experience
- ✅ Color-coded output (Green=success, Yellow=warning, Red=error)
- ✅ Clear step headers
- ✅ Suggested versions
- ✅ Default values (press Enter for patch version)
- ✅ Progress indicators

### Integration
- ✅ Git automation
- ✅ GitHub CLI integration (optional)
- ✅ GitHub Actions trigger
- ✅ Changelog management

## Command Reference

### Release Script
```powershell
# Run interactive release
.\release.ps1

# Get help
Get-Help .\release.ps1 -Full
```

### Manual Commands
```powershell
# Update version manually
echo "1.16.2" | Out-File -NoNewline src\VERSION

# Create changelog manually
New-Item -Path "Changelog\1.16.2.md" -ItemType File

# Commit manually
git add src\VERSION Changelog\1.16.2.md
git commit -m "Release v1.16.2"

# Tag manually
git tag v1.16.2
git push origin v1.16.2

# Monitor workflow
gh run list
gh run watch
gh run view --log
```

### Verification Commands
```powershell
# Check current version
cat src\VERSION

# List tags
git tag -l

# View latest release
gh release view

# List workflow runs
gh run list --limit 5
```

## Support

For issues with the release script:
1. Check this documentation
2. Review BUILD.md for release process details
3. Check CLAUDE.md for architecture information
4. Review `.github/workflows/build.yml` for workflow details

## Related Documentation

- **BUILD.md** - Build instructions and release process
- **CLAUDE.md** - Architecture and project instructions for AI
- **README.md** - User-facing documentation
- **EXAMPLES.md** - Usage examples

---

**Last Updated:** 2026-01-05
**Script Version:** 2.0 (added security scanning)
**Compatibility:** PowerShell 5.1+, Windows 10+

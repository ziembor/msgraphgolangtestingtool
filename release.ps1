# Interactive Release Script for Microsoft Graph GoLang Testing Tool
# This script guides you through creating a new release that triggers GitHub Actions
#
# QUICK REFERENCE:
#   Usage:      .\release.ps1
#   Version:    Updates src/VERSION only (auto-embedded via go:embed)
#   Changelog:  Creates Changelog/{version}.md with interactive prompts
#   Git Tag:    Creates v{version} tag - TRIGGERS GITHUB ACTIONS
#   Output:     Windows & Linux binaries + GitHub Release with ZIPs
#
# Full documentation: See RELEASE.md

<#
.SYNOPSIS
    Interactive release script for msgraphgolangtestingtool

.DESCRIPTION
    This script helps you:
    1. Update version in src/VERSION (auto-embedded into binary via go:embed)
    2. Create/update changelog entry in Changelog/{version}.md
    3. Commit changes to git with formatted message
    4. Create and push git tag to trigger GitHub Actions build
    5. Optionally create a Pull Request
    6. Optionally monitor GitHub Actions workflow

    IMPORTANT: Only updates src/VERSION - version is embedded at compile time.

.EXAMPLE
    .\release.ps1

    Runs the interactive release process

.EXAMPLE
    Get-Help .\release.ps1 -Full

    Shows complete help documentation

.NOTES
    - Major version is LOCKED at 1 (only minor/patch updates allowed)
    - Version format: 1.x.y (e.g., 1.16.2, 1.17.0)
    - Version file: src/VERSION (NOT project root)
    - Requires: Git (required), GitHub CLI (optional for PR creation)
    - Pushing tag triggers: .github/workflows/build.yml
    - Builds: Windows (msgraphgolangtestingtool.exe) and Linux (msgraphgolangtestingtool)
    - Documentation: See RELEASE.md for complete documentation
#>

[CmdletBinding()]
param()

# Color functions
function Write-Info { param($Message) Write-Host $Message -ForegroundColor Cyan }
function Write-Success { param($Message) Write-Host "✓ $Message" -ForegroundColor Green }
function Write-Warning { param($Message) Write-Host "⚠ $Message" -ForegroundColor Yellow }
function Write-Error { param($Message) Write-Host "✗ $Message" -ForegroundColor Red }
function Write-Header { param($Message) Write-Host "`n========================================" -ForegroundColor Magenta; Write-Host $Message -ForegroundColor Magenta; Write-Host "========================================`n" -ForegroundColor Magenta }

# Check if we're in the right directory
if (-not (Test-Path "src\VERSION")) {
    Write-Error "This script must be run from the project root directory"
    Write-Error "Current directory: $(Get-Location)"
    exit 1
}

# Check git status
Write-Header "Step 1: Checking Git Status"
$gitStatus = git status --porcelain
if ($gitStatus) {
    Write-Warning "You have uncommitted changes:"
    git status --short
    $continue = Read-Host "`nDo you want to continue anyway? (y/N)"
    if ($continue -ne 'y' -and $continue -ne 'Y') {
        Write-Info "Release cancelled. Please commit or stash your changes first."
        exit 0
    }
}
Write-Success "Git status checked"

# Read current version
Write-Header "Step 2: Version Information"
$currentVersion = (Get-Content "src\VERSION" -Raw).Trim()
Write-Info "Current version: $currentVersion"

# Validate current version format (must be 1.x.y)
if ($currentVersion -notmatch '^1\.\d+\.\d+$') {
    Write-Error "Current version '$currentVersion' is invalid. Must be in format 1.x.y"
    exit 1
}

# Parse current version
$versionParts = $currentVersion -split '\.'
$currentMajor = [int]$versionParts[0]
$currentMinor = [int]$versionParts[1]
$currentPatch = [int]$versionParts[2]

# Suggest next versions
Write-Info "`nSuggested versions:"
Write-Host "  1. Patch release (bug fixes):    1.$currentMinor.$($currentPatch + 1)"
Write-Host "  2. Minor release (new features): 1.$($currentMinor + 1).0"

# Prompt for new version
$newVersion = $null
while ($true) {
    $input = Read-Host "`nEnter new version (must be 1.x.y format, or press Enter for patch: 1.$currentMinor.$($currentPatch + 1))"

    if ([string]::IsNullOrWhiteSpace($input)) {
        # Default to patch version
        $newVersion = "1.$currentMinor.$($currentPatch + 1)"
        break
    }

    # Validate format
    if ($input -notmatch '^1\.\d+\.\d+$') {
        Write-Warning "Invalid format. Must be 1.x.y (major version locked at 1)"
        continue
    }

    # Check if version is greater than current
    $newParts = $input -split '\.'
    $newMinor = [int]$newParts[1]
    $newPatch = [int]$newParts[2]

    if ($newMinor -lt $currentMinor -or ($newMinor -eq $currentMinor -and $newPatch -le $currentPatch)) {
        Write-Warning "New version must be greater than current version ($currentVersion)"
        continue
    }

    $newVersion = $input
    break
}

Write-Success "New version: $newVersion"

# Determine release type
$releaseType = if ($currentMinor -eq $newMinor) { "Patch" } else { "Minor" }
Write-Info "Release type: $releaseType"

# Update src/VERSION file
Write-Header "Step 3: Updating Version File"
Write-Info "Updating src\VERSION..."
Set-Content -Path "src\VERSION" -Value $newVersion -NoNewline
Write-Success "Updated src\VERSION to $newVersion"

Write-Info "`nNote: Version is automatically embedded into the binary at compile time via go:embed directive"
Write-Info "      (No need to update source code - it reads from VERSION file)"

# Create/update changelog entry
Write-Header "Step 4: Changelog Entry"
$changelogPath = "CHANGELOG.md"
$changelogEntryPath = "Changelog\$newVersion.md"

# Create Changelog directory if it doesn't exist
if (-not (Test-Path "Changelog")) {
    New-Item -ItemType Directory -Path "Changelog" | Out-Null
    Write-Info "Created Changelog directory"
}

# Check if changelog entry already exists
if (Test-Path $changelogEntryPath) {
    Write-Info "Changelog entry already exists at $changelogEntryPath"
    $existingChangelog = Get-Content $changelogEntryPath -Raw
    Write-Host "`nCurrent changelog entry:"
    Write-Host "------------------------"
    Write-Host $existingChangelog
    Write-Host "------------------------"

    $editChangelog = Read-Host "`nDo you want to edit this changelog entry? (y/N)"
    if ($editChangelog -eq 'y' -or $editChangelog -eq 'Y') {
        notepad.exe $changelogEntryPath
        Write-Info "Waiting for notepad to close..."
        Start-Sleep -Seconds 1
    }
} else {
    Write-Info "Creating new changelog entry at $changelogEntryPath"

    # Prompt for changelog sections
    Write-Info "`nEnter changelog entries (press Enter on empty line to finish each section)"

    # Added
    Write-Host "`n[Added] New features:"
    $added = @()
    while ($true) {
        $line = Read-Host "  -"
        if ([string]::IsNullOrWhiteSpace($line)) { break }
        $added += "- $line"
    }

    # Changed
    Write-Host "`n[Changed] Changes to existing functionality:"
    $changed = @()
    while ($true) {
        $line = Read-Host "  -"
        if ([string]::IsNullOrWhiteSpace($line)) { break }
        $changed += "- $line"
    }

    # Fixed
    Write-Host "`n[Fixed] Bug fixes:"
    $fixed = @()
    while ($true) {
        $line = Read-Host "  -"
        if ([string]::IsNullOrWhiteSpace($line)) { break }
        $fixed += "- $line"
    }

    # Security
    Write-Host "`n[Security] Security improvements:"
    $security = @()
    while ($true) {
        $line = Read-Host "  -"
        if ([string]::IsNullOrWhiteSpace($line)) { break }
        $security += "- $line"
    }

    # Build changelog content
    $changelogContent = @"
## [$newVersion] $(Get-Date -Format 'yyyy-MM-dd')

"@

    if ($added.Count -gt 0) {
        $changelogContent += "`n### Added`n`n"
        $changelogContent += ($added -join "`n") + "`n"
    }

    if ($changed.Count -gt 0) {
        $changelogContent += "`n### Changed`n`n"
        $changelogContent += ($changed -join "`n") + "`n"
    }

    if ($fixed.Count -gt 0) {
        $changelogContent += "`n### Fixed`n`n"
        $changelogContent += ($fixed -join "`n") + "`n"
    }

    if ($security.Count -gt 0) {
        $changelogContent += "`n### Security`n`n"
        $changelogContent += ($security -join "`n") + "`n"
    }

    # If no entries, add a default message
    if ($added.Count -eq 0 -and $changed.Count -eq 0 -and $fixed.Count -eq 0 -and $security.Count -eq 0) {
        $changelogContent += "`n### Changed`n`n- Updated default action from getevents to getinbox`n"
    }

    # Save changelog entry
    Set-Content -Path $changelogEntryPath -Value $changelogContent
    Write-Success "Created $changelogEntryPath"

    Write-Host "`nChangelog entry:"
    Write-Host "------------------------"
    Write-Host $changelogContent
    Write-Host "------------------------"
}

# Read final changelog for commit message
$changelogForCommit = Get-Content $changelogEntryPath -Raw

# Build commit message
$commitMessage = @"
Release v$newVersion - $releaseType Release

$changelogForCommit

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
"@

# Show summary
Write-Header "Step 5: Review Changes"
Write-Info "The following changes will be made:"
Write-Host "  • src\VERSION: $currentVersion → $newVersion (embedded via go:embed)"
Write-Host "  • ${changelogEntryPath}: $(if (Test-Path $changelogEntryPath) { "Updated" } else { "Created" })"
Write-Host "`nCommit message:"
Write-Host "------------------------"
Write-Host $commitMessage
Write-Host "------------------------"

# Confirm
$confirm = Read-Host "`nDo you want to proceed with these changes? (y/N)"
if ($confirm -ne 'y' -and $confirm -ne 'Y') {
    Write-Warning "Release cancelled"
    # Restore VERSION file
    Set-Content -Path "src\VERSION" -Value $currentVersion -NoNewline
    Write-Info "Restored src\VERSION to $currentVersion"
    exit 0
}

# Stage changes
Write-Header "Step 6: Committing Changes"
Write-Info "Staging changes..."
git add src\VERSION $changelogEntryPath

# Show what will be committed
Write-Info "`nFiles to be committed:"
git status --short

# Commit
Write-Info "`nCreating commit..."
git commit -m $commitMessage

if ($LASTEXITCODE -ne 0) {
    Write-Error "Git commit failed"
    exit 1
}
Write-Success "Committed changes"

# Get current branch
$currentBranch = git rev-parse --abbrev-ref HEAD
Write-Info "Current branch: $currentBranch"

# Ask about pushing
Write-Header "Step 7: Push to Remote"
$pushBranch = Read-Host "Push commit to remote branch '$currentBranch'? (Y/n)"
if ($pushBranch -ne 'n' -and $pushBranch -ne 'N') {
    Write-Info "Pushing to origin/$currentBranch..."
    git push origin $currentBranch

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Git push failed"
        Write-Warning "You may need to push manually: git push origin $currentBranch"
    } else {
        Write-Success "Pushed to origin/$currentBranch"
    }
}

# Create and push tag (THIS TRIGGERS GITHUB ACTIONS)
Write-Header "Step 8: Create Git Tag (Triggers GitHub Actions)"
$tagName = "v$newVersion"
Write-Warning "Creating and pushing tag '$tagName' will trigger GitHub Actions workflow!"
Write-Info "This will:"
Write-Host "  • Build Windows and Linux binaries"
Write-Host "  • Create GitHub Release"
Write-Host "  • Attach ZIP files with binaries to the release"

$createTag = Read-Host "`nCreate and push tag '$tagName'? (y/N)"
if ($createTag -eq 'y' -or $createTag -eq 'Y') {
    Write-Info "Creating tag $tagName..."
    git tag $tagName

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create tag"
        exit 1
    }

    Write-Info "Pushing tag $tagName..."
    git push origin $tagName

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push tag"
        Write-Warning "You may need to push manually: git push origin $tagName"
    } else {
        Write-Success "Tag pushed successfully"
        Write-Success "GitHub Actions workflow should start building now!"
    }
} else {
    Write-Warning "Tag not created. You can create it manually later with:"
    Write-Host "  git tag $tagName"
    Write-Host "  git push origin $tagName"
}

# Optionally create PR
Write-Header "Step 9: Pull Request (Optional)"
if ($currentBranch -ne 'main') {
    $createPR = Read-Host "Create Pull Request to merge '$currentBranch' into 'main'? (y/N)"

    if ($createPR -eq 'y' -or $createPR -eq 'Y') {
        # Check if gh CLI is installed
        $ghInstalled = Get-Command gh -ErrorAction SilentlyContinue

        if (-not $ghInstalled) {
            Write-Warning "GitHub CLI (gh) is not installed"
            Write-Info "Install it from: https://cli.github.com/"
            Write-Info "Or create PR manually at: https://github.com/your-repo/pulls"
        } else {
            Write-Info "Creating Pull Request..."

            $prBody = @"
## Summary
$releaseType release v$newVersion

## Changes
$changelogForCommit

## Test plan
- [x] Version updated in all required files
- [x] Changelog entry created
- [x] Changes committed to branch

🤖 Generated with [Claude Code](https://claude.com/claude-code)
"@

            gh pr create --title "Release v$newVersion" --body $prBody

            if ($LASTEXITCODE -eq 0) {
                Write-Success "Pull Request created"

                $mergePR = Read-Host "`nMerge PR immediately? (y/N)"
                if ($mergePR -eq 'y' -or $mergePR -eq 'Y') {
                    gh pr merge --delete-branch --squash
                    if ($LASTEXITCODE -eq 0) {
                        Write-Success "PR merged and branch deleted"
                    }
                }
            } else {
                Write-Warning "Failed to create PR. You can create it manually."
            }
        }
    }
} else {
    Write-Info "Already on main branch, skipping PR creation"
}

# Monitor GitHub Actions (optional)
Write-Header "Step 10: Monitor GitHub Actions (Optional)"
$monitor = Read-Host "Monitor GitHub Actions workflow? (y/N)"
if ($monitor -eq 'y' -or $monitor -eq 'Y') {
    $ghInstalled = Get-Command gh -ErrorAction SilentlyContinue

    if (-not $ghInstalled) {
        Write-Warning "GitHub CLI (gh) is not installed"
        Write-Info "View workflow at: https://github.com/your-repo/actions"
    } else {
        Write-Info "Recent workflow runs:"
        gh run list --limit 5

        Write-Info "`nTo watch the current run:"
        Write-Host "  gh run watch"
    }
}

# Final summary
Write-Header "Release Summary"
Write-Success "Release v$newVersion completed!"
Write-Info "`nWhat happened:"
Write-Host "  ✓ Version updated: $currentVersion → $newVersion"
Write-Host "  ✓ Changelog created/updated: $changelogEntryPath"
Write-Host "  ✓ Changes committed to git"
if ($pushBranch -ne 'n' -and $pushBranch -ne 'N') {
    Write-Host "  ✓ Pushed to origin/$currentBranch"
}
if ($createTag -eq 'y' -or $createTag -eq 'Y') {
    Write-Host "  ✓ Tag $tagName created and pushed (GitHub Actions triggered)"
}

Write-Info "`nNext steps:"
if ($createTag -eq 'y' -or $createTag -eq 'Y') {
    Write-Host "  1. Monitor GitHub Actions: gh run watch"
    Write-Host "  2. Verify release created: https://github.com/your-repo/releases"
    Write-Host "  3. Test downloaded binaries from release"
} else {
    Write-Host "  1. Push tag to trigger build: git push origin $tagName"
    Write-Host "  2. Monitor GitHub Actions: gh run watch"
}

Write-Success "`nRelease process complete!"

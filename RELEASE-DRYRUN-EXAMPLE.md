# Release Script Dry-Run Example

This document shows exactly what happens when you run `.\release.ps1`

## Current State

```powershell
Current Version: 1.16.1
Current Branch:  main
Git Status:      Clean (no uncommitted changes)
```

## Release Script Execution

### Step 1: Check Git Status

```
========================================
Step 1: Checking Git Status
========================================

✓ Git status checked
```

**What it does:** Checks for uncommitted changes and warns if found.

---

### Step 2: Version Information

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

**User input:** Press **Enter** (accepts default 1.16.2)

```
✓ New version: 1.16.2
Release type: Patch
```

**What it does:** Prompts for new version, validates format (1.x.y only)

---

### Step 3: Update Version File

```
========================================
Step 3: Updating Version File
========================================

Updating src\VERSION...
✓ Updated src\VERSION to 1.16.2

Note: Version is automatically embedded into the binary at compile time via go:embed directive
      (No need to update source code - it reads from VERSION file)
```

**What it does:**
- Updates `src/VERSION` from `1.16.1` to `1.16.2`
- This is the ONLY file that needs updating (go:embed reads it)

---

### Step 4: Changelog Entry

```
========================================
Step 4: Changelog Entry
========================================

Creating new changelog entry at Changelog\1.16.2.md

Enter changelog entries (press Enter on empty line to finish each section)

[Added] New features:
  -
```

**User input:** Press **Enter** (no additions)

```
[Changed] Changes to existing functionality:
  - Changed default action from getevents to getinbox
  - Updated all documentation to reflect new default
  - Created interactive release script (release.ps1)
  - Added comprehensive release documentation (RELEASE.md)
  -
```

**User input:** Entries above, then press **Enter** on empty line

```
[Fixed] Bug fixes:
  -
```

**User input:** Press **Enter** (no fixes)

```
[Security] Security improvements:
  -
```

**User input:** Press **Enter** (no security updates)

```
Changelog entry:
------------------------
## [1.16.2] 2026-01-04

### Changed

- Changed default action from getevents to getinbox
- Updated all documentation to reflect new default
- Created interactive release script (release.ps1)
- Added comprehensive release documentation (RELEASE.md)
------------------------

✓ Created Changelog\1.16.2.md
```

**What it does:**
- Creates `Changelog/1.16.2.md` with your entries
- Formats with date and proper markdown sections

---

### Step 5: Review Changes

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
- Updated all documentation to reflect new default
- Created interactive release script (release.ps1)
- Added comprehensive release documentation (RELEASE.md)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
------------------------

Do you want to proceed with these changes? (y/N):
```

**User input:** Type **y** and press **Enter**

**What it does:**
- Shows you exactly what will be committed
- Gives you a chance to cancel before making changes

---

### Step 6: Commit Changes

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

**What it does:**
- Stages: `src/VERSION` and `Changelog/1.16.2.md`
- Creates git commit with formatted message

---

### Step 7: Push to Remote

```
========================================
Step 7: Push to Remote
========================================

Push commit to remote branch 'main'? (Y/n):
```

**User input:** Press **Enter** (yes)

```
Pushing to origin/main...
✓ Pushed to origin/main
```

**What it does:**
- Pushes commit to GitHub repository

---

### Step 8: Create Git Tag (⚡TRIGGERS GITHUB ACTIONS)

```
========================================
Step 8: Create Git Tag (Triggers GitHub Actions)
========================================

⚠ Creating and pushing tag 'v1.16.2' will trigger GitHub Actions workflow!
This will:
  • Build Windows and Linux binaries
  • Create GitHub Release
  • Attach ZIP files with binaries to the release

Create and push tag 'v1.16.2'? (y/N):
```

**User input:** Type **y** and press **Enter**

```
Creating tag v1.16.2...
Pushing tag v1.16.2...
✓ Tag pushed successfully
✓ GitHub Actions workflow should start building now!
```

**What it does:**
- Creates tag `v1.16.2` locally
- Pushes tag to GitHub
- **⚡ THIS TRIGGERS GITHUB ACTIONS BUILD**

**GitHub Actions Workflow:**
1. Checks out code at tag v1.16.2
2. Sets up Go 1.21
3. Builds Windows binary: `msgraphgolangtestingtool.exe`
4. Builds Linux binary: `msgraphgolangtestingtool`
5. Creates ZIP files:
   - `msgraphgolangtestingtool-windows.zip` (exe + EXAMPLES.md + LICENSE + README.md)
   - `msgraphgolangtestingtool-linux.zip` (binary + EXAMPLES.md + LICENSE + README.md)
6. Creates GitHub Release titled "v1.16.2"
7. Attaches both ZIP files to the release
8. Generates release notes from changelog

---

### Step 9: Pull Request (Optional)

```
========================================
Step 9: Pull Request (Optional)
========================================

Already on main branch, skipping PR creation
```

**What it does:**
- If on feature branch, offers to create PR to main
- Since we're on main, skips this step

---

### Step 10: Monitor GitHub Actions (Optional)

```
========================================
Step 10: Monitor GitHub Actions (Optional)
========================================

Monitor GitHub Actions workflow? (y/N):
```

**User input:** Type **y** and press **Enter**

```
Recent workflow runs:
✓  Build and Release  v1.16.2  main  1m ago

To watch the current run:
  gh run watch
```

**What it does:**
- Lists recent GitHub Actions runs
- Suggests command to watch live: `gh run watch`

---

### Final Summary

```
========================================
Release Summary
========================================

✓ Release v1.16.2 completed!

What happened:
  ✓ Version updated: 1.16.1 → 1.16.2
  ✓ Changelog created/updated: Changelog\1.16.2.md
  ✓ Changes committed to git
  ✓ Pushed to origin/main
  ✓ Tag v1.16.2 created and pushed (GitHub Actions triggered)

Next steps:
  1. Monitor GitHub Actions: gh run watch
  2. Verify release created: https://github.com/ziembor/msgraphgolangtestingtool/releases
  3. Test downloaded binaries from release

✓ Release process complete!
```

---

## What Files Changed

### Before

```
src/VERSION:
1.16.1

Changelog/: 
  1.16.1.md
```

### After

```
src/VERSION:
1.16.2

Changelog/: 
  1.16.1.md
  1.16.2.md  ← NEW

Git tags:
  v1.16.1
  v1.16.2  ← NEW

GitHub Release:
  v1.16.2  ← NEW (with Windows & Linux ZIPs attached)
```

---

## GitHub Actions Build Output

After tag push, GitHub Actions builds:

```
.github/workflows/build.yml
├─ Windows Runner
│  ├─ Checkout code at v1.16.2
│  ├─ Setup Go 1.21
│  ├─ Build: go build -C src -o msgraphgolangtestingtool.exe
│  ├─ Verify build output
│  ├─ Create ZIP: msgraphgolangtestingtool-windows.zip
│  │  ├─ msgraphgolangtestingtool.exe
│  │  ├─ EXAMPLES.md
│  │  ├─ LICENSE
│  │  └─ README.md
│  └─ Upload to release
│
└─ Linux Runner
   ├─ Checkout code at v1.16.2
   ├─ Setup Go 1.21
   ├─ Build: go build -C src -o msgraphgolangtestingtool
   ├─ Verify build output
   ├─ Create ZIP: msgraphgolangtestingtool-linux.zip
   │  ├─ msgraphgolangtestingtool
   │  ├─ EXAMPLES.md
   │  ├─ LICENSE
   │  └─ README.md
   └─ Upload to release

GitHub Release Created:
  Title: v1.16.2
  Tag: v1.16.2
  Assets:
    - msgraphgolangtestingtool-windows.zip (~ 15-20 MB)
    - msgraphgolangtestingtool-linux.zip (~ 15-20 MB)
  Body: (Auto-generated from changelog)
```

---

## Verification Commands

After release, verify everything worked:

```powershell
# Check local version file
cat src\VERSION
# Output: 1.16.2

# Check changelog created
cat Changelog\1.16.2.md

# Check git log
git log -1 --oneline
# Output: abc1234 Release v1.16.2 - Patch Release

# Check tags
git tag -l "v1.16*"
# Output: v1.16.1
#         v1.16.2

# Check GitHub release
gh release view v1.16.2

# List workflow runs
gh run list --limit 5

# Watch workflow (live)
gh run watch

# Download and test binary
gh release download v1.16.2 -p "msgraphgolangtestingtool-windows.zip"
Expand-Archive msgraphgolangtestingtool-windows.zip -DestinationPath test
.\test\msgraphgolangtestingtool.exe -version
# Output: Microsoft Graph EXO Mails/Calendar Golang Testing Tool - Version 1.16.2
```

---

## Total Time

Typical execution time:
- **User interaction**: 2-3 minutes (entering changelog, confirmations)
- **Local operations**: 5-10 seconds (git commit, tag, push)
- **GitHub Actions build**: 3-5 minutes (compile binaries, create release)

**Total**: ~5-10 minutes from start to finished release

---

## Safety Features

The script has multiple safety checks:

1. ✅ **Git Status Check** - Warns about uncommitted changes
2. ✅ **Version Format Validation** - Must be 1.x.y
3. ✅ **Version Increment Validation** - Must be greater than current
4. ✅ **Review Step** - Shows all changes before committing
5. ✅ **Confirmation Prompts** - Asks before each major action
6. ✅ **Rollback on Cancel** - Restores VERSION file if you cancel
7. ✅ **No Force Operations** - Never uses git --force

---

## Cancellation

You can cancel at any time by pressing **N** or **Ctrl+C**:

- **Before Step 5 (Review)**: No changes made, VERSION file restored
- **After Step 6 (Commit)**: Changes committed locally, but not pushed
- **After Step 7 (Push)**: Changes pushed, but tag not created (no GitHub Actions)
- **After Step 8 (Tag)**: Full release process completed

---

## Next Release

After completing v1.16.2, next run would suggest:
- Patch: 1.16.3
- Minor: 1.17.0

Version history maintained in:
- `src/VERSION` - Current version
- `Changelog/` directory - All version changelogs
- Git tags - All released versions

---

**Ready to run the real release?**

See **[RELEASE.md](RELEASE.md)** for comprehensive release documentation.

```powershell
.\release.ps1
```
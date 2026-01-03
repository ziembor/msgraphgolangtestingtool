# Version Management Guide

## Current Version
**1.12.6** (stored in `/VERSION` file)

## Version Update Procedure

When updating the version, you MUST update all four locations:

### 1. VERSION File
```bash
echo "1.x.y" > VERSION
```

### 2. Source Code Constant
File: `src/msgraphgolangtestingtool.go`
```go
const version = "1.x.y"
```

### 3. CHANGELOG.md
Add a new entry at the top with the version, date, and changes:
```markdown
## [1.x.y] - YYYY-MM-DD

### Added
- New features

### Changed
- Changes to existing functionality

### Fixed
- Bug fixes

### Security
- Security improvements
```

### 4. Verification
```bash
# Rebuild
cd src && go build -o ../msgraphgolangtestingtool.exe

# Verify all locations are updated
cat VERSION
grep "const version" src/msgraphgolangtestingtool.go
./msgraphgolangtestingtool.exe -version
head -10 CHANGELOG.md  # Check first entry is the new version
```

## Versioning Policy

- **Major version is LOCKED at 1** - Never increment to 2.x.y
- Use minor version for new features: 1.X.0
- Use patch version for bug fixes: 1.12.X
- All breaking changes stay within 1.x.y range

## Release Process

1. Update version in VERSION file, source code, and CHANGELOG.md
2. Document all changes in CHANGELOG.md following Keep a Changelog format
3. Update documentation if needed
4. Rebuild the application
5. Verify all four locations match
6. Commit changes with message: "Bump version to 1.x.y"
7. Create git tag: `git tag v1.x.y`
8. Push tag: `git push origin v1.x.y`
9. GitHub Actions will automatically build and create release

## Version History Quick Reference

- **1.12.6** - Current version
- **1.12.5** - (intermediate version)
- **1.12.4** - Added verbose mode with env var display, renamed MSGRAPHTENANT to MSGRAPHTENANTID
- **1.12.0** - Base version (before tracking in VERSION file)

## For Future AI Assistants

⚠️ **CRITICAL**: Always check and update ALL version-related files when making version changes!

Steps:
1. Read current version: `cat VERSION`
2. Update VERSION file with new version
3. Update `src/msgraphgolangtestingtool.go` const version
4. Update CHANGELOG.md with new entry at the top (include date and all changes)
5. Rebuild and test
6. Verify all four locations match (VERSION file, source code, CHANGELOG.md, compiled binary)

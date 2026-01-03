# Version Management Guide

## Current Version
**1.12.4** (stored in `/VERSION` file)

## Version Update Procedure

When updating the version, you MUST update all three locations:

### 1. VERSION File
```bash
echo "1.x.y" > VERSION
```

### 2. Source Code Constant
File: `src/msgraphgolangtestingtool.go`
```go
const version = "1.x.y"
```

### 3. Verification
```bash
# Rebuild
cd src && go build -o ../msgraphgolangtestingtool.exe

# Verify all three match
cat VERSION
grep "const version" src/msgraphgolangtestingtool.go
./msgraphgolangtestingtool.exe -version
```

## Versioning Policy

- **Major version is LOCKED at 1** - Never increment to 2.x.y
- Use minor version for new features: 1.X.0
- Use patch version for bug fixes: 1.12.X
- All breaking changes stay within 1.x.y range

## Release Process

1. Update version in VERSION file and source code
2. Update documentation if needed
3. Rebuild the application
4. Commit changes with message: "Bump version to 1.x.y"
5. Create git tag: `git tag v1.x.y`
6. Push tag: `git push origin v1.x.y`
7. GitHub Actions will automatically build and create release

## Version History Quick Reference

- **1.12.4** - Added verbose mode with env var display, renamed MSGRAPHTENANT to MSGRAPHTENANTID
- **1.12.0** - Base version (before tracking in VERSION file)

## For Future AI Assistants

⚠️ **CRITICAL**: Always check and update the VERSION file when making version changes!

Steps:
1. Read current version: `cat VERSION`
2. Update VERSION file with new version
3. Update `src/msgraphgolangtestingtool.go` const version
4. Rebuild and test
5. Verify all three locations match

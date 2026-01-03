# Version Management Guide

## Single Source of Truth

**Version is stored in `src/VERSION` file only.**

The Go source code uses `//go:embed VERSION` to read the version at compile time - no need to update any source code.

## Version Update Procedure

When updating the version, you MUST update these locations:

### 1. VERSION File (Single Source of Truth)
```bash
echo "1.x.y" > src/VERSION
```

### 2. Changelog Entry
Create a new file in `Changelog/` folder named `{version}.md`:

File: `Changelog/1.x.y.md`
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

### 3. Verification
```bash
# Rebuild (version is embedded at compile time)
cd src && go build -o ../msgraphgolangtestingtool.exe

# Verify version
cat src/VERSION
./msgraphgolangtestingtool.exe -version
ls Changelog/1.x.y.md  # Check changelog file exists
```

## Changelog Format

Changelogs are stored as individual files in the `Changelog/` folder:
- **Location**: `Changelog/{version}.md`
- **Format**: Keep a Changelog format (https://keepachangelog.com)
- **Naming**: Use exact version number as filename (e.g., `1.14.0.md`, `1.12.7.md`)

Example structure:
```
Changelog/
├── 1.0.0.md
├── 1.0.1.md
├── 1.12.7.md
├── 1.14.0.md
└── ...
```

## Versioning Policy

- **Major version is LOCKED at 1** - Never increment to 2.x.y
- Use minor version for new features: 1.X.0
- Use patch version for bug fixes: 1.12.X
- All breaking changes stay within 1.x.y range

## Release Process

1. Update version in `src/VERSION` file
2. Create changelog file `Changelog/{version}.md` following Keep a Changelog format
3. Update documentation if needed
4. Rebuild the application (version is embedded automatically)
5. Verify version matches in binary output
6. Commit changes with message: "Bump version to 1.x.y"
7. Create git tag: `git tag v1.x.y`
8. Push tag: `git push origin v1.x.y`
9. GitHub Actions will automatically build and create release

## For Future AI Assistants

Steps:
1. Read current version: `cat src/VERSION`
2. Update `src/VERSION` file with new version
3. Create `Changelog/{version}.md` with all changes (include date)
4. Rebuild: `cd src && go build -o ../msgraphgolangtestingtool.exe`
5. Verify: `./msgraphgolangtestingtool.exe -version`

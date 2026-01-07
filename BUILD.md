# Build Instructions

This document provides instructions for building the `msgraphgolangtestingtool` executable.

## Prerequisites

1. **Go 1.25+**: [Download Go](https://golang.org/dl/)
2. **Git**: [Download Git](https://git-scm.com/downloads)

## Build Steps

All Go source code is located in the `src/` directory.

### 1. Standard Build
Creates `msgraphgolangtestingtool.exe` in the project root.

```powershell
# Build from project root
go build -C src -o msgraphgolangtestingtool.exe
```

### 2. Optimized Build (Smaller Binary)
Strips debug information and symbol tables.

```powershell
go build -C src -ldflags="-s -w" -o msgraphgolangtestingtool.exe
```

### 3. Cross-Platform Builds

**Linux:**
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -C src -o msgraphgolangtestingtool
Remove-Item Env:\GOOS; Remove-Item Env:\GOARCH
```
*Note: Windows Certificate Store auth (`-thumbprint`) is not available on Linux.*

**macOS:**
```powershell
$env:GOOS="darwin"; $env:GOARCH="amd64"
go build -C src -o msgraphgolangtestingtool
Remove-Item Env:\GOOS; Remove-Item Env:\GOARCH
```

## Release Process

> **Moved:** The complete release and versioning guide is now located in **[RELEASE.md](RELEASE.md)**.

To cut a new release, use the automated script:
```powershell
.\release.ps1
```

## Troubleshooting

- **"go: command not found"**: Ensure Go is in your PATH.
- **"package X is not in GOROOT"**: Run `go mod download` inside `src/`.
- **"Access Denied"**: Close any running instances of the tool.

## Development

Run directly without building:
```powershell
cd src
go run . -action getinbox ...
```

                          ..ooOO END OOoo..



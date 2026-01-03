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
cd C:\Workspace\golang\msgraphgolangtestingtool
```

### 2. Download Dependencies
Go will automatically download required dependencies during the build, but you can explicitly fetch them:
```powershell
go mod download
```

This downloads:
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity`
- `github.com/microsoftgraph/msgraph-sdk-go`
- `golang.org/x/crypto/pkcs12`
- And all transitive dependencies

### 3. Build the Executable

#### Standard Build
Creates `msgraphgolangtestingtool.exe` in the current directory:
```powershell
go build -o msgraphgolangtestingtool.exe msgraphgolangtestingtool.go
```

#### Optimized Build (Smaller Binary)
Removes debug information and symbol table for a smaller executable:
```powershell
go build -ldflags="-s -w" -o msgraphgolangtestingtool.exe msgraphgolangtestingtool.go
```

#### Build with Version Information
Include version metadata in the binary:
```powershell
go build -ldflags="-s -w -X main.version=1.0.0" -o msgraphgolangtestingtool.exe msgraphgolangtestingtool.go
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
go build -o msgraphgolangtestingtool
```
**Note**: Windows Certificate Store authentication (`-thumbprint`) will not work on Linux.

### Build for macOS
```powershell
$env:GOOS="darwin"
$env:GOARCH="amd64"
go build -o msgraphgolangtestingtool msgraphgolangtestingtool.go
```
**Note**: Windows Certificate Store authentication (`-thumbprint`) will not work on macOS.

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
go run msgraphgolangtestingtool.go -tenant "xxx" -client "xxx" -secret "xxx" -mailbox "user@example.com" -action getevents
```

## Clean Build

To remove build artifacts and start fresh:
```powershell
# Remove the executable
Remove-Item msgraphgolangtestingtool.exe -ErrorAction SilentlyContinue

# Clean Go build cache
go clean -cache

# Re-download dependencies
go mod download

# Rebuild
go build -o msgraphgolangtestingtool.exe msgraphgolangtestingtool.go
```

## Automated Build Script

Create a `build.ps1` file for automated builds:
```powershell
# build.ps1
Write-Host "Building msgraphgolangtestingtool..." -ForegroundColor Green

# Clean previous build
Remove-Item msgraphgolangtestingtool.exe -ErrorAction SilentlyContinue

# Build with optimization
go build -ldflags="-s -w" -o msgraphgolangtestingtool.exe msgraphgolangtestingtool.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    $size = (Get-Item msgraphgolangtestingtool.exe).Length / 1MB
    Write-Host ("Binary size: {0:N2} MB" -f $size) -ForegroundColor Cyan
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

After building, see the [GEMINI.md](GEMINI.md) or [CLAUDE.md](CLAUDE.md) files for usage instructions.

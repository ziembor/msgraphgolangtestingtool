#!/usr/bin/env pwsh
# Build script for both msgraphgolangtestingtool and smtptool
# Builds optimized binaries for both tools with version embedding

param(
    [switch]$Verbose,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

# Colors for output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Header
Write-ColorOutput "`n═══════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "  Microsoft Graph & SMTP Tools - Build Script" "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════`n" "Cyan"

# Read version from version.go
$versionFile = Join-Path $PSScriptRoot "internal" "common" "version" "version.go"
if (-not (Test-Path $versionFile)) {
    Write-ColorOutput "ERROR: version.go not found at $versionFile" "Red"
    exit 1
}
$versionContent = Get-Content $versionFile -Raw
if ($versionContent -match 'const Version = "([^"]+)"') {
    $version = $matches[1]
} else {
    Write-ColorOutput "ERROR: Could not extract version from version.go" "Red"
    exit 1
}
Write-ColorOutput "Version: $version`n" "Yellow"

# Build Microsoft Graph Tool
Write-ColorOutput "Building Microsoft Graph Tool..." "Cyan"
Write-ColorOutput "  Location: cmd/msgraphtool" "Gray"
Write-ColorOutput "  Output:   msgraphgolangtestingtool.exe`n" "Gray"

try {
    $buildDir = Join-Path $PSScriptRoot "cmd" "msgraphtool"
    $outputFile = Join-Path $PSScriptRoot "msgraphgolangtestingtool.exe"

    Push-Location $buildDir
    if ($Verbose) {
        go build -v -ldflags="-s -w" -o $outputFile
    } else {
        go build -ldflags="-s -w" -o $outputFile
    }
    Pop-Location

    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $outputFile).Length / 1MB
        Write-ColorOutput "  ✓ Build successful (Size: $($size.ToString('N2')) MB)" "Green"
    } else {
        throw "Build failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-ColorOutput "  ✗ Build failed: $_" "Red"
    exit 1
}

# Build SMTP Tool
Write-ColorOutput "`nBuilding SMTP Connectivity Tool..." "Cyan"
Write-ColorOutput "  Location: cmd/smtptool" "Gray"
Write-ColorOutput "  Output:   smtptool.exe`n" "Gray"

try {
    $buildDir = Join-Path $PSScriptRoot "cmd" "smtptool"
    $outputFile = Join-Path $PSScriptRoot "smtptool.exe"

    Push-Location $buildDir
    if ($Verbose) {
        go build -v -ldflags="-s -w" -o $outputFile
    } else {
        go build -ldflags="-s -w" -o $outputFile
    }
    Pop-Location

    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $outputFile).Length / 1MB
        Write-ColorOutput "  ✓ Build successful (Size: $($size.ToString('N2')) MB)" "Green"
    } else {
        throw "Build failed with exit code $LASTEXITCODE"
    }
} catch {
    Write-ColorOutput "  ✗ Build failed: $_" "Red"
    exit 1
}

# Run tests (optional)
if (-not $SkipTests) {
    Write-ColorOutput "`nRunning tests..." "Cyan"

    # Test Graph tool version
    Write-ColorOutput "  Testing msgraphgolangtestingtool version..." "Gray"
    $graphExe = Join-Path $PSScriptRoot "msgraphgolangtestingtool.exe"
    $graphVersion = & $graphExe -version
    if ($graphVersion -match $version) {
        Write-ColorOutput "    ✓ Version correct: $version" "Green"
    } else {
        Write-ColorOutput "    ⚠ Version mismatch (expected: $version)" "Yellow"
    }

    # Test SMTP tool version
    Write-ColorOutput "  Testing smtptool version..." "Gray"
    $smtpExe = Join-Path $PSScriptRoot "smtptool.exe"
    $smtpVersion = & $smtpExe -version
    if ($smtpVersion -match $version) {
        Write-ColorOutput "    ✓ Version correct: $version" "Green"
    } else {
        Write-ColorOutput "    ⚠ Version mismatch (expected: $version)" "Yellow"
    }
}

# Summary
Write-ColorOutput "`n═══════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "  Build Complete!" "Green"
Write-ColorOutput "═══════════════════════════════════════════════════════════" "Cyan"

Write-ColorOutput "`nBuilt executables:" "White"
Write-ColorOutput "  • msgraphgolangtestingtool.exe - Microsoft Graph API tool" "White"
Write-ColorOutput "  • smtptool.exe                  - SMTP connectivity testing tool" "White"

Write-ColorOutput "`nUsage examples:" "Yellow"
Write-ColorOutput "  .\msgraphgolangtestingtool.exe -version" "Gray"
Write-ColorOutput "  .\smtptool.exe -action testconnect -host smtp.example.com -port 25" "Gray"
Write-ColorOutput "  .\smtptool.exe -action teststarttls -host smtp.example.com -port 587`n" "Gray"

Write-ColorOutput "For more information, see BUILD.md and SMTP_TOOL_README.md`n" "Cyan"

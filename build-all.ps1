#!/usr/bin/env pwsh
# Build script for all gomailtesttool binaries
# Builds optimized binaries for all 5 tools with version embedding

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
Write-ColorOutput "  gomailtesttool Suite - Build Script" "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════`n" "Cyan"

# Ensure bin directory exists
$binDir = Join-Path $PSScriptRoot "bin"
if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir | Out-Null
    Write-ColorOutput "Created bin/ directory`n" "Yellow"
}

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

# Define tools to build
$tools = @(
    @{ Name = "msgraphtool"; Desc = "Microsoft Graph API tool" },
    @{ Name = "smtptool"; Desc = "SMTP connectivity testing" },
    @{ Name = "imaptool"; Desc = "IMAP server testing" },
    @{ Name = "pop3tool"; Desc = "POP3 server testing" },
    @{ Name = "jmaptool"; Desc = "JMAP protocol testing" }
)

# Build each tool
foreach ($tool in $tools) {
    Write-ColorOutput "Building $($tool.Name)..." "Cyan"
    Write-ColorOutput "  Location: cmd/$($tool.Name)" "Gray"
    Write-ColorOutput "  Output:   $($tool.Name).exe`n" "Gray"

    try {
        $buildDir = Join-Path $PSScriptRoot "cmd" $tool.Name
        $outputFile = Join-Path $binDir "$($tool.Name).exe"

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
}

# Run tests (optional)
if (-not $SkipTests) {
    Write-ColorOutput "`nRunning version tests..." "Cyan"

    foreach ($tool in $tools) {
        Write-ColorOutput "  Testing $($tool.Name) version..." "Gray"
        $exe = Join-Path $binDir "$($tool.Name).exe"
        $toolVersion = & $exe -version 2>&1
        if ($toolVersion -match $version) {
            Write-ColorOutput "    ✓ Version correct: $version" "Green"
        } else {
            Write-ColorOutput "    ⚠ Version mismatch (expected: $version)" "Yellow"
        }
    }
}

# Summary
Write-ColorOutput "`n═══════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "  Build Complete!" "Green"
Write-ColorOutput "═══════════════════════════════════════════════════════════" "Cyan"

Write-ColorOutput "`nBuilt executables in bin/:" "White"
foreach ($tool in $tools) {
    Write-ColorOutput "  • bin\$($tool.Name).exe - $($tool.Desc)" "White"
}

Write-ColorOutput "`nUsage examples:" "Yellow"
Write-ColorOutput "  .\bin\msgraphtool.exe -version" "Gray"
Write-ColorOutput "  .\bin\smtptool.exe -action testconnect -host smtp.example.com -port 25" "Gray"
Write-ColorOutput "  .\bin\imaptool.exe -action testconnect -host imap.gmail.com -imaps" "Gray"
Write-ColorOutput "  .\bin\pop3tool.exe -action testconnect -host pop.gmail.com -pop3s" "Gray"
Write-ColorOutput "  .\bin\jmaptool.exe -action testconnect -host jmap.fastmail.com`n" "Gray"

Write-ColorOutput "For more information, see BUILD.md and tool-specific READMEs`n" "Cyan"

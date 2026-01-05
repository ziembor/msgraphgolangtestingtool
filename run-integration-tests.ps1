# PowerShell script to run integration tests for Microsoft Graph EXO Mails/Calendar Golang Testing Tool
#
# This script:
# 1. Checks for required environment variables
# 2. Validates Go is installed
# 3. Builds and runs the integration test tool
# 4. Displays results

param(
    [switch]$SetEnv,      # Set environment variables interactively
    [switch]$ShowEnv,     # Display current environment variables
    [switch]$ClearEnv,    # Clear all MSGRAPH* environment variables
    [switch]$AutoConfirm  # Auto-confirm all prompts (for CI/CD)
)

$ErrorActionPreference = "Stop"

# Color output functions
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Warning { Write-Host $args -ForegroundColor Yellow }
function Write-Error { Write-Host $args -ForegroundColor Red }
function Write-Info { Write-Host $args -ForegroundColor Cyan }

# Banner
Write-Host ""
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "Integration Test Runner - Microsoft Graph EXO Mails/Calendar Golang Testing Tool" -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host ""

# Handle -SetEnv flag
if ($SetEnv) {
    Write-Info "Setting environment variables interactively..."
    Write-Host ""

    $env:MSGRAPHTENANTID = Read-Host "Enter Tenant ID (GUID)"
    $env:MSGRAPHCLIENTID = Read-Host "Enter Client ID (GUID)"
    $secret = Read-Host "Enter Client Secret" -AsSecureString
    $env:MSGRAPHSECRET = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret))
    $env:MSGRAPHMAILBOX = Read-Host "Enter Test Mailbox Email"

    Write-Success "✅ Environment variables set"
    Write-Warning "Note: These are session-only. They will be cleared when you close PowerShell."
    Write-Host ""
}

# Handle -ShowEnv flag
if ($ShowEnv) {
    Write-Info "Current MSGRAPH* environment variables:"
    Write-Host ""

    if ($env:MSGRAPHTENANTID) {
        $masked = $env:MSGRAPHTENANTID.Substring(0,4) + "****-****-****-****" + $env:MSGRAPHTENANTID.Substring($env:MSGRAPHTENANTID.Length-4)
        Write-Host "  MSGRAPHTENANTID: $masked"
    } else {
        Write-Warning "  MSGRAPHTENANTID: [NOT SET]"
    }

    if ($env:MSGRAPHCLIENTID) {
        $masked = $env:MSGRAPHCLIENTID.Substring(0,4) + "****-****-****-****" + $env:MSGRAPHCLIENTID.Substring($env:MSGRAPHCLIENTID.Length-4)
        Write-Host "  MSGRAPHCLIENTID: $masked"
    } else {
        Write-Warning "  MSGRAPHCLIENTID: [NOT SET]"
    }

    if ($env:MSGRAPHSECRET) {
        if ($env:MSGRAPHSECRET.Length -gt 8) {
            $masked = $env:MSGRAPHSECRET.Substring(0,4) + "********" + $env:MSGRAPHSECRET.Substring($env:MSGRAPHSECRET.Length-4)
        } else {
            $masked = "********"
        }
        Write-Host "  MSGRAPHSECRET: $masked"
    } else {
        Write-Warning "  MSGRAPHSECRET: [NOT SET]"
    }

    if ($env:MSGRAPHMAILBOX) {
        Write-Host "  MSGRAPHMAILBOX: $env:MSGRAPHMAILBOX"
    } else {
        Write-Warning "  MSGRAPHMAILBOX: [NOT SET]"
    }

    Write-Host ""
    exit 0
}

# Handle -ClearEnv flag
if ($ClearEnv) {
    Write-Warning "Clearing all MSGRAPH* environment variables..."
    Remove-Item Env:\MSGRAPHTENANTID -ErrorAction SilentlyContinue
    Remove-Item Env:\MSGRAPHCLIENTID -ErrorAction SilentlyContinue
    Remove-Item Env:\MSGRAPHSECRET -ErrorAction SilentlyContinue
    Remove-Item Env:\MSGRAPHMAILBOX -ErrorAction SilentlyContinue
    Remove-Item Env:\MSGRAPHPROXY -ErrorAction SilentlyContinue
    Write-Success "✅ Environment variables cleared"
    exit 0
}

# Check for required environment variables
Write-Info "Checking environment variables..."
$missingVars = @()

if (-not $env:MSGRAPHTENANTID) { $missingVars += "MSGRAPHTENANTID" }
if (-not $env:MSGRAPHCLIENTID) { $missingVars += "MSGRAPHCLIENTID" }
if (-not $env:MSGRAPHSECRET) { $missingVars += "MSGRAPHSECRET" }
if (-not $env:MSGRAPHMAILBOX) { $missingVars += "MSGRAPHMAILBOX" }

if ($missingVars.Count -gt 0) {
    Write-Error "❌ Missing required environment variables:"
    foreach ($var in $missingVars) {
        Write-Host "  - $var" -ForegroundColor Red
    }
    Write-Host ""
    Write-Info "To set environment variables, use:"
    Write-Host "  .\run-integration-tests.ps1 -SetEnv" -ForegroundColor Yellow
    Write-Host ""
    Write-Info "Or set them manually:"
    Write-Host '  $env:MSGRAPHTENANTID = "your-tenant-id"' -ForegroundColor Yellow
    Write-Host '  $env:MSGRAPHCLIENTID = "your-client-id"' -ForegroundColor Yellow
    Write-Host '  $env:MSGRAPHSECRET = "your-secret"' -ForegroundColor Yellow
    Write-Host '  $env:MSGRAPHMAILBOX = "test@example.com"' -ForegroundColor Yellow
    exit 1
}

Write-Success "✅ All required environment variables are set"

# Check for Go installation
Write-Info "Checking Go installation..."
try {
    $goVersion = go version 2>&1
    Write-Success "✅ Go is installed: $goVersion"
} catch {
    Write-Error "❌ Go is not installed or not in PATH"
    Write-Info "Download Go from: https://go.dev/dl/"
    exit 1
}

# Build integration test tool
Write-Info "Building integration test tool..."
Push-Location src

try {
    $buildOutput = go build -tags integration -o ../integration_test_tool.exe integration_test_tool.go msgraphgolangtestingtool_lib.go cert_windows.go 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-Error "❌ Build failed:"
        Write-Host $buildOutput
        exit 1
    }

    Write-Success "✅ Build successful"
} finally {
    Pop-Location
}

# Set auto-confirm if flag is set
if ($AutoConfirm) {
    $env:MSGRAPH_AUTO_CONFIRM = "true"
    Write-Info "Auto-confirm mode enabled (all prompts will be auto-accepted)"
}

# Run integration tests
Write-Info "Running integration tests..."
Write-Host ""
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host ""

.\integration_test_tool.exe

$exitCode = $LASTEXITCODE

# Cleanup
if ($AutoConfirm) {
    Remove-Item Env:\MSGRAPH_AUTO_CONFIRM -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "=================================================================" -ForegroundColor Cyan

if ($exitCode -eq 0) {
    Write-Success "✅ Integration tests completed successfully"
} else {
    Write-Error "❌ Integration tests failed (exit code: $exitCode)"
}

Write-Host ""
Write-Warning "Remember to clear sensitive environment variables after testing:"
Write-Host "  .\run-integration-tests.ps1 -ClearEnv" -ForegroundColor Yellow

exit $exitCode

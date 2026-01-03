# Test script for environment variable support

Write-Host "Testing Environment Variable Support" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Environment variables only (should fail with missing parameters message)
Write-Host "Test 1: No env vars, no flags (should fail with missing parameters)" -ForegroundColor Yellow
.\msgraphgolangtestingtool.exe 2>&1 | Select-String "Missing required parameters"
Write-Host ""

# Test 2: Set environment variables
Write-Host "Test 2: Setting environment variables" -ForegroundColor Yellow
$env:MSGRAPHTENANT = "test-tenant-123"
$env:MSGRAPHCLIENTID = "test-client-456"
$env:MSGRAPHMAILBOX = "test@example.com"
Write-Host "  MSGRAPHTENANT = $env:MSGRAPHTENANT" -ForegroundColor Green
Write-Host "  MSGRAPHCLIENTID = $env:MSGRAPHCLIENTID" -ForegroundColor Green
Write-Host "  MSGRAPHMAILBOX = $env:MSGRAPHMAILBOX" -ForegroundColor Green
Write-Host ""

# Test 3: Try with just action flag (should use env vars for credentials)
Write-Host "Test 3: Using env vars with -action flag only" -ForegroundColor Yellow
Write-Host "  Command: .\msgraphgolangtestingtool.exe -action getevents" -ForegroundColor Gray
Write-Host "  Expected: Should fail at authentication (no valid auth method), NOT at parameter validation" -ForegroundColor Gray
.\msgraphgolangtestingtool.exe -action getevents 2>&1 | Select-String -Pattern "Missing required parameters|no valid authentication"
Write-Host ""

# Test 4: Override with command-line flag (flag should take precedence)
Write-Host "Test 4: Override env var with command-line flag" -ForegroundColor Yellow
Write-Host "  Command: .\msgraphgolangtestingtool.exe -mailbox override@example.com -action getevents" -ForegroundColor Gray
Write-Host "  Expected: Should use 'override@example.com' instead of env var" -ForegroundColor Gray
.\msgraphgolangtestingtool.exe -mailbox "override@example.com" -action getevents 2>&1 | Select-String -Pattern "Missing required parameters|no valid authentication"
Write-Host ""

# Test 5: Version flag (should work regardless of env vars)
Write-Host "Test 5: Version flag test" -ForegroundColor Yellow
.\msgraphgolangtestingtool.exe -version
Write-Host ""

# Cleanup
Write-Host "Cleaning up environment variables..." -ForegroundColor Cyan
Remove-Item Env:\MSGRAPHTENANT -ErrorAction SilentlyContinue
Remove-Item Env:\MSGRAPHCLIENTID -ErrorAction SilentlyContinue
Remove-Item Env:\MSGRAPHMAILBOX -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Tests completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Note: All tests should show authentication errors (not parameter validation errors)" -ForegroundColor Yellow
Write-Host "This confirms that environment variables are being read correctly." -ForegroundColor Yellow

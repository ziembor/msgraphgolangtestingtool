# Test-SendMail.ps1
# Pester test for sendmail action

BeforeAll {
    # Source environment variables from secure location
    $envPath = Join-Path $env:OneDrive "Documents\Safe\zblab21eu_env.ps1"

    if (-not (Test-Path $envPath)) {
        throw "Environment file not found: $envPath"
    }

    Write-Host "Sourcing environment from: $envPath" -ForegroundColor Cyan
    . $envPath

    # Get the path to the executable
    $script:exePath = Join-Path $PSScriptRoot "..\msgraphgolangtestingtool.exe"

    if (-not (Test-Path $script:exePath)) {
        throw "Executable not found: $script:exePath"
    }

    Write-Host "Using executable: $script:exePath" -ForegroundColor Cyan
}

Describe "Send Mail Tests" {

    Context "When sending email with verbose output" {

        It "Should execute sendmail command successfully" {
            # Execute the command
            $output = & $script:exePath -action sendmail -verbose 2>&1
            $exitCode = $LASTEXITCODE

            # Display output
            Write-Host "Command Output:" -ForegroundColor Yellow
            $output | ForEach-Object { Write-Host $_ }

            # Verify exit code
            $exitCode | Should -Be 0
        }

#         It "Should show verbose configuration output" {
#             # Execute the command and capture output
#             $output = & $script:exePath -action sendmail -verbose 2>&1 | Out-String

#             # Verify verbose output contains expected sections
#             $output | Should -Match "Environment Variables:"
#             $output | Should -Match "Final Configuration:"
#             $output | Should -Match "Authentication Details:"
#         }

#         It "Should use environment variables from sourced file" {
#             # Execute the command and capture output
#             $output = & $script:exePath -action sendmail -verbose 2>&1 | Out-String

#             # Verify that environment variables are being used
#             $output | Should -Match "MSGRAPH"
#         }

#         It "Should complete without errors" {
#             # Execute the command
#             $errorOutput = & $script:exePath -action sendmail -verbose 2>&1 |
#                 Where-Object { $_ -match "ERROR|error|Error" }

#             # There should be no error messages (unless expected authentication errors in test environment)
#             # Adjust this assertion based on your test environment
#             Write-Host "Checking for errors..." -ForegroundColor Cyan
#             if ($errorOutput) {
#                 Write-Host "Errors found:" -ForegroundColor Yellow
#                 $errorOutput | ForEach-Object { Write-Host $_ -ForegroundColor Red }
#             }
#         }
     }

    Context "When checking CSV log output" {

        It "Should create CSV log file" {
            # Execute the command
            & $script:exePath -action sendmail -verbose 2>&1 | Out-Null

            # Check for CSV log file in temp directory
            $dateStr = Get-Date -Format "yyyy-MM-dd"
            $csvPath = Join-Path $env:TEMP "_msgraphgolangtestingtool_sendmail_$dateStr.csv"

            Write-Host "Checking for CSV log: $csvPath" -ForegroundColor Cyan
            Test-Path $csvPath | Should -Be $true
        }

#         It "Should write sendmail entry to CSV log" {
#             # Execute the command
#             & $script:exePath -action sendmail -verbose 2>&1 | Out-Null

#             # Read the CSV log
#             $dateStr = Get-Date -Format "yyyy-MM-dd"
#             $csvPath = Join-Path $env:TEMP "_msgraphgolangtestingtool_$dateStr.csv"

#             if (Test-Path $csvPath) {
#                 $csvContent = Get-Content $csvPath
#                 Write-Host "CSV Content (last 5 lines):" -ForegroundColor Cyan
#                 $csvContent | Select-Object -Last 5 | ForEach-Object { Write-Host $_ }

#                 # Verify sendmail action is logged
#                 $csvContent | Should -Match "sendmail"
#             }
#         }
     }
}

AfterAll {
    Write-Host ""
    Write-Host "Test execution completed!" -ForegroundColor Green
    Write-Host "Check the output above for detailed results." -ForegroundColor Cyan
}

# Tests Directory

This directory contains test scripts for the Microsoft Graph Golang Testing Tool.

## Purpose

This directory is for:
- Unit tests for PowerShell build scripts
- Integration tests for the tool
- Test fixtures and sample data
- Mock environments for testing

## Directory Structure

```
tests/
├── README.md              # This file
└── (Future test scripts will be added here)
```

## Test Naming Convention

Test scripts should follow the naming convention: `Test-*.ps1`

## Running Tests

To run all test scripts:

```powershell
Get-ChildItem -Path tests -Filter 'Test-*.ps1' | ForEach-Object { & $_.FullName }
```

## Creating New Tests

1. Create a new file named `Test-<Feature>.ps1` in this directory
2. Add test cases using Pester framework (recommended)
3. Document test prerequisites and expected outcomes
4. Include clear pass/fail output

## Example Test Files (To Be Created)

- `Test-VersionConsistency.ps1` - Verify VERSION file matches source code and CHANGELOG.md
- `Test-BuildProcess.ps1` - Test build script execution
- `Test-GitOperations.ps1` - Test ghabuild.ps1 script
- `Test-EnvVariables.ps1` - Test environment variable processing
- `Test-CountParameter.ps1` - Test -count flag functionality

## Testing Framework

Consider using [Pester](https://pester.dev/) - PowerShell's testing framework:

```powershell
# Install Pester
Install-Module -Name Pester -Force -SkipPublisherCheck

# Example test structure
Describe "Version Consistency Tests" {
    It "VERSION file should match source code constant" {
        $versionFile = Get-Content VERSION
        $sourceVersion = Select-String -Path src/msgraphgolangtestingtool.go -Pattern 'const version = "(.+)"'
        $versionFile | Should -Be $sourceVersion.Matches.Groups[1].Value
    }
}
```

## Best Practices

- Keep tests independent and isolated
- Use descriptive test names
- Clean up test artifacts after execution
- Mock external dependencies where possible
- Document expected behavior clearly

## For More Information

See the main project README.md and CLAUDE.md for additional documentation.

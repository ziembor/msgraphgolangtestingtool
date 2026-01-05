# msgraphgolangtestingtool PowerShell completion script
# Installation:
#   Add to your PowerShell profile: notepad $PROFILE
#   Or run manually: . .\msgraphgolangtestingtool-completion.ps1

Register-ArgumentCompleter -CommandName msgraphgolangtestingtool.exe,msgraphgolangtestingtool,'.\msgraphgolangtestingtool.exe','.\msgraphgolangtestingtool' -ScriptBlock {
    param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)

    # Define valid actions
    $actions = @('getevents', 'sendmail', 'sendinvite', 'getinbox')

    # Define log levels
    $logLevels = @('DEBUG', 'INFO', 'WARN', 'ERROR')

    # Define shell types for completion flag
    $shellTypes = @('bash', 'powershell')

    # All flags that accept values
    $flags = @(
        '-action', '-tenantid', '-clientid', '-secret', '-pfx', '-pfxpass',
        '-thumbprint', '-mailbox', '-to', '-cc', '-bcc', '-subject', '-body',
        '-bodyHTML', '-attachments', '-invite-subject', '-start', '-end',
        '-proxy', '-count', '-maxretries', '-retrydelay', '-loglevel',
        '-completion', '-verbose', '-version', '-help'
    )

    # Get the last word from command line
    $lastWord = ''
    if ($commandAst.CommandElements.Count -gt 1) {
        $lastWord = $commandAst.CommandElements[-2].ToString()
    }

    # Provide context-specific completions based on the previous flag
    switch ($lastWord) {
        '-action' {
            $actions | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Action: $_")
            }
            return
        }
        '-loglevel' {
            $logLevels | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Log Level: $_")
            }
            return
        }
        '-completion' {
            $shellTypes | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Shell: $_")
            }
            return
        }
        '-pfx' {
            # File completion for PFX files
            Get-ChildItem -Path "$wordToComplete*" -File -ErrorAction SilentlyContinue |
                Where-Object { $_.Extension -in @('.pfx', '.p12') -or $wordToComplete -eq '' } |
                ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new(
                        $_.FullName,
                        $_.Name,
                        'ParameterValue',
                        "Certificate: $($_.Name)"
                    )
                }
            return
        }
        '-attachments' {
            # File completion for any file type
            Get-ChildItem -Path "$wordToComplete*" -File -ErrorAction SilentlyContinue |
                ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new(
                        $_.FullName,
                        $_.Name,
                        'ParameterValue',
                        "File: $($_.Name)"
                    )
                }
            return
        }
    }

    # Default: complete with flag names
    $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        $description = switch ($_) {
            '-action' { 'Operation to perform (getevents, sendmail, sendinvite, getinbox)' }
            '-tenantid' { 'Azure Tenant ID (GUID)' }
            '-clientid' { 'Application (Client) ID (GUID)' }
            '-secret' { 'Client Secret for authentication' }
            '-pfx' { 'Path to .pfx certificate file' }
            '-pfxpass' { 'Password for .pfx certificate' }
            '-thumbprint' { 'Certificate thumbprint (Windows Certificate Store)' }
            '-mailbox' { 'Target user email address' }
            '-to' { 'Comma-separated TO recipients' }
            '-cc' { 'Comma-separated CC recipients' }
            '-bcc' { 'Comma-separated BCC recipients' }
            '-subject' { 'Email subject line' }
            '-body' { 'Email body (text)' }
            '-bodyHTML' { 'Email body (HTML)' }
            '-attachments' { 'Comma-separated file paths to attach' }
            '-invite-subject' { 'Calendar invite subject' }
            '-start' { 'Start time for calendar invite (RFC3339)' }
            '-end' { 'End time for calendar invite (RFC3339)' }
            '-proxy' { 'HTTP/HTTPS proxy URL' }
            '-count' { 'Number of items to retrieve (default: 3)' }
            '-maxretries' { 'Maximum retry attempts (default: 3)' }
            '-retrydelay' { 'Retry delay in milliseconds (default: 2000)' }
            '-loglevel' { 'Logging level (DEBUG, INFO, WARN, ERROR)' }
            '-completion' { 'Generate completion script (bash or powershell)' }
            '-verbose' { 'Enable verbose output' }
            '-version' { 'Show version information' }
            '-help' { 'Show help message' }
            default { $_ }
        }
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $description)
    }
}

Write-Host "PowerShell completion for msgraphgolangtestingtool loaded successfully!" -ForegroundColor Green
Write-Host "Try typing: msgraphgolangtestingtool.exe -<TAB>" -ForegroundColor Cyan

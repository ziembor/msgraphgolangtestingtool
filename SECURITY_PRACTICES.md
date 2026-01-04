# Security Best Practices

This guide outlines security best practices for using the Microsoft Graph GoLang Testing Tool in production environments.

---

## Credential Management

### Client Secrets

**Never commit secrets to source control**

```powershell
# WRONG - Secret in script file committed to Git
$secret = "very-secret-value-12345"
.\msgraphgolangtestingtool.exe -secret $secret ...

# CORRECT - Secret in environment variable or secure vault
$env:MSGRAPHSECRET = Get-Content "C:\SecureLocation\secret.txt"
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -mailbox "..." -action getevents
```

**Best Practices:**
- ✅ Store secrets in Azure Key Vault, HashiCorp Vault, or encrypted files
- ✅ Use environment variables for secrets (not command-line arguments visible in process list)
- ✅ Rotate secrets regularly (every 90 days recommended)
- ✅ Use separate secrets for dev/test/prod environments
- ✅ Set expiration dates on secrets in Azure AD
- ❌ Never hardcode secrets in scripts
- ❌ Never commit secrets to version control (add to `.gitignore`)
- ❌ Never share secrets via email or chat

**Using Azure Key Vault:**
```powershell
# Retrieve secret from Azure Key Vault
Connect-AzAccount
$secret = (Get-AzKeyVaultSecret -VaultName "MyVault" -Name "GraphSecret").SecretValue
$env:MSGRAPHSECRET = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret))

# Run tool
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -mailbox "..." -action getevents

# Clear secret from memory
Remove-Item Env:\MSGRAPHSECRET
```

**Using Encrypted Files (PowerShell):**
```powershell
# One-time: Create encrypted credential file
$secret = Read-Host "Enter Graph API Secret" -AsSecureString
$secret | ConvertFrom-SecureString | Out-File "C:\SecureLocation\graph-secret.enc"

# In automation script: Load encrypted credential
$encSecret = Get-Content "C:\SecureLocation\graph-secret.enc" | ConvertTo-SecureString
$plainSecret = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($encSecret))
$env:MSGRAPHSECRET = $plainSecret

# Run tool
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -mailbox "..." -action getevents
```

---

### Certificates (Preferred for Production)

**Certificates are more secure than client secrets:**
- Cannot be easily copied/leaked like text secrets
- Support automatic rotation via certificate management
- Provide stronger authentication guarantees
- Can be stored in hardware security modules (HSM)

**Best Practices:**
- ✅ **Use certificates instead of secrets for production**
- ✅ Store certificates in Windows Certificate Store (no files on disk)
- ✅ Use strong passwords for PFX files (16+ characters, mixed case, symbols)
- ✅ Restrict file permissions on PFX files (`NTFS permissions: Administrators only`)
- ✅ Rotate certificates before expiration (set 90-day reminder)
- ✅ Use CA-signed certificates for production (not self-signed)
- ✅ Monitor certificate expiration dates
- ❌ Never store PFX files in shared network locations
- ❌ Never use weak passwords on PFX files
- ❌ Never commit PFX files to version control

**Windows Certificate Store (Most Secure):**
```powershell
# Recommended approach - certificate in Windows store, no file on disk
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -thumbprint "CD817B3329802E692CF30D8DDF896FE811B048AB" -mailbox "..." -action getevents

# Benefits:
# - No certificate file on disk (reduced attack surface)
# - Private key protected by Windows security
# - Supports TPM/HSM integration
# - Automatic access control via Windows ACLs
```

**PFX File with Strong Security:**
```powershell
# If using PFX file, secure the file and password
$pfxPath = "C:\SecureLocation\cert.pfx"

# Set restrictive NTFS permissions (Administrators only)
$acl = Get-Acl $pfxPath
$acl.SetAccessRuleProtection($true, $false)  # Disable inheritance
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule("BUILTIN\Administrators", "FullControl", "Allow")
$acl.SetAccessRule($rule)
Set-Acl $pfxPath $acl

# Use encrypted password
$encPass = Get-Content "C:\SecureLocation\pfx-password.enc" | ConvertTo-SecureString
$plainPass = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($encPass))
$env:MSGRAPHPFXPASS = $plainPass

# Run tool
.\msgraphgolangtestingtool.exe -tenantid "..." -clientid "..." -pfx $pfxPath -mailbox "..." -action getevents
```

**Certificate Rotation Strategy:**
```powershell
# 1. Generate new certificate before old one expires
.\selfsignedcert.ps1  # Or request from CA

# 2. Upload new certificate to Azure AD App Registration
# (Azure Portal → App Registrations → Certificates & secrets)

# 3. Test with new certificate
.\msgraphgolangtestingtool.exe -thumbprint "NEW_THUMBPRINT" ...

# 4. Update production automation with new thumbprint
# 5. Remove old certificate from Azure AD after grace period (7-30 days)
```

---

## Least Privilege Principle

### API Permissions

**Only grant the minimum permissions required for your use case:**

| Action | Minimum Required Permission | Avoid Over-Permissioning |
|--------|----------------------------|---------------------------|
| `getevents` | `Calendars.Read` | ❌ Don't use `Calendars.ReadWrite` |
| `getinbox` | `Mail.Read` | ❌ Don't use `Mail.ReadWrite` or `Mail.ReadWrite.All` |
| `sendmail` | `Mail.Send` | ✅ Appropriate permission |
| `sendinvite` | `Calendars.ReadWrite` | ✅ Required for creating events |

**Multi-Action Scenarios:**

If you need multiple actions, grant only the necessary permissions:

```
Scenario: Read inbox and send emails
Required permissions:
  ✅ Mail.Read
  ✅ Mail.Send
  ❌ NOT Mail.ReadWrite.All (too broad)
```

**Avoid Wildcard Permissions:**
- ❌ `Mail.ReadWrite.All` - Grants access to ALL mailboxes
- ❌ `Calendars.ReadWrite.All` - Grants access to ALL calendars
- ✅ `Mail.Send` - Only allows sending (no read access)
- ✅ `Calendars.Read` - Only allows reading (no write access)

**Application vs. Delegated Permissions:**

This tool uses **Application permissions** (not Delegated):
- Application permissions work without a signed-in user
- Require admin consent
- Grant access to all mailboxes (unless restricted by Exchange policies)
- Should be monitored and audited regularly

**Restricting Application Access to Specific Mailboxes:**

Use Exchange Online Application Access Policies to limit which mailboxes the app can access:

```powershell
# Connect to Exchange Online
Connect-ExchangeOnline

# Create policy to restrict app to specific mailboxes only
New-ApplicationAccessPolicy -AppId "YOUR_CLIENT_ID" -PolicyScopeGroupId "allowed-mailboxes@example.com" -AccessRight RestrictAccess -Description "Restrict Graph tool to specific mailboxes"

# Test the policy
Test-ApplicationAccessPolicy -Identity "user@example.com" -AppId "YOUR_CLIENT_ID"
```

---

## Environment Variables Security

**Secure Usage:**
```powershell
# Set for current PowerShell session only (not permanent)
$env:MSGRAPHSECRET = "your-secret"
.\msgraphgolangtestingtool.exe -action getevents
Remove-Item Env:\MSGRAPHSECRET  # Clear after use

# For automation, use encrypted storage
$encSecret = Get-Content "C:\SecureLocation\secret.enc" | ConvertTo-SecureString
$env:MSGRAPHSECRET = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($encSecret))
```

**Insecure Practices to Avoid:**
```powershell
# ❌ WRONG - Sets permanent user environment variable (persists in registry)
[System.Environment]::SetEnvironmentVariable("MSGRAPHSECRET", "secret", "User")

# ❌ WRONG - Sets permanent machine environment variable (visible to all users)
[System.Environment]::SetEnvironmentVariable("MSGRAPHSECRET", "secret", "Machine")

# ❌ WRONG - Visible in process list and command history
.\msgraphgolangtestingtool.exe -secret "my-secret-value" ...
```

**Best Practice:**
- Use session-scoped environment variables (`$env:VAR = "value"`)
- Clear sensitive variables after use (`Remove-Item Env:\MSGRAPHSECRET`)
- Never set permanent environment variables for secrets
- Use secure credential storage (Azure Key Vault, encrypted files)

---

## Logging and Auditing

### CSV Logs

**What's Logged:**
- ✅ Timestamps of all operations
- ✅ Actions performed (getevents, sendmail, etc.)
- ✅ Mailbox accessed
- ✅ Email recipients (To/CC/BCC)
- ✅ Email subjects
- ✅ Success/failure status
- ❌ Secrets are NOT logged
- ❌ Email bodies are NOT logged
- ❌ Authentication tokens are NOT logged

**Log Location:**
```
%TEMP%\_msgraphgolangtestingtool_{action}_{date}.csv

Examples:
C:\Users\Admin\AppData\Local\Temp\_msgraphgolangtestingtool_sendmail_2026-01-04.csv
C:\Users\Admin\AppData\Local\Temp\_msgraphgolangtestingtool_getevents_2026-01-04.csv
```

**Log Retention Best Practices:**
```powershell
# Review logs periodically for unauthorized usage
Get-ChildItem "$env:TEMP\_msgraphgolangtestingtool_*.csv" |
    Sort-Object LastWriteTime -Descending |
    Import-Csv |
    Where-Object {$_.Status -ne "Success"} |
    Format-Table

# Archive logs to secure location (retention: 90 days recommended)
$archivePath = "C:\SecureArchive\GraphToolLogs"
Get-ChildItem "$env:TEMP\_msgraphgolangtestingtool_*.csv" |
    Where-Object {$_.LastWriteTime -lt (Get-Date).AddDays(-7)} |
    Move-Item -Destination $archivePath

# Delete old archives (after 90 days)
Get-ChildItem $archivePath |
    Where-Object {$_.LastWriteTime -lt (Get-Date).AddDays(-90)} |
    Remove-Item
```

**Centralized Logging (Enterprise):**
```powershell
# Send logs to central SIEM or log aggregation system
$csvFiles = Get-ChildItem "$env:TEMP\_msgraphgolangtestingtool_*.csv"
foreach ($file in $csvFiles) {
    # Parse and send to Splunk, ELK, Azure Log Analytics, etc.
    $logs = Import-Csv $file.FullName
    # Send-LogsToSIEM -Data $logs
}
```

---

### Verbose Mode Security

**Verbose mode is safe for troubleshooting:**
- Secrets are masked (shows first/last 4 characters: `secr********cret`)
- Tokens are truncated (shows first 10 characters: `eyJ0eXAi...`)
- Full credentials are never displayed

**Example verbose output:**
```
Authentication: Client Secret (secr********cret)
Token: eyJ0eXAi... (expires in 59m 59s)
```

**When to use verbose mode:**
- ✅ Troubleshooting authentication issues
- ✅ Verifying configuration
- ✅ Debugging API call failures
- ❌ Not needed in production automation (adds overhead)

---

## Network Security

### Proxy Configuration

**Secure Proxy Usage:**
```powershell
# Use corporate proxy for traffic monitoring and compliance
$env:MSGRAPHPROXY = "http://proxy.company.com:8080"
.\msgraphgolangtestingtool.exe -action getevents

# Proxy with authentication (if required)
# Configure Windows proxy settings via:
# Settings → Network & Internet → Proxy
```

**Best Practices:**
- ✅ Use HTTPS proxies when possible (tool supports HTTP/HTTPS proxy URLs)
- ✅ Monitor proxy logs for unauthorized usage
- ✅ Use authenticated proxies in enterprise environments
- ✅ Whitelist required Microsoft domains: `*.graph.microsoft.com`, `login.microsoftonline.com`
- ❌ Don't bypass corporate proxy policies
- ❌ Don't disable certificate validation

---

### TLS/SSL Security

**Built-in Security:**
- ✅ All Graph API calls use HTTPS (TLS 1.2+)
- ✅ Certificate validation is enforced
- ✅ Uses Go's standard crypto libraries
- ✅ No SSL/TLS configuration needed

**What NOT to do:**
- ❌ Never disable certificate validation
- ❌ Never use HTTP proxies for production (use HTTPS)
- ❌ Never accept self-signed certificates in production

---

## Access Control and Operational Security

### Script Deployment

**Restrict who can execute the tool:**

```powershell
# Set file permissions (Administrators and specific users only)
$toolPath = "C:\Tools\msgraphgolangtestingtool.exe"
$acl = Get-Acl $toolPath
$acl.SetAccessRuleProtection($true, $false)  # Remove inherited permissions

# Allow Administrators
$adminRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "BUILTIN\Administrators", "FullControl", "Allow"
)
$acl.SetAccessRule($adminRule)

# Allow specific automation account
$automationRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "DOMAIN\AutomationAccount", "ReadAndExecute", "Allow"
)
$acl.AddAccessRule($automationRule)

Set-Acl $toolPath $acl
```

**Audit Script Execution:**
```powershell
# Enable PowerShell script block logging (Group Policy or registry)
# This logs all PowerShell commands including tool execution

# View execution audit logs
Get-WinEvent -LogName "Microsoft-Windows-PowerShell/Operational" |
    Where-Object {$_.Message -like "*msgraphgolangtestingtool*"} |
    Select-Object TimeCreated, Message |
    Format-List
```

---

### Principle of Least Access

**Service Accounts for Automation:**
```powershell
# ✅ CORRECT - Dedicated service account
# Create dedicated Azure AD App Registration for each automation task
# Grant only required permissions
# Monitor usage via Azure AD Sign-in logs

# ❌ WRONG - Using personal admin account
# Don't use personal accounts with elevated privileges
# Don't share credentials across multiple automations
```

**Mailbox Access Control:**
```powershell
# Limit mailbox access using Exchange Application Access Policies
Connect-ExchangeOnline

# Create dedicated security group
New-DistributionGroup -Name "Graph-Tool-Allowed-Mailboxes" -Type "Security"
Add-DistributionGroupMember -Identity "Graph-Tool-Allowed-Mailboxes" -Member "automation@example.com"

# Restrict app to only access mailboxes in this group
New-ApplicationAccessPolicy -AppId "YOUR_CLIENT_ID" -PolicyScopeGroupId "Graph-Tool-Allowed-Mailboxes@example.com" -AccessRight RestrictAccess

# Verify policy
Test-ApplicationAccessPolicy -Identity "automation@example.com" -AppId "YOUR_CLIENT_ID"
# Should show: Access granted
Test-ApplicationAccessPolicy -Identity "random-user@example.com" -AppId "YOUR_CLIENT_ID"
# Should show: Access denied
```

---

## Monitoring and Alerting

### Azure AD Sign-in Logs

Monitor application usage in Azure AD:

1. Navigate to: Azure Portal → Azure AD → Enterprise Applications → Your App → Sign-in logs
2. Review:
   - Sign-in frequency
   - Source IP addresses
   - Success/failure rates
   - Anomalous activity

**Set up alerts for:**
- Failed authentication attempts (potential credential compromise)
- Sign-ins from unexpected IP addresses
- High volume of API calls (potential abuse)
- Permission changes to the application

---

### Microsoft 365 Audit Logs

Track mailbox and calendar operations:

1. Navigate to: Microsoft 365 Compliance Center → Audit → Search
2. Filter by:
   - User: Your app's mailbox access
   - Activities: `Send`, `MailItemsAccessed`, `Update`, `Create`
   - Date range: Last 7-90 days

**Create alerts for:**
- Mass email sending (potential spam/phishing)
- Unusual access patterns
- Operations outside business hours

---

## Incident Response

### If Credentials are Compromised

**Immediate Actions:**
1. **Revoke the compromised credential:**
   ```powershell
   # For Client Secret:
   # Azure Portal → App Registrations → Your App → Certificates & secrets → Delete secret

   # For Certificate:
   # Azure Portal → App Registrations → Your App → Certificates & secrets → Delete certificate
   ```

2. **Review audit logs for unauthorized usage:**
   ```powershell
   # Check Azure AD sign-in logs for suspicious activity
   # Check mailbox audit logs for unauthorized email sending
   ```

3. **Generate new credential:**
   ```powershell
   # Create new secret or certificate
   # Update automation scripts with new credential
   ```

4. **Notify security team and stakeholders**

5. **Document incident and lessons learned**

---

### If Malicious Activity Detected

**Immediate Actions:**
1. **Revoke application permissions:**
   - Azure Portal → App Registrations → Your App → API permissions → Remove permissions

2. **Disable the application:**
   - Azure Portal → Enterprise Applications → Your App → Properties → Enabled for users to sign-in? → No

3. **Investigate scope of breach:**
   - Review all sign-in logs
   - Review all mailbox audit logs
   - Identify affected mailboxes

4. **Remediate:**
   - Recall sent emails (if applicable)
   - Delete unauthorized calendar events
   - Notify affected users

5. **Re-enable with enhanced security:**
   - New credentials
   - Restricted mailbox access policies
   - Enhanced monitoring and alerting

---

## Compliance and Data Protection

### GDPR / Data Privacy

**Personal Data Handling:**
- Email addresses are logged in CSV files
- Email subjects may contain personal data
- CSV logs stored in local temp directory (not encrypted)

**Best Practices:**
- ✅ Encrypt CSV logs if they contain sensitive personal data
- ✅ Implement retention policies (delete logs after 90 days)
- ✅ Restrict access to CSV log files
- ✅ Include tool usage in data processing records (GDPR Article 30)
- ✅ Inform users if their emails are being processed by automation

---

### Regulatory Compliance

**For regulated industries (HIPAA, SOC 2, ISO 27001):**
- Document application usage in security policies
- Include in risk assessments
- Implement change management for updates
- Conduct regular security reviews
- Maintain audit trails (Azure AD logs + CSV logs)
- Encrypt data at rest and in transit (TLS enforced, consider CSV encryption)

---

## Security Checklist

Before deploying to production:

- [ ] Using certificates instead of client secrets
- [ ] Certificates stored in Windows Certificate Store (not PFX files)
- [ ] Minimum required API permissions granted
- [ ] Admin consent granted for permissions
- [ ] Application access policies configured (restricted mailboxes)
- [ ] Secrets/certificates rotated regularly (90-day schedule)
- [ ] Service account created (not personal account)
- [ ] File permissions restricted on executable
- [ ] CSV log retention policy implemented
- [ ] Azure AD sign-in monitoring enabled
- [ ] Alerts configured for anomalous activity
- [ ] Incident response plan documented
- [ ] Compliance requirements reviewed
- [ ] Security team notified of deployment

---

## Additional Resources

- **Microsoft Graph Security Best Practices:** https://learn.microsoft.com/graph/security-authorization
- **Azure AD Application Security:** https://learn.microsoft.com/azure/active-directory/develop/security-best-practices-for-app-registration
- **Exchange Application Access Policies:** https://learn.microsoft.com/graph/auth-limit-mailbox-access
- **TROUBLESHOOTING.md** - Troubleshooting common issues
- **README.md** - Usage guide and examples

---

*Last Updated: 2026-01-04 - Version 1.15.1*

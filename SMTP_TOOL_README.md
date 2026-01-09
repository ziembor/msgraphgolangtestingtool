# SMTP Connectivity Testing Tool

A comprehensive command-line tool for testing SMTP connectivity, TLS/SSL configuration, authentication, and email sending. Part of the **msgraphgolangtestingtool** suite.

## Overview

The **smtptool** provides production-grade SMTP diagnostics that go far beyond basic connectivity testing:

- **Comprehensive TLS Analysis**: Certificate chain validation, cipher suite assessment, protocol version detection
- **Exchange Server Detection**: Automatic detection of Microsoft Exchange with version mapping and targeted diagnostics
- **Authentication Testing**: Support for PLAIN, LOGIN, and CRAM-MD5 mechanisms
- **End-to-End Testing**: Complete email sending pipeline validation
- **CSV Logging**: All operations automatically logged for audit and troubleshooting

**Target Use Cases:**
- On-premises Exchange server connectivity validation
- TLS/SSL certificate troubleshooting
- SMTP relay configuration testing
- Email delivery pipeline diagnostics
- Security compliance verification (cipher strength, certificate expiration)

## Features

✅ **4 Comprehensive Actions**:
- `testconnect` - TCP connectivity and capability detection
- `teststarttls` - Comprehensive TLS/SSL diagnostics (certificates, ciphers, warnings)
- `testauth` - SMTP authentication validation
- `sendmail` - End-to-end email sending test

✅ **No External Dependencies**: Pure Go stdlib implementation
✅ **Cross-Platform**: Windows, Linux, macOS
✅ **CSV Logging**: Automatic logging to `%TEMP%\_smtptool_{action}_{date}.csv`
✅ **Exchange-Specific Diagnostics**: Targeted recommendations for on-premises Exchange

## Installation

### Build from Source

```powershell
# Build SMTP tool only
go build -C cmd/smtptool -ldflags="-s -w" -o smtptool.exe

# Or build both tools
.\build-all.ps1
```

See [BUILD.md](BUILD.md) for detailed build instructions.

## Quick Start

```powershell
# Test basic connectivity
.\smtptool.exe -action testconnect -host smtp.example.com -port 25

# Test TLS with comprehensive diagnostics
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587

# Test authentication
.\smtptool.exe -action testauth -host smtp.example.com -port 587 \
  -username user@example.com -password yourpassword

# Send test email
.\smtptool.exe -action sendmail -host smtp.example.com -port 587 \
  -username user@example.com -password yourpassword \
  -from sender@example.com -to recipient@example.com \
  -subject "Test Email" -body "This is a test message"
```

## Actions

### 1. testconnect - Basic Connectivity

Tests TCP connection and SMTP capabilities.

**What it does:**
- Establishes TCP connection to SMTP server
- Reads server banner (220 response)
- Sends EHLO command
- Parses and displays server capabilities (STARTTLS, AUTH, SIZE, 8BITMIME, etc.)
- Detects Microsoft Exchange servers (with version detection)
- Logs results to CSV

**Example:**
```powershell
.\smtptool.exe -action testconnect -host mail.contoso.com -port 25
```

**Output:**
```
Testing SMTP connectivity to mail.contoso.com:25...

✓ Connected successfully
  Banner: 220 mail.contoso.com Microsoft ESMTP MAIL Service ready

Server Capabilities:
  • STARTTLS
  • AUTH: PLAIN, LOGIN
  • SIZE: 35882577
  • 8BITMIME
  • PIPELINING

═══════════════════════════════════════════════════════════
  Microsoft Exchange Server Detected
═══════════════════════════════════════════════════════════
Version: Exchange 2019 (15.2.1118.30)
Banner:  220 mail.contoso.com Microsoft ESMTP MAIL Service ready

Exchange Capabilities:
  • Maximum message size: 35882577 bytes (34.22 MB)
  • Supported authentication: PLAIN, LOGIN
  • STARTTLS is supported
  • 8-bit MIME is supported
  • Command pipelining is supported

Exchange Notes:
  ⚠ Exchange typically restricts relay for unauthenticated connections
  ⚠ Authentication usually requires TLS on port 587
  ⚠ On-premises Exchange: Ensure proper SMTP connector configuration
═══════════════════════════════════════════════════════════

✓ Connectivity test completed successfully
```

### 2. teststarttls - Comprehensive TLS Diagnostics

Performs in-depth TLS/SSL testing with certificate chain analysis.

**What it does:**
- Connects to SMTP server
- Verifies STARTTLS capability
- Performs TLS handshake
- **Analyzes TLS connection**:
  * Protocol version (TLS 1.0, 1.1, 1.2, 1.3)
  * Cipher suite and strength assessment
  * Server Name Indication (SNI)
- **Analyzes certificate chain**:
  * Subject and Issuer
  * Serial number
  * Validity period (from/to dates)
  * Subject Alternative Names (SANs)
  * Key usage and extended key usage
  * Signature algorithm
  * Public key algorithm and size
  * Verification status (valid, expired, hostname_mismatch, self_signed)
  * Days until expiration
- **Generates warnings**:
  * Deprecated TLS versions (1.0, 1.1)
  * Weak cipher suites
  * Certificate expiration (expired or expiring soon)
  * Hostname mismatches
  * Self-signed certificates
  * Weak public keys (< 2048 bits)
- Tests encrypted connection (EHLO after STARTTLS)

**Example:**
```powershell
.\smtptool.exe -action teststarttls -host smtp.office365.com -port 587
```

**Output:**
```
Testing STARTTLS on smtp.office365.com:587...

✓ Connected
✓ STARTTLS capability available

Performing TLS handshake...
✓ TLS handshake successful

TLS Connection Details:
═══════════════════════════════════════════════════════════
  Protocol Version:    TLS 1.3
  Cipher Suite:        TLS_AES_256_GCM_SHA384
  Cipher Strength:     STRONG
  Server Name (SNI):   smtp.office365.com
═══════════════════════════════════════════════════════════

Certificate Information:
═══════════════════════════════════════════════════════════
  Subject:             CN=*.outlook.com
  Issuer:              CN=DigiCert Global G2 TLS RSA SHA256 2020 CA1
  Serial Number:       0A3F...D21E
  Valid From:          2024-11-19 00:00:00 UTC
  Valid To:            2025-12-10 23:59:59 UTC
  Days Until Expiry:   335
  Subject Alternative Names:
    • *.outlook.com
    • *.office365.com
    • smtp.office365.com
    • outlook.com
  Signature Algorithm: SHA256-RSA
  Public Key:          RSA (2048 bits)
  Key Usage:           DigitalSignature, KeyEncipherment
  Extended Key Usage:  ServerAuth, ClientAuth
  Verification:        VALID
  Chain Length:        3 certificate(s)
═══════════════════════════════════════════════════════════

✓ Testing encrypted connection...
  ✓ Encrypted connection working

✓ STARTTLS test completed successfully
```

**Advanced Options:**
```powershell
# Skip certificate verification (insecure, for testing only)
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587 -skipverify

# Specify minimum TLS version
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587 -tlsversion 1.3

# Verbose output
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587 -verbose
```

### 3. testauth - Authentication Testing

Tests SMTP authentication without sending email.

**What it does:**
- Connects to SMTP server
- Sends EHLO and detects supported AUTH mechanisms
- Upgrades to TLS if on port 25/587 and STARTTLS available
- Re-runs EHLO on encrypted connection
- Attempts authentication with specified credentials
- Supports PLAIN, LOGIN, and CRAM-MD5 mechanisms
- Logs authentication result

**Example:**
```powershell
.\smtptool.exe -action testauth \
  -host smtp.example.com -port 587 \
  -username user@example.com \
  -password "yourpassword"
```

**Output:**
```
Testing SMTP authentication on smtp.example.com:587...

✓ Connected
✓ Server supports AUTH mechanisms: PLAIN, LOGIN

Upgrading to TLS before authentication...
✓ TLS upgrade successful

Attempting authentication with method: PLAIN

✓ Authentication successful

✓ Authentication test completed successfully
```

**Options:**
```powershell
# Specify auth method explicitly
.\smtptool.exe -action testauth -host smtp.example.com -port 587 \
  -username user@example.com -password "secret" -authmethod CRAM-MD5

# Auto-select best method (default)
.\smtptool.exe -action testauth -host smtp.example.com -port 587 \
  -username user@example.com -password "secret" -authmethod auto
```

### 4. sendmail - End-to-End Email Sending

Sends a test email through the SMTP server.

**What it does:**
- Connects to SMTP server
- Sends EHLO
- Upgrades to TLS if on port 25/587 and STARTTLS available
- Authenticates if credentials provided
- Sends complete RFC 5322 formatted message
  * MAIL FROM command
  * RCPT TO command(s)
  * DATA command
  * Message headers (Message-ID, Date, From, To, Subject)
  * Message body
- Logs full transaction details

**Example:**
```powershell
.\smtptool.exe -action sendmail \
  -host smtp.example.com -port 587 \
  -username user@example.com -password "yourpassword" \
  -from sender@example.com \
  -to recipient@example.com \
  -subject "Test Email from smtptool" \
  -body "This is a test message to verify SMTP functionality."
```

**Multiple Recipients:**
```powershell
.\smtptool.exe -action sendmail \
  -host smtp.example.com -port 587 \
  -username user@example.com -password "secret" \
  -from sender@example.com \
  -to "recipient1@example.com,recipient2@example.com,recipient3@example.com" \
  -subject "Test Email" \
  -body "Message sent to multiple recipients"
```

**Output:**
```
Sending test email via smtp.example.com:587...

From:    sender@example.com
To:      recipient@example.com
Subject: Test Email from smtptool

✓ Connected
Upgrading to TLS...
✓ TLS upgrade successful
Authenticating...
✓ Authentication successful

Sending message...
✓ Message sent successfully
  Message-ID: <1736443200123456789.smtptool@smtp.example.com>

✓ Email sending test completed successfully
```

## Command-Line Flags

### Core Flags

| Flag | Description | Environment Variable | Default |
|------|-------------|---------------------|---------|
| `-action` | Action to perform (required) | `SMTPACTION` | - |
| `-host` | SMTP server hostname or IP (required) | `SMTPHOST` | - |
| `-port` | SMTP server port | `SMTPPORT` | 25 |
| `-timeout` | Connection timeout (seconds) | `SMTPTIMEOUT` | 30 |

### Authentication Flags

| Flag | Description | Environment Variable |
|------|-------------|---------------------|
| `-username` | SMTP username | `SMTPUSERNAME` |
| `-password` | SMTP password | `SMTPPASSWORD` |
| `-authmethod` | Auth method: PLAIN, LOGIN, CRAM-MD5, auto | `SMTPAUTHMETHOD` |

### Email Flags (sendmail action)

| Flag | Description | Environment Variable |
|------|-------------|---------------------|
| `-from` | Sender email address | `SMTPFROM` |
| `-to` | Recipient email addresses (comma-separated) | `SMTPTO` |
| `-subject` | Email subject | `SMTPSUBJECT` |
| `-body` | Email body text | `SMTPBODY` |

### TLS Flags

| Flag | Description | Environment Variable | Default |
|------|-------------|---------------------|---------|
| `-starttls` | Force STARTTLS usage | `SMTPSTARTTLS` | false (auto-detect) |
| `-skipverify` | Skip TLS certificate verification (insecure) | `SMTPSKIPVERIFY` | false |
| `-tlsversion` | Minimum TLS version: 1.2, 1.3 | `SMTPTLSVERSION` | 1.2 |

### Runtime Flags

| Flag | Description | Environment Variable | Default |
|------|-------------|---------------------|---------|
| `-verbose` | Enable verbose output | `SMTPVERBOSE` | false |
| `-loglevel` | Logging level: DEBUG, INFO, WARN, ERROR | `SMTPLOGLEVEL` | INFO |
| `-output` | Output format: text, json | `SMTPOUTPUT` | text |
| `-version` | Show version information | - | - |

## Environment Variables

All flags can be set via environment variables with the `SMTP` prefix:

```powershell
# Windows PowerShell
$env:SMTPHOST = "smtp.example.com"
$env:SMTPPORT = "587"
$env:SMTPUSERNAME = "user@example.com"
$env:SMTPPASSWORD = "yourpassword"

.\smtptool.exe -action testauth

# Linux/macOS Bash
export SMTPHOST="smtp.example.com"
export SMTPPORT="587"
export SMTPUSERNAME="user@example.com"
export SMTPPASSWORD="yourpassword"

./smtptool -action testauth
```

**Note:** Command-line flags take precedence over environment variables.

## CSV Logging

All operations are automatically logged to action-specific CSV files:

**Location:** `%TEMP%\_smtptool_{action}_{date}.csv`

**Examples:**
- `C:\Users\username\AppData\Local\Temp\_smtptool_testconnect_2026-01-09.csv`
- `C:\Users\username\AppData\Local\Temp\_smtptool_teststarttls_2026-01-09.csv`

### CSV Schemas

**testconnect:**
```
Timestamp, Action, Status, Server, Port, Connected, Banner, Capabilities, Exchange_Detected, Error
```

**teststarttls:**
```
Timestamp, Action, Status, Server, Port, STARTTLS_Available, TLS_Version, Cipher_Suite, Cert_Subject, Cert_Issuer, Cert_Valid_From, Cert_Valid_To, Cert_SANs, Verification_Status, Warnings, Error
```

**testauth:**
```
Timestamp, Action, Status, Server, Port, Username, Auth_Mechanisms_Available, Auth_Method_Used, Auth_Result, Error
```

**sendmail:**
```
Timestamp, Action, Status, Server, Port, From, To, Subject, SMTP_Response_Code, Message_ID, Error
```

## Common SMTP Ports

| Port | Usage | TLS |
|------|-------|-----|
| 25 | SMTP (server-to-server relay) | Optional STARTTLS |
| 587 | Message submission (client-to-server) | STARTTLS required |
| 465 | SMTP over implicit TLS (SMTPS) | Implicit TLS |
| 2525 | Alternative submission port | Optional STARTTLS |

**Recommendations:**
- **Port 587**: Use for authenticated mail submission with STARTTLS
- **Port 465**: Use for implicit TLS/SSL connections
- **Port 25**: Typically for server-to-server relay (may not allow authentication)

## Troubleshooting

### Connection Issues

**"Connection refused"**
- Verify server hostname and port
- Check firewall rules
- Ensure SMTP service is running

**"Connection timeout"**
- Increase `-timeout` value
- Check network connectivity (`ping`, `telnet`)
- Verify DNS resolution

### TLS Issues

**"STARTTLS not advertised by server"**
- Server may not support STARTTLS on this port
- Try port 587 (usually supports STARTTLS)
- Try port 465 (implicit TLS, doesn't use STARTTLS)

**"TLS handshake failed"**
- Check minimum TLS version: `-tlsversion 1.2`
- Verify server certificate is valid
- Use `-skipverify` temporarily to test (insecure)

**"Certificate hostname mismatch"**
- Server certificate doesn't match hostname
- Check SANs in certificate (use `teststarttls`)
- May need to connect using certificate's CN/SAN hostname

### Authentication Issues

**"Authentication failed"**
- Verify username and password
- Ensure STARTTLS completed before auth (on ports 25/587)
- Try different auth method: `-authmethod PLAIN` or `-authmethod LOGIN`
- Check if account is locked or password expired

**"No compatible authentication mechanism found"**
- Server may not support requested auth method
- Run `testconnect` to see available AUTH mechanisms
- Use `-authmethod auto` to auto-select best method

### Exchange-Specific Issues

**"Relay access denied" on Exchange**
- Authenticated connections usually required for relay
- Ensure proper SMTP connector configuration
- Check Exchange receive connector permissions
- Verify sender/recipient domains are accepted

**"Must issue STARTTLS first" on Exchange**
- Exchange typically requires TLS for authentication on port 587
- Let tool auto-upgrade (default behavior)
- Or use `-starttls` to force STARTTLS

## Security Best Practices

⚠ **Important Security Notes:**

1. **Password Safety**:
   - Never commit passwords to version control
   - Use environment variables for credentials
   - Consider using application-specific passwords

2. **Certificate Verification**:
   - Never use `-skipverify` in production
   - Only use for debugging with self-signed certificates
   - Always validate certificate hostnames

3. **TLS Configuration**:
   - Use TLS 1.2 or higher (`-tlsversion 1.2`)
   - Avoid deprecated TLS 1.0/1.1
   - Monitor cipher suite strength warnings

4. **Logging**:
   - CSV logs may contain sensitive information
   - Review and secure log files in `%TEMP%`
   - Consider log rotation for long-term operations

## Advanced Usage

### Test Complete SMTP Pipeline

```powershell
# 1. Test connectivity
.\smtptool.exe -action testconnect -host smtp.example.com -port 587

# 2. Verify TLS configuration
.\smtptool.exe -action teststarttls -host smtp.example.com -port 587

# 3. Test authentication
.\smtptool.exe -action testauth -host smtp.example.com -port 587 \
  -username user@example.com -password "secret"

# 4. Send test email
.\smtptool.exe -action sendmail -host smtp.example.com -port 587 \
  -username user@example.com -password "secret" \
  -from user@example.com -to admin@example.com \
  -subject "SMTP Pipeline Test" -body "All tests passed"
```

### Automated Testing Script

```powershell
# test-smtp.ps1
$host = "smtp.example.com"
$port = 587
$username = "user@example.com"
$password = "secret"

Write-Host "Testing SMTP connectivity..." -ForegroundColor Cyan
& .\smtptool.exe -action testconnect -host $host -port $port

if ($LASTEXITCODE -eq 0) {
    Write-Host "Testing TLS..." -ForegroundColor Cyan
    & .\smtptool.exe -action teststarttls -host $host -port $port
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "Testing authentication..." -ForegroundColor Cyan
    & .\smtptool.exe -action testauth -host $host -port $port `
        -username $username -password $password
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✓ All tests passed!" -ForegroundColor Green
} else {
    Write-Host "`n✗ Tests failed" -ForegroundColor Red
    exit 1
}
```

## Related Documentation

- **Build Instructions**: [BUILD.md](BUILD.md)
- **Microsoft Graph Tool**: [README.md](README.md)
- **Project Overview**: [CLAUDE.md](CLAUDE.md)
- **Security Policy**: [SECURITY.md](SECURITY.md)

## Support

**Issues and Feedback:**
- GitHub: [https://github.com/ziembor/msgraphgolangtestingtool/issues](https://github.com/ziembor/msgraphgolangtestingtool/issues)

                          ..ooOO END OOoo..

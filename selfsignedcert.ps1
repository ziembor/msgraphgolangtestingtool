# Self-Signed Certificate Generator for msgraphgolangtestingtool
# This script creates a self-signed certificate and exports both private and public keys

# Enable strict error handling
$ErrorActionPreference = "Stop"

try {
    # Customize the subject name if you want
    $certName = "MyPortableTool"

    Write-Host "Creating self-signed certificate: $certName" -ForegroundColor Cyan
    Write-Host ""

    # Create the certificate
    $cert = New-SelfSignedCertificate `
        -Subject "CN=$certName" `
        -CertStoreLocation "Cert:\CurrentUser\My" `
        -KeyExportPolicy Exportable `
        -KeySpec Signature `
        -KeyLength 2048 `
        -HashAlgorithm "SHA256" `
        -NotAfter (Get-Date).AddYears(2)

    if ($null -eq $cert) {
        throw "Failed to create certificate"
    }

    Write-Host "Certificate created successfully!" -ForegroundColor Green
    Write-Host "  Thumbprint: $($cert.Thumbprint)" -ForegroundColor Yellow
    Write-Host "  Subject: $($cert.Subject)" -ForegroundColor Yellow
    Write-Host "  Valid Until: $($cert.NotAfter)" -ForegroundColor Yellow
    Write-Host ""

    # Set a password for the PFX file
    $password = Read-Host -AsSecureString "Set a password for the PFX file"

    # Validate password is not empty
    $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
    $plainPassword = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($BSTR)

    if ([string]::IsNullOrWhiteSpace($plainPassword)) {
        throw "Password cannot be empty"
    }

    Write-Host ""
    Write-Host "Exporting certificate files..." -ForegroundColor Cyan

    # Export the private key (PFX)
    $pfxPath = ".\$certName.pfx"
    $pfxResult = Export-PfxCertificate `
        -Cert $cert `
        -FilePath $pfxPath `
        -Password $password

    if ($null -eq $pfxResult) {
        throw "Failed to export PFX file"
    }

    Write-Host "  Private key (PFX): $pfxPath" -ForegroundColor Green

    # Export the public key (CER) - this is what you upload to Azure AD
    $cerPath = ".\$certName.cer"
    $cerResult = Export-Certificate `
        -Cert $cert `
        -FilePath $cerPath `
        -Type CERT

    if ($null -eq $cerResult) {
        throw "Failed to export CER file"
    }

    Write-Host "  Public key (CER):  $cerPath" -ForegroundColor Green
    Write-Host ""
    Write-Host "SUCCESS! Certificate files created:" -ForegroundColor Green
    Write-Host "  1. Upload '$cerPath' to your Azure AD App Registration" -ForegroundColor White
    Write-Host "  2. Use '$pfxPath' with the -pfx flag for authentication" -ForegroundColor White
    Write-Host "  3. Or use thumbprint '$($cert.Thumbprint)' with the -thumbprint flag" -ForegroundColor White
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Certificate generation failed. Please check the error above." -ForegroundColor Red
    exit 1
}

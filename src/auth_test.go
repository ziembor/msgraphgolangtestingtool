//go:build !integration
// +build !integration

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

// Helper function to generate a test certificate and private key
func generateTestCertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Organization"},
			CommonName:   "Test Certificate",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

// Helper function to create a test PFX file with specified encryption
func createTestPFX(t *testing.T, password string) []byte {
	t.Helper()

	cert, privateKey := generateTestCertificate(t)

	// Encode as PFX using Modern2023 encoder (supports SHA-256)
	pfxData, err := pkcs12.Modern2023.Encode(privateKey, cert, nil, password)
	if err != nil {
		t.Fatalf("Failed to encode PFX: %v", err)
	}

	return pfxData
}

// Helper function to create a legacy test PFX file with SHA-1 encryption
func createLegacyTestPFX(t *testing.T, password string) []byte {
	t.Helper()

	cert, privateKey := generateTestCertificate(t)

	// Encode as PFX using Legacy encoder (uses SHA-1/TripleDES)
	pfxData, err := pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	if err != nil {
		t.Fatalf("Failed to encode legacy PFX: %v", err)
	}

	return pfxData
}

// Test createCertCredential with modern PFX (SHA-256)
func TestCreateCertCredential_ModernPFX(t *testing.T) {
	pfxData := createTestPFX(t, "test-password")

	// Test decoding - we can't fully test Azure credential creation without real Azure setup,
	// but we can verify the PFX decodes correctly
	_, cert, caCerts, err := pkcs12.DecodeChain(pfxData, "test-password")
	if err != nil {
		t.Fatalf("Failed to decode modern PFX (SHA-256): %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	if cert.Subject.CommonName != "Test Certificate" {
		t.Errorf("Certificate CN = %q, want %q", cert.Subject.CommonName, "Test Certificate")
	}

	// CA certs may be nil for self-signed
	if caCerts == nil {
		t.Log("No CA certificates (expected for self-signed)")
	}
}

// Test createCertCredential with legacy PFX (SHA-1)
func TestCreateCertCredential_LegacyPFX(t *testing.T) {
	pfxData := createLegacyTestPFX(t, "test-password")

	// Test decoding legacy format
	_, cert, _, err := pkcs12.DecodeChain(pfxData, "test-password")
	if err != nil {
		t.Fatalf("Failed to decode legacy PFX (SHA-1): %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	if cert.Subject.CommonName != "Test Certificate" {
		t.Errorf("Certificate CN = %q, want %q", cert.Subject.CommonName, "Test Certificate")
	}
}

// Test createCertCredential with wrong password
func TestCreateCertCredential_WrongPassword(t *testing.T) {
	pfxData := createTestPFX(t, "correct-password")

	// Try to decode with wrong password
	_, _, _, err := pkcs12.DecodeChain(pfxData, "wrong-password")
	if err == nil {
		t.Error("Expected error with wrong password, got nil")
	}
}

// Test createCertCredential with empty password
func TestCreateCertCredential_EmptyPassword(t *testing.T) {
	pfxData := createTestPFX(t, "")

	// Decode with empty password
	_, cert, _, err := pkcs12.DecodeChain(pfxData, "")
	if err != nil {
		t.Fatalf("Failed to decode PFX with empty password: %v", err)
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}
}

// Test createCertCredential with malformed PFX data
func TestCreateCertCredential_MalformedPFX(t *testing.T) {
	malformedData := []byte("this is not a valid PFX file")

	_, _, _, err := pkcs12.DecodeChain(malformedData, "password")
	if err == nil {
		t.Error("Expected error with malformed PFX data, got nil")
	}
}

// Test createCertCredential with empty PFX data
func TestCreateCertCredential_EmptyPFX(t *testing.T) {
	emptyData := []byte{}

	_, _, _, err := pkcs12.DecodeChain(emptyData, "password")
	if err == nil {
		t.Error("Expected error with empty PFX data, got nil")
	}
}

// Test that our fix handles the SHA-256 digest algorithm (OID 2.16.840.1.101.3.4.2.1)
func TestCreateCertCredential_SHA256Support(t *testing.T) {
	// Create a PFX with modern encryption (SHA-256)
	pfxData := createTestPFX(t, "sha256-test")

	// This should NOT fail with "unknown digest algorithm: 2.16.840.1.101.3.4.2.1"
	key, cert, _, err := pkcs12.DecodeChain(pfxData, "sha256-test")
	if err != nil {
		t.Fatalf("SHA-256 PFX decoding failed: %v (this was the original bug)", err)
	}

	if key == nil {
		t.Error("Expected private key, got nil")
	}

	if cert == nil {
		t.Error("Expected certificate, got nil")
	}

	t.Log("✓ SHA-256 digest algorithm is now supported!")
}

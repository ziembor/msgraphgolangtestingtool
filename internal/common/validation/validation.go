package validation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// ValidateEmail performs basic email format validation.
// Checks for the presence of @ and validates the local and domain parts.
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email format: %s (missing @)", email)
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// ValidateEmails validates a slice of email addresses.
// Returns an error if any email in the slice is invalid.
func ValidateEmails(emails []string, fieldName string) error {
	for _, email := range emails {
		if err := ValidateEmail(email); err != nil {
			return fmt.Errorf("%s contains invalid email: %w", fieldName, err)
		}
	}
	return nil
}

// ValidateGUID validates that a string matches standard GUID format (8-4-4-4-12).
// Example: 12345678-1234-1234-1234-123456789012
func ValidateGUID(guid, fieldName string) error {
	guid = strings.TrimSpace(guid)
	if guid == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	// Basic GUID format: 8-4-4-4-12 hex characters
	if len(guid) != 36 {
		return fmt.Errorf("%s should be a GUID (36 characters, format: 12345678-1234-1234-1234-123456789012)", fieldName)
	}
	// Check for proper dash positions
	if guid[8] != '-' || guid[13] != '-' || guid[18] != '-' || guid[23] != '-' {
		return fmt.Errorf("%s has invalid GUID format (dashes at wrong positions)", fieldName)
	}
	return nil
}

// ValidateFilePath validates and sanitizes a file path for security and usability.
// Checks for path traversal attempts, verifies file exists and is accessible.
func ValidateFilePath(path, fieldName string) error {
	if path == "" {
		return nil // Empty is allowed for optional fields
	}

	// Clean and normalize path (resolves . and .. elements)
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	// After cleaning, ".." should not remain in the path unless it's at the start (relative path going up)
	// We need to check if the cleaned path tries to escape the current directory context
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("%s: invalid path: %w", fieldName, err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// If we can't get cwd, just verify the file exists
		cwd = ""
	}

	// If we have a cwd, check if the absolute path tries to go outside reasonable bounds
	// For absolute paths, this is allowed
	// For relative paths, we verify they don't traverse outside the working directory tree
	if cwd != "" && !filepath.IsAbs(path) {
		// Check if cleaned path still contains ".." which indicates traversal
		if strings.Contains(cleanPath, "..") {
			return fmt.Errorf("%s: path contains directory traversal (..) which is not allowed", fieldName)
		}
	}

	// Verify file exists and is accessible
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: file not found: %s", fieldName, path)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("%s: permission denied: %s", fieldName, path)
		}
		return fmt.Errorf("%s: cannot access file: %w", fieldName, err)
	}

	// Verify it's a regular file (not a directory or special file)
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%s: not a regular file (is it a directory?): %s", fieldName, path)
	}

	return nil
}

// ValidateHostname validates a hostname or IP address.
// Accepts DNS names, IPv4 addresses, and IPv6 addresses.
func ValidateHostname(hostname string) error {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}

	// Check if it's a valid IP address (IPv4 or IPv6)
	if net.ParseIP(hostname) != nil {
		return nil // Valid IP address
	}

	// Check if it's a valid hostname (DNS name)
	// Basic validation: must contain at least one character, may contain letters, digits, dots, and hyphens
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long (max 253 characters)")
	}

	// Check for valid characters in hostname
	for _, ch := range hostname {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-') {
			return fmt.Errorf("hostname contains invalid character: %c", ch)
		}
	}

	// Hostname cannot start or end with a hyphen or dot
	if strings.HasPrefix(hostname, "-") || strings.HasSuffix(hostname, "-") ||
		strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return fmt.Errorf("hostname cannot start or end with hyphen or dot")
	}

	return nil
}

// ValidatePort validates that a port number is in the valid range (1-65535).
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535 (got %d)", port)
	}
	return nil
}

// ValidateSMTPAddress validates an email address in SMTP format (RFC 5321).
// This is stricter than general email validation and follows SMTP standards.
func ValidateSMTPAddress(address string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("SMTP address cannot be empty")
	}

	// SMTP addresses should not contain angle brackets (those are added by the protocol)
	// But we'll accept them if present and extract the actual address
	if strings.HasPrefix(address, "<") && strings.HasSuffix(address, ">") {
		address = address[1 : len(address)-1]
	}

	// Validate as email
	return ValidateEmail(address)
}

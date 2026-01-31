// Package security provides security-related utilities for the gomailtesttool suite.
// It includes functions for safely masking sensitive data in logs and output.
package security

// MaskUsername masks a username for safe logging.
// Shows first 2 and last 2 characters with **** in between.
// Short usernames (4 characters or less) are fully masked.
func MaskUsername(username string) string {
	if len(username) <= 4 {
		return "****"
	}
	return username[:2] + "****" + username[len(username)-2:]
}

// MaskPassword masks a password for safe logging.
// Shows first 2 and last 2 characters with **** in between.
// Short passwords (4 characters or less) are fully masked.
// Empty passwords return empty string.
func MaskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}

// MaskAccessToken masks an access token for safe logging.
// Shows first 8 and last 4 characters with ... in between for long tokens.
// For shorter tokens, shows half on each side.
// Empty tokens return empty string.
func MaskAccessToken(token string) string {
	if len(token) == 0 {
		return ""
	}
	if len(token) <= 16 {
		return token[:len(token)/2] + "..." + token[len(token)/2:]
	}
	return token[:8] + "..." + token[len(token)-4:]
}

// MaskSecret masks a client secret or similar credential.
// Shows first 4 characters followed by asterisks indicating length.
// Empty secrets return empty string.
func MaskSecret(secret string) string {
	if len(secret) == 0 {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:4] + "****"
}

// MaskGUID masks a GUID/UUID for safe logging.
// Shows first 8 characters followed by asterisks.
// Preserves enough to identify the resource while hiding the full value.
func MaskGUID(guid string) string {
	if len(guid) <= 8 {
		return guid + "****"
	}
	return guid[:8] + "****"
}

// MaskEmail masks an email address for safe logging.
// Shows first 2 characters of local part and domain.
// Example: "user@example.com" becomes "us****@ex****"
func MaskEmail(email string) string {
	if len(email) == 0 {
		return ""
	}

	// Find @ symbol
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}

	if atIndex == -1 {
		// No @ found, treat as username
		return MaskUsername(email)
	}

	localPart := email[:atIndex]
	domain := email[atIndex+1:]

	maskedLocal := "****"
	if len(localPart) > 2 {
		maskedLocal = localPart[:2] + "****"
	}

	maskedDomain := "****"
	if len(domain) > 2 {
		maskedDomain = domain[:2] + "****"
	}

	return maskedLocal + "@" + maskedDomain
}

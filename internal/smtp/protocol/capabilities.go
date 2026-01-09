package protocol

import (
	"fmt"
	"strings"
)

// Capabilities represents SMTP server capabilities returned by EHLO command.
// The map key is the capability name (e.g., "STARTTLS", "AUTH", "SIZE").
// The map value is a slice of parameters for that capability.
//
// Example EHLO response:
//
//	250-smtp.example.com Hello
//	250-STARTTLS
//	250-AUTH PLAIN LOGIN CRAM-MD5
//	250-SIZE 35882577
//	250 8BITMIME
//
// Would be parsed as:
//
//	{
//	  "STARTTLS": [],
//	  "AUTH": ["PLAIN", "LOGIN", "CRAM-MD5"],
//	  "SIZE": ["35882577"],
//	  "8BITMIME": []
//	}
type Capabilities map[string][]string

// ParseCapabilities parses EHLO response lines into a Capabilities map.
// The first line (greeting) is typically skipped as it doesn't contain capabilities.
func ParseCapabilities(lines []string) Capabilities {
	caps := make(Capabilities)

	for i, line := range lines {
		// Skip the first line (greeting: "smtp.example.com Hello")
		if i == 0 {
			continue
		}

		// Split line into capability name and parameters
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		capName := strings.ToUpper(parts[0])
		capParams := []string{}

		if len(parts) > 1 {
			capParams = parts[1:]
		}

		caps[capName] = capParams
	}

	return caps
}

// Has checks if a specific capability is supported.
// Capability names are case-insensitive.
func (c Capabilities) Has(capability string) bool {
	_, exists := c[strings.ToUpper(capability)]
	return exists
}

// Get retrieves the parameters for a specific capability.
// Returns an empty slice if the capability doesn't exist.
func (c Capabilities) Get(capability string) []string {
	params, exists := c[strings.ToUpper(capability)]
	if !exists {
		return []string{}
	}
	return params
}

// GetAuthMechanisms extracts supported authentication mechanisms from AUTH capability.
// Returns a slice of mechanism names (e.g., ["PLAIN", "LOGIN", "CRAM-MD5"]).
func (c Capabilities) GetAuthMechanisms() []string {
	return c.Get("AUTH")
}

// GetMaxMessageSize extracts the maximum message size from SIZE capability.
// Returns 0 if SIZE capability is not present or has no parameter.
func (c Capabilities) GetMaxMessageSize() int64 {
	sizeParams := c.Get("SIZE")
	if len(sizeParams) == 0 {
		return 0
	}

	// Parse size parameter
	var size int64
	_, err := parseSize(sizeParams[0])
	if err != nil {
		return 0
	}

	return size
}

// SupportsSTARTTLS checks if the server supports STARTTLS command.
func (c Capabilities) SupportsSTARTTLS() bool {
	return c.Has("STARTTLS")
}

// SupportsAuth checks if the server supports SMTP authentication.
func (c Capabilities) SupportsAuth() bool {
	return c.Has("AUTH")
}

// Supports8BITMIME checks if the server supports 8-bit MIME encoding.
func (c Capabilities) Supports8BITMIME() bool {
	return c.Has("8BITMIME")
}

// SupportsPipelining checks if the server supports command pipelining.
func (c Capabilities) SupportsPipelining() bool {
	return c.Has("PIPELINING")
}

// SupportsChunking checks if the server supports CHUNKING extension.
func (c Capabilities) SupportsChunking() bool {
	return c.Has("CHUNKING")
}

// SupportsSMTPUTF8 checks if the server supports UTF-8 email addresses.
func (c Capabilities) SupportsSMTPUTF8() bool {
	return c.Has("SMTPUTF8")
}

// String returns a formatted string representation of all capabilities.
func (c Capabilities) String() string {
	var result []string
	for cap, params := range c {
		if len(params) > 0 {
			result = append(result, cap+": "+strings.Join(params, ", "))
		} else {
			result = append(result, cap)
		}
	}
	return strings.Join(result, "; ")
}

// parseSize parses a size parameter (handles both numeric and unit suffixes).
func parseSize(s string) (int64, error) {
	// Simple numeric parsing for now
	var size int64
	_, err := fmt.Sscanf(s, "%d", &size)
	return size, err
}

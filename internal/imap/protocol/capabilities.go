// Package protocol provides IMAP capability parsing utilities.
package protocol

import (
	"strings"
)

// Common IMAP capabilities
const (
	CapabilityIMAP4rev1  = "IMAP4rev1"
	CapabilityIMAP4rev2  = "IMAP4rev2"
	CapabilitySTARTTLS   = "STARTTLS"
	CapabilityLOGINDISABLED = "LOGINDISABLED"
	CapabilityAUTH       = "AUTH="
	CapabilityIDLE       = "IDLE"
	CapabilityNAMESPACE  = "NAMESPACE"
	CapabilityQUOTA      = "QUOTA"
	CapabilitySORT       = "SORT"
	CapabilitySEARCH     = "SEARCH"
	CapabilityTHREAD     = "THREAD"
	CapabilityMOVE       = "MOVE"
	CapabilityUNSELECT   = "UNSELECT"
	CapabilityUIDPLUS    = "UIDPLUS"
	CapabilityCONDSTORE  = "CONDSTORE"
	CapabilityQRESYNC    = "QRESYNC"
	CapabilityLITERALPLUS = "LITERAL+"
	CapabilityLITERALMINUS = "LITERAL-"
	CapabilitySASLIR     = "SASL-IR"
	CapabilityID         = "ID"
	CapabilityENABLE     = "ENABLE"
)

// Capabilities represents IMAP server capabilities.
type Capabilities struct {
	// Raw capabilities list
	raw []string

	// Parsed set of capabilities (uppercase)
	caps map[string]bool

	// Auth mechanisms (from AUTH= capabilities)
	authMechanisms []string
}

// NewCapabilities creates a new Capabilities from a list of capability strings.
func NewCapabilities(caps []string) *Capabilities {
	c := &Capabilities{
		raw:  caps,
		caps: make(map[string]bool),
	}
	c.parse()
	return c
}

// parse parses the raw capabilities.
func (c *Capabilities) parse() {
	for _, cap := range c.raw {
		capUpper := strings.ToUpper(cap)
		c.caps[capUpper] = true

		// Extract AUTH mechanisms
		if strings.HasPrefix(capUpper, "AUTH=") {
			mechanism := strings.TrimPrefix(cap, "AUTH=")
			mechanism = strings.TrimPrefix(mechanism, "auth=")
			c.authMechanisms = append(c.authMechanisms, mechanism)
		}
	}
}

// Has returns true if the server advertises the given capability.
func (c *Capabilities) Has(name string) bool {
	return c.caps[strings.ToUpper(name)]
}

// All returns all capability strings.
func (c *Capabilities) All() []string {
	return c.raw
}

// String returns a comma-separated list of capabilities.
func (c *Capabilities) String() string {
	return strings.Join(c.raw, ", ")
}

// SupportsIMAP4rev1 returns true if the server supports IMAP4rev1.
func (c *Capabilities) SupportsIMAP4rev1() bool {
	return c.Has(CapabilityIMAP4rev1)
}

// SupportsIMAP4rev2 returns true if the server supports IMAP4rev2.
func (c *Capabilities) SupportsIMAP4rev2() bool {
	return c.Has(CapabilityIMAP4rev2)
}

// SupportsSTARTTLS returns true if the server supports STARTTLS.
func (c *Capabilities) SupportsSTARTTLS() bool {
	return c.Has(CapabilitySTARTTLS)
}

// IsLoginDisabled returns true if LOGIN is disabled (usually pre-TLS).
func (c *Capabilities) IsLoginDisabled() bool {
	return c.Has(CapabilityLOGINDISABLED)
}

// GetAuthMechanisms returns the list of supported SASL mechanisms.
func (c *Capabilities) GetAuthMechanisms() []string {
	return c.authMechanisms
}

// SupportsAuth returns true if any AUTH mechanism is supported.
func (c *Capabilities) SupportsAuth() bool {
	return len(c.authMechanisms) > 0
}

// SupportsXOAUTH2 returns true if XOAUTH2 is supported.
func (c *Capabilities) SupportsXOAUTH2() bool {
	for _, mech := range c.authMechanisms {
		if strings.EqualFold(mech, "XOAUTH2") {
			return true
		}
	}
	return false
}

// SupportsPlain returns true if PLAIN authentication is supported.
func (c *Capabilities) SupportsPlain() bool {
	for _, mech := range c.authMechanisms {
		if strings.EqualFold(mech, "PLAIN") {
			return true
		}
	}
	return false
}

// SupportsLogin returns true if LOGIN authentication is supported.
// Note: This is different from AUTH=LOGIN - it checks if direct LOGIN command works.
func (c *Capabilities) SupportsLogin() bool {
	// LOGIN is available unless explicitly disabled
	return !c.IsLoginDisabled()
}

// SupportsIDLE returns true if the IDLE extension is supported.
func (c *Capabilities) SupportsIDLE() bool {
	return c.Has(CapabilityIDLE)
}

// SupportsNAMESPACE returns true if the NAMESPACE extension is supported.
func (c *Capabilities) SupportsNAMESPACE() bool {
	return c.Has(CapabilityNAMESPACE)
}

// SupportsQUOTA returns true if the QUOTA extension is supported.
func (c *Capabilities) SupportsQUOTA() bool {
	return c.Has(CapabilityQUOTA)
}

// SupportsSORT returns true if the SORT extension is supported.
func (c *Capabilities) SupportsSORT() bool {
	return c.Has(CapabilitySORT)
}

// SupportsMOVE returns true if the MOVE extension is supported.
func (c *Capabilities) SupportsMOVE() bool {
	return c.Has(CapabilityMOVE)
}

// SupportsUIDPLUS returns true if the UIDPLUS extension is supported.
func (c *Capabilities) SupportsUIDPLUS() bool {
	return c.Has(CapabilityUIDPLUS)
}

// SupportsCONDSTORE returns true if the CONDSTORE extension is supported.
func (c *Capabilities) SupportsCONDSTORE() bool {
	return c.Has(CapabilityCONDSTORE)
}

// SupportsSASLIR returns true if SASL Initial Response is supported.
// This allows sending the initial auth response with the AUTH command.
func (c *Capabilities) SupportsSASLIR() bool {
	return c.Has(CapabilitySASLIR)
}

// SupportsID returns true if the ID extension is supported.
func (c *Capabilities) SupportsID() bool {
	return c.Has(CapabilityID)
}

// SupportsENABLE returns true if the ENABLE extension is supported.
func (c *Capabilities) SupportsENABLE() bool {
	return c.Has(CapabilityENABLE)
}

// SelectBestAuthMechanism selects the best available auth mechanism.
// Priority: XOAUTH2 (if token provided) > PLAIN > LOGIN
func (c *Capabilities) SelectBestAuthMechanism(hasAccessToken bool) string {
	if hasAccessToken && c.SupportsXOAUTH2() {
		return "XOAUTH2"
	}
	if c.SupportsPlain() {
		return "PLAIN"
	}
	// Check for AUTH=LOGIN
	for _, mech := range c.authMechanisms {
		if strings.EqualFold(mech, "LOGIN") {
			return "LOGIN"
		}
	}
	// Fallback to direct LOGIN command if available
	if c.SupportsLogin() {
		return "LOGIN"
	}
	return ""
}

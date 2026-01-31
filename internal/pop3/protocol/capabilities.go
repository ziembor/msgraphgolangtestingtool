// Package protocol provides POP3 capability parsing utilities.
package protocol

import (
	"strings"
)

// Capabilities represents POP3 server capabilities from the CAPA command.
type Capabilities struct {
	// Raw capability lines
	raw []string

	// Parsed capabilities map (capability name -> arguments)
	caps map[string][]string
}

// NewCapabilities creates a new Capabilities from CAPA response lines.
func NewCapabilities(lines []string) *Capabilities {
	c := &Capabilities{
		raw:  lines,
		caps: make(map[string][]string),
	}
	c.parse()
	return c
}

// parse parses the raw capability lines into the caps map.
func (c *Capabilities) parse() {
	for _, line := range c.raw {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToUpper(parts[0])
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		c.caps[name] = args
	}
}

// Has returns true if the server advertises the given capability.
func (c *Capabilities) Has(name string) bool {
	_, ok := c.caps[strings.ToUpper(name)]
	return ok
}

// Get returns the arguments for a capability, or nil if not present.
func (c *Capabilities) Get(name string) []string {
	return c.caps[strings.ToUpper(name)]
}

// All returns all capability names.
func (c *Capabilities) All() []string {
	var names []string
	for name := range c.caps {
		names = append(names, name)
	}
	return names
}

// Raw returns the raw capability lines.
func (c *Capabilities) Raw() []string {
	return c.raw
}

// String returns a string representation of capabilities.
func (c *Capabilities) String() string {
	return strings.Join(c.All(), ", ")
}

// SupportsSTLS returns true if the server supports STLS (STARTTLS).
func (c *Capabilities) SupportsSTLS() bool {
	return c.Has("STLS")
}

// SupportsAuth returns true if the server supports SASL authentication.
func (c *Capabilities) SupportsAuth() bool {
	return c.Has("SASL")
}

// GetAuthMechanisms returns the list of supported SASL mechanisms.
// Returns nil if SASL is not supported.
func (c *Capabilities) GetAuthMechanisms() []string {
	return c.Get("SASL")
}

// SupportsUIDL returns true if the server supports UIDL.
func (c *Capabilities) SupportsUIDL() bool {
	return c.Has("UIDL")
}

// SupportsTOP returns true if the server supports TOP.
func (c *Capabilities) SupportsTOP() bool {
	return c.Has("TOP")
}

// SupportsUSER returns true if the server supports USER/PASS authentication.
func (c *Capabilities) SupportsUSER() bool {
	return c.Has("USER")
}

// SupportsPipelining returns true if the server supports command pipelining.
func (c *Capabilities) SupportsPipelining() bool {
	return c.Has("PIPELINING")
}

// SupportsRESPCodes returns true if the server returns extended response codes.
func (c *Capabilities) SupportsRESPCodes() bool {
	return c.Has("RESP-CODES")
}

// SupportsXOAUTH2 returns true if the server supports XOAUTH2.
func (c *Capabilities) SupportsXOAUTH2() bool {
	mechanisms := c.GetAuthMechanisms()
	for _, m := range mechanisms {
		if strings.EqualFold(m, "XOAUTH2") {
			return true
		}
	}
	return false
}

// SupportsPlain returns true if the server supports PLAIN authentication.
func (c *Capabilities) SupportsPlain() bool {
	mechanisms := c.GetAuthMechanisms()
	for _, m := range mechanisms {
		if strings.EqualFold(m, "PLAIN") {
			return true
		}
	}
	return false
}

// GetExpirePolicy returns the EXPIRE policy if advertised.
// Format: EXPIRE <days> or EXPIRE NEVER
func (c *Capabilities) GetExpirePolicy() string {
	args := c.Get("EXPIRE")
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// GetImplementation returns the IMPLEMENTATION string if advertised.
func (c *Capabilities) GetImplementation() string {
	args := c.Get("IMPLEMENTATION")
	if len(args) > 0 {
		return strings.Join(args, " ")
	}
	return ""
}

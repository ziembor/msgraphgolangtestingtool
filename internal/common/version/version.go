package version

import (
	_ "embed"
	"strings"
)

// Version information embedded from VERSION file
// This package provides centralized version management for all tools in the repository.
// The VERSION file is located at src/VERSION and is embedded at compile time.

//go:embed ../../../src/VERSION
var versionRaw string

// Version is the current version of the tool suite, trimmed of whitespace.
// All tools (msgraphtool, smtptool) share the same version number.
var Version = strings.TrimSpace(versionRaw)

// Get returns the current version string.
// This is a convenience function for accessing the Version variable.
func Get() string {
	return Version
}

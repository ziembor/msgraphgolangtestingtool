package main

import (
	_ "embed"
	"strings"
)

// Version information embedded from VERSION file
// This is shared across all build configurations (regular and integration builds)

//go:embed VERSION
var versionRaw string

// version is the current version of the tool, trimmed of whitespace
var version = strings.TrimSpace(versionRaw)

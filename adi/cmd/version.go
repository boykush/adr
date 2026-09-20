package cmd

import "runtime/debug"

// adiVersion reports the build version, falling back to "(unknown)" outside a
// versioned build (e.g. `go run`). It reaches the consuming agent through the
// MCP handshake, which is the only surface that asks.
func adiVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(unknown)"
}

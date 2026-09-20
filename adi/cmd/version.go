package cmd

import "runtime/debug"

// version is stamped with -ldflags by builds that carry no module version of
// their own: the image builds from a source copy with no VCS metadata, and a
// server that reports "(unknown)" in every handshake is no use.
var version string

// adiVersion reports the build version, falling back to "(unknown)" outside a
// versioned build (e.g. `go run`). It reaches the consuming agent through the
// MCP handshake, which is the only surface that asks.
func adiVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(unknown)"
}

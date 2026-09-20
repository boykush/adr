package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of adi",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("adi version %s\n", adiVersion())
	},
}

// adiVersion reports the build version, falling back to "(unknown)" outside a
// versioned build (e.g. `go run`).
func adiVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(unknown)"
}

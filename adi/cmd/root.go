package cmd

import (
	"github.com/spf13/cobra"
)

// modelDir is where the decisions are read from. It is a flag rather than
// anything baked in so the tool carries no decisions of its own and can be
// pointed at a model that ships separately from the binary.
var modelDir string

func init() {
	rootCmd.PersistentFlags().StringVar(&modelDir, "model", "decisions", "path to the decision model directory")
}

var rootCmd = &cobra.Command{
	Use:   "adi",
	Short: "Architectural Decision Injection - serve the decisions that bind to the work at hand",
}

func Execute() error {
	return rootCmd.Execute()
}

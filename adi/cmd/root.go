package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "adi",
	Short: "Architectural Decision Injection - serve the decisions that bind to the work at hand",
}

func Execute() error {
	return rootCmd.Execute()
}

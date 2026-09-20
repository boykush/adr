package cmd

import (
	"fmt"

	"github.com/boykush/adr/adi/internal/decision"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the decisions in the model",
	Args:  cobra.NoArgs,
	// A model that fails to read is not a misuse of the command.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		decisions, err := decision.Load(modelDir)
		if err != nil {
			return err
		}
		for _, d := range decisions {
			fmt.Printf("AD%s\t%s\t%s\n", d.ID, d.Status, d.Title)
		}
		return nil
	},
}

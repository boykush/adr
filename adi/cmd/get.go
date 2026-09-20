package cmd

import (
	"fmt"

	"github.com/boykush/adr/adi/internal/decision"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:          "get <id>",
	Short:        "Print one decision",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := decision.Find(modelDir, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("AD%s\t%s\t%s\n\n%s\n", d.ID, d.Status, d.Title, d.Body)
		return nil
	},
}

package cmd

import (
	"github.com/spf13/cobra"
)

// evalprojCmd represents the evalproj command
var evalprojCmd = &cobra.Command{
	Use:   "evalproj",
	Short: "Evaluate a project using a set of rules",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(evalprojCmd)
}

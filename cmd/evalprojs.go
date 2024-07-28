package cmd

import (
	"github.com/philippsommer27/calabash-sat/internal"
	"github.com/spf13/cobra"
)

// evalprojCmd represents the evalproj command
var evalprojsCmd = &cobra.Command{
	Use:   "evalprojs",
	Short: "Evaluate a project using a set of rules",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		internal.EvalProjects(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(evalprojsCmd)
}

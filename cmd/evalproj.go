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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// evalprojCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// evalprojCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

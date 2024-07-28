package cmd

import (
	"github.com/spf13/cobra"
	"github.com/philippsommer27/calabash-sat/internal"
)

// evalruleCmd represents the evalrule command
var evalruleCmd = &cobra.Command{
	Use:   "evalrule",
	Short: "Evaluate a rule on a group of projects",
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		print, _ := cmd.Flags().GetBool("print")
		multi, _ := cmd.Flags().GetBool("multi")
		internal.EvalRule(args[0], args[1], args[2], args[3], print, multi)
	},
}

func init() {
	rootCmd.AddCommand(evalruleCmd)
	rootCmd.PersistentFlags().BoolP("print", "p", false, "Print semgrep output to stdout")
	rootCmd.PersistentFlags().BoolP("multi", "m", false, "Use multiple threads for analysis")
}

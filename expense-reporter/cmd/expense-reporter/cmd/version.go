package cmd

import (
	"expense-reporter/internal/cli"
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version number of expense-reporter.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("expense-reporter version %s\n", cli.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "git-visible %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

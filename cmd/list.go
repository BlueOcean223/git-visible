package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

var listVerify bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List added repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repo.LoadRepos()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(repos) == 0 {
			fmt.Fprintln(out, "no repositories added")
			return nil
		}

		if !listVerify {
			for _, p := range repos {
				fmt.Fprintln(out, p)
			}
			return nil
		}

		_, invalid, err := repo.VerifyRepos()
		if err != nil {
			return err
		}
		invalidSet := make(map[string]struct{}, len(invalid))
		for _, p := range invalid {
			invalidSet[p] = struct{}{}
		}

		for _, p := range repos {
			if _, ok := invalidSet[p]; ok {
				fmt.Fprintf(out, "%s (invalid)\n", p)
				continue
			}
			fmt.Fprintln(out, p)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listVerify, "verify", false, "Verify repositories on disk")
	rootCmd.AddCommand(listCmd)
}

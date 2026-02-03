package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

var removeInvalid bool

var removeCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Remove a repository",
	Args: func(cmd *cobra.Command, args []string) error {
		if removeInvalid {
			if len(args) != 0 {
				return fmt.Errorf("usage: git-visible remove --invalid")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("usage: git-visible remove <path>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		if removeInvalid {
			_, invalid, err := repo.VerifyRepos()
			if err != nil {
				return err
			}
			if len(invalid) == 0 {
				fmt.Fprintln(out, "no invalid repositories")
				return nil
			}

			removed := 0
			for _, p := range invalid {
				if err := repo.RemoveRepo(p); err != nil {
					return err
				}
				removed++
				fmt.Fprintln(out, p)
			}
			fmt.Fprintf(out, "removed %d repositories\n", removed)
			return nil
		}

		if err := repo.RemoveRepo(args[0]); err != nil {
			return err
		}
		fmt.Fprintln(out, args[0])
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeInvalid, "invalid", false, "Remove all invalid repositories")
	rootCmd.AddCommand(removeCmd)
}

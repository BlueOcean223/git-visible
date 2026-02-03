package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

var (
	addDepth    int
	addExcludes []string
	addDryRun   bool
)

var addCmd = &cobra.Command{
	Use:   "add <folder>",
	Short: "Scan and add git repositories under a folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if addDepth < -1 {
			return fmt.Errorf("depth must be >= -1, got %d", addDepth)
		}

		found, err := repo.ScanRepos(args[0], addDepth, addExcludes)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(found) == 0 {
			fmt.Fprintln(out, "no repositories found")
			return nil
		}

		if addDryRun {
			fmt.Fprintln(out, "dry run; repositories found:")
			for _, p := range found {
				fmt.Fprintln(out, p)
			}
			return nil
		}

		existing, err := repo.LoadRepos()
		if err != nil {
			return err
		}
		existingSet := make(map[string]struct{}, len(existing))
		for _, p := range existing {
			existingSet[p] = struct{}{}
		}

		added := 0
		for _, p := range found {
			if _, ok := existingSet[p]; ok {
				continue
			}
			if err := repo.AddRepo(p); err != nil {
				return err
			}
			existingSet[p] = struct{}{}
			added++
			fmt.Fprintln(out, p)
		}

		if added == 0 {
			fmt.Fprintln(out, "no new repositories to add")
			return nil
		}
		fmt.Fprintf(out, "added %d repositories\n", added)
		return nil
	},
}

func init() {
	addCmd.Flags().IntVarP(&addDepth, "depth", "d", -1, "Maximum recursion depth (-1 for unlimited)")
	addCmd.Flags().StringArrayVarP(&addExcludes, "exclude", "x", nil, "Exclude directories (repeatable)")
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Preview repositories without adding")

	rootCmd.AddCommand(addCmd)
}

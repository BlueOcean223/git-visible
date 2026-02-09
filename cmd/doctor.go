package cmd

import (
	"fmt"
	"io"

	"git-visible/internal/config"
	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment and configuration issues",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "Running diagnostics...")

	hasError := false

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		hasError = true
		fmt.Fprintf(out, "❌ Config: %v\n", cfgErr)
	} else {
		issues := config.ValidateConfig(cfg)
		if len(issues) == 0 {
			fmt.Fprintln(out, "✅ Config: OK")
		} else {
			fmt.Fprintf(out, "⚠️  Config: %d issue(s)\n", len(issues))
			printLines(out, issues)
		}
	}

	validRepos, invalidRepos, verifyErr := repo.VerifyRepos()
	if verifyErr != nil {
		hasError = true
		fmt.Fprintf(out, "❌ Repositories: %v\n", verifyErr)
	} else {
		total := len(validRepos) + len(invalidRepos)
		switch {
		case total == 0:
			fmt.Fprintln(out, "⚠️  Repositories: no repositories added")
		case len(invalidRepos) == 0:
			fmt.Fprintf(out, "✅ Repositories: %d/%d valid\n", len(validRepos), total)
		default:
			hasError = true
			fmt.Fprintf(out, "❌ Repositories: %d/%d valid, %d invalid\n", len(validRepos), total, len(invalidRepos))
			printLines(out, invalidRepos)
		}
	}

	if len(validRepos) == 0 {
		fmt.Fprintln(out, "⚠️  Branch reachability: skipped (no valid repositories)")
	} else {
		branchErrors := make([]string, 0)
		for _, repoPath := range validRepos {
			if err := repo.CheckBranchReachability(repoPath, ""); err != nil {
				branchErrors = append(branchErrors, fmt.Sprintf("%s: %v", repoPath, err))
			}
		}
		if len(branchErrors) == 0 {
			fmt.Fprintln(out, "✅ Branch reachability: OK")
		} else {
			hasError = true
			fmt.Fprintf(out, "❌ Branch reachability: %d issue(s)\n", len(branchErrors))
			printLines(out, branchErrors)
		}
	}

	if len(validRepos) == 0 {
		fmt.Fprintln(out, "⚠️  Permissions: skipped (no valid repositories)")
	} else {
		permissionErrors := make([]string, 0)
		for _, repoPath := range validRepos {
			if err := repo.CheckPermissions(repoPath); err != nil {
				permissionErrors = append(permissionErrors, fmt.Sprintf("%s: %v", repoPath, err))
			}
		}
		if len(permissionErrors) == 0 {
			fmt.Fprintln(out, "✅ Permissions: OK")
		} else {
			hasError = true
			fmt.Fprintf(out, "❌ Permissions: %d issue(s)\n", len(permissionErrors))
			printLines(out, permissionErrors)
		}
	}

	performanceWarnings := repo.CheckPerformance(validRepos)
	if len(performanceWarnings) == 0 {
		fmt.Fprintln(out, "✅ Performance: OK")
	} else {
		fmt.Fprintf(out, "⚠️  Performance: %d warning(s)\n", len(performanceWarnings))
		printLines(out, performanceWarnings)
	}

	if hasError {
		return fmt.Errorf("doctor found issues")
	}
	return nil
}

func printLines(out io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintf(out, "   - %s\n", line)
	}
}

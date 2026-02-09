package cmd

import (
	"fmt"
	"io"

	"git-visible/internal/config"
	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

// doctorCmd 实现 doctor 子命令，一站式诊断环境和配置问题。
// 依次执行 5 项检查：配置合法性、仓库路径有效性、分支可达性、读权限、性能预警。
// 有错误时返回非零退出码，仅警告时返回 0。
// 用法: git-visible doctor
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment and configuration issues",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

// init 注册 doctor 命令。
func init() {
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor 是 doctor 命令的核心逻辑，按顺序执行 5 项诊断检查：
//  1. 配置合法性（months、email 格式）
//  2. 仓库路径有效性（路径存在且包含 .git）
//  3. 分支可达性（HEAD 和指定分支有提交且可解析）
//  4. 读权限（.git/HEAD 可读）
//  5. 性能预警（仓库数量 >50 或 .git 体积 >1GB）
//
// 输出使用 ✅/⚠️/❌ 分类显示，有错误时返回 error（exit 非零）。
func runDoctor(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "Running diagnostics...")

	hasError := false

	// 1. 配置合法性检查
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

	// 2. 仓库路径有效性检查
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

	// 3. 分支可达性检查（需要有效仓库）
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

	// 4. 读权限检查（需要有效仓库）
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

	// 5. 性能预警（仓库数量、.git 体积）
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

// printLines 将字符串列表以缩进列表形式输出，每行前加 "   - " 前缀。
func printLines(out io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintf(out, "   - %s\n", line)
	}
}

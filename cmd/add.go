package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

// 命令行标志变量
var (
	addDepth    int      // 扫描的最大递归深度，-1 表示无限制
	addExcludes []string // 要排除的目录列表
	addDryRun   bool     // 预览模式，不实际保存
)

// addCmd 实现 add 子命令，用于扫描并添加指定目录下的 Git 仓库。
// 用法: git-visible add <folder>
// 示例: git-visible add ~/code
var addCmd = &cobra.Command{
	Use:   "add <folder>",
	Short: "Scan and add git repositories under a folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 验证深度参数
		if addDepth < -1 {
			return fmt.Errorf("depth must be >= -1, got %d", addDepth)
		}

		// 扫描指定目录下的所有 Git 仓库
		found, err := repo.ScanRepos(args[0], addDepth, addExcludes)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(found) == 0 {
			fmt.Fprintln(out, "no repositories found")
			return nil
		}

		// 预览模式：仅显示找到的仓库，不保存
		if addDryRun {
			fmt.Fprintln(out, "dry run; repositories found:")
			for _, p := range found {
				fmt.Fprintln(out, p)
			}
			return nil
		}

		added, err := repo.AddRepos(found)
		if err != nil {
			return err
		}

		if len(added) == 0 {
			fmt.Fprintln(out, "no new repositories to add")
			return nil
		}

		for _, p := range added {
			fmt.Fprintln(out, p)
		}

		fmt.Fprintf(out, "added %d repositories\n", len(added))
		return nil
	},
}

// init 注册 add 命令及其标志。
func init() {
	addCmd.Flags().IntVarP(&addDepth, "depth", "d", -1, "Maximum recursion depth (-1 for unlimited)")
	addCmd.Flags().StringArrayVarP(&addExcludes, "exclude", "x", nil, "Exclude directories (repeatable)")
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Preview repositories without adding")

	rootCmd.AddCommand(addCmd)
}

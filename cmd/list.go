package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

// listVerify 标志控制是否验证仓库路径的有效性。
var listVerify bool

// listCmd 实现 list 子命令，用于列出所有已添加的仓库。
// 用法: git-visible list [--verify]
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List added repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 加载已保存的仓库列表
		repos, err := repo.LoadRepos()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(repos) == 0 {
			fmt.Fprintln(out, "no repositories added")
			return nil
		}

		// 不验证时直接输出列表
		if !listVerify {
			for _, p := range repos {
				fmt.Fprintln(out, p)
			}
			return nil
		}

		// 验证模式：检查每个仓库路径是否有效
		_, invalid, err := repo.VerifyRepos()
		if err != nil {
			return err
		}
		invalidSet := make(map[string]struct{}, len(invalid))
		for _, p := range invalid {
			invalidSet[p] = struct{}{}
		}

		// 输出列表，无效仓库标记 (invalid)
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

// init 注册 list 命令及其标志。
func init() {
	listCmd.Flags().BoolVar(&listVerify, "verify", false, "Verify repositories on disk")

	rootCmd.AddCommand(listCmd)
}

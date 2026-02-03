package cmd

import (
	"fmt"

	"git-visible/internal/repo"

	"github.com/spf13/cobra"
)

// removeInvalid 标志控制是否移除所有无效仓库。
var removeInvalid bool

// removeCmd 实现 remove 子命令，用于从列表中移除仓库。
// 支持两种模式：
// 1. git-visible remove <path> - 移除指定路径的仓库
// 2. git-visible remove --invalid - 移除所有无效（路径不存在或不是 Git 仓库）的仓库
var removeCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Remove a repository",
	Args: func(cmd *cobra.Command, args []string) error {
		// --invalid 模式不需要参数
		if removeInvalid {
			if len(args) != 0 {
				return fmt.Errorf("usage: git-visible remove --invalid")
			}
			return nil
		}
		// 普通模式需要一个路径参数
		if len(args) != 1 {
			return fmt.Errorf("usage: git-visible remove <path>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		// --invalid 模式：批量移除所有无效仓库
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

		// 普通模式：移除指定路径的仓库
		if err := repo.RemoveRepo(args[0]); err != nil {
			return err
		}
		fmt.Fprintln(out, args[0])
		return nil
	},
}

// init 注册 remove 命令及其标志。
func init() {
	removeCmd.Flags().BoolVar(&removeInvalid, "invalid", false, "Remove all invalid repositories")

	rootCmd.AddCommand(removeCmd)
}

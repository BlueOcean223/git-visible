package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version 定义当前应用版本号。
// 可在构建时通过 -ldflags 覆盖此值。
var (
	Version = "0.5.0"
)

// versionCmd 实现 version 子命令，用于显示版本信息。
// 用法: git-visible version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "git-visible %s\n", Version)
	},
}

// init 注册 version 命令。
func init() {
	rootCmd.AddCommand(versionCmd)
}

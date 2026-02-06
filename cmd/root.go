package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd 是 CLI 的根命令。
// 当不带子命令运行时，默认执行 show 命令显示贡献热力图。
var rootCmd = &cobra.Command{
	Use:   "git-visible",
	Short: "Git contribution heatmap from local repositories",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(cmd, args)
	},
}

// Execute 执行根命令，是 CLI 的入口点。
// 如果执行过程中发生错误，程序将以退出码 1 终止。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}


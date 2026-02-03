package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// init 在包初始化时注册配置初始化回调。
func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig 初始化应用配置。
// 从 ~/.config/git-visible/config.yaml 读取配置文件。
// 如果配置文件不存在，则静默忽略；其他读取错误会输出到 stderr。
func initConfig() {
	// 获取用户主目录路径（如 /home/user 或 C:\Users\user）
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "get home dir:", err)
		return
	}

	configFile := filepath.Join(home, ".config", "git-visible", "config.yaml")

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 读取并加载配置文件到 viper 内存中
	// 加载后可通过 viper.GetString("email")、viper.GetInt("months") 等方法获取配置值
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时静默忽略（首次使用时文件可能尚未创建）
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			return
		}
		// 其他错误（如文件权限、格式错误等）输出到 stderr
		fmt.Fprintln(os.Stderr, "read config:", err)
	}
}

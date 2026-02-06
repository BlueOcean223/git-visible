package cmd

import (
	"fmt"
	"strconv"

	"git-visible/internal/config"

	"github.com/spf13/cobra"
)

// setCmd 实现 set 子命令，用于查看或修改默认配置。
// 支持两种模式：
// 1. git-visible set - 显示当前配置
// 2. git-visible set <key> <value> - 设置配置项（支持 email 和 months）
var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set or show default configuration",
	Args: func(cmd *cobra.Command, args []string) error {
		// 无参数：显示配置
		if len(args) == 0 {
			return nil
		}
		// 设置配置需要正好两个参数
		if len(args) != 2 {
			return fmt.Errorf("usage: git-visible set [email|months] <value>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 加载当前配置
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// 无参数时显示当前配置
		if len(args) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "email: %s\nmonths: %d\n", cfg.Email, cfg.Months)
			return nil
		}

		key := args[0]
		val := args[1]

		// 根据 key 修改对应配置项
		switch key {
		case "email":
			cfg.Email = val
		case "months":
			months, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid months %q: %w", val, err)
			}
			if months <= 0 {
				return fmt.Errorf("months must be > 0, got %d", months)
			}
			cfg.Months = months
		default:
			return fmt.Errorf("unsupported key %q (supported: email, months)", key)
		}

		// 保存修改后的配置
		return config.Save(*cfg)
	},
}

// init 注册 set 命令。
func init() {
	rootCmd.AddCommand(setCmd)
}

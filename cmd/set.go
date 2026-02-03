package cmd

import (
	"fmt"
	"strconv"

	"git-visible/internal/config"

	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set or show default configuration",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if len(args) != 2 {
			return fmt.Errorf("usage: git-visible set [email|months] <value>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "email: %s\nmonths: %d\n", cfg.Email, cfg.Months)
			return nil
		}

		key := args[0]
		val := args[1]

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

		return config.Save(cfg)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}

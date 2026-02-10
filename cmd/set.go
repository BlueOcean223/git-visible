package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"git-visible/internal/config"

	"github.com/spf13/cobra"
)

// setCmd 实现 set 子命令，用于查看或修改默认配置。
// 支持两种模式：
// 1. git-visible set - 显示当前配置
// 2. git-visible set <key> <value> - 设置配置项（支持 email 和 months）
var setCmd = newSetCmd()

// newSetCmd 构建 set 命令，便于在测试中复用。
func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set or show default configuration",
		Long: `View or modify default configuration (email, months, aliases).

Without arguments, displays the current configuration.
With key/value, sets the specified option.
Use "set alias" subcommands to manage email alias groups.`,
		Example: `  git-visible set
  git-visible set email your@email.com
  git-visible set months 12
  git-visible set alias add Alice alice@company.com alice@gmail.com
  git-visible set alias list`,
		Args: validateSetArgs,
		RunE: runSet,
	}
	cmd.AddCommand(newSetAliasCmd())
	return cmd
}

// validateSetArgs 校验 set 顶层参数格式。
func validateSetArgs(cmd *cobra.Command, args []string) error {
	// 无参数：显示配置
	if len(args) == 0 {
		return nil
	}
	// 设置配置需要正好两个参数
	if len(args) != 2 {
		return fmt.Errorf("usage: git-visible set [email|months] <value>")
	}
	return nil
}

// runSet 执行 set 顶层逻辑（显示或设置 email/months）。
func runSet(cmd *cobra.Command, args []string) error {
	// 加载当前配置
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 无参数时显示当前配置
	if len(args) == 0 {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "email: %s\nmonths: %d\n", cfg.Email, cfg.Months)
		printAliases(out, cfg.Aliases, "aliases: (none)")
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
}

// newSetAliasCmd 构建 alias 子命令组。
func newSetAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage email aliases",
		Long: `Manage email alias groups for mapping multiple addresses to one author.

When aliases are configured, all emails in a group are treated as the same
person during commit collection. The first email in the group is the primary.
Each email can only belong to one alias group.`,
		Example: `  git-visible set alias add Alice alice@company.com alice@gmail.com
  git-visible set alias add Bob bob@work.com bob@personal.com
  git-visible set alias list
  git-visible set alias remove Alice`,
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(newSetAliasAddCmd())
	cmd.AddCommand(newSetAliasRemoveCmd())
	cmd.AddCommand(newSetAliasListCmd())
	return cmd
}

// newSetAliasAddCmd 构建 alias add 子命令。
func newSetAliasAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <email1> [email2...]",
		Short: "Add or update an alias group",
		Long: `Add a new alias group or update an existing one.

The first email becomes the primary address that all others map to.
If the name already exists, its email list is replaced.
An email cannot appear in multiple alias groups.`,
		Example: `  git-visible set alias add Alice alice@company.com alice@gmail.com`,
		Args:    validateSetAliasAddArgs,
		RunE:    runSetAliasAdd,
	}
}

// validateSetAliasAddArgs 校验 alias add 参数。
func validateSetAliasAddArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: git-visible set alias add <name> <email1> [email2...]")
	}
	return nil
}

// runSetAliasAdd 添加或更新 alias。
func runSetAliasAdd(cmd *cobra.Command, args []string) error {
	name, emails, err := normalizeAliasInput(args[0], args[1:])
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 添加前检查跨组邮箱冲突（同组更新不算冲突）。
	if err := checkAliasEmailConflicts(cfg.Aliases, name, emails); err != nil {
		return err
	}

	alias := config.Alias{
		Name:   name,
		Emails: emails,
	}

	index := findAliasIndex(cfg.Aliases, name)
	if index >= 0 {
		cfg.Aliases[index] = alias
	} else {
		cfg.Aliases = append(cfg.Aliases, alias)
	}

	if err := config.Save(*cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "alias %q saved: %s\n", name, strings.Join(emails, ", "))
	return nil
}

// newSetAliasRemoveCmd 构建 alias remove 子命令。
func newSetAliasRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove an alias group",
		Example: `  git-visible set alias remove Alice`,
		Args:    cobra.ExactArgs(1),
		RunE:    runSetAliasRemove,
	}
}

// runSetAliasRemove 删除指定 alias。
func runSetAliasRemove(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	index := findAliasIndex(cfg.Aliases, name)
	if index < 0 {
		return fmt.Errorf("alias %q not found", name)
	}

	cfg.Aliases = append(cfg.Aliases[:index], cfg.Aliases[index+1:]...)
	if err := config.Save(*cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "alias %q removed\n", name)
	return nil
}

// newSetAliasListCmd 构建 alias list 子命令。
func newSetAliasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all aliases",
		Args:  cobra.NoArgs,
		RunE:  runSetAliasList,
	}
}

// runSetAliasList 列出当前所有 alias。
func runSetAliasList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	printAliases(cmd.OutOrStdout(), cfg.Aliases, "No aliases configured")
	return nil
}

// printAliases 按统一格式输出 aliases 列表。
func printAliases(out io.Writer, aliases []config.Alias, emptyMsg string) {
	if len(aliases) == 0 {
		fmt.Fprintln(out, emptyMsg)
		return
	}

	fmt.Fprintln(out, "aliases:")
	for _, alias := range aliases {
		aliasName := strings.TrimSpace(alias.Name)
		emails := make([]string, 0, len(alias.Emails))
		for _, email := range alias.Emails {
			trimmed := strings.TrimSpace(email)
			if trimmed == "" {
				continue
			}
			emails = append(emails, trimmed)
		}
		fmt.Fprintf(out, "  %s: %s\n", aliasName, strings.Join(emails, ", "))
	}
}

// normalizeAliasInput 对 alias 名称和邮箱列表执行 TrimSpace，并去重重复邮箱。
func normalizeAliasInput(name string, emails []string) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("alias name cannot be empty")
	}

	normalized := make([]string, 0, len(emails))
	seen := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		trimmed := strings.TrimSpace(email)
		if trimmed == "" {
			return "", nil, fmt.Errorf("alias email cannot be empty")
		}
		if !strings.Contains(trimmed, "@") {
			return "", nil, fmt.Errorf("invalid email format %q: must contain @", trimmed)
		}

		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return "", nil, fmt.Errorf("at least one email is required")
	}

	return name, normalized, nil
}

// checkAliasEmailConflicts 检查待写入邮箱是否已存在于其他 alias 组（大小写不敏感）。
func checkAliasEmailConflicts(aliases []config.Alias, name string, emails []string) error {
	// 构建 email→aliasName 索引（跳过同名组）
	index := make(map[string]string)
	for _, alias := range aliases {
		if strings.EqualFold(alias.Name, name) {
			continue
		}
		for _, existing := range alias.Emails {
			key := strings.ToLower(strings.TrimSpace(existing))
			if key == "" {
				continue
			}
			index[key] = alias.Name
		}
	}

	for _, email := range emails {
		key := strings.ToLower(strings.TrimSpace(email))
		if owner, ok := index[key]; ok {
			return fmt.Errorf("email %q already belongs to alias %q", email, owner)
		}
	}
	return nil
}

// findAliasIndex 按名称查找 alias 下标（大小写不敏感），未找到返回 -1。
func findAliasIndex(aliases []config.Alias, name string) int {
	for i, alias := range aliases {
		if strings.EqualFold(alias.Name, name) {
			return i
		}
	}
	return -1
}

// init 注册 set 命令。
func init() {
	rootCmd.AddCommand(setCmd)
}

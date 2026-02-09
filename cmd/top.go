package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

// 命令行标志变量
var (
	topEmails []string // 要过滤的邮箱列表
	topMonths int      // 统计的月份数
	topSince  string   // 起始日期
	topUntil  string   // 结束日期
	topFormat string   // 输出格式：table/json/csv
	topNoCache bool    // 是否禁用缓存

	topNumber int  // 显示的仓库数量
	topAll    bool // 是否显示所有仓库
)

// topCmd 实现 top 子命令，用于显示贡献最多的仓库排行榜。
// 用法: git-visible top [-n number|--all] [-e email] [-m months] [--since/--until] [-f format]
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top repositories by commits",
	Args:  cobra.NoArgs,
	RunE:  runTop,
}

// init 注册 top 命令及其标志。
func init() {
	topCmd.Flags().IntVarP(&topNumber, "number", "n", 10, "Number of repositories to show")
	topCmd.Flags().BoolVar(&topAll, "all", false, "Show all repositories")
	topCmd.MarkFlagsMutuallyExclusive("number", "all")

	topCmd.Flags().StringArrayVarP(&topEmails, "email", "e", nil, "Email filter (repeatable)")
	topCmd.Flags().IntVarP(&topMonths, "months", "m", 0, "Months to include (default: config value; ignored when --since/--until is set)")
	topCmd.Flags().StringVar(&topSince, "since", "", "Start date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	topCmd.Flags().StringVar(&topUntil, "until", "", "End date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	topCmd.Flags().StringVarP(&topFormat, "format", "f", "table", "Output format: table/json/csv")
	topCmd.Flags().BoolVar(&topNoCache, "no-cache", false, "Disable cache, force full scan")

	rootCmd.AddCommand(topCmd)
}

// runTop 是 top 命令的核心逻辑，收集并输出仓库提交排行榜。
func runTop(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	runCtx, err := prepareRun(topEmails, topMonths, topSince, topUntil)
	if err != nil {
		if errors.Is(err, errNoRepositoriesAdded) {
			fmt.Fprintln(out, "no repositories added")
			return nil
		}
		return err
	}

	if !topAll && topNumber <= 0 {
		return fmt.Errorf("number must be > 0, got %d", topNumber)
	}

	since := strings.TrimSpace(topSince)
	until := strings.TrimSpace(topUntil)

	var normalizeEmail func(string) string
	if runCtx.Config != nil && len(runCtx.Config.Aliases) > 0 {
		normalizeEmail = runCtx.Config.NormalizeEmail
	}

	// 按仓库分别收集提交统计
	perRepo, collectErr := stats.CollectStatsPerRepoWithOptions(runCtx.Repos, runCtx.Emails, runCtx.Since, runCtx.Until, stats.BranchOption{}, normalizeEmail, !topNoCache)
	if collectErr != nil {
		if len(perRepo) == 0 {
			return fmt.Errorf("all repositories failed to collect stats: %w", collectErr)
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: some repositories failed, showing partial results:", collectErr)
	}

	// 确定显示数量：--all 时 limit=0 表示不限制
	limit := topNumber
	if topAll {
		limit = 0
	}

	// 计算排行榜（按提交数降序，百分比保证合计 100.0%）
	ranking := stats.RankRepositories(perRepo, limit)

	// 根据指定格式输出结果
	format := strings.ToLower(strings.TrimSpace(topFormat))
	switch format {
	case "", "table":
		if ranking.TotalCommits == 0 {
			fmt.Fprintln(out, "no commits found")
			return nil
		}
		return writeTopTable(out, ranking, topRangeLabel(since, until, runCtx.months, runCtx.Since, runCtx.Until))
	case "json":
		return writeTopJSON(out, ranking)
	case "csv":
		return writeTopCSV(out, ranking)
	default:
		return fmt.Errorf("unsupported format %q (supported: table, json, csv)", topFormat)
	}
}

// topRangeLabel 生成时间范围的显示标签。
func topRangeLabel(since, until string, months int, start, end time.Time) string {
	since = strings.TrimSpace(since)
	until = strings.TrimSpace(until)
	if since == "" && until == "" {
		return fmt.Sprintf("last %d months", months)
	}
	return fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
}

// writeTopTable 以表格格式输出排行榜。
func writeTopTable(out io.Writer, ranking stats.RepoRanking, rangeLabel string) error {
	// 将绝对路径转换为 ~/... 的短路径，并计算仓库列宽度
	displayPaths := make([]string, 0, len(ranking.Repositories))
	repoWidth := len("Repository")
	for _, r := range ranking.Repositories {
		p := displayRepoPath(r.Repository)
		displayPaths = append(displayPaths, p)
		if len(p) > repoWidth {
			repoWidth = len(p)
		}
	}

	// 计算各列宽度，确保表头和数据对齐
	rankWidth := len(fmt.Sprintf("%d", len(ranking.Repositories)))
	rankWidth = max(rankWidth, 2)

	commitWidth := len("Commits")
	for _, r := range ranking.Repositories {
		w := len(fmt.Sprintf("%d", r.Commits))
		if w > commitWidth {
			commitWidth = w
		}
	}
	totalWidth := len(fmt.Sprintf("%d", ranking.TotalCommits))
	if totalWidth > commitWidth {
		commitWidth = totalWidth
	}

	percentWidth := len("100.0%")

	// 绘制表格：标题 → 分隔线 → 表头 → 分隔线 → 数据行 → 分隔线 → 汇总行
	lineLen := rankWidth + 3 + repoWidth + 1 + commitWidth + 1 + percentWidth
	rule := strings.Repeat("─", lineLen)

	fmt.Fprintf(out, "Top %d repositories (%s)\n", len(ranking.Repositories), rangeLabel)
	fmt.Fprintln(out, rule)
	fmt.Fprintf(out, "%*s   %-*s %*s %*s\n", rankWidth, "#", repoWidth, "Repository", commitWidth, "Commits", percentWidth, "%")
	fmt.Fprintln(out, rule)

	for i, r := range ranking.Repositories {
		percentStr := fmt.Sprintf("%.1f%%", r.Percent)
		fmt.Fprintf(out, "%*d   %-*s %*d %*s\n", rankWidth, i+1, repoWidth, displayPaths[i], commitWidth, r.Commits, percentWidth, percentStr)
	}

	fmt.Fprintln(out, rule)
	fmt.Fprintf(out, "%*s   %-*s %*d %*s\n", rankWidth, "", repoWidth, "Total", commitWidth, ranking.TotalCommits, percentWidth, "100.0%")

	return nil
}

// writeTopJSON 以 JSON 格式输出排行榜。
func writeTopJSON(out io.Writer, ranking stats.RepoRanking) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(ranking)
}

// writeTopCSV 以 CSV 格式输出排行榜。
func writeTopCSV(out io.Writer, ranking stats.RepoRanking) error {
	w := csv.NewWriter(out)
	if err := w.Write([]string{"repository", "commits", "percent"}); err != nil {
		return err
	}
	for _, r := range ranking.Repositories {
		if err := w.Write([]string{
			r.Repository,
			fmt.Sprintf("%d", r.Commits),
			fmt.Sprintf("%.1f", r.Percent),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// displayRepoPath 将绝对路径转换为更友好的显示格式（~ 替代 home 目录）。
func displayRepoPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return p
	}

	home = filepath.Clean(home)
	p = filepath.Clean(p)

	if p == home {
		return "~"
	}

	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(p, prefix) {
		return "~" + p[len(home):]
	}
	return p
}

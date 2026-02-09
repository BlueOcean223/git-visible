package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

// 命令行标志变量
var (
	compareEmails  []string // 要对比的邮箱列表
	comparePeriods []string // 要对比的时间段列表
	compareYears   []int    // 要对比的年份列表（--period YYYY 的快捷方式）
	compareFormat  string   // 输出格式：table/json/csv
	compareNoCache bool     // 是否禁用缓存
)

// compareCmd 实现 compare 子命令，用于对比多个邮箱或多个时间段的贡献统计。
// 用法:
//   - git-visible compare -e a@x.com -e b@y.com
//   - git-visible compare --period 2024-H1 --period 2024-H2
//   - git-visible compare --year 2024 --year 2025
var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare contribution stats by email or period",
	Args:  cobra.NoArgs,
	RunE:  runCompare,
}

// init 注册 compare 命令及其标志。
func init() {
	compareCmd.Flags().StringArrayVarP(&compareEmails, "email", "e", nil, "Emails to compare (repeatable)")
	compareCmd.Flags().StringArrayVar(&comparePeriods, "period", nil, "Periods to compare (repeatable): YYYY, YYYY-HN, YYYY-QN, YYYY-MM")
	compareCmd.Flags().IntSliceVar(&compareYears, "year", nil, "Years to compare (repeatable; shortcut for --period YYYY)")
	compareCmd.Flags().StringVarP(&compareFormat, "format", "f", "table", "Output format: table/json/csv")
	compareCmd.Flags().BoolVar(&compareNoCache, "no-cache", false, "Disable cache, force full scan")

	compareCmd.MarkFlagsMutuallyExclusive("email", "period")
	compareCmd.MarkFlagsMutuallyExclusive("email", "year")

	rootCmd.AddCommand(compareCmd)
}

// emailCompareItem 表示按邮箱对比时的单项结果。
type emailCompareItem struct {
	Email   string
	Metrics stats.CompareMetrics
}

// periodCompareItem 表示按时间段对比时的单项结果。
type periodCompareItem struct {
	Period  stats.Period
	Metrics stats.CompareMetrics
}

// runCompare 是 compare 命令的核心逻辑。
func runCompare(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	format := strings.ToLower(strings.TrimSpace(compareFormat))

	// 防御性清洗 compareEmails，用于模式判定，避免被配置邮箱回填影响。
	emails := cleanNonEmpty(compareEmails)
	// 清理并合并对比参数：--period 和 --year 合并为统一的时间段列表
	periodArgs := cleanNonEmpty(append(append([]string{}, comparePeriods...), yearsToPeriods(compareYears)...))

	prepareSince := "1970-01-01"
	prepareUntil := "1970-01-01"
	if len(emails) >= 2 {
		prepareSince = ""
		prepareUntil = ""
	}

	runCtx, err := prepareRun(compareEmails, 0, prepareSince, prepareUntil)
	if err != nil {
		if errors.Is(err, errNoRepositoriesAdded) {
			fmt.Fprintln(out, "no repositories added")
			return nil
		}
		return err
	}

	switch {
	case len(emails) > 0:
		if len(emails) < 2 {
			return fmt.Errorf("at least 2 emails are required to compare")
		}

		items, collectErr, allFailed := collectCompareByEmail(runCtx.Repos, emails, runCtx.Since, runCtx.Until, runCtx.NormalizeEmail, !compareNoCache)
		if collectErr != nil {
			if allFailed {
				return fmt.Errorf("all repositories failed to collect stats: %w", collectErr)
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "warning: some repositories failed, showing partial results:", collectErr)
		}

		switch format {
		case "", "table":
			return writeCompareEmailTable(out, items)
		case "json":
			return writeCompareEmailJSON(out, items)
		case "csv":
			return writeCompareEmailCSV(out, items)
		default:
			return fmt.Errorf("unsupported format %q (supported: table, json, csv)", compareFormat)
		}

	case len(periodArgs) > 0:
		if len(periodArgs) < 2 {
			return fmt.Errorf("at least 2 periods are required to compare")
		}

		periods := make([]stats.Period, 0, len(periodArgs))
		for _, p := range periodArgs {
			period, err := stats.ParsePeriod(p)
			if err != nil {
				return err
			}
			periods = append(periods, period)
		}

		items, collectErr, allFailed := collectCompareByPeriod(runCtx.Repos, periods, runCtx.Emails, runCtx.NormalizeEmail, !compareNoCache)
		if collectErr != nil {
			if allFailed {
				return fmt.Errorf("all repositories failed to collect stats: %w", collectErr)
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "warning: some repositories failed, showing partial results:", collectErr)
		}

		switch format {
		case "", "table":
			return writeComparePeriodTable(out, items)
		case "json":
			return writeComparePeriodJSON(out, items)
		case "csv":
			return writeComparePeriodCSV(out, items)
		default:
			return fmt.Errorf("unsupported format %q (supported: table, json, csv)", compareFormat)
		}

	default:
		return fmt.Errorf("at least 2 compare items are required (use -e/--email or --period/--year)")
	}
}

// collectCompareByEmail 按邮箱收集对比数据。
func collectCompareByEmail(repos []string, emails []string, start, end time.Time, normalizeEmail func(string) string, useCache bool) ([]emailCompareItem, error, bool) {
	byEmail, err := stats.CollectStatsByEmails(repos, emails, start, end, stats.BranchOption{}, normalizeEmail, useCache)
	allFailed := err != nil && byEmail == nil

	items := make([]emailCompareItem, 0, len(emails))
	for _, email := range emails {
		lookupEmail := email
		if normalizeEmail != nil {
			lookupEmail = normalizeEmail(email)
		}
		daily := byEmail[lookupEmail]
		if daily == nil {
			daily = make(map[time.Time]int)
		}
		items = append(items, emailCompareItem{
			Email:   email,
			Metrics: stats.CalculateCompareMetrics(daily),
		})
	}
	return items, err, allFailed
}

// collectCompareByPeriod 按时间段收集对比数据。
func collectCompareByPeriod(repos []string, periods []stats.Period, emails []string, normalizeEmail func(string) string, useCache bool) ([]periodCompareItem, error, bool) {
	items := make([]periodCompareItem, 0, len(periods))
	var errs []error
	allFailed := true
	for _, period := range periods {
		perRepo, err := stats.CollectStatsPerRepo(repos, emails, period.Start, period.End, stats.BranchOption{}, normalizeEmail, useCache)
		if err != nil {
			errs = append(errs, err)
		}
		if len(perRepo) > 0 {
			allFailed = false
		}
		daily := mergePerRepoStats(perRepo)
		items = append(items, periodCompareItem{
			Period:  period,
			Metrics: stats.CalculateCompareMetrics(daily),
		})
	}
	return items, errors.Join(errs...), allFailed
}

func mergePerRepoStats(perRepo map[string]map[time.Time]int) map[time.Time]int {
	merged := make(map[time.Time]int)
	for _, daily := range perRepo {
		for day, count := range daily {
			merged[day] += count
		}
	}
	return merged
}

// yearsToPeriods 将年份列表转换为 YYYY 格式的时间段字符串。
func yearsToPeriods(years []int) []string {
	if len(years) == 0 {
		return nil
	}
	out := make([]string, 0, len(years))
	for _, y := range years {
		if y <= 0 {
			continue
		}
		out = append(out, fmt.Sprintf("%04d", y))
	}
	return out
}

// cleanNonEmpty 过滤掉空字符串并去除首尾空白。
func cleanNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

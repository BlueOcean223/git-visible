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

	// 清理并合并对比参数：--period 和 --year 合并为统一的时间段列表
	emails := cleanNonEmpty(compareEmails)
	periodArgs := cleanNonEmpty(append(append([]string{}, comparePeriods...), yearsToPeriods(compareYears)...))

	prepareSince := "1970-01-01"
	prepareUntil := "1970-01-01"
	if len(emails) >= 2 {
		prepareSince = ""
		prepareUntil = ""
	}

	runCtx, err := prepareRun(nil, 0, prepareSince, prepareUntil)
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

		items, collectErr := collectCompareByEmail(runCtx.Repos, emails, runCtx.Since, runCtx.Until)
		if collectErr != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
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

		items, collectErr := collectCompareByPeriod(runCtx.Repos, periods, runCtx.Emails)
		if collectErr != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
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
func collectCompareByEmail(repos []string, emails []string, start, end time.Time) ([]emailCompareItem, error) {
	items := make([]emailCompareItem, 0, len(emails))
	var errs []error
	for _, email := range emails {
		daily, err := stats.CollectStats(repos, []string{email}, start, end, stats.BranchOption{})
		if err != nil {
			errs = append(errs, err)
		}
		items = append(items, emailCompareItem{
			Email:   email,
			Metrics: stats.CalculateCompareMetrics(daily),
		})
	}
	return items, errors.Join(errs...)
}

// collectCompareByPeriod 按时间段收集对比数据。
func collectCompareByPeriod(repos []string, periods []stats.Period, emails []string) ([]periodCompareItem, error) {
	items := make([]periodCompareItem, 0, len(periods))
	var errs []error
	for _, period := range periods {
		daily, err := stats.CollectStats(repos, emails, period.Start, period.End, stats.BranchOption{})
		if err != nil {
			errs = append(errs, err)
		}
		items = append(items, periodCompareItem{
			Period:  period,
			Metrics: stats.CalculateCompareMetrics(daily),
		})
	}
	return items, errors.Join(errs...)
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

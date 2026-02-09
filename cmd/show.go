package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

// 命令行标志变量
var (
	showEmails    []string // 要过滤的邮箱列表
	showMonths    int      // 统计的月份数
	showSince     string   // 起始日期：YYYY-MM-DD / YYYY-MM / 2m/1w/1y
	showUntil     string   // 结束日期：YYYY-MM-DD / YYYY-MM / 2m/1w/1y
	showBranch    string   // 指定分支名（仅统计该分支）
	showAllBranch bool     // 是否统计所有分支（去重）
	showFormat    string   // 输出格式：table/json/csv
	showNoLegend  bool     // 是否隐藏图例（仅 table 输出）
	showLegend    bool     // 是否显示图例（仅 table 输出）
	showNoSummary bool     // 是否隐藏摘要信息
	showSummary   bool     // 是否显示摘要信息
)

// showCmd 实现 show 子命令，用于显示贡献热力图。
// 这是默认命令，当不带子命令运行 git-visible 时也会执行。
// 用法: git-visible show [-e email] [-m months] [-f format]
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show contribution heatmap",
	Args:  cobra.NoArgs,
	RunE:  runShow,
}

// init 注册 show 命令并为根命令和 show 命令添加相同的标志。
func init() {
	addShowFlags(rootCmd)
	addShowFlags(showCmd)

	rootCmd.AddCommand(showCmd)
}

// addShowFlags 为指定命令添加 show 相关的标志。
// 这样根命令和 show 子命令可以共享相同的标志。
func addShowFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&showEmails, "email", "e", nil, "Email filter (repeatable)")
	cmd.Flags().IntVarP(&showMonths, "months", "m", 0, "Months to include (default: config value; ignored when --since/--until is set)")
	cmd.Flags().StringVar(&showSince, "since", "", "Start date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	cmd.Flags().StringVar(&showUntil, "until", "", "End date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	cmd.Flags().StringVarP(&showBranch, "branch", "b", "", "Branch to include (default: HEAD)")
	cmd.Flags().BoolVar(&showAllBranch, "all-branches", false, "Include all local branches (deduplicated by commit hash)")
	cmd.MarkFlagsMutuallyExclusive("branch", "all-branches")
	cmd.Flags().StringVarP(&showFormat, "format", "f", "table", "Output format: table/json/csv")
	cmd.Flags().BoolVar(&showNoLegend, "no-legend", false, "Hide legend in table output")
	cmd.Flags().BoolVar(&showNoSummary, "no-summary", false, "Hide summary")
}

// runShow 是 show 命令的核心逻辑。
// 它从配置和已添加的仓库中收集提交统计，然后以指定格式输出。
func runShow(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	runCtx, err := prepareRun(showEmails, showMonths, showSince, showUntil)
	if err != nil {
		if errors.Is(err, errNoRepositoriesAdded) {
			fmt.Fprintln(out, "no repositories added")
			return nil
		}
		return err
	}

	// 收集所有仓库的提交统计
	branchOpt := stats.BranchOption{
		Branch:      strings.TrimSpace(showBranch),
		AllBranches: showAllBranch,
	}
	st, collectErr := stats.CollectStats(runCtx.Repos, runCtx.Emails, runCtx.Since, runCtx.Until, branchOpt)
	if collectErr != nil {
		if len(st) == 0 {
			return fmt.Errorf("all repositories failed to collect stats: %w", collectErr)
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: some repositories failed, showing partial results:", collectErr)
	}

	showLegend = !showNoLegend
	showSummary = !showNoSummary

	// 根据指定格式输出结果
	switch strings.ToLower(strings.TrimSpace(showFormat)) {
	case "", "table":
		fmt.Fprint(out, stats.RenderHeatmapWithOptions(st, stats.HeatmapOptions{
			ShowLegend:  showLegend,
			ShowSummary: showSummary,
			Since:       runCtx.Since,
			Until:       runCtx.Until,
		}))
		return nil
	case "json":
		return writeJSON(out, st, showSummary)
	case "csv":
		return writeCSV(out, st)
	default:
		return fmt.Errorf("unsupported format %q (supported: table, json, csv)", showFormat)
	}
}

// dayStat 表示单日的提交统计，用于 JSON 输出。
type dayStat struct {
	Date  string `json:"date"`  // 日期，格式为 YYYY-MM-DD
	Count int    `json:"count"` // 当日提交数
}

// summaryStreak 表示 JSON 输出中的连续提交天数信息。
type summaryStreak struct {
	Days  int    `json:"days"`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// summaryWeekday 表示 JSON 输出中最活跃的星期几信息。
type summaryWeekday struct {
	Weekday string `json:"weekday"`
	Commits int    `json:"commits"`
}

// summaryPeakDay 表示 JSON 输出中提交数最多的单日信息。
type summaryPeakDay struct {
	Date    string `json:"date,omitempty"`
	Commits int    `json:"commits"`
}

// summaryOut 表示 JSON 输出中的统计摘要。
type summaryOut struct {
	TotalCommits      int            `json:"totalCommits"`
	ActiveDays        int            `json:"activeDays"`
	CurrentStreak     int            `json:"currentStreak"`
	LongestStreak     summaryStreak  `json:"longestStreak"`
	MostActiveWeekday summaryWeekday `json:"mostActiveWeekday"`
	PeakDay           summaryPeakDay `json:"peakDay"`
}

// jsonOutput 是 show 命令 JSON 格式的顶层输出结构。
type jsonOutput struct {
	Days    []dayStat   `json:"days"`
	Summary *summaryOut `json:"summary,omitempty"`
}

// writeJSON 将统计数据以 JSON 格式输出。
// 输出包含 days 数组与可选 summary 字段。
func writeJSON(out io.Writer, st map[time.Time]int, includeSummary bool) error {
	// 按日期排序
	keys := make([]time.Time, 0, len(st))
	for k := range st {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	// 转换为输出结构
	rows := make([]dayStat, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, dayStat{
			Date:  k.Format("2006-01-02"),
			Count: st[k],
		})
	}

	outObj := jsonOutput{Days: rows}
	if includeSummary {
		s := stats.CalculateSummary(st)

		so := summaryOut{
			TotalCommits:  s.TotalCommits,
			ActiveDays:    s.ActiveDays,
			CurrentStreak: s.CurrentStreak,
			LongestStreak: summaryStreak{Days: s.LongestStreak.Days},
			MostActiveWeekday: summaryWeekday{
				Weekday: stats.WeekdayAbbrev(s.MostActiveWeekday.Weekday),
				Commits: s.MostActiveWeekday.Commits,
			},
			PeakDay: summaryPeakDay{
				Commits: s.PeakDay.Commits,
			},
		}
		if !s.LongestStreak.Start.IsZero() {
			so.LongestStreak.Start = s.LongestStreak.Start.Format("2006-01-02")
		}
		if !s.LongestStreak.End.IsZero() {
			so.LongestStreak.End = s.LongestStreak.End.Format("2006-01-02")
		}
		if !s.PeakDay.Date.IsZero() {
			so.PeakDay.Date = s.PeakDay.Date.Format("2006-01-02")
		}

		outObj.Summary = &so
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(outObj)
}

// writeCSV 将统计数据以 CSV 格式输出。
// 输出包含表头 date,count，数据按日期排序。
func writeCSV(out io.Writer, st map[time.Time]int) error {
	// 按日期排序
	keys := make([]time.Time, 0, len(st))
	for k := range st {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	w := csv.NewWriter(out)
	// 写入表头
	if err := w.Write([]string{"date", "count"}); err != nil {
		return err
	}
	// 写入数据行
	for _, k := range keys {
		if err := w.Write([]string{k.Format("2006-01-02"), fmt.Sprintf("%d", st[k])}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

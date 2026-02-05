package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"git-visible/internal/config"
	"git-visible/internal/repo"
	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

// 命令行标志变量
var (
	showEmails    []string // 要过滤的邮箱列表
	showMonths    int      // 统计的月份数
	showSince     string   // 起始日期：YYYY-MM-DD / YYYY-MM / 2m/1w/1y
	showUntil     string   // 结束日期：YYYY-MM-DD / YYYY-MM / 2m/1w/1y
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
	cmd.Flags().StringVarP(&showFormat, "format", "f", "table", "Output format: table/json/csv")
	cmd.Flags().BoolVar(&showNoLegend, "no-legend", false, "Hide legend in table output")
	cmd.Flags().BoolVar(&showNoSummary, "no-summary", false, "Hide summary")
}

// runShow 是 show 命令的核心逻辑。
// 它从配置和已添加的仓库中收集提交统计，然后以指定格式输出。
func runShow(cmd *cobra.Command, _ []string) error {
	// 加载配置获取默认值
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 加载已添加的仓库列表
	repos, err := repo.LoadRepos()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(repos) == 0 {
		fmt.Fprintln(out, "no repositories added")
		return nil
	}

	since := strings.TrimSpace(showSince)
	until := strings.TrimSpace(showUntil)

	// 确定统计月份数：优先使用命令行参数，否则使用配置值。
	// 当指定 --since/--until 时忽略 -m/--months（仅在只指定 --until 时使用配置默认 months 计算起点）。
	months := showMonths
	if months == 0 {
		months = cfg.Months
	}
	rangeMonths := months
	if since != "" || until != "" {
		rangeMonths = cfg.Months
	}

	start, end, err := stats.TimeRange(since, until, rangeMonths)
	if err != nil {
		return err
	}

	// 确定邮箱过滤条件：优先使用命令行参数，否则使用配置值
	emails := showEmails
	if len(emails) == 0 && strings.TrimSpace(cfg.Email) != "" {
		emails = []string{strings.TrimSpace(cfg.Email)}
	}

	// 收集所有仓库的提交统计
	st, collectErr := stats.CollectStats(repos, emails, start, end)
	if collectErr != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
	}

	showLegend = !showNoLegend
	showSummary = !showNoSummary

	// 根据指定格式输出结果
	switch strings.ToLower(strings.TrimSpace(showFormat)) {
	case "", "table":
		switch {
		case showLegend && showSummary:
			fmt.Fprint(out, stats.RenderHeatmapRange(st, start, end))
		case showLegend && !showSummary:
			fmt.Fprint(out, stats.RenderHeatmapRangeNoSummary(st, start, end))
		case !showLegend && showSummary:
			fmt.Fprint(out, stats.RenderHeatmapRangeNoLegend(st, start, end))
		default:
			fmt.Fprint(out, stats.RenderHeatmapRangeNoLegendNoSummary(st, start, end))
		}
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

type summaryStreak struct {
	Days  int    `json:"days"`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type summaryWeekday struct {
	Weekday string `json:"weekday"`
	Commits int    `json:"commits"`
}

type summaryPeakDay struct {
	Date    string `json:"date,omitempty"`
	Commits int    `json:"commits"`
}

type summaryOut struct {
	TotalCommits      int            `json:"totalCommits"`
	ActiveDays        int            `json:"activeDays"`
	CurrentStreak     int            `json:"currentStreak"`
	LongestStreak     summaryStreak  `json:"longestStreak"`
	MostActiveWeekday summaryWeekday `json:"mostActiveWeekday"`
	PeakDay           summaryPeakDay `json:"peakDay"`
}

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
				Weekday: weekdayAbbrev(s.MostActiveWeekday.Weekday),
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

func weekdayAbbrev(wd time.Weekday) string {
	name := wd.String()
	if len(name) > 3 {
		return name[:3]
	}
	return name
}

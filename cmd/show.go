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
	showEmails []string // 要过滤的邮箱列表
	showMonths int      // 统计的月份数
	showFormat string   // 输出格式：table/json/csv
	showNoLegend bool   // 是否隐藏图例（仅 table 输出）
	showLegend   bool   // 是否显示图例（仅 table 输出）
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
	cmd.Flags().IntVarP(&showMonths, "months", "m", 0, "Months to include (default: config value)")
	cmd.Flags().StringVarP(&showFormat, "format", "f", "table", "Output format: table/json/csv")
	cmd.Flags().BoolVar(&showNoLegend, "no-legend", false, "Hide legend in table output")
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

	// 确定统计月份数：优先使用命令行参数，否则使用配置值
	months := showMonths
	if months == 0 {
		months = cfg.Months
	}
	if months <= 0 {
		return fmt.Errorf("months must be > 0, got %d", months)
	}

	// 确定邮箱过滤条件：优先使用命令行参数，否则使用配置值
	emails := showEmails
	if len(emails) == 0 && strings.TrimSpace(cfg.Email) != "" {
		emails = []string{strings.TrimSpace(cfg.Email)}
	}

	// 收集所有仓库的提交统计
	st, collectErr := stats.CollectStats(repos, emails, months)
	if collectErr != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
	}

	showLegend = !showNoLegend

	// 根据指定格式输出结果
	switch strings.ToLower(strings.TrimSpace(showFormat)) {
	case "", "table":
		if showLegend {
			fmt.Fprint(out, stats.RenderHeatmap(st, months))
			return nil
		}
		fmt.Fprint(out, stats.RenderHeatmapNoLegend(st, months))
		return nil
	case "json":
		return writeJSON(out, st)
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

// writeJSON 将统计数据以 JSON 格式输出。
// 输出为按日期排序的数组，每个元素包含日期和提交数。
func writeJSON(out io.Writer, st map[time.Time]int) error {
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

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
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

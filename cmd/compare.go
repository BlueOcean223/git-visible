package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"git-visible/internal/config"
	"git-visible/internal/repo"
	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

var (
	compareEmails  []string
	comparePeriods []string
	compareYears   []int
	compareFormat  string
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

func init() {
	compareCmd.Flags().StringArrayVarP(&compareEmails, "email", "e", nil, "Emails to compare (repeatable)")
	compareCmd.Flags().StringArrayVar(&comparePeriods, "period", nil, "Periods to compare (repeatable): YYYY, YYYY-HN, YYYY-QN, YYYY-MM")
	compareCmd.Flags().IntSliceVar(&compareYears, "year", nil, "Years to compare (repeatable; shortcut for --period YYYY)")
	compareCmd.Flags().StringVarP(&compareFormat, "format", "f", "table", "Output format: table/json/csv")

	compareCmd.MarkFlagsMutuallyExclusive("email", "period")
	compareCmd.MarkFlagsMutuallyExclusive("email", "year")

	rootCmd.AddCommand(compareCmd)
}

type emailCompareItem struct {
	Email   string
	Metrics stats.CompareMetrics
}

type periodCompareItem struct {
	Period  stats.Period
	Metrics stats.CompareMetrics
}

func runCompare(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	repos, err := repo.LoadRepos()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(repos) == 0 {
		fmt.Fprintln(out, "no repositories added")
		return nil
	}

	format := strings.ToLower(strings.TrimSpace(compareFormat))

	emails := cleanNonEmpty(compareEmails)
	periodArgs := cleanNonEmpty(append(append([]string{}, comparePeriods...), yearsToPeriods(compareYears)...))

	switch {
	case len(emails) > 0:
		if len(emails) < 2 {
			return fmt.Errorf("at least 2 emails are required to compare")
		}

		start, end, err := stats.TimeRange("", "", cfg.Months)
		if err != nil {
			return err
		}

		items, collectErr := collectCompareByEmail(repos, emails, start, end)
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

		collectEmails := []string(nil)
		if strings.TrimSpace(cfg.Email) != "" {
			collectEmails = []string{strings.TrimSpace(cfg.Email)}
		}

		items, collectErr := collectCompareByPeriod(repos, periods, collectEmails)
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

func writeCompareEmailTable(out io.Writer, items []emailCompareItem) error {
	if len(items) == 0 {
		return nil
	}

	metricLabels := []string{
		"Total commits",
		"Active days",
		"Avg commits/day",
		"Most active day",
		"Longest streak",
	}

	values := make([][]string, len(metricLabels))
	for i := range values {
		values[i] = make([]string, 0, len(items))
	}

	for _, it := range items {
		values[0] = append(values[0], fmt.Sprintf("%d", it.Metrics.TotalCommits))
		values[1] = append(values[1], fmt.Sprintf("%d", it.Metrics.ActiveDays))
		values[2] = append(values[2], fmt.Sprintf("%.1f", it.Metrics.AvgCommitsPerDay))
		values[3] = append(values[3], mostActiveDayLabel(it.Metrics))
		values[4] = append(values[4], streakLabel(it.Metrics.LongestStreakDays))
	}

	headers := make([]string, 0, len(items))
	for _, it := range items {
		headers = append(headers, it.Email)
	}

	return writeCompareMatrixTable(out, headers, metricLabels, values)
}

func writeComparePeriodTable(out io.Writer, items []periodCompareItem) error {
	if len(items) == 0 {
		return nil
	}

	metricLabels := []string{
		"Total commits",
		"Active days",
		"Avg commits/day",
	}

	values := make([][]string, len(metricLabels))
	for i := range values {
		values[i] = make([]string, 0, len(items)+len(items)-1)
	}

	headers := make([]string, 0, len(items)+len(items)-1)

	// Columns are interleaved as: P1, P2, Change, P3, Change, ...
	for i, it := range items {
		if i == 0 {
			headers = append(headers, it.Period.Label)
			continue
		}
		headers = append(headers, it.Period.Label, "Change")
	}

	prev := items[0].Metrics
	appendValues := func(cur stats.CompareMetrics, changeTotal, changeActive, changeAvg stats.PercentChange) {
		values[0] = append(values[0], fmt.Sprintf("%d", cur.TotalCommits))
		values[1] = append(values[1], fmt.Sprintf("%d", cur.ActiveDays))
		values[2] = append(values[2], fmt.Sprintf("%.1f", cur.AvgCommitsPerDay))

		values[0] = append(values[0], percentLabel(changeTotal))
		values[1] = append(values[1], percentLabel(changeActive))
		values[2] = append(values[2], percentLabel(changeAvg))
	}

	// First period: values only.
	values[0] = append(values[0], fmt.Sprintf("%d", prev.TotalCommits))
	values[1] = append(values[1], fmt.Sprintf("%d", prev.ActiveDays))
	values[2] = append(values[2], fmt.Sprintf("%.1f", prev.AvgCommitsPerDay))

	for i := 1; i < len(items); i++ {
		cur := items[i].Metrics
		appendValues(
			cur,
			stats.CalculatePercentChange(float64(prev.TotalCommits), float64(cur.TotalCommits)),
			stats.CalculatePercentChange(float64(prev.ActiveDays), float64(cur.ActiveDays)),
			stats.CalculatePercentChange(prev.AvgCommitsPerDay, cur.AvgCommitsPerDay),
		)
		prev = cur
	}

	return writeCompareMatrixTable(out, headers, metricLabels, values)
}

func writeCompareMatrixTable(out io.Writer, headers []string, rowLabels []string, values [][]string) error {
	if len(headers) == 0 || len(rowLabels) == 0 || len(values) != len(rowLabels) {
		return nil
	}

	sep := "    "

	labelWidth := 0
	for _, l := range rowLabels {
		labelWidth = max(labelWidth, len(l))
	}

	colWidths := make([]int, len(headers))
	for j := range headers {
		colWidths[j] = len(headers[j])
		for i := range values {
			if j < len(values[i]) {
				colWidths[j] = max(colWidths[j], len(values[i][j]))
			}
		}
	}

	// Header row.
	fmt.Fprintf(out, "%-*s%s", labelWidth, "", sep)
	for j, h := range headers {
		fmt.Fprintf(out, "%-*s", colWidths[j], h)
		if j < len(headers)-1 {
			fmt.Fprint(out, sep)
		}
	}
	fmt.Fprint(out, "\n")

	ruleLen := labelWidth + len(sep)
	for j := range headers {
		ruleLen += colWidths[j]
		if j < len(headers)-1 {
			ruleLen += len(sep)
		}
	}
	fmt.Fprintln(out, strings.Repeat("─", ruleLen))

	// Rows.
	for i, label := range rowLabels {
		fmt.Fprintf(out, "%-*s%s", labelWidth, label, sep)
		for j := range headers {
			cell := ""
			if j < len(values[i]) {
				cell = values[i][j]
			}
			fmt.Fprintf(out, "%*s", colWidths[j], cell)
			if j < len(headers)-1 {
				fmt.Fprint(out, sep)
			}
		}
		fmt.Fprint(out, "\n")
	}

	return nil
}

func mostActiveDayLabel(m stats.CompareMetrics) string {
	if m.MostActiveWeekdayCommits <= 0 {
		return "-"
	}
	return weekdayAbbrev(m.MostActiveWeekday)
}

func streakLabel(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func percentLabel(pc stats.PercentChange) string {
	if !pc.Defined {
		return "N/A"
	}
	p := round1(pc.Percent)
	sign := ""
	if p > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.1f%%", sign, p)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

type compareJSONOutput struct {
	Dimension string             `json:"dimension"`
	Items     []compareJSONItem  `json:"items"`
	Changes   []compareJSONDelta `json:"changes,omitempty"`
}

type compareJSONItem struct {
	Label              string  `json:"label"`
	Start              string  `json:"start,omitempty"`
	End                string  `json:"end,omitempty"`
	TotalCommits       int     `json:"totalCommits"`
	ActiveDays         int     `json:"activeDays"`
	AvgCommitsPerDay   float64 `json:"avgCommitsPerDay"`
	MostActiveDay      string  `json:"mostActiveDay,omitempty"`
	LongestStreakDays  int     `json:"longestStreakDays,omitempty"`
	LongestStreakLabel string  `json:"longestStreak,omitempty"`
}

type compareJSONDelta struct {
	From                string   `json:"from"`
	To                  string   `json:"to"`
	TotalCommitsPercent *float64 `json:"totalCommitsPercent"`
	ActiveDaysPercent   *float64 `json:"activeDaysPercent"`
	AvgCommitsPerDayPct *float64 `json:"avgCommitsPerDayPercent"`
}

func writeCompareEmailJSON(out io.Writer, items []emailCompareItem) error {
	outItems := make([]compareJSONItem, 0, len(items))
	for _, it := range items {
		outItems = append(outItems, compareJSONItem{
			Label:              it.Email,
			TotalCommits:       it.Metrics.TotalCommits,
			ActiveDays:         it.Metrics.ActiveDays,
			AvgCommitsPerDay:   it.Metrics.AvgCommitsPerDay,
			MostActiveDay:      mostActiveDayLabel(it.Metrics),
			LongestStreakDays:  it.Metrics.LongestStreakDays,
			LongestStreakLabel: streakLabel(it.Metrics.LongestStreakDays),
		})
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(compareJSONOutput{
		Dimension: "email",
		Items:     outItems,
	})
}

func writeComparePeriodJSON(out io.Writer, items []periodCompareItem) error {
	outItems := make([]compareJSONItem, 0, len(items))
	for _, it := range items {
		outItems = append(outItems, compareJSONItem{
			Label:              it.Period.Label,
			Start:              it.Period.Start.Format("2006-01-02"),
			End:                it.Period.End.Format("2006-01-02"),
			TotalCommits:       it.Metrics.TotalCommits,
			ActiveDays:         it.Metrics.ActiveDays,
			AvgCommitsPerDay:   it.Metrics.AvgCommitsPerDay,
			MostActiveDay:      mostActiveDayLabel(it.Metrics),
			LongestStreakDays:  it.Metrics.LongestStreakDays,
			LongestStreakLabel: streakLabel(it.Metrics.LongestStreakDays),
		})
	}

	deltas := make([]compareJSONDelta, 0, max(len(items)-1, 0))
	for i := 1; i < len(items); i++ {
		prev := items[i-1].Metrics
		cur := items[i].Metrics

		total := percentPtr(stats.CalculatePercentChange(float64(prev.TotalCommits), float64(cur.TotalCommits)))
		active := percentPtr(stats.CalculatePercentChange(float64(prev.ActiveDays), float64(cur.ActiveDays)))
		avg := percentPtr(stats.CalculatePercentChange(prev.AvgCommitsPerDay, cur.AvgCommitsPerDay))

		deltas = append(deltas, compareJSONDelta{
			From:                items[i-1].Period.Label,
			To:                  items[i].Period.Label,
			TotalCommitsPercent: total,
			ActiveDaysPercent:   active,
			AvgCommitsPerDayPct: avg,
		})
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(compareJSONOutput{
		Dimension: "period",
		Items:     outItems,
		Changes:   deltas,
	})
}

func percentPtr(pc stats.PercentChange) *float64 {
	if !pc.Defined {
		return nil
	}
	v := pc.Percent
	return &v
}

func writeCompareEmailCSV(out io.Writer, items []emailCompareItem) error {
	if len(items) == 0 {
		return nil
	}

	w := csv.NewWriter(out)
	header := []string{"metric"}
	for _, it := range items {
		header = append(header, it.Email)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	rows := []struct {
		label string
		val   func(m stats.CompareMetrics) string
	}{
		{"totalCommits", func(m stats.CompareMetrics) string { return fmt.Sprintf("%d", m.TotalCommits) }},
		{"activeDays", func(m stats.CompareMetrics) string { return fmt.Sprintf("%d", m.ActiveDays) }},
		{"avgCommitsPerDay", func(m stats.CompareMetrics) string { return fmt.Sprintf("%.1f", m.AvgCommitsPerDay) }},
		{"mostActiveDay", func(m stats.CompareMetrics) string { return mostActiveDayLabel(m) }},
		{"longestStreakDays", func(m stats.CompareMetrics) string { return fmt.Sprintf("%d", m.LongestStreakDays) }},
	}

	for _, row := range rows {
		r := []string{row.label}
		for _, it := range items {
			r = append(r, row.val(it.Metrics))
		}
		if err := w.Write(r); err != nil {
			return err
		}
	}

	w.Flush()
	return w.Error()
}

func writeComparePeriodCSV(out io.Writer, items []periodCompareItem) error {
	if len(items) == 0 {
		return nil
	}

	w := csv.NewWriter(out)

	header := []string{"metric", items[0].Period.Label}
	for i := 1; i < len(items); i++ {
		header = append(header,
			items[i].Period.Label,
			fmt.Sprintf("change(%s->%s)", items[i-1].Period.Label, items[i].Period.Label),
		)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	writeMetric := func(metric string, values []string) error {
		row := append([]string{metric}, values...)
		return w.Write(row)
	}

	totalValues := []string{fmt.Sprintf("%d", items[0].Metrics.TotalCommits)}
	activeValues := []string{fmt.Sprintf("%d", items[0].Metrics.ActiveDays)}
	avgValues := []string{fmt.Sprintf("%.1f", items[0].Metrics.AvgCommitsPerDay)}

	prev := items[0].Metrics
	for i := 1; i < len(items); i++ {
		cur := items[i].Metrics
		totalValues = append(totalValues,
			fmt.Sprintf("%d", cur.TotalCommits),
			percentLabel(stats.CalculatePercentChange(float64(prev.TotalCommits), float64(cur.TotalCommits))),
		)
		activeValues = append(activeValues,
			fmt.Sprintf("%d", cur.ActiveDays),
			percentLabel(stats.CalculatePercentChange(float64(prev.ActiveDays), float64(cur.ActiveDays))),
		)
		avgValues = append(avgValues,
			fmt.Sprintf("%.1f", cur.AvgCommitsPerDay),
			percentLabel(stats.CalculatePercentChange(prev.AvgCommitsPerDay, cur.AvgCommitsPerDay)),
		)
		prev = cur
	}

	if err := writeMetric("totalCommits", totalValues); err != nil {
		return err
	}
	if err := writeMetric("activeDays", activeValues); err != nil {
		return err
	}
	if err := writeMetric("avgCommitsPerDay", avgValues); err != nil {
		return err
	}

	w.Flush()
	return w.Error()
}

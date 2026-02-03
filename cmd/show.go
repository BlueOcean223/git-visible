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

var (
	showEmails []string
	showMonths int
	showFormat string
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show contribution heatmap",
	Args:  cobra.NoArgs,
	RunE:  runShow,
}

func init() {
	addShowFlags(rootCmd)
	addShowFlags(showCmd)
	rootCmd.AddCommand(showCmd)
}

func addShowFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&showEmails, "email", "e", nil, "Email filter (repeatable)")
	cmd.Flags().IntVarP(&showMonths, "months", "m", 0, "Months to include (default: config value)")
	cmd.Flags().StringVarP(&showFormat, "format", "f", "table", "Output format: table/json/csv")
}

func runShow(cmd *cobra.Command, _ []string) error {
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

	months := showMonths
	if months == 0 {
		months = cfg.Months
	}
	if months <= 0 {
		return fmt.Errorf("months must be > 0, got %d", months)
	}

	emails := showEmails
	if len(emails) == 0 && strings.TrimSpace(cfg.Email) != "" {
		emails = []string{strings.TrimSpace(cfg.Email)}
	}

	st, collectErr := stats.CollectStats(repos, emails, months)
	if collectErr != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
	}

	switch strings.ToLower(strings.TrimSpace(showFormat)) {
	case "", "table":
		fmt.Fprint(out, stats.RenderHeatmap(st, months))
		return nil
	case "json":
		return writeJSON(out, st)
	case "csv":
		return writeCSV(out, st)
	default:
		return fmt.Errorf("unsupported format %q (supported: table, json, csv)", showFormat)
	}
}

type dayStat struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func writeJSON(out io.Writer, st map[time.Time]int) error {
	keys := make([]time.Time, 0, len(st))
	for k := range st {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

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

func writeCSV(out io.Writer, st map[time.Time]int) error {
	keys := make([]time.Time, 0, len(st))
	for k := range st {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	w := csv.NewWriter(out)
	if err := w.Write([]string{"date", "count"}); err != nil {
		return err
	}
	for _, k := range keys {
		if err := w.Write([]string{k.Format("2006-01-02"), fmt.Sprintf("%d", st[k])}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

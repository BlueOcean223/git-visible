package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git-visible/internal/config"
	"git-visible/internal/repo"
	"git-visible/internal/stats"

	"github.com/spf13/cobra"
)

var (
	topEmails []string
	topMonths int
	topSince  string
	topUntil  string
	topFormat string

	topNumber int
	topAll    bool
)

// topCmd 实现 top 子命令，用于显示贡献最多的仓库排行榜。
// 用法: git-visible top [-n number|--all] [-e email] [-m months] [--since/--until] [-f format]
var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Show top repositories by commits",
	Args:  cobra.NoArgs,
	RunE:  runTop,
}

func init() {
	topCmd.Flags().IntVarP(&topNumber, "number", "n", 10, "Number of repositories to show")
	topCmd.Flags().BoolVar(&topAll, "all", false, "Show all repositories")
	topCmd.MarkFlagsMutuallyExclusive("number", "all")

	topCmd.Flags().StringArrayVarP(&topEmails, "email", "e", nil, "Email filter (repeatable)")
	topCmd.Flags().IntVarP(&topMonths, "months", "m", 0, "Months to include (default: config value; ignored when --since/--until is set)")
	topCmd.Flags().StringVar(&topSince, "since", "", "Start date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	topCmd.Flags().StringVar(&topUntil, "until", "", "End date (YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)")
	topCmd.Flags().StringVarP(&topFormat, "format", "f", "table", "Output format: table/json/csv")

	rootCmd.AddCommand(topCmd)
}

func runTop(cmd *cobra.Command, _ []string) error {
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

	if !topAll && topNumber <= 0 {
		return fmt.Errorf("number must be > 0, got %d", topNumber)
	}

	since := strings.TrimSpace(topSince)
	until := strings.TrimSpace(topUntil)

	months := topMonths
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

	emails := topEmails
	if len(emails) == 0 && strings.TrimSpace(cfg.Email) != "" {
		emails = []string{strings.TrimSpace(cfg.Email)}
	}

	perRepo, collectErr := stats.CollectStatsPerRepo(repos, emails, start, end)
	if collectErr != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "warning:", collectErr)
	}

	limit := topNumber
	if topAll {
		limit = 0
	}

	ranking := stats.RankRepositories(perRepo, limit)

	format := strings.ToLower(strings.TrimSpace(topFormat))
	switch format {
	case "", "table":
		if ranking.TotalCommits == 0 {
			fmt.Fprintln(out, "no commits found")
			return nil
		}
		return writeTopTable(out, ranking, topRangeLabel(since, until, months, start, end))
	case "json":
		return writeTopJSON(out, ranking)
	case "csv":
		return writeTopCSV(out, ranking)
	default:
		return fmt.Errorf("unsupported format %q (supported: table, json, csv)", topFormat)
	}
}

func topRangeLabel(since, until string, months int, start, end time.Time) string {
	since = strings.TrimSpace(since)
	until = strings.TrimSpace(until)
	if since == "" && until == "" {
		return fmt.Sprintf("last %d months", months)
	}
	return fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func writeTopTable(out io.Writer, ranking stats.RepoRanking, rangeLabel string) error {
	displayPaths := make([]string, 0, len(ranking.Repositories))
	repoWidth := len("Repository")
	for _, r := range ranking.Repositories {
		p := displayRepoPath(r.Repository)
		displayPaths = append(displayPaths, p)
		if len(p) > repoWidth {
			repoWidth = len(p)
		}
	}

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

func writeTopJSON(out io.Writer, ranking stats.RepoRanking) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(ranking)
}

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

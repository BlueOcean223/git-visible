package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"git-visible/internal/config"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type commitSpec struct {
	Email string
	When  time.Time
}

func TestCompare_TwoEmails_TableColumns(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "repo-1")
	base := timeNowLocal().AddDate(0, 0, -10)

	specs := []commitSpec{
		{Email: "work@company.com", When: base.Add(12 * time.Hour)},
		{Email: "work@company.com", When: base.Add(12*time.Hour + 1*time.Minute)},
		{Email: "work@company.com", When: base.AddDate(0, 0, 1).Add(12 * time.Hour)},
		{Email: "personal@gmail.com", When: base.AddDate(0, 0, 2).Add(12 * time.Hour)},
		{Email: "personal@gmail.com", When: base.AddDate(0, 0, 2).Add(12*time.Hour + 1*time.Minute)},
	}

	createRepoWithCommitSpecs(t, repoPath, specs)
	writeReposFile(t, home, []string{repoPath})

	resetCompareFlags()
	compareEmails = []string{"work@company.com", "personal@gmail.com"}
	compareFormat = "table"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runCompare(c, nil)
	require.NoError(t, err, "stderr=%s", errBuf.String())

	s := out.String()
	assert.Contains(t, s, "work@company.com")
	assert.Contains(t, s, "personal@gmail.com")

	// Ensure per-email stats are in separate columns and in the same order as input.
	totalLine := findLineWithPrefix(s, "Total commits")
	require.NotEmpty(t, totalLine)

	re := regexp.MustCompile(`^Total commits\s+(\d+)\s+(\d+)\s*$`)
	m := re.FindStringSubmatch(totalLine)
	require.Len(t, m, 3, "line=%q", totalLine)
	assert.Equal(t, "3", m[1])
	assert.Equal(t, "2", m[2])

	activeLine := findLineWithPrefix(s, "Active days")
	require.NotEmpty(t, activeLine)

	re = regexp.MustCompile(`^Active days\s+(\d+)\s+(\d+)\s*$`)
	m = re.FindStringSubmatch(activeLine)
	require.Len(t, m, 3, "line=%q", activeLine)
	assert.Equal(t, "2", m[1])
	assert.Equal(t, "1", m[2])
}

func TestCompare_ThreePeriods_ChangePercent(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "repo-1")

	var specs []commitSpec
	// 2024-01: total=10, activeDays=5, avg=2.0
	for d := 1; d <= 5; d++ {
		for i := 0; i < 2; i++ {
			specs = append(specs, commitSpec{
				Email: "test@example.com",
				When:  time.Date(2024, time.January, d, 12, i, 0, 0, time.Local),
			})
		}
	}
	// 2024-02: total=20, activeDays=4, avg=5.0
	for d := 1; d <= 4; d++ {
		for i := 0; i < 5; i++ {
			specs = append(specs, commitSpec{
				Email: "test@example.com",
				When:  time.Date(2024, time.February, d, 12, i, 0, 0, time.Local),
			})
		}
	}
	// 2024-03: total=15, activeDays=3, avg=5.0
	for d := 1; d <= 3; d++ {
		for i := 0; i < 5; i++ {
			specs = append(specs, commitSpec{
				Email: "test@example.com",
				When:  time.Date(2024, time.March, d, 12, i, 0, 0, time.Local),
			})
		}
	}

	createRepoWithCommitSpecs(t, repoPath, specs)
	writeReposFile(t, home, []string{repoPath})

	resetCompareFlags()
	comparePeriods = []string{"2024-01", "2024-02", "2024-03"}
	compareFormat = "json"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runCompare(c, nil)
	require.NoError(t, err, "stderr=%s", errBuf.String())

	var got compareJSONOutput
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	require.Equal(t, "period", got.Dimension)
	require.Len(t, got.Items, 3)
	require.Len(t, got.Changes, 2)

	require.Equal(t, "2024-01", got.Items[0].Label)
	require.Equal(t, 10, got.Items[0].TotalCommits)
	require.Equal(t, 5, got.Items[0].ActiveDays)
	require.InDelta(t, 2.0, got.Items[0].AvgCommitsPerDay, 1e-9)

	require.Equal(t, "2024-02", got.Items[1].Label)
	require.Equal(t, 20, got.Items[1].TotalCommits)
	require.Equal(t, 4, got.Items[1].ActiveDays)
	require.InDelta(t, 5.0, got.Items[1].AvgCommitsPerDay, 1e-9)

	require.Equal(t, "2024-03", got.Items[2].Label)
	require.Equal(t, 15, got.Items[2].TotalCommits)
	require.Equal(t, 3, got.Items[2].ActiveDays)
	require.InDelta(t, 5.0, got.Items[2].AvgCommitsPerDay, 1e-9)

	// Changes are computed between consecutive periods.
	require.Equal(t, "2024-01", got.Changes[0].From)
	require.Equal(t, "2024-02", got.Changes[0].To)
	require.NotNil(t, got.Changes[0].TotalCommitsPercent)
	require.NotNil(t, got.Changes[0].ActiveDaysPercent)
	require.NotNil(t, got.Changes[0].AvgCommitsPerDayPct)
	assert.InDelta(t, 100.0, *got.Changes[0].TotalCommitsPercent, 1e-9)
	assert.InDelta(t, -20.0, *got.Changes[0].ActiveDaysPercent, 1e-9)
	assert.InDelta(t, 150.0, *got.Changes[0].AvgCommitsPerDayPct, 1e-9)

	require.Equal(t, "2024-02", got.Changes[1].From)
	require.Equal(t, "2024-03", got.Changes[1].To)
	require.NotNil(t, got.Changes[1].TotalCommitsPercent)
	require.NotNil(t, got.Changes[1].ActiveDaysPercent)
	require.NotNil(t, got.Changes[1].AvgCommitsPerDayPct)
	assert.InDelta(t, -25.0, *got.Changes[1].TotalCommitsPercent, 1e-9)
	assert.InDelta(t, -25.0, *got.Changes[1].ActiveDaysPercent, 1e-9)
	assert.InDelta(t, 0.0, *got.Changes[1].AvgCommitsPerDayPct, 1e-9)
}

func TestCompare_SingleItem_Error(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "repo-1")
	require.NoError(t, os.MkdirAll(repoPath, 0o755))
	writeReposFile(t, home, []string{repoPath})

	resetCompareFlags()
	compareEmails = []string{"only@example.com"}

	c := &cobra.Command{}
	err := runCompare(c, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2")
}

func TestCompare_JSONOutput_Structure(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "repo-1")
	specs := []commitSpec{
		{Email: "a@example.com", When: timeNowLocal().AddDate(0, 0, -3)},
		{Email: "b@example.com", When: timeNowLocal().AddDate(0, 0, -2)},
	}
	createRepoWithCommitSpecs(t, repoPath, specs)
	writeReposFile(t, home, []string{repoPath})

	resetCompareFlags()
	compareEmails = []string{"a@example.com", "b@example.com"}
	compareFormat = "json"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)

	err := runCompare(c, nil)
	require.NoError(t, err)

	var got compareJSONOutput
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	assert.Equal(t, "email", got.Dimension)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "a@example.com", got.Items[0].Label)
	assert.Equal(t, "b@example.com", got.Items[1].Label)
}

func TestCompare_AllRepositoriesFail_ReturnsError(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "missing-repo")})

	resetCompareFlags()
	compareEmails = []string{"a@example.com", "b@example.com"}
	compareFormat = "json"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runCompare(c, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all repositories failed")
}

func TestCompare_PartialFailure_WarnsAndContinues(t *testing.T) {
	home := withTempHome(t)
	repoPath := filepath.Join(home, "code", "repo-1")
	base := timeNowLocal().AddDate(0, 0, -1)
	specs := []commitSpec{
		{Email: "a@example.com", When: base.Add(12 * time.Hour)},
		{Email: "b@example.com", When: base.Add(12*time.Hour + 1*time.Minute)},
	}
	createRepoWithCommitSpecs(t, repoPath, specs)
	writeReposFile(t, home, []string{repoPath, filepath.Join(home, "missing-repo")})

	resetCompareFlags()
	compareEmails = []string{"a@example.com", "b@example.com"}
	compareFormat = "json"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runCompare(c, nil)
	require.NoError(t, err)
	assert.Contains(t, errBuf.String(), "warning:")

	var got compareJSONOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got), "output=%s", out.String())
	require.Len(t, got.Items, 2)
	assert.Equal(t, 1, got.Items[0].TotalCommits)
	assert.Equal(t, 1, got.Items[1].TotalCommits)
}

func TestCompare_EmailFlagsWithWhitespace_AreTrimmed(t *testing.T) {
	home := withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	repoPath := filepath.Join(home, "code", "repo-1")
	base := timeNowLocal().AddDate(0, 0, -1)
	specs := []commitSpec{
		{Email: "user@example.com", When: base.Add(12 * time.Hour)},
		{Email: "user@example.com", When: base.Add(12*time.Hour + 1*time.Minute)},
		{Email: "other@example.com", When: base.Add(13 * time.Hour)},
	}
	createRepoWithCommitSpecs(t, repoPath, specs)
	writeReposFile(t, home, []string{repoPath})

	resetCompareFlags()
	c := &cobra.Command{
		Use:   "compare",
		Short: "Compare contribution stats by email or period",
		Args:  cobra.NoArgs,
		RunE:  runCompare,
	}
	addCompareFlagsForTest(c)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"-e", " user@example.com ", "-e", " other@example.com ", "--format", "json"})

	err := c.Execute()
	require.NoError(t, err, "stderr=%s", errBuf.String())

	var got compareJSONOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got), "output=%s", out.String())
	require.Equal(t, "email", got.Dimension)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "user@example.com", got.Items[0].Label)
	assert.Equal(t, 2, got.Items[0].TotalCommits)
	assert.Equal(t, "other@example.com", got.Items[1].Label)
	assert.Equal(t, 1, got.Items[1].TotalCommits)
}

func resetCompareFlags() {
	compareEmails = nil
	comparePeriods = nil
	compareYears = nil
	compareFormat = "table"
	compareNoCache = false
}

func addCompareFlagsForTest(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&compareEmails, "email", "e", nil, "Emails to compare (repeatable)")
	cmd.Flags().StringArrayVar(&comparePeriods, "period", nil, "Periods to compare (repeatable): YYYY, YYYY-HN, YYYY-QN, YYYY-MM")
	cmd.Flags().IntSliceVar(&compareYears, "year", nil, "Years to compare (repeatable; shortcut for --period YYYY)")
	cmd.Flags().StringVarP(&compareFormat, "format", "f", "table", "Output format: table/json/csv")
	cmd.Flags().BoolVar(&compareNoCache, "no-cache", false, "Disable cache, force full scan")

	cmd.MarkFlagsMutuallyExclusive("email", "period")
	cmd.MarkFlagsMutuallyExclusive("email", "year")
}

func createRepoWithCommitSpecs(tb testing.TB, path string, specs []commitSpec) {
	tb.Helper()

	require.NoError(tb, os.MkdirAll(path, 0o755))

	r, err := git.PlainInit(path, false)
	require.NoError(tb, err)

	wt, err := r.Worktree()
	require.NoError(tb, err)

	for i, spec := range specs {
		fileName := filepath.Join(path, "file.txt")
		content := []byte(fmt.Sprintf("commit %d\n", i))
		require.NoError(tb, os.WriteFile(fileName, content, 0o644))

		_, err := wt.Add("file.txt")
		require.NoError(tb, err)

		sig := &object.Signature{
			Name:  "Test",
			Email: spec.Email,
			When:  spec.When,
		}

		_, err = wt.Commit("test commit", &git.CommitOptions{
			Author:    sig,
			Committer: sig,
		})
		require.NoError(tb, err)
	}
}

func findLineWithPrefix(s, prefix string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func timeNowLocal() time.Time {
	return time.Now().In(time.Local)
}

func BenchmarkCollectCompare_SinglePass(b *testing.B) {
	root := b.TempDir()
	emails := []string{
		"alice@x.com",
		"bob@y.com",
		"carol@z.com",
		"dave@w.com",
		"eve@v.com",
	}

	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	repos := make([]string, 0, 3)
	for repoIdx := 0; repoIdx < 3; repoIdx++ {
		repoPath := filepath.Join(root, fmt.Sprintf("repo-%d", repoIdx))

		specs := make([]commitSpec, 0, 320)
		for i := 0; i < 320; i++ {
			specs = append(specs, commitSpec{
				Email: emails[i%5],
				When:  base.AddDate(0, 0, i%180).Add(time.Duration(i) * time.Minute),
			})
		}
		createRepoWithCommitSpecs(b, repoPath, specs)
		repos = append(repos, repoPath)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.Local)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, err, allFailed := collectCompareByEmail(repos, emails, start, end, nil, false)
		if err != nil {
			b.Fatalf("collect compare failed: %v", err)
		}
		if allFailed {
			b.Fatalf("all repositories failed unexpectedly")
		}
		if len(items) != len(emails) {
			b.Fatalf("unexpected items length: got=%d want=%d", len(items), len(emails))
		}
	}
}

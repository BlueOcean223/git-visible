package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"git-visible/internal/config"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShow_BranchFlagsMutuallyExclusive(t *testing.T) {
	resetShowFlags()

	c := &cobra.Command{
		Use:   "show",
		Short: "Show contribution heatmap",
		Args:  cobra.NoArgs,
		RunE:  runShow,
	}
	addShowFlags(c)

	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)

	c.SetArgs([]string{"--branch", "main", "--all-branches"})
	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch")
	assert.Contains(t, err.Error(), "all-branches")
}

func TestShow_AllRepositoriesFail_ReturnsError(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "missing-repo")})

	resetShowFlags()
	showFormat = "json"
	showSince = "2025-01-01"
	showUntil = "2025-12-31"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runShow(c, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all repositories failed")
}

func TestShow_PartialFailure_WarnsAndContinues(t *testing.T) {
	home := withTempHome(t)
	repoPath := filepath.Join(home, "code", "repo-1")
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithCommits(t, repoPath, 2, "test@example.com", base)
	writeReposFile(t, home, []string{repoPath, filepath.Join(home, "missing-repo")})

	resetShowFlags()
	showFormat = "json"
	showSince = "2025-01-01"
	showUntil = "2025-12-31"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runShow(c, nil)
	require.NoError(t, err)
	assert.Contains(t, errBuf.String(), "warning:")

	var got jsonOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got), "output=%s", out.String())
	assert.NotEmpty(t, got.Days)
}

func TestShow_EmailFlagWithWhitespace_IsTrimmed(t *testing.T) {
	home := withTempHome(t)
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	repoPath := filepath.Join(home, "code", "repo-1")
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithCommits(t, repoPath, 2, "user@example.com", base)
	writeReposFile(t, home, []string{repoPath})

	resetShowFlags()
	c := &cobra.Command{
		Use:   "show",
		Short: "Show contribution heatmap",
		Args:  cobra.NoArgs,
		RunE:  runShow,
	}
	addShowFlags(c)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errBuf)
	c.SetArgs([]string{"-e", " user@example.com ", "--since", "2025-01-01", "--until", "2025-12-31", "--format", "json"})

	err := c.Execute()
	require.NoError(t, err, "stderr=%s", errBuf.String())

	var got jsonOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &got), "output=%s", out.String())
	require.NotEmpty(t, got.Days)

	total := 0
	for _, day := range got.Days {
		total += day.Count
	}
	assert.Equal(t, 2, total)
}

func resetShowFlags() {
	showEmails = nil
	showMonths = 0
	showSince = ""
	showUntil = ""
	showBranch = ""
	showAllBranch = false
	showFormat = "table"
	showNoLegend = false
	showLegend = false
	showNoSummary = false
	showSummary = false
	showNoCache = false
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git-visible/internal/stats"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTop_DefaultTop10_Sorted(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 12)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 12; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, 12-i, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	resetTopFlags()
	topFormat = "json"
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&out)

	err := runTop(c, nil)
	require.NoError(t, err)

	var got stats.RepoRanking
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	require.Len(t, got.Repositories, 10, "default should show top 10")

	wantCommits := []int{12, 11, 10, 9, 8, 7, 6, 5, 4, 3}
	for i, want := range wantCommits {
		assert.Equal(t, want, got.Repositories[i].Commits, "repositories[%d].commits", i)
	}
}

func TestTop_Number3(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 5)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 5; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, 5-i, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	resetTopFlags()
	topFormat = "json"
	topNumber = 3
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&out)

	err := runTop(c, nil)
	require.NoError(t, err)

	var got stats.RepoRanking
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	require.Len(t, got.Repositories, 3, "-n 3 should show 3 repos")

	wantCommits := []int{5, 4, 3}
	for i, want := range wantCommits {
		assert.Equal(t, want, got.Repositories[i].Commits, "repositories[%d].commits", i)
	}
}

func TestTop_All(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 6)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 6; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, i+1, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	resetTopFlags()
	topFormat = "json"
	topAll = true
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&out)

	err := runTop(c, nil)
	require.NoError(t, err)

	var got stats.RepoRanking
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	assert.Len(t, got.Repositories, 6, "--all should show all 6 repos")
}

func TestTop_EmptyStats_NoCommitsFound(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 3)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 3; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, i+1, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	resetTopFlags()
	topFormat = "table"
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&out)

	err := runTop(c, nil)
	require.NoError(t, err)

	assert.Equal(t, "no commits found\n", out.String())
}

func TestTop_PercentSum100(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 12)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 12; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, 12-i, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	resetTopFlags()
	topFormat = "json"
	topAll = true
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&out)

	err := runTop(c, nil)
	require.NoError(t, err)

	var got stats.RepoRanking
	err = json.Unmarshal(out.Bytes(), &got)
	require.NoError(t, err, "output=%s", out.String())

	require.Greater(t, got.TotalCommits, 0, "totalCommits should be > 0")

	sumUnits := 0
	for _, r := range got.Repositories {
		sumUnits += int(math.Round(r.Percent * 10))
	}
	assert.Equal(t, 1000, sumUnits, "percent sum should be 100.0%%")
}

func TestTop_AllRepositoriesFail_ReturnsError(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "missing-repo")})

	resetTopFlags()
	topFormat = "json"
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runTop(c, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all repositories failed")
}

func TestTop_PartialFailure_WarnsAndContinues(t *testing.T) {
	home := withTempHome(t)
	repoPath := filepath.Join(home, "code", "repo-1")
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithCommits(t, repoPath, 3, "test@example.com", base)
	writeReposFile(t, home, []string{repoPath, filepath.Join(home, "missing-repo")})

	resetTopFlags()
	topFormat = "json"
	topAll = true
	topSince = "2025-01-01"
	topUntil = "2025-12-31"

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runTop(c, nil)
	require.NoError(t, err)
	assert.Contains(t, errBuf.String(), "warning:")

	var got stats.RepoRanking
	require.NoError(t, json.Unmarshal(out.Bytes(), &got), "output=%s", out.String())
	require.NotEmpty(t, got.Repositories)
	assert.Equal(t, repoPath, got.Repositories[0].Repository)
}

func resetTopFlags() {
	topEmails = nil
	topMonths = 0
	topSince = ""
	topUntil = ""
	topFormat = "table"
	topNumber = 10
	topAll = false
}

func withTempHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", home))
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	return home
}

func writeReposFile(t *testing.T, home string, repos []string) {
	t.Helper()

	dir := filepath.Join(home, ".config", "git-visible")
	require.NoError(t, os.MkdirAll(dir, 0o700))

	path := filepath.Join(dir, "repos")
	data := strings.Join(repos, "\n")
	if len(repos) > 0 {
		data += "\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
}

func createRepoWithCommits(t *testing.T, path string, commits int, email string, when time.Time) {
	t.Helper()

	require.NoError(t, os.MkdirAll(path, 0o755))

	r, err := git.PlainInit(path, false)
	require.NoError(t, err)

	wt, err := r.Worktree()
	require.NoError(t, err)

	for i := 0; i < commits; i++ {
		fileName := filepath.Join(path, "file.txt")
		content := []byte(fmt.Sprintf("commit %d\n", i))
		require.NoError(t, os.WriteFile(fileName, content, 0o644))

		_, err := wt.Add("file.txt")
		require.NoError(t, err)

		sig := &object.Signature{
			Name:  "Test",
			Email: email,
			When:  when.Add(time.Duration(i) * time.Minute),
		}

		_, err = wt.Commit("test commit", &git.CommitOptions{
			Author:    sig,
			Committer: sig,
		})
		require.NoError(t, err)
	}
}

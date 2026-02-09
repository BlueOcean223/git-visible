package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctor_NormalRepository_AllOK(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "repo-1")
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithCommits(t, repoPath, 2, "test@example.com", base)
	writeReposFile(t, home, []string{repoPath})

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runDoctor(c, nil)
	require.NoError(t, err)

	s := out.String()
	assert.Contains(t, s, "Running diagnostics...")
	assert.Contains(t, s, "✅ Config: OK")
	assert.Contains(t, s, "✅ Repositories: 1/1 valid")
	assert.Contains(t, s, "✅ Branch reachability: OK")
	assert.Contains(t, s, "✅ Permissions: OK")
	assert.Contains(t, s, "✅ Performance: OK")
}

func TestDoctor_BrokenRepository_ReturnsError(t *testing.T) {
	home := withTempHome(t)

	repoPath := filepath.Join(home, "code", "empty-repo")
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)
	writeReposFile(t, home, []string{repoPath})

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err = runDoctor(c, nil)
	require.Error(t, err)
	assert.Contains(t, out.String(), "❌ Branch reachability")
}

func TestDoctor_ManyRepositories_WarnOnly(t *testing.T) {
	home := withTempHome(t)

	repos := make([]string, 0, 51)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	for i := 0; i < 51; i++ {
		repoPath := filepath.Join(home, "code", fmt.Sprintf("repo-%02d", i+1))
		createRepoWithCommits(t, repoPath, 1, "test@example.com", base)
		repos = append(repos, repoPath)
	}
	writeReposFile(t, home, repos)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&out)
	c.SetErr(&errBuf)

	err := runDoctor(c, nil)
	require.NoError(t, err)

	s := out.String()
	assert.Contains(t, s, "✅ Repositories: 51/51 valid")
	assert.Contains(t, s, "⚠️  Performance:")
	assert.Contains(t, s, "large number of repos (51)")
}

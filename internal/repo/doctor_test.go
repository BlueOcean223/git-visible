package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"git-visible/internal/config"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	t.Run("months=0 should fail", func(t *testing.T) {
		issues := config.ValidateConfig(&config.Config{Months: 0, Email: "test@example.com"})
		require.NotEmpty(t, issues)
		assert.Contains(t, strings.Join(issues, "\n"), "months must be > 0")
	})

	t.Run("months=6 should pass", func(t *testing.T) {
		issues := config.ValidateConfig(&config.Config{Months: 6, Email: "test@example.com"})
		assert.Empty(t, issues)
	})

	t.Run("email without @ should fail", func(t *testing.T) {
		issues := config.ValidateConfig(&config.Config{Months: 6, Email: "invalid-email"})
		require.NotEmpty(t, issues)
		assert.Contains(t, strings.Join(issues, "\n"), "invalid email format")
	})
}

func TestCheckBranchReachability(t *testing.T) {
	t.Run("normal repository should pass", func(t *testing.T) {
		repoPath := createRepoWithCommit(t)
		require.NoError(t, CheckBranchReachability(repoPath, ""))
	})

	t.Run("empty repository should fail", func(t *testing.T) {
		repoPath := t.TempDir()
		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		err = CheckBranchReachability(repoPath, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot resolve HEAD")
	})

	t.Run("missing branch should fail", func(t *testing.T) {
		repoPath := createRepoWithCommit(t)
		err := CheckBranchReachability(repoPath, "missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch \"missing\" not found")
	})
}

func TestCheckPermissions(t *testing.T) {
	repoPath := createRepoWithCommit(t)
	require.NoError(t, CheckPermissions(repoPath))

	if runtime.GOOS == "windows" {
		t.Skip("permission mode test is not stable on windows")
	}

	gitDir := filepath.Join(repoPath, ".git")
	require.NoError(t, os.Chmod(gitDir, 0o000))
	t.Cleanup(func() {
		_ = os.Chmod(gitDir, 0o755)
	})

	err := CheckPermissions(repoPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read .git/HEAD")
}

func TestCheckPerformance(t *testing.T) {
	t.Run("51 repositories should warn", func(t *testing.T) {
		repos := make([]string, 0, 51)
		for i := 0; i < 51; i++ {
			repos = append(repos, fmt.Sprintf("/tmp/repo-%d", i))
		}

		warnings := CheckPerformance(repos)
		require.NotEmpty(t, warnings)
		assert.Contains(t, strings.Join(warnings, "\n"), "large number of repos (51)")
	})

	t.Run("large git directory should warn", func(t *testing.T) {
		repoPath := t.TempDir()
		gitDir := filepath.Join(repoPath, ".git", "objects", "pack")
		require.NoError(t, os.MkdirAll(gitDir, 0o755))

		packFile := filepath.Join(gitDir, "pack-test.pack")
		f, err := os.Create(packFile)
		require.NoError(t, err)
		require.NoError(t, f.Truncate(gitSizeWarnThreshold+1))
		require.NoError(t, f.Close())

		warnings := CheckPerformance([]string{repoPath})
		require.NotEmpty(t, warnings)
		assert.Contains(t, strings.Join(warnings, "\n"), "is large")
	})
}

func createRepoWithCommit(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	r, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	wt, err := r.Worktree()
	require.NoError(t, err)

	filePath := filepath.Join(repoPath, "README.md")
	require.NoError(t, os.WriteFile(filePath, []byte("hello\n"), 0o644))

	_, err = wt.Add("README.md")
	require.NoError(t, err)

	sig := &object.Signature{
		Name:  "Test",
		Email: "test@example.com",
		When:  time.Now(),
	}

	_, err = wt.Commit("init", &git.CommitOptions{Author: sig, Committer: sig})
	require.NoError(t, err)

	return repoPath
}

package stats

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectStats_Branch_MainOnly(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithMainAndFeature(t, repoPath, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{Branch: "main"})
	require.NoError(t, err)
	assert.Equal(t, 3, sumCounts(got), "should only include commits reachable from main")
}

func TestCollectStats_Branch_Nonexistent_WarningSkip(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 2, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{Branch: "nonexistent"})
	require.Error(t, err)
	assert.Empty(t, got)
	assert.Contains(t, err.Error(), `branch "nonexistent" not found`)
}

func TestCollectStats_AllBranches_DedupByHash(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithMainAndFeature(t, repoPath, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{AllBranches: true})
	require.NoError(t, err)
	assert.Equal(t, 4, sumCounts(got), "should de-duplicate commits reachable from multiple branches")
}

func TestCollectRepo_AllBranches_PruningIdempotent(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithMainAndFeature(t, repoPath, "test@example.com", base)

	loc := time.Local
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, loc)

	legacy := collectRepoAllBranchesWithoutPruning(t, repoPath, start, end, loc, map[string]struct{}{})
	got, err := collectRepo(repoPath, start, end, loc, map[string]struct{}{}, BranchOption{AllBranches: true})
	require.NoError(t, err)
	assert.Equal(t, legacy, got, "pruning must not change --all-branches results")
}

func TestCollectStats_Branch_MissingInOneRepo_Continue(t *testing.T) {
	repoMain := t.TempDir()
	repoNoMain := t.TempDir()

	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoMain, "main", 2, "test@example.com", base)
	createRepoWithBranchCommits(t, repoNoMain, "develop", 5, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoMain, repoNoMain}, nil, start, end, BranchOption{Branch: "main"})
	require.Error(t, err, "missing branch should be a warning, not fatal")
	assert.Equal(t, 2, sumCounts(got))
	assert.Contains(t, err.Error(), repoNoMain)
	assert.Contains(t, err.Error(), `branch "main" not found`)
}

func createRepoWithMainAndFeature(t *testing.T, repoPath string, email string, base time.Time) {
	t.Helper()

	r := initRepo(t, repoPath)
	wt, err := r.Worktree()
	require.NoError(t, err)

	commitFile(t, wt, repoPath, "file.txt", "init\n", email, base.Add(0*time.Minute))

	require.NoError(t, wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
		Create: true,
	}))

	commitFile(t, wt, repoPath, "file.txt", "main-1\n", email, base.Add(1*time.Minute))

	require.NoError(t, wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	}))
	commitFile(t, wt, repoPath, "file.txt", "feature-1\n", email, base.Add(2*time.Minute))

	require.NoError(t, wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
	}))
	commitFile(t, wt, repoPath, "file.txt", "main-2\n", email, base.Add(3*time.Minute))
}

func createRepoWithBranchCommits(t *testing.T, repoPath, branchName string, commits int, email string, base time.Time) {
	t.Helper()

	r := initRepo(t, repoPath)
	wt, err := r.Worktree()
	require.NoError(t, err)

	if commits <= 0 {
		return
	}

	commitFile(t, wt, repoPath, "file.txt", "init\n", email, base.Add(0*time.Minute))

	require.NoError(t, wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
	}))

	for i := 1; i < commits; i++ {
		commitFile(t, wt, repoPath, "file.txt", fmt.Sprintf("%s-%d\n", branchName, i), email, base.Add(time.Duration(i)*time.Minute))
	}
}

func initRepo(t *testing.T, repoPath string) *git.Repository {
	t.Helper()

	require.NoError(t, os.MkdirAll(repoPath, 0o755))
	r, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)
	return r
}

func commitFile(t *testing.T, wt *git.Worktree, repoPath, name, content, email string, when time.Time) {
	t.Helper()

	fileName := filepath.Join(repoPath, name)
	require.NoError(t, os.WriteFile(fileName, []byte(content), 0o644))

	_, err := wt.Add(name)
	require.NoError(t, err)

	sig := &object.Signature{
		Name:  "Test",
		Email: email,
		When:  when,
	}

	_, err = wt.Commit("test commit", &git.CommitOptions{
		Author:    sig,
		Committer: sig,
	})
	require.NoError(t, err)
}

func sumCounts(stats map[time.Time]int) int {
	total := 0
	for _, v := range stats {
		total += v
	}
	return total
}

func collectRepoAllBranchesWithoutPruning(t *testing.T, repoPath string, start, end time.Time, loc *time.Location, emailSet map[string]struct{}) map[time.Time]int {
	t.Helper()

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	startPoints, err := collectStartPoints(repo, repoPath, BranchOption{AllBranches: true})
	require.NoError(t, err)

	out := make(map[time.Time]int)
	seenCommits := make(map[plumbing.Hash]struct{})

	for _, from := range startPoints {
		iterator, err := repo.Log(&git.LogOptions{From: from})
		require.NoError(t, err)

		iterErr := iterator.ForEach(func(c *object.Commit) error {
			commitDay := beginningOfDay(c.Author.When, loc)
			if commitDay.After(end) {
				return nil
			}
			if commitDay.Before(start) {
				return storer.ErrStop
			}

			if len(emailSet) > 0 {
				if _, ok := emailSet[c.Author.Email]; !ok {
					return nil
				}
			}

			if _, seen := seenCommits[c.Hash]; seen {
				// Legacy behavior: deduplicate counting only, but keep traversing.
				return nil
			}
			seenCommits[c.Hash] = struct{}{}
			out[commitDay]++
			return nil
		})
		iterator.Close()
		require.True(t, iterErr == nil || errors.Is(iterErr, storer.ErrStop))
	}

	return out
}

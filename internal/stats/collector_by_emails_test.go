package stats

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectStatsByEmails_MemoryRepoBuckets(t *testing.T) {
	loc := time.UTC

	repoA, wtA := initMemoryGitRepo(t)
	repoB, wtB := initMemoryGitRepo(t)

	alice := "alice@x.com"
	bob := "bob@y.com"
	carol := "carol@z.com"
	dave := "dave@w.com"

	commitMemoryFile(t, wtA, "a.txt", "alice-1\n", alice, time.Date(2024, 1, 1, 12, 0, 0, 0, loc))
	commitMemoryFile(t, wtA, "a.txt", "bob-1\n", bob, time.Date(2024, 1, 2, 12, 0, 0, 0, loc))
	commitMemoryFile(t, wtA, "a.txt", "dave-1\n", dave, time.Date(2024, 1, 2, 12, 1, 0, 0, loc))

	commitMemoryFile(t, wtB, "b.txt", "alice-2\n", alice, time.Date(2024, 1, 3, 12, 0, 0, 0, loc))
	commitMemoryFile(t, wtB, "b.txt", "alice-3\n", alice, time.Date(2024, 1, 3, 12, 1, 0, 0, loc))
	commitMemoryFile(t, wtB, "b.txt", "bob-2\n", bob, time.Date(2024, 1, 3, 12, 2, 0, 0, loc))
	commitMemoryFile(t, wtB, "b.txt", "carol-1\n", carol, time.Date(2024, 1, 4, 12, 0, 0, 0, loc))

	repos := map[string]*git.Repository{
		"mem://repo-a": repoA,
		"mem://repo-b": repoB,
	}

	originalCollect := collectRepoByEmailsFn
	collectRepoByEmailsFn = func(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string) (map[string]map[int]int, error) {
		repo, ok := repos[repoPath]
		if !ok {
			return nil, fmt.Errorf("unknown repo %s", repoPath)
		}
		return collectRepoByEmailsFromRepository(repo, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail)
	}
	t.Cleanup(func() {
		collectRepoByEmailsFn = originalCollect
	})

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 1, 5, 0, 0, 0, 0, loc)

	got, err := CollectStatsByEmails(
		[]string{"mem://repo-a", "mem://repo-b"},
		[]string{alice, bob, carol},
		start,
		end,
		BranchOption{},
		nil,
	)
	require.NoError(t, err)

	require.Contains(t, got, alice)
	require.Contains(t, got, bob)
	require.Contains(t, got, carol)
	assert.NotContains(t, got, dave)

	assert.Equal(t, 3, sumCounts(got[alice]))
	assert.Equal(t, 2, sumCounts(got[bob]))
	assert.Equal(t, 1, sumCounts(got[carol]))

	assert.Equal(t, 1, got[alice][time.Date(2024, 1, 1, 0, 0, 0, 0, loc)])
	assert.Equal(t, 2, got[alice][time.Date(2024, 1, 3, 0, 0, 0, 0, loc)])
	assert.Equal(t, 1, got[bob][time.Date(2024, 1, 2, 0, 0, 0, 0, loc)])
	assert.Equal(t, 1, got[bob][time.Date(2024, 1, 3, 0, 0, 0, 0, loc)])
	assert.Equal(t, 1, got[carol][time.Date(2024, 1, 4, 0, 0, 0, 0, loc)])
}

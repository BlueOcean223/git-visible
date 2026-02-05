package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankRepositories_Empty(t *testing.T) {
	result := RankRepositories(nil, 10)

	assert.Empty(t, result.Repositories)
	assert.Equal(t, 0, result.TotalCommits)
}

func TestRankRepositories_SortedByCommitsDescending(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5},
		"/repo/b": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 10},
		"/repo/c": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 3},
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 3)
	assert.Equal(t, "/repo/b", result.Repositories[0].Repository)
	assert.Equal(t, 10, result.Repositories[0].Commits)
	assert.Equal(t, "/repo/a", result.Repositories[1].Repository)
	assert.Equal(t, 5, result.Repositories[1].Commits)
	assert.Equal(t, "/repo/c", result.Repositories[2].Repository)
	assert.Equal(t, 3, result.Repositories[2].Commits)
}

func TestRankRepositories_TieBreakByName(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/z": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5},
		"/repo/a": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5},
		"/repo/m": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5},
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 3)
	assert.Equal(t, "/repo/a", result.Repositories[0].Repository)
	assert.Equal(t, "/repo/m", result.Repositories[1].Repository)
	assert.Equal(t, "/repo/z", result.Repositories[2].Repository)
}

func TestRankRepositories_LimitTop3(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 1},
		"/repo/b": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 2},
		"/repo/c": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 3},
		"/repo/d": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 4},
		"/repo/e": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5},
	}

	result := RankRepositories(perRepo, 3)

	require.Len(t, result.Repositories, 3)
	assert.Equal(t, 5, result.Repositories[0].Commits)
	assert.Equal(t, 4, result.Repositories[1].Commits)
	assert.Equal(t, 3, result.Repositories[2].Commits)
	assert.Equal(t, 12, result.TotalCommits) // 5+4+3 = 12 (only top 3)
}

func TestRankRepositories_PercentSumTo100(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 33},
		"/repo/b": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 33},
		"/repo/c": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 34},
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 3)

	// Percentages should sum to exactly 100.0
	sumPercent := 0.0
	for _, r := range result.Repositories {
		sumPercent += r.Percent
	}
	assert.InDelta(t, 100.0, sumPercent, 0.01, "percent sum should be 100.0")
}

func TestRankRepositories_MultiDayCommits(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {
			time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 5,
			time.Date(2025, 1, 2, 0, 0, 0, 0, time.Local): 3,
			time.Date(2025, 1, 3, 0, 0, 0, 0, time.Local): 2,
		},
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 1)
	assert.Equal(t, 10, result.Repositories[0].Commits) // 5+3+2
	assert.Equal(t, 10, result.TotalCommits)
	assert.Equal(t, 100.0, result.Repositories[0].Percent)
}

func TestRankRepositories_ZeroCommitsRepo(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local): 10},
		"/repo/b": {}, // no commits
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 2)
	assert.Equal(t, "/repo/a", result.Repositories[0].Repository)
	assert.Equal(t, 10, result.Repositories[0].Commits)
	assert.Equal(t, "/repo/b", result.Repositories[1].Repository)
	assert.Equal(t, 0, result.Repositories[1].Commits)
}

func TestRankRepositories_AllZeroCommits(t *testing.T) {
	perRepo := map[string]map[time.Time]int{
		"/repo/a": {},
		"/repo/b": {},
	}

	result := RankRepositories(perRepo, 0)

	require.Len(t, result.Repositories, 2)
	assert.Equal(t, 0, result.TotalCommits)
	// When all commits are 0, percentages should all be 0
	for _, r := range result.Repositories {
		assert.Equal(t, 0.0, r.Percent)
	}
}

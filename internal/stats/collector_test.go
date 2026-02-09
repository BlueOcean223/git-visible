package stats

import (
	"fmt"
	"testing"
	"time"

	"git-visible/internal/config"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginningOfDay(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name  string
		input time.Time
		want  time.Time
	}{
		{
			name:  "morning time",
			input: time.Date(2024, 6, 15, 9, 30, 45, 123, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:  "midnight",
			input: time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of day",
			input: time.Date(2024, 6, 15, 23, 59, 59, 999999999, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := beginningOfDay(tt.input, loc)
			assert.True(t, got.Equal(tt.want), "beginningOfDay() = %v, want %v", got, tt.want)
		})
	}
}

func TestBeginningOfDay_DifferentTimezones(t *testing.T) {
	utc := time.UTC
	inputUTC := time.Date(2024, 6, 15, 10, 30, 0, 0, utc)

	loc := time.Local
	got := beginningOfDay(inputUTC, loc)

	assert.Equal(t, loc, got.Location(), "should be in local timezone")
	assert.Equal(t, 0, got.Hour(), "hour should be 0")
	assert.Equal(t, 0, got.Minute(), "minute should be 0")
	assert.Equal(t, 0, got.Second(), "second should be 0")
}

func TestHeatmapStart(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name   string
		now    time.Time
		months int
	}{
		{
			name:   "6 months",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 6,
		},
		{
			name:   "12 months",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 12,
		},
		{
			name:   "1 month",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := heatmapStart(tt.now, tt.months)

			assert.Equal(t, time.Sunday, got.Weekday(), "should start on Sunday")
			assert.Equal(t, 0, got.Hour(), "hour should be 0")
			assert.Equal(t, 0, got.Minute(), "minute should be 0")
			assert.Equal(t, 0, got.Second(), "second should be 0")

			expectedEarliest := tt.now.AddDate(0, -tt.months, -7)
			assert.False(t, got.Before(expectedEarliest), "should not be before expected earliest")
		})
	}
}

func TestCollectStats_InvalidMonths(t *testing.T) {
	_, err := CollectStatsMonths(nil, nil, 0)
	assert.Error(t, err, "months=0 should return error")

	_, err = CollectStatsMonths(nil, nil, -1)
	assert.Error(t, err, "months=-1 should return error")
}

func TestCollectStats_EmptyRepos(t *testing.T) {
	stats, err := CollectStatsMonths([]string{}, nil, 6)
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestCollectStats_NonExistentRepo(t *testing.T) {
	stats, err := CollectStatsMonths([]string{"/non/existent/repo"}, nil, 6)

	assert.Error(t, err, "non-existent repo should return error")
	assert.Empty(t, stats)
}

func TestMaxConcurrency(t *testing.T) {
	assert.GreaterOrEqual(t, maxConcurrency, 1, "maxConcurrency should be >= 1")
}

func TestCollectStats_AllReposFail_ReturnsErrorAndEmptyMap(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{"/non/existent/repo-a", "/non/existent/repo-b"}, nil, start, end, BranchOption{}, nil)
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestCollectStats_PartialRepoFailure_ReturnsErrorAndPartialData(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 2, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath, "/non/existent/repo"}, nil, start, end, BranchOption{}, nil)
	require.Error(t, err)
	assert.NotEmpty(t, got)
	assert.Greater(t, sumCounts(got), 0)
}

func TestCollectStats_AllReposSuccess_ReturnsNilError(t *testing.T) {
	repoA := t.TempDir()
	repoB := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoA, "main", 2, "test@example.com", base)
	createRepoWithBranchCommits(t, repoB, "main", 3, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoA, repoB}, nil, start, end, BranchOption{}, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Greater(t, sumCounts(got), 0)
}

func TestCollectStats_AliasMergeCountsCombined(t *testing.T) {
	loc := time.UTC
	repo, wt := initMemoryGitRepo(t)

	commitMemoryFile(t, wt, "activity.txt", "alice-work-1\n", "alice@company.com", time.Date(2024, 1, 1, 10, 0, 0, 0, loc))
	commitMemoryFile(t, wt, "activity.txt", "alice-gmail\n", "alice@gmail.com", time.Date(2024, 1, 2, 10, 0, 0, 0, loc))
	commitMemoryFile(t, wt, "activity.txt", "alice-work-2\n", "alice@company.com", time.Date(2024, 1, 3, 10, 0, 0, 0, loc))

	repos := map[string]*git.Repository{
		"mem://alias-repo": repo,
	}

	originalCollect := collectRepoFn
	collectRepoFn = func(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string) (map[int]int, error) {
		repository, ok := repos[repoPath]
		if !ok {
			return nil, fmt.Errorf("unknown repo %s", repoPath)
		}
		return collectRepoFromRepository(repository, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail)
	}
	t.Cleanup(func() {
		collectRepoFn = originalCollect
	})

	cfg := &config.Config{
		Aliases: []config.Alias{
			{
				Name:   "Alice",
				Emails: []string{"alice@company.com", "alice@gmail.com"},
			},
		},
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 1, 3, 0, 0, 0, 0, loc)
	got, err := CollectStats([]string{"mem://alias-repo"}, []string{"alice@company.com"}, start, end, BranchOption{}, cfg.NormalizeEmail)
	require.NoError(t, err)
	assert.Equal(t, 3, sumCounts(got))
}

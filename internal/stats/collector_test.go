package stats

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git-visible/internal/cache"
	"git-visible/internal/config"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Basic unit tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Error handling: all-fail / partial-fail / all-success
// ---------------------------------------------------------------------------

func TestCollectStats_AllReposFail_ReturnsErrorAndEmptyMap(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{"/non/existent/repo-a", "/non/existent/repo-b"}, nil, start, end, BranchOption{}, nil, true)
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestCollectStats_PartialRepoFailure_ReturnsErrorAndPartialData(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 2, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath, "/non/existent/repo"}, nil, start, end, BranchOption{}, nil, true)
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

	got, err := CollectStats([]string{repoA, repoB}, nil, start, end, BranchOption{}, nil, true)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Greater(t, sumCounts(got), 0)
}

func TestCollectStats_FirstCollectionCreatesCacheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 3, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{}, nil, true)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)
	headRef, err := repo.Head()
	require.NoError(t, err)

	startDayKey := dayKeyFromTime(start, start.Location())
	endDayKey := dayKeyFromTime(end, end.Location())
	key := buildRepoCacheKey(repoPath, headRef.Hash().String(), startDayKey, endDayKey, nil, BranchOption{})

	cachePath := filepath.Join(home, ".config", "git-visible", "cache", key.String())
	assert.FileExists(t, cachePath)

	entry, err := cache.LoadCache(key)
	require.NoError(t, err)
	assert.Equal(t, toCachedStats(toDayKeyStats(got)), entry.Stats)
}

func TestCollectStats_SecondCollectionHitsCacheWhenHeadUnchanged(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 2, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	first, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{}, nil, true)
	require.NoError(t, err)

	originalScan := collectRepoFromRepositoryFn
	collectRepoFromRepositoryFn = func(_ *git.Repository, _ string, _, _ int, _ *time.Location, _ map[string]struct{}, _ BranchOption, _ func(string) string) (map[int]int, error) {
		return nil, fmt.Errorf("scan should be skipped on cache hit")
	}
	t.Cleanup(func() {
		collectRepoFromRepositoryFn = originalScan
	})

	second, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{}, nil, true)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

// ---------------------------------------------------------------------------
// Alias merging
// ---------------------------------------------------------------------------

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
	collectRepoFn = func(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, _ bool) (map[int]int, error) {
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
	got, err := CollectStats([]string{"mem://alias-repo"}, []string{"alice@company.com"}, start, end, BranchOption{}, cfg.NormalizeEmail, true)
	require.NoError(t, err)
	assert.Equal(t, 3, sumCounts(got))
}

// ---------------------------------------------------------------------------
// Out-of-order author timestamps (乱序场景回归)
// ---------------------------------------------------------------------------

func TestCollectRepo_OutOfOrderAuthorWhen_NoMissInRange(t *testing.T) {
	loc := time.UTC
	repo, wt := initMemoryGitRepo(t)
	email := "test@example.com"

	commitOrder := []time.Time{
		time.Date(2024, 1, 8, 10, 0, 0, 0, loc),
		time.Date(2024, 1, 15, 10, 0, 0, 0, loc),
		time.Date(2024, 1, 10, 10, 0, 0, 0, loc),
		time.Date(2024, 1, 20, 10, 0, 0, 0, loc),
		time.Date(2024, 1, 25, 10, 0, 0, 0, loc),
	}
	for i, when := range commitOrder {
		commitMemoryFile(t, wt, "activity.txt", fmt.Sprintf("commit-%d\n", i), email, when)
	}

	startDayKey := dayKeyFromTime(time.Date(2024, 1, 10, 0, 0, 0, 0, loc), loc)
	endDayKey := dayKeyFromTime(time.Date(2024, 1, 20, 0, 0, 0, 0, loc), loc)

	got, err := collectRepoFromRepository(repo, "mem://out-of-order", startDayKey, endDayKey, loc, map[string]struct{}{}, BranchOption{}, nil)
	require.NoError(t, err)

	want := map[int]int{
		20240110: 1,
		20240115: 1,
		20240120: 1,
	}
	assert.Equal(t, want, got)
	assert.Equal(t, 3, sumDayKeyCounts(got))
}

func TestCollectRepo_TimeRangeBoundaryScenarios(t *testing.T) {
	loc := time.UTC
	email := "test@example.com"

	tests := []struct {
		name        string
		commitTimes []time.Time
		start       time.Time
		end         time.Time
		want        map[int]int
	}{
		{
			name: "all in range",
			commitTimes: []time.Time{
				time.Date(2024, 1, 10, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 11, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 12, 12, 0, 0, 0, loc),
			},
			start: time.Date(2024, 1, 10, 0, 0, 0, 0, loc),
			end:   time.Date(2024, 1, 12, 0, 0, 0, 0, loc),
			want: map[int]int{
				20240110: 1,
				20240111: 1,
				20240112: 1,
			},
		},
		{
			name: "all out of range",
			commitTimes: []time.Time{
				time.Date(2024, 1, 1, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 2, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 25, 12, 0, 0, 0, loc),
			},
			start: time.Date(2024, 1, 10, 0, 0, 0, 0, loc),
			end:   time.Date(2024, 1, 20, 0, 0, 0, 0, loc),
			want:  map[int]int{},
		},
		{
			name: "only one in range",
			commitTimes: []time.Time{
				time.Date(2024, 1, 9, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 15, 12, 0, 0, 0, loc),
				time.Date(2024, 1, 21, 12, 0, 0, 0, loc),
			},
			start: time.Date(2024, 1, 10, 0, 0, 0, 0, loc),
			end:   time.Date(2024, 1, 20, 0, 0, 0, 0, loc),
			want: map[int]int{
				20240115: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, wt := initMemoryGitRepo(t)
			for i, when := range tt.commitTimes {
				commitMemoryFile(t, wt, "activity.txt", fmt.Sprintf("%s-%d\n", tt.name, i), email, when)
			}

			startDayKey := dayKeyFromTime(tt.start, loc)
			endDayKey := dayKeyFromTime(tt.end, loc)
			got, err := collectRepoFromRepository(repo, "mem://boundary", startDayKey, endDayKey, loc, map[string]struct{}{}, BranchOption{}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Branch tests
// ---------------------------------------------------------------------------

func TestCollectStats_Branch_MainOnly(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithMainAndFeature(t, repoPath, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{Branch: "main"}, nil, true)
	require.NoError(t, err)
	assert.Equal(t, 3, sumCounts(got), "should only include commits reachable from main")
}

func TestCollectStats_Branch_Nonexistent_WarningSkip(t *testing.T) {
	repoPath := t.TempDir()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.Local)
	createRepoWithBranchCommits(t, repoPath, "main", 2, "test@example.com", base)

	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{Branch: "nonexistent"}, nil, true)
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

	got, err := CollectStats([]string{repoPath}, nil, start, end, BranchOption{AllBranches: true}, nil, true)
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
	startDayKey := dayKeyFromTime(start, loc)
	endDayKey := dayKeyFromTime(end, loc)

	legacy := collectRepoAllBranchesWithoutPruning(t, repoPath, startDayKey, endDayKey, loc, map[string]struct{}{})
	got, err := collectRepo(repoPath, startDayKey, endDayKey, loc, map[string]struct{}{}, BranchOption{AllBranches: true}, nil, false)
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

	got, err := CollectStats([]string{repoMain, repoNoMain}, nil, start, end, BranchOption{Branch: "main"}, nil, true)
	require.Error(t, err, "missing branch should be a warning, not fatal")
	assert.Equal(t, 2, sumCounts(got))
	assert.Contains(t, err.Error(), repoNoMain)
	assert.Contains(t, err.Error(), `branch "main" not found`)
}

// ---------------------------------------------------------------------------
// CollectStatsByEmails (per-email bucketing)
// ---------------------------------------------------------------------------

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
	collectRepoByEmailsFn = func(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, _ bool) (map[string]map[int]int, error) {
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
		true,
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

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkCollectRepo_HotPath(b *testing.B) {
	loc := time.UTC
	emailSet := map[string]struct{}{
		"target@example.com": {},
	}

	linearRepo, linearStart, linearEnd := buildLinearHistoryRepoForBenchmark(b, 2500, false)
	unorderedRepo, unorderedStart, unorderedEnd := buildLinearHistoryRepoForBenchmark(b, 2500, true)
	multiBranchRepo, multiStart, multiEnd := buildMultiBranchRepoForBenchmark(b, 1200, 600, 600)

	benchmarks := []struct {
		name      string
		repo      *git.Repository
		start     int
		end       int
		branchOpt BranchOption
	}{
		{
			name:      "linear-history",
			repo:      linearRepo,
			start:     linearStart,
			end:       linearEnd,
			branchOpt: BranchOption{},
		},
		{
			name:      "rebase-out-of-order",
			repo:      unorderedRepo,
			start:     unorderedStart,
			end:       unorderedEnd,
			branchOpt: BranchOption{},
		},
		{
			name:      "multi-branch",
			repo:      multiBranchRepo,
			start:     multiStart,
			end:       multiEnd,
			branchOpt: BranchOption{AllBranches: true},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				stats, err := collectRepoFromRepository(bm.repo, "mem://"+bm.name, bm.start, bm.end, loc, emailSet, bm.branchOpt, nil)
				if err != nil {
					b.Fatalf("collect failed: %v", err)
				}
				if stats == nil {
					b.Fatal("stats should not be nil")
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test helpers: memory-backed git repos
// ---------------------------------------------------------------------------

func initMemoryGitRepo(tb testing.TB) (*git.Repository, *git.Worktree) {
	tb.Helper()

	repo, err := git.Init(memory.NewStorage(), memfs.New())
	require.NoError(tb, err)

	wt, err := repo.Worktree()
	require.NoError(tb, err)
	return repo, wt
}

func commitMemoryFile(tb testing.TB, wt *git.Worktree, name, content, email string, when time.Time) {
	tb.Helper()

	file, err := wt.Filesystem.Create(name)
	require.NoError(tb, err)
	_, err = file.Write([]byte(content))
	require.NoError(tb, err)
	require.NoError(tb, file.Close())

	_, err = wt.Add(name)
	require.NoError(tb, err)

	sig := &object.Signature{
		Name:  "Test",
		Email: email,
		When:  when,
	}
	_, err = wt.Commit("test commit", &git.CommitOptions{
		Author:    sig,
		Committer: sig,
	})
	require.NoError(tb, err)
}

// ---------------------------------------------------------------------------
// Test helpers: on-disk git repos
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Test helpers: counting & verification
// ---------------------------------------------------------------------------

func sumCounts(stats map[time.Time]int) int {
	total := 0
	for _, v := range stats {
		total += v
	}
	return total
}

func toDayKeyStats(stats map[time.Time]int) map[int]int {
	out := make(map[int]int, len(stats))
	for day, count := range stats {
		out[dayKeyFromTime(day, day.Location())] = count
	}
	return out
}

func sumDayKeyCounts(stats map[int]int) int {
	total := 0
	for _, c := range stats {
		total += c
	}
	return total
}

func collectRepoAllBranchesWithoutPruning(t *testing.T, repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}) map[int]int {
	t.Helper()

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	startPoints, err := collectStartPoints(repo, repoPath, BranchOption{AllBranches: true})
	require.NoError(t, err)

	out := make(map[int]int)
	seenCommits := make(map[plumbing.Hash]struct{})

	for _, from := range startPoints {
		iterator, err := repo.Log(&git.LogOptions{From: from})
		require.NoError(t, err)

		iterErr := iterator.ForEach(func(c *object.Commit) error {
			commitDayKey := dayKeyFromTime(c.Author.When, loc)
			if commitDayKey > endDayKey {
				return nil
			}
			if commitDayKey < startDayKey {
				return storer.ErrStop
			}

			if len(emailSet) > 0 {
				if _, ok := emailSet[c.Author.Email]; !ok {
					return nil
				}
			}

			if _, seen := seenCommits[c.Hash]; seen {
				return nil
			}
			seenCommits[c.Hash] = struct{}{}
			out[commitDayKey]++
			return nil
		})
		iterator.Close()
		require.True(t, iterErr == nil || errors.Is(iterErr, storer.ErrStop))
	}

	return out
}

// ---------------------------------------------------------------------------
// Benchmark helpers
// ---------------------------------------------------------------------------

func buildLinearHistoryRepoForBenchmark(tb testing.TB, commitCount int, outOfOrder bool) (*git.Repository, int, int) {
	tb.Helper()
	loc := time.UTC
	repo, wt := initMemoryGitRepo(tb)
	base := time.Date(2020, 1, 1, 12, 0, 0, 0, loc)

	for i := 0; i < commitCount; i++ {
		when := base.AddDate(0, 0, i)
		if outOfOrder && i%17 == 0 {
			when = when.AddDate(0, 0, -45)
		}

		email := "target@example.com"
		if i%4 == 0 {
			email = "other@example.com"
		}
		commitMemoryFile(tb, wt, "linear.txt", fmt.Sprintf("linear-%d\n", i), email, when)
	}

	start := dayKeyFromTime(base.AddDate(0, 0, commitCount-120), loc)
	end := dayKeyFromTime(base.AddDate(0, 0, commitCount-1), loc)
	return repo, start, end
}

func buildMultiBranchRepoForBenchmark(tb testing.TB, shared, featureOnly, mainOnly int) (*git.Repository, int, int) {
	tb.Helper()
	loc := time.UTC
	repo, wt := initMemoryGitRepo(tb)
	base := time.Date(2021, 1, 1, 12, 0, 0, 0, loc)

	commitIndex := 0
	commitNext := func(fileName string, email string) {
		when := base.AddDate(0, 0, commitIndex)
		commitMemoryFile(tb, wt, fileName, fmt.Sprintf("%s-%d\n", fileName, commitIndex), email, when)
		commitIndex++
	}

	for i := 0; i < shared; i++ {
		email := "target@example.com"
		if i%3 == 0 {
			email = "other@example.com"
		}
		commitNext("shared.txt", email)
	}

	head, err := repo.Head()
	require.NoError(tb, err)
	mainBranch := head.Name()

	require.NoError(tb, wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	}))
	for i := 0; i < featureOnly; i++ {
		email := "target@example.com"
		if i%5 == 0 {
			email = "other@example.com"
		}
		commitNext("feature.txt", email)
	}

	require.NoError(tb, wt.Checkout(&git.CheckoutOptions{
		Branch: mainBranch,
	}))
	for i := 0; i < mainOnly; i++ {
		email := "target@example.com"
		if i%2 == 0 {
			email = "other@example.com"
		}
		commitNext("main.txt", email)
	}

	start := dayKeyFromTime(base.AddDate(0, 0, commitIndex-120), loc)
	end := dayKeyFromTime(base.AddDate(0, 0, commitIndex-1), loc)
	return repo, start, end
}

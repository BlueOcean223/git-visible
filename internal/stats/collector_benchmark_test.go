package stats

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

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

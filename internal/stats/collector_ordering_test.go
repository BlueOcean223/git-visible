package stats

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectRepo_OutOfOrderAuthorWhen_NoMissInRange(t *testing.T) {
	loc := time.UTC
	repo, wt := initMemoryGitRepo(t)
	email := "test@example.com"

	// 拓扑链（HEAD -> root）要求为：
	// 01-25 -> 01-20 -> 01-10 -> 01-15 -> 01-08
	// 因此按 root->HEAD 的顺序写入提交，且 Author.When 刻意乱序。
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

func sumDayKeyCounts(stats map[int]int) int {
	total := 0
	for _, c := range stats {
		total += c
	}
	return total
}

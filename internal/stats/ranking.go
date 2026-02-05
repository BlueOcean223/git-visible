package stats

import (
	"sort"
	"time"
)

// RepoRank 表示单个仓库在排行榜中的统计结果。
type RepoRank struct {
	Repository string  `json:"repository"`
	Commits    int     `json:"commits"`
	Percent    float64 `json:"percent"`
}

// RepoRanking 表示仓库排行榜结果。
type RepoRanking struct {
	Repositories []RepoRank `json:"repositories"`
	TotalCommits int        `json:"totalCommits"`
}

type percentRemainder struct {
	index     int
	remainder int
	commits   int
	repo      string
}

// RankRepositories 计算仓库提交排行榜。
// limit:
//   - limit <= 0: 返回全部仓库
//   - limit > 0: 返回 Top N
//
// 排序规则：按 commits 倒序；相同 commits 时按 repository 字符串升序。
// Percent 以 1 位小数输出，并保证所有输出行的百分比之和为 100.0（当 TotalCommits > 0）。
func RankRepositories(statsPerRepo map[string]map[time.Time]int, limit int) RepoRanking {
	rows := make([]RepoRank, 0, len(statsPerRepo))
	for repoPath, daily := range statsPerRepo {
		total := 0
		for _, c := range daily {
			if c <= 0 {
				continue
			}
			total += c
		}
		rows = append(rows, RepoRank{
			Repository: repoPath,
			Commits:    total,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Commits != rows[j].Commits {
			return rows[i].Commits > rows[j].Commits
		}
		return rows[i].Repository < rows[j].Repository
	})

	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}

	totalCommits := 0
	for _, r := range rows {
		totalCommits += r.Commits
	}

	if totalCommits <= 0 {
		return RepoRanking{
			Repositories: rows,
			TotalCommits: 0,
		}
	}

	// 使用 0.1% 作为最小单位，确保显示到 1 位小数时合计为 100.0%。
	// 100.0% == 1000 units。
	const totalUnits = 1000

	units := make([]int, len(rows))
	rems := make([]percentRemainder, 0, len(rows))

	sumUnits := 0
	for i, r := range rows {
		numerator := r.Commits * totalUnits
		u := numerator / totalCommits
		rem := numerator % totalCommits
		units[i] = u
		sumUnits += u
		rems = append(rems, percentRemainder{
			index:     i,
			remainder: rem,
			commits:   r.Commits,
			repo:      r.Repository,
		})
	}

	left := totalUnits - sumUnits
	if left > 0 {
		sort.Slice(rems, func(i, j int) bool {
			if rems[i].remainder != rems[j].remainder {
				return rems[i].remainder > rems[j].remainder
			}
			if rems[i].commits != rems[j].commits {
				return rems[i].commits > rems[j].commits
			}
			return rems[i].repo < rems[j].repo
		})
		for i := 0; i < left && i < len(rems); i++ {
			units[rems[i].index]++
		}
	}

	for i := range rows {
		rows[i].Percent = float64(units[i]) / 10.0
	}

	return RepoRanking{
		Repositories: rows,
		TotalCommits: totalCommits,
	}
}

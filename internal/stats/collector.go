package stats

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// maxConcurrency 是并发处理仓库的最大数量，默认为 CPU 核心数。
var maxConcurrency = runtime.NumCPU()

// CollectStats 并发收集多个仓库的提交统计。
// 参数:
//   - repos: 要统计的仓库路径列表
//   - emails: 邮箱过滤列表，为空时统计所有提交
//   - start: 起始日期（包含），按天统计的边界，建议传入当天 00:00:00
//   - end: 结束日期（包含），按天统计的边界，建议传入当天 00:00:00
//
// 返回以日期（当天 00:00:00）为键、提交数为值的映射。
// 如果部分仓库收集失败，会返回已成功收集的数据和聚合的错误。
func CollectStats(repos []string, emails []string, start, end time.Time) (map[time.Time]int, error) {
	if start.IsZero() {
		return nil, fmt.Errorf("start must be set")
	}
	if end.IsZero() {
		return nil, fmt.Errorf("end must be set")
	}

	loc := end.Location()
	start = beginningOfDay(start, loc)
	end = beginningOfDay(end, loc)

	if start.After(end) {
		return nil, fmt.Errorf("start must be <= end (start=%s, end=%s)", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	// 构建邮箱过滤集合
	emailSet := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		if email == "" {
			continue
		}
		emailSet[email] = struct{}{}
	}

	out := make(map[time.Time]int)

	// 并发控制和结果聚合
	var (
		wg   sync.WaitGroup // 等待所有 goroutine 完成
		mu   sync.Mutex     // 保护 out 的写入
		emu  sync.Mutex     // 保护 errs 的写入
		pmu  sync.Mutex     // 保护进度条更新
		errs []error        // 收集所有错误
	)

	bar := newRepoProgressBar(len(repos))
	if bar != nil {
		defer func() { _ = bar.Finish() }()
	}

	// 使用信号量限制并发数
	sem := make(chan struct{}, maxConcurrency)

	// 并发处理每个仓库
	for _, repoPath := range repos {
		wg.Add(1)
		go func(repoPath string) {
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量
			defer wg.Done()
			defer func() {
				if bar == nil {
					return
				}
				pmu.Lock()
				_ = bar.Add(1)
				pmu.Unlock()
			}()

			stats, err := collectRepo(repoPath, start, end, loc, emailSet)
			if err != nil {
				emu.Lock()
				errs = append(errs, err)
				emu.Unlock()
				return
			}

			// 合并结果
			mu.Lock()
			for day, count := range stats {
				out[day] += count
			}
			mu.Unlock()
		}(repoPath)
	}

	wg.Wait()

	return out, errors.Join(errs...)
}

// CollectStatsMonths 兼容旧接口：按最近 N 个月（对齐到周日）并截止到今天统计。
func CollectStatsMonths(repos []string, emails []string, months int) (map[time.Time]int, error) {
	start, end, err := TimeRange("", "", months)
	if err != nil {
		return nil, err
	}
	return CollectStats(repos, emails, start, end)
}

// newRepoProgressBar 创建仓库处理进度条。
// 仅当仓库数量 > 1 且在终端环境下才显示。
func newRepoProgressBar(total int) *progressbar.ProgressBar {
	if total <= 1 {
		return nil
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}

	return progressbar.NewOptions(
		total,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("collecting stats"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(65*time.Millisecond),
	)
}

// collectRepo 收集单个仓库在指定时间范围内的提交统计。
// 遍历从 HEAD 开始的提交历史，按作者邮箱过滤（如果指定），
// 按日期聚合提交数量。
func collectRepo(repoPath string, start, end time.Time, loc *time.Location, emailSet map[string]struct{}) (map[time.Time]int, error) {
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("stat repo %s: %w", repoPath, err)
	}

	// 打开 Git 仓库
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repo %s: %w", repoPath, err)
	}

	// 获取 HEAD 引用
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("head repo %s: %w", repoPath, err)
	}

	// 获取提交迭代器
	iterator, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("log repo %s: %w", repoPath, err)
	}
	defer iterator.Close()

	// 遍历提交，按日期聚合提交数
	out := make(map[time.Time]int)
	err = iterator.ForEach(func(c *object.Commit) error {
		// 邮箱过滤：如果指定了邮箱列表，只统计匹配的提交
		if len(emailSet) > 0 {
			if _, ok := emailSet[c.Author.Email]; !ok {
				return nil
			}
		}

		commitDay := beginningOfDay(c.Author.When, loc)
		// 跳过未来的提交（理论上不应该发生）
		if commitDay.After(end) {
			return nil
		}
		// 提交时间早于统计范围，可以停止遍历（提交按时间倒序排列）
		if commitDay.Before(start) {
			return storer.ErrStop
		}

		out[commitDay]++
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return nil, fmt.Errorf("iterate repo %s: %w", repoPath, err)
	}

	return out, nil
}

// beginningOfDay 返回给定时间当天 00:00:00 的时间点。
// 用于将提交时间归一化到日期维度进行聚合。
func beginningOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// heatmapStart 计算热力图的起始日期。
// 从当前日期向前推 months 个月，然后调整到最近的周日（热力图从周日开始）。
func heatmapStart(now time.Time, months int) time.Time {
	loc := now.Location()
	start := beginningOfDay(now.AddDate(0, -months, 0), loc)
	// 调整到周日（热力图的行从周日开始）
	for start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -1)
	}
	return start
}

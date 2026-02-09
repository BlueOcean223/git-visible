package stats

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"git-visible/internal/cache"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// maxConcurrency 是并发处理仓库的最大数量，默认为 CPU 核心数。
var maxConcurrency = runtime.NumCPU()
var collectRepoFn = collectRepo
var collectRepoByEmailsFn = collectRepoByEmails
var collectRepoFromRepositoryFn = collectRepoFromRepository
var collectRepoByEmailsFromRepositoryFn = collectRepoByEmailsFromRepository

// BranchOption controls which branch(es) to collect commits from.
// Default behavior (zero value) is to collect from HEAD only.
type BranchOption struct {
	// Branch specifies a single local branch name (e.g. "main").
	Branch string
	// AllBranches collects commits from all local branches (refs/heads/*),
	// de-duplicated by commit hash.
	AllBranches bool
}

type CollectOptions struct {
	Repos          []string
	Emails         []string
	Since          time.Time
	Until          time.Time
	Branch         string
	AllBranch      bool
	UseCache       bool
	NormalizeEmail func(string) string
}

// CollectStats 并发收集多个仓库的提交统计。
// 参数:
//   - repos: 要统计的仓库路径列表
//   - emails: 邮箱过滤列表，为空时统计所有提交
//   - start: 起始日期（包含），按天统计的边界，建议传入当天 00:00:00
//   - end: 结束日期（包含），按天统计的边界，建议传入当天 00:00:00
//   - useCache: 是否启用缓存
//
// 返回以日期（当天 00:00:00）为键、提交数为值的映射。
// 如果部分仓库收集失败，会返回已成功收集的数据和聚合的错误。
func CollectStats(repos []string, emails []string, start, end time.Time, branch BranchOption, normalizeEmail func(string) string, useCache bool) (map[time.Time]int, error) {
	loc := end.Location()
	out := make(map[time.Time]int)
	done, err := collectCommonGeneric[map[int]int](CollectOptions{
		Repos:          repos,
		Emails:         emails,
		Since:          start,
		Until:          end,
		Branch:         branch.Branch,
		AllBranch:      branch.AllBranches,
		UseCache:       useCache,
		NormalizeEmail: normalizeEmail,
	}, collectRepoFn, func(_ string, daily map[int]int) {
		for dayKey, count := range daily {
			out[dayKeyToTime(dayKey, loc)] += count
		}
	})
	if err != nil && len(done) == 0 {
		return nil, err
	}
	return out, err
}

// CollectStatsPerRepo 并发收集多个仓库的提交统计，并按仓库分别返回结果。
// 返回 map[repoPath]map[day]count，其中 day 为当天 00:00:00（由 end 的时区决定）。
// 如果部分仓库收集失败，会返回已成功收集的数据和聚合的错误。
func CollectStatsPerRepo(repos []string, emails []string, start, end time.Time, branch BranchOption, normalizeEmail func(string) string, useCache bool) (map[string]map[time.Time]int, error) {
	loc := end.Location()
	out := make(map[string]map[time.Time]int)
	done, err := collectCommonGeneric[map[int]int](CollectOptions{
		Repos:          repos,
		Emails:         emails,
		Since:          start,
		Until:          end,
		Branch:         branch.Branch,
		AllBranch:      branch.AllBranches,
		UseCache:       useCache,
		NormalizeEmail: normalizeEmail,
	}, collectRepoFn, func(repoPath string, daily map[int]int) {
		stats := make(map[time.Time]int, len(daily))
		for dayKey, count := range daily {
			stats[dayKeyToTime(dayKey, loc)] = count
		}
		out[repoPath] = stats
	})
	if err != nil && len(done) == 0 {
		return nil, err
	}
	return out, err
}

// CollectStatsByEmails 并发收集多个仓库的提交统计，并按邮箱分桶聚合。
// 返回 map[email]map[day]count，其中 day 为当天 00:00:00（由 end 的时区决定）。
// 如果部分仓库收集失败，会返回已成功收集的数据和聚合的错误。
func CollectStatsByEmails(repos []string, emails []string, start, end time.Time, branch BranchOption, normalizeEmail func(string) string, useCache bool) (map[string]map[time.Time]int, error) {
	loc := end.Location()
	out := make(map[string]map[int]int, len(emails))
	done, err := collectCommonGeneric[map[string]map[int]int](CollectOptions{
		Repos:          repos,
		Emails:         emails,
		Since:          start,
		Until:          end,
		Branch:         branch.Branch,
		AllBranch:      branch.AllBranches,
		UseCache:       useCache,
		NormalizeEmail: normalizeEmail,
	}, collectRepoByEmailsFn, func(_ string, byEmail map[string]map[int]int) {
		for email, daily := range byEmail {
			target := out[email]
			if target == nil {
				target = make(map[int]int, len(daily))
				out[email] = target
			}
			for dayKey, count := range daily {
				target[dayKey] += count
			}
		}
	})
	if err != nil && len(done) == 0 {
		return nil, err
	}

	converted := make(map[string]map[time.Time]int, len(out))
	for email, daily := range out {
		dayStats := make(map[time.Time]int, len(daily))
		for dayKey, count := range daily {
			dayStats[dayKeyToTime(dayKey, loc)] = count
		}
		converted[email] = dayStats
	}
	return converted, err
}

func collectCommonGeneric[T any](
	opts CollectOptions,
	collectFn func(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, useCache bool) (T, error),
	aggregator func(repoPath string, result T),
) ([]string, error) {
	if opts.Since.IsZero() {
		return nil, fmt.Errorf("start must be set")
	}
	if opts.Until.IsZero() {
		return nil, fmt.Errorf("end must be set")
	}

	branch, err := normalizeBranchOption(BranchOption{
		Branch:      opts.Branch,
		AllBranches: opts.AllBranch,
	})
	if err != nil {
		return nil, err
	}

	loc := opts.Until.Location()
	start := beginningOfDay(opts.Since, loc)
	end := beginningOfDay(opts.Until, loc)

	if start.After(end) {
		return nil, fmt.Errorf("start must be <= end (start=%s, end=%s)", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	startDayKey := dayKeyFromTime(start, loc)
	endDayKey := dayKeyFromTime(end, loc)

	normalizeEmail := resolveNormalizeEmail(opts.NormalizeEmail)
	emailSet := make(map[string]struct{}, len(opts.Emails))
	for _, email := range opts.Emails {
		if email == "" {
			continue
		}
		email = normalizeEmail(email)
		if email == "" {
			continue
		}
		emailSet[email] = struct{}{}
	}

	done := make([]string, 0, len(opts.Repos))

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		emu  sync.Mutex
		pmu  sync.Mutex
		errs []error
	)

	bar := newRepoProgressBar(len(opts.Repos))
	if bar != nil {
		defer func() { _ = bar.Finish() }()
	}

	sem := make(chan struct{}, maxConcurrency)

	for _, repoPath := range opts.Repos {
		wg.Add(1)
		go func(repoPath string) {
			sem <- struct{}{}
			defer func() { <-sem }()
			defer wg.Done()
			defer func() {
				if bar == nil {
					return
				}
				pmu.Lock()
				_ = bar.Add(1)
				pmu.Unlock()
			}()

			stats, err := collectFn(repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail, opts.UseCache)
			if err != nil {
				emu.Lock()
				errs = append(errs, err)
				emu.Unlock()
				return
			}

			mu.Lock()
			aggregator(repoPath, stats)
			done = append(done, repoPath)
			mu.Unlock()
		}(repoPath)
	}

	wg.Wait()
	return done, errors.Join(errs...)
}

// CollectStatsMonths 兼容旧接口：按最近 N 个月（对齐到周日）并截止到今天统计。
func CollectStatsMonths(repos []string, emails []string, months int) (map[time.Time]int, error) {
	start, end, err := TimeRange("", "", months)
	if err != nil {
		return nil, err
	}
	return CollectStats(repos, emails, start, end, BranchOption{}, nil, true)
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
//
// 设计约束：
//   - 统计口径基于 Author.When，且 Author.When 不保证单调，禁止据此提前终止遍历。
//   - 禁止基于 Author.When 或 Committer.When 的 < start 重新引入 ErrStop。
//   - 性能保障依赖邮箱过滤前移、dayKey 轻量聚合、以及 --all-branches 下的 hash 剪枝。
func collectRepo(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, useCache bool) (map[int]int, error) {
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("stat repo %s: %w", repoPath, err)
	}

	// 打开 Git 仓库
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repo %s: %w", repoPath, err)
	}

	var cacheKey cache.CacheKey
	if useCache {
		headRef, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("head repo %s: %w", repoPath, err)
		}
		cacheKey = buildRepoCacheKey(repoPath, headRef.Hash().String(), startDayKey, endDayKey, emailSet, branch)

		entry, err := cache.LoadCache(cacheKey)
		if err == nil {
			daily, convErr := fromCachedStats(entry.Stats)
			if convErr == nil {
				return daily, nil
			}
		}
	}

	stats, err := collectRepoFromRepositoryFn(repo, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail)
	if err != nil {
		return nil, err
	}

	if useCache {
		_ = cache.SaveCache(cacheKey, toCachedStats(stats))
	}

	return stats, nil
}

func collectRepoByEmails(repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, useCache bool) (map[string]map[int]int, error) {
	// 按邮箱分桶的缓存收益较低且缓存体积更大，当前实现不启用缓存；保留参数仅为复用统一 collectFn 签名。
	_ = useCache
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("stat repo %s: %w", repoPath, err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repo %s: %w", repoPath, err)
	}

	return collectRepoByEmailsFromRepositoryFn(repo, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail)
}

func collectRepoFromRepository(repo *git.Repository, repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string) (map[int]int, error) {
	out := make(map[int]int)
	if err := walkRepoCommits(repo, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail, func(_ string, dayKey int) {
		out[dayKey]++
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func collectRepoByEmailsFromRepository(repo *git.Repository, repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string) (map[string]map[int]int, error) {
	out := make(map[string]map[int]int)
	if err := walkRepoCommits(repo, repoPath, startDayKey, endDayKey, loc, emailSet, branch, normalizeEmail, func(email string, dayKey int) {
		daily := out[email]
		if daily == nil {
			daily = make(map[int]int)
			out[email] = daily
		}
		daily[dayKey]++
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func walkRepoCommits(repo *git.Repository, repoPath string, startDayKey, endDayKey int, loc *time.Location, emailSet map[string]struct{}, branch BranchOption, normalizeEmail func(string) string, visitor func(email string, dayKey int)) error {
	startPoints, err := collectStartPoints(repo, repoPath, branch)
	if err != nil {
		return err
	}
	normalizeEmail = resolveNormalizeEmail(normalizeEmail)

	seenCommits := make(map[plumbing.Hash]struct{})

	for _, from := range startPoints {
		iterator, err := repo.Log(&git.LogOptions{From: from})
		if err != nil {
			return fmt.Errorf("log repo %s: %w", repoPath, err)
		}

		iterErr := iterator.ForEach(func(c *object.Commit) error {
			if branch.AllBranches {
				if _, seen := seenCommits[c.Hash]; seen {
					// 该提交及其祖先已在先前分支遍历中处理过，提前剪枝。
					return storer.ErrStop
				}
				seenCommits[c.Hash] = struct{}{}
			}

			email := normalizeEmail(c.Author.Email)
			// 邮箱过滤前移：无关邮箱直接跳过，避免后续时间归一化开销。
			if len(emailSet) > 0 {
				if _, ok := emailSet[email]; !ok {
					return nil
				}
			}

			commitDayKey := dayKeyFromTime(c.Author.When, loc)
			if commitDayKey > endDayKey {
				return nil
			}
			if commitDayKey < startDayKey {
				return nil
			}

			visitor(email, commitDayKey)
			return nil
		})
		iterator.Close()
		if iterErr != nil && !errors.Is(iterErr, storer.ErrStop) {
			return fmt.Errorf("iterate repo %s: %w", repoPath, iterErr)
		}
	}

	return nil
}

func buildRepoCacheKey(repoPath string, headHash string, startDayKey, endDayKey int, emailSet map[string]struct{}, branch BranchOption) cache.CacheKey {
	return cache.CacheKey{
		RepoPath:  repoPath,
		HEADHash:  headHash,
		Emails:    sortedEmails(emailSet),
		TimeRange: fmt.Sprintf("%s_%s", dayKeyToDateString(startDayKey), dayKeyToDateString(endDayKey)),
		Branch:    branch.Branch,
		AllBranch: branch.AllBranches,
	}
}

func sortedEmails(emailSet map[string]struct{}) []string {
	if len(emailSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(emailSet))
	for email := range emailSet {
		out = append(out, email)
	}
	sort.Strings(out)
	return out
}

func toCachedStats(daily map[int]int) map[string]int {
	out := make(map[string]int, len(daily))
	for dayKey, count := range daily {
		out[dayKeyToDateString(dayKey)] = count
	}
	return out
}

func fromCachedStats(stats map[string]int) (map[int]int, error) {
	out := make(map[int]int, len(stats))
	for day, count := range stats {
		dayKey, err := dateStringToDayKey(day)
		if err != nil {
			return nil, err
		}
		out[dayKey] = count
	}
	return out, nil
}

func dayKeyToDateString(dayKey int) string {
	return dayKeyToTime(dayKey, time.UTC).Format("2006-01-02")
}

func dateStringToDayKey(day string) (int, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(day))
	if err != nil {
		return 0, fmt.Errorf("parse cache day %q: %w", day, err)
	}
	return dayKeyFromTime(t, time.UTC), nil
}

func resolveNormalizeEmail(normalizeEmail func(string) string) func(string) string {
	if normalizeEmail == nil {
		return func(email string) string { return email }
	}
	return normalizeEmail
}

// normalizeBranchOption 验证并标准化分支选项。
func normalizeBranchOption(opt BranchOption) (BranchOption, error) {
	opt.Branch = strings.TrimSpace(opt.Branch)
	if opt.Branch != "" && opt.AllBranches {
		return BranchOption{}, fmt.Errorf("--branch and --all-branches are mutually exclusive")
	}
	return opt, nil
}

// collectStartPoints 根据分支选项确定遍历的起始 commit hash 列表。
func collectStartPoints(repo *git.Repository, repoPath string, branch BranchOption) ([]plumbing.Hash, error) {
	switch {
	case branch.AllBranches:
		iter, err := repo.Branches()
		if err != nil {
			return nil, fmt.Errorf("list branches repo %s: %w", repoPath, err)
		}
		defer iter.Close()

		tips := make([]plumbing.Hash, 0)
		seenTips := make(map[plumbing.Hash]struct{})
		err = iter.ForEach(func(ref *plumbing.Reference) error {
			if ref == nil {
				return nil
			}
			h := ref.Hash()
			if h.IsZero() {
				return nil
			}
			if _, ok := seenTips[h]; ok {
				return nil
			}
			seenTips[h] = struct{}{}
			tips = append(tips, h)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("iterate branches repo %s: %w", repoPath, err)
		}
		return tips, nil
	case branch.Branch != "":
		refName := plumbing.NewBranchReferenceName(branch.Branch)
		if strings.HasPrefix(branch.Branch, "refs/") {
			refName = plumbing.ReferenceName(branch.Branch)
		}
		ref, err := repo.Reference(refName, true)
		if err != nil {
			return nil, fmt.Errorf("repo %s: branch %q not found", repoPath, branch.Branch)
		}
		if ref.Hash().IsZero() {
			return nil, fmt.Errorf("repo %s: branch %q has no commits", repoPath, branch.Branch)
		}
		return []plumbing.Hash{ref.Hash()}, nil
	default:
		ref, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("head repo %s: %w", repoPath, err)
		}
		if ref.Hash().IsZero() {
			return nil, fmt.Errorf("repo %s: HEAD has no commits", repoPath)
		}
		return []plumbing.Hash{ref.Hash()}, nil
	}
}

// beginningOfDay 返回给定时间当天 00:00:00 的时间点。
// 用于将提交时间归一化到日期维度进行聚合。
func beginningOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// dayKeyFromTime 将时间转换为日粒度整数键（yyyymmdd）。
func dayKeyFromTime(t time.Time, loc *time.Location) int {
	t = t.In(loc)
	return t.Year()*10000 + int(t.Month())*100 + t.Day()
}

// dayKeyToTime 将日粒度整数键（yyyymmdd）转换为当天 00:00:00。
func dayKeyToTime(dayKey int, loc *time.Location) time.Time {
	year := dayKey / 10000
	month := (dayKey / 100) % 100
	day := dayKey % 100
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
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

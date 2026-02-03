package stats

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

func CollectStats(repos []string, emails []string, months int) (map[time.Time]int, error) {
	if months <= 0 {
		return nil, fmt.Errorf("months must be > 0, got %d", months)
	}

	now := time.Now()
	loc := now.Location()

	start := heatmapStart(now, months)
	end := beginningOfDay(now, loc)

	emailSet := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		if email == "" {
			continue
		}
		emailSet[email] = struct{}{}
	}

	out := make(map[time.Time]int)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		emu  sync.Mutex
		pmu  sync.Mutex
		errs []error
	)

	bar := newRepoProgressBar(len(repos))
	if bar != nil {
		defer func() { _ = bar.Finish() }()
	}

	for _, repoPath := range repos {
		repoPath := repoPath
		wg.Add(1)
		go func() {
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

			mu.Lock()
			for day, count := range stats {
				out[day] += count
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	return out, errors.Join(errs...)
}

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

func collectRepo(repoPath string, start, end time.Time, loc *time.Location, emailSet map[string]struct{}) (map[time.Time]int, error) {
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("stat repo %s: %w", repoPath, err)
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repo %s: %w", repoPath, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("head repo %s: %w", repoPath, err)
	}

	iterator, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("log repo %s: %w", repoPath, err)
	}
	defer iterator.Close()

	out := make(map[time.Time]int)
	err = iterator.ForEach(func(c *object.Commit) error {
		if len(emailSet) > 0 {
			if _, ok := emailSet[c.Author.Email]; !ok {
				return nil
			}
		}

		commitDay := beginningOfDay(c.Author.When, loc)
		if commitDay.After(end) {
			return nil
		}
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

func beginningOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func heatmapStart(now time.Time, months int) time.Time {
	loc := now.Location()
	start := beginningOfDay(now.AddDate(0, -months, 0), loc)
	for start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -1)
	}
	return start
}

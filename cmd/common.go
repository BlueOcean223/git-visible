package cmd

import (
	"errors"
	"strings"
	"time"

	"git-visible/internal/config"
	"git-visible/internal/repo"
	"git-visible/internal/stats"
)

var errNoRepositoriesAdded = errors.New("no repositories added")

// RunContext holds the common initialization result for commands.
type RunContext struct {
	Repos  []string
	Emails []string
	Since  time.Time
	Until  time.Time

	months int
}

// prepareRun performs common command initialization:
// load config, load repos, parse time range, merge emails.
func prepareRun(emails []string, months int, since, until string) (*RunContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	repos, err := repo.LoadRepos()
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, errNoRepositoriesAdded
	}

	since = strings.TrimSpace(since)
	until = strings.TrimSpace(until)

	resolvedMonths := months
	if resolvedMonths == 0 {
		resolvedMonths = cfg.Months
	}
	rangeMonths := resolvedMonths
	if since != "" || until != "" {
		rangeMonths = cfg.Months
	}

	start, end, err := stats.TimeRange(since, until, rangeMonths)
	if err != nil {
		return nil, err
	}

	mergedEmails := emails
	if len(mergedEmails) == 0 && strings.TrimSpace(cfg.Email) != "" {
		mergedEmails = []string{strings.TrimSpace(cfg.Email)}
	}

	return &RunContext{
		Repos:  repos,
		Emails: mergedEmails,
		Since:  start,
		Until:  end,
		months: resolvedMonths,
	}, nil
}

package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setFixedNow(t *testing.T, now time.Time) {
	t.Helper()
	old := timeNow
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = old })
}

func TestCalculateSummary_Empty(t *testing.T) {
	setFixedNow(t, time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC))

	got := CalculateSummary(map[time.Time]int{})

	assert.Equal(t, 0, got.TotalCommits)
	assert.Equal(t, 0, got.ActiveDays)
	assert.Equal(t, 0, got.CurrentStreak)
	assert.Equal(t, 0, got.LongestStreak.Days)
	assert.True(t, got.LongestStreak.Start.IsZero())
	assert.True(t, got.LongestStreak.End.IsZero())
	assert.Equal(t, 0, got.MostActiveWeekday.Commits)
	assert.Equal(t, 0, got.PeakDay.Commits)
	assert.True(t, got.PeakDay.Date.IsZero())
}

func TestCalculateSummary_SingleDay(t *testing.T) {
	now := time.Date(2024, 6, 12, 15, 0, 0, 0, time.UTC) // Wed
	setFixedNow(t, now)
	loc := now.Location()
	today := beginningOfDay(now, loc)

	st := map[time.Time]int{today: 3}
	got := CalculateSummary(st)

	assert.Equal(t, 3, got.TotalCommits)
	assert.Equal(t, 1, got.ActiveDays)
	assert.Equal(t, 1, got.CurrentStreak)
	assert.Equal(t, 1, got.LongestStreak.Days)
	assert.True(t, got.LongestStreak.Start.Equal(today))
	assert.True(t, got.LongestStreak.End.Equal(today))
	assert.Equal(t, time.Wednesday, got.MostActiveWeekday.Weekday)
	assert.Equal(t, 3, got.MostActiveWeekday.Commits)
	assert.True(t, got.PeakDay.Date.Equal(today))
	assert.Equal(t, 3, got.PeakDay.Commits)
}

func TestCalculateSummary_Consecutive7Days(t *testing.T) {
	now := time.Date(2024, 6, 7, 10, 0, 0, 0, time.UTC) // Fri
	setFixedNow(t, now)
	loc := now.Location()

	st := make(map[time.Time]int)
	for i := 0; i < 7; i++ {
		day := beginningOfDay(now.AddDate(0, 0, -i), loc)
		st[day] = 1
	}
	wed := beginningOfDay(time.Date(2024, 6, 5, 12, 0, 0, 0, loc), loc)
	st[wed] = 5

	got := CalculateSummary(st)

	assert.Equal(t, 11, got.TotalCommits)
	assert.Equal(t, 7, got.ActiveDays)
	assert.Equal(t, 7, got.CurrentStreak)
	assert.Equal(t, 7, got.LongestStreak.Days)

	start := beginningOfDay(now.AddDate(0, 0, -6), loc)
	end := beginningOfDay(now, loc)
	assert.True(t, got.LongestStreak.Start.Equal(start))
	assert.True(t, got.LongestStreak.End.Equal(end))

	assert.Equal(t, time.Wednesday, got.MostActiveWeekday.Weekday)
	assert.Equal(t, 5, got.MostActiveWeekday.Commits)
	assert.True(t, got.PeakDay.Date.Equal(wed))
	assert.Equal(t, 5, got.PeakDay.Commits)
}

func TestCalculateSummary_WithGap(t *testing.T) {
	now := time.Date(2024, 6, 10, 10, 0, 0, 0, time.UTC) // Mon
	setFixedNow(t, now)
	loc := now.Location()

	today := beginningOfDay(now, loc)
	yesterday := beginningOfDay(now.AddDate(0, 0, -1), loc)
	wed := beginningOfDay(now.AddDate(0, 0, -5), loc)
	thu := beginningOfDay(now.AddDate(0, 0, -4), loc)
	fri := beginningOfDay(now.AddDate(0, 0, -3), loc)

	st := map[time.Time]int{
		today:     1,
		yesterday: 1,
		wed:       2,
		thu:       2,
		fri:       2,
	}

	got := CalculateSummary(st)

	assert.Equal(t, 8, got.TotalCommits)
	assert.Equal(t, 5, got.ActiveDays)
	assert.Equal(t, 2, got.CurrentStreak)
	assert.Equal(t, 3, got.LongestStreak.Days)
	assert.True(t, got.LongestStreak.Start.Equal(wed))
	assert.True(t, got.LongestStreak.End.Equal(fri))
}

func TestCalculateSummary_TodayNoCommits_CurrentStreakZero(t *testing.T) {
	now := time.Date(2024, 6, 10, 10, 0, 0, 0, time.UTC) // Mon
	setFixedNow(t, now)
	loc := now.Location()

	yesterday := beginningOfDay(now.AddDate(0, 0, -1), loc)
	twoDaysAgo := beginningOfDay(now.AddDate(0, 0, -2), loc)

	st := map[time.Time]int{
		yesterday:  1,
		twoDaysAgo: 1,
	}

	got := CalculateSummary(st)

	assert.Equal(t, 0, got.CurrentStreak)
	assert.Equal(t, 2, got.LongestStreak.Days)
	assert.True(t, got.LongestStreak.Start.Equal(twoDaysAgo))
	assert.True(t, got.LongestStreak.End.Equal(yesterday))
}

func TestRenderSummary_Format(t *testing.T) {
	s := Summary{
		TotalCommits:  247,
		ActiveDays:    89,
		CurrentStreak: 5,
		LongestStreak: Streak{
			Days:  14,
			Start: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC),
		},
		MostActiveWeekday: WeekdayStat{Weekday: time.Wednesday, Commits: 67},
		PeakDay:           DayStat{Date: time.Date(2024, 12, 12, 0, 0, 0, 0, time.UTC), Commits: 18},
	}

	want := "────────────────────────────────────\n" +
		"Total: 247 commits │ Active days: 89 │ Current streak: 5 days\n" +
		"Longest streak: 14 days (Dec 01 - Dec 14)\n" +
		"Most active: Wed (67 commits) │ Peak day: Dec 12 (18 commits)\n"

	assert.Equal(t, want, RenderSummary(s))
}

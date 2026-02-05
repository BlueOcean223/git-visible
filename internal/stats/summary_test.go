package stats

import (
	"testing"
	"time"
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
	if got.TotalCommits != 0 {
		t.Fatalf("TotalCommits = %d, want 0", got.TotalCommits)
	}
	if got.ActiveDays != 0 {
		t.Fatalf("ActiveDays = %d, want 0", got.ActiveDays)
	}
	if got.CurrentStreak != 0 {
		t.Fatalf("CurrentStreak = %d, want 0", got.CurrentStreak)
	}
	if got.LongestStreak.Days != 0 {
		t.Fatalf("LongestStreak.Days = %d, want 0", got.LongestStreak.Days)
	}
	if !got.LongestStreak.Start.IsZero() || !got.LongestStreak.End.IsZero() {
		t.Fatalf("LongestStreak range = (%v-%v), want zero values", got.LongestStreak.Start, got.LongestStreak.End)
	}
	if got.MostActiveWeekday.Commits != 0 {
		t.Fatalf("MostActiveWeekday.Commits = %d, want 0", got.MostActiveWeekday.Commits)
	}
	if got.PeakDay.Commits != 0 {
		t.Fatalf("PeakDay.Commits = %d, want 0", got.PeakDay.Commits)
	}
	if !got.PeakDay.Date.IsZero() {
		t.Fatalf("PeakDay.Date = %v, want zero value", got.PeakDay.Date)
	}
}

func TestCalculateSummary_SingleDay(t *testing.T) {
	now := time.Date(2024, 6, 12, 15, 0, 0, 0, time.UTC) // Wed
	setFixedNow(t, now)
	loc := now.Location()
	today := beginningOfDay(now, loc)

	st := map[time.Time]int{today: 3}
	got := CalculateSummary(st)

	if got.TotalCommits != 3 {
		t.Fatalf("TotalCommits = %d, want 3", got.TotalCommits)
	}
	if got.ActiveDays != 1 {
		t.Fatalf("ActiveDays = %d, want 1", got.ActiveDays)
	}
	if got.CurrentStreak != 1 {
		t.Fatalf("CurrentStreak = %d, want 1", got.CurrentStreak)
	}
	if got.LongestStreak.Days != 1 {
		t.Fatalf("LongestStreak.Days = %d, want 1", got.LongestStreak.Days)
	}
	if !got.LongestStreak.Start.Equal(today) || !got.LongestStreak.End.Equal(today) {
		t.Fatalf("LongestStreak range = (%v-%v), want (%v-%v)", got.LongestStreak.Start, got.LongestStreak.End, today, today)
	}
	if got.MostActiveWeekday.Weekday != time.Wednesday || got.MostActiveWeekday.Commits != 3 {
		t.Fatalf("MostActiveWeekday = (%v,%d), want (Wednesday,3)", got.MostActiveWeekday.Weekday, got.MostActiveWeekday.Commits)
	}
	if !got.PeakDay.Date.Equal(today) || got.PeakDay.Commits != 3 {
		t.Fatalf("PeakDay = (%v,%d), want (%v,3)", got.PeakDay.Date, got.PeakDay.Commits, today)
	}
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
	// Make one weekday clearly the most active & peak day.
	wed := beginningOfDay(time.Date(2024, 6, 5, 12, 0, 0, 0, loc), loc) // Wed in range
	st[wed] = 5

	got := CalculateSummary(st)
	if got.TotalCommits != 11 {
		t.Fatalf("TotalCommits = %d, want 11", got.TotalCommits)
	}
	if got.ActiveDays != 7 {
		t.Fatalf("ActiveDays = %d, want 7", got.ActiveDays)
	}
	if got.CurrentStreak != 7 {
		t.Fatalf("CurrentStreak = %d, want 7", got.CurrentStreak)
	}
	if got.LongestStreak.Days != 7 {
		t.Fatalf("LongestStreak.Days = %d, want 7", got.LongestStreak.Days)
	}
	start := beginningOfDay(now.AddDate(0, 0, -6), loc)
	end := beginningOfDay(now, loc)
	if !got.LongestStreak.Start.Equal(start) || !got.LongestStreak.End.Equal(end) {
		t.Fatalf("LongestStreak range = (%v-%v), want (%v-%v)", got.LongestStreak.Start, got.LongestStreak.End, start, end)
	}
	if got.MostActiveWeekday.Weekday != time.Wednesday || got.MostActiveWeekday.Commits != 5 {
		t.Fatalf("MostActiveWeekday = (%v,%d), want (Wednesday,5)", got.MostActiveWeekday.Weekday, got.MostActiveWeekday.Commits)
	}
	if !got.PeakDay.Date.Equal(wed) || got.PeakDay.Commits != 5 {
		t.Fatalf("PeakDay = (%v,%d), want (%v,5)", got.PeakDay.Date, got.PeakDay.Commits, wed)
	}
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
	if got.TotalCommits != 8 {
		t.Fatalf("TotalCommits = %d, want 8", got.TotalCommits)
	}
	if got.ActiveDays != 5 {
		t.Fatalf("ActiveDays = %d, want 5", got.ActiveDays)
	}
	if got.CurrentStreak != 2 {
		t.Fatalf("CurrentStreak = %d, want 2", got.CurrentStreak)
	}
	if got.LongestStreak.Days != 3 {
		t.Fatalf("LongestStreak.Days = %d, want 3", got.LongestStreak.Days)
	}
	if !got.LongestStreak.Start.Equal(wed) || !got.LongestStreak.End.Equal(fri) {
		t.Fatalf("LongestStreak range = (%v-%v), want (%v-%v)", got.LongestStreak.Start, got.LongestStreak.End, wed, fri)
	}
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
	if got.CurrentStreak != 0 {
		t.Fatalf("CurrentStreak = %d, want 0", got.CurrentStreak)
	}
	if got.LongestStreak.Days != 2 {
		t.Fatalf("LongestStreak.Days = %d, want 2", got.LongestStreak.Days)
	}
	if !got.LongestStreak.Start.Equal(twoDaysAgo) || !got.LongestStreak.End.Equal(yesterday) {
		t.Fatalf("LongestStreak range = (%v-%v), want (%v-%v)", got.LongestStreak.Start, got.LongestStreak.End, twoDaysAgo, yesterday)
	}
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

	if got := RenderSummary(s); got != want {
		t.Fatalf("RenderSummary() =\n%q\nwant:\n%q", got, want)
	}
}

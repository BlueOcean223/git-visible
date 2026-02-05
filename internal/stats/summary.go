package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// timeNow 抽象 time.Now 以便在测试中注入固定时间。
var timeNow = time.Now

// Streak 表示连续提交天数的区间信息。
type Streak struct {
	Days  int
	Start time.Time
	End   time.Time
}

// WeekdayStat 表示按星期几聚合的提交统计。
type WeekdayStat struct {
	Weekday time.Weekday
	Commits int
}

// DayStat 表示单日最高提交统计。
type DayStat struct {
	Date    time.Time
	Commits int
}

// Summary 表示热力图的统计摘要信息。
// 结构体包含 6 项指标：
//   - TotalCommits
//   - ActiveDays
//   - CurrentStreak
//   - LongestStreak
//   - MostActiveWeekday
//   - PeakDay
type Summary struct {
	TotalCommits      int
	ActiveDays        int
	CurrentStreak     int
	LongestStreak     Streak
	MostActiveWeekday WeekdayStat
	PeakDay           DayStat
}

// CalculateSummary 基于按天聚合的提交统计计算摘要信息。
func CalculateSummary(stats map[time.Time]int) Summary {
	var out Summary
	if len(stats) == 0 {
		return out
	}

	now := timeNow()
	loc := now.Location()
	today := beginningOfDay(now, loc)

	var weekdayTotals [7]int
	days := make([]time.Time, 0, len(stats))

	for day, count := range stats {
		if count <= 0 {
			continue
		}

		out.TotalCommits += count
		out.ActiveDays++

		weekdayTotals[day.Weekday()] += count

		if count > out.PeakDay.Commits || (count == out.PeakDay.Commits && day.After(out.PeakDay.Date)) {
			out.PeakDay = DayStat{Date: day, Commits: count}
		}

		days = append(days, day)
	}

	if len(days) == 0 {
		return out
	}

	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	// Current streak: from today backwards, consecutive days with commits.
	if stats[today] > 0 {
		for d := today; stats[d] > 0; d = d.AddDate(0, 0, -1) {
			out.CurrentStreak++
		}
	}

	// Longest streak over all commit days.
	curStart := days[0]
	curLen := 1

	longestStart := curStart
	longestEnd := curStart
	longestLen := 1

	for i := 1; i < len(days); i++ {
		prev := days[i-1]
		day := days[i]
		if prev.AddDate(0, 0, 1).Equal(day) {
			curLen++
			continue
		}

		if curLen > longestLen || (curLen == longestLen && prev.After(longestEnd)) {
			longestLen = curLen
			longestStart = curStart
			longestEnd = prev
		}

		curStart = day
		curLen = 1
	}

	last := days[len(days)-1]
	if curLen > longestLen || (curLen == longestLen && last.After(longestEnd)) {
		longestLen = curLen
		longestStart = curStart
		longestEnd = last
	}
	out.LongestStreak = Streak{Days: longestLen, Start: longestStart, End: longestEnd}

	// Most active weekday by total commits.
	mostWeekday := time.Sunday
	mostCommits := weekdayTotals[mostWeekday]
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		if weekdayTotals[wd] <= mostCommits {
			continue
		}
		mostCommits = weekdayTotals[wd]
		mostWeekday = wd
	}
	out.MostActiveWeekday = WeekdayStat{Weekday: mostWeekday, Commits: mostCommits}

	return out
}

const summaryRuleLen = 36

// RenderSummary 渲染摘要信息，输出为多行纯文本，可直接输出到终端。
func RenderSummary(s Summary) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("─", summaryRuleLen))
	b.WriteByte('\n')

	b.WriteString(fmt.Sprintf(
		"Total: %d commits │ Active days: %d │ Current streak: %d %s\n",
		s.TotalCommits,
		s.ActiveDays,
		s.CurrentStreak,
		pluralize(s.CurrentStreak, "day", "days"),
	))

	if s.LongestStreak.Days == 0 || s.LongestStreak.Start.IsZero() || s.LongestStreak.End.IsZero() {
		b.WriteString(fmt.Sprintf("Longest streak: %d %s\n", s.LongestStreak.Days, pluralize(s.LongestStreak.Days, "day", "days")))
	} else {
		b.WriteString(fmt.Sprintf(
			"Longest streak: %d %s (%s - %s)\n",
			s.LongestStreak.Days,
			pluralize(s.LongestStreak.Days, "day", "days"),
			formatShortDate(s.LongestStreak.Start),
			formatShortDate(s.LongestStreak.End),
		))
	}

	mostLabel := "-"
	if s.MostActiveWeekday.Commits > 0 {
		mostLabel = weekdayAbbrev(s.MostActiveWeekday.Weekday)
	}

	peakLabel := "-"
	if s.PeakDay.Commits > 0 && !s.PeakDay.Date.IsZero() {
		peakLabel = formatShortDate(s.PeakDay.Date)
	}

	b.WriteString(fmt.Sprintf(
		"Most active: %s (%d commits) │ Peak day: %s (%d commits)\n",
		mostLabel,
		s.MostActiveWeekday.Commits,
		peakLabel,
		s.PeakDay.Commits,
	))

	return b.String()
}

func weekdayAbbrev(wd time.Weekday) string {
	name := wd.String()
	if len(name) > 3 {
		return name[:3]
	}
	return name
}

func formatShortDate(t time.Time) string {
	return t.Format("Jan 02")
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

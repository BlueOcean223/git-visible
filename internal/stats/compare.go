package stats

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CompareMetrics 表示 compare 命令需要的核心统计指标。
type CompareMetrics struct {
	TotalCommits             int
	ActiveDays               int
	AvgCommitsPerDay         float64
	MostActiveWeekday        time.Weekday
	MostActiveWeekdayCommits int
	LongestStreakDays        int
}

// CalculateCompareMetrics 基于按天聚合的提交统计计算 compare 指标。
func CalculateCompareMetrics(daily map[time.Time]int) CompareMetrics {
	s := CalculateSummary(daily)

	avg := 0.0
	if s.ActiveDays > 0 {
		avg = float64(s.TotalCommits) / float64(s.ActiveDays)
	}

	return CompareMetrics{
		TotalCommits:             s.TotalCommits,
		ActiveDays:               s.ActiveDays,
		AvgCommitsPerDay:         avg,
		MostActiveWeekday:        s.MostActiveWeekday.Weekday,
		MostActiveWeekdayCommits: s.MostActiveWeekday.Commits,
		LongestStreakDays:        s.LongestStreak.Days,
	}
}

// PercentChange 表示一个变化百分比结果。
// Defined=false 表示无法计算（例如 from=0 且 to!=0）。
type PercentChange struct {
	Percent float64
	Defined bool
}

// CalculatePercentChange 计算从 from 到 to 的变化百分比：(to-from)/from*100。
func CalculatePercentChange(from, to float64) PercentChange {
	if from == 0 {
		if to == 0 {
			return PercentChange{Percent: 0, Defined: true}
		}
		return PercentChange{Defined: false}
	}
	return PercentChange{Percent: (to - from) / from * 100, Defined: true}
}

// CalculatePercentChanges 计算相邻元素之间的变化百分比，长度为 len(values)-1。
func CalculatePercentChanges(values []float64) []PercentChange {
	if len(values) < 2 {
		return nil
	}
	out := make([]PercentChange, 0, len(values)-1)
	for i := 1; i < len(values); i++ {
		out = append(out, CalculatePercentChange(values[i-1], values[i]))
	}
	return out
}

// Period 表示一个可对比的时间段（包含起止日期，均为包含边界）。
type Period struct {
	Label string
	Start time.Time
	End   time.Time
}

// ParsePeriod 解析 compare 的时间段参数，支持：
//   - YYYY: 整年
//   - YYYY-HN: 半年 (H1=1-6月, H2=7-12月)
//   - YYYY-QN: 季度 (Q1-Q4)
//   - YYYY-MM: 单月
func ParsePeriod(s string) (Period, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Period{}, fmt.Errorf("period is empty")
	}

	loc := timeNow().Location()

	// YYYY
	if len(s) == 4 && isDigits(s) {
		year, _ := strconv.Atoi(s)
		start := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
		end := time.Date(year+1, time.January, 0, 0, 0, 0, 0, loc)
		return Period{Label: s, Start: start, End: end}, nil
	}

	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return Period{}, fmt.Errorf("invalid period %q (expected YYYY, YYYY-HN, YYYY-QN, or YYYY-MM)", s)
	}

	yearStr := parts[0]
	rest := parts[1]

	if len(yearStr) != 4 || !isDigits(yearStr) {
		return Period{}, fmt.Errorf("invalid period %q: invalid year", s)
	}
	year, _ := strconv.Atoi(yearStr)

	// YYYY-MM
	if len(rest) == 2 && isDigits(rest) {
		month, _ := strconv.Atoi(rest)
		if month < 1 || month > 12 {
			return Period{}, fmt.Errorf("invalid period %q: month must be 01-12", s)
		}
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
		end := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, loc)
		return Period{Label: s, Start: start, End: end}, nil
	}

	// YYYY-HN / YYYY-QN
	if len(rest) == 2 {
		prefix := rest[0]
		num := rest[1]
		if num < '0' || num > '9' {
			return Period{}, fmt.Errorf("invalid period %q (expected YYYY-HN or YYYY-QN)", s)
		}
		n := int(num - '0')

		switch prefix {
		case 'H', 'h':
			if n < 1 || n > 2 {
				return Period{}, fmt.Errorf("invalid period %q: half must be H1 or H2", s)
			}
			startMonth := 1
			if n == 2 {
				startMonth = 7
			}
			endMonth := startMonth + 5
			start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, loc)
			end := time.Date(year, time.Month(endMonth+1), 0, 0, 0, 0, 0, loc)
			return Period{Label: s, Start: start, End: end}, nil
		case 'Q', 'q':
			if n < 1 || n > 4 {
				return Period{}, fmt.Errorf("invalid period %q: quarter must be Q1-Q4", s)
			}
			startMonth := 1 + 3*(n-1)
			endMonth := startMonth + 2
			start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, loc)
			end := time.Date(year, time.Month(endMonth+1), 0, 0, 0, 0, 0, loc)
			return Period{Label: s, Start: start, End: end}, nil
		}
	}

	return Period{}, fmt.Errorf("invalid period %q (expected YYYY, YYYY-HN, YYYY-QN, or YYYY-MM)", s)
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

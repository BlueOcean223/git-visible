package stats

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDate 解析日期字符串，支持：
//   - ISO 日期: "2025-01-15"
//   - 年月: "2025-01" → 2025-01-01
//   - 相对日期: "1w"/"2m"/"1y" → 1周前/2月前/1年前
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("date is empty")
	}

	now := timeNow()
	loc := now.Location()

	// ISO date: YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return beginningOfDay(t, loc), nil
	}

	// Year-month: YYYY-MM -> first day of the month.
	if t, err := time.ParseInLocation("2006-01", s, loc); err == nil {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc), nil
	}

	// Relative: <n><unit>, where unit is w/m/y.
	if len(s) >= 2 {
		unit := s[len(s)-1]
		nStr := strings.TrimSpace(s[:len(s)-1])
		n, err := strconv.Atoi(nStr)
		if err == nil {
			if n <= 0 {
				return time.Time{}, fmt.Errorf("relative date must be > 0, got %q", s)
			}

			base := beginningOfDay(now, loc)
			switch unit {
			case 'w', 'W':
				return base.AddDate(0, 0, -7*n), nil
			case 'm', 'M':
				return base.AddDate(0, -n, 0), nil
			case 'y', 'Y':
				return base.AddDate(-n, 0, 0), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD, YYYY-MM, or relative like 2m/1w/1y)", s)
}

// TimeRange 计算时间范围。
// 优先级: since/until > months
func TimeRange(since, until string, months int) (start, end time.Time, err error) {
	since = strings.TrimSpace(since)
	until = strings.TrimSpace(until)

	now := timeNow()
	loc := now.Location()

	// Default: use months ending today.
	if since == "" && until == "" {
		if months <= 0 {
			return time.Time{}, time.Time{}, fmt.Errorf("months must be > 0, got %d", months)
		}
		end = beginningOfDay(now, loc)
		start = heatmapStart(now, months)
		return start, end, nil
	}

	if since != "" {
		t, parseErr := ParseDate(since)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse --since: %w", parseErr)
		}
		start = beginningOfDay(t, loc)
	}

	if until != "" {
		t, parseErr := ParseDate(until)
		if parseErr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse --until: %w", parseErr)
		}
		end = beginningOfDay(t, loc)
	}

	switch {
	case since != "" && until == "":
		end = beginningOfDay(now, loc)
	case since == "" && until != "":
		if months <= 0 {
			return time.Time{}, time.Time{}, fmt.Errorf("months must be > 0, got %d", months)
		}
		start = heatmapStart(end, months)
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"since must be <= until (since=%s, until=%s)",
			start.Format("2006-01-02"),
			end.Format("2006-01-02"),
		)
	}

	return start, end, nil
}

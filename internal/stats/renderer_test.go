package stats

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// --- RenderLegend ---

func TestRenderLegend_IncludesColorCodes(t *testing.T) {
	legend := RenderLegend()

	wantParts := []string{
		colorEmpty + "░░" + colorReset,
		colorLow + "██" + colorReset,
		colorMedium + "██" + colorReset,
		colorHigh + "██" + colorReset,
	}
	for _, part := range wantParts {
		assert.Contains(t, legend, part, "should contain color code %q", part)
	}
}

func TestRenderLegend_Format(t *testing.T) {
	legend := RenderLegend()
	assert.True(t, strings.HasSuffix(legend, "\n"), "should end with newline")

	lines := strings.Split(strings.TrimSuffix(legend, "\n"), "\n")
	assert.Len(t, lines, 2, "should have 2 lines")

	assert.Equal(t, "Less ░░ ██ ██ ██ More", stripANSI(lines[0]))
	assert.Equal(t, "     0  1-4 5-9 10+", lines[1])
}

// --- renderCell ---

func TestRenderCell_ColorLevels(t *testing.T) {
	tests := []struct {
		count     int
		today     bool
		wantColor string
		wantBlock string
	}{
		{0, false, colorEmpty, "░░"},
		{1, false, colorLow, "██"},
		{4, false, colorLow, "██"},
		{5, false, colorMedium, "██"},
		{9, false, colorMedium, "██"},
		{10, false, colorHigh, "██"},
		{100, false, colorHigh, "██"},
		{0, true, colorToday, "░░"},
		{5, true, colorToday, "██"},
	}
	for _, tt := range tests {
		cell := renderCell(tt.count, tt.today)
		assert.Contains(t, cell, tt.wantColor, "count=%d today=%v", tt.count, tt.today)
		assert.Contains(t, cell, tt.wantBlock, "count=%d today=%v", tt.count, tt.today)
		assert.Contains(t, cell, colorReset, "count=%d today=%v", tt.count, tt.today)
	}
}

// --- RenderHeatmapWithOptions ---

func TestRenderHeatmapWithOptions_EmptyStats(t *testing.T) {
	loc := time.Local
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, loc)

	result := RenderHeatmapWithOptions(nil, HeatmapOptions{
		ShowLegend:  false,
		ShowSummary: false,
		Since:       start,
		Until:       end,
	})

	// 应有月份标题行和 7 行星期
	plain := stripANSI(result)
	lines := strings.Split(strings.TrimSuffix(plain, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 8, "header + 7 weekday rows")

	// 无提交，只有空心方块
	assert.NotContains(t, result, colorLow)
	assert.NotContains(t, result, colorMedium)
	assert.NotContains(t, result, colorHigh)
}

func TestRenderHeatmapWithOptions_WithData(t *testing.T) {
	loc := time.Local
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, loc)

	stats := map[time.Time]int{
		time.Date(2024, 6, 5, 0, 0, 0, 0, loc):  2,  // low
		time.Date(2024, 6, 10, 0, 0, 0, 0, loc): 7,  // medium
		time.Date(2024, 6, 15, 0, 0, 0, 0, loc): 15, // high
	}

	result := RenderHeatmapWithOptions(stats, HeatmapOptions{
		ShowLegend:  false,
		ShowSummary: false,
		Since:       start,
		Until:       end,
	})

	assert.Contains(t, result, colorLow)
	assert.Contains(t, result, colorMedium)
	assert.Contains(t, result, colorHigh)
}

func TestRenderHeatmapWithOptions_LegendToggle(t *testing.T) {
	loc := time.Local
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, loc)
	empty := map[time.Time]int{}

	withLegend := RenderHeatmapWithOptions(empty, HeatmapOptions{
		ShowLegend: true, ShowSummary: false,
		Since: start, Until: end,
	})
	withoutLegend := RenderHeatmapWithOptions(empty, HeatmapOptions{
		ShowLegend: false, ShowSummary: false,
		Since: start, Until: end,
	})

	assert.Contains(t, stripANSI(withLegend), "Less")
	assert.Contains(t, stripANSI(withLegend), "More")
	assert.NotContains(t, stripANSI(withoutLegend), "Less")
}

func TestRenderHeatmapWithOptions_SummaryToggle(t *testing.T) {
	loc := time.Local
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, loc)
	data := map[time.Time]int{
		time.Date(2024, 6, 10, 0, 0, 0, 0, loc): 3,
	}

	with := RenderHeatmapWithOptions(data, HeatmapOptions{
		ShowLegend: false, ShowSummary: true,
		Since: start, Until: end,
	})
	without := RenderHeatmapWithOptions(data, HeatmapOptions{
		ShowLegend: false, ShowSummary: false,
		Since: start, Until: end,
	})

	assert.Contains(t, with, "Total:")
	assert.NotContains(t, without, "Total:")
}

func TestRenderHeatmapWithOptions_TodayHighlight(t *testing.T) {
	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -7)

	data := map[time.Time]int{today: 1}

	result := RenderHeatmapWithOptions(data, HeatmapOptions{
		ShowLegend: false, ShowSummary: false,
		Since: start, Until: today,
	})

	assert.Contains(t, result, colorToday, "today should use special highlight color")
}

func TestRenderHeatmapWithOptions_InvalidRange(t *testing.T) {
	loc := time.Local
	start := time.Date(2024, 7, 1, 0, 0, 0, 0, loc)
	end := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)

	result := RenderHeatmapWithOptions(nil, HeatmapOptions{
		Since: start, Until: end,
	})
	assert.Empty(t, result, "start > end should return empty")
}

// --- weekdayLabel ---

func TestWeekdayLabel(t *testing.T) {
	assert.Equal(t, "Mon ", weekdayLabel(1))
	assert.Equal(t, "Wed ", weekdayLabel(3))
	assert.Equal(t, "Fri ", weekdayLabel(5))
	assert.Equal(t, "    ", weekdayLabel(0))
	assert.Equal(t, "    ", weekdayLabel(2))
	assert.Equal(t, "    ", weekdayLabel(4))
	assert.Equal(t, "    ", weekdayLabel(6))
}

// --- writeMonthHeader ---

func TestWriteMonthHeader_ShowsMonthNames(t *testing.T) {
	loc := time.Local
	// 4 weeks spanning Jun-Jul
	weekStarts := []time.Time{
		time.Date(2024, 6, 16, 0, 0, 0, 0, loc),
		time.Date(2024, 6, 23, 0, 0, 0, 0, loc),
		time.Date(2024, 6, 30, 0, 0, 0, 0, loc),
		time.Date(2024, 7, 7, 0, 0, 0, 0, loc),
	}

	var b strings.Builder
	writeMonthHeader(&b, weekStarts)
	header := b.String()

	assert.Contains(t, header, "Jun")
	assert.Contains(t, header, "Jul")
}

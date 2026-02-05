// Package stats 提供 Git 提交统计的收集和渲染功能。
package stats

import (
	"strings"
	"time"
)

// ANSI 颜色代码常量，用于终端热力图渲染。
const (
	colorReset = "\033[0m" // 重置颜色

	colorEmpty  = "\033[38;5;240m" // 灰色 - 无提交
	colorLow    = "\033[38;5;120m" // 浅绿 - 1-4 次提交
	colorMedium = "\033[38;5;76m"  // 中绿 - 5-9 次提交
	colorHigh   = "\033[38;5;34m"  // 深绿 - 10+ 次提交
	colorToday  = "\033[38;5;199m" // 粉色 - 今天（高亮显示）
)

// RenderHeatmap 将统计数据渲染为类似 GitHub 的贡献热力图。
// 热力图以周为列、星期几为行，使用不同颜色表示提交数量级别。
// 返回包含 ANSI 颜色代码的字符串，可直接输出到终端。
func RenderHeatmap(stats map[time.Time]int, months int) string {
	return renderHeatmap(stats, months, true, true)
}

// RenderHeatmapNoLegend 渲染热力图但不包含图例。
func RenderHeatmapNoLegend(stats map[time.Time]int, months int) string {
	return renderHeatmap(stats, months, false, true)
}

// RenderHeatmapNoSummary 渲染热力图但不包含摘要信息。
func RenderHeatmapNoSummary(stats map[time.Time]int, months int) string {
	return renderHeatmap(stats, months, true, false)
}

// RenderHeatmapNoLegendNoSummary 渲染热力图但不包含图例和摘要信息。
func RenderHeatmapNoLegendNoSummary(stats map[time.Time]int, months int) string {
	return renderHeatmap(stats, months, false, false)
}

func renderHeatmap(stats map[time.Time]int, months int, includeLegend bool, includeSummary bool) string {
	if months <= 0 {
		return ""
	}

	now := time.Now()
	loc := now.Location()

	// 计算时间范围
	start := heatmapStart(now, months)
	end := beginningOfDay(now, loc)

	// 构建每周的起始日期列表（用作列）
	weekStarts := make([]time.Time, 0, 32)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 7) {
		weekStarts = append(weekStarts, d)
	}

	var b strings.Builder
	// 写入月份标题行
	writeMonthHeader(&b, weekStarts)

	// 按行（星期几）渲染热力图
	for row := 0; row < 7; row++ {
		b.WriteString(weekdayLabel(row)) // 左侧星期标签

		for _, ws := range weekStarts {
			day := ws.AddDate(0, 0, row)
			// 跳过范围外的日期
			if day.Before(start) || day.After(end) {
				b.WriteString("    ")
				continue
			}

			key := beginningOfDay(day, loc)
			count := stats[key]
			isToday := key.Equal(end)
			b.WriteString(renderCell(count, isToday))
		}
		b.WriteByte('\n')
	}

	if includeLegend {
		b.WriteString(RenderLegend())
	}

	if includeSummary {
		b.WriteString(RenderSummary(CalculateSummary(stats)))
	}

	return b.String()
}

// writeMonthHeader 写入月份标题行。
// 在每月第一周的位置显示月份缩写（如 Jan, Feb）。
func writeMonthHeader(b *strings.Builder, weekStarts []time.Time) {
	b.WriteString("    ") // 左侧对齐空格（与星期标签对齐）

	lastMonth := time.Month(0)
	for _, ws := range weekStarts {
		month := ws.Month()
		if month != lastMonth {
			// 新月份开始，显示月份名称
			name := month.String()
			if len(name) > 3 {
				name = name[:3] // 截取前3个字符
			}
			b.WriteString(name)
			if len(name) < 3 {
				b.WriteString(strings.Repeat(" ", 3-len(name)))
			}
			b.WriteByte(' ')
			lastMonth = month
			continue
		}
		// 同一月份内的周，显示空格
		b.WriteString("    ")
	}
	b.WriteByte('\n')
}

// weekdayLabel 返回指定行（0-6，对应周日到周六）的星期标签。
// 只在周一、周三、周五显示标签，其他行显示空格。
func weekdayLabel(row int) string {
	switch row {
	case 1:
		return "Mon "
	case 3:
		return "Wed "
	case 5:
		return "Fri "
	default:
		return "    "
	}
}

// renderCell 渲染单个日期的热力图单元格。
// 根据提交数量选择颜色：
//   - 0: 灰色空心方块
//   - 1-4: 浅绿实心方块
//   - 5-9: 中绿实心方块
//   - 10+: 深绿实心方块
//   - 今天: 粉色高亮
func renderCell(count int, today bool) string {
	color := colorEmpty
	switch {
	case count > 0 && count < 5:
		color = colorLow
	case count >= 5 && count < 10:
		color = colorMedium
	case count >= 10:
		color = colorHigh
	}
	// 今天使用特殊颜色高亮
	if today {
		color = colorToday
	}

	// 无提交显示空心方块，有提交显示实心方块
	if count == 0 {
		return color + "░░" + colorReset + "  "
	}
	return color + "██" + colorReset + "  "
}

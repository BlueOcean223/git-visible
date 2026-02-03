package stats

import (
	"strings"
	"time"
)

const (
	colorReset = "\033[0m"

	colorEmpty  = "\033[38;5;240m"
	colorLow    = "\033[38;5;120m"
	colorMedium = "\033[38;5;76m"
	colorHigh   = "\033[38;5;34m"
	colorToday  = "\033[38;5;199m"
)

func RenderHeatmap(stats map[time.Time]int, months int) string {
	if months <= 0 {
		return ""
	}

	now := time.Now()
	loc := now.Location()

	start := heatmapStart(now, months)
	end := beginningOfDay(now, loc)

	weekStarts := make([]time.Time, 0, 32)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 7) {
		weekStarts = append(weekStarts, d)
	}

	var b strings.Builder
	writeMonthHeader(&b, weekStarts)

	for row := 0; row < 7; row++ {
		b.WriteString(weekdayLabel(row))

		for _, ws := range weekStarts {
			day := ws.AddDate(0, 0, row)
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

	return b.String()
}

func writeMonthHeader(b *strings.Builder, weekStarts []time.Time) {
	b.WriteString("    ")

	lastMonth := time.Month(0)
	for _, ws := range weekStarts {
		month := ws.Month()
		if month != lastMonth {
			name := month.String()
			if len(name) > 3 {
				name = name[:3]
			}
			b.WriteString(name)
			if len(name) < 3 {
				b.WriteString(strings.Repeat(" ", 3-len(name)))
			}
			b.WriteByte(' ')
			lastMonth = month
			continue
		}
		b.WriteString("    ")
	}
	b.WriteByte('\n')
}

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
	if today {
		color = colorToday
	}

	if count == 0 {
		return color + "░░" + colorReset + "  "
	}
	return color + "██" + colorReset + "  "
}

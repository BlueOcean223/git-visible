package stats

import "strings"

// RenderLegend 渲染热力图图例。
// 输出包含 ANSI 颜色代码的字符串，可直接输出到终端。
func RenderLegend() string {
	var b strings.Builder

	b.WriteString("Less ")
	b.WriteString(colorEmpty)
	b.WriteString("░░")
	b.WriteString(colorReset)
	b.WriteByte(' ')
	b.WriteString(colorLow)
	b.WriteString("██")
	b.WriteString(colorReset)
	b.WriteByte(' ')
	b.WriteString(colorMedium)
	b.WriteString("██")
	b.WriteString(colorReset)
	b.WriteByte(' ')
	b.WriteString(colorHigh)
	b.WriteString("██")
	b.WriteString(colorReset)
	b.WriteString(" More\n")

	b.WriteString("     0  1-4 5-9 10+\n")
	return b.String()
}


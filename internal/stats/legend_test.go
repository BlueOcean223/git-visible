package stats

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

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

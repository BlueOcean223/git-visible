package stats

import (
	"regexp"
	"strings"
	"testing"
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
		if !strings.Contains(legend, part) {
			t.Errorf("RenderLegend() missing %q in %q", part, legend)
		}
	}
}

func TestRenderLegend_Format(t *testing.T) {
	legend := RenderLegend()
	if !strings.HasSuffix(legend, "\n") {
		t.Fatalf("RenderLegend() should end with newline, got %q", legend)
	}

	lines := strings.Split(strings.TrimSuffix(legend, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("RenderLegend() should have 2 lines, got %d: %q", len(lines), legend)
	}

	if got := stripANSI(lines[0]); got != "Less ░░ ██ ██ ██ More" {
		t.Fatalf("RenderLegend() first line = %q, want %q", got, "Less ░░ ██ ██ ██ More")
	}
	if got := lines[1]; got != "     0  1-4 5-9 10+" {
		t.Fatalf("RenderLegend() second line = %q, want %q", got, "     0  1-4 5-9 10+")
	}
}


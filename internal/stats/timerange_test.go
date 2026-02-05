package stats

import (
	"testing"
	"time"
)

func TestParseDate_ISODate(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	got, err := ParseDate("2025-01-15")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	want := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ParseDate() = %v, want %v", got, want)
	}
}

func TestParseDate_YearMonth(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	got, err := ParseDate("2025-01")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	want := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ParseDate() = %v, want %v", got, want)
	}
}

func TestParseDate_Relative(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	tests := []struct {
		in   string
		want time.Time
	}{
		{"1w", time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC)},
		{"2m", time.Date(2025, 12, 5, 0, 0, 0, 0, time.UTC)},
		{"1y", time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseDate(tt.in)
			if err != nil {
				t.Fatalf("ParseDate(%q) error = %v", tt.in, err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("ParseDate(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDate_Invalid(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	_, err := ParseDate("not-a-date")
	if err == nil {
		t.Fatal("ParseDate(invalid) should return error")
	}
}

func TestTimeRange_SinceAfterUntil(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	_, _, err := TimeRange("2025-02-01", "2025-01-01", 6)
	if err == nil {
		t.Fatal("TimeRange(since > until) should return error")
	}
}

func TestTimeRange_Priority_SinceUntilOverridesMonths(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	start, end, err := TimeRange("2025-01-01", "2025-06-30", 1)
	if err != nil {
		t.Fatalf("TimeRange() error = %v", err)
	}

	wantStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
}

func TestTimeRange_OnlySince(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	start, end, err := TimeRange("2025-01-01", "", 0)
	if err != nil {
		t.Fatalf("TimeRange() error = %v", err)
	}

	wantStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
}

func TestTimeRange_OnlyUntil_UsesMonths(t *testing.T) {
	origNow := timeNow
	defer func() { timeNow = origNow }()

	timeNow = func() time.Time { return time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC) }

	start, end, err := TimeRange("", "2025-06-30", 6)
	if err != nil {
		t.Fatalf("TimeRange() error = %v", err)
	}

	wantEnd := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}

	// until (2025-06-30) - 6 months = 2024-12-30, then align back to Sunday => 2024-12-29.
	wantStart := time.Date(2024, 12, 29, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
}


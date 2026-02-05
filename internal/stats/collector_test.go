package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeginningOfDay(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name  string
		input time.Time
		want  time.Time
	}{
		{
			name:  "morning time",
			input: time.Date(2024, 6, 15, 9, 30, 45, 123, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:  "midnight",
			input: time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name:  "end of day",
			input: time.Date(2024, 6, 15, 23, 59, 59, 999999999, loc),
			want:  time.Date(2024, 6, 15, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := beginningOfDay(tt.input, loc)
			assert.True(t, got.Equal(tt.want), "beginningOfDay() = %v, want %v", got, tt.want)
		})
	}
}

func TestBeginningOfDay_DifferentTimezones(t *testing.T) {
	utc := time.UTC
	inputUTC := time.Date(2024, 6, 15, 10, 30, 0, 0, utc)

	loc := time.Local
	got := beginningOfDay(inputUTC, loc)

	assert.Equal(t, loc, got.Location(), "should be in local timezone")
	assert.Equal(t, 0, got.Hour(), "hour should be 0")
	assert.Equal(t, 0, got.Minute(), "minute should be 0")
	assert.Equal(t, 0, got.Second(), "second should be 0")
}

func TestHeatmapStart(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name   string
		now    time.Time
		months int
	}{
		{
			name:   "6 months",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 6,
		},
		{
			name:   "12 months",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 12,
		},
		{
			name:   "1 month",
			now:    time.Date(2024, 6, 15, 10, 0, 0, 0, loc),
			months: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := heatmapStart(tt.now, tt.months)

			assert.Equal(t, time.Sunday, got.Weekday(), "should start on Sunday")
			assert.Equal(t, 0, got.Hour(), "hour should be 0")
			assert.Equal(t, 0, got.Minute(), "minute should be 0")
			assert.Equal(t, 0, got.Second(), "second should be 0")

			expectedEarliest := tt.now.AddDate(0, -tt.months, -7)
			assert.False(t, got.Before(expectedEarliest), "should not be before expected earliest")
		})
	}
}

func TestCollectStats_InvalidMonths(t *testing.T) {
	_, err := CollectStatsMonths(nil, nil, 0)
	assert.Error(t, err, "months=0 should return error")

	_, err = CollectStatsMonths(nil, nil, -1)
	assert.Error(t, err, "months=-1 should return error")
}

func TestCollectStats_EmptyRepos(t *testing.T) {
	stats, err := CollectStatsMonths([]string{}, nil, 6)
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestCollectStats_NonExistentRepo(t *testing.T) {
	stats, err := CollectStatsMonths([]string{"/non/existent/repo"}, nil, 6)

	assert.Error(t, err, "non-existent repo should return error")
	assert.Empty(t, stats)
}

func TestMaxConcurrency(t *testing.T) {
	assert.GreaterOrEqual(t, maxConcurrency, 1, "maxConcurrency should be >= 1")
}

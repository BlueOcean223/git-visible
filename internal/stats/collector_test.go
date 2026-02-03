package stats

import (
	"testing"
	"time"
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
			if !got.Equal(tt.want) {
				t.Errorf("beginningOfDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBeginningOfDay_DifferentTimezones(t *testing.T) {
	// UTC 时间
	utc := time.UTC
	inputUTC := time.Date(2024, 6, 15, 10, 30, 0, 0, utc)

	// 转换到 Local 时区后取当天开始
	loc := time.Local
	got := beginningOfDay(inputUTC, loc)

	// 结果应该是 Local 时区的当天 00:00:00
	if got.Location() != loc {
		t.Errorf("beginningOfDay() location = %v, want %v", got.Location(), loc)
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("beginningOfDay() time = %v, want 00:00:00", got)
	}
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

			// 应该是周日
			if got.Weekday() != time.Sunday {
				t.Errorf("heatmapStart() weekday = %v, want Sunday", got.Weekday())
			}

			// 应该是当天开始（00:00:00）
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Errorf("heatmapStart() time = %v, want 00:00:00", got)
			}

			// 应该在指定月份之前
			expectedEarliest := tt.now.AddDate(0, -tt.months, -7) // 最多再往前 7 天找周日
			if got.Before(expectedEarliest) {
				t.Errorf("heatmapStart() = %v, too early (expected after %v)", got, expectedEarliest)
			}
		})
	}
}

func TestCollectStats_InvalidMonths(t *testing.T) {
	_, err := CollectStats(nil, nil, 0)
	if err == nil {
		t.Error("CollectStats(months=0) should return error")
	}

	_, err = CollectStats(nil, nil, -1)
	if err == nil {
		t.Error("CollectStats(months=-1) should return error")
	}
}

func TestCollectStats_EmptyRepos(t *testing.T) {
	stats, err := CollectStats([]string{}, nil, 6)
	if err != nil {
		t.Fatalf("CollectStats() error = %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("CollectStats() = %v, want empty map", stats)
	}
}

func TestCollectStats_NonExistentRepo(t *testing.T) {
	stats, err := CollectStats([]string{"/non/existent/repo"}, nil, 6)

	// 应该返回空结果和错误
	if err == nil {
		t.Error("CollectStats() with non-existent repo should return error")
	}
	if len(stats) != 0 {
		t.Errorf("CollectStats() = %v, want empty map", stats)
	}
}

func TestMaxConcurrency(t *testing.T) {
	// 验证 maxConcurrency 已设置为合理值
	if maxConcurrency < 1 {
		t.Errorf("maxConcurrency = %d, want >= 1", maxConcurrency)
	}
}

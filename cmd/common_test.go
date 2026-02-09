package cmd

import (
	"path/filepath"
	"testing"

	"git-visible/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareRun_EmailCleaning_TrimWhitespace(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "repo-1")})
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	runCtx, err := prepareRun([]string{" test@example.com "}, 0, "2025-01-01", "2025-12-31")
	require.NoError(t, err)
	assert.Equal(t, []string{"test@example.com"}, runCtx.Emails)
}

func TestPrepareRun_EmailCleaning_FilterBlankValues(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "repo-1")})
	setTestConfig(t, config.Config{Email: "", Months: config.DefaultMonths})

	runCtx, err := prepareRun([]string{"  "}, 0, "2025-01-01", "2025-12-31")
	require.NoError(t, err)
	assert.Empty(t, runCtx.Emails)
}

func TestPrepareRun_EmailCleaning_FallbackToTrimmedConfigEmail(t *testing.T) {
	home := withTempHome(t)
	writeReposFile(t, home, []string{filepath.Join(home, "repo-1")})
	setTestConfig(t, config.Config{Email: " config@example.com ", Months: config.DefaultMonths})

	runCtx, err := prepareRun(nil, 0, "2025-01-01", "2025-12-31")
	require.NoError(t, err)
	assert.Equal(t, []string{"config@example.com"}, runCtx.Emails)
}

func setTestConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	current, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, current)
	original := *current

	require.NoError(t, config.Save(cfg))
	t.Cleanup(func() {
		require.NoError(t, config.Save(original))
	})
}

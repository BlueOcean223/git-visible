package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	key := CacheKey{
		RepoPath:  "/tmp/example-repo",
		HEADHash:  "abc123def456",
		Emails:    []string{"b@example.com", "a@example.com"},
		TimeRange: "2024-01-01_2024-06-30",
		Branch:    "main",
		AllBranch: false,
	}
	stats := map[string]int{
		"2024-01-02": 2,
		"2024-01-03": 5,
	}

	require.NoError(t, SaveCache(key, stats))

	cachePath, err := getCachePath(key)
	require.NoError(t, err)
	assert.FileExists(t, cachePath)

	entry, err := LoadCache(key)
	require.NoError(t, err)
	assert.Equal(t, stats, entry.Stats)
	assert.False(t, entry.CreatedAt.IsZero())
	assert.Equal(t, key.String(), entry.Key.String())
}

func TestCacheKeyStability(t *testing.T) {
	key1 := CacheKey{
		RepoPath:  "/tmp/repo",
		HEADHash:  "deadbeef",
		Emails:    []string{"z@example.com", "a@example.com"},
		TimeRange: "2024-01-01_2024-01-31",
		Branch:    "main",
		AllBranch: true,
	}
	key2 := CacheKey{
		RepoPath:  "/tmp/repo",
		HEADHash:  "deadbeef",
		Emails:    []string{"a@example.com", "z@example.com"},
		TimeRange: "2024-01-01_2024-01-31",
		Branch:    "main",
		AllBranch: true,
	}

	assert.Equal(t, key1.String(), key2.String())
}

func TestCacheMiss(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	key := CacheKey{
		RepoPath:  filepath.Join(home, "repo"),
		HEADHash:  "missing",
		TimeRange: "2024-01-01_2024-01-31",
	}

	_, err := LoadCache(key)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

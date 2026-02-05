package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty path", "", true},
		{"whitespace only", "   ", true},
		{"relative path", ".", false},
		{"absolute path", "/tmp", false},
		{"tilde expansion", "~", false},
		{"tilde with subpath", "~/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizePath(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, filepath.IsAbs(result), "should return absolute path")
		})
	}
}

func TestNormalizePath_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get user home dir")
	}

	result, err := normalizePath("~")
	require.NoError(t, err)
	assert.Equal(t, home, result)

	result, err = normalizePath("~/foo")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "foo"), result)
}

func TestIsValidRepo(t *testing.T) {
	tmpDir := t.TempDir()

	validRepo := filepath.Join(tmpDir, "valid-repo")
	require.NoError(t, os.MkdirAll(filepath.Join(validRepo, ".git"), 0o755))

	invalidRepo := filepath.Join(tmpDir, "invalid-repo")
	require.NoError(t, os.MkdirAll(invalidRepo, 0o755))

	nonExistent := filepath.Join(tmpDir, "non-existent")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"valid repo", validRepo, true},
		{"invalid repo (no .git)", invalidRepo, false},
		{"non-existent path", nonExistent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidRepo(tt.path))
		})
	}
}

func TestLoadRepos_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	repos, err := LoadRepos()
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestAddAndRemoveRepo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "git-visible")
	require.NoError(t, os.MkdirAll(configDir, 0o700))

	repoPath := filepath.Join(tmpDir, "my-repo")
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	// Add repo
	require.NoError(t, AddRepo(repoPath))

	repos, err := LoadRepos()
	require.NoError(t, err)
	assert.Len(t, repos, 1)

	// Duplicate add should be silent
	require.NoError(t, AddRepo(repoPath))
	repos, _ = LoadRepos()
	assert.Len(t, repos, 1, "duplicate add should not create duplicate")

	// Remove repo
	require.NoError(t, RemoveRepo(repoPath))
	repos, _ = LoadRepos()
	assert.Empty(t, repos)

	// Remove non-existent should be silent
	require.NoError(t, RemoveRepo(repoPath))
}

func TestVerifyRepos(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "git-visible")
	require.NoError(t, os.MkdirAll(configDir, 0o700))

	validRepo := filepath.Join(tmpDir, "valid-repo")
	require.NoError(t, os.MkdirAll(filepath.Join(validRepo, ".git"), 0o755))

	invalidRepo := filepath.Join(tmpDir, "invalid-repo")
	require.NoError(t, os.MkdirAll(invalidRepo, 0o755))

	require.NoError(t, AddRepo(validRepo))
	require.NoError(t, AddRepo(invalidRepo))

	valid, invalid, err := VerifyRepos()
	require.NoError(t, err)

	assert.Equal(t, []string{validRepo}, valid)
	assert.Equal(t, []string{invalidRepo}, invalid)
}

package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRepos_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	repo1 := filepath.Join(tmpDir, "repo1")
	repo2 := filepath.Join(tmpDir, "subdir", "repo2")
	require.NoError(t, os.MkdirAll(filepath.Join(repo1, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repo2, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "not-a-repo"), 0o755))

	repos, err := ScanRepos(tmpDir, -1, nil)
	require.NoError(t, err)
	assert.Len(t, repos, 2)
}

func TestScanRepos_DepthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	deepRepo := filepath.Join(tmpDir, "a", "b", "c", "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(deepRepo, ".git"), 0o755))

	repos, err := ScanRepos(tmpDir, 2, nil)
	require.NoError(t, err)
	assert.Empty(t, repos, "depth=2 should not find deep repo")

	repos, err = ScanRepos(tmpDir, 4, nil)
	require.NoError(t, err)
	assert.Len(t, repos, 1, "depth=4 should find the repo")
}

func TestScanRepos_Excludes(t *testing.T) {
	tmpDir := t.TempDir()

	excludedRepo := filepath.Join(tmpDir, "excluded", "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(excludedRepo, ".git"), 0o755))

	includedRepo := filepath.Join(tmpDir, "included", "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(includedRepo, ".git"), 0o755))

	repos, err := ScanRepos(tmpDir, -1, []string{"excluded"})
	require.NoError(t, err)

	assert.Len(t, repos, 1)
	if len(repos) > 0 {
		assert.Equal(t, includedRepo, repos[0])
	}
}

func TestScanRepos_DefaultExcludes(t *testing.T) {
	tmpDir := t.TempDir()

	nodeModulesRepo := filepath.Join(tmpDir, "project", "node_modules", "some-pkg")
	require.NoError(t, os.MkdirAll(filepath.Join(nodeModulesRepo, ".git"), 0o755))

	normalRepo := filepath.Join(tmpDir, "project", "src")
	require.NoError(t, os.MkdirAll(filepath.Join(normalRepo, ".git"), 0o755))

	repos, err := ScanRepos(tmpDir, -1, nil)
	require.NoError(t, err)

	assert.Len(t, repos, 1, "should only find normalRepo, not node_modules")
	if len(repos) > 0 {
		assert.Equal(t, normalRepo, repos[0])
	}
}

func TestScanRepos_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("test"), 0o644))

	_, err := ScanRepos(filePath, -1, nil)
	assert.Error(t, err, "scanning a file should return error")
}

func TestScanRepos_NonExistent(t *testing.T) {
	_, err := ScanRepos("/non/existent/path", -1, nil)
	assert.Error(t, err, "non-existent path should return error")
}

func TestIsExcluded(t *testing.T) {
	rootPath := "/home/user/code"

	tests := []struct {
		name     string
		path     string
		dirName  string
		excludes []string
		want     bool
	}{
		{
			name:     "match by directory name",
			path:     "/home/user/code/project/vendor",
			dirName:  "vendor",
			excludes: []string{"vendor"},
			want:     true,
		},
		{
			name:     "match by absolute path",
			path:     "/home/user/code/secret",
			dirName:  "secret",
			excludes: []string{"/home/user/code/secret"},
			want:     true,
		},
		{
			name:     "match by relative path",
			path:     "/home/user/code/project/build",
			dirName:  "build",
			excludes: []string{"project/build"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "/home/user/code/src",
			dirName:  "src",
			excludes: []string{"vendor", "node_modules"},
			want:     false,
		},
		{
			name:     "empty excludes",
			path:     "/home/user/code/anything",
			dirName:  "anything",
			excludes: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcluded(rootPath, tt.path, tt.dirName, tt.excludes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeExcludes(t *testing.T) {
	input := []string{"  vendor  ", "", "node_modules", "   ", "build"}
	got := normalizeExcludes(input)

	want := []string{"vendor", "node_modules", "build"}
	assert.Equal(t, want, got)
}

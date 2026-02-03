package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRepos_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建两个 git 仓库
	repo1 := filepath.Join(tmpDir, "repo1")
	repo2 := filepath.Join(tmpDir, "subdir", "repo2")
	if err := os.MkdirAll(filepath.Join(repo1, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo2, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 创建一个普通目录
	if err := os.MkdirAll(filepath.Join(tmpDir, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := ScanRepos(tmpDir, -1, nil)
	if err != nil {
		t.Fatalf("ScanRepos() error = %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("ScanRepos() found %d repos, want 2", len(repos))
	}
}

func TestScanRepos_DepthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建深层仓库
	deepRepo := filepath.Join(tmpDir, "a", "b", "c", "repo")
	if err := os.MkdirAll(filepath.Join(deepRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 深度限制为 2，应该找不到深层仓库
	repos, err := ScanRepos(tmpDir, 2, nil)
	if err != nil {
		t.Fatalf("ScanRepos() error = %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("ScanRepos(depth=2) found %d repos, want 0", len(repos))
	}

	// 深度限制为 4，应该找到
	repos, err = ScanRepos(tmpDir, 4, nil)
	if err != nil {
		t.Fatalf("ScanRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("ScanRepos(depth=4) found %d repos, want 1", len(repos))
	}
}

func TestScanRepos_Excludes(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建要排除的仓库
	excludedRepo := filepath.Join(tmpDir, "excluded", "repo")
	if err := os.MkdirAll(filepath.Join(excludedRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 创建要包含的仓库
	includedRepo := filepath.Join(tmpDir, "included", "repo")
	if err := os.MkdirAll(filepath.Join(includedRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := ScanRepos(tmpDir, -1, []string{"excluded"})
	if err != nil {
		t.Fatalf("ScanRepos() error = %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("ScanRepos() found %d repos, want 1", len(repos))
	}
	if len(repos) > 0 && repos[0] != includedRepo {
		t.Errorf("ScanRepos() found %s, want %s", repos[0], includedRepo)
	}
}

func TestScanRepos_DefaultExcludes(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 node_modules 内的仓库（应被默认排除）
	nodeModulesRepo := filepath.Join(tmpDir, "project", "node_modules", "some-pkg")
	if err := os.MkdirAll(filepath.Join(nodeModulesRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 创建正常仓库
	normalRepo := filepath.Join(tmpDir, "project", "src")
	if err := os.MkdirAll(filepath.Join(normalRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := ScanRepos(tmpDir, -1, nil)
	if err != nil {
		t.Fatalf("ScanRepos() error = %v", err)
	}

	// 应该只找到 normalRepo，node_modules 被默认排除
	if len(repos) != 1 {
		t.Errorf("ScanRepos() found %d repos, want 1", len(repos))
	}
	if len(repos) > 0 && repos[0] != normalRepo {
		t.Errorf("ScanRepos() found %s, want %s", repos[0], normalRepo)
	}
}

func TestScanRepos_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ScanRepos(filePath, -1, nil)
	if err == nil {
		t.Error("ScanRepos() on file should return error")
	}
}

func TestScanRepos_NonExistent(t *testing.T) {
	_, err := ScanRepos("/non/existent/path", -1, nil)
	if err == nil {
		t.Error("ScanRepos() on non-existent path should return error")
	}
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
			if got := isExcluded(rootPath, tt.path, tt.dirName, tt.excludes); got != tt.want {
				t.Errorf("isExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeExcludes(t *testing.T) {
	input := []string{"  vendor  ", "", "node_modules", "   ", "build"}
	got := normalizeExcludes(input)

	want := []string{"vendor", "node_modules", "build"}
	if len(got) != len(want) {
		t.Fatalf("normalizeExcludes() got %d items, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("normalizeExcludes()[%d] = %q, want %q", i, got[i], v)
		}
	}
}

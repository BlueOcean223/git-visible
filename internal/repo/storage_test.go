package repo

import (
	"os"
	"path/filepath"
	"testing"
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
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !filepath.IsAbs(result) {
				t.Errorf("normalizePath(%q) = %q, want absolute path", tt.input, result)
			}
		})
	}
}

func TestNormalizePath_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get user home dir")
	}

	result, err := normalizePath("~")
	if err != nil {
		t.Fatalf("normalizePath(~) error = %v", err)
	}
	if result != home {
		t.Errorf("normalizePath(~) = %q, want %q", result, home)
	}

	result, err = normalizePath("~/foo")
	if err != nil {
		t.Fatalf("normalizePath(~/foo) error = %v", err)
	}
	want := filepath.Join(home, "foo")
	if result != want {
		t.Errorf("normalizePath(~/foo) = %q, want %q", result, want)
	}
}

func TestIsValidRepo(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()

	// 有效仓库：包含 .git 目录
	validRepo := filepath.Join(tmpDir, "valid-repo")
	if err := os.MkdirAll(filepath.Join(validRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 无效仓库：不包含 .git 目录
	invalidRepo := filepath.Join(tmpDir, "invalid-repo")
	if err := os.MkdirAll(invalidRepo, 0o755); err != nil {
		t.Fatal(err)
	}

	// 不存在的路径
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
			if got := isValidRepo(tt.path); got != tt.want {
				t.Errorf("isValidRepo(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadRepos_EmptyFile(t *testing.T) {
	// 设置临时配置目录
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	repos, err := LoadRepos()
	if err != nil {
		t.Fatalf("LoadRepos() error = %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("LoadRepos() = %v, want empty slice", repos)
	}
}

func TestAddAndRemoveRepo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// 创建配置目录
	configDir := filepath.Join(tmpDir, ".config", "git-visible")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// 创建一个假仓库目录
	repoPath := filepath.Join(tmpDir, "my-repo")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}

	// 添加仓库
	if err := AddRepo(repoPath); err != nil {
		t.Fatalf("AddRepo() error = %v", err)
	}

	// 验证已添加
	repos, err := LoadRepos()
	if err != nil {
		t.Fatalf("LoadRepos() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("LoadRepos() got %d repos, want 1", len(repos))
	}

	// 重复添加应该静默成功
	if err := AddRepo(repoPath); err != nil {
		t.Fatalf("AddRepo() duplicate error = %v", err)
	}
	repos, _ = LoadRepos()
	if len(repos) != 1 {
		t.Errorf("After duplicate add: got %d repos, want 1", len(repos))
	}

	// 移除仓库
	if err := RemoveRepo(repoPath); err != nil {
		t.Fatalf("RemoveRepo() error = %v", err)
	}

	repos, _ = LoadRepos()
	if len(repos) != 0 {
		t.Errorf("After remove: got %d repos, want 0", len(repos))
	}

	// 重复移除应该静默成功
	if err := RemoveRepo(repoPath); err != nil {
		t.Fatalf("RemoveRepo() non-existent error = %v", err)
	}
}

func TestVerifyRepos(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// 创建配置目录
	configDir := filepath.Join(tmpDir, ".config", "git-visible")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// 创建有效仓库
	validRepo := filepath.Join(tmpDir, "valid-repo")
	if err := os.MkdirAll(filepath.Join(validRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 创建无效仓库
	invalidRepo := filepath.Join(tmpDir, "invalid-repo")
	if err := os.MkdirAll(invalidRepo, 0o755); err != nil {
		t.Fatal(err)
	}

	// 添加两个仓库
	if err := AddRepo(validRepo); err != nil {
		t.Fatal(err)
	}
	if err := AddRepo(invalidRepo); err != nil {
		t.Fatal(err)
	}

	valid, invalid, err := VerifyRepos()
	if err != nil {
		t.Fatalf("VerifyRepos() error = %v", err)
	}

	if len(valid) != 1 || valid[0] != validRepo {
		t.Errorf("VerifyRepos() valid = %v, want [%s]", valid, validRepo)
	}
	if len(invalid) != 1 || invalid[0] != invalidRepo {
		t.Errorf("VerifyRepos() invalid = %v, want [%s]", invalid, invalidRepo)
	}
}

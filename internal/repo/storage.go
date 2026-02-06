package repo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"git-visible/internal/config"
)

// reposFileName 是存储仓库列表的文件名。
const reposFileName = "repos"

// reposFile 返回仓库列表文件的完整路径。
func reposFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, reposFileName), nil
}

// normalizePath 标准化路径：
// 1. 去除首尾空白
// 2. 展开 ~ 为用户主目录
// 3. 转换为绝对路径
// 4. 清理路径（移除多余的分隔符和 . 或 ..）
func normalizePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}

	// 展开 ~ 为用户主目录
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = home
		} else {
			p = filepath.Join(home, p[2:])
		}
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// LoadRepos 从存储文件加载仓库列表。
// 返回的路径列表已去重和标准化。
// 如果存储文件不存在，返回空列表而不是错误。
func LoadRepos() ([]string, error) {
	path, err := reposFile()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	seen := make(map[string]struct{}, len(lines)) // 用于去重
	repos := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized, err := normalizePath(line)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue // 跳过重复项
		}
		seen[normalized] = struct{}{}
		repos = append(repos, normalized)
	}

	return repos, nil
}

// saveRepos 将仓库列表保存到存储文件。
func saveRepos(repos []string) error {
	if err := config.EnsureDir(); err != nil {
		return err
	}

	path, err := reposFile()
	if err != nil {
		return err
	}

	data := strings.Join(repos, "\n")
	if len(repos) > 0 {
		data += "\n" // 确保文件以换行符结尾
	}
	return os.WriteFile(path, []byte(data), 0o600)
}

// AddRepos 批量添加仓库到列表（如果不存在）。
// 路径会被标准化后存储，已存在的仓库会被静默忽略。
// 返回实际新增的仓库数量。
func AddRepos(paths []string) (added int, err error) {
	repos, err := LoadRepos()
	if err != nil {
		return 0, err
	}

	existing := make(map[string]struct{}, len(repos))
	for _, p := range repos {
		existing[p] = struct{}{}
	}

	toAdd := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized, err := normalizePath(path)
		if err != nil {
			return 0, err
		}
		if _, ok := existing[normalized]; ok {
			continue
		}
		existing[normalized] = struct{}{}
		toAdd = append(toAdd, normalized)
	}

	if len(toAdd) == 0 {
		return 0, nil
	}

	repos = append(repos, toAdd...)
	if err := saveRepos(repos); err != nil {
		return 0, err
	}

	return len(toAdd), nil
}

// AddRepo 添加仓库到列表（如果不存在）。
// 路径会被标准化后存储，已存在的仓库会被静默忽略。
func AddRepo(path string) error {
	_, err := AddRepos([]string{path})
	return err
}

// RemoveRepo 从列表中移除指定仓库。
// 如果仓库不在列表中，静默返回成功。
func RemoveRepo(path string) error {
	normalized, err := normalizePath(path)
	if err != nil {
		return err
	}

	repos, err := LoadRepos()
	if err != nil {
		return err
	}

	// 过滤掉要移除的仓库
	kept := make([]string, 0, len(repos))
	for _, existing := range repos {
		if existing == normalized {
			continue
		}
		kept = append(kept, existing)
	}

	return saveRepos(kept)
}

// isValidRepo 检查路径是否指向有效的 Git 仓库。
// 有效的仓库需要满足：路径存在、是目录、包含 .git 子目录。
func isValidRepo(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// VerifyRepos 验证所有已添加的仓库，返回有效和无效的仓库列表。
func VerifyRepos() (valid []string, invalid []string, err error) {
	repos, err := LoadRepos()
	if err != nil {
		return nil, nil, err
	}

	for _, path := range repos {
		if isValidRepo(path) {
			valid = append(valid, path)
		} else {
			invalid = append(invalid, path)
		}
	}
	return valid, invalid, nil
}

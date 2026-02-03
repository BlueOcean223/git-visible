package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// defaultExcludes 是默认排除的目录名列表，这些目录不可能是 git 仓库且通常体积较大。
var defaultExcludes = map[string]struct{}{
	// Node.js
	"node_modules": {},
	// Go / PHP
	"vendor": {},
	// Python
	".venv": {},
	"venv":  {},
	"env":   {},
	"__pycache__": {},
	".tox":        {},
	// Build outputs
	"dist":   {},
	"build":  {},
	"target": {},
	"out":    {},
	// Java/Gradle/Maven
	".gradle": {},
	".m2":     {},
	// iOS
	"Pods": {},
	// Package manager caches
	".npm":  {},
	".yarn": {},
	".pnpm-store": {},
	"bower_components": {},
	// IDE / Editor
	".idea":   {},
	".vscode": {},
	// Misc caches
	".cache": {},
	".tmp":   {},
}

// ScanRepos 递归扫描指定目录下的所有 Git 仓库。
// 参数:
//   - root: 要扫描的根目录
//   - depth: 最大递归深度，-1 表示无限制
//   - excludes: 要排除的目录列表（支持目录名、相对路径和绝对路径）
//
// 返回按路径排序的仓库路径列表。
func ScanRepos(root string, depth int, excludes []string) ([]string, error) {
	rootPath, err := normalizePath(root)
	if err != nil {
		return nil, err
	}

	// 检查根路径是否存在且为目录
	st, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", rootPath)
	}

	// 标准化排除目录列表
	excludes = normalizeExcludes(excludes)

	var repos []string
	seen := make(map[string]struct{}) // 用于去重

	bar := newScanProgressBar()
	if bar != nil {
		defer func() { _ = bar.Finish() }()
	}

	// 开始扫描
	if err := scanDir(bar, rootPath, rootPath, 0, depth, excludes, &repos, seen); err != nil {
		return nil, err
	}

	sort.Strings(repos)
	return repos, nil
}

// normalizeExcludes 清理和标准化排除目录列表。
func normalizeExcludes(excludes []string) []string {
	out := make([]string, 0, len(excludes))
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}
		out = append(out, ex)
	}
	return out
}

// scanDir 递归扫描目录，查找包含 .git 子目录的 Git 仓库。
// 当找到 Git 仓库时停止继续深入该仓库（不扫描子模块）。
func scanDir(bar *progressbar.ProgressBar, rootPath, dir string, currentDepth, depthLimit int, excludes []string, repos *[]string, seen map[string]struct{}) error {
	if bar != nil {
		_ = bar.Add(1)
	}

	// 检查当前目录是否是 Git 仓库
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		// 找到一个 Git 仓库，且之前未记录过，加入结果列表
		if _, ok := seen[dir]; !ok {
			seen[dir] = struct{}{}
			*repos = append(*repos, dir)
			if bar != nil {
				bar.Describe(fmt.Sprintf("scanning (%d found)", len(*repos)))
			}
		}
		return nil // 找到仓库后不再深入扫描
	}

	// 检查是否达到深度限制
	if depthLimit >= 0 && currentDepth >= depthLimit {
		return nil
	}

	// 读取目录内容
	entries, err := os.ReadDir(dir)
	if err != nil {
		// 忽略权限错误，继续扫描其他目录
		if os.IsPermission(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		// 跳过文件
		if !entry.IsDir() {
			continue
		}
		// 跳过符号链接，避免循环引用
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		name := entry.Name()
		// 跳过 .git 目录本身
		if name == ".git" {
			continue
		}
		// 跳过默认排除的目录（node_modules、vendor 等）
		if _, skip := defaultExcludes[name]; skip {
			continue
		}

		child := filepath.Join(dir, name)
		// 检查是否在用户指定的排除列表中
		if isExcluded(rootPath, child, name, excludes) {
			continue
		}

		// 递归扫描子目录
		if err := scanDir(bar, rootPath, child, currentDepth+1, depthLimit, excludes, repos, seen); err != nil {
			return err
		}
	}

	return nil
}

// isExcluded 检查给定路径是否应被排除。
// 支持三种排除模式：
// 1. 目录名匹配（如 "node_modules"）
// 2. 绝对路径匹配
// 3. 相对于扫描根目录的路径匹配
func isExcluded(rootPath, path, name string, excludes []string) bool {
	path = filepath.Clean(path)
	sep := string(os.PathSeparator)

	for _, ex := range excludes {
		// 直接匹配目录名
		if ex == name {
			return true
		}

		// 展开 ~ 为用户主目录
		if ex == "~" || strings.HasPrefix(ex, "~/") {
			expanded, err := normalizePath(ex)
			if err == nil {
				ex = expanded
			}
		}

		// 构建完整的排除路径
		var exPath string
		if filepath.IsAbs(ex) {
			exPath = filepath.Clean(ex)
		} else {
			exPath = filepath.Join(rootPath, filepath.Clean(ex))
		}

		// 检查路径是否匹配或是其子路径
		if path == exPath || strings.HasPrefix(path, exPath+sep) {
			return true
		}
	}

	return false
}

// newScanProgressBar 创建扫描进度条。
// 仅在终端环境下显示，非终端环境返回 nil。
func newScanProgressBar() *progressbar.ProgressBar {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}

	return progressbar.NewOptions(
		-1, // 未知总数
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("scanning"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(65*time.Millisecond),
	)
}

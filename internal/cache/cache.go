// Package cache 提供基于 JSON 文件的统计结果缓存。
// 缓存文件存储在 ~/.config/git-visible/cache/ 目录下，
// 以仓库名 + 参数哈希命名，通过 HEAD hash 实现自动失效。
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CacheKey 唯一标识一次仓库扫描的上下文参数。
// 任何参数变化（包括 HEAD 推进）都会产生不同的缓存键，从而自动失效旧缓存。
type CacheKey struct {
	RepoPath  string
	HEADHash  string
	Emails    []string // 排序后存储，保证顺序无关
	TimeRange string   // 格式 "2024-01-01_2024-06-30"
	Branch    string
	AllBranch bool
}

// CacheEntry 是持久化到磁盘的缓存条目。
type CacheEntry struct {
	Key       CacheKey       `json:"key"`
	Stats     map[string]int `json:"stats"` // 日期字符串 -> 提交数
	CreatedAt time.Time      `json:"created_at"`
}

// String 返回稳定的短文件名，格式为 "{repoName}_{hash}.json"。
// 对 key 先做规范化（路径清理、邮箱排序），再取 SHA-256 前 8 字节作为摘要，
// 保证相同参数（不论邮箱顺序）总是映射到同一文件。
func (k CacheKey) String() string {
	normalized := normalizeKey(k)
	repoName := sanitizeFileComponent(filepath.Base(normalized.RepoPath))
	if repoName == "" {
		repoName = "repo"
	}

	payload := strings.Join([]string{
		normalized.RepoPath,
		normalized.HEADHash,
		strings.Join(normalized.Emails, ","),
		normalized.TimeRange,
		normalized.Branch,
		fmt.Sprintf("%t", normalized.AllBranch),
	}, "\n")
	digest := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%s_%x.json", repoName, digest[:8])
}

// LoadCache 从磁盘读取并反序列化一条缓存。
// 缓存未命中时返回 os.ErrNotExist。
func LoadCache(key CacheKey) (*CacheEntry, error) {
	cachePath, err := getCachePath(key)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// SaveCache 将统计结果序列化并写入磁盘。
// 写入使用 tmp + rename 的原子策略，避免并发读到半写文件。
// stats 会被拷贝一份，调用方可安全修改原 map。
func SaveCache(key CacheKey, stats map[string]int) error {
	cachePath, err := getCachePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		return err
	}

	// 拷贝 stats，避免持有调用方的 map 引用
	statsCopy := make(map[string]int, len(stats))
	maps.Copy(statsCopy, stats)

	entry := CacheEntry{
		Key:       normalizeKey(key),
		Stats:     statsCopy,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	// 原子写入：先写临时文件，再 rename
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// getCachePath 返回缓存文件的完整路径。
func getCachePath(key CacheKey) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".config", "git-visible", "cache")
	return filepath.Join(cacheDir, key.String()), nil
}

// normalizeKey 规范化缓存键：清理路径、去除空白、邮箱排序。
// 保证相同语义的参数产生相同的键。
func normalizeKey(key CacheKey) CacheKey {
	normalized := key
	normalized.RepoPath = filepath.Clean(strings.TrimSpace(normalized.RepoPath))
	normalized.HEADHash = strings.TrimSpace(normalized.HEADHash)
	normalized.TimeRange = strings.TrimSpace(normalized.TimeRange)
	normalized.Branch = strings.TrimSpace(normalized.Branch)
	normalized.Emails = normalizeEmails(normalized.Emails)
	return normalized
}

// normalizeEmails 清洗邮箱列表：去空白、去空串、按字典序排序。
func normalizeEmails(emails []string) []string {
	if len(emails) == 0 {
		return nil
	}

	cleaned := make([]string, 0, len(emails))
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		cleaned = append(cleaned, email)
	}
	sort.Strings(cleaned)
	return cleaned
}

// sanitizeFileComponent 清理文件名组成部分，将路径分隔符、空格、冒号替换为下划线。
func sanitizeFileComponent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return ""
	}
	replacer := strings.NewReplacer(
		string(filepath.Separator), "_",
		" ", "_",
		":", "_",
	)
	return replacer.Replace(name)
}

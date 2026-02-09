package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CacheKey uniquely identifies a cached result for one repository scan context.
type CacheKey struct {
	RepoPath  string
	HEADHash  string
	Emails    []string // sorted
	TimeRange string   // "2024-01-01_2024-06-30"
	Branch    string
	AllBranch bool
}

// CacheEntry is the persisted cache payload.
type CacheEntry struct {
	Key       CacheKey       `json:"key"`
	Stats     map[string]int `json:"stats"` // day -> count
	CreatedAt time.Time      `json:"created_at"`
}

// String returns a stable, short cache file name for the key.
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

// LoadCache reads and deserializes one cache entry from disk.
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

// SaveCache serializes and writes one cache entry to disk.
func SaveCache(key CacheKey, stats map[string]int) error {
	cachePath, err := getCachePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}

	statsCopy := make(map[string]int, len(stats))
	for day, count := range stats {
		statsCopy[day] = count
	}

	entry := CacheEntry{
		Key:       normalizeKey(key),
		Stats:     statsCopy,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func getCachePath(key CacheKey) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".config", "git-visible", "cache")
	return filepath.Join(cacheDir, key.String()), nil
}

func normalizeKey(key CacheKey) CacheKey {
	normalized := key
	normalized.RepoPath = filepath.Clean(strings.TrimSpace(normalized.RepoPath))
	normalized.HEADHash = strings.TrimSpace(normalized.HEADHash)
	normalized.TimeRange = strings.TrimSpace(normalized.TimeRange)
	normalized.Branch = strings.TrimSpace(normalized.Branch)
	normalized.Emails = normalizeEmails(normalized.Emails)
	return normalized
}

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

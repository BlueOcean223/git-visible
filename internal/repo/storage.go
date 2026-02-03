package repo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"git-visible/internal/config"
)

const reposFileName = "repos"

func reposFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, reposFileName), nil
}

func normalizePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}

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
	seen := make(map[string]struct{}, len(lines))
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
			continue
		}
		seen[normalized] = struct{}{}
		repos = append(repos, normalized)
	}

	return repos, nil
}

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
		data += "\n"
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func AddRepo(path string) error {
	normalized, err := normalizePath(path)
	if err != nil {
		return err
	}

	repos, err := LoadRepos()
	if err != nil {
		return err
	}

	for _, existing := range repos {
		if existing == normalized {
			return nil
		}
	}

	repos = append(repos, normalized)
	return saveRepos(repos)
}

func RemoveRepo(path string) error {
	normalized, err := normalizePath(path)
	if err != nil {
		return err
	}

	repos, err := LoadRepos()
	if err != nil {
		return err
	}

	kept := make([]string, 0, len(repos))
	for _, existing := range repos {
		if existing == normalized {
			continue
		}
		kept = append(kept, existing)
	}

	return saveRepos(kept)
}

func isValidRepo(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

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

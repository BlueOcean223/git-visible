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

func ScanRepos(root string, depth int, excludes []string) ([]string, error) {
	rootPath, err := normalizePath(root)
	if err != nil {
		return nil, err
	}

	st, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", rootPath)
	}

	excludes = normalizeExcludes(excludes)

	var repos []string
	seen := make(map[string]struct{})

	bar := newScanProgressBar()
	if bar != nil {
		defer func() { _ = bar.Finish() }()
	}

	if err := scanDir(bar, rootPath, rootPath, 0, depth, excludes, &repos, seen); err != nil {
		return nil, err
	}

	sort.Strings(repos)
	return repos, nil
}

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

func scanDir(bar *progressbar.ProgressBar, rootPath, dir string, currentDepth, depthLimit int, excludes []string, repos *[]string, seen map[string]struct{}) error {
	if bar != nil {
		_ = bar.Add(1)
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		if _, ok := seen[dir]; !ok {
			seen[dir] = struct{}{}
			*repos = append(*repos, dir)
			if bar != nil {
				bar.Describe(fmt.Sprintf("scanning (%d found)", len(*repos)))
			}
		}
		return nil
	}

	if depthLimit >= 0 && currentDepth >= depthLimit {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsPermission(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		name := entry.Name()
		if name == ".git" {
			continue
		}

		child := filepath.Join(dir, name)
		if isExcluded(rootPath, child, name, excludes) {
			continue
		}

		if err := scanDir(bar, rootPath, child, currentDepth+1, depthLimit, excludes, repos, seen); err != nil {
			return err
		}
	}

	return nil
}

func isExcluded(rootPath, path, name string, excludes []string) bool {
	path = filepath.Clean(path)
	sep := string(os.PathSeparator)

	for _, ex := range excludes {
		if ex == name {
			return true
		}

		if ex == "~" || strings.HasPrefix(ex, "~/") {
			expanded, err := normalizePath(ex)
			if err == nil {
				ex = expanded
			}
		}

		var exPath string
		if filepath.IsAbs(ex) {
			exPath = filepath.Clean(ex)
		} else {
			exPath = filepath.Join(rootPath, filepath.Clean(ex))
		}

		if path == exPath || strings.HasPrefix(path, exPath+sep) {
			return true
		}
	}

	return false
}

func newScanProgressBar() *progressbar.ProgressBar {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}

	return progressbar.NewOptions(
		-1,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("scanning"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(65*time.Millisecond),
	)
}

package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	repoCountWarnThreshold = 50
	gitSizeWarnThreshold   = int64(1 << 30) // 1GB
)

// CheckBranchReachability 检查仓库 HEAD 和指定分支是否可达（有提交）。
func CheckBranchReachability(repoPath string, branch string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("cannot open repo: %w", err)
	}

	headRef, err := r.Head()
	if err != nil {
		return fmt.Errorf("cannot resolve HEAD: %w", err)
	}
	if headRef.Hash().IsZero() {
		return fmt.Errorf("HEAD has no commits")
	}
	if _, err := r.CommitObject(headRef.Hash()); err != nil {
		return fmt.Errorf("HEAD commit is unreachable: %w", err)
	}

	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil
	}

	branchRef, err := r.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return fmt.Errorf("branch %q not found", branch)
	}
	if branchRef.Hash().IsZero() {
		return fmt.Errorf("branch %q has no commits", branch)
	}
	if _, err := r.CommitObject(branchRef.Hash()); err != nil {
		return fmt.Errorf("branch %q commit is unreachable: %w", branch, err)
	}

	return nil
}

// CheckPermissions 检查仓库读取权限（通过读取 .git/HEAD）。
func CheckPermissions(repoPath string) error {
	headPath := filepath.Join(repoPath, ".git", "HEAD")
	f, err := os.Open(headPath)
	if err != nil {
		return fmt.Errorf("cannot read .git/HEAD: %w", err)
	}
	_ = f.Close()
	return nil
}

// CheckPerformance 检查性能预警项。
func CheckPerformance(repos []string) []string {
	warnings := make([]string, 0)

	if len(repos) > repoCountWarnThreshold {
		warnings = append(warnings, fmt.Sprintf("large number of repos (%d) may slow down collection", len(repos)))
	}

	for _, repoPath := range repos {
		size, err := getRepoSize(repoPath)
		if err != nil {
			continue
		}
		if size > gitSizeWarnThreshold {
			warnings = append(warnings, fmt.Sprintf("%s is large (%.1f GB), may be slow", repoPath, float64(size)/float64(1<<30)))
		}
	}

	return warnings
}

func getRepoSize(repoPath string) (int64, error) {
	gitPath := filepath.Join(repoPath, ".git")
	var size int64

	err := filepath.Walk(gitPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return size, nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func BenchmarkCollectCompare_SinglePass(b *testing.B) {
	root := b.TempDir()
	emails := []string{
		"alice@x.com",
		"bob@y.com",
		"carol@z.com",
		"dave@w.com",
		"eve@v.com",
	}

	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
	repos := make([]string, 0, 3)
	for repoIdx := 0; repoIdx < 3; repoIdx++ {
		repoPath := filepath.Join(root, fmt.Sprintf("repo-%d", repoIdx))
		createRepoWithEmailsForBenchmark(b, repoPath, emails, base, 320)
		repos = append(repos, repoPath)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.Local)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, err, allFailed := collectCompareByEmail(repos, emails, start, end, nil)
		if err != nil {
			b.Fatalf("collect compare failed: %v", err)
		}
		if allFailed {
			b.Fatalf("all repositories failed unexpectedly")
		}
		if len(items) != len(emails) {
			b.Fatalf("unexpected items length: got=%d want=%d", len(items), len(emails))
		}
	}
}

func createRepoWithEmailsForBenchmark(tb testing.TB, path string, emails []string, base time.Time, commits int) {
	tb.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("mkdir repo: %v", err)
	}

	repo, err := git.PlainInit(path, false)
	if err != nil {
		tb.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		tb.Fatalf("worktree: %v", err)
	}

	for i := 0; i < commits; i++ {
		fileName := filepath.Join(path, "bench.txt")
		content := []byte(fmt.Sprintf("repo:%s commit:%d\n", path, i))
		if err := os.WriteFile(fileName, content, 0o644); err != nil {
			tb.Fatalf("write file: %v", err)
		}

		if _, err := wt.Add("bench.txt"); err != nil {
			tb.Fatalf("git add: %v", err)
		}

		email := emails[i%len(emails)]
		sig := &object.Signature{
			Name:  "Bench",
			Email: email,
			When:  base.AddDate(0, 0, i%180).Add(time.Duration(i) * time.Minute),
		}
		if _, err := wt.Commit("bench commit", &git.CommitOptions{
			Author:    sig,
			Committer: sig,
		}); err != nil {
			tb.Fatalf("git commit: %v", err)
		}
	}
}

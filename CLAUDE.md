# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o git-visible .    # Build
go run .                     # Run without building
go install                   # Install to $GOPATH/bin
```

## Test & Lint

```bash
go test ./...                # Run all tests
go vet ./...                 # Static analysis
```

## Architecture

CLI tool using Cobra/Viper that aggregates git commit history from multiple local repositories and renders a contribution heatmap.

**Entry flow**: `main.go` → `cmd.Execute()` → Cobra routes to subcommand

**Packages**:
- `cmd/` - Cobra commands (root, show, top, compare, add, list, remove, set, doctor, version), plus shared command init (`common.go`) and compare output formatting (`compare_output.go`)
- `internal/config/` - Config file management (`~/.config/git-visible/config.yaml`), loaded as a singleton via `sync.Once`. Supports email aliases (`NormalizeEmail`)
- `internal/repo/` - Repository storage (`~/.config/git-visible/repos`), directory scanning, and environment diagnostics (`doctor.go`)
- `internal/stats/` - Commit collection (with per-repo HEAD-based caching), heatmap rendering, ranking, and comparison metrics. Supports single-pass multi-email bucketing (`CollectStatsByEmails`)
- `internal/cache/` - JSON-based result cache (`~/.config/git-visible/cache/`), keyed by repo path + HEAD hash + emails + time range + branch

**Key dependencies**: go-git (git operations), cobra (CLI), viper (config), progressbar (progress display)

**Default command**: Running `git-visible` without subcommand executes `show` (see `cmd/root.go:16`)

## Documentation

详细设计文档位于 `docs/` 目录：

| 文档 | 内容 |
|------|------|
| [architecture.md](docs/architecture.md) | 整体架构图、数据流、文件存储 |
| [commands.md](docs/commands.md) | 命令列表、参数设计、Cobra 注册方式 |
| [tech-stack.md](docs/tech-stack.md) | 核心依赖、go-git/Cobra/Viper 使用要点、并发模式 |
| [features.md](docs/features.md) | 功能清单、实现映射、扩展点 |

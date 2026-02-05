# 核心功能

## 功能清单

### 1. 仓库管理
- **扫描添加** (`add`)：递归扫描目录，自动发现 .git 仓库
- **列表查看** (`list`)：展示所有已添加仓库，可验证有效性
- **移除仓库** (`remove`)：单个移除或批量清理无效仓库

### 2. 统计展示
- **热力图** (`show`)：GitHub 风格的贡献热力图
- **仓库排行** (`top`)：按提交数排行的仓库列表
- **对比统计** (`compare`)：多邮箱/时间段贡献对比
- **邮箱过滤**：支持多邮箱筛选
- **分支过滤**：支持指定分支或统计所有分支
- **时间范围**：可配置统计月数，支持 --since/--until
- **多格式输出**：table（默认）、json、csv

### 3. 配置管理
- **持久化配置** (`set`)：默认邮箱、统计月数
- **配置查看**：无参数时显示当前配置

## 功能实现映射

| 功能 | 入口 | 核心实现 |
|------|------|----------|
| 扫描仓库 | `cmd/add.go` | `internal/repo/scanner.go:ScanRepos()` |
| 存储仓库 | `cmd/add.go` | `internal/repo/storage.go:AddRepo()` |
| 加载仓库 | `cmd/show.go` | `internal/repo/storage.go:LoadRepos()` |
| 收集提交 | `cmd/show.go` | `internal/stats/collector.go:CollectStats()` |
| 渲染热力图 | `cmd/show.go` | `internal/stats/renderer.go:RenderHeatmap()` |
| 仓库排行 | `cmd/top.go` | `internal/stats/ranking.go:RankRepositories()` |
| 对比统计 | `cmd/compare.go` | `internal/stats/compare.go:CalculateCompareMetrics()` |
| 读写配置 | `cmd/set.go` | `internal/config/config.go:Load()/Save()` |

## 扩展点

### 添加新命令
1. 在 `cmd/` 创建 `xxx.go`
2. 定义 `xxxCmd` 变量
3. `init()` 中 `rootCmd.AddCommand(xxxCmd)`

### 添加新输出格式
修改 `cmd/show.go` 的 `runShow()` 函数，在 format switch 中添加新 case

### 修改热力图样式
修改 `internal/stats/renderer.go` 的颜色定义和渲染逻辑

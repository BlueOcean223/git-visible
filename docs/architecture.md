# 架构设计

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                      main.go                            │
│                   cmd.Execute()                         │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                      cmd/                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │ root.go  │ │ show.go  │ │  top.go  │ │compare.go│   │
│  │ (默认)   │ │ (热力图) │ │ (排行榜) │ │ (对比)   │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │ add.go   │ │ list.go  │ │remove.go │ │ set.go   │   │
│  │ (扫描)   │ │ (列表)   │ │ (移除)   │ │ (配置)   │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌─────────────────┐        │
│  │version.go│ │common.go │ │compare_output.go│        │
│  │ (版本)   │ │ (公共初始化)│ │ (对比输出格式) │        │
│  └──────────┘ └──────────┘ └─────────────────┘        │
│  ┌──────────┐                                         │
│  │doctor.go │                                         │
│  │ (诊断)   │                                         │
│  └──────────┘                                         │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                    internal/                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │   config/   │  │    repo/    │  │     stats/      │ │
│  │ config.go   │  │ storage.go  │  │ collector.go    │ │
│  │ (配置读写)  │  │ scanner.go  │  │ renderer.go     │ │
│  │ (邮箱别名)  │  │ doctor.go   │  │ ranking.go      │ │
│  │             │  │ (仓库管理)  │  │ compare.go      │ │
│  │             │  │ (环境诊断)  │  │ summary.go      │ │
│  │             │  │             │  │ timerange.go    │ │
│  │             │  │             │  │ (统计/渲染/对比)│ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
│  ┌─────────────┐                                       │
│  │   cache/    │                                       │
│  │ cache.go    │                                       │
│  │ (结果缓存)  │                                       │
│  └─────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

## 数据流

### show 命令（热力图展示）
```
show 命令
    │
    ├─► prepareRun() ──► common.go 公共初始化（配置/仓库/时间范围）
    │
    ▼
repo.LoadRepos() ──► 读取 ~/.config/git-visible/repos
    │
    ▼
stats.CollectStats() ──► 缓存命中时直接返回（跳过 go-git 扫描）
    │                     缓存未命中：并发遍历仓库，go-git 读取提交
    │                     按邮箱/时间过滤，结果写入缓存
    │                     返回 map[time.Time]int
    ▼
stats.RenderHeatmapWithOptions() ──► 渲染热力图到终端
```

### top 命令（仓库排行榜）
```
top 命令
    │
    ├─► prepareRun() ──► common.go 公共初始化（配置/仓库/时间范围）
    │
    ▼
repo.LoadRepos() ──► 读取 ~/.config/git-visible/repos
    │
    ▼
stats.CollectStatsPerRepo() ──► 并发遍历仓库，按仓库分别统计
    │                            返回 map[repo]map[time.Time]int
    ▼
stats.RankRepositories() ──► 按提交数排序，计算百分比
    │
    ▼
输出排行榜（table/json/csv）
```

### compare 命令（对比统计）
```
compare 命令
    │
    ├─► prepareRun() ──► common.go 公共初始化（配置/仓库/时间范围）
    │
    ├─► 邮箱对比模式：stats.CollectStatsByEmails() 单次扫描按邮箱分桶
    │                  stats.CalculateCompareMetrics() 计算指标
    │
    └─► 时间段对比模式：stats.ParsePeriod() 解析时间段
                        按时间段分别收集 stats.CollectStats()
                        stats.CalculatePercentChange() 计算变化
    │
    ▼
输出对比表格（table/json/csv）
```

### add 命令（添加仓库）
```
add 命令
    │
    ▼
repo.ScanRepos() ──► 递归扫描目录，查找 .git
    │                 支持深度限制、排除目录
    ▼
repo.AddRepos() ──► 写入 ~/.config/git-visible/repos
```

## 文件存储

| 文件 | 路径 | 格式 |
|------|------|------|
| 配置 | `~/.config/git-visible/config.yaml` | YAML |
| 仓库列表 | `~/.config/git-visible/repos` | 纯文本，每行一个路径 |
| 统计缓存 | `~/.config/git-visible/cache/` | JSON，按仓库+HEAD hash 分文件 |

### config.yaml 示例
```yaml
email: "your@email.com"
months: 6
aliases:
  - name: "Alice"
    emails:
      - alice@company.com
      - alice@gmail.com
```

### repos 示例
```
/Users/xxx/code/project1
/Users/xxx/code/project2
```

# 命令设计

## 命令列表

| 命令 | 说明 | 实现文件 |
|------|------|----------|
| `git-visible` | 默认执行 show | `cmd/root.go` |
| `git-visible show` | 显示热力图 | `cmd/show.go` |
| `git-visible top` | 仓库贡献排行榜 | `cmd/top.go` |
| `git-visible compare` | 对比邮箱/时间段统计 | `cmd/compare.go` |
| `git-visible add <folder>` | 扫描并添加仓库 | `cmd/add.go` |
| `git-visible list` | 列出已添加仓库 | `cmd/list.go` |
| `git-visible remove <path>` | 移除仓库 | `cmd/remove.go` |
| `git-visible set [key] [value]` | 配置管理 | `cmd/set.go` |
| `git-visible doctor` | 环境诊断 | `cmd/doctor.go` |
| `git-visible version` | 显示版本 | `cmd/version.go` |

## 参数设计

### show
| 参数 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--email` | `-e` | stringArray | 配置值 | 邮箱过滤，可多次指定 |
| `--months` | `-m` | int | 配置值(6) | 统计月数 |
| `--since` | - | string | - | 起始日期 |
| `--until` | - | string | - | 结束日期 |
| `--branch` | `-b` | string | - | 指定分支（默认 HEAD） |
| `--all-branches` | - | bool | false | 统计所有本地分支（去重） |
| `--format` | `-f` | string | table | 输出格式：table/json/csv |
| `--no-legend` | - | bool | false | 隐藏图例 |
| `--no-summary` | - | bool | false | 隐藏摘要信息 |

### top
| 参数 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--number` | `-n` | int | 10 | 显示数量 |
| `--all` | - | bool | false | 显示全部仓库 |
| `--email` | `-e` | stringArray | 配置值 | 邮箱过滤 |
| `--months` | `-m` | int | 配置值 | 统计月数 |
| `--since` | - | string | - | 起始日期 |
| `--until` | - | string | - | 结束日期 |
| `--format` | `-f` | string | table | 输出格式：table/json/csv |

### compare
| 参数 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--email` | `-e` | stringArray | - | 对比的邮箱（至少 2 个） |
| `--period` | - | stringArray | - | 对比的时间段（至少 2 个） |
| `--year` | - | intSlice | - | 对比的年份（--period YYYY 快捷方式） |
| `--format` | `-f` | string | table | 输出格式：table/json/csv |

**时间段格式**：`YYYY`（整年）、`YYYY-H1`/`YYYY-H2`（半年）、`YYYY-Q1`~`YYYY-Q4`（季度）、`YYYY-MM`（单月）

### add
| 参数 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--depth` | `-d` | int | -1 | 递归深度，-1 不限制 |
| `--exclude` | `-x` | stringArray | - | 排除目录，可多次指定 |
| `--dry-run` | - | bool | false | 仅预览不写入 |

**默认排除目录**（无需手动指定）：
`node_modules`、`vendor`、`.venv`、`venv`、`env`、`__pycache__`、`.tox`、`dist`、`build`、`target`、`out`、`.gradle`、`.m2`、`Pods`、`.npm`、`.yarn`、`.pnpm-store`、`bower_components`、`.idea`、`.vscode`、`.cache`、`.tmp`

### list
| 参数 | 类型 | 说明 |
|------|------|------|
| `--verify` | bool | 检查路径有效性 |

### remove
| 参数 | 类型 | 说明 |
|------|------|------|
| `--invalid` | bool | 移除所有无效仓库 |

### doctor
| 参数 | 类型 | 说明 |
|------|------|------|
| - | - | 无参数，执行配置、仓库、分支、权限、性能诊断 |

## Cobra 注册方式

所有子命令在各自文件的 `init()` 中注册到 `rootCmd`：
```go
func init() {
    rootCmd.AddCommand(showCmd)
}
```

默认命令通过 `rootCmd.RunE` 调用 `runShow()` 实现。

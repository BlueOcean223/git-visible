# git-visible CLI 升级方案

## 一、目标架构

```
git-visible/
├── main.go                    # 程序入口，调用 cmd.Execute()
├── cmd/
│   ├── root.go                # 根命令，全局 flag，配置初始化
│   ├── add.go                 # add 子命令
│   ├── show.go                # show 子命令（默认命令）
│   ├── list.go                # list 子命令
│   ├── remove.go              # remove 子命令
│   ├── set.go                 # set 子命令（设置配置项）
│   └── version.go             # version 子命令
├── internal/
│   ├── repo/
│   │   ├── scanner.go         # 仓库扫描
│   │   └── storage.go         # 仓库路径存储
│   ├── stats/
│   │   ├── collector.go       # 提交统计
│   │   └── renderer.go        # 热力图渲染
│   └── config/
│       └── config.go          # 配置管理
├── go.mod
└── README.md
```

---

## 二、命令设计

```
git-visible                      # 等同于 git-visible show
git-visible show [flags]         # 显示热力图
git-visible add <folder>         # 添加仓库
git-visible list                 # 列出已添加的仓库
git-visible remove <path>        # 移除指定仓库
git-visible set <key> <value>    # 设置配置项
git-visible version              # 显示版本
```

### show 命令

| Flag | Short | 说明 |
|------|-------|------|
| `--email` | `-e` | 邮箱（可多次指定） |
| `--months` | `-m` | 统计月数（默认 6） |
| `--format` | `-f` | 输出格式: table/json/csv |

### add 命令

| Flag | Short | 说明 |
|------|-------|------|
| `--depth` | `-d` | 最大递归深度 |
| `--exclude` | `-x` | 排除目录 |
| `--dry-run` | | 仅预览 |

### list 命令

| Flag | 说明 |
|------|------|
| `--verify` | 检查路径是否存在 |

### remove 命令

| Flag | 说明 |
|------|------|
| `--invalid` | 移除所有无效仓库 |

### set 命令

```bash
git-visible set email your@email.com    # 设置默认邮箱
git-visible set months 12               # 设置默认月数
git-visible set                         # 无参数时显示当前配置
```

支持的配置项：
- `email` - 默认邮箱
- `months` - 默认统计月数

---

## 三、配置文件

位置：`~/.config/git-visible/config.yaml`

```yaml
email: "your@email.com"      # 默认邮箱
months: 6                    # 默认时间范围
```

仓库列表存储：`~/.config/git-visible/repos`

---

## 四、新增依赖

```
github.com/spf13/cobra v1.8.0        # CLI 框架
github.com/spf13/viper v1.18.0       # 配置管理
github.com/schollz/progressbar/v3    # 进度条
```

---

## 五、分步实施计划

每个步骤完成后执行 `git commit`。

### Step 1: 初始化项目结构

**任务：**
- 创建目录结构：`cmd/`、`internal/repo/`、`internal/stats/`、`internal/config/`
- 添加 cobra、viper 依赖到 go.mod

**Git commit:** `feat: 初始化 CLI 项目结构和依赖`

---

### Step 2: 实现 root 命令框架

**任务：**
- 创建 `cmd/root.go`：定义根命令，初始化 viper 配置
- 修改 `main.go`：调用 `cmd.Execute()`

**文件：**
- `cmd/root.go`
- `main.go`

**Git commit:** `feat: 实现 root 命令框架`

---

### Step 3: 实现配置管理模块

**任务：**
- 创建 `internal/config/config.go`：
  - 配置文件路径管理
  - 读取/写入配置
  - 默认值设置

**文件：**
- `internal/config/config.go`

**Git commit:** `feat: 实现配置管理模块`

---

### Step 4: 实现 set 命令

**任务：**
- 创建 `cmd/set.go`：
  - `git-visible set` 显示当前配置
  - `git-visible set email xxx` 设置邮箱
  - `git-visible set months N` 设置月数

**文件：**
- `cmd/set.go`

**Git commit:** `feat: 实现 set 命令`

---

### Step 5: 实现仓库存储模块

**任务：**
- 创建 `internal/repo/storage.go`：
  - 读取仓库列表
  - 添加仓库
  - 移除仓库
  - 验证仓库是否存在

**文件：**
- `internal/repo/storage.go`

**Git commit:** `feat: 实现仓库存储模块`

---

### Step 6: 实现仓库扫描模块

**任务：**
- 创建 `internal/repo/scanner.go`：
  - 递归扫描目录查找 .git
  - 支持深度限制
  - 支持排除目录

**文件：**
- `internal/repo/scanner.go`

**Git commit:** `feat: 实现仓库扫描模块`

---

### Step 7: 实现 add 命令

**任务：**
- 创建 `cmd/add.go`：
  - 调用 scanner 扫描目录
  - 调用 storage 存储新仓库
  - 支持 --depth、--exclude、--dry-run 参数

**文件：**
- `cmd/add.go`

**Git commit:** `feat: 实现 add 命令`

---

### Step 8: 实现 list 命令

**任务：**
- 创建 `cmd/list.go`：
  - 列出所有已添加仓库
  - 支持 --verify 参数验证路径

**文件：**
- `cmd/list.go`

**Git commit:** `feat: 实现 list 命令`

---

### Step 9: 实现 remove 命令

**任务：**
- 创建 `cmd/remove.go`：
  - 移除指定仓库
  - 支持 --invalid 移除所有无效仓库

**文件：**
- `cmd/remove.go`

**Git commit:** `feat: 实现 remove 命令`

---

### Step 10: 实现统计收集模块

**任务：**
- 创建 `internal/stats/collector.go`：
  - 读取仓库提交历史
  - 按邮箱过滤
  - 按时间范围过滤
  - 并发处理多仓库

**文件：**
- `internal/stats/collector.go`

**Git commit:** `feat: 实现统计收集模块`

---

### Step 11: 实现热力图渲染模块

**任务：**
- 创建 `internal/stats/renderer.go`：
  - 热力图网格渲染
  - 颜色编码
  - 月份/星期标签

**文件：**
- `internal/stats/renderer.go`

**Git commit:** `feat: 实现热力图渲染模块`

---

### Step 12: 实现 show 命令

**任务：**
- 创建 `cmd/show.go`：
  - 调用 collector 收集统计
  - 调用 renderer 渲染热力图
  - 支持 --email、--months、--format 参数
  - 设为默认命令（无子命令时执行）

**文件：**
- `cmd/show.go`

**Git commit:** `feat: 实现 show 命令`

---

### Step 13: 实现 version 命令

**任务：**
- 创建 `cmd/version.go`：显示版本信息

**文件：**
- `cmd/version.go`

**Git commit:** `feat: 实现 version 命令`

---

### Step 14: 添加进度条支持

**任务：**
- 修改 `internal/stats/collector.go`：添加进度条显示
- 修改 `internal/repo/scanner.go`：添加进度条显示

**Git commit:** `feat: 添加进度条支持`

---

### Step 15: 清理旧代码

**任务：**
- 删除 `scan.go`
- 删除 `stats.go`

**Git commit:** `chore: 清理旧代码`

---

### Step 16: 更新文档

**任务：**
- 更新 `README.md`：新命令用法说明

**Git commit:** `docs: 更新 README`

---

## 六、文件变更清单

### 新增

| 文件 | 步骤 |
|------|------|
| `cmd/root.go` | Step 2 |
| `cmd/set.go` | Step 4 |
| `cmd/add.go` | Step 7 |
| `cmd/list.go` | Step 8 |
| `cmd/remove.go` | Step 9 |
| `cmd/show.go` | Step 12 |
| `cmd/version.go` | Step 13 |
| `internal/config/config.go` | Step 3 |
| `internal/repo/storage.go` | Step 5 |
| `internal/repo/scanner.go` | Step 6 |
| `internal/stats/collector.go` | Step 10 |
| `internal/stats/renderer.go` | Step 11 |

### 修改

| 文件 | 步骤 |
|------|------|
| `main.go` | Step 2 |
| `go.mod` | Step 1 |
| `README.md` | Step 16 |

### 删除

| 文件 | 步骤 |
|------|------|
| `scan.go` | Step 15 |
| `stats.go` | Step 15 |

---

## 七、验证方法

```bash
# Step 4 验证
git-visible set email test@example.com
git-visible set months 12
git-visible set

# Step 7 验证
git-visible add ~/code --dry-run
git-visible add ~/code

# Step 8 验证
git-visible list
git-visible list --verify

# Step 9 验证
git-visible remove /some/path
git-visible remove --invalid

# Step 12 验证
git-visible show
git-visible show -e test@example.com -m 12
git-visible show --format json

# Step 13 验证
git-visible version
```

# git-visible

一个基于 Go 的 CLI 工具：从本地多个 Git 仓库汇总提交记录，并以热力图形式展示贡献情况。

效果演示：

![图片](/resources/img1.png)

## 安装

前提：已安装 Go。

克隆本仓库后，在项目根目录执行：

```bash
go install
```

安装完成后即可在任意目录运行 `git-visible`（可执行文件会被安装到你的 Go bin 目录中，例如 `$(go env GOPATH)/bin`）。

也可以直接运行（不安装）：

```bash
go run .
```

## 命令列表

`git-visible` 默认等同于 `git-visible show`。

- `git-visible show`：显示贡献热力图
- `git-visible top`：显示贡献最多的仓库排行榜
- `git-visible add <folder>`：扫描并添加目录下的 Git 仓库
- `git-visible list`：列出已添加的仓库
- `git-visible remove <path>`：移除指定仓库
- `git-visible remove --invalid`：移除所有无效仓库
- `git-visible set`：显示当前默认配置
- `git-visible set <key> <value>`：设置默认配置（支持 `email` / `months`）
- `git-visible version`：显示版本信息

## 使用示例

添加仓库（扫描目录下所有 Git 仓库并保存）：

```bash
git-visible add ~/code
```

仅预览扫描结果（不保存）：

```bash
git-visible add ~/code --dry-run
```

显示热力图（默认使用配置中的 email/months）：

```bash
git-visible
```

指定邮箱与统计月份：

```bash
git-visible show -e your@email.com -m 12
```

输出 JSON/CSV：

```bash
git-visible show --format json
git-visible show --format csv
```

查看贡献最多的仓库：

```bash
git-visible top
git-visible top -n 5
git-visible top --all -f json
```

查看/设置默认配置：

```bash
git-visible set
git-visible set email your@email.com
git-visible set months 12
```

## 各命令参数（Flags）

### show

- `--email`, `-e`：邮箱过滤（可重复指定）
- `--months`, `-m`：统计月数（不传时使用配置值）
- `--since`：起始日期（`YYYY-MM-DD` / `YYYY-MM` / `2m`/`1w`/`1y`）
- `--until`：结束日期（`YYYY-MM-DD` / `YYYY-MM` / `2m`/`1w`/`1y`）
- `--format`, `-f`：输出格式：`table` / `json` / `csv`（默认 `table`）
- `--no-legend`：隐藏图例（仅 `table`）
- `--no-summary`：隐藏摘要信息（仅 `table`）

### top

- `--number`, `-n`：显示数量（默认 10）
- `--all`：显示全部仓库
- `--email`, `-e`：邮箱过滤（可重复指定）
- `--months`, `-m`：统计月数（不传时使用配置值）
- `--since`：起始日期（`YYYY-MM-DD` / `YYYY-MM` / `2m`/`1w`/`1y`）
- `--until`：结束日期（`YYYY-MM-DD` / `YYYY-MM` / `2m`/`1w`/`1y`）
- `--format`, `-f`：输出格式：`table` / `json` / `csv`（默认 `table`）

### add

- `--depth`, `-d`：最大递归深度（`-1` 表示不限制，默认 `-1`）
- `--exclude`, `-x`：排除目录（可重复指定；支持相对路径/绝对路径）
- `--dry-run`：仅预览，不写入仓库列表

> 默认排除目录：`node_modules`、`vendor`、`.venv`、`dist`、`build`、`target`、`.gradle`、`Pods` 等常见依赖/构建目录，无需手动指定。

### list

- `--verify`：检查仓库路径是否有效（会标注 `(invalid)`）

### remove

- `--invalid`：移除所有无效仓库（使用时不需要传 `path` 参数）

## 配置与数据文件

配置文件位置：`~/.config/git-visible/config.yaml`

示例：

```yaml
email: "your@email.com"
months: 6
```

仓库列表存储：`~/.config/git-visible/repos`

## 帮助

使用 `--help` 查看完整帮助：

```bash
git-visible --help
git-visible show --help
git-visible add --help
```

本项目参考自 https://flaviocopes.com/go-git-contributions/

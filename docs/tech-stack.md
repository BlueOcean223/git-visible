# 技术栈

## 核心依赖

| 库 | 版本 | 用途 |
|----|------|------|
| `github.com/spf13/cobra` | v1.8.0 | CLI 框架 |
| `github.com/spf13/viper` | v1.18.0 | 配置管理 |
| `github.com/go-git/go-git/v5` | v5.16.2 | Git 操作（纯 Go 实现） |
| `github.com/schollz/progressbar/v3` | v3.19.0 | 进度条显示 |
| `golang.org/x/term` | v0.31.0 | 终端检测（仅终端下显示进度条） |
| `github.com/stretchr/testify` | v1.10.0 | 测试断言库 |

## go-git 使用要点

### 打开仓库
```go
repo, err := git.PlainOpen(repoPath)
```

### 读取提交历史
```go
ref, _ := repo.Head()
iterator, _ := repo.Log(&git.LogOptions{From: ref.Hash()})
iterator.ForEach(func(c *object.Commit) error {
    // c.Author.Email - 提交者邮箱
    // c.Author.When  - 提交时间
    return nil
})
```

### 提前终止遍历
```go
return storer.ErrStop  // 返回此错误可提前终止
```

## Cobra/Viper 使用要点

### 定义命令
```go
var cmd = &cobra.Command{
    Use:   "name",
    Short: "description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 实现
    },
}
```

### 定义 Flag
```go
cmd.Flags().StringP("email", "e", "", "description")      // 局部 flag
cmd.PersistentFlags().StringP("config", "c", "", "desc")  // 继承到子命令
```

### Viper 读取配置
```go
cfg, err := config.Load() // 内部通过 sync.Once 保证单例加载
if err != nil {
    return err
}
email := cfg.Email
months := cfg.Months
```
命令层通过 `internal/config.Load()` 读取配置，不直接调用 viper。

## 并发模式

`stats/collector.go` 的并发逻辑集中在 `collectCommonGeneric[T]()`；`CollectStats()`、`CollectStatsPerRepo()` 和 `CollectStatsByEmails()` 通过泛型参数复用同一套并发处理：
```go
var wg sync.WaitGroup
var mu sync.Mutex  // 保护共享 map

// 信号量限制并发数为 CPU 核心数
sem := make(chan struct{}, runtime.NumCPU())

for _, repo := range repos {
    wg.Add(1)
    go func() {
        sem <- struct{}{}        // 获取信号量
        defer func() { <-sem }() // 释放信号量
        defer wg.Done()
        // 收集单个仓库
        mu.Lock()
        // 合并结果
        mu.Unlock()
    }()
}
wg.Wait()
```

## 热力图渲染

`stats/renderer.go` 使用 ANSI 转义码输出彩色方块：
- 主入口是 `RenderHeatmapWithOptions(stats, HeatmapOptions{...})`，通过 `HeatmapOptions` 控制时间范围/图例/摘要输出
- 按周列排列（每列 7 天）
- 根据提交数量映射颜色深浅
- 输出月份和星期标签

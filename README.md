# git-visible
这是一款基于go编写的，让你的git提交记录清晰可见的CLI工具。它让你的贡献记录清晰可见。让你一览过去的勤奋贡献

效果演示

![图片](/resources/img1.png)

# 运行
添加git仓库
```bash
go run . -add "D:/code"
```

打印贡献信息
```bash
go run . -email "your@email.com"
```
当然，你也可以将`main.go`中的`email`直接修改为你的邮箱，这样就可以省去指明邮箱，直接运行
```bash
go run .
```

# 全局运行
>在实现全局运行功能前，请确保你的环境变量中已经正确配置了`GOPATH`

保存可执行文件至`$GOPATH/bin`
```bash
go install
```
接下来你就可以在任何地方运行它了


本项目参考自 https://flaviocopes.com/go-git-contributions/
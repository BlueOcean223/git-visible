// git-visible 是一个从本地多个 Git 仓库汇总提交记录，并以热力图形式展示贡献情况的工具。
package main

import (
	"git-visible/cmd"
)

// main 是程序的入口函数，负责启动 CLI 命令执行。
func main() {
	cmd.Execute()
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// getDotFilePath 获取存储git仓库路径的文件地址
func getDotFilePath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	// 使用 filepath.Join 提升多平台兼容性
	filePath := filepath.Join(usr.HomeDir, ".goGitLocalStats")

	// 如果文件不存在，则创建一个空的文件
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	return filePath
}

// parseFileLinesToSlice 将文件内容按行切割为string切片
func parseFileLinesToSlice(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return strings.Split(strings.TrimSpace(string(content)), "\n")
}

// joinSlices 将新增的文件路径添加到已有的 slice 中，已有文件路径的不重复添加
func joinSlices(new []string, existing []string) []string {
	hash := make(map[string]bool, len(existing))
	for _, path := range existing {
		hash[path] = true
	}

	for _, i := range new {
		if !hash[i] {
			existing = append(existing, i)
			hash[i] = true
		}
	}
	return existing
}

// dumpStringsSliceToFile 将 slice 内容写入文件
func dumpStringsSliceToFile(repos []string, filePath string) {
	content := strings.Join(repos, "\n")
	err := os.WriteFile(filePath, []byte(content), 0755)
	if err != nil {
		panic(err)
	}
}

// addNewSliceElementsToFile 添加新的 slice 元素到文件
func addNewSliceElementsToFile(filePath string, newRepos []string) {
	existingRepos := parseFileLinesToSlice(filePath)
	repos := joinSlices(newRepos, existingRepos)
	dumpStringsSliceToFile(repos, filePath)
}

// scan 扫描给定文件夹中的 git 仓库
func scan(folder string) {
	fmt.Printf("开始扫描git仓库，目标路径:%s \n\n", folder)

	repositories := scanGitFolders(folder)
	if len(repositories) == 0 {
		fmt.Println("没有找到git仓库")
		return
	}

	filePath := getDotFilePath()
	addNewSliceElementsToFile(filePath, repositories)
	fmt.Printf("\n添加git仓库成功\n")
}

// scanGitFolders 返回包含 .git 文件夹的父文件夹路径
func scanGitFolders(folder string) []string {
	var folders []string
	// 遍历文件夹
	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" {
				// 获取 .git 文件夹的父文件夹路径
				repoPath := filepath.Dir(path)
				fmt.Println(repoPath)
				folders = append(folders, repoPath)
				// 跳过当前文件夹，即不遍历其子目录
				return filepath.SkipDir
			}

			if name == "node_modules" {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return folders
}

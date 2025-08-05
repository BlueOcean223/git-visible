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

// getNewRepos 获取新增的仓库路径
func getNewRepos(new, existing []string) []string {
	hash := make(map[string]bool, len(existing))
	for _, path := range existing {
		hash[path] = true
	}

	var newRepos []string
	for _, i := range new {
		if !hash[i] {
			newRepos = append(newRepos, i)
			hash[i] = true
		}
	}
	return newRepos
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
	// 向用户展示已有的仓库路径
	if len(existingRepos) > 0 {
		fmt.Println("已有的git仓库路径:")
		for _, repo := range existingRepos {
			fmt.Println(repo)
		}
		fmt.Printf("\n")
	}
	// 获取新增仓库路径，并展示
	newRepos = getNewRepos(newRepos, existingRepos)
	if len(newRepos) > 0 {
		fmt.Println("新增的git仓库路径:")
		for _, repo := range newRepos {
			fmt.Println(repo)
		}
	} else {
		fmt.Println("没有新增的git仓库路径")
	}
	// 合并仓库路径
	repos := append(existingRepos, newRepos...)
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

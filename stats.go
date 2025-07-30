package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

const outOfRange = 99999

var daysInLastSixMonths = getSixMonthsAgoSunday()

// 打印格子颜色
const (
	colorReset   = "\033[0m"
	colorDefault = "\033[0;37;30m"
	colorLow     = "\033[1;30;47m"
	colorMedium  = "\033[1;30;43m"
	colorHigh    = "\033[1;30;42m"
	colorToday   = "\033[1;37;45m"
)

// getSixMonthsAgoSunday 获取六个月前的星期天
func getSixMonthsAgoSunday() int {
	now := getBeginningOfDay(time.Now())
	sixMonthsAgo := now.AddDate(0, -6, 0)
	days := int(now.Sub(sixMonthsAgo).Hours() / 24)
	weekday := sixMonthsAgo.Weekday()
	if weekday != time.Sunday {
		if weekday <= time.Wednesday {
			days += int(weekday)
		} else {
			days -= int(time.Saturday-weekday) + 1
		}
	}

	return days
}

// stats 计算并打印git提交信息
func stats(email string) {
	// 获取用户过去半年的提交信息
	commits := processRepositories(email)
	// 打印提交信息
	printCommits(commits)
}

// getBeginningOfDay 获取给定日期的零点时间
func getBeginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// countDaysSinceDate 计算给定日期至今的天数，上限为183（六个月）
func countDaysSinceDate(date time.Time) int {
	now := getBeginningOfDay(time.Now())
	date = getBeginningOfDay(date)
	days := int(now.Sub(date).Hours() / 24)
	// 超出期望时间范围
	if days > daysInLastSixMonths {
		return outOfRange
	}

	return days
}

// fillCommits 遍历指定仓库的提交历史，统计用户的提交次数
func fillCommits(email string, path string) map[int]int {
	// 根据指定路径打开git仓库
	repo, err := git.PlainOpen(path)
	if err != nil {
		panic(err)
	}
	// 获取HEAD引用
	ref, err := repo.Head()
	if err != nil {
		panic(err)
	}
	// 获取仓库提交历史
	iterator, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		panic(err)
	}

	commits := make(map[int]int, daysInLastSixMonths)

	// 遍历仓库提交历史
	err = iterator.ForEach(func(c *object.Commit) error {
		daysAgo := countDaysSinceDate(c.Author.When)
		// 过滤非指定邮箱的提交记录
		if c.Author.Email != email {
			return nil
		}
		// 如果提交记录未超出六个月，则记录
		if daysAgo != outOfRange {
			commits[daysAgo]++
		} else {
			// 超出六个月则停止迭代
			return storer.ErrStop
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return commits
}

// processRepositories 获取用户的git提交信息
func processRepositories(email string) map[int]int {
	// 获取存储仓库路径的文件路径
	filePath := getDotFilePath()
	// 切分各仓库路径
	repos := parseFileLinesToSlice(filePath)
	daysInMap := daysInLastSixMonths

	commits := make(map[int]int, daysInMap)
	for i := daysInMap; i >= 0; i-- {
		commits[i] = 0
	}

	var wa sync.WaitGroup
	var mu sync.Mutex

	// 根据仓库填充提交信息
	for _, path := range repos {
		// 检查路径指向的文件夹是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Println("路径不存在：" + path)
			continue
		}
		wa.Add(1)
		go func(path string) {
			defer wa.Done()
			newCommits := fillCommits(email, path)

			mu.Lock()
			for k, v := range newCommits {
				commits[k] += v
			}
			mu.Unlock()
		}(path)
	}
	wa.Wait()

	return commits
}

// printCell 根据提交量显示单元格，特殊显示"今天"
func printCell(val int, today bool) {
	escape := colorDefault
	// 根据提交量显示不同颜色
	switch {
	case val > 0 && val < 5:
		escape = colorLow
	case val >= 5 && val < 10:
		escape = colorMedium
	case val >= 10:
		escape = colorHigh
	}

	if today {
		escape = colorToday
	}

	if val == 0 {
		fmt.Print(escape + "  - " + colorReset)
		return
	}

	str := "  %d "
	switch {
	case val >= 10 && val < 100:
		str = " %d "
	case val >= 100:
		str = "%d "
	}

	fmt.Printf(escape+str+colorReset, val)
}

// printCommits 打印提交信息
func printCommits(commits map[int]int) {
	// 打印月份信息
	printMonths()

	for i := 0; i < 7; i++ {
		// 打印星期提示
		printDayCol(i)

		for j := daysInLastSixMonths - i; j >= 0; j -= 7 {
			// 特殊处理今天
			if j == 0 {
				printCell(commits[j], true)
			} else {
				printCell(commits[j], false)
			}
		}
		fmt.Printf("\n")
	}

}

// printMonths 在第一行打印月份信息
func printMonths() {
	now := getBeginningOfDay(time.Now())
	startDate := now.AddDate(0, 0, -daysInLastSixMonths)

	fmt.Printf("       ")

	// 计算每个位置对应的日期和月份
	currentDate := startDate
	lastPrintedMonth := -1

	// 遍历所有列
	for currentDate.Before(now) || currentDate.Equal(now) {
		currentMonth := int(currentDate.Month())

		// 如果是新月份的第一次出现，打印月份名称
		if currentMonth != lastPrintedMonth {
			fmt.Printf("%s", currentDate.Month().String()[:3])
			lastPrintedMonth = currentMonth
		} else {
			// 如果是同一个月，打印空格占位
			fmt.Printf("    ")
		}

		// 移动到下一列
		currentDate = currentDate.AddDate(0, 0, 7)
	}

	fmt.Printf("\n")
}

// printDayCol 打印日期信息
func printDayCol(day int) {
	out := "     "
	switch day {
	case 1:
		out = " Mon "
	case 3:
		out = " Wed "
	case 6:
		out = " Sat "
	}

	fmt.Print(out)
}

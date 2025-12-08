package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type DecisionRecord struct {
	CycleNumber int            `json:"cycle_number"`
	Decisions   []DecisionAction `json:"decisions"`
}

type DecisionAction struct {
	Action  string `json:"action"`
	Symbol  string `json:"symbol"`
	Success bool   `json:"success"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run analyze_trading_cycles.go <log_dir> <start_cycle> <end_cycle>")
		fmt.Println("Example: go run analyze_trading_cycles.go decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728 400 1892")
		os.Exit(1)
	}

	logDir := os.Args[1]
	startCycle, _ := strconv.Atoi(os.Args[2])
	endCycle, _ := strconv.Atoi(os.Args[3])

	// 读取所有决策日志文件
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	cyclesWithTradesMap := make(map[int]bool) // 使用map去重
	cyclePattern := regexp.MustCompile(`cycle(\d+)\.json$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 从文件名提取cycle number
		matches := cyclePattern.FindStringSubmatch(file.Name())
		if len(matches) < 2 {
			continue
		}

		cycleNum, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		// 检查cycle是否在范围内
		if cycleNum < startCycle || cycleNum > endCycle {
			continue
		}

		// 读取并解析JSON文件
		filePath := filepath.Join(logDir, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\n", file.Name(), err)
			continue
		}

		var record DecisionRecord
		if err := json.Unmarshal(data, &record); err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\n", file.Name(), err)
			continue
		}

		// 检查是否有成功的交易行为
		hasTrade := false
		for _, decision := range record.Decisions {
			if decision.Success && (decision.Action == "open_long" || decision.Action == "open_short" ||
				decision.Action == "close_long" || decision.Action == "close_short") {
				hasTrade = true
				break
			}
		}

		if hasTrade {
			cyclesWithTradesMap[cycleNum] = true
		}
	}

	// 转换为slice并排序
	var cyclesWithTrades []int
	for cycle := range cyclesWithTradesMap {
		cyclesWithTrades = append(cyclesWithTrades, cycle)
	}
	sort.Ints(cyclesWithTrades)

	// 输出结果
	fmt.Printf("\n=== Cycle %d 到 %d 中有交易行为的Cycle ===\n\n", startCycle, endCycle)
	fmt.Printf("总数: %d\n\n", len(cyclesWithTrades))
	
	if len(cyclesWithTrades) > 0 {
		fmt.Println("Cycle列表:")
		for i, cycle := range cyclesWithTrades {
			fmt.Printf("%d", cycle)
			if i < len(cyclesWithTrades)-1 {
				fmt.Print(", ")
			}
			if (i+1)%20 == 0 {
				fmt.Println()
			}
		}
		fmt.Println("\n")
		
		// 保存到文件
		outputFile := fmt.Sprintf("trading_cycles_%d_%d.txt", startCycle, endCycle)
		file, err := os.Create(outputFile)
		if err == nil {
			defer file.Close()
			fmt.Fprintf(file, "Cycle %d 到 %d 中有交易行为的Cycle\n", startCycle, endCycle)
			fmt.Fprintf(file, "总数: %d\n\n", len(cyclesWithTrades))
			fmt.Fprintf(file, "Cycle列表:\n")
			for i, cycle := range cyclesWithTrades {
				fmt.Fprintf(file, "%d", cycle)
				if i < len(cyclesWithTrades)-1 {
					fmt.Fprintf(file, ", ")
				}
				if (i+1)%20 == 0 {
					fmt.Fprintf(file, "\n")
				}
			}
			fmt.Fprintf(file, "\n")
			fmt.Printf("结果已保存到: %s\n", outputFile)
		}
	} else {
		fmt.Println("未找到有交易行为的cycle")
	}
}


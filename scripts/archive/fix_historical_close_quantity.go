package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DecisionRecord 决策记录（与logger包中的结构一致）
type DecisionRecord struct {
	Timestamp   time.Time          `json:"timestamp"`
	CycleNumber int                `json:"cycle_number"`
	Decisions   []DecisionAction   `json:"decisions"`
	Positions   []PositionSnapshot `json:"positions"`
}

// DecisionAction 决策动作
type DecisionAction struct {
	Action    string    `json:"action"`
	Symbol    string    `json:"symbol"`
	Quantity  float64   `json:"quantity"`
	Leverage  int       `json:"leverage"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// PositionSnapshot 持仓快照
type PositionSnapshot struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	PositionAmt      float64 `json:"position_amt"`
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	UnrealizedProfit float64 `json:"unrealized_profit"`
	Leverage         float64 `json:"leverage"`
}

// FileInfo 文件信息
type FileInfo struct {
	Path      string
	Timestamp time.Time
	Filename  string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run fix_historical_close_quantity.go <日志目录>")
		fmt.Println("示例: go run fix_historical_close_quantity.go decision_logs/hyperliquid_a5e99c3c-bef8-4b5b-a19a-01328b593128_deepseek_1762195291")
		os.Exit(1)
	}

	logDir := os.Args[1]
	fmt.Printf("🔧 开始修复历史数据：%s\n\n", logDir)

	// 读取所有日志文件并按时间排序
	files, err := getSortedFiles(logDir)
	if err != nil {
		fmt.Printf("❌ 读取目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📁 找到 %d 个日志文件\n\n", len(files))

	totalFixed := 0
	totalFiles := 0
	totalCloseActions := 0

	// 用于记录所有开仓记录（按时间顺序，FIFO匹配）
	openPositionsByTime := make(map[string][]map[string]interface{})

	// 按时间顺序处理文件
	fmt.Println("🔧 开始修复平仓记录的quantity...")
	for _, fileInfo := range files {
		filePath := fileInfo.Path
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		var record DecisionRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		// 收集开仓记录（按时间顺序）
		for _, action := range record.Decisions {
			if !action.Success {
				continue
			}

			symbol := action.Symbol
			side := ""
			if action.Action == "open_long" || action.Action == "close_long" {
				side = "long"
			} else if action.Action == "open_short" || action.Action == "close_short" {
				side = "short"
			}

			posKey := symbol + "_" + side

			switch action.Action {
			case "open_long", "open_short":
				if openPositionsByTime[posKey] == nil {
					openPositionsByTime[posKey] = []map[string]interface{}{}
				}
				openPositionsByTime[posKey] = append(openPositionsByTime[posKey], map[string]interface{}{
					"side":      side,
					"quantity":  action.Quantity,
					"leverage":  action.Leverage,
					"timestamp": action.Timestamp,
				})
			case "close_long", "close_short":
				// 移除最早的未平仓记录（FIFO）
				if len(openPositionsByTime[posKey]) > 0 {
					openPositionsByTime[posKey] = openPositionsByTime[posKey][1:]
				}
			}
		}

		// 检查并修复平仓记录
		modified := false
		for i := range record.Decisions {
			action := &record.Decisions[i]
			if !action.Success {
				continue
			}

			if action.Action != "close_long" && action.Action != "close_short" {
				continue
			}

			if action.Quantity > 0 {
				continue // 已经有quantity，跳过
			}

			totalCloseActions++
			symbol := action.Symbol
			side := ""
			if action.Action == "close_long" {
				side = "long"
			} else if action.Action == "close_short" {
				side = "short"
			}

			// 方案1：从positions快照中恢复
			quantity := 0.0
			leverage := 0

			for _, pos := range record.Positions {
				if pos.Symbol == symbol && pos.Side == side {
					quantity = pos.PositionAmt
					leverage = int(pos.Leverage)
					break
				}
			}

			// 方案2：如果positions中没有，从开仓记录中恢复
			if quantity == 0 {
				posKey := symbol + "_" + side
				if len(openPositionsByTime[posKey]) > 0 {
					openPos := openPositionsByTime[posKey][0]
					quantity = openPos["quantity"].(float64)
					leverage = openPos["leverage"].(int)
				}
			}

			// 如果找到了quantity，更新记录
			if quantity > 0 {
				action.Quantity = quantity
				if leverage > 0 {
					action.Leverage = leverage
				}
				modified = true
				totalFixed++
				fmt.Printf("  ✅ %s: %s %s quantity=%.8f leverage=%d\n",
					fileInfo.Filename, action.Action, symbol, quantity, leverage)

				// 从开仓记录中移除（FIFO匹配）
				posKey := symbol + "_" + side
				if len(openPositionsByTime[posKey]) > 0 {
					openPositionsByTime[posKey] = openPositionsByTime[posKey][1:]
				}
			} else {
				fmt.Printf("  ⚠️  %s: %s %s 无法恢复quantity（positions中没有，也没有找到开仓记录）\n",
					fileInfo.Filename, action.Action, symbol)
			}
		}

		// 如果修改了记录，保存文件
		if modified {
			// 重新读取完整文件（保留所有字段）
			var fullRecord map[string]interface{}
			if err := json.Unmarshal(data, &fullRecord); err != nil {
				fmt.Printf("  ⚠️  无法读取完整记录 %s: %v\n", fileInfo.Filename, err)
				continue
			}

			// 更新decisions字段
			fullRecord["decisions"] = record.Decisions

			// 保存文件
			outputData, err := json.MarshalIndent(fullRecord, "", "  ")
			if err != nil {
				fmt.Printf("  ⚠️  序列化失败 %s: %v\n", fileInfo.Filename, err)
				continue
			}

			if err := ioutil.WriteFile(filePath, outputData, 0644); err != nil {
				fmt.Printf("  ⚠️  写入失败 %s: %v\n", fileInfo.Filename, err)
				continue
			}

			totalFiles++
		}
	}

	fmt.Printf("\n✅ 修复完成！\n")
	fmt.Printf("   - 处理的平仓记录: %d\n", totalCloseActions)
	fmt.Printf("   - 成功修复: %d\n", totalFixed)
	fmt.Printf("   - 修改的文件数: %d\n", totalFiles)
}

// getSortedFiles 获取所有文件并按时间戳排序
func getSortedFiles(logDir string) ([]FileInfo, error) {
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var fileInfos []FileInfo
	timePattern := regexp.MustCompile(`decision_(\d{8})_(\d{6})_cycle\d+\.json`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasPrefix(file.Name(), "decision_") || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 从文件名提取时间戳: decision_YYYYMMDD_HHMMSS_cycleN.json
		matches := timePattern.FindStringSubmatch(file.Name())
		if len(matches) != 3 {
			// 尝试从文件修改时间获取
			timestamp := file.ModTime()
			fileInfos = append(fileInfos, FileInfo{
				Path:      filepath.Join(logDir, file.Name()),
				Timestamp: timestamp,
				Filename:  file.Name(),
			})
			continue
		}

		dateStr := matches[1]
		timeStr := matches[2]

		// 解析时间戳: YYYYMMDD_HHMMSS
		timestamp, err := time.Parse("20060102_150405", dateStr+"_"+timeStr)
		if err != nil {
			// 如果解析失败，使用文件修改时间
			timestamp = file.ModTime()
		}

		fileInfos = append(fileInfos, FileInfo{
			Path:      filepath.Join(logDir, file.Name()),
			Timestamp: timestamp,
			Filename:  file.Name(),
		})
	}

	// 按时间戳排序（从旧到新）
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Timestamp.Before(fileInfos[j].Timestamp)
	})

	return fileInfos, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DecisionRecord 决策记录（简化版，只包含需要验证的字段）
type DecisionRecord struct {
	Timestamp time.Time       `json:"timestamp"`
	Decisions []DecisionAction `json:"decisions"`
	Success   bool            `json:"success"`
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
	Error     string    `json:"error"`
}

// TradeOutcome 单笔交易结果
type TradeOutcome struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Quantity      float64   `json:"quantity"`
	Leverage      int       `json:"leverage"`
	OpenPrice     float64   `json:"open_price"`
	ClosePrice    float64   `json:"close_price"`
	PositionValue float64   `json:"position_value"`
	MarginUsed    float64   `json:"margin_used"`
	PnL           float64   `json:"pn_l"`
	PnLPct        float64   `json:"pn_l_pct"`
	OpenTime      time.Time `json:"open_time"`
	CloseTime     time.Time `json:"close_time"`
}

// OpenPosition 未平仓的持仓
type OpenPosition struct {
	Symbol    string
	Side      string
	Quantity  float64
	Leverage  int
	OpenPrice float64
	OpenTime  time.Time
}

// SymbolPerformance 币种表现统计
type SymbolPerformance struct {
	Symbol        string  `json:"symbol"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pn_l"`
	AvgPnL        float64 `json:"avg_pn_l"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	TotalCycles       int                           `json:"total_cycles"`
	CyclesWithTrades  int                           `json:"cycles_with_trades"`
	TotalTrades       int                           `json:"total_trades"`
	WinningTrades     int                           `json:"winning_trades"`
	LosingTrades      int                           `json:"losing_trades"`
	WinRate           float64                       `json:"win_rate"`
	AvgWin            float64                       `json:"avg_win"`
	AvgLoss           float64                       `json:"avg_loss"`
	ProfitFactor      float64                       `json:"profit_factor"`
	CompletedTrades   []TradeOutcome                `json:"completed_trades"`
	SymbolStats       map[string]*SymbolPerformance `json:"symbol_stats"`
	BestSymbol        string                        `json:"best_symbol"`
	WorstSymbol       string                        `json:"worst_symbol"`
}

// FileInfo 文件信息（包含时间戳用于排序）
type FileInfo struct {
	Path      string
	Timestamp time.Time
	Filename  string
}

func main() {
	// 日志目录路径
	logDir := "decision_logs/hyperliquid_deepseek"
	if len(os.Args) > 1 {
		logDir = os.Args[1]
	}

	fmt.Printf("🔍 开始验证 performance 分析数据...\n")
	fmt.Printf("📁 日志目录: %s\n\n", logDir)

	// 1. 读取所有文件并按时间戳排序
	files, err := getSortedFiles(logDir)
	if err != nil {
		fmt.Printf("❌ 读取文件失败: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("⚠️  未找到任何日志文件\n")
		os.Exit(1)
	}

	// 2. 分析所有文件（不限制数量）
	fmt.Printf("📊 分析所有周期 (共 %d 个文件)\n", len(files))

	// 3. 解析所有文件并提取交易记录
	var allRecords []*DecisionRecord
	cyclesWithTrades := 0

	for _, fileInfo := range files {
		record, err := parseRecord(fileInfo.Path)
		if err != nil {
			fmt.Printf("⚠️  解析文件失败 %s: %v\n", fileInfo.Filename, err)
			continue
		}

		// 检查是否包含成交记录（success=true且action是交易相关的）
		hasTrades := false
		for _, decision := range record.Decisions {
			if decision.Success && isTradeAction(decision.Action) {
				hasTrades = true
				break
			}
		}

		if hasTrades {
			cyclesWithTrades++
		}

		allRecords = append(allRecords, record)
	}

	fmt.Printf("✅ 找到 %d 个包含交易记录的周期\n\n", cyclesWithTrades)

	// 4. 匹配交易对（开仓-平仓）
	completedTrades := matchTrades(allRecords)

	fmt.Printf("📈 匹配到 %d 笔完整交易 (开仓→平仓)\n\n", len(completedTrades))

	// 5. 计算统计信息
	result := calculateStatistics(completedTrades, cyclesWithTrades, len(allRecords))

	// 6. 输出结果（JSON格式）
	outputJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		os.Exit(1)
	}

	// 输出到文件
	outputFile := "validation_result.json"
	if err := ioutil.WriteFile(outputFile, outputJSON, 0644); err != nil {
		fmt.Printf("❌ 写入文件失败: %v\n", err)
		os.Exit(1)
	}

	// 同时输出到控制台
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("📊 验证结果 (已保存到 validation_result.json)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("\n📈 总体统计:\n")
	fmt.Printf("  • 总周期数: %d\n", result.TotalCycles)
	fmt.Printf("  • 包含交易的周期: %d\n", result.CyclesWithTrades)
	fmt.Printf("  • 总交易数: %d\n", result.TotalTrades)
	fmt.Printf("  • 盈利交易: %d\n", result.WinningTrades)
	fmt.Printf("  • 亏损交易: %d\n", result.LosingTrades)
	fmt.Printf("  • 胜率: %.2f%%\n", result.WinRate)
	fmt.Printf("  • 平均盈利: %.2f USDT\n", result.AvgWin)
	fmt.Printf("  • 平均亏损: %.2f USDT\n", result.AvgLoss)
	fmt.Printf("  • 盈亏比: %.2f\n", result.ProfitFactor)

	if result.BestSymbol != "" {
		if stats, ok := result.SymbolStats[result.BestSymbol]; ok {
			fmt.Printf("\n🏆 最佳币种: %s (总盈亏: %.2f USDT, 胜率: %.1f%%, 交易数: %d)\n",
				result.BestSymbol, stats.TotalPnL, stats.WinRate, stats.TotalTrades)
		}
	}

	if result.WorstSymbol != "" {
		if stats, ok := result.SymbolStats[result.WorstSymbol]; ok {
			fmt.Printf("⚠️  最差币种: %s (总盈亏: %.2f USDT, 胜率: %.1f%%, 交易数: %d)\n",
				result.WorstSymbol, stats.TotalPnL, stats.WinRate, stats.TotalTrades)
		}
	}

	fmt.Printf("\n📋 币种统计 (共 %d 个币种):\n", len(result.SymbolStats))
	// 按总盈亏排序显示前10个
	type SymbolRank struct {
		Symbol string
		PnL    float64
	}
	var ranks []SymbolRank
	for symbol, stats := range result.SymbolStats {
		ranks = append(ranks, SymbolRank{Symbol: symbol, PnL: stats.TotalPnL})
	}
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].PnL > ranks[j].PnL
	})

	displayCount := 10
	if len(ranks) < displayCount {
		displayCount = len(ranks)
	}
	for i := 0; i < displayCount; i++ {
		stats := result.SymbolStats[ranks[i].Symbol]
		fmt.Printf("  %d. %s: 总盈亏=%.2f, 交易数=%d, 胜率=%.1f%%\n",
			i+1, ranks[i].Symbol, stats.TotalPnL, stats.TotalTrades, stats.WinRate)
	}

	fmt.Printf("\n✅ 验证完成！结果已保存到 %s\n", outputFile)
	fmt.Println("💡 此结果可用于与前端 AI Learning 页面数据进行比对")
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

// parseRecord 解析决策记录文件
func parseRecord(filePath string) (*DecisionRecord, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var record DecisionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return &record, nil
}

// isTradeAction 判断是否是交易相关的action
func isTradeAction(action string) bool {
	return action == "open_long" || action == "open_short" ||
		action == "close_long" || action == "close_short"
}

// matchTrades 匹配交易对（开仓-平仓，FIFO）
func matchTrades(records []*DecisionRecord) []TradeOutcome {
	// 追踪未平仓的持仓：symbol_side -> []OpenPosition
	openPositions := make(map[string][]OpenPosition)
	var completedTrades []TradeOutcome

	// 按时间顺序处理所有记录
	for _, record := range records {
		for _, decision := range record.Decisions {
			if !decision.Success {
				continue
			}

			symbol := decision.Symbol
			var side string
			var isOpen, isClose bool

			switch decision.Action {
			case "open_long":
				side = "long"
				isOpen = true
			case "open_short":
				side = "short"
				isOpen = true
			case "close_long":
				side = "long"
				isClose = true
			case "close_short":
				side = "short"
				isClose = true
			default:
				continue
			}

			posKey := symbol + "_" + side

			if isOpen {
				// 开仓：添加到列表
				openPositions[posKey] = append(openPositions[posKey], OpenPosition{
					Symbol:    symbol,
					Side:      side,
					Quantity:   decision.Quantity,
					Leverage:   decision.Leverage,
					OpenPrice:  decision.Price,
					OpenTime:   decision.Timestamp,
				})
			} else if isClose {
				// 平仓：FIFO匹配
				remainingQty := decision.Quantity
				if remainingQty == 0 {
					// quantity为0表示全平仓，匹配所有未平仓的
					for len(openPositions[posKey]) > 0 {
						openPos := openPositions[posKey][0]
						openPositions[posKey] = openPositions[posKey][1:]

						trade := matchSingleTrade(openPos, decision)
						completedTrades = append(completedTrades, trade)
					}
					continue
				}

				// FIFO匹配（处理部分平仓）
				newList := []OpenPosition{}
				for _, openPos := range openPositions[posKey] {
					if remainingQty <= 0 {
						// 已匹配完成，保留剩余开仓
						newList = append(newList, openPos)
						continue
					}

					if remainingQty >= openPos.Quantity {
						// 完全匹配这笔开仓
						remainingQty -= openPos.Quantity
						trade := matchSingleTrade(openPos, decision)
						completedTrades = append(completedTrades, trade)
					} else {
						// 部分匹配：只匹配remainingQty部分
						// 创建部分匹配的交易（创建新副本避免修改原值）
						partialOpenPos := OpenPosition{
							Symbol:    openPos.Symbol,
							Side:      openPos.Side,
							Quantity:  remainingQty,
							Leverage:  openPos.Leverage,
							OpenPrice: openPos.OpenPrice,
							OpenTime:  openPos.OpenTime,
						}
						
						// 创建修改后的closeDecision用于部分匹配（创建新副本）
						partialCloseDecision := DecisionAction{
							Action:    decision.Action,
							Symbol:    decision.Symbol,
							Quantity:  remainingQty,
							Leverage:  decision.Leverage,
							Price:     decision.Price,
							Timestamp: decision.Timestamp,
							Success:   decision.Success,
							Error:     decision.Error,
						}
						
						trade := matchSingleTrade(partialOpenPos, partialCloseDecision)
						completedTrades = append(completedTrades, trade)
						
						// 保留剩余部分的开仓（创建新副本）
						remainingOpenPos := OpenPosition{
							Symbol:    openPos.Symbol,
							Side:      openPos.Side,
							Quantity:  openPos.Quantity - remainingQty,
							Leverage:  openPos.Leverage,
							OpenPrice: openPos.OpenPrice,
							OpenTime:  openPos.OpenTime,
						}
						newList = append(newList, remainingOpenPos)
						
						remainingQty = 0
					}
				}
				openPositions[posKey] = newList
			}
		}
	}

	return completedTrades
}

// matchSingleTrade 匹配单笔交易（计算盈亏）
func matchSingleTrade(openPos OpenPosition, closeDecision DecisionAction) TradeOutcome {
	positionValue := openPos.Quantity * openPos.OpenPrice
	marginUsed := positionValue / float64(openPos.Leverage)

	// 计算实际盈亏（USDT）
	// 合约交易 PnL 计算：quantity × 价格差
	// 注意：杠杆不影响绝对盈亏，只影响保证金需求
	var pnl float64
	if openPos.Side == "long" {
		pnl = openPos.Quantity * (closeDecision.Price - openPos.OpenPrice)
	} else {
		pnl = openPos.Quantity * (openPos.OpenPrice - closeDecision.Price)
	}

	// 计算盈亏百分比（相对保证金）
	pnlPct := 0.0
	if marginUsed > 0 {
		pnlPct = (pnl / marginUsed) * 100
	}

	return TradeOutcome{
		Symbol:        openPos.Symbol,
		Side:          openPos.Side,
		Quantity:      openPos.Quantity,
		Leverage:      openPos.Leverage,
		OpenPrice:     openPos.OpenPrice,
		ClosePrice:    closeDecision.Price,
		PositionValue: positionValue,
		MarginUsed:    marginUsed,
		PnL:           pnl,
		PnLPct:        pnlPct,
		OpenTime:      openPos.OpenTime,
		CloseTime:     closeDecision.Timestamp,
	}
}

// calculateStatistics 计算统计信息
func calculateStatistics(trades []TradeOutcome, cyclesWithTrades, totalCycles int) *ValidationResult {
	result := &ValidationResult{
		TotalCycles:      totalCycles,
		CyclesWithTrades: cyclesWithTrades,
		TotalTrades:      len(trades),
		CompletedTrades:  trades,
		SymbolStats:      make(map[string]*SymbolPerformance),
	}

	if len(trades) == 0 {
		return result
	}

	var totalWin, totalLoss float64

	for _, trade := range trades {
		// 更新币种统计
		if _, ok := result.SymbolStats[trade.Symbol]; !ok {
			result.SymbolStats[trade.Symbol] = &SymbolPerformance{
				Symbol: trade.Symbol,
			}
		}

		stats := result.SymbolStats[trade.Symbol]
		stats.TotalTrades++
		stats.TotalPnL += trade.PnL
		stats.AvgPnL = stats.TotalPnL / float64(stats.TotalTrades)

		// 总体统计
		if trade.PnL > 0 {
			result.WinningTrades++
			stats.WinningTrades++
			totalWin += trade.PnL
		} else if trade.PnL < 0 {
			result.LosingTrades++
			stats.LosingTrades++
			totalLoss += math.Abs(trade.PnL)
		}

		if stats.TotalTrades > 0 {
			stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
		}
	}

	// 计算总体胜率
	if result.TotalTrades > 0 {
		result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades) * 100
	}

	// 计算平均盈亏
	if result.WinningTrades > 0 {
		result.AvgWin = totalWin / float64(result.WinningTrades)
	}
	if result.LosingTrades > 0 {
		result.AvgLoss = totalLoss / float64(result.LosingTrades)
	}

	// 计算盈亏比
	if totalLoss > 0 {
		result.ProfitFactor = totalWin / totalLoss
	}

	// 找出最佳和最差币种
	bestPnL := -1e10
	worstPnL := 1e10
	for symbol, stats := range result.SymbolStats {
		if stats.TotalPnL > bestPnL {
			bestPnL = stats.TotalPnL
			result.BestSymbol = symbol
		}
		if stats.TotalPnL < worstPnL {
			worstPnL = stats.TotalPnL
			result.WorstSymbol = symbol
		}
	}

	return result
}


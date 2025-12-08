package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DecisionRecord 决策记录（简化版，用于Phase 1）
type DecisionRecord struct {
	Timestamp      time.Time `json:"timestamp"`
	CycleNumber    int       `json:"cycle_number"`
	CoTTrace       string    `json:"cot_trace"`
	DecisionJSON   string    `json:"decision_json"`
	AccountState   AccountSnapshot
	Positions      []PositionSnapshot
	Decisions      []DecisionAction
	ExecutionLog   []string
	Success        bool
	ErrorMessage   string
}

type AccountSnapshot struct {
	TotalBalance          float64 `json:"total_balance"`
	AvailableBalance      float64 `json:"available_balance"`
	TotalUnrealizedProfit float64 `json:"total_unrealized_profit"`
	PositionCount         int     `json:"position_count"`
	MarginUsedPct         float64 `json:"margin_used_pct"`
}

type PositionSnapshot struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	PositionAmt      float64 `json:"position_amt"`
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	UnrealizedProfit float64 `json:"unrealized_profit"`
	Leverage         float64 `json:"leverage"`
	LiquidationPrice float64 `json:"liquidation_price"`
}

type DecisionAction struct {
	Action          string    `json:"action"`
	Symbol          string    `json:"symbol"`
	Quantity        float64   `json:"quantity"`
	Leverage        int       `json:"leverage"`
	Price           float64   `json:"price"`
	OrderID         int64     `json:"order_id"`
	Timestamp       time.Time `json:"timestamp"`
	Success         bool      `json:"success"`
	Error           string    `json:"error"`
	IsAutoTriggered bool      `json:"is_auto_triggered"`
	WasStopLoss     bool      `json:"was_stop_loss"`
}

// TradeRecord 交易记录
type TradeRecord struct {
	ID             int64
	TraderID       string
	Symbol         string
	Action         string
	Side           string
	Quantity       float64
	ExecutionPrice float64
	PnL            *float64
	Fee            float64
	Timestamp      time.Time
}

// AnalysisResult Phase 1分析结果
type AnalysisResult struct {
	TraderID           string
	StartCycle         int
	EndCycle           int
	TotalCycles        int
	CyclesWithTrades   int
	DecisionRecords    []DecisionRecord
	TradeRecords       []TradeRecord
	MatchedTrades      []MatchedTrade
	TimeSeries         []TimeSeriesPoint
	BasicStats         BasicStatistics
	ReportPath         string
}

type MatchedTrade struct {
	CycleNumber    int
	OpenAction     DecisionAction
	CloseAction    DecisionAction
	OpenTrade      TradeRecord
	CloseTrade     TradeRecord
	PnL            float64
	Duration        time.Duration
	RiskRewardRatio float64
}

type TimeSeriesPoint struct {
	CycleNumber    int
	Timestamp      time.Time
	CumulativePnL  float64
	AccountBalance float64
	PositionCount  int
	TradeCount     int
}

type BasicStatistics struct {
	TotalTrades        int
	WinningTrades      int
	LosingTrades       int
	WinRate            float64
	TotalPnL           float64
	NetPnL             float64
	AvgWin             float64
	AvgLoss            float64
	ProfitFactor       float64
	TotalFees          float64
	MaxProfit          float64
	MaxLoss            float64
	MaxDrawdown        float64
	SharpeRatio        float64
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run analyze_negative_profit_phase1.go <trader_id> <log_dir> <db_path> <start_cycle> <end_cycle>")
		fmt.Println("Example: go run analyze_negative_profit_phase1.go hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728 decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728 config.db.test 400 1892")
		os.Exit(1)
	}

	traderID := os.Args[1]
	logDir := os.Args[2]
	dbPath := os.Args[3]
	startCycle, _ := strconv.Atoi(os.Args[4])
	endCycle, _ := strconv.Atoi(os.Args[5])

	fmt.Printf("\n=== Phase 1: 数据收集与预处理 ===\n\n")
	fmt.Printf("Trader ID: %s\n", traderID)
	fmt.Printf("Cycle范围: %d - %d\n", startCycle, endCycle)
	fmt.Printf("决策日志目录: %s\n", logDir)
	fmt.Printf("数据库路径: %s\n\n", dbPath)

	// 1. 收集决策日志
	fmt.Println("📊 Step 1: 收集决策日志...")
	decisionRecords, err := collectDecisionLogs(logDir, startCycle, endCycle)
	if err != nil {
		log.Fatalf("收集决策日志失败: %v", err)
	}
	fmt.Printf("✅ 收集到 %d 个决策记录\n\n", len(decisionRecords))

	// 2. 收集交易历史
	fmt.Println("📊 Step 2: 收集交易历史...")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	tradeRecords, err := collectTradeHistory(context.Background(), db, traderID, decisionRecords)
	if err != nil {
		log.Fatalf("收集交易历史失败: %v", err)
	}
	fmt.Printf("✅ 收集到 %d 条交易记录\n\n", len(tradeRecords))

	// 3. 数据关联与匹配
	fmt.Println("📊 Step 3: 关联决策与交易...")
	matchedTrades, err := matchDecisionsWithTrades(decisionRecords, tradeRecords)
	if err != nil {
		log.Fatalf("关联数据失败: %v", err)
	}
	fmt.Printf("✅ 匹配到 %d 笔完整交易\n\n", len(matchedTrades))

	// 4. 计算基础统计
	fmt.Println("📊 Step 4: 计算基础统计指标...")
	basicStats := calculateBasicStats(matchedTrades, tradeRecords)
	fmt.Printf("✅ 统计计算完成\n\n")

	// 5. 生成时间序列
	fmt.Println("📊 Step 5: 生成时间序列数据...")
	timeSeries := generateTimeSeries(decisionRecords, matchedTrades)
	fmt.Printf("✅ 时间序列生成完成\n\n")

	// 6. 构建分析结果
	result := &AnalysisResult{
		TraderID:         traderID,
		StartCycle:       startCycle,
		EndCycle:         endCycle,
		TotalCycles:      len(decisionRecords),
		CyclesWithTrades: countCyclesWithTrades(decisionRecords),
		DecisionRecords:  decisionRecords,
		TradeRecords:     tradeRecords,
		MatchedTrades:    matchedTrades,
		TimeSeries:       timeSeries,
		BasicStats:       basicStats,
	}

	// 7. 生成报告
	fmt.Println("📊 Step 6: 生成分析报告...")
	reportPath, err := generateReport(result)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	result.ReportPath = reportPath
	fmt.Printf("✅ 报告已生成: %s\n\n", reportPath)

	// 8. 保存结果到JSON
	jsonPath := fmt.Sprintf("phase1_result_%d_%d.json", startCycle, endCycle)
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	ioutil.WriteFile(jsonPath, jsonData, 0644)
	fmt.Printf("✅ 结果已保存到: %s\n\n", jsonPath)

	// 9. 输出摘要
	printSummary(result)
}

// collectDecisionLogs 收集决策日志
// 从指定的起始文件名开始，收集连续的404个cycle的记录
func collectDecisionLogs(logDir string, startCycle, endCycle int) ([]DecisionRecord, error) {
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %w", err)
	}

	cyclePattern := regexp.MustCompile(`cycle(\d+)\.json$`)
	
	// 第一步：收集所有文件，按修改时间排序
	type fileInfo struct {
		name      string
		cycleNum  int
		modTime   time.Time
		filePath  string
	}
	
	var allFiles []fileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := cyclePattern.FindStringSubmatch(file.Name())
		if len(matches) < 2 {
			continue
		}

		cycleNum, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		// 只处理在范围内的cycle
		if cycleNum < startCycle || cycleNum > endCycle {
			continue
		}

		filePath := filepath.Join(logDir, file.Name())
		allFiles = append(allFiles, fileInfo{
			name:     file.Name(),
			cycleNum: cycleNum,
			modTime:  file.ModTime(),
			filePath: filePath,
		})
	}

	// 按修改时间排序
	sort.Slice(allFiles, func(i, j int) bool {
		if allFiles[i].modTime.Equal(allFiles[j].modTime) {
			return allFiles[i].cycleNum < allFiles[j].cycleNum
		}
		return allFiles[i].modTime.Before(allFiles[j].modTime)
	})

	// 第二步：找到起始文件（cycle 1，时间在11月25日14:37左右）
	startFileIdx := -1
	for i, f := range allFiles {
		if f.cycleNum == startCycle && f.modTime.After(time.Date(2025, 11, 25, 14, 30, 0, 0, time.Local)) {
			startFileIdx = i
			break
		}
	}

	if startFileIdx == -1 {
		return nil, fmt.Errorf("未找到起始cycle %d的文件", startCycle)
	}

	// 第三步：从起始文件开始，收集404个唯一cycle的记录
	var records []DecisionRecord
	collectedCycles := make(map[int]bool)
	targetCycles := make(map[int]bool)
	for i := startCycle; i <= endCycle; i++ {
		targetCycles[i] = true
	}

	for i := startFileIdx; i < len(allFiles); i++ {
		f := allFiles[i]
		
		// 如果已经收集了404个cycle，停止
		if len(collectedCycles) >= (endCycle - startCycle + 1) {
			break
		}

		// 只收集目标cycle范围内的文件
		if !targetCycles[f.cycleNum] {
			continue
		}

		// 如果这个cycle已经收集过，跳过（每个cycle只收集第一个文件）
		if collectedCycles[f.cycleNum] {
			continue
		}

		// 读取并解析文件
		data, err := ioutil.ReadFile(f.filePath)
		if err != nil {
			continue
		}

		var record DecisionRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		// 强制使用文件名中的cycle号（最可靠）
		// 因为JSON解析可能有问题，文件名中的cycle号是准确的
		record.CycleNumber = f.cycleNum

		records = append(records, record)
		collectedCycles[record.CycleNumber] = true
	}

	// 按cycle number排序
	sort.Slice(records, func(i, j int) bool {
		return records[i].CycleNumber < records[j].CycleNumber
	})

	return records, nil
}

// collectTradeHistory 收集交易历史
func collectTradeHistory(ctx context.Context, db *sql.DB, traderID string, decisions []DecisionRecord) ([]TradeRecord, error) {
	if len(decisions) == 0 {
		return []TradeRecord{}, nil
	}

	// 获取时间范围
	startTime := decisions[0].Timestamp
	endTime := decisions[len(decisions)-1].Timestamp

	// 扩展时间范围，确保包含所有相关交易
	startTime = startTime.Add(-1 * time.Hour)
	endTime = endTime.Add(1 * time.Hour)

	query := `
		SELECT id, trader_id, symbol, action, side, quantity, execution_price, pnl, fee, timestamp
		FROM trade_history
		WHERE trader_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := db.QueryContext(ctx, query, traderID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("查询交易历史失败: %w", err)
	}
	defer rows.Close()

	var trades []TradeRecord
	for rows.Next() {
		var trade TradeRecord
		var pnl sql.NullFloat64

		err := rows.Scan(
			&trade.ID, &trade.TraderID, &trade.Symbol, &trade.Action, &trade.Side,
			&trade.Quantity, &trade.ExecutionPrice, &pnl, &trade.Fee, &trade.Timestamp,
		)
		if err != nil {
			continue
		}

		if pnl.Valid {
			trade.PnL = &pnl.Float64
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// matchDecisionsWithTrades 匹配决策与交易
func matchDecisionsWithTrades(decisions []DecisionRecord, trades []TradeRecord) ([]MatchedTrade, error) {
	var matchedTrades []MatchedTrade

	// 按symbol和side分组开仓和平仓
	openTrades := make(map[string][]TradeRecord) // key: symbol_side
	closeTrades := make(map[string][]TradeRecord)

	for _, trade := range trades {
		key := fmt.Sprintf("%s_%s", trade.Symbol, trade.Side)
		if trade.Action == "open_long" || trade.Action == "open_short" {
			openTrades[key] = append(openTrades[key], trade)
		} else if trade.Action == "close_long" || trade.Action == "close_short" {
			closeTrades[key] = append(closeTrades[key], trade)
		}
	}

	// 匹配开仓和平仓
	for key, opens := range openTrades {
		closes := closeTrades[key]
		if len(closes) == 0 {
			continue
		}

		// 简单匹配：按时间顺序匹配
		openIdx := 0
		for _, closeTrade := range closes {
			if openIdx >= len(opens) {
				break
			}

			openTrade := opens[openIdx]
			if closeTrade.Timestamp.Before(openTrade.Timestamp) {
				continue
			}

			// 找到对应的决策
			var cycleNum int
			var openAction, closeAction DecisionAction
			for _, decision := range decisions {
				for _, action := range decision.Decisions {
					if action.Symbol == openTrade.Symbol && action.Action == openTrade.Action && action.Success {
						if openAction.Symbol == "" {
							openAction = action
							cycleNum = decision.CycleNumber
						}
					}
					if action.Symbol == closeTrade.Symbol && action.Action == closeTrade.Action && action.Success {
						closeAction = action
					}
				}
			}

			if openAction.Symbol != "" && closeAction.Symbol != "" && closeTrade.PnL != nil {
				matchedTrades = append(matchedTrades, MatchedTrade{
					CycleNumber:     cycleNum,
					OpenAction:      openAction,
					CloseAction:     closeAction,
					OpenTrade:       openTrade,
					CloseTrade:      closeTrade,
					PnL:             *closeTrade.PnL,
					Duration:        closeTrade.Timestamp.Sub(openTrade.Timestamp),
					RiskRewardRatio: 0, // 稍后计算
				})
				openIdx++
			}
		}
	}

	return matchedTrades, nil
}

// calculateBasicStats 计算基础统计
func calculateBasicStats(matchedTrades []MatchedTrade, allTrades []TradeRecord) BasicStatistics {
	stats := BasicStatistics{}

	// 统计完整交易
	stats.TotalTrades = len(matchedTrades)
	var totalWin, totalLoss float64

	for _, trade := range matchedTrades {
		if trade.PnL > 0 {
			stats.WinningTrades++
			totalWin += trade.PnL
			if trade.PnL > stats.MaxProfit {
				stats.MaxProfit = trade.PnL
			}
		} else if trade.PnL < 0 {
			stats.LosingTrades++
			totalLoss += trade.PnL
			if trade.PnL < stats.MaxLoss {
				stats.MaxLoss = trade.PnL
			}
		}
		stats.TotalPnL += trade.PnL
	}

	// 计算费用
	for _, trade := range allTrades {
		stats.TotalFees += trade.Fee
	}

	stats.NetPnL = stats.TotalPnL - stats.TotalFees

	// 计算胜率
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	}

	// 计算平均盈亏
	if stats.WinningTrades > 0 {
		stats.AvgWin = totalWin / float64(stats.WinningTrades)
	}
	if stats.LosingTrades > 0 {
		stats.AvgLoss = totalLoss / float64(stats.LosingTrades)
	}

	// 计算盈亏比
	if stats.AvgLoss != 0 {
		stats.ProfitFactor = stats.AvgWin / (-stats.AvgLoss)
	}

	// 计算最大回撤
	var cumulativePnL float64
	var peak float64
	for _, trade := range matchedTrades {
		cumulativePnL += trade.PnL
		if cumulativePnL > peak {
			peak = cumulativePnL
		}
		drawdown := peak - cumulativePnL
		if drawdown > stats.MaxDrawdown {
			stats.MaxDrawdown = drawdown
		}
	}

	// 计算Sharpe比率（简化版）
	if len(matchedTrades) > 1 {
		var returns []float64
		var prevPnL float64
		for _, trade := range matchedTrades {
			if prevPnL != 0 {
				returns = append(returns, (trade.PnL-prevPnL)/math.Abs(prevPnL))
			}
			prevPnL = trade.PnL
		}
		if len(returns) > 0 {
			mean := 0.0
			for _, r := range returns {
				mean += r
			}
			mean /= float64(len(returns))

			variance := 0.0
			for _, r := range returns {
				variance += (r - mean) * (r - mean)
			}
			variance /= float64(len(returns))
			stdDev := math.Sqrt(variance)

			if stdDev > 0 {
				stats.SharpeRatio = mean / stdDev
			}
		}
	}

	return stats
}

// generateTimeSeries 生成时间序列
func generateTimeSeries(decisions []DecisionRecord, matchedTrades []MatchedTrade) []TimeSeriesPoint {
	// 按cycle组织数据
	cycleMap := make(map[int]*TimeSeriesPoint)
	for _, decision := range decisions {
		cycleMap[decision.CycleNumber] = &TimeSeriesPoint{
			CycleNumber:    decision.CycleNumber,
			Timestamp:      decision.Timestamp,
			AccountBalance: decision.AccountState.TotalBalance,
			PositionCount:  decision.AccountState.PositionCount,
		}
	}

	// 按时间排序所有交易，累积PnL
	sort.Slice(matchedTrades, func(i, j int) bool {
		return matchedTrades[i].CloseTrade.Timestamp.Before(matchedTrades[j].CloseTrade.Timestamp)
	})

	var cumulativePnL float64
	tradeIdx := 0

	// 按cycle顺序遍历，累积PnL
	var points []TimeSeriesPoint
	for _, decision := range decisions {
		point := TimeSeriesPoint{
			CycleNumber:    decision.CycleNumber,
			Timestamp:      decision.Timestamp,
			AccountBalance: decision.AccountState.TotalBalance,
			PositionCount:  decision.AccountState.PositionCount,
			CumulativePnL:   cumulativePnL,
		}

		// 检查是否有交易在这个cycle或之前完成
		for tradeIdx < len(matchedTrades) {
			trade := matchedTrades[tradeIdx]
			// 找到对应的决策cycle（使用开仓的cycle）
			if trade.CycleNumber <= decision.CycleNumber {
				// 如果平仓时间在这个cycle或之前，累积PnL
				if !trade.CloseTrade.Timestamp.After(decision.Timestamp) {
					cumulativePnL += trade.PnL
					point.TradeCount++
					tradeIdx++
				} else {
					break
				}
			} else {
				break
			}
		}

		point.CumulativePnL = cumulativePnL
		points = append(points, point)
	}

	return points
}

// countCyclesWithTrades 统计有交易的cycle数
func countCyclesWithTrades(decisions []DecisionRecord) int {
	count := 0
	for _, decision := range decisions {
		hasTrade := false
		for _, action := range decision.Decisions {
			if action.Success && (action.Action == "open_long" || action.Action == "open_short" ||
				action.Action == "close_long" || action.Action == "close_short") {
				hasTrade = true
				break
			}
		}
		if hasTrade {
			count++
		}
	}
	return count
}

// generateReport 生成报告
func generateReport(result *AnalysisResult) (string, error) {
	reportPath := fmt.Sprintf("phase1_report_%d_%d.md", result.StartCycle, result.EndCycle)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Phase 1: 数据收集与预处理报告\n\n")
	fmt.Fprintf(file, "**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "## 基本信息\n\n")
	fmt.Fprintf(file, "- **Trader ID**: %s\n", result.TraderID)
	fmt.Fprintf(file, "- **Cycle范围**: %d - %d\n", result.StartCycle, result.EndCycle)
	fmt.Fprintf(file, "- **总Cycle数**: %d\n", result.TotalCycles)
	fmt.Fprintf(file, "- **有交易的Cycle数**: %d\n", result.CyclesWithTrades)
	fmt.Fprintf(file, "- **决策记录数**: %d\n", len(result.DecisionRecords))
	fmt.Fprintf(file, "- **交易记录数**: %d\n", len(result.TradeRecords))
	fmt.Fprintf(file, "- **完整交易数**: %d\n\n", len(result.MatchedTrades))

	fmt.Fprintf(file, "## 基础统计指标\n\n")
	stats := result.BasicStats
	fmt.Fprintf(file, "### 交易统计\n\n")
	fmt.Fprintf(file, "- **总交易数**: %d\n", stats.TotalTrades)
	fmt.Fprintf(file, "- **盈利交易数**: %d\n", stats.WinningTrades)
	fmt.Fprintf(file, "- **亏损交易数**: %d\n", stats.LosingTrades)
	fmt.Fprintf(file, "- **胜率**: %.2f%%\n\n", stats.WinRate)

	fmt.Fprintf(file, "### 盈亏统计\n\n")
	fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", stats.TotalPnL)
	fmt.Fprintf(file, "- **净盈亏**: %.2f USDT\n", stats.NetPnL)
	fmt.Fprintf(file, "- **总手续费**: %.2f USDT\n", stats.TotalFees)
	fmt.Fprintf(file, "- **平均盈利**: %.2f USDT\n", stats.AvgWin)
	fmt.Fprintf(file, "- **平均亏损**: %.2f USDT\n", stats.AvgLoss)
	fmt.Fprintf(file, "- **盈亏比**: %.2f\n", stats.ProfitFactor)
	fmt.Fprintf(file, "- **最大盈利**: %.2f USDT\n", stats.MaxProfit)
	fmt.Fprintf(file, "- **最大亏损**: %.2f USDT\n\n", stats.MaxLoss)

	fmt.Fprintf(file, "### 风险指标\n\n")
	fmt.Fprintf(file, "- **最大回撤**: %.2f USDT\n", stats.MaxDrawdown)
	fmt.Fprintf(file, "- **Sharpe比率**: %.2f\n\n", stats.SharpeRatio)

	fmt.Fprintf(file, "## 时间序列摘要\n\n")
	fmt.Fprintf(file, "| Cycle | 时间 | 累计PnL | 账户余额 | 持仓数 |\n")
	fmt.Fprintf(file, "|-------|------|---------|----------|--------|\n")
	for i, point := range result.TimeSeries {
		if i%50 == 0 || i == len(result.TimeSeries)-1 { // 每50个cycle显示一次
			fmt.Fprintf(file, "| %d | %s | %.2f | %.2f | %d |\n",
				point.CycleNumber,
				point.Timestamp.Format("2006-01-02 15:04:05"),
				point.CumulativePnL,
				point.AccountBalance,
				point.PositionCount)
		}
	}

	fmt.Fprintf(file, "\n## 关键发现\n\n")
	if stats.TotalPnL < 0 {
		fmt.Fprintf(file, "⚠️ **收益为负**: 总盈亏 %.2f USDT\n", stats.TotalPnL)
	}
	if stats.WinRate < 50 {
		fmt.Fprintf(file, "⚠️ **胜率偏低**: %.2f%% (低于50%%)\n", stats.WinRate)
	}
	if stats.ProfitFactor < 1 {
		fmt.Fprintf(file, "⚠️ **盈亏比偏低**: %.2f (低于1.0)\n", stats.ProfitFactor)
	}
	if stats.MaxDrawdown > math.Abs(stats.TotalPnL)*0.5 {
		fmt.Fprintf(file, "⚠️ **回撤较大**: 最大回撤 %.2f USDT\n", stats.MaxDrawdown)
	}

	fmt.Fprintf(file, "\n## 下一步分析建议\n\n")
	fmt.Fprintf(file, "1. **Phase 2**: 交易表现分析（按币种、方向、周期等维度）\n")
	fmt.Fprintf(file, "2. **Phase 3**: 决策质量分析（规则违反、决策逻辑）\n")
	fmt.Fprintf(file, "3. **Phase 4**: Prompt问题诊断\n")
	fmt.Fprintf(file, "4. **Phase 5**: 根因分析与修正方案\n")

	return reportPath, nil
}

// printSummary 打印摘要
func printSummary(result *AnalysisResult) {
	fmt.Println("\n=== Phase 1 分析摘要 ===")
	fmt.Printf("总Cycle数: %d\n", result.TotalCycles)
	fmt.Printf("有交易的Cycle数: %d\n", result.CyclesWithTrades)
	fmt.Printf("完整交易数: %d\n", len(result.MatchedTrades))
	fmt.Printf("\n基础统计:\n")
	fmt.Printf("  总盈亏: %.2f USDT\n", result.BasicStats.TotalPnL)
	fmt.Printf("  胜率: %.2f%%\n", result.BasicStats.WinRate)
	fmt.Printf("  盈亏比: %.2f\n", result.BasicStats.ProfitFactor)
	fmt.Printf("  最大回撤: %.2f USDT\n", result.BasicStats.MaxDrawdown)
	fmt.Printf("\n报告路径: %s\n", result.ReportPath)
}


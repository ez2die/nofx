package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"
)

// 从Phase 1结果加载数据结构
type Phase1Result struct {
	TraderID         string
	StartCycle       int
	EndCycle         int
	TotalCycles      int
	CyclesWithTrades int
	DecisionRecords  []DecisionRecord
	TradeRecords     []TradeRecord
	MatchedTrades    []MatchedTrade
	TimeSeries       []TimeSeriesPoint
	BasicStats       BasicStatistics
}

type DecisionRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	CycleNumber  int       `json:"cycle_number"`
	CoTTrace     string    `json:"cot_trace"`
	DecisionJSON string    `json:"decision_json"`
	AccountState AccountSnapshot
	Positions    []PositionSnapshot
	Decisions    []DecisionAction
	ExecutionLog []string
	Success      bool
	ErrorMessage string
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

type MatchedTrade struct {
	CycleNumber     int
	OpenAction      DecisionAction
	CloseAction     DecisionAction
	OpenTrade       TradeRecord
	CloseTrade      TradeRecord
	PnL             float64
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
	TotalTrades   int
	WinningTrades int
	LosingTrades int
	WinRate       float64
	TotalPnL      float64
	NetPnL        float64
	AvgWin        float64
	AvgLoss       float64
	ProfitFactor  float64
	TotalFees     float64
	MaxProfit     float64
	MaxLoss       float64
	MaxDrawdown   float64
	SharpeRatio   float64
}

// Phase 2分析结果
type Phase2Analysis struct {
	SymbolAnalysis      map[string]*SymbolStats
	DirectionAnalysis   *DirectionStats
	CycleAnalysis       []*CycleStats
	LeverageAnalysis    map[int]*LeverageStats
	TradePairAnalysis   []*TradePairStats
	ReportPath          string
}

type SymbolStats struct {
	Symbol         string
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        float64
	TotalPnL       float64
	AvgPnL         float64
	MaxProfit      float64
	MaxLoss        float64
	TotalFees       float64
	AvgDuration     time.Duration
	StopLossTriggers int
	TakeProfitTriggers int
}

type DirectionStats struct {
	LongTrades     int
	ShortTrades   int
	LongPnL        float64
	ShortPnL       float64
	LongWinRate    float64
	ShortWinRate   float64
	LongAvgPnL     float64
	ShortAvgPnL    float64
}

type CycleStats struct {
	CycleRange     string
	StartCycle     int
	EndCycle       int
	TotalTrades    int
	TotalPnL        float64
	WinRate         float64
	AvgPnL          float64
}

type LeverageStats struct {
	Leverage       int
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        float64
	TotalPnL       float64
	AvgPnL         float64
	MaxProfit      float64
	MaxLoss        float64
}

type TradePairStats struct {
	Trade          MatchedTrade
	EntryTiming    string // "good", "bad", "neutral"
	ExitReason     string // "stop_loss", "take_profit", "manual", "auto"
	HoldDuration   time.Duration
	PriceMovement  float64 // 价格变动百分比
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run analyze_negative_profit_phase2.go <phase1_result.json>")
		fmt.Println("Example: go run analyze_negative_profit_phase2.go phase1_result_400_1892.json")
		os.Exit(1)
	}

	phase1File := os.Args[1]

	fmt.Printf("\n=== Phase 2: 交易表现分析 ===\n\n")
	fmt.Printf("读取Phase 1结果: %s\n\n", phase1File)

	// 1. 加载Phase 1数据
	fmt.Println("📊 Step 1: 加载Phase 1数据...")
	phase1Data, err := loadPhase1Data(phase1File)
	if err != nil {
		log.Fatalf("加载Phase 1数据失败: %v", err)
	}
	fmt.Printf("✅ 加载完成: %d笔交易, %d个cycle\n\n", len(phase1Data.MatchedTrades), len(phase1Data.DecisionRecords))

	// 2. 按币种维度分析
	fmt.Println("📊 Step 2: 按币种维度分析...")
	symbolAnalysis := analyzeBySymbol(phase1Data.MatchedTrades)
	fmt.Printf("✅ 分析了 %d 个币种\n\n", len(symbolAnalysis))

	// 3. 按方向维度分析
	fmt.Println("📊 Step 3: 按方向维度分析...")
	directionAnalysis := analyzeByDirection(phase1Data.MatchedTrades)
	fmt.Printf("✅ 方向分析完成\n\n")

	// 4. 按周期维度分析
	fmt.Println("📊 Step 4: 按周期维度分析...")
	cycleAnalysis := analyzeByCycle(phase1Data.MatchedTrades, phase1Data.StartCycle, phase1Data.EndCycle)
	fmt.Printf("✅ 分析了 %d 个周期段\n\n", len(cycleAnalysis))

	// 5. 按杠杆维度分析
	fmt.Println("📊 Step 5: 按杠杆维度分析...")
	leverageAnalysis := analyzeByLeverage(phase1Data.MatchedTrades)
	fmt.Printf("✅ 分析了 %d 个杠杆级别\n\n", len(leverageAnalysis))

	// 6. 交易配对分析
	fmt.Println("📊 Step 6: 交易配对分析...")
	tradePairAnalysis := analyzeTradePairs(phase1Data.MatchedTrades)
	fmt.Printf("✅ 分析了 %d 笔交易配对\n\n", len(tradePairAnalysis))

	// 7. 构建分析结果
	analysis := &Phase2Analysis{
		SymbolAnalysis:    symbolAnalysis,
		DirectionAnalysis: directionAnalysis,
		CycleAnalysis:     cycleAnalysis,
		LeverageAnalysis: leverageAnalysis,
		TradePairAnalysis: tradePairAnalysis,
	}

	// 8. 生成报告
	fmt.Println("📊 Step 7: 生成分析报告...")
	reportPath, err := generatePhase2Report(phase1Data, analysis)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	analysis.ReportPath = reportPath
	fmt.Printf("✅ 报告已生成: %s\n\n", reportPath)

	// 9. 保存结果
	jsonPath := fmt.Sprintf("phase2_result_%d_%d.json", phase1Data.StartCycle, phase1Data.EndCycle)
	jsonData, _ := json.MarshalIndent(analysis, "", "  ")
	ioutil.WriteFile(jsonPath, jsonData, 0644)
	fmt.Printf("✅ 结果已保存到: %s\n\n", jsonPath)

	// 10. 输出摘要
	printPhase2Summary(analysis)
}

func loadPhase1Data(filename string) (*Phase1Result, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var result Phase1Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func analyzeBySymbol(trades []MatchedTrade) map[string]*SymbolStats {
	stats := make(map[string]*SymbolStats)

	for _, trade := range trades {
		symbol := trade.OpenTrade.Symbol
		if _, exists := stats[symbol]; !exists {
			stats[symbol] = &SymbolStats{
				Symbol: symbol,
			}
		}

		s := stats[symbol]
		s.TotalTrades++
		s.TotalPnL += trade.PnL
		s.TotalFees += trade.OpenTrade.Fee + trade.CloseTrade.Fee
		s.AvgDuration += trade.Duration

		if trade.PnL > 0 {
			s.WinningTrades++
			if trade.PnL > s.MaxProfit {
				s.MaxProfit = trade.PnL
			}
		} else if trade.PnL < 0 {
			s.LosingTrades++
			if trade.PnL < s.MaxLoss {
				s.MaxLoss = trade.PnL
			}
		}

		// 检查止损/止盈触发
		if trade.CloseAction.WasStopLoss {
			s.StopLossTriggers++
		} else if trade.CloseAction.IsAutoTriggered && !trade.CloseAction.WasStopLoss {
			s.TakeProfitTriggers++
		}
	}

	// 计算统计指标
	for _, s := range stats {
		if s.TotalTrades > 0 {
			s.WinRate = float64(s.WinningTrades) / float64(s.TotalTrades) * 100
			s.AvgPnL = s.TotalPnL / float64(s.TotalTrades)
			s.AvgDuration = s.AvgDuration / time.Duration(s.TotalTrades)
		}
	}

	return stats
}

func analyzeByDirection(trades []MatchedTrade) *DirectionStats {
	stats := &DirectionStats{}

	for _, trade := range trades {
		side := trade.OpenTrade.Side
		if side == "long" {
			stats.LongTrades++
			stats.LongPnL += trade.PnL
			if trade.PnL > 0 {
				stats.LongWinRate += 1
			}
		} else if side == "short" {
			stats.ShortTrades++
			stats.ShortPnL += trade.PnL
			if trade.PnL > 0 {
				stats.ShortWinRate += 1
			}
		}
	}

	if stats.LongTrades > 0 {
		stats.LongWinRate = stats.LongWinRate / float64(stats.LongTrades) * 100
		stats.LongAvgPnL = stats.LongPnL / float64(stats.LongTrades)
	}
	if stats.ShortTrades > 0 {
		stats.ShortWinRate = stats.ShortWinRate / float64(stats.ShortTrades) * 100
		stats.ShortAvgPnL = stats.ShortPnL / float64(stats.ShortTrades)
	}

	return stats
}

func analyzeByCycle(trades []MatchedTrade, startCycle, endCycle int) []*CycleStats {
	// 将cycle范围分成10个区间
	numIntervals := 10
	intervalSize := (endCycle - startCycle) / numIntervals
	if intervalSize == 0 {
		intervalSize = 1
	}

	var cycleStats []*CycleStats
	for i := 0; i < numIntervals; i++ {
		start := startCycle + i*intervalSize
		end := startCycle + (i+1)*intervalSize
		if i == numIntervals-1 {
			end = endCycle
		}

		stats := &CycleStats{
			CycleRange: fmt.Sprintf("%d-%d", start, end),
			StartCycle: start,
			EndCycle:   end,
		}

		for _, trade := range trades {
			if trade.CycleNumber >= start && trade.CycleNumber < end {
				stats.TotalTrades++
				stats.TotalPnL += trade.PnL
				if trade.PnL > 0 {
					stats.WinRate += 1
				}
			}
		}

		if stats.TotalTrades > 0 {
			stats.WinRate = stats.WinRate / float64(stats.TotalTrades) * 100
			stats.AvgPnL = stats.TotalPnL / float64(stats.TotalTrades)
		}

		cycleStats = append(cycleStats, stats)
	}

	return cycleStats
}

func analyzeByLeverage(trades []MatchedTrade) map[int]*LeverageStats {
	stats := make(map[int]*LeverageStats)

	for _, trade := range trades {
		leverage := trade.OpenAction.Leverage
		if leverage == 0 {
			leverage = 1 // 默认值
		}

		if _, exists := stats[leverage]; !exists {
			stats[leverage] = &LeverageStats{
				Leverage: leverage,
			}
		}

		s := stats[leverage]
		s.TotalTrades++
		s.TotalPnL += trade.PnL

		if trade.PnL > 0 {
			s.WinningTrades++
			if trade.PnL > s.MaxProfit {
				s.MaxProfit = trade.PnL
			}
		} else if trade.PnL < 0 {
			s.LosingTrades++
			if trade.PnL < s.MaxLoss {
				s.MaxLoss = trade.PnL
			}
		}
	}

	// 计算统计指标
	for _, s := range stats {
		if s.TotalTrades > 0 {
			s.WinRate = float64(s.WinningTrades) / float64(s.TotalTrades) * 100
			s.AvgPnL = s.TotalPnL / float64(s.TotalTrades)
		}
	}

	return stats
}

func analyzeTradePairs(trades []MatchedTrade) []*TradePairStats {
	var pairs []*TradePairStats

	for _, trade := range trades {
		pair := &TradePairStats{
			Trade:        trade,
			HoldDuration: trade.Duration,
		}

		// 分析入场时机（简化版：基于后续价格走势）
		entryPrice := trade.OpenTrade.ExecutionPrice
		exitPrice := trade.CloseTrade.ExecutionPrice
		side := trade.OpenTrade.Side

		var priceMovement float64
		if side == "long" {
			priceMovement = (exitPrice - entryPrice) / entryPrice * 100
		} else {
			priceMovement = (entryPrice - exitPrice) / entryPrice * 100
		}
		pair.PriceMovement = priceMovement

		// 判断入场时机
		if trade.PnL > 0 {
			pair.EntryTiming = "good"
		} else if trade.PnL < -1.0 { // 亏损超过1 USDT
			pair.EntryTiming = "bad"
		} else {
			pair.EntryTiming = "neutral"
		}

		// 分析退出原因
		if trade.CloseAction.WasStopLoss {
			pair.ExitReason = "stop_loss"
		} else if trade.CloseAction.IsAutoTriggered && !trade.CloseAction.WasStopLoss {
			pair.ExitReason = "take_profit"
		} else {
			pair.ExitReason = "manual"
		}

		pairs = append(pairs, pair)
	}

	return pairs
}

func generatePhase2Report(phase1Data *Phase1Result, analysis *Phase2Analysis) (string, error) {
	reportPath := fmt.Sprintf("phase2_report_%d_%d.md", phase1Data.StartCycle, phase1Data.EndCycle)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Phase 2: 交易表现分析报告\n\n")
	fmt.Fprintf(file, "**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**基于Phase 1数据**: Cycle %d - %d\n\n", phase1Data.StartCycle, phase1Data.EndCycle)

	// 1. 按币种维度分析
	fmt.Fprintf(file, "## 1. 按币种维度分析\n\n")
	fmt.Fprintf(file, "| 币种 | 交易数 | 胜率 | 总盈亏 | 平均盈亏 | 最大盈利 | 最大亏损 | 止损触发 | 止盈触发 | 平均持仓时长 |\n")
	fmt.Fprintf(file, "|------|--------|------|--------|----------|----------|----------|----------|----------|--------------|\n")

	// 按总盈亏排序
	var symbols []*SymbolStats
	for _, s := range analysis.SymbolAnalysis {
		symbols = append(symbols, s)
	}
	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i].TotalPnL < symbols[j].TotalPnL // 从最差到最好
	})

	for _, s := range symbols {
		fmt.Fprintf(file, "| %s | %d | %.2f%% | %.2f | %.2f | %.2f | %.2f | %d | %d | %s |\n",
			s.Symbol, s.TotalTrades, s.WinRate, s.TotalPnL, s.AvgPnL,
			s.MaxProfit, s.MaxLoss, s.StopLossTriggers, s.TakeProfitTriggers,
			formatDuration(s.AvgDuration))
	}

	// 2. 按方向维度分析
	fmt.Fprintf(file, "\n## 2. 按方向维度分析\n\n")
	dir := analysis.DirectionAnalysis
	fmt.Fprintf(file, "### 做多(Long)\n\n")
	fmt.Fprintf(file, "- **交易数**: %d\n", dir.LongTrades)
	fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", dir.LongPnL)
	fmt.Fprintf(file, "- **胜率**: %.2f%%\n", dir.LongWinRate)
	fmt.Fprintf(file, "- **平均盈亏**: %.2f USDT\n\n", dir.LongAvgPnL)

	fmt.Fprintf(file, "### 做空(Short)\n\n")
	fmt.Fprintf(file, "- **交易数**: %d\n", dir.ShortTrades)
	fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", dir.ShortPnL)
	fmt.Fprintf(file, "- **胜率**: %.2f%%\n", dir.ShortWinRate)
	fmt.Fprintf(file, "- **平均盈亏**: %.2f USDT\n\n", dir.ShortAvgPnL)

	// 3. 按周期维度分析
	fmt.Fprintf(file, "## 3. 按周期维度分析\n\n")
	fmt.Fprintf(file, "| Cycle范围 | 交易数 | 总盈亏 | 胜率 | 平均盈亏 |\n")
	fmt.Fprintf(file, "|-----------|--------|--------|------|----------|\n")
	for _, c := range analysis.CycleAnalysis {
		fmt.Fprintf(file, "| %s | %d | %.2f | %.2f%% | %.2f |\n",
			c.CycleRange, c.TotalTrades, c.TotalPnL, c.WinRate, c.AvgPnL)
	}

	// 4. 按杠杆维度分析
	fmt.Fprintf(file, "\n## 4. 按杠杆维度分析\n\n")
	fmt.Fprintf(file, "| 杠杆 | 交易数 | 胜率 | 总盈亏 | 平均盈亏 | 最大盈利 | 最大亏损 |\n")
	fmt.Fprintf(file, "|------|--------|------|--------|----------|----------|----------|\n")

	var leverages []*LeverageStats
	for _, l := range analysis.LeverageAnalysis {
		leverages = append(leverages, l)
	}
	sort.Slice(leverages, func(i, j int) bool {
		return leverages[i].Leverage < leverages[j].Leverage
	})

	for _, l := range leverages {
		fmt.Fprintf(file, "| %dx | %d | %.2f%% | %.2f | %.2f | %.2f | %.2f |\n",
			l.Leverage, l.TotalTrades, l.WinRate, l.TotalPnL, l.AvgPnL, l.MaxProfit, l.MaxLoss)
	}

	// 5. 交易配对分析摘要
	fmt.Fprintf(file, "\n## 5. 交易配对分析摘要\n\n")

	// 入场时机统计
	goodTiming := 0
	badTiming := 0
	neutralTiming := 0
	for _, p := range analysis.TradePairAnalysis {
		switch p.EntryTiming {
		case "good":
			goodTiming++
		case "bad":
			badTiming++
		default:
			neutralTiming++
		}
	}

	fmt.Fprintf(file, "### 入场时机分析\n\n")
	fmt.Fprintf(file, "- **良好时机**: %d (%.1f%%)\n", goodTiming, float64(goodTiming)/float64(len(analysis.TradePairAnalysis))*100)
	fmt.Fprintf(file, "- **不良时机**: %d (%.1f%%)\n", badTiming, float64(badTiming)/float64(len(analysis.TradePairAnalysis))*100)
	fmt.Fprintf(file, "- **中性时机**: %d (%.1f%%)\n\n", neutralTiming, float64(neutralTiming)/float64(len(analysis.TradePairAnalysis))*100)

	// 退出原因统计
	stopLossExits := 0
	takeProfitExits := 0
	manualExits := 0
	for _, p := range analysis.TradePairAnalysis {
		switch p.ExitReason {
		case "stop_loss":
			stopLossExits++
		case "take_profit":
			takeProfitExits++
		default:
			manualExits++
		}
	}

	fmt.Fprintf(file, "### 退出原因分析\n\n")
	fmt.Fprintf(file, "- **止损退出**: %d (%.1f%%)\n", stopLossExits, float64(stopLossExits)/float64(len(analysis.TradePairAnalysis))*100)
	fmt.Fprintf(file, "- **止盈退出**: %d (%.1f%%)\n", takeProfitExits, float64(takeProfitExits)/float64(len(analysis.TradePairAnalysis))*100)
	fmt.Fprintf(file, "- **手动退出**: %d (%.1f%%)\n\n", manualExits, float64(manualExits)/float64(len(analysis.TradePairAnalysis))*100)

	// 持仓时长分析
	var durations []time.Duration
	for _, p := range analysis.TradePairAnalysis {
		durations = append(durations, p.HoldDuration)
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	var avgDuration time.Duration
	for _, d := range durations {
		avgDuration += d
	}
	if len(durations) > 0 {
		avgDuration /= time.Duration(len(durations))
	}

	fmt.Fprintf(file, "### 持仓时长分析\n\n")
	fmt.Fprintf(file, "- **平均持仓时长**: %s\n", formatDuration(avgDuration))
	if len(durations) > 0 {
		fmt.Fprintf(file, "- **最短持仓**: %s\n", formatDuration(durations[0]))
		fmt.Fprintf(file, "- **最长持仓**: %s\n", formatDuration(durations[len(durations)-1]))
		fmt.Fprintf(file, "- **中位数持仓**: %s\n", formatDuration(durations[len(durations)/2]))
	}

	// 关键发现
	fmt.Fprintf(file, "\n## 关键发现\n\n")

	// 找出最差的币种
	if len(symbols) > 0 {
		worstSymbol := symbols[0]
		fmt.Fprintf(file, "### 最差表现币种\n\n")
		fmt.Fprintf(file, "- **币种**: %s\n", worstSymbol.Symbol)
		fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", worstSymbol.TotalPnL)
		fmt.Fprintf(file, "- **胜率**: %.2f%%\n", worstSymbol.WinRate)
		fmt.Fprintf(file, "- **交易数**: %d\n\n", worstSymbol.TotalTrades)
	}

	// 方向偏好
	fmt.Fprintf(file, "### 方向偏好分析\n\n")
	if dir.LongPnL < dir.ShortPnL {
		fmt.Fprintf(file, "⚠️ **做多表现更差**: 做多亏损 %.2f USDT vs 做空亏损 %.2f USDT\n", dir.LongPnL, dir.ShortPnL)
	} else {
		fmt.Fprintf(file, "⚠️ **做空表现更差**: 做空亏损 %.2f USDT vs 做多亏损 %.2f USDT\n", dir.ShortPnL, dir.LongPnL)
	}

	// 杠杆影响
	if len(leverages) > 0 {
		fmt.Fprintf(file, "\n### 杠杆影响分析\n\n")
		worstLeverage := leverages[0]
		for _, l := range leverages {
			if l.TotalPnL < worstLeverage.TotalPnL {
				worstLeverage = l
			}
		}
		fmt.Fprintf(file, "⚠️ **最差杠杆**: %dx (总盈亏: %.2f USDT, 胜率: %.2f%%)\n", worstLeverage.Leverage, worstLeverage.TotalPnL, worstLeverage.WinRate)
	}

	// 止损/止盈分析
	fmt.Fprintf(file, "\n### 止损/止盈分析\n\n")
	totalStopLoss := 0
	totalTakeProfit := 0
	for _, s := range symbols {
		totalStopLoss += s.StopLossTriggers
		totalTakeProfit += s.TakeProfitTriggers
	}
	fmt.Fprintf(file, "- **止损触发总数**: %d\n", totalStopLoss)
	fmt.Fprintf(file, "- **止盈触发总数**: %d\n", totalTakeProfit)
	if totalStopLoss+totalTakeProfit > 0 {
		stopLossRate := float64(totalStopLoss) / float64(totalStopLoss+totalTakeProfit) * 100
		fmt.Fprintf(file, "- **止损触发率**: %.1f%%\n", stopLossRate)
		if stopLossRate > 60 {
			fmt.Fprintf(file, "⚠️ **止损触发率过高**: 超过60%%，可能存在止损设置过窄的问题\n")
		}
	}

	fmt.Fprintf(file, "\n## 下一步分析建议\n\n")
	fmt.Fprintf(file, "1. **Phase 3**: 决策质量分析（规则违反、决策逻辑）\n")
	fmt.Fprintf(file, "2. **Phase 4**: Prompt问题诊断\n")
	fmt.Fprintf(file, "3. **Phase 5**: 根因分析与修正方案\n")

	return reportPath, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f秒", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1f分钟", d.Minutes())
	} else {
		hours := d.Hours()
		return fmt.Sprintf("%.1f小时", hours)
	}
}

func printPhase2Summary(analysis *Phase2Analysis) {
	fmt.Println("\n=== Phase 2 分析摘要 ===")
	fmt.Printf("币种分析: %d 个币种\n", len(analysis.SymbolAnalysis))
	fmt.Printf("周期分析: %d 个周期段\n", len(analysis.CycleAnalysis))
	fmt.Printf("杠杆分析: %d 个杠杆级别\n", len(analysis.LeverageAnalysis))
	fmt.Printf("交易配对分析: %d 笔交易\n", len(analysis.TradePairAnalysis))
	fmt.Printf("\n报告路径: %s\n", analysis.ReportPath)
}


package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"nofx/market"
)

// 从Phase 1和Phase 2加载的数据结构
type Phase1Result struct {
	TraderID         string
	StartCycle       int
	EndCycle         int
	DecisionRecords  []DecisionRecord
	MatchedTrades    []MatchedTrade
	BasicStats       BasicStatistics
}

type Phase2Result struct {
	SymbolAnalysis    map[string]*SymbolStats
	DirectionAnalysis *DirectionStats
	TradePairAnalysis []*TradePairStats
}

type DecisionRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	CycleNumber  int       `json:"cycle_number"`
	CoTTrace     string    `json:"cot_trace"`
	DecisionJSON string    `json:"decision_json"`
	AccountState AccountSnapshot
	Positions    []PositionSnapshot
	Decisions    []DecisionAction
	Success      bool
}

type AccountSnapshot struct {
	TotalBalance float64 `json:"total_balance"`
}

type PositionSnapshot struct {
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	PositionAmt float64 `json:"position_amt"`
	EntryPrice  float64 `json:"entry_price"`
	MarkPrice   float64 `json:"mark_price"`
}

type DecisionAction struct {
	Action          string    `json:"action"`
	Symbol          string    `json:"symbol"`
	Quantity        float64   `json:"quantity"`
	Leverage        int       `json:"leverage"`
	Price           float64   `json:"price"`
	Timestamp       time.Time `json:"timestamp"`
	Success         bool      `json:"success"`
	IsAutoTriggered bool      `json:"is_auto_triggered"`
	WasStopLoss     bool      `json:"was_stop_loss"`
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

type TradeRecord struct {
	Symbol         string
	Action         string
	Side           string
	ExecutionPrice float64
	PnL            *float64
	Timestamp      time.Time
}

type BasicStatistics struct {
	TotalPnL float64
	WinRate  float64
}

type SymbolStats struct {
	Symbol        string
	TotalTrades   int
	TotalPnL      float64
	WinRate       float64
}

type DirectionStats struct {
	LongPnL  float64
	ShortPnL float64
}

type TradePairStats struct {
	Trade        MatchedTrade
	EntryTiming  string
	ExitReason   string
	HoldDuration time.Duration
}

// Phase 3分析结果
type Phase3Analysis struct {
	MarketBenchmark    map[string]*MarketBenchmark
	RuleViolations     []*RuleViolation
	DecisionLogic      []*DecisionLogicAnalysis
	SymbolAnalyses     map[string]*SymbolAnalysis // 所有币种的专项分析
	AIvsBenchmark      *AIBenchmarkComparison
	ReportPath         string
}

type MarketBenchmark struct {
	Symbol            string
	BuyHoldReturn     float64 // 买入持有收益率
	LongOnlyReturn    float64 // 做多策略收益率
	ShortOnlyReturn   float64 // 做空策略收益率
	PriceChange       float64 // 期间价格变动
	MaxDrawdown       float64 // 最大回撤
	Volatility        float64 // 波动率
	StartPrice        float64
	EndPrice          float64
	StartTime         time.Time
	EndTime           time.Time
}

type RuleViolation struct {
	CycleNumber    int
	Symbol         string
	RuleID         string
	RuleName       string
	Severity       string
	Description    string
	Details        map[string]interface{}
	DecisionJSON   string
}

type DecisionLogicAnalysis struct {
	CycleNumber    int
	Symbol         string
	Action         string
	CoTTrace       string
	DecisionJSON   string
	LogicQuality   string // "good", "bad", "neutral"
	Issues         []string
	Strengths      []string
}

type SymbolAnalysis struct {
	Symbol               string
	TotalTrades          int
	AllStopLoss         bool
	AverageEntryPrice   float64
	AverageExitPrice    float64
	AverageStopLoss     float64
	AverageTakeProfit   float64
	MarketTrend         string // "up", "down", "sideways"
	EntryTimingIssues   []string
	StopLossIssues      []string
	MarketBenchmark     *MarketBenchmark
	DetailedTrades      []*SymbolTradeDetail
	TotalPnL            float64
	WinRate             float64
}

type SymbolTradeDetail struct {
	CycleNumber         int
	EntryPrice          float64
	ExitPrice           float64
	StopLoss            float64
	TakeProfit          float64
	MarketPriceAtEntry  float64
	MarketPriceAtExit   float64
	MarketReturn        float64
	AITradeReturn       float64
	WasStopLoss         bool
	Duration            time.Duration
	EntryTimingIssue    string // 入场时机问题描述
}

type AIBenchmarkComparison struct {
	TotalTrades        int
	AITotalPnL         float64
	BuyHoldTotalPnL    float64
	LongOnlyTotalPnL   float64
	Outperformance     float64 // AI表现 vs 买入持有
	OutperformancePct  float64
	BetterThanBuyHold  int
	WorseThanBuyHold   int
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run analyze_negative_profit_phase3.go <phase1_result.json> <phase2_result.json>")
		fmt.Println("Example: go run analyze_negative_profit_phase3.go phase1_result_400_1892.json phase2_result_400_1892.json")
		os.Exit(1)
	}

	phase1File := os.Args[1]
	phase2File := os.Args[2]

	fmt.Printf("\n=== Phase 3: 决策质量分析（含市场基准）===\n\n")
	fmt.Printf("读取Phase 1结果: %s\n", phase1File)
	fmt.Printf("读取Phase 2结果: %s\n\n", phase2File)

	// 1. 加载Phase 1和Phase 2数据
	fmt.Println("📊 Step 1: 加载Phase 1和Phase 2数据...")
	phase1Data, err := loadPhase1Data(phase1File)
	if err != nil {
		log.Fatalf("加载Phase 1数据失败: %v", err)
	}
	phase2Data, err := loadPhase2Data(phase2File)
	if err != nil {
		log.Fatalf("加载Phase 2数据失败: %v", err)
	}
	fmt.Printf("✅ 加载完成\n\n")

	// 2. 设置市场数据源为Hyperliquid
	fmt.Println("📊 Step 2: 设置市场数据源为Hyperliquid...")
	if err := market.SetDataSource(market.DataSourceHyperliquid); err != nil {
		log.Printf("⚠️ 设置Hyperliquid数据源失败: %v，尝试使用默认客户端", err)
	}
	
	// 3. 计算市场基准
	fmt.Println("📊 Step 3: 计算市场基准（买入持有、做多/做空策略）...")
	marketBenchmark, err := calculateMarketBenchmark(phase1Data.MatchedTrades)
	if err != nil {
		log.Printf("⚠️ 计算市场基准失败: %v，继续分析", err)
		marketBenchmark = make(map[string]*MarketBenchmark)
	}
	fmt.Printf("✅ 计算了 %d 个币种的市场基准\n\n", len(marketBenchmark))

	// 4. 规则违反检查
	fmt.Println("📊 Step 4: 检查规则违反...")
	ruleViolations := checkRuleViolations(phase1Data.DecisionRecords)
	fmt.Printf("✅ 发现 %d 个规则违反\n\n", len(ruleViolations))

	// 5. 决策逻辑分析
	fmt.Println("📊 Step 5: 分析决策逻辑（CoTTrace）...")
	decisionLogic := analyzeDecisionLogic(phase1Data.DecisionRecords, phase1Data.MatchedTrades)
	fmt.Printf("✅ 分析了 %d 个决策逻辑\n\n", len(decisionLogic))

	// 6. 各币种专项分析
	fmt.Println("📊 Step 6: 各币种专项分析...")
	symbolAnalyses := analyzeAllSymbols(phase1Data.MatchedTrades, marketBenchmark)
	fmt.Printf("✅ 分析了 %d 个币种\n\n", len(symbolAnalyses))

	// 7. AI vs Benchmark对比
	fmt.Println("📊 Step 7: AI决策 vs 市场基准对比...")
	aiBenchmark := compareAIvsBenchmark(phase1Data.MatchedTrades, marketBenchmark)
	fmt.Printf("✅ 对比分析完成\n\n")

	// 8. 构建分析结果
	analysis := &Phase3Analysis{
		MarketBenchmark: marketBenchmark,
		RuleViolations:  ruleViolations,
		DecisionLogic:   decisionLogic,
		SymbolAnalyses:  symbolAnalyses,
		AIvsBenchmark:   aiBenchmark,
	}

	// 9. 生成报告
	fmt.Println("📊 Step 8: 生成分析报告...")
	reportPath, err := generatePhase3Report(phase1Data, phase2Data, analysis)
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	analysis.ReportPath = reportPath
	fmt.Printf("✅ 报告已生成: %s\n\n", reportPath)

	// 10. 保存结果
	jsonPath := fmt.Sprintf("phase3_result_%d_%d.json", phase1Data.StartCycle, phase1Data.EndCycle)
	jsonData, _ := json.MarshalIndent(analysis, "", "  ")
	ioutil.WriteFile(jsonPath, jsonData, 0644)
	fmt.Printf("✅ 结果已保存到: %s\n\n", jsonPath)

	// 11. 输出摘要
	printPhase3Summary(analysis)
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

func loadPhase2Data(filename string) (*Phase2Result, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result Phase2Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func calculateMarketBenchmark(trades []MatchedTrade) (map[string]*MarketBenchmark, error) {
	benchmarks := make(map[string]*MarketBenchmark)

	// 按币种分组
	symbolTrades := make(map[string][]MatchedTrade)
	for _, trade := range trades {
		symbol := trade.OpenTrade.Symbol
		symbolTrades[symbol] = append(symbolTrades[symbol], trade)
	}

	// 初始化市场数据客户端
	apiClient := market.GetMarketDataClient()
	if apiClient == nil {
		return benchmarks, fmt.Errorf("无法获取市场数据客户端")
	}

	// 为每个币种计算基准
	for symbol, symbolTradeList := range symbolTrades {
		if len(symbolTradeList) == 0 {
			continue
		}

		// 获取时间范围（找到最早的开仓和最晚的平仓）
		startTime := symbolTradeList[0].OpenTrade.Timestamp
		endTime := symbolTradeList[0].CloseTrade.Timestamp
		for _, trade := range symbolTradeList {
			if trade.OpenTrade.Timestamp.Before(startTime) {
				startTime = trade.OpenTrade.Timestamp
			}
			if trade.CloseTrade.Timestamp.After(endTime) {
				endTime = trade.CloseTrade.Timestamp
			}
		}

		// 扩展时间范围以获取更多数据
		startTime = startTime.Add(-24 * time.Hour)
		endTime = endTime.Add(24 * time.Hour)

		// 计算需要多少根1小时K线（Hyperliquid API需要limit参数）
		duration := endTime.Sub(startTime)
		hoursNeeded := int(duration.Hours()) + 48 // 额外48小时缓冲
		if hoursNeeded < 100 {
			hoursNeeded = 100 // 最少100根
		}
		if hoursNeeded > 1000 {
			hoursNeeded = 1000 // 最多1000根
		}

		// 获取历史K线数据（使用1小时K线）
		log.Printf("  获取 %s 的历史K线 (需要约 %d 小时的数据)...", symbol, hoursNeeded)
		klines, err := apiClient.GetKlines(symbol, "1h", hoursNeeded)
		if err != nil {
			log.Printf("⚠️ 获取 %s 的历史K线失败: %v", symbol, err)
			continue
		}

		if len(klines) == 0 {
			log.Printf("⚠️ %s 没有K线数据", symbol)
			continue
		}

		// 找到对应时间段的K线
		var relevantKlines []market.Kline
		for _, kline := range klines {
			// Kline.OpenTime可能是毫秒或秒，需要判断
			var klineTime time.Time
			if kline.OpenTime > 1e12 {
				// 毫秒时间戳
				klineTime = time.Unix(kline.OpenTime/1000, 0)
			} else {
				// 秒时间戳
				klineTime = time.Unix(kline.OpenTime, 0)
			}
			// 包含边界：klineTime >= startTime && klineTime <= endTime
			if (klineTime.Equal(startTime) || klineTime.After(startTime)) && 
			   (klineTime.Equal(endTime) || klineTime.Before(endTime)) {
				relevantKlines = append(relevantKlines, kline)
			}
		}

		if len(relevantKlines) == 0 {
			// 如果没有精确匹配，使用所有K线（可能是时间戳格式问题）
			log.Printf("⚠️ %s 未找到精确匹配的K线，使用所有获取到的K线", symbol)
			relevantKlines = klines
		} else {
			log.Printf("  ✓ %s 找到 %d 根相关K线（从 %d 根中）", symbol, len(relevantKlines), len(klines))
		}

		if len(relevantKlines) < 2 {
			log.Printf("⚠️ %s 的K线数据不足", symbol)
			continue
		}

		// 计算基准指标
		benchmark := &MarketBenchmark{
			Symbol:    symbol,
			StartTime: startTime,
			EndTime:   endTime,
		}

		// 获取起始和结束价格
		benchmark.StartPrice = relevantKlines[0].Close
		benchmark.EndPrice = relevantKlines[len(relevantKlines)-1].Close

		// 计算买入持有收益率
		if benchmark.StartPrice > 0 {
			benchmark.PriceChange = ((benchmark.EndPrice - benchmark.StartPrice) / benchmark.StartPrice) * 100
			benchmark.BuyHoldReturn = benchmark.PriceChange
		}

		// 计算做多策略（假设在开始时间买入，结束时间卖出）
		benchmark.LongOnlyReturn = benchmark.BuyHoldReturn

		// 计算做空策略（假设在开始时间做空，结束时间平仓）
		benchmark.ShortOnlyReturn = -benchmark.BuyHoldReturn

		// 计算最大回撤和波动率
		var prices []float64
		var peak float64
		var maxDrawdown float64
		for _, kline := range relevantKlines {
			price := kline.Close
			prices = append(prices, price)
			if price > peak {
				peak = price
			}
			drawdown := ((peak - price) / peak) * 100
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
		benchmark.MaxDrawdown = maxDrawdown

		// 计算波动率（标准差）
		if len(prices) > 1 {
			mean := 0.0
			for _, p := range prices {
				mean += p
			}
			mean /= float64(len(prices))

			variance := 0.0
			for _, p := range prices {
				variance += (p - mean) * (p - mean)
			}
			variance /= float64(len(prices))
			benchmark.Volatility = math.Sqrt(variance) / mean * 100 // 百分比波动率
		}

		benchmarks[symbol] = benchmark
	}

	return benchmarks, nil
}

func checkRuleViolations(decisions []DecisionRecord) []*RuleViolation {
	var violations []*RuleViolation

	for _, decision := range decisions {
		// 跳过空的DecisionJSON
		if decision.DecisionJSON == "" {
			continue
		}

		// 解析DecisionJSON - 格式是数组 [{...}]
		var decisionArray []struct {
			Symbol     string  `json:"symbol"`
			Action     string  `json:"action"`
			StopLoss   float64 `json:"stop_loss"`
			TakeProfit float64 `json:"take_profit"`
			Leverage   int     `json:"leverage"`
			Confidence int     `json:"confidence"`
		}

		if err := json.Unmarshal([]byte(decision.DecisionJSON), &decisionArray); err != nil {
			// 如果解析失败，跳过这条记录
			continue
		}

		// 检查每个决策
		for _, d := range decisionArray {
			if d.Action != "open_long" && d.Action != "open_short" {
				continue
			}

			// 查找对应的执行动作
			var executedAction *DecisionAction
			for _, action := range decision.Decisions {
				if action.Symbol == d.Symbol && action.Action == d.Action && action.Success {
					executedAction = &action
					break
				}
			}

			if executedAction == nil {
				continue
			}

			entryPrice := executedAction.Price

			// 1. 检查风险回报比
			if d.StopLoss > 0 && d.TakeProfit > 0 && entryPrice > 0 {
				var risk, reward float64
				if d.Action == "open_long" {
					risk = entryPrice - d.StopLoss
					reward = d.TakeProfit - entryPrice
				} else {
					risk = d.StopLoss - entryPrice
					reward = entryPrice - d.TakeProfit
				}

				if risk > 0 && reward > 0 {
					ratio := reward / risk
					if ratio < 3.0 {
						violations = append(violations, &RuleViolation{
							CycleNumber: decision.CycleNumber,
							Symbol:      d.Symbol,
							RuleID:      "risk_reward_ratio",
							RuleName:    "风险回报比要求",
							Severity:    "critical",
							Description: fmt.Sprintf("风险回报比 %.2f:1 低于要求 3.0:1", ratio),
							Details: map[string]interface{}{
								"ratio":      ratio,
								"risk":       risk,
								"reward":     reward,
								"entry_price": entryPrice,
								"stop_loss":   d.StopLoss,
								"take_profit": d.TakeProfit,
							},
							DecisionJSON: decision.DecisionJSON,
						})
					}
				}
			}

			// 2. 检查止损方向
			if d.Action == "open_long" {
				if d.StopLoss >= entryPrice {
					violations = append(violations, &RuleViolation{
						CycleNumber: decision.CycleNumber,
						Symbol:      d.Symbol,
						RuleID:      "stop_loss_direction",
						RuleName:    "止损方向错误",
						Severity:    "critical",
						Description: fmt.Sprintf("做多时止损 %.2f 应低于入场价 %.2f", d.StopLoss, entryPrice),
						Details: map[string]interface{}{
							"action":      d.Action,
							"entry_price": entryPrice,
							"stop_loss":   d.StopLoss,
						},
						DecisionJSON: decision.DecisionJSON,
					})
				}
			} else if d.Action == "open_short" {
				if d.StopLoss <= entryPrice {
					violations = append(violations, &RuleViolation{
						CycleNumber: decision.CycleNumber,
						Symbol:      d.Symbol,
						RuleID:      "stop_loss_direction",
						RuleName:    "止损方向错误",
						Severity:    "critical",
						Description: fmt.Sprintf("做空时止损 %.2f 应高于入场价 %.2f", d.StopLoss, entryPrice),
						Details: map[string]interface{}{
							"action":      d.Action,
							"entry_price": entryPrice,
							"stop_loss":   d.StopLoss,
						},
						DecisionJSON: decision.DecisionJSON,
					})
				}
			}

			// 3. 检查置信度
			if d.Confidence < 70 {
				violations = append(violations, &RuleViolation{
					CycleNumber: decision.CycleNumber,
					Symbol:      d.Symbol,
					RuleID:      "confidence_requirement",
					RuleName:    "置信度不足",
					Severity:    "high",
					Description: fmt.Sprintf("置信度 %d 低于要求 70", d.Confidence),
					Details: map[string]interface{}{
						"confidence": d.Confidence,
					},
					DecisionJSON: decision.DecisionJSON,
				})
			}

			// 4. 检查杠杆限制
			if d.Leverage > 25 {
				violations = append(violations, &RuleViolation{
					CycleNumber: decision.CycleNumber,
					Symbol:      d.Symbol,
					RuleID:      "leverage_limit",
					RuleName:    "杠杆超限",
					Severity:    "critical",
					Description: fmt.Sprintf("杠杆 %d 超过限制 25", d.Leverage),
					Details: map[string]interface{}{
						"leverage": d.Leverage,
					},
					DecisionJSON: decision.DecisionJSON,
				})
			}
		}
	}

	return violations
}

func analyzeDecisionLogic(decisions []DecisionRecord, trades []MatchedTrade) []*DecisionLogicAnalysis {
	var analyses []*DecisionLogicAnalysis

	// 创建交易映射以便快速查找
	tradeMap := make(map[int][]MatchedTrade)
	for _, trade := range trades {
		tradeMap[trade.CycleNumber] = append(tradeMap[trade.CycleNumber], trade)
	}

	for _, decision := range decisions {
		if decision.CoTTrace == "" {
			continue
		}

		// 查找该cycle的交易
		cycleTrades := tradeMap[decision.CycleNumber]
		if len(cycleTrades) == 0 {
			continue
		}

		for _, trade := range cycleTrades {
			analysis := &DecisionLogicAnalysis{
				CycleNumber:  decision.CycleNumber,
				Symbol:       trade.OpenTrade.Symbol,
				Action:       trade.OpenAction.Action,
				CoTTrace:     decision.CoTTrace,
				DecisionJSON: decision.DecisionJSON,
			}

			// 分析CoTTrace质量（简化版）
			cotLower := strings.ToLower(decision.CoTTrace)
			
			// 检查是否有明确的风险回报比计算
			if strings.Contains(cotLower, "risk") && strings.Contains(cotLower, "reward") {
				analysis.Strengths = append(analysis.Strengths, "明确计算了风险回报比")
			} else {
				analysis.Issues = append(analysis.Issues, "未明确计算风险回报比")
			}

			// 检查是否有技术分析
			if strings.Contains(cotLower, "rsi") || strings.Contains(cotLower, "macd") || strings.Contains(cotLower, "ema") {
				analysis.Strengths = append(analysis.Strengths, "使用了技术指标分析")
			}

			// 检查是否有趋势分析
			if strings.Contains(cotLower, "trend") || strings.Contains(cotLower, "趋势") {
				analysis.Strengths = append(analysis.Strengths, "分析了市场趋势")
			}

			// 检查是否有止损止盈设置
			if strings.Contains(cotLower, "stop") || strings.Contains(cotLower, "止损") {
				analysis.Strengths = append(analysis.Strengths, "考虑了止损设置")
			} else {
				analysis.Issues = append(analysis.Issues, "未明确考虑止损")
			}

			// 判断逻辑质量
			if len(analysis.Issues) == 0 && len(analysis.Strengths) >= 3 {
				analysis.LogicQuality = "good"
			} else if len(analysis.Issues) > len(analysis.Strengths) {
				analysis.LogicQuality = "bad"
			} else {
				analysis.LogicQuality = "neutral"
			}

			analyses = append(analyses, analysis)
		}
	}

	return analyses
}

func analyzeAllSymbols(trades []MatchedTrade, benchmarks map[string]*MarketBenchmark) map[string]*SymbolAnalysis {
	// 按币种分组
	symbolTrades := make(map[string][]MatchedTrade)
	for _, trade := range trades {
		symbol := trade.OpenTrade.Symbol
		symbolTrades[symbol] = append(symbolTrades[symbol], trade)
	}

	analyses := make(map[string]*SymbolAnalysis)

	// 对每个币种进行分析
	for symbol, symbolTradeList := range symbolTrades {
		analysis := analyzeSymbol(symbol, symbolTradeList, benchmarks)
		if analysis != nil {
			analyses[symbol] = analysis
		}
	}

	return analyses
}

func analyzeSymbol(symbol string, trades []MatchedTrade, benchmarks map[string]*MarketBenchmark) *SymbolAnalysis {
	if len(trades) == 0 {
		return nil
	}

	analysis := &SymbolAnalysis{
		Symbol:          symbol,
		TotalTrades:     len(trades),
		DetailedTrades:  []*SymbolTradeDetail{},
		EntryTimingIssues: []string{},
		StopLossIssues:    []string{},
	}

	// 计算总盈亏和胜率
	var totalPnL float64
	var winCount int
	for _, trade := range trades {
		totalPnL += trade.PnL
		if trade.PnL > 0 {
			winCount++
		}
	}
	analysis.TotalPnL = totalPnL
	analysis.WinRate = float64(winCount) / float64(len(trades)) * 100

	// 检查是否全部止损
	allStopLoss := true
	for _, trade := range trades {
		if !trade.CloseAction.WasStopLoss {
			allStopLoss = false
			break
		}
	}
	analysis.AllStopLoss = allStopLoss

	// 计算平均价格
	var totalEntry, totalExit float64
	for _, trade := range trades {
		totalEntry += trade.OpenTrade.ExecutionPrice
		totalExit += trade.CloseTrade.ExecutionPrice
	}
	if len(trades) > 0 {
		analysis.AverageEntryPrice = totalEntry / float64(len(trades))
		analysis.AverageExitPrice = totalExit / float64(len(trades))
	}

	// 获取市场基准
	if benchmark, exists := benchmarks[symbol]; exists {
		analysis.MarketBenchmark = benchmark
		if benchmark.PriceChange < -5 {
			analysis.MarketTrend = "down"
		} else if benchmark.PriceChange > 5 {
			analysis.MarketTrend = "up"
		} else {
			analysis.MarketTrend = "sideways"
		}
	}

	// 分析入场时机问题
	analyzeEntryTiming(analysis, trades)

	// 分析止损问题
	if analysis.AllStopLoss {
		analysis.StopLossIssues = append(analysis.StopLossIssues, 
			fmt.Sprintf("所有%d笔交易都触发了止损", len(trades)))
		analysis.StopLossIssues = append(analysis.StopLossIssues, 
			"止损设置可能过窄或入场时机不佳")
	} else {
		stopLossCount := 0
		for _, trade := range trades {
			if trade.CloseAction.WasStopLoss {
				stopLossCount++
			}
		}
		if stopLossCount > len(trades)/2 {
			analysis.StopLossIssues = append(analysis.StopLossIssues,
				fmt.Sprintf("止损触发率过高：%d/%d (%.1f%%)", stopLossCount, len(trades), 
					float64(stopLossCount)/float64(len(trades))*100))
		}
	}

	return analysis
}

func analyzeEntryTiming(analysis *SymbolAnalysis, trades []MatchedTrade) {
	if analysis.MarketBenchmark == nil {
		return
	}

	benchmark := analysis.MarketBenchmark
	
	// 1. 市场趋势分析
	if analysis.MarketTrend == "down" {
		// 检查是否在下跌趋势中做多
		longCount := 0
		for _, trade := range trades {
			if trade.OpenAction.Action == "open_long" {
				longCount++
			}
		}
		if longCount > 0 {
			analysis.EntryTimingIssues = append(analysis.EntryTimingIssues,
				fmt.Sprintf("在下跌趋势中做多：%d笔做多交易，市场期间下跌%.2f%%", 
					longCount, benchmark.PriceChange))
		}
	} else if analysis.MarketTrend == "sideways" {
		// 横盘趋势中做多也可能有问题
		longCount := 0
		var longPnL float64
		for _, trade := range trades {
			if trade.OpenAction.Action == "open_long" {
				longCount++
				longPnL += trade.PnL
			}
		}
		if longCount > 0 && longPnL < 0 {
			analysis.EntryTimingIssues = append(analysis.EntryTimingIssues,
				fmt.Sprintf("在横盘趋势中做多表现不佳：%d笔做多交易，总亏损%.2f USDT，市场期间变动%.2f%%",
					longCount, longPnL, benchmark.PriceChange))
		}
	}

	// 2. 入场价格与市场趋势的对比
	if benchmark.StartPrice > 0 && benchmark.EndPrice > 0 {
		// 计算平均入场价格相对于市场起始价格的位置
		avgEntryRelativeToStart := (analysis.AverageEntryPrice - benchmark.StartPrice) / benchmark.StartPrice * 100
		
		// 如果平均入场价格高于起始价格很多，且市场下跌，说明在高位入场
		if avgEntryRelativeToStart > 2 && benchmark.PriceChange < 0 {
			analysis.EntryTimingIssues = append(analysis.EntryTimingIssues,
				fmt.Sprintf("入场价格偏高：平均入场价较市场起始价高%.2f%%，但市场期间下跌%.2f%%",
					avgEntryRelativeToStart, benchmark.PriceChange))
		}
	}

	// 3. 分析每笔交易的入场时机
	badTimingCount := 0
	for _, trade := range trades {
		// 如果做多但市场下跌，或者做空但市场上涨
		if trade.OpenAction.Action == "open_long" && benchmark.PriceChange < -3 {
			badTimingCount++
		} else if trade.OpenAction.Action == "open_short" && benchmark.PriceChange > 3 {
			badTimingCount++
		}
	}
	
	if badTimingCount > len(trades)/3 {
		analysis.EntryTimingIssues = append(analysis.EntryTimingIssues,
			fmt.Sprintf("入场时机不佳的交易占比高：%d/%d (%.1f%%)",
				badTimingCount, len(trades), float64(badTimingCount)/float64(len(trades))*100))
	}

	// 4. 如果没有任何入场时机问题，但表现很差，也添加提示
	if len(analysis.EntryTimingIssues) == 0 && analysis.TotalPnL < 0 && analysis.WinRate < 30 {
		analysis.EntryTimingIssues = append(analysis.EntryTimingIssues,
			"虽然未发现明显的趋势冲突，但整体表现较差，可能存在入场时机选择问题")
	}
}

func compareAIvsBenchmark(trades []MatchedTrade, benchmarks map[string]*MarketBenchmark) *AIBenchmarkComparison {
	comparison := &AIBenchmarkComparison{
		TotalTrades: len(trades),
	}

	var totalBuyHoldPnL float64
	var totalLongOnlyPnL float64

	for _, trade := range trades {
		comparison.AITotalPnL += trade.PnL

		// 获取该币种的基准
		if benchmark, exists := benchmarks[trade.OpenTrade.Symbol]; exists {
			// 计算买入持有在该交易期间的收益
			// 简化：使用整个期间的收益率
			tradeDuration := trade.Duration.Hours()
			totalDuration := benchmark.EndTime.Sub(benchmark.StartTime).Hours()
			if totalDuration > 0 {
				dailyReturn := benchmark.BuyHoldReturn / (totalDuration / 24)
				tradeReturn := dailyReturn * (tradeDuration / 24)
				// 简化计算：假设每笔交易价值100 USDT
				totalBuyHoldPnL += tradeReturn * 100 / 100
			}
		}
	}

	comparison.BuyHoldTotalPnL = totalBuyHoldPnL
	comparison.LongOnlyTotalPnL = totalLongOnlyPnL
	comparison.Outperformance = comparison.AITotalPnL - comparison.BuyHoldTotalPnL

	if comparison.BuyHoldTotalPnL != 0 {
		comparison.OutperformancePct = (comparison.Outperformance / math.Abs(comparison.BuyHoldTotalPnL)) * 100
	}

	// 逐笔对比
	for _, trade := range trades {
		if benchmark, exists := benchmarks[trade.OpenTrade.Symbol]; exists {
			// 简化对比
			if trade.PnL > benchmark.BuyHoldReturn {
				comparison.BetterThanBuyHold++
			} else {
				comparison.WorseThanBuyHold++
			}
		}
	}

	return comparison
}

func generatePhase3Report(phase1Data *Phase1Result, phase2Data *Phase2Result, analysis *Phase3Analysis) (string, error) {
	reportPath := fmt.Sprintf("phase3_report_%d_%d.md", phase1Data.StartCycle, phase1Data.EndCycle)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Phase 3: 决策质量分析报告（含市场基准）\n\n")
	fmt.Fprintf(file, "**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**基于Phase 1和Phase 2数据**: Cycle %d - %d\n\n", phase1Data.StartCycle, phase1Data.EndCycle)

	// 1. 市场基准分析
	fmt.Fprintf(file, "## 1. 市场基准分析\n\n")
	fmt.Fprintf(file, "| 币种 | 买入持有收益率 | 做多策略收益率 | 做空策略收益率 | 价格变动 | 最大回撤 | 波动率 |\n")
	fmt.Fprintf(file, "|------|----------------|----------------|----------------|----------|----------|--------|\n")

	var symbols []string
	for symbol := range analysis.MarketBenchmark {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	for _, symbol := range symbols {
		bm := analysis.MarketBenchmark[symbol]
		fmt.Fprintf(file, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n",
			symbol, bm.BuyHoldReturn, bm.LongOnlyReturn, bm.ShortOnlyReturn,
			bm.PriceChange, bm.MaxDrawdown, bm.Volatility)
	}

	// 2. AI vs Benchmark对比
	fmt.Fprintf(file, "\n## 2. AI决策 vs 市场基准对比\n\n")
	if analysis.AIvsBenchmark != nil {
		ab := analysis.AIvsBenchmark
		fmt.Fprintf(file, "- **AI总盈亏**: %.2f USDT\n", ab.AITotalPnL)
		fmt.Fprintf(file, "- **买入持有总盈亏**: %.2f USDT\n", ab.BuyHoldTotalPnL)
		fmt.Fprintf(file, "- **超额收益**: %.2f USDT (%.2f%%)\n", ab.Outperformance, ab.OutperformancePct)
		fmt.Fprintf(file, "- **优于买入持有**: %d 笔\n", ab.BetterThanBuyHold)
		fmt.Fprintf(file, "- **劣于买入持有**: %d 笔\n\n", ab.WorseThanBuyHold)
	}

	// 3. 规则违反分析
	fmt.Fprintf(file, "## 3. 规则违反分析\n\n")
	fmt.Fprintf(file, "| Cycle | 币种 | 规则 | 严重程度 | 描述 |\n")
	fmt.Fprintf(file, "|-------|------|------|----------|------|\n")

	// 按规则类型分组
	violationByRule := make(map[string][]*RuleViolation)
	for _, v := range analysis.RuleViolations {
		violationByRule[v.RuleID] = append(violationByRule[v.RuleID], v)
	}

	for _, v := range analysis.RuleViolations {
		fmt.Fprintf(file, "| %d | %s | %s | %s | %s |\n",
			v.CycleNumber, v.Symbol, v.RuleName, v.Severity, v.Description)
	}

	fmt.Fprintf(file, "\n### 规则违反统计\n\n")
	for _, violations := range violationByRule {
		if len(violations) > 0 {
			fmt.Fprintf(file, "- **%s**: %d 次违反\n", violations[0].RuleName, len(violations))
		}
	}

	// 4. 决策逻辑分析摘要
	fmt.Fprintf(file, "\n## 4. 决策逻辑分析摘要\n\n")
	var goodLogic, badLogic, neutralLogic int
	for _, dl := range analysis.DecisionLogic {
		switch dl.LogicQuality {
		case "good":
			goodLogic++
		case "bad":
			badLogic++
		default:
			neutralLogic++
		}
	}
	totalLogic := len(analysis.DecisionLogic)
	if totalLogic > 0 {
		fmt.Fprintf(file, "- **良好逻辑**: %d (%.1f%%)\n", goodLogic, float64(goodLogic)/float64(totalLogic)*100)
		fmt.Fprintf(file, "- **不良逻辑**: %d (%.1f%%)\n", badLogic, float64(badLogic)/float64(totalLogic)*100)
		fmt.Fprintf(file, "- **中性逻辑**: %d (%.1f%%)\n\n", neutralLogic, float64(neutralLogic)/float64(totalLogic)*100)
	}

	// 5. 各币种专项分析
	fmt.Fprintf(file, "## 5. 各币种专项分析\n\n")
	
	// 按总盈亏排序（最差的在前）
	var analysisSymbols []string
	for symbol := range analysis.SymbolAnalyses {
		analysisSymbols = append(analysisSymbols, symbol)
	}
	sort.Slice(analysisSymbols, func(i, j int) bool {
		return analysis.SymbolAnalyses[analysisSymbols[i]].TotalPnL < analysis.SymbolAnalyses[analysisSymbols[j]].TotalPnL
	})

	for _, symbol := range analysisSymbols {
		sa := analysis.SymbolAnalyses[symbol]
		fmt.Fprintf(file, "### %s 专项分析\n\n", symbol)
		fmt.Fprintf(file, "#### 核心问题\n\n")
		fmt.Fprintf(file, "- **总交易数**: %d\n", sa.TotalTrades)
		fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", sa.TotalPnL)
		fmt.Fprintf(file, "- **胜率**: %.2f%%\n", sa.WinRate)
		fmt.Fprintf(file, "- **全部止损**: %v\n", sa.AllStopLoss)
		fmt.Fprintf(file, "- **市场趋势**: %s\n", sa.MarketTrend)
		if sa.MarketBenchmark != nil {
			fmt.Fprintf(file, "- **市场收益率**: %.2f%%\n", sa.MarketBenchmark.BuyHoldReturn)
			fmt.Fprintf(file, "- **平均入场价**: %.2f\n", sa.AverageEntryPrice)
			fmt.Fprintf(file, "- **平均出场价**: %.2f\n", sa.AverageExitPrice)
		}

		fmt.Fprintf(file, "\n#### 入场时机问题\n\n")
		if len(sa.EntryTimingIssues) > 0 {
			for _, issue := range sa.EntryTimingIssues {
				fmt.Fprintf(file, "- %s\n", issue)
			}
		} else {
			fmt.Fprintf(file, "- 未发现明显的入场时机问题\n")
		}

		fmt.Fprintf(file, "\n#### 止损设置问题\n\n")
		if len(sa.StopLossIssues) > 0 {
			for _, issue := range sa.StopLossIssues {
				fmt.Fprintf(file, "- %s\n", issue)
			}
		} else {
			fmt.Fprintf(file, "- 止损设置正常\n")
		}
		fmt.Fprintf(file, "\n")
	}

	// 6. 关键发现
	fmt.Fprintf(file, "## 6. 关键发现\n\n")
	
	// 6.1 市场基准对比总结
	fmt.Fprintf(file, "### 6.1 市场基准对比\n\n")
	if analysis.AIvsBenchmark != nil {
		ab := analysis.AIvsBenchmark
		fmt.Fprintf(file, "- **AI总盈亏**: %.2f USDT\n", ab.AITotalPnL)
		fmt.Fprintf(file, "- **买入持有总盈亏**: %.2f USDT\n", ab.BuyHoldTotalPnL)
		fmt.Fprintf(file, "- **超额收益**: %.2f USDT (%.2f%%)\n", ab.Outperformance, ab.OutperformancePct)
		if ab.Outperformance < 0 {
			fmt.Fprintf(file, "- ⚠️ **AI表现劣于买入持有策略**，超额亏损%.2f USDT\n", -ab.Outperformance)
		}
		fmt.Fprintf(file, "- **优于买入持有**: %d 笔 (%.1f%%)\n", ab.BetterThanBuyHold, 
			float64(ab.BetterThanBuyHold)/float64(ab.TotalTrades)*100)
		fmt.Fprintf(file, "- **劣于买入持有**: %d 笔 (%.1f%%)\n\n", ab.WorseThanBuyHold,
			float64(ab.WorseThanBuyHold)/float64(ab.TotalTrades)*100)
	}
	
	// 6.2 规则违反总结
	fmt.Fprintf(file, "### 6.2 规则违反分析\n\n")
	if len(analysis.RuleViolations) > 0 {
		// 按规则类型统计
		violationByRule := make(map[string][]*RuleViolation)
		for _, v := range analysis.RuleViolations {
			violationByRule[v.RuleID] = append(violationByRule[v.RuleID], v)
		}
		
		fmt.Fprintf(file, "- **总违反次数**: %d\n", len(analysis.RuleViolations))
		for _, violations := range violationByRule {
			if len(violations) > 0 {
				fmt.Fprintf(file, "- **%s**: %d 次违反", violations[0].RuleName, len(violations))
				// 统计涉及的币种
				symbolSet := make(map[string]bool)
				for _, v := range violations {
					symbolSet[v.Symbol] = true
				}
				if len(symbolSet) > 0 {
					var symbols []string
					for s := range symbolSet {
						symbols = append(symbols, s)
					}
					fmt.Fprintf(file, " (涉及币种: %s)", strings.Join(symbols, ", "))
				}
				fmt.Fprintf(file, "\n")
			}
		}
		fmt.Fprintf(file, "- ⚠️ **风险回报比违反最严重**：%d次违反，说明AI在部分交易中未严格遵守3.0:1的风险回报比要求\n\n", 
			len(violationByRule["risk_reward_ratio"]))
	} else {
		fmt.Fprintf(file, "- ✅ 未发现规则违反\n\n")
	}
	
	// 6.3 决策逻辑质量总结
	fmt.Fprintf(file, "### 6.3 决策逻辑质量\n\n")
	// 重用前面计算的变量
	if totalLogic > 0 {
		fmt.Fprintf(file, "- **良好逻辑**: %d (%.1f%%)\n", goodLogic, float64(goodLogic)/float64(totalLogic)*100)
		fmt.Fprintf(file, "- **不良逻辑**: %d (%.1f%%)\n", badLogic, float64(badLogic)/float64(totalLogic)*100)
		fmt.Fprintf(file, "- **中性逻辑**: %d (%.1f%%)\n", neutralLogic, float64(neutralLogic)/float64(totalLogic)*100)
		if badLogic > goodLogic {
			fmt.Fprintf(file, "- ⚠️ **不良逻辑占比高于良好逻辑**，说明AI的决策思维链质量有待提升\n\n")
		} else {
			fmt.Fprintf(file, "\n")
		}
	}
	
	// 6.4 各币种问题总结
	fmt.Fprintf(file, "### 6.4 各币种表现总结\n\n")
	if len(analysis.SymbolAnalyses) > 0 {
		// 按总盈亏排序
		var symbols []string
		for symbol := range analysis.SymbolAnalyses {
			symbols = append(symbols, symbol)
		}
		sort.Slice(symbols, func(i, j int) bool {
			return analysis.SymbolAnalyses[symbols[i]].TotalPnL < analysis.SymbolAnalyses[symbols[j]].TotalPnL
		})
		
		for _, symbol := range symbols {
			sa := analysis.SymbolAnalyses[symbol]
			fmt.Fprintf(file, "#### %s\n\n", symbol)
			fmt.Fprintf(file, "- **交易数**: %d\n", sa.TotalTrades)
			fmt.Fprintf(file, "- **总盈亏**: %.2f USDT\n", sa.TotalPnL)
			fmt.Fprintf(file, "- **胜率**: %.2f%%\n", sa.WinRate)
			fmt.Fprintf(file, "- **市场趋势**: %s", sa.MarketTrend)
			if sa.MarketBenchmark != nil {
				fmt.Fprintf(file, " (市场收益率: %.2f%%)\n", sa.MarketBenchmark.BuyHoldReturn)
			} else {
				fmt.Fprintf(file, "\n")
			}
			
			// 入场时机问题
			if len(sa.EntryTimingIssues) > 0 {
				fmt.Fprintf(file, "- **入场时机问题**: ")
				for i, issue := range sa.EntryTimingIssues {
					if i > 0 {
						fmt.Fprintf(file, "; ")
					}
					fmt.Fprintf(file, "%s", issue)
				}
				fmt.Fprintf(file, "\n")
			}
			
			// 止损问题
			if len(sa.StopLossIssues) > 0 {
				fmt.Fprintf(file, "- **止损问题**: ")
				for i, issue := range sa.StopLossIssues {
					if i > 0 {
						fmt.Fprintf(file, "; ")
					}
					fmt.Fprintf(file, "%s", issue)
				}
				fmt.Fprintf(file, "\n")
			}
			fmt.Fprintf(file, "\n")
		}
	}
	
	// 6.5 综合根因分析
	fmt.Fprintf(file, "### 6.5 综合根因分析\n\n")
	
	// 统计入场时机问题
	totalEntryTimingIssues := 0
	for _, sa := range analysis.SymbolAnalyses {
		totalEntryTimingIssues += len(sa.EntryTimingIssues)
	}
	
	// 统计止损问题
	totalStopLossIssues := 0
	allStopLossCount := 0
	for _, sa := range analysis.SymbolAnalyses {
		if sa.AllStopLoss {
			allStopLossCount++
		}
		totalStopLossIssues += len(sa.StopLossIssues)
	}
	
	fmt.Fprintf(file, "#### 主要问题模式\n\n")
	fmt.Fprintf(file, "1. **入场时机问题**：")
	if totalEntryTimingIssues > 0 {
		fmt.Fprintf(file, "发现%d个入场时机问题，主要表现为：\n", totalEntryTimingIssues)
		fmt.Fprintf(file, "   - 在下跌/横盘趋势中做多\n")
		fmt.Fprintf(file, "   - 入场价格偏高\n")
		fmt.Fprintf(file, "   - 入场时机不佳的交易占比高\n")
	} else {
		fmt.Fprintf(file, "未发现明显的入场时机问题\n")
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "2. **止损设置问题**：")
	if totalStopLossIssues > 0 {
		fmt.Fprintf(file, "发现%d个止损相关问题，主要表现为：\n", totalStopLossIssues)
		if allStopLossCount > 0 {
			fmt.Fprintf(file, "   - %d个币种的所有交易都触发止损\n", allStopLossCount)
		}
		fmt.Fprintf(file, "   - 止损设置可能过窄\n")
		fmt.Fprintf(file, "   - 止损触发率过高\n")
	} else {
		fmt.Fprintf(file, "止损设置正常\n")
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "3. **规则遵守问题**：")
	if len(analysis.RuleViolations) > 0 {
		fmt.Fprintf(file, "发现%d次规则违反，主要是风险回报比低于3.0:1的要求\n", len(analysis.RuleViolations))
	} else {
		fmt.Fprintf(file, "未发现规则违反\n")
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "4. **决策逻辑质量**：")
	if badLogic > goodLogic {
		fmt.Fprintf(file, "不良逻辑占比(%.1f%%)高于良好逻辑(%.1f%%)，决策思维链质量需要提升\n", 
			float64(badLogic)/float64(totalLogic)*100, float64(goodLogic)/float64(totalLogic)*100)
	} else {
		fmt.Fprintf(file, "决策逻辑质量整体良好\n")
	}
	fmt.Fprintf(file, "\n")
	
	// 6.6 核心结论
	fmt.Fprintf(file, "### 6.6 核心结论\n\n")
	fmt.Fprintf(file, "导致负收益的主要原因：\n\n")
	fmt.Fprintf(file, "1. **入场时机选择不当**：在下跌/横盘趋势中大量做多，导致入场时机不佳的交易占比极高\n")
	fmt.Fprintf(file, "2. **止损设置过窄**：多个币种的所有交易都触发止损，说明止损设置可能过于严格\n")
	if len(analysis.RuleViolations) > 0 {
		fmt.Fprintf(file, "3. **规则遵守不严格**：存在风险回报比低于3.0:1的情况，未严格遵守交易规则\n")
	}
	if badLogic > goodLogic {
		fmt.Fprintf(file, "4. **决策逻辑质量有待提升**：不良逻辑占比高于良好逻辑，思维链分析质量需要改进\n")
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "\n## 下一步分析建议\n\n")
	fmt.Fprintf(file, "1. **Phase 4**: Prompt问题诊断\n")
	fmt.Fprintf(file, "2. **Phase 5**: 根因分析与修正方案\n")

	return reportPath, nil
}

func printPhase3Summary(analysis *Phase3Analysis) {
	fmt.Println("\n=== Phase 3 分析摘要 ===")
	fmt.Printf("市场基准分析: %d 个币种\n", len(analysis.MarketBenchmark))
	fmt.Printf("规则违反: %d 个\n", len(analysis.RuleViolations))
	fmt.Printf("决策逻辑分析: %d 个\n", len(analysis.DecisionLogic))
	fmt.Printf("币种专项分析: %d 个币种\n", len(analysis.SymbolAnalyses))
	for symbol, sa := range analysis.SymbolAnalyses {
		fmt.Printf("  - %s: %d笔交易, 总盈亏%.2f USDT, 胜率%.1f%%\n", 
			symbol, sa.TotalTrades, sa.TotalPnL, sa.WinRate)
	}
	if analysis.AIvsBenchmark != nil {
		fmt.Printf("AI vs 基准: 超额收益 %.2f USDT\n", analysis.AIvsBenchmark.Outperformance)
	}
	fmt.Printf("\n报告路径: %s\n", analysis.ReportPath)
}


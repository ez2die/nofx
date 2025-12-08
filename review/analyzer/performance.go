package analyzer

import (
	"fmt"
	"log"
	"math"

	"nofx/review"
	"nofx/trade_history"
)

// PerformanceAnalyzer 表现分析器
type PerformanceAnalyzer struct {
	analyticsService review.TradeAnalyticsService
}

// NewPerformanceAnalyzer 创建表现分析器
func NewPerformanceAnalyzer(analyticsService review.TradeAnalyticsService) *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		analyticsService: analyticsService,
	}
}

// AnalyzePerformance 分析交易表现
func (a *PerformanceAnalyzer) AnalyzePerformance(trades []*trade_history.TradeRecord, decisions []*review.DecisionRecord) (*review.PerformanceAnalysis, error) {
	log.Printf("📊 开始分析交易表现 (交易数: %d, 决策数: %d)", len(trades), len(decisions))

	analysis := &review.PerformanceAnalysis{
		SymbolStats: make(map[string]*review.SymbolPerformance),
	}

	// 基础统计
	a.analyzeBasicStats(trades, decisions, analysis)

	// 盈亏统计
	a.analyzePnLStats(trades, analysis)

	// 胜率统计
	a.analyzeWinRateStats(trades, analysis)

	// 费用统计
	a.analyzeFeeStats(trades, analysis)

	// 风险指标
	a.analyzeRiskMetrics(trades, analysis)

	// 币种表现
	a.analyzeSymbolStats(trades, analysis)

	log.Printf("✅ 交易表现分析完成 (总交易数: %d, 胜率: %.2f%%)", analysis.TotalTrades, analysis.WinRate)
	return analysis, nil
}

// analyzeBasicStats 分析基础统计
func (a *PerformanceAnalyzer) analyzeBasicStats(trades []*trade_history.TradeRecord, decisions []*review.DecisionRecord, analysis *review.PerformanceAnalysis) {
	// 统计交易类型
	for _, trade := range trades {
		switch trade.Action {
		case "open_long", "open_short":
			analysis.OpenTrades++
		case "close_long", "close_short":
			analysis.CloseTrades++
		}
	}

	// 配对开仓和平仓，统计完整交易数
	openTrades := make(map[string]*trade_history.TradeRecord) // key = symbol_side
	for _, trade := range trades {
		if trade.Action == "open_long" || trade.Action == "open_short" {
			side := ""
			if trade.Action == "open_long" {
				side = "long"
			} else {
				side = "short"
			}
			key := fmt.Sprintf("%s_%s", trade.Symbol, side)
			openTrades[key] = trade
		}
	}

	for _, trade := range trades {
		if trade.Action == "close_long" || trade.Action == "close_short" {
			side := ""
			if trade.Action == "close_long" {
				side = "long"
			} else {
				side = "short"
			}
			key := fmt.Sprintf("%s_%s", trade.Symbol, side)
			if _, exists := openTrades[key]; exists {
				analysis.CompletedTrades++
			}
		}
	}

	analysis.TotalTrades = analysis.CompletedTrades

	// 决策相关统计（从决策记录中提取）
	analysis.TotalTrades = len(decisions)
}

// analyzePnLStats 分析盈亏统计
func (a *PerformanceAnalyzer) analyzePnLStats(trades []*trade_history.TradeRecord, analysis *review.PerformanceAnalysis) {
	for _, trade := range trades {
		if trade.PnL == nil {
			continue
		}

		pnl := *trade.PnL
		analysis.TotalPnL += pnl

		if pnl > analysis.MaxProfit {
			analysis.MaxProfit = pnl
		}
		if pnl < analysis.MaxLoss {
			analysis.MaxLoss = pnl
		}
	}

	// 计算净盈亏（扣除手续费）
	analysis.NetPnL = analysis.TotalPnL
	for _, trade := range trades {
		analysis.NetPnL -= trade.Fee
	}

	// 计算平均盈亏
	if analysis.CompletedTrades > 0 {
		analysis.AvgPnL = analysis.TotalPnL / float64(analysis.CompletedTrades)
	}
}

// analyzeWinRateStats 分析胜率统计
func (a *PerformanceAnalyzer) analyzeWinRateStats(trades []*trade_history.TradeRecord, analysis *review.PerformanceAnalysis) {
	for _, trade := range trades {
		if trade.PnL == nil {
			continue
		}

		pnl := *trade.PnL
		if pnl > 0 {
			analysis.WinTrades++
		} else if pnl < 0 {
			analysis.LossTrades++
		}
	}

	// 计算胜率
	if analysis.CompletedTrades > 0 {
		analysis.WinRate = (float64(analysis.WinTrades) / float64(analysis.CompletedTrades)) * 100
	}
}

// analyzeFeeStats 分析费用统计
func (a *PerformanceAnalyzer) analyzeFeeStats(trades []*trade_history.TradeRecord, analysis *review.PerformanceAnalysis) {
	for _, trade := range trades {
		analysis.TotalFees += trade.Fee
	}

	// 计算平均费用
	if len(trades) > 0 {
		analysis.AvgFee = analysis.TotalFees / float64(len(trades))
	}

	// 计算费用比率
	if analysis.TotalPnL != 0 {
		analysis.FeeRatio = (analysis.TotalFees / math.Abs(analysis.TotalPnL)) * 100
	}
}

// analyzeRiskMetrics 分析风险指标
func (a *PerformanceAnalyzer) analyzeRiskMetrics(trades []*trade_history.TradeRecord, analysis *review.PerformanceAnalysis) {
	if len(trades) == 0 {
		return
	}

	// 计算最大回撤（简化版本）
	var cumulativePnL float64
	var peak float64
	var maxDrawdown float64

	for _, trade := range trades {
		if trade.PnL == nil {
			continue
		}
		cumulativePnL += *trade.PnL
		if cumulativePnL > peak {
			peak = cumulativePnL
		}
		drawdown := peak - cumulativePnL
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	analysis.MaxDrawdown = maxDrawdown

	// 计算波动率（简化版本：标准差）
	var pnlValues []float64
	for _, trade := range trades {
		if trade.PnL != nil {
			pnlValues = append(pnlValues, *trade.PnL)
		}
	}

	if len(pnlValues) > 1 {
		mean := 0.0
		for _, v := range pnlValues {
			mean += v
		}
		mean /= float64(len(pnlValues))

		variance := 0.0
		for _, v := range pnlValues {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(pnlValues))
		analysis.Volatility = math.Sqrt(variance)
	}

	// 计算Sharpe比率（简化版本）
	if analysis.Volatility > 0 {
		analysis.SharpeRatio = analysis.AvgPnL / analysis.Volatility
	}
}

// analyzeSymbolStats 分析币种表现
func (a *PerformanceAnalyzer) analyzeSymbolStats(trades []*trade_history.TradeRecord, analysis *review.PerformanceAnalysis) {
	// 按币种分组统计
	symbolStats := make(map[string]*review.SymbolPerformance)

	for _, trade := range trades {
		symbol := trade.Symbol
		if _, exists := symbolStats[symbol]; !exists {
			symbolStats[symbol] = &review.SymbolPerformance{
				Symbol: symbol,
			}
		}

		stats := symbolStats[symbol]
		stats.TotalTrades++

		if trade.PnL != nil {
			pnl := *trade.PnL
			stats.TotalPnL += pnl

			if pnl > 0 {
				stats.WinningTrades++
			} else if pnl < 0 {
				stats.LosingTrades++
			}
		}
	}

	// 计算各币种的胜率和平均盈亏
	for symbol, stats := range symbolStats {
		if stats.TotalTrades > 0 {
			stats.WinRate = (float64(stats.WinningTrades) / float64(stats.TotalTrades)) * 100
			stats.AvgPnL = stats.TotalPnL / float64(stats.TotalTrades)
		}
		symbolStats[symbol] = stats
	}

	analysis.SymbolStats = symbolStats
}


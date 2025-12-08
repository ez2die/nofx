package analyzer

import (
	"log"

	"nofx/review"
	"nofx/trade_history"
)

// DEXValidationAnalyzer DEX验证分析器
type DEXValidationAnalyzer struct{}

// NewDEXValidationAnalyzer 创建DEX验证分析器
func NewDEXValidationAnalyzer() *DEXValidationAnalyzer {
	return &DEXValidationAnalyzer{}
}

// AnalyzeExecutionQuality 分析执行质量
func (a *DEXValidationAnalyzer) AnalyzeExecutionQuality(matches []*review.DecisionMatch) (*review.ExecutionQuality, error) {
	log.Printf("📊 开始分析执行质量 (匹配数: %d)", len(matches))

	if len(matches) == 0 {
		return &review.ExecutionQuality{
			SlippageDistribution:      make(map[string]int),
			ExecutionDelayDistribution: make(map[string]int),
		}, nil
	}

	quality := &review.ExecutionQuality{
		TotalMatches:             len(matches),
		SlippageDistribution:      make(map[string]int),
		ExecutionDelayDistribution: make(map[string]int),
	}

	// 计算滑点统计
	totalSlippage := 0.0
	maxSlippage := 0.0
	for _, match := range matches {
		slippage := match.Slippage
		totalSlippage += slippage
		if slippage > maxSlippage {
			maxSlippage = slippage
		}

		// 滑点分布：<0.1%, 0.1-0.5%, 0.5-1%, >1%
		slippagePct := slippage * 100
		if slippagePct < 0.1 {
			quality.SlippageDistribution["<0.1%"]++
		} else if slippagePct < 0.5 {
			quality.SlippageDistribution["0.1-0.5%"]++
		} else if slippagePct < 1.0 {
			quality.SlippageDistribution["0.5-1%"]++
		} else {
			quality.SlippageDistribution[">1%"]++
		}
	}

	quality.AverageSlippage = totalSlippage / float64(len(matches))
	quality.MaxSlippage = maxSlippage

	// 计算执行延迟统计
	totalDelay := int64(0)
	maxDelay := int64(0)
	for _, match := range matches {
		delay := match.ExecutionDelay
		if delay < 0 {
			delay = -delay
		}
		totalDelay += delay
		if delay > maxDelay {
			maxDelay = delay
		}

		// 执行延迟分布：<1s, 1-5s, 5-10s, >10s
		delaySec := delay / 1000
		if delaySec < 1 {
			quality.ExecutionDelayDistribution["<1s"]++
		} else if delaySec < 5 {
			quality.ExecutionDelayDistribution["1-5s"]++
		} else if delaySec < 10 {
			quality.ExecutionDelayDistribution["5-10s"]++
		} else {
			quality.ExecutionDelayDistribution[">10s"]++
		}
	}

	quality.AverageExecutionDelay = totalDelay / int64(len(matches))
	quality.MaxExecutionDelay = maxDelay

	log.Printf("✅ 执行质量分析完成 (平均滑点: %.2f%%, 平均延迟: %dms)", quality.AverageSlippage*100, quality.AverageExecutionDelay)
	return quality, nil
}

// CalculateDEXMetrics 计算DEX验证指标
func (a *DEXValidationAnalyzer) CalculateDEXMetrics(
	matches []*review.DecisionMatch,
	unmatchedDecisions []*review.DecisionRecord,
	unmatchedDEXTrades []trade_history.ExchangeFill,
) (*review.DEXValidationMetrics, error) {
	log.Printf("📊 开始计算DEX验证指标 (匹配: %d, 未匹配决策: %d, 未匹配DEX交易: %d)",
		len(matches), len(unmatchedDecisions), len(unmatchedDEXTrades))

	metrics := &review.DEXValidationMetrics{
		UnmatchedDecisions: len(unmatchedDecisions),
		UnmatchedDEXTrades: len(unmatchedDEXTrades),
	}

	// 计算决策执行率
	totalDecisions := len(matches) + len(unmatchedDecisions)
	if totalDecisions > 0 {
		metrics.DecisionExecutionRate = (float64(len(matches)) / float64(totalDecisions)) * 100
	}

	// 计算平均滑点和平均执行延迟
	if len(matches) > 0 {
		totalSlippage := 0.0
		totalDelay := int64(0)
		totalFeeDiff := 0.0

		for _, match := range matches {
			totalSlippage += match.Slippage
			totalDelay += match.ExecutionDelay
			totalFeeDiff += match.FeeDifference
		}

		metrics.AverageSlippage = totalSlippage / float64(len(matches))
		metrics.AverageExecutionDelay = totalDelay / int64(len(matches))
		metrics.FeeDifference = totalFeeDiff
	}

	// 提取总资金费（从DEX交易中提取，如果有的话）
	// 注意：ExchangeFill中可能没有资金费字段，这里简化处理
	totalFundingFees := 0.0
	for _, dexTrade := range unmatchedDEXTrades {
		// 如果DEX交易中有资金费字段，累加
		// 这里简化处理，实际需要根据DEX数据结构调整
		_ = dexTrade
	}
	metrics.TotalFundingFees = totalFundingFees

	log.Printf("✅ DEX验证指标计算完成 (决策执行率: %.2f%%)", metrics.DecisionExecutionRate)
	return metrics, nil
}


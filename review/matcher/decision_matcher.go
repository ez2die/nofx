package matcher

import (
	"fmt"
	"log"
	"math"
	"time"

	"nofx/review"
	"nofx/trade_history"
)

// DecisionMatcher 决策匹配器
type DecisionMatcher struct {
	timeWindow      time.Duration
	quantityTolerance float64 // 5% (0.05)
	priceTolerance   float64   // 1% (0.01)
}

// NewDecisionMatcher 创建决策匹配器
func NewDecisionMatcher(timeWindow time.Duration, quantityTolerance, priceTolerance float64) *DecisionMatcher {
	if timeWindow <= 0 {
		timeWindow = 10 * time.Minute // 默认10分钟
	}
	if quantityTolerance <= 0 {
		quantityTolerance = 0.05 // 默认5%
	}
	if priceTolerance <= 0 {
		priceTolerance = 0.01 // 默认1%
	}

	return &DecisionMatcher{
		timeWindow:      timeWindow,
		quantityTolerance: quantityTolerance,
		priceTolerance:   priceTolerance,
	}
}

// Match 匹配决策与DEX交易
// 返回：匹配结果、未匹配的决策、未匹配的DEX交易
func (m *DecisionMatcher) Match(decisions []*review.DecisionRecord, dexTrades []trade_history.ExchangeFill) ([]*review.DecisionMatch, []*review.DecisionRecord, []trade_history.ExchangeFill, error) {
	log.Printf("🔍 开始匹配决策与DEX交易 (决策数: %d, DEX交易数: %d)", len(decisions), len(dexTrades))

	var matches []*review.DecisionMatch
	matchedDecisionIndices := make(map[int]bool) // 已匹配的决策索引
	matchedDEXTradeIndices := make(map[int]bool) // 已匹配的DEX交易索引

	// 优先级1: ExchangeOrderID匹配（最准确）
	matches, matchedDecisionIndices, matchedDEXTradeIndices = m.matchByOrderID(decisions, dexTrades, matchedDecisionIndices, matchedDEXTradeIndices)

	// 优先级2: 时间窗口匹配（备选）
	timeWindowMatches, newMatchedDecisions, newMatchedDEXTrades := m.matchByTimeWindow(decisions, dexTrades, matchedDecisionIndices, matchedDEXTradeIndices)
	matches = append(matches, timeWindowMatches...)
	for idx := range newMatchedDecisions {
		matchedDecisionIndices[idx] = true
	}
	for idx := range newMatchedDEXTrades {
		matchedDEXTradeIndices[idx] = true
	}

	// 优先级3: 数量+价格匹配（最后备选）
	quantityPriceMatches, newMatchedDecisions2, newMatchedDEXTrades2 := m.matchByQuantityPrice(decisions, dexTrades, matchedDecisionIndices, matchedDEXTradeIndices)
	matches = append(matches, quantityPriceMatches...)
	for idx := range newMatchedDecisions2 {
		matchedDecisionIndices[idx] = true
	}
	for idx := range newMatchedDEXTrades2 {
		matchedDEXTradeIndices[idx] = true
	}

	// 处理部分匹配：一个决策对应多个DEX交易
	matches = m.mergeMultipleDEXTrades(matches)

	// 收集未匹配的决策和DEX交易
	unmatchedDecisions := make([]*review.DecisionRecord, 0)
	for i, decision := range decisions {
		if !matchedDecisionIndices[i] {
			unmatchedDecisions = append(unmatchedDecisions, decision)
		}
	}

	unmatchedDEXTrades := make([]trade_history.ExchangeFill, 0)
	for i, dexTrade := range dexTrades {
		if !matchedDEXTradeIndices[i] {
			unmatchedDEXTrades = append(unmatchedDEXTrades, dexTrade)
		}
	}

	log.Printf("✅ 匹配完成 (匹配数: %d, 未匹配决策: %d, 未匹配DEX交易: %d)",
		len(matches), len(unmatchedDecisions), len(unmatchedDEXTrades))

	return matches, unmatchedDecisions, unmatchedDEXTrades, nil
}

// matchByOrderID 通过订单ID匹配（优先级1，最准确）
func (m *DecisionMatcher) matchByOrderID(
	decisions []*review.DecisionRecord,
	dexTrades []trade_history.ExchangeFill,
	matchedDecisions map[int]bool,
	matchedDEXTrades map[int]bool,
) ([]*review.DecisionMatch, map[int]bool, map[int]bool) {
	var matches []*review.DecisionMatch
	newMatchedDecisions := make(map[int]bool)
	newMatchedDEXTrades := make(map[int]bool)

	for i, decision := range decisions {
		if matchedDecisions[i] {
			continue
		}

		// 遍历决策中的所有动作
		for _, action := range decision.Decisions {
			if action.OrderID == 0 {
				continue
			}

			// 查找匹配的DEX交易
			for j, dexTrade := range dexTrades {
				if matchedDEXTrades[j] {
					continue
				}

				// 订单ID匹配
				if dexTrade.ExchangeOid == action.OrderID {
					match := m.createMatch(decision, &action, &dexTrade, "order_id", 1.0)
					matches = append(matches, match)
					newMatchedDecisions[i] = true
					newMatchedDEXTrades[j] = true
					break // 一个动作只能匹配一个DEX交易
				}
			}
		}
	}

	return matches, newMatchedDecisions, newMatchedDEXTrades
}

// matchByTimeWindow 通过时间窗口匹配（优先级2）
func (m *DecisionMatcher) matchByTimeWindow(
	decisions []*review.DecisionRecord,
	dexTrades []trade_history.ExchangeFill,
	matchedDecisions map[int]bool,
	matchedDEXTrades map[int]bool,
) ([]*review.DecisionMatch, map[int]bool, map[int]bool) {
	var matches []*review.DecisionMatch
	newMatchedDecisions := make(map[int]bool)
	newMatchedDEXTrades := make(map[int]bool)

	for i, decision := range decisions {
		if matchedDecisions[i] {
			continue
		}

		// 遍历决策中的所有动作
		for _, action := range decision.Decisions {
			if !action.Success {
				continue
			}

			// 提取币种和方向
			symbol := action.Symbol
			side := m.extractSideFromAction(action.Action)

			if symbol == "" || side == "" {
				continue
			}

			// 查找匹配的DEX交易
			for j, dexTrade := range dexTrades {
				if matchedDEXTrades[j] {
					continue
				}

				// 检查币种和方向是否匹配
				if dexTrade.Symbol != symbol {
					continue
				}

				dexSide := m.extractSideFromDEXTrade(dexTrade)
				if dexSide != side {
					continue
				}

				// 检查时间窗口
				timeDiff := action.Timestamp.Sub(dexTrade.Timestamp)
				if timeDiff < 0 {
					timeDiff = -timeDiff
				}

				if timeDiff <= m.timeWindow {
					match := m.createMatch(decision, &action, &dexTrade, "time_window", 0.8)
					matches = append(matches, match)
					newMatchedDecisions[i] = true
					newMatchedDEXTrades[j] = true
					break // 一个动作只能匹配一个DEX交易
				}
			}
		}
	}

	return matches, newMatchedDecisions, newMatchedDEXTrades
}

// matchByQuantityPrice 通过数量+价格匹配（优先级3）
func (m *DecisionMatcher) matchByQuantityPrice(
	decisions []*review.DecisionRecord,
	dexTrades []trade_history.ExchangeFill,
	matchedDecisions map[int]bool,
	matchedDEXTrades map[int]bool,
) ([]*review.DecisionMatch, map[int]bool, map[int]bool) {
	var matches []*review.DecisionMatch
	newMatchedDecisions := make(map[int]bool)
	newMatchedDEXTrades := make(map[int]bool)

	for i, decision := range decisions {
		if matchedDecisions[i] {
			continue
		}

		// 遍历决策中的所有动作
		for _, action := range decision.Decisions {
			if !action.Success || action.Quantity <= 0 || action.Price <= 0 {
				continue
			}

			// 提取币种和方向
			symbol := action.Symbol
			side := m.extractSideFromAction(action.Action)

			if symbol == "" || side == "" {
				continue
			}

			// 查找匹配的DEX交易
			for j, dexTrade := range dexTrades {
				if matchedDEXTrades[j] {
					continue
				}

				// 检查币种和方向是否匹配
				if dexTrade.Symbol != symbol {
					continue
				}

				dexSide := m.extractSideFromDEXTrade(dexTrade)
				if dexSide != side {
					continue
				}

				// 检查数量差异
				quantityDiff := math.Abs(dexTrade.Quantity - action.Quantity)
				quantityDiffPct := 0.0
				if action.Quantity > 0 {
					quantityDiffPct = quantityDiff / action.Quantity
				}

				// 检查价格差异
				priceDiff := math.Abs(dexTrade.Price - action.Price)
				priceDiffPct := 0.0
				if action.Price > 0 {
					priceDiffPct = priceDiff / action.Price
				}

				// 数量差异 < 5%，价格差异 < 1%
				if quantityDiffPct <= m.quantityTolerance && priceDiffPct <= m.priceTolerance {
					match := m.createMatch(decision, &action, &dexTrade, "quantity_price", 0.6)
					matches = append(matches, match)
					newMatchedDecisions[i] = true
					newMatchedDEXTrades[j] = true
					break // 一个动作只能匹配一个DEX交易
				}
			}
		}
	}

	return matches, newMatchedDecisions, newMatchedDEXTrades
}

// createMatch 创建匹配结果
func (m *DecisionMatcher) createMatch(
	decision *review.DecisionRecord,
	action *review.DecisionAction,
	dexTrade *trade_history.ExchangeFill,
	matchMethod string,
	confidence float64,
) *review.DecisionMatch {
	// 计算滑点
	slippage := 0.0
	if action.Price > 0 {
		slippage = math.Abs(dexTrade.Price-action.Price) / action.Price
	}

	// 计算执行延迟（毫秒）
	executionDelay := int64(0)
	if !action.Timestamp.IsZero() && !dexTrade.Timestamp.IsZero() {
		executionDelay = dexTrade.Timestamp.Sub(action.Timestamp).Milliseconds()
		if executionDelay < 0 {
			executionDelay = -executionDelay
		}
	}

	// 计算费用差异（这里简化处理，实际费用需要从DEX数据中获取）
	feeDifference := dexTrade.Fee // 假设预期费用为0，实际需要从决策中提取

	// 计算数量差异
	quantityDifference := math.Abs(dexTrade.Quantity - action.Quantity)

	return &review.DecisionMatch{
		DecisionID:        fmt.Sprintf("%d_%d", decision.CycleNumber, action.OrderID),
		DEXTradeID:        fmt.Sprintf("%d_%d", dexTrade.ExchangeOid, dexTrade.ExchangeTid),
		MatchMethod:       matchMethod,
		MatchConfidence:   confidence,
		Slippage:          slippage,
		ExecutionDelay:    executionDelay,
		FeeDifference:     feeDifference,
		QuantityDifference: quantityDifference,
		Decision:          decision,
		DEXTrade:          dexTrade,
	}
}

// mergeMultipleDEXTrades 合并一个决策对应的多个DEX交易
func (m *DecisionMatcher) mergeMultipleDEXTrades(matches []*review.DecisionMatch) []*review.DecisionMatch {
	// 按决策ID分组
	matchesByDecision := make(map[string][]*review.DecisionMatch)
	for _, match := range matches {
		decisionKey := fmt.Sprintf("%d", match.Decision.CycleNumber)
		matchesByDecision[decisionKey] = append(matchesByDecision[decisionKey], match)
	}

	var mergedMatches []*review.DecisionMatch
	for _, decisionMatches := range matchesByDecision {
		if len(decisionMatches) == 1 {
			mergedMatches = append(mergedMatches, decisionMatches[0])
			continue
		}

		// 多个DEX交易对应一个决策，需要合并
		// 累加数量、平均价格
		firstMatch := decisionMatches[0]
		totalQuantity := 0.0
		totalPrice := 0.0
		totalFee := 0.0
		var totalExecutionDelay int64 = 0
		var maxSlippage float64 = 0

		for _, match := range decisionMatches {
			if match.DEXTrade != nil {
				totalQuantity += match.DEXTrade.Quantity
				totalPrice += match.DEXTrade.Price
				totalFee += match.DEXTrade.Fee
				totalExecutionDelay += match.ExecutionDelay
				if match.Slippage > maxSlippage {
					maxSlippage = match.Slippage
				}
			}
		}

		avgExecutionDelay := totalExecutionDelay / int64(len(decisionMatches))

		// 创建合并后的匹配结果
		mergedMatch := &review.DecisionMatch{
			DecisionID:        firstMatch.DecisionID,
			DEXTradeID:        fmt.Sprintf("merged_%d", len(decisionMatches)),
			MatchMethod:       firstMatch.MatchMethod,
			MatchConfidence:   firstMatch.MatchConfidence * 0.9, // 降低置信度
			Slippage:          maxSlippage,
			ExecutionDelay:    avgExecutionDelay,
			FeeDifference:     totalFee,
			QuantityDifference: totalQuantity - firstMatch.Decision.Decisions[0].Quantity,
			Decision:          firstMatch.Decision,
			DEXTrade:          firstMatch.DEXTrade, // 保留第一个DEX交易作为代表
		}

		log.Printf("⚠️ 发现一个决策对应多个DEX交易 (决策ID: %s, DEX交易数: %d)", firstMatch.DecisionID, len(decisionMatches))
		mergedMatches = append(mergedMatches, mergedMatch)
	}

	return mergedMatches
}

// extractSideFromAction 从决策动作中提取方向
func (m *DecisionMatcher) extractSideFromAction(action string) string {
	switch action {
	case "open_long", "close_long":
		return "long"
	case "open_short", "close_short":
		return "short"
	default:
		return ""
	}
}

// extractSideFromDEXTrade 从DEX交易中提取方向
func (m *DecisionMatcher) extractSideFromDEXTrade(dexTrade trade_history.ExchangeFill) string {
	// 优先使用 Side 字段
	if dexTrade.Side != "" {
		return dexTrade.Side
	}

	// 如果没有，从 ExchangeSide 推断
	if dexTrade.ExchangeSide != "" {
		if dexTrade.ExchangeSide == "A" || dexTrade.ExchangeSide == "B" {
			// Hyperliquid的side：A=long, B=short
			if dexTrade.ExchangeSide == "A" {
				return "long"
			}
			return "short"
		}
	}

	return ""
}


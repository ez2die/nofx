package trade_analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// OpenPosition 未平仓持仓（用于配对）
type OpenPosition struct {
	Record       *TradeRecord // 原始开仓记录
	RemainingQty float64     // 剩余未匹配数量
}

// PairMatcher 交易配对器接口
type PairMatcher interface {
	// MatchTradePairs 匹配交易对
	MatchTradePairs(ctx context.Context, filter *AnalyticsFilter) ([]*TradePair, error)

	// GetPairStatistics 获取配对统计
	GetPairStatistics(ctx context.Context, filter *AnalyticsFilter) (*PairStatistics, error)
}

// pairMatcher 实现
type pairMatcher struct {
	repo Repository
}

// NewPairMatcher 创建PairMatcher实例
func NewPairMatcher(repo Repository) PairMatcher {
	return &pairMatcher{repo: repo}
}

// getAbsoluteQuantity 获取绝对数量（用于配对）
func getAbsoluteQuantity(record *TradeRecord) float64 {
	if record.SignedQuantity != nil {
		return math.Abs(*record.SignedQuantity) // 使用绝对值
	}
	return record.Quantity // 回退到 Quantity
}

// isOpenAction 判断是否为开仓操作
func isOpenAction(action string) bool {
	return action == "open_long" || action == "open_short"
}

// isCloseAction 判断是否为平仓操作
func isCloseAction(action string) bool {
	return action == "close_long" || action == "close_short"
}

// MatchTradePairs 匹配交易对
func (p *pairMatcher) MatchTradePairs(ctx context.Context, filter *AnalyticsFilter) ([]*TradePair, error) {
	// 获取所有交易记录
	records, err := p.repo.GetRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	// 按时间排序（稳定排序：时间戳相同时使用ID作为次要排序键）
	sort.Slice(records, func(i, j int) bool {
		if records[i].Timestamp.Equal(records[j].Timestamp) {
			// 时间戳相同时，使用ID作为次要排序键，确保排序稳定
			// 这对于多笔成交场景很重要（同一订单的多笔成交可能时间戳相同）
			return records[i].ID < records[j].ID
		}
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	// 初始化
	openPositions := make(map[string][]*OpenPosition) // key: "symbol_side"
	var pairs []*TradePair

	// 遍历记录
	for _, record := range records {
		// 使用 Symbol + Side 作为 key（不依赖 signed_quantity 符号）
		key := record.Symbol + "_" + record.Side

		if isOpenAction(record.Action) {
			// 开仓：添加到队列
			qty := getAbsoluteQuantity(record) // 使用绝对值
			openPositions[key] = append(openPositions[key], &OpenPosition{
				Record:       record,
				RemainingQty: qty,
			})

		} else if isCloseAction(record.Action) {
			// 平仓：FIFO 匹配
			closeQty := getAbsoluteQuantity(record) // 使用绝对值
			remainingCloseQty := closeQty

			// 从队列头部开始匹配
			for i := 0; i < len(openPositions[key]) && remainingCloseQty > 0; {
				openPos := openPositions[key][i]

				// 确保时间顺序：开仓时间必须早于平仓时间
				if !openPos.Record.Timestamp.Before(record.Timestamp) {
					// 时间顺序异常，跳过
					i++
					continue
				}

				if openPos.RemainingQty <= remainingCloseQty {
					// 完全匹配或部分匹配（开仓全部用完）
					matchedQty := openPos.RemainingQty

					pair := &TradePair{
						OpenRecord:  openPos.Record,
						CloseRecord: record,
						MatchedQty:  matchedQty,
						HoldingTime: int64(record.Timestamp.Sub(openPos.Record.Timestamp).Seconds()),
						PnL:         calculateProportionalPnL(record, matchedQty, closeQty),
						OpenFee:     calculateProportionalFee(openPos.Record, matchedQty),
						CloseFee:    calculateProportionalFee(record, matchedQty),
					}
					pairs = append(pairs, pair)

					remainingCloseQty -= matchedQty
					// 移除已完全匹配的开仓
					openPositions[key] = append(openPositions[key][:i], openPositions[key][i+1:]...)

				} else {
					// 部分匹配（平仓全部用完，开仓还有剩余）
					matchedQty := remainingCloseQty

					pair := &TradePair{
						OpenRecord:  openPos.Record,
						CloseRecord: record,
						MatchedQty:  matchedQty,
						HoldingTime: int64(record.Timestamp.Sub(openPos.Record.Timestamp).Seconds()),
						PnL:         calculateProportionalPnL(record, matchedQty, closeQty),
						OpenFee:     calculateProportionalFee(openPos.Record, matchedQty),
						CloseFee:    calculateProportionalFee(record, matchedQty),
					}
					pairs = append(pairs, pair)

					// 更新开仓剩余数量
					openPos.RemainingQty -= matchedQty
					remainingCloseQty = 0
					i++ // 继续下一个开仓（但当前平仓已用完）
				}
			}
		}
	}

	return pairs, nil
}

// calculateProportionalPnL 按比例计算PnL
func calculateProportionalPnL(closeRecord *TradeRecord, matchedQty, totalCloseQty float64) float64 {
	if closeRecord.PnL == nil || totalCloseQty == 0 {
		return 0
	}

	// 按数量比例分配
	ratio := matchedQty / totalCloseQty
	return *closeRecord.PnL * ratio
}

// calculateProportionalFee 按比例计算费用
func calculateProportionalFee(record *TradeRecord, matchedQty float64) float64 {
	recordQty := getAbsoluteQuantity(record)
	if recordQty == 0 {
		return 0
	}

	// 按数量比例分配
	ratio := matchedQty / recordQty
	return record.Fee * ratio
}

// GetPairStatistics 获取配对统计
func (p *pairMatcher) GetPairStatistics(ctx context.Context, filter *AnalyticsFilter) (*PairStatistics, error) {
	// 匹配交易对
	pairs, err := p.MatchTradePairs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("匹配交易对失败: %w", err)
	}

	// 获取开仓记录数（用于计算配对成功率）
	overview, err := p.repo.GetOverviewStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取概览统计失败: %w", err)
	}

	stats := &PairStatistics{
		TotalPairs: len(pairs),
		Pairs:      pairs,
	}

	// 计算配对成功率和未配对记录数
	if overview.OpenTrades > 0 {
		stats.PairSuccessRate = float64(len(pairs)) / float64(overview.OpenTrades) * 100.0
		stats.UnpairedTrades = overview.OpenTrades - len(pairs)
		if stats.UnpairedTrades < 0 {
			stats.UnpairedTrades = 0
		}
	}

	// 计算持仓时间统计
	if len(pairs) > 0 {
		var totalHoldingTime int64
		var minHoldingTime int64 = math.MaxInt64
		var maxHoldingTime int64

		for _, pair := range pairs {
			totalHoldingTime += pair.HoldingTime
			if pair.HoldingTime < minHoldingTime {
				minHoldingTime = pair.HoldingTime
			}
			if pair.HoldingTime > maxHoldingTime {
				maxHoldingTime = pair.HoldingTime
			}
		}

		stats.AvgHoldingTime = totalHoldingTime / int64(len(pairs))
		stats.MinHoldingTime = minHoldingTime
		stats.MaxHoldingTime = maxHoldingTime
	}

	return stats, nil
}


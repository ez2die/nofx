package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"nofx/review"
	"nofx/trade_history"
)

// TradeHistoryCollector 交易历史收集器
type TradeHistoryCollector struct {
	reader review.TradeHistoryReader
}

// NewTradeHistoryCollector 创建交易历史收集器
func NewTradeHistoryCollector(reader review.TradeHistoryReader) *TradeHistoryCollector {
	return &TradeHistoryCollector{
		reader: reader,
	}
}

// Collect 收集指定时间范围内的交易历史
func (c *TradeHistoryCollector) Collect(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*trade_history.TradeRecord, error) {
	log.Printf("📊 开始收集交易历史 (trader_id: %s, 时间范围: %s ~ %s)",
		traderID, startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// 构建过滤器
	filter := &trade_history.TradeRecordFilter{
		TraderID:  traderID,
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     10000, // 设置一个较大的限制，实际应该使用分页查询
		Offset:    0,
		OrderBy:   "timestamp ASC",
	}

	// 查询交易历史
	records, err := c.reader.FindByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("收集交易历史失败: %w", err)
	}

	// 边界处理：只统计在复盘窗口内完整完成的交易
	// 如果窗口开始时间点的第一笔交易是 close 操作，需要追溯到上一个窗口找到对应的 open 操作
	// 如果窗口结束时间点的最后一笔交易是 open 操作，对应的 close 操作在窗口外，则该交易不计入本次复盘
	completedTrades, err := c.filterCompletedTrades(ctx, traderID, records, startTime, endTime)
	if err != nil {
		log.Printf("⚠️ 边界处理失败: %v，使用原始记录", err)
		completedTrades = records
	}

	log.Printf("✅ 交易历史收集完成，共 %d 条记录（完整交易: %d）", len(records), len(completedTrades))
	return completedTrades, nil
}

// filterCompletedTrades 过滤出完整的交易（开仓+平仓都在窗口内或边界处理）
func (c *TradeHistoryCollector) filterCompletedTrades(
	ctx context.Context,
	traderID string,
	records []*trade_history.TradeRecord,
	startTime, endTime time.Time,
) ([]*trade_history.TradeRecord, error) {
	if len(records) == 0 {
		return records, nil
	}

	// 用于追踪开仓记录：key = symbol_side
	openPositions := make(map[string]*trade_history.TradeRecord)
	var completedTrades []*trade_history.TradeRecord

	// 检查窗口开始处的 close 操作，需要追溯到上一个窗口
	// 这里简化处理：如果第一条记录是 close，尝试查找对应的 open
	firstRecord := records[0]
	if firstRecord != nil && (firstRecord.Action == "close_long" || firstRecord.Action == "close_short") {
		// 尝试找到对应的 open 记录（在 startTime 之前）
		prevStartTime := startTime.Add(-24 * time.Hour) // 往前查找24小时
		prevFilter := &trade_history.TradeRecordFilter{
			TraderID:  traderID,
			Symbol:    firstRecord.Symbol,
			StartTime: &prevStartTime,
			EndTime:   &startTime,
			Limit:     100,
			Offset:    0,
			OrderBy:   "timestamp DESC",
		}

		prevRecords, err := c.reader.FindByFilter(ctx, prevFilter)
		if err == nil {
			// 查找匹配的 open 记录
			side := ""
			if firstRecord.Action == "close_long" {
				side = "long"
			} else if firstRecord.Action == "close_short" {
				side = "short"
			}

			for _, prevRecord := range prevRecords {
				if prevRecord.Symbol == firstRecord.Symbol {
					if (side == "long" && prevRecord.Action == "open_long") ||
						(side == "short" && prevRecord.Action == "open_short") {
						// 找到匹配的 open，将其添加到结果中
						completedTrades = append(completedTrades, prevRecord)
						break
					}
				}
			}
		}
	}

	// 遍历窗口内的记录，配对开仓和平仓
	for _, record := range records {
		symbol := record.Symbol
		side := record.Side
		if side == "" {
			// 从 action 推断 side
			if record.Action == "open_long" || record.Action == "close_long" {
				side = "long"
			} else if record.Action == "open_short" || record.Action == "close_short" {
				side = "short"
			}
		}

		posKey := fmt.Sprintf("%s_%s", symbol, side)

		switch record.Action {
		case "open_long", "open_short":
			// 记录开仓
			openPositions[posKey] = record

		case "close_long", "close_short":
			// 查找对应的开仓记录
			if openRecord, exists := openPositions[posKey]; exists {
				// 找到匹配，添加到结果中
				completedTrades = append(completedTrades, openRecord)
				completedTrades = append(completedTrades, record)
				delete(openPositions, posKey)
			} else {
				// 如果窗口开始处的 close 没有找到 open，则只添加 close（边界处理已处理）
				// 对于窗口内的 close，如果没有对应的 open，可能是数据不完整，跳过
				log.Printf("⚠️ 发现未匹配的平仓记录: symbol=%s, action=%s, timestamp=%s",
					symbol, record.Action, record.Timestamp.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// 检查窗口结束处的 open 操作（这些 open 的 close 可能在窗口外）
	// 这些 open 不计入本次复盘，因为交易未完成
	for _, openRecord := range openPositions {
		log.Printf("ℹ️ 发现未完成的交易（窗口结束时仍为开仓状态）: symbol=%s, action=%s, timestamp=%s",
			openRecord.Symbol, openRecord.Action, openRecord.Timestamp.Format("2006-01-02 15:04:05"))
	}

	return completedTrades, nil
}


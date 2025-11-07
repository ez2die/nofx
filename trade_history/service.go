package trade_history

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Service 交易历史服务接口
type Service interface {
	// RecordTrade 记录交易（从API返回的数据）
	RecordTrade(ctx context.Context, record *TradeRecord) error

	// RecordAutoTriggeredClose 记录自动触发的平仓（止盈止损）
	RecordAutoTriggeredClose(ctx context.Context, record *TradeRecord) error

	// GetTradeHistory 获取交易历史
	GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, int, error)

	// GetTradeStatistics 获取交易统计
	GetTradeStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error)

	// SyncFromExchange 从交易所同步交易记录
	SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error
}

// service 实现
type service struct {
	repo Repository
}

// NewService 创建Service实例
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// RecordTrade 记录交易（从API返回的数据）
func (s *service) RecordTrade(ctx context.Context, record *TradeRecord) error {
	// 1. 可选：验证trader是否存在（不阻止保存）
	exists, err := s.repo.ValidateTraderExists(ctx, record.TraderID)
	if err != nil {
		log.Printf("⚠️ 警告：验证trader是否存在失败: %v", err)
	} else if !exists {
		log.Printf("⚠️ 警告：trader %s 不存在，但交易历史仍会保存", record.TraderID)
	}

	// 2. 时间戳处理：优先使用交易所时间戳
	// 如果exchange_timestamp_ms已设置，使用它作为主时间戳
	if record.ExchangeTimestampMs != nil && *record.ExchangeTimestampMs > 0 {
		exchangeTime := time.Unix(*record.ExchangeTimestampMs/1000, (*record.ExchangeTimestampMs%1000)*1000000)
		record.ExchangeTimestamp = &exchangeTime
		record.Timestamp = exchangeTime // 主时间戳使用交易所时间
	} else if record.Timestamp.IsZero() {
		// 如果都没有设置，使用本地时间作为备用
		record.Timestamp = time.Now()
	}

	// 3. 检查是否已存在（通过exchange_order_id或exchange_trade_id）
	var exchangeOid, exchangeTid *int64
	var exchangeHash *string

	if record.ExchangeOrderID != nil {
		var oid int64
		if _, err := fmt.Sscanf(*record.ExchangeOrderID, "%d", &oid); err == nil {
			exchangeOid = &oid
		}
	}
	if record.ExchangeTradeID != nil {
		var tid int64
		if _, err := fmt.Sscanf(*record.ExchangeTradeID, "%d", &tid); err == nil {
			exchangeTid = &tid
		}
	}
	if record.ExchangeHash != nil {
		exchangeHash = record.ExchangeHash
	}

	existsByExchange, err := s.repo.ExistsByExchangeID(ctx, exchangeOid, exchangeTid, exchangeHash)
	if err != nil {
		log.Printf("⚠️ 警告：检查交易记录是否存在失败: %v", err)
	} else if existsByExchange {
		// 如果已存在，跳过（避免重复）
		log.Printf("ℹ️ 交易记录已存在，跳过重复记录 (trader_id=%s, symbol=%s, action=%s)", record.TraderID, record.Symbol, record.Action)
		return nil
	}

	// 4. 如果是平仓，计算PnL（在插入前计算）
	if record.Action == "close_long" || record.Action == "close_short" {
		if record.EntryPrice != nil && record.ExitPrice != nil {
			// 计算PnL
			var pnl float64
			var pnlPct float64
			if record.Action == "close_long" {
				// 做多：平仓价格 - 开仓价格
				pnl = (*record.ExitPrice - *record.EntryPrice) * record.Quantity * float64(record.Leverage)
				pnlPct = ((*record.ExitPrice - *record.EntryPrice) / *record.EntryPrice) * 100.0 * float64(record.Leverage)
			} else {
				// 做空：开仓价格 - 平仓价格
				pnl = (*record.EntryPrice - *record.ExitPrice) * record.Quantity * float64(record.Leverage)
				pnlPct = ((*record.EntryPrice - *record.ExitPrice) / *record.EntryPrice) * 100.0 * float64(record.Leverage)
			}

			// 减去手续费
			pnl -= record.Fee

			record.PnL = &pnl
			record.PnLPct = &pnlPct
		}
	}

	// 5. 如果不存在，插入
	if err := s.repo.Save(ctx, record); err != nil {
		return fmt.Errorf("保存交易记录失败: %w", err)
	}

	return nil
}

// RecordAutoTriggeredClose 记录自动触发的平仓（止盈止损）
func (s *service) RecordAutoTriggeredClose(ctx context.Context, record *TradeRecord) error {
	// 标记为自动触发
	record.IsAutoTriggered = true
	record.Source = "api" // 自动触发也来自API

	// 调用RecordTrade记录
	return s.RecordTrade(ctx, record)
}

// GetTradeHistory 获取交易历史
func (s *service) GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, int, error) {
	// 查询记录
	records, err := s.repo.FindByFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("查询交易历史失败: %w", err)
	}

	// 查询总数
	total, err := s.repo.CountByFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("统计交易历史失败: %w", err)
	}

	return records, total, nil
}

// GetTradeStatistics 获取交易统计
func (s *service) GetTradeStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx, traderID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("获取交易统计失败: %w", err)
	}

	return stats, nil
}

// SyncFromExchange 从交易所同步交易记录
func (s *service) SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
	// 获取最近的成交记录（默认最近100条）
	fills, err := provider.GetRecentFills(100)
	if err != nil {
		return fmt.Errorf("获取交易所成交记录失败: %w", err)
	}

	if len(fills) == 0 {
		log.Printf("ℹ️ 未找到新的成交记录（trader_id=%s）", traderID)
		return nil
	}

	// 转换为TradeRecord并批量保存
	var records []*TradeRecord
	for _, fill := range fills {
		// 转换ExchangeFill到TradeRecord
		record := s.convertFillToRecord(traderID, &fill)
		record.Source = "sync"

		// 检查是否已存在
		var exchangeOid, exchangeTid *int64
		var exchangeHash *string

		if fill.ExchangeOid > 0 {
			exchangeOid = &fill.ExchangeOid
		}
		if fill.ExchangeTid > 0 {
			exchangeTid = &fill.ExchangeTid
		}
		if fill.ExchangeHash != "" {
			exchangeHash = &fill.ExchangeHash
		}

		exists, err := s.repo.ExistsByExchangeID(ctx, exchangeOid, exchangeTid, exchangeHash)
		if err != nil {
			log.Printf("⚠️ 警告：检查交易记录是否存在失败: %v", err)
			continue
		}
		if exists {
			// 已存在，跳过
			continue
		}

		records = append(records, record)
	}

	if len(records) == 0 {
		log.Printf("ℹ️ 所有成交记录已存在（trader_id=%s）", traderID)
		return nil
	}

	// 批量保存
	if err := s.repo.SaveBatch(ctx, records); err != nil {
		return fmt.Errorf("批量保存交易记录失败: %w", err)
	}

	log.Printf("✅ 同步完成：保存了 %d 条新交易记录（trader_id=%s）", len(records), traderID)
	return nil
}

// convertFillToRecord 将ExchangeFill转换为TradeRecord
func (s *service) convertFillToRecord(traderID string, fill *ExchangeFill) *TradeRecord {
	record := &TradeRecord{
		TraderID:       traderID,
		Symbol:         fill.Symbol,
		Side:           fill.Side,
		Quantity:       fill.Quantity,
		ExecutionPrice: fill.Price,
		Fee:            fill.Fee,
		FeeToken:       &fill.FeeToken,
		Timestamp:      fill.Timestamp,
		Source:         "sync",
	}

	// 设置Action
	if fill.Dir == "Open" {
		if fill.Side == "long" {
			record.Action = "open_long"
		} else {
			record.Action = "open_short"
		}
	} else {
		if fill.Side == "long" {
			record.Action = "close_long"
		} else {
			record.Action = "close_short"
		}
	}

	// 设置交易所时间戳
	if fill.TimestampMs > 0 {
		record.ExchangeTimestampMs = &fill.TimestampMs
		exchangeTime := time.Unix(fill.TimestampMs/1000, (fill.TimestampMs%1000)*1000000)
		record.ExchangeTimestamp = &exchangeTime
		record.Timestamp = exchangeTime
	}

	// 设置交易所订单信息
	if fill.ExchangeOid > 0 {
		oidStr := fmt.Sprintf("%d", fill.ExchangeOid)
		record.ExchangeOrderID = &oidStr
	}
	if fill.ExchangeTid > 0 {
		tidStr := fmt.Sprintf("%d", fill.ExchangeTid)
		record.ExchangeTradeID = &tidStr
	}
	if fill.ExchangeHash != "" {
		record.ExchangeHash = &fill.ExchangeHash
	}

	// 设置盈亏信息（仅平仓时有效）
	if fill.ClosedPnl != nil {
		record.PnL = fill.ClosedPnl
		// 计算PnL百分比（如果有entry_price）
		if fill.StartPosition != nil && *fill.StartPosition != 0 {
			pnlPct := (*fill.ClosedPnl / (*fill.StartPosition * fill.Price)) * 100.0
			record.PnLPct = &pnlPct
		}
	}

	// 设置杠杆（默认值，可能需要从其他地方获取）
	record.Leverage = 1 // 默认值，实际值需要从其他来源获取

	return record
}


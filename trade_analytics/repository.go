package trade_analytics

import (
	"context"
	"database/sql"
	"fmt"
	"nofx/trade_history"
	"sort"
	"strings"
	"time"
)

// Repository 交易分析数据仓库接口
type Repository interface {
	// GetRecordsByFilter 根据过滤器获取交易记录
	GetRecordsByFilter(ctx context.Context, filter *AnalyticsFilter) ([]*TradeRecord, error)

	// GetAggregatedRecordsByFilter 获取聚合后的交易记录（将同一订单的多笔执行合并）
	GetAggregatedRecordsByFilter(ctx context.Context, filter *AnalyticsFilter) ([]*TradeRecord, error)

	// GetOverviewStats 获取概览统计
	GetOverviewStats(ctx context.Context, filter *AnalyticsFilter) (*TradeOverview, error)

	// GetPnLStats 获取盈亏统计
	GetPnLStats(ctx context.Context, filter *AnalyticsFilter) (*PnLStatistics, error)

	// GetWinRateStats 获取胜率统计
	GetWinRateStats(ctx context.Context, filter *AnalyticsFilter) (*WinRateStatistics, error)

	// GetFeeStats 获取费用统计
	GetFeeStats(ctx context.Context, filter *AnalyticsFilter) (*FeeStatistics, error)

	// GetDirectionStats 获取方向统计
	GetDirectionStats(ctx context.Context, filter *AnalyticsFilter) (*DirectionStatistics, error)

	// GetSymbolStats 获取币种统计
	GetSymbolStats(ctx context.Context, filter *AnalyticsFilter) (map[string]*SymbolStatistics, error)

	// GetDailyStats 获取每日统计
	GetDailyStats(ctx context.Context, filter *AnalyticsFilter) ([]DailyStatistics, error)

	// GetWeeklyStats 获取每周统计
	GetWeeklyStats(ctx context.Context, filter *AnalyticsFilter) ([]WeeklyStatistics, error)

	// GetMonthlyStats 获取每月统计
	GetMonthlyStats(ctx context.Context, filter *AnalyticsFilter) ([]MonthlyStatistics, error)

	// GetFrequencyStats 获取交易频率统计
	GetFrequencyStats(ctx context.Context, filter *AnalyticsFilter) (*TradeFrequencyStats, error)

	// GetActionStats 获取交易类型统计
	GetActionStats(ctx context.Context, filter *AnalyticsFilter) (*ActionStatistics, error)

	// GetTrendAnalysis 获取趋势分析
	GetTrendAnalysis(ctx context.Context, filter *AnalyticsFilter) (*TrendAnalysis, error)
}

// repository 实现
type repository struct {
	db           *sql.DB
	tradeHistory trade_history.Repository
}

// NewRepository 创建Repository实例
func NewRepository(db *sql.DB, tradeHistoryRepo trade_history.Repository) Repository {
	return &repository{
		db:           db,
		tradeHistory: tradeHistoryRepo,
	}
}

// buildWhereClause 构建WHERE子句
func (r *repository) buildWhereClause(filter *AnalyticsFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter.TraderID != "" {
		conditions = append(conditions, "trader_id = ?")
		args = append(args, filter.TraderID)
	}
	if filter.Symbol != "" {
		conditions = append(conditions, "symbol = ?")
		args = append(args, filter.Symbol)
	}
	if filter.Side != "" {
		conditions = append(conditions, "side = ?")
		args = append(args, filter.Side)
	}
	if filter.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// GetRecordsByFilter 根据过滤器获取交易记录
func (r *repository) GetRecordsByFilter(ctx context.Context, filter *AnalyticsFilter) ([]*TradeRecord, error) {
	// 使用 trade_history 的过滤器
	tradeFilter := &trade_history.TradeRecordFilter{
		TraderID:  filter.TraderID,
		Symbol:    filter.Symbol,
		Side:      filter.Side,
		Action:    filter.Action,
		StartTime: filter.StartTime,
		EndTime:   filter.EndTime,
		Limit:     10000, // 获取足够多的记录用于分析
		OrderBy:   "timestamp ASC",
	}

	records, err := r.tradeHistory.FindByFilter(ctx, tradeFilter)
	if err != nil {
		return nil, fmt.Errorf("查询交易记录失败: %w", err)
	}

	// 转换为 TradeRecord
	result := make([]*TradeRecord, len(records))
	for i, rec := range records {
		result[i] = convertTradeRecord(rec)
	}

	return result, nil
}

// GetAggregatedRecordsByFilter 获取聚合后的交易记录（将同一订单的多笔执行合并）
func (r *repository) GetAggregatedRecordsByFilter(ctx context.Context, filter *AnalyticsFilter) ([]*TradeRecord, error) {
	// 先获取原始记录
	records, err := r.GetRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 如果没有记录，直接返回
	if len(records) == 0 {
		return records, nil
	}

	// 使用 map 进行聚合：key = exchange_order_id + action + symbol + side
	// 如果 exchange_order_id 为 nil，则使用 exchange_trade_id 作为 key（单笔执行）
	type aggregateKey struct {
		OrderID string
		Action  string
		Symbol  string
		Side    string
	}

	type aggregateData struct {
		Records        []*TradeRecord
		TotalQuantity  float64
		TotalPnL       *float64
		TotalFee       float64
		WeightedPrice  float64 // quantity 加权价格
		FirstTimestamp time.Time
		FirstRecord    *TradeRecord // 保存第一条记录用于复制其他字段
	}

	aggregates := make(map[aggregateKey]*aggregateData)

	for _, rec := range records {
		// 确定聚合 key
		var key aggregateKey
		if rec.ExchangeOrderID != nil && *rec.ExchangeOrderID != "" {
			// 有 exchange_order_id，可以聚合
			key = aggregateKey{
				OrderID: *rec.ExchangeOrderID,
				Action:  rec.Action,
				Symbol:  rec.Symbol,
				Side:    rec.Side,
			}
		} else {
			// 没有 exchange_order_id，使用 exchange_trade_id 作为唯一标识（不聚合）
			orderID := ""
			if rec.ExchangeTradeID != nil {
				orderID = *rec.ExchangeTradeID
			} else {
				// 如果连 exchange_trade_id 都没有，使用 ID 作为唯一标识
				orderID = fmt.Sprintf("single_%d", rec.ID)
			}
			key = aggregateKey{
				OrderID: orderID,
				Action:  rec.Action,
				Symbol:  rec.Symbol,
				Side:    rec.Side,
			}
		}

		// 获取或创建聚合数据
		agg, exists := aggregates[key]
		if !exists {
			agg = &aggregateData{
				Records:        []*TradeRecord{rec},
				TotalQuantity:  rec.Quantity,
				TotalFee:       rec.Fee,
				WeightedPrice:  rec.ExecutionPrice * rec.Quantity,
				FirstTimestamp: rec.Timestamp,
				FirstRecord:    rec,
			}

			// 初始化 PnL
			if rec.PnL != nil {
				totalPnL := *rec.PnL
				agg.TotalPnL = &totalPnL
			}

			aggregates[key] = agg
		} else {
			// 合并数据
			agg.Records = append(agg.Records, rec)
			agg.TotalQuantity += rec.Quantity
			agg.TotalFee += rec.Fee
			agg.WeightedPrice += rec.ExecutionPrice * rec.Quantity

			// 合并 PnL
			if rec.PnL != nil {
				if agg.TotalPnL == nil {
					totalPnL := *rec.PnL
					agg.TotalPnL = &totalPnL
				} else {
					*agg.TotalPnL += *rec.PnL
				}
			}

			// 更新最早的时间戳
			if rec.Timestamp.Before(agg.FirstTimestamp) {
				agg.FirstTimestamp = rec.Timestamp
				agg.FirstRecord = rec
			}
		}
	}

	// 构建聚合后的记录列表
	result := make([]*TradeRecord, 0, len(aggregates))
	for _, agg := range aggregates {
		// 计算加权平均价格（安全检查：防止除零）
		var avgPrice float64
		if agg.TotalQuantity > 0 {
			avgPrice = agg.WeightedPrice / agg.TotalQuantity
		} else {
			// 如果 quantity 为 0（理论上不应该发生），使用第一条记录的价格
			avgPrice = agg.FirstRecord.ExecutionPrice
		}

		// 创建聚合记录
		aggRecord := &TradeRecord{
			ID:                agg.FirstRecord.ID, // 使用第一条记录的 ID
			TraderID:          agg.FirstRecord.TraderID,
			Symbol:            agg.FirstRecord.Symbol,
			Action:            agg.FirstRecord.Action,
			Side:              agg.FirstRecord.Side,
			Quantity:          agg.TotalQuantity,
			SignedQuantity:    agg.FirstRecord.SignedQuantity, // 保持原值
			ExecutionPrice:    avgPrice,
			PnL:               agg.TotalPnL,
			Fee:               agg.TotalFee,
			FeeToken:          agg.FirstRecord.FeeToken,
			ExchangeOrderID:   agg.FirstRecord.ExchangeOrderID,
			ExchangeTradeID:   agg.FirstRecord.ExchangeTradeID,
			ExchangeHash:      agg.FirstRecord.ExchangeHash,
			RawDir:            agg.FirstRecord.RawDir,
			StartPosition:     agg.FirstRecord.StartPosition,
			BuilderFee:        agg.FirstRecord.BuilderFee,
			ExchangeSide:      agg.FirstRecord.ExchangeSide,
			Timestamp:         agg.FirstTimestamp, // 使用最早的时间戳
			ExchangeTimestamp: agg.FirstRecord.ExchangeTimestamp,
			ExchangeTimestampMs: agg.FirstRecord.ExchangeTimestampMs,
			CreatedAt:         agg.FirstRecord.CreatedAt,
			UpdatedAt:         agg.FirstRecord.UpdatedAt,
		}

		result = append(result, aggRecord)
	}

	// 按时间排序（聚合后的记录）
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

// convertTradeRecord 转换 trade_history.TradeRecord 到 TradeRecord
func convertTradeRecord(rec *trade_history.TradeRecord) *TradeRecord {
	return &TradeRecord{
		ID:                  rec.ID,
		TraderID:            rec.TraderID,
		Symbol:              rec.Symbol,
		Action:              rec.Action,
		Side:                rec.Side,
		Quantity:            rec.Quantity,
		SignedQuantity:      rec.SignedQuantity,
		ExecutionPrice:      rec.ExecutionPrice,
		PnL:                 rec.PnL,
		Fee:                 rec.Fee,
		FeeToken:            rec.FeeToken,
		ExchangeOrderID:     rec.ExchangeOrderID,
		ExchangeTradeID:     rec.ExchangeTradeID,
		ExchangeHash:        rec.ExchangeHash,
		RawDir:              rec.RawDir,
		StartPosition:       rec.StartPosition,
		BuilderFee:          rec.BuilderFee,
		ExchangeSide:        rec.ExchangeSide,
		Timestamp:           rec.Timestamp,
		ExchangeTimestamp:   rec.ExchangeTimestamp,
		ExchangeTimestampMs: rec.ExchangeTimestampMs,
		CreatedAt:           rec.CreatedAt,
		UpdatedAt:           rec.UpdatedAt,
	}
}

// GetOverviewStats 获取概览统计
func (r *repository) GetOverviewStats(ctx context.Context, filter *AnalyticsFilter) (*TradeOverview, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN action IN ('open_long', 'open_short') THEN 1 ELSE 0 END), 0) as open_trades,
			COALESCE(SUM(CASE WHEN action IN ('close_long', 'close_short') THEN 1 ELSE 0 END), 0) as close_trades,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades
		FROM trade_history
		%s
	`, whereClause)

	stats := &TradeOverview{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalTrades,
		&stats.OpenTrades,
		&stats.CloseTrades,
		&stats.CompletedTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取概览统计失败: %w", err)
	}

	// 计算未平仓交易数
	stats.UnclosedTrades = stats.OpenTrades - stats.CloseTrades
	if stats.UnclosedTrades < 0 {
		stats.UnclosedTrades = 0
	}

	return stats, nil
}

// GetPnLStats 获取盈亏统计
func (r *repository) GetPnLStats(ctx context.Context, filter *AnalyticsFilter) (*PnLStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	// 只统计有PnL的记录（平仓记录）
	whereClauseWithPnL := whereClause
	if whereClause == "" {
		whereClauseWithPnL = "WHERE pnl IS NOT NULL"
	} else {
		whereClauseWithPnL = whereClause + " AND pnl IS NOT NULL"
	}

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN pnl ELSE 0 END), 0) as total_profit,
			COALESCE(SUM(CASE WHEN pnl < 0 THEN ABS(pnl) ELSE 0 END), 0) as total_loss,
			COALESCE(MAX(CASE WHEN pnl > 0 THEN pnl ELSE NULL END), 0) as max_win,
			COALESCE(MAX(CASE WHEN pnl < 0 THEN ABS(pnl) ELSE NULL END), 0) as max_loss,
			COUNT(*) as completed_trades
		FROM trade_history
		%s
	`, whereClauseWithPnL)

	stats := &PnLStatistics{}
	var completedTrades int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalPnL,
		&stats.TotalProfit,
		&stats.TotalLoss,
		&stats.MaxWin,
		&stats.MaxLoss,
		&completedTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取盈亏统计失败: %w", err)
	}

	// 计算平均盈亏
	if completedTrades > 0 {
		stats.AvgPnL = stats.TotalPnL / float64(completedTrades)
	}

	// 计算盈亏比
	if stats.TotalLoss > 0 {
		stats.ProfitFactor = stats.TotalProfit / stats.TotalLoss
	}

	// 计算净盈亏（需要获取总费用，但不调用 GetFeeStats 避免循环）
	feeQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(fee), 0) as total_fees
		FROM trade_history
		%s
	`, whereClauseWithPnL)
	var totalFees float64
	if err := r.db.QueryRowContext(ctx, feeQuery, args...).Scan(&totalFees); err == nil {
		stats.NetPnL = stats.TotalPnL - totalFees
	}

	return stats, nil
}

// GetWinRateStats 获取胜率统计
func (r *repository) GetWinRateStats(ctx context.Context, filter *AnalyticsFilter) (*WinRateStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	// 只统计有PnL的记录
	whereClauseWithPnL := whereClause
	if whereClause == "" {
		whereClauseWithPnL = "WHERE pnl IS NOT NULL"
	} else {
		whereClauseWithPnL = whereClause + " AND pnl IS NOT NULL"
	}

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades,
			COALESCE(SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END), 0) as losing_trades,
			COALESCE(SUM(CASE WHEN pnl = 0 THEN 1 ELSE 0 END), 0) as break_even_trades,
			COUNT(*) as total_trades
		FROM trade_history
		%s
	`, whereClauseWithPnL)

	stats := &WinRateStatistics{}
	var totalTrades int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.WinningTrades,
		&stats.LosingTrades,
		&stats.BreakEvenTrades,
		&totalTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取胜率统计失败: %w", err)
	}

	// 计算胜率
	if totalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(totalTrades) * 100.0
	}

	// 获取平均盈利和平均亏损
	if stats.WinningTrades > 0 || stats.LosingTrades > 0 {
		avgQuery := fmt.Sprintf(`
			SELECT 
				COALESCE(AVG(CASE WHEN pnl > 0 THEN pnl END), 0) as avg_win,
				COALESCE(AVG(CASE WHEN pnl < 0 THEN ABS(pnl) END), 0) as avg_loss
			FROM trade_history
			%s
		`, whereClauseWithPnL)

		err = r.db.QueryRowContext(ctx, avgQuery, args...).Scan(
			&stats.AvgWin,
			&stats.AvgLoss,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("获取平均盈亏失败: %w", err)
		}
	}

	return stats, nil
}

// GetFeeStats 获取费用统计
func (r *repository) GetFeeStats(ctx context.Context, filter *AnalyticsFilter) (*FeeStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(SUM(CASE WHEN action IN ('open_long', 'open_short') THEN fee ELSE 0 END), 0) as open_fees,
			COALESCE(SUM(CASE WHEN action IN ('close_long', 'close_short') THEN fee ELSE 0 END), 0) as close_fees,
			COALESCE(SUM(builder_fee), 0) as builder_fees,
			COUNT(*) as total_trades
		FROM trade_history
		%s
	`, whereClause)

	stats := &FeeStatistics{}
	var totalTrades int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalFees,
		&stats.OpenFees,
		&stats.CloseFees,
		&stats.BuilderFees,
		&totalTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取费用统计失败: %w", err)
	}

	// 计算平均费用
	if totalTrades > 0 {
		stats.AvgFee = stats.TotalFees / float64(totalTrades)
	}

	// 计算费用占比（相对于总盈亏，但不调用 GetPnLStats 避免循环）
	whereClauseWithPnL := whereClause
	if whereClause == "" {
		whereClauseWithPnL = "WHERE pnl IS NOT NULL"
	} else {
		whereClauseWithPnL = whereClause + " AND pnl IS NOT NULL"
	}
	pnlQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(pnl), 0) as total_pnl
		FROM trade_history
		%s
	`, whereClauseWithPnL)
	var totalPnL float64
	if err := r.db.QueryRowContext(ctx, pnlQuery, args...).Scan(&totalPnL); err == nil {
		absTotalPnL := totalPnL
		if absTotalPnL < 0 {
			absTotalPnL = -absTotalPnL
		}
		if absTotalPnL > 0 {
			stats.FeeRatio = stats.TotalFees / absTotalPnL * 100.0
		}
	}

	return stats, nil
}

// GetDirectionStats 获取方向统计
func (r *repository) GetDirectionStats(ctx context.Context, filter *AnalyticsFilter) (*DirectionStatistics, error) {
	stats := &DirectionStatistics{}

	// 获取做多统计
	longFilter := *filter
	longFilter.Side = "long"
	longStats, err := r.getSideStats(ctx, &longFilter)
	if err != nil {
		return nil, fmt.Errorf("获取做多统计失败: %w", err)
	}
	stats.LongStats = longStats

	// 获取做空统计
	shortFilter := *filter
	shortFilter.Side = "short"
	shortStats, err := r.getSideStats(ctx, &shortFilter)
	if err != nil {
		return nil, fmt.Errorf("获取做空统计失败: %w", err)
	}
	stats.ShortStats = shortStats

	// 计算方向偏好
	preference, err := r.calculateDirectionPreference(ctx, filter, longStats, shortStats)
	if err == nil {
		stats.Preference = preference
	}

	return stats, nil
}

// getSideStats 获取单方向统计
func (r *repository) getSideStats(ctx context.Context, filter *AnalyticsFilter) (*SideStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	// 只统计有PnL的记录
	whereClauseWithPnL := whereClause
	if whereClause == "" {
		whereClauseWithPnL = "WHERE pnl IS NOT NULL"
	} else {
		whereClauseWithPnL = whereClause + " AND pnl IS NOT NULL"
	}

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades
		FROM trade_history
		%s
	`, whereClauseWithPnL)

	stats := &SideStatistics{}
	var winningTrades int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalTrades,
		&stats.TotalPnL,
		&winningTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取方向统计失败: %w", err)
	}

	// 计算胜率和平均盈亏
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(winningTrades) / float64(stats.TotalTrades) * 100.0
		stats.AvgPnL = stats.TotalPnL / float64(stats.TotalTrades)
	}

	return stats, nil
}

// GetSymbolStats 获取币种统计
func (r *repository) GetSymbolStats(ctx context.Context, filter *AnalyticsFilter) (map[string]*SymbolStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	// 只统计有PnL的记录
	whereClauseWithPnL := whereClause
	if whereClause == "" {
		whereClauseWithPnL = "WHERE pnl IS NOT NULL"
	} else {
		whereClauseWithPnL = whereClause + " AND pnl IS NOT NULL"
	}

	query := fmt.Sprintf(`
		SELECT 
			symbol,
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades,
			COALESCE(MAX(CASE WHEN pnl > 0 THEN pnl ELSE NULL END), 0) as max_win,
			COALESCE(MAX(CASE WHEN pnl < 0 THEN ABS(pnl) ELSE NULL END), 0) as max_loss,
			COALESCE(SUM(fee), 0) as total_fees
		FROM trade_history
		%s
		GROUP BY symbol
		ORDER BY symbol
	`, whereClauseWithPnL)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询币种统计失败: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*SymbolStatistics)
	for rows.Next() {
		stats := &SymbolStatistics{}
		var winningTrades int
		err := rows.Scan(
			&stats.Symbol,
			&stats.TotalTrades,
			&stats.CompletedTrades,
			&stats.TotalPnL,
			&winningTrades,
			&stats.MaxWin,
			&stats.MaxLoss,
			&stats.TotalFees,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描币种统计失败: %w", err)
		}

		// 计算胜率和平均盈亏
		if stats.CompletedTrades > 0 {
			stats.WinRate = float64(winningTrades) / float64(stats.CompletedTrades) * 100.0
			stats.AvgPnL = stats.TotalPnL / float64(stats.CompletedTrades)
		}

		result[stats.Symbol] = stats
	}

	return result, nil
}

// GetDailyStats 获取每日统计
func (r *repository) GetDailyStats(ctx context.Context, filter *AnalyticsFilter) ([]DailyStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			DATE(timestamp) as date,
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades
		FROM trade_history
		%s
		GROUP BY DATE(timestamp)
		ORDER BY date DESC
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询每日统计失败: %w", err)
	}
	defer rows.Close()

	var result []DailyStatistics
	for rows.Next() {
		stats := DailyStatistics{}
		var dateStr string
		var winningTrades int
		err := rows.Scan(
			&dateStr,
			&stats.TotalTrades,
			&stats.CompletedTrades,
			&stats.TotalPnL,
			&stats.TotalFees,
			&winningTrades,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描每日统计失败: %w", err)
		}

		// SQLite 的 DATE() 函数返回字符串格式 YYYY-MM-DD
		stats.Date = dateStr
		if stats.CompletedTrades > 0 {
			stats.WinRate = float64(winningTrades) / float64(stats.CompletedTrades) * 100.0
		}

		result = append(result, stats)
	}

	return result, nil
}

// GetWeeklyStats 获取每周统计
func (r *repository) GetWeeklyStats(ctx context.Context, filter *AnalyticsFilter) ([]WeeklyStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			strftime('%%Y-W%%W', timestamp) as week,
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades
		FROM trade_history
		%s
		GROUP BY strftime('%%Y-W%%W', timestamp)
		ORDER BY week DESC
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询每周统计失败: %w", err)
	}
	defer rows.Close()

	var result []WeeklyStatistics
	for rows.Next() {
		stats := WeeklyStatistics{}
		var winningTrades int
		err := rows.Scan(
			&stats.Week,
			&stats.TotalTrades,
			&stats.CompletedTrades,
			&stats.TotalPnL,
			&stats.TotalFees,
			&winningTrades,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描每周统计失败: %w", err)
		}

		if stats.CompletedTrades > 0 {
			stats.WinRate = float64(winningTrades) / float64(stats.CompletedTrades) * 100.0
		}

		result = append(result, stats)
	}

	return result, nil
}

// GetMonthlyStats 获取每月统计
func (r *repository) GetMonthlyStats(ctx context.Context, filter *AnalyticsFilter) ([]MonthlyStatistics, error) {
	whereClause, args := r.buildWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT 
			strftime('%%Y-%%m', timestamp) as month,
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as winning_trades
		FROM trade_history
		%s
		GROUP BY strftime('%%Y-%%m', timestamp)
		ORDER BY month DESC
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询每月统计失败: %w", err)
	}
	defer rows.Close()

	var result []MonthlyStatistics
	for rows.Next() {
		stats := MonthlyStatistics{}
		var winningTrades int
		err := rows.Scan(
			&stats.Month,
			&stats.TotalTrades,
			&stats.CompletedTrades,
			&stats.TotalPnL,
			&stats.TotalFees,
			&winningTrades,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描每月统计失败: %w", err)
		}

		if stats.CompletedTrades > 0 {
			stats.WinRate = float64(winningTrades) / float64(stats.CompletedTrades) * 100.0
		}

		result = append(result, stats)
	}

	return result, nil
}

// calculateDirectionPreference 计算方向偏好
func (r *repository) calculateDirectionPreference(ctx context.Context, filter *AnalyticsFilter, longStats, shortStats *SideStatistics) (*DirectionPreference, error) {
	preference := &DirectionPreference{
		TotalTrades: longStats.TotalTrades + shortStats.TotalTrades,
	}

	if preference.TotalTrades > 0 {
		preference.LongRatio = float64(longStats.TotalTrades) / float64(preference.TotalTrades) * 100.0
		preference.ShortRatio = float64(shortStats.TotalTrades) / float64(preference.TotalTrades) * 100.0

		// 确定偏好方向
		if longStats.TotalTrades > shortStats.TotalTrades {
			preference.PreferredSide = "long"
		} else if shortStats.TotalTrades > longStats.TotalTrades {
			preference.PreferredSide = "short"
		} else {
			preference.PreferredSide = "balanced" // 如果相等，标记为平衡
		}
	}

	return preference, nil
}

// GetFrequencyStats 获取交易频率统计
func (r *repository) GetFrequencyStats(ctx context.Context, filter *AnalyticsFilter) (*TradeFrequencyStats, error) {
	// 使用聚合后的记录（将同一订单的多笔执行合并）
	records, err := r.GetAggregatedRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	if len(records) == 0 {
		return &TradeFrequencyStats{}, nil
	}

	stats := &TradeFrequencyStats{}

	// 1. 计算日均交易数
	stats.DailyAverageTrades = calculateDailyAverageTrades(records, filter)

	// 2. 计算平均交易间隔
	stats.AvgTradeInterval = calculateAvgTradeInterval(records)

	// 3. 计算最活跃时段
	stats.MostActiveHours = calculateMostActiveHours(records)

	// 4. 计算最活跃日期
	stats.MostActiveDays = calculateMostActiveDays(records)

	return stats, nil
}

// calculateDailyAverageTrades 计算日均交易数
func calculateDailyAverageTrades(records []*TradeRecord, filter *AnalyticsFilter) float64 {
	if len(records) == 0 {
		return 0
	}

	// 确定时间范围
	var startTime, endTime time.Time
	if filter.StartTime != nil {
		startTime = *filter.StartTime
	} else {
		// 如果没有指定开始时间，使用第一条记录的时间
		// GetRecordsByFilter 已经按时间排序（timestamp ASC），所以第一条是最早的
		startTime = records[0].Timestamp
	}
	if filter.EndTime != nil {
		endTime = *filter.EndTime
	} else {
		// 如果没有指定结束时间，使用最后一条记录的时间
		// GetRecordsByFilter 已经按时间排序（timestamp ASC），所以最后一条是最晚的
		endTime = records[len(records)-1].Timestamp
	}

	// 计算天数（至少1天）
	days := endTime.Sub(startTime).Hours() / 24.0
	if days < 1.0 {
		days = 1.0
	}

	return float64(len(records)) / days
}

// calculateAvgTradeInterval 计算平均交易间隔（秒）
func calculateAvgTradeInterval(records []*TradeRecord) int64 {
	if len(records) < 2 {
		return 0
	}

	// 按时间排序
	sortedRecords := make([]*TradeRecord, len(records))
	copy(sortedRecords, records)
	sort.Slice(sortedRecords, func(i, j int) bool {
		return sortedRecords[i].Timestamp.Before(sortedRecords[j].Timestamp)
	})

	// 计算所有相邻交易之间的间隔总和
	var totalInterval int64
	for i := 1; i < len(sortedRecords); i++ {
		interval := sortedRecords[i].Timestamp.Sub(sortedRecords[i-1].Timestamp).Seconds()
		totalInterval += int64(interval)
	}

	// 返回平均间隔
	return totalInterval / int64(len(sortedRecords)-1)
}

// calculateMostActiveHours 计算最活跃时段（返回前5个最活跃的小时）
func calculateMostActiveHours(records []*TradeRecord) []int {
	if len(records) == 0 {
		return nil
	}

	// 统计每个小时的交易数
	hourCounts := make(map[int]int)
	for _, rec := range records {
		hour := rec.Timestamp.Hour()
		hourCounts[hour]++
	}

	// 转换为切片并排序
	type hourCount struct {
		hour  int
		count int
	}
	var hours []hourCount
	for hour, count := range hourCounts {
		hours = append(hours, hourCount{hour: hour, count: count})
	}

	// 按交易数降序排序
	sort.Slice(hours, func(i, j int) bool {
		if hours[i].count == hours[j].count {
			return hours[i].hour < hours[j].hour // 如果数量相同，按小时升序
		}
		return hours[i].count > hours[j].count
	})

	// 返回前5个最活跃的小时
	result := make([]int, 0, 5)
	for i := 0; i < len(hours) && i < 5; i++ {
		result = append(result, hours[i].hour)
	}

	return result
}

// calculateMostActiveDays 计算最活跃日期（返回前5个最活跃的日期）
func calculateMostActiveDays(records []*TradeRecord) []string {
	if len(records) == 0 {
		return nil
	}

	// 统计每个日期的交易数
	dayCounts := make(map[string]int)
	for _, rec := range records {
		date := rec.Timestamp.Format("2006-01-02")
		dayCounts[date]++
	}

	// 转换为切片并排序
	type dayCount struct {
		date  string
		count int
	}
	var days []dayCount
	for date, count := range dayCounts {
		days = append(days, dayCount{date: date, count: count})
	}

	// 按交易数降序排序
	sort.Slice(days, func(i, j int) bool {
		if days[i].count == days[j].count {
			return days[i].date > days[j].date // 如果数量相同，按日期降序（最新的在前）
		}
		return days[i].count > days[j].count
	})

	// 返回前5个最活跃的日期
	result := make([]string, 0, 5)
	for i := 0; i < len(days) && i < 5; i++ {
		result = append(result, days[i].date)
	}

	return result
}

// GetActionStats 获取交易类型统计
func (r *repository) GetActionStats(ctx context.Context, filter *AnalyticsFilter) (*ActionStatistics, error) {
	stats := &ActionStatistics{}

	// 获取开仓统计
	openStats, err := r.getActionTypeStats(ctx, filter, "open")
	if err != nil {
		return nil, fmt.Errorf("获取开仓统计失败: %w", err)
	}
	stats.OpenStats = openStats

	// 获取平仓统计
	closeStats, err := r.getActionTypeStats(ctx, filter, "close")
	if err != nil {
		return nil, fmt.Errorf("获取平仓统计失败: %w", err)
	}
	stats.CloseStats = closeStats

	return stats, nil
}

// getActionTypeStats 获取指定类型的统计（open 或 close）
func (r *repository) getActionTypeStats(ctx context.Context, filter *AnalyticsFilter, actionType string) (*ActionStats, error) {
	whereClause, args := r.buildWhereClause(filter)

	// 添加 action 类型过滤
	actionFilter := ""
	if actionType == "open" {
		actionFilter = "AND (action = 'open_long' OR action = 'open_short')"
	} else if actionType == "close" {
		actionFilter = "AND (action = 'close_long' OR action = 'close_short')"
	}

	// 构建完整的 WHERE 子句
	fullWhereClause := whereClause
	if whereClause == "" {
		fullWhereClause = "WHERE 1=1 " + actionFilter
	} else {
		fullWhereClause = whereClause + " " + actionFilter
	}

	// 查询统计信息
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_trades,
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN pnl ELSE 0 END), 0) as total_pnl,
			COALESCE(SUM(CASE WHEN pnl IS NOT NULL THEN 1 ELSE 0 END), 0) as completed_trades
		FROM trade_history
		%s
	`, fullWhereClause)

	stats := &ActionStats{}
	var completedTrades int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalTrades,
		&stats.TotalFees,
		&stats.TotalPnL,
		&completedTrades,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取交易类型统计失败: %w", err)
	}

	// 计算平均盈亏（仅平仓有 PnL）
	if completedTrades > 0 {
		stats.AvgPnL = stats.TotalPnL / float64(completedTrades)
	}

	return stats, nil
}

// GetTrendAnalysis 获取趋势分析
func (r *repository) GetTrendAnalysis(ctx context.Context, filter *AnalyticsFilter) (*TrendAnalysis, error) {
	// 使用聚合后的记录（将同一订单的多笔执行合并）
	records, err := r.GetAggregatedRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	if len(records) == 0 {
		return &TrendAnalysis{}, nil
	}

	analysis := &TrendAnalysis{}

	// 1. 计算累计盈亏曲线
	analysis.CumulativePnLSeries = calculateCumulativePnLSeries(records)

	// 2. 计算盈亏分布
	analysis.PnLDistribution = calculatePnLDistribution(records)

	// 3. 计算交易频率趋势
	analysis.TradeFrequencyTrend = calculateTradeFrequencyTrend(records)

	return analysis, nil
}

// calculateCumulativePnLSeries 计算累计盈亏曲线
func calculateCumulativePnLSeries(records []*TradeRecord) []CumulativePnLPoint {
	// 过滤出有PnL的记录并按时间排序
	var pnlRecords []*TradeRecord
	for _, rec := range records {
		if rec.PnL != nil {
			pnlRecords = append(pnlRecords, rec)
		}
	}

	if len(pnlRecords) == 0 {
		return nil
	}

	// 按时间排序
	sort.Slice(pnlRecords, func(i, j int) bool {
		return pnlRecords[i].Timestamp.Before(pnlRecords[j].Timestamp)
	})

	// 计算累计盈亏
	var series []CumulativePnLPoint
	var cumulativePnL float64
	for _, rec := range pnlRecords {
		cumulativePnL += *rec.PnL
		series = append(series, CumulativePnLPoint{
			Timestamp:     rec.Timestamp,
			CumulativePnL: cumulativePnL,
		})
	}

	return series
}

// calculatePnLDistribution 计算盈亏分布
func calculatePnLDistribution(records []*TradeRecord) []PnLDistributionBin {
	// 过滤出有PnL的记录
	var pnlValues []float64
	for _, rec := range records {
		if rec.PnL != nil {
			pnlValues = append(pnlValues, *rec.PnL)
		}
	}

	if len(pnlValues) == 0 {
		return nil
	}

	// 找到最小值和最大值
	minPnL := pnlValues[0]
	maxPnL := pnlValues[0]
	for _, pnl := range pnlValues {
		if pnl < minPnL {
			minPnL = pnl
		}
		if pnl > maxPnL {
			maxPnL = pnl
		}
	}

	// 确定区间大小（使用合理的区间数，比如20个区间）
	numBins := 20
	rangeSize := (maxPnL - minPnL) / float64(numBins)
	if rangeSize == 0 {
		// 如果所有值相同，使用固定区间
		rangeSize = 1.0
	}

	// 创建分布区间
	bins := make(map[string]int)
	for _, pnl := range pnlValues {
		binIndex := int((pnl - minPnL) / rangeSize)
		if binIndex >= numBins {
			binIndex = numBins - 1
		}
		if binIndex < 0 {
			binIndex = 0
		}

		binStart := minPnL + float64(binIndex)*rangeSize
		binEnd := binStart + rangeSize
		rangeKey := fmt.Sprintf("%.2f-%.2f", binStart, binEnd)
		bins[rangeKey]++
	}

	// 转换为切片并排序
	type binEntry struct {
		rangeKey string
		count    int
		start    float64
	}
	var entries []binEntry
	for rangeKey, count := range bins {
		// 解析范围开始值用于排序
		var start float64
		fmt.Sscanf(rangeKey, "%f-", &start)
		entries = append(entries, binEntry{
			rangeKey: rangeKey,
			count:    count,
			start:    start,
		})
	}

	// 按范围开始值排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].start < entries[j].start
	})

	// 转换为结果
	result := make([]PnLDistributionBin, len(entries))
	for i, entry := range entries {
		result[i] = PnLDistributionBin{
			Range: entry.rangeKey,
			Count: entry.count,
		}
	}

	return result
}

// calculateTradeFrequencyTrend 计算交易频率趋势
func calculateTradeFrequencyTrend(records []*TradeRecord) []TradeFrequencyPoint {
	if len(records) == 0 {
		return nil
	}

	// 按时间排序
	sortedRecords := make([]*TradeRecord, len(records))
	copy(sortedRecords, records)
	sort.Slice(sortedRecords, func(i, j int) bool {
		return sortedRecords[i].Timestamp.Before(sortedRecords[j].Timestamp)
	})

	// 按小时分组统计交易频率
	hourlyCounts := make(map[string]int)
	for _, rec := range sortedRecords {
		// 使用小时作为键（格式：YYYY-MM-DD HH:00:00）
		hourKey := rec.Timestamp.Format("2006-01-02 15:00:00")
		hourlyCounts[hourKey]++
	}

	// 转换为时间点并排序
	type freqEntry struct {
		timestamp time.Time
		count     int
	}
	var entries []freqEntry
	for hourKey, count := range hourlyCounts {
		timestamp, err := time.Parse("2006-01-02 15:04:05", hourKey)
		if err != nil {
			continue
		}
		entries = append(entries, freqEntry{
			timestamp: timestamp,
			count:     count,
		})
	}

	// 按时间排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	// 转换为结果
	result := make([]TradeFrequencyPoint, len(entries))
	for i, entry := range entries {
		result[i] = TradeFrequencyPoint{
			Timestamp:  entry.timestamp,
			TradeCount: entry.count,
		}
	}

	return result
}


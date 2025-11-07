package trade_history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository 交易历史数据仓库接口
type Repository interface {
	// Save 保存交易记录
	Save(ctx context.Context, record *TradeRecord) error

	// SaveBatch 批量保存交易记录
	SaveBatch(ctx context.Context, records []*TradeRecord) error

	// FindByID 根据ID查询
	FindByID(ctx context.Context, id int64) (*TradeRecord, error)

	// FindByFilter 根据过滤器查询
	FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error)

	// CountByFilter 统计数量
	CountByFilter(ctx context.Context, filter *TradeRecordFilter) (int, error)

	// ExistsByExchangeID 检查是否存在（通过交易所订单ID或交易ID）
	ExistsByExchangeID(ctx context.Context, exchangeOid *int64, exchangeTid *int64, exchangeHash *string) (bool, error)

	// UpdateExecutionPrice 更新成交价格
	UpdateExecutionPrice(ctx context.Context, id int64, price float64) error

	// GetStatistics 获取统计信息
	GetStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error)

	// GetLatestByTrader 获取交易员最近的交易记录
	GetLatestByTrader(ctx context.Context, traderID string, limit int) ([]*TradeRecord, error)

	// ValidateTraderExists 验证trader是否存在（可选，用于数据一致性检查）
	ValidateTraderExists(ctx context.Context, traderID string) (bool, error)
}

// repository 实现
type repository struct {
	db *sql.DB
}

// NewRepository 创建Repository实例
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Save 保存交易记录
func (r *repository) Save(ctx context.Context, record *TradeRecord) error {
	query := `
		INSERT INTO trade_history (
			trader_id, symbol, side, action, quantity, leverage,
			entry_price, exit_price, execution_price,
			pnl, pnl_pct,
			order_id, exchange_order_id, exchange_trade_id, exchange_hash,
			fee, fee_token,
			is_auto_triggered, was_stop_loss, was_take_profit,
			cycle_number, source,
			timestamp, exchange_timestamp, exchange_timestamp_ms,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	// 处理可选字段
	var entryPrice, exitPrice, pnl, pnlPct sql.NullFloat64
	if record.EntryPrice != nil {
		entryPrice = sql.NullFloat64{Float64: *record.EntryPrice, Valid: true}
	}
	if record.ExitPrice != nil {
		exitPrice = sql.NullFloat64{Float64: *record.ExitPrice, Valid: true}
	}
	if record.PnL != nil {
		pnl = sql.NullFloat64{Float64: *record.PnL, Valid: true}
	}
	if record.PnLPct != nil {
		pnlPct = sql.NullFloat64{Float64: *record.PnLPct, Valid: true}
	}

	var exchangeTimestamp sql.NullTime
	if record.ExchangeTimestamp != nil {
		exchangeTimestamp = sql.NullTime{Time: *record.ExchangeTimestamp, Valid: true}
	}

	var exchangeTimestampMs sql.NullInt64
	if record.ExchangeTimestampMs != nil {
		exchangeTimestampMs = sql.NullInt64{Int64: *record.ExchangeTimestampMs, Valid: true}
	}

	var orderID, exchangeOrderID, exchangeTradeID, exchangeHash, feeToken sql.NullString
	var cycleNumber sql.NullInt64

	if record.OrderID != nil {
		orderID = sql.NullString{String: *record.OrderID, Valid: true}
	}
	if record.ExchangeOrderID != nil {
		exchangeOrderID = sql.NullString{String: *record.ExchangeOrderID, Valid: true}
	}
	if record.ExchangeTradeID != nil {
		exchangeTradeID = sql.NullString{String: *record.ExchangeTradeID, Valid: true}
	}
	if record.ExchangeHash != nil {
		exchangeHash = sql.NullString{String: *record.ExchangeHash, Valid: true}
	}
	if record.FeeToken != nil {
		feeToken = sql.NullString{String: *record.FeeToken, Valid: true}
	}
	if record.CycleNumber != nil {
		cycleNumber = sql.NullInt64{Int64: int64(*record.CycleNumber), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		record.TraderID, record.Symbol, record.Side, record.Action,
		record.Quantity, record.Leverage,
		entryPrice, exitPrice, record.ExecutionPrice,
		pnl, pnlPct,
		orderID, exchangeOrderID, exchangeTradeID, exchangeHash,
		record.Fee, feeToken,
		record.IsAutoTriggered, record.WasStopLoss, record.WasTakeProfit,
		cycleNumber, record.Source,
		record.Timestamp, exchangeTimestamp, exchangeTimestampMs,
	)
	if err != nil {
		return fmt.Errorf("保存交易记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		record.ID = id
	}

	return nil
}

// SaveBatch 批量保存交易记录
func (r *repository) SaveBatch(ctx context.Context, records []*TradeRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO trade_history (
			trader_id, symbol, side, action, quantity, leverage,
			entry_price, exit_price, execution_price,
			pnl, pnl_pct,
			order_id, exchange_order_id, exchange_trade_id, exchange_hash,
			fee, fee_token,
			is_auto_triggered, was_stop_loss, was_take_profit,
			cycle_number, source,
			timestamp, exchange_timestamp, exchange_timestamp_ms,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("准备语句失败: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// 处理可选字段
		var entryPrice, exitPrice, pnl, pnlPct sql.NullFloat64
		if record.EntryPrice != nil {
			entryPrice = sql.NullFloat64{Float64: *record.EntryPrice, Valid: true}
		}
		if record.ExitPrice != nil {
			exitPrice = sql.NullFloat64{Float64: *record.ExitPrice, Valid: true}
		}
		if record.PnL != nil {
			pnl = sql.NullFloat64{Float64: *record.PnL, Valid: true}
		}
		if record.PnLPct != nil {
			pnlPct = sql.NullFloat64{Float64: *record.PnLPct, Valid: true}
		}

		var exchangeTimestamp sql.NullTime
		if record.ExchangeTimestamp != nil {
			exchangeTimestamp = sql.NullTime{Time: *record.ExchangeTimestamp, Valid: true}
		}

		var exchangeTimestampMs sql.NullInt64
		if record.ExchangeTimestampMs != nil {
			exchangeTimestampMs = sql.NullInt64{Int64: *record.ExchangeTimestampMs, Valid: true}
		}

		var orderID, exchangeOrderID, exchangeTradeID, exchangeHash, feeToken sql.NullString
		var cycleNumber sql.NullInt64

		if record.OrderID != nil {
			orderID = sql.NullString{String: *record.OrderID, Valid: true}
		}
		if record.ExchangeOrderID != nil {
			exchangeOrderID = sql.NullString{String: *record.ExchangeOrderID, Valid: true}
		}
		if record.ExchangeTradeID != nil {
			exchangeTradeID = sql.NullString{String: *record.ExchangeTradeID, Valid: true}
		}
		if record.ExchangeHash != nil {
			exchangeHash = sql.NullString{String: *record.ExchangeHash, Valid: true}
		}
		if record.FeeToken != nil {
			feeToken = sql.NullString{String: *record.FeeToken, Valid: true}
		}
		if record.CycleNumber != nil {
			cycleNumber = sql.NullInt64{Int64: int64(*record.CycleNumber), Valid: true}
		}

		_, err := stmt.ExecContext(ctx,
			record.TraderID, record.Symbol, record.Side, record.Action,
			record.Quantity, record.Leverage,
			entryPrice, exitPrice, record.ExecutionPrice,
			pnl, pnlPct,
			orderID, exchangeOrderID, exchangeTradeID, exchangeHash,
			record.Fee, feeToken,
			record.IsAutoTriggered, record.WasStopLoss, record.WasTakeProfit,
			cycleNumber, record.Source,
			record.Timestamp, exchangeTimestamp, exchangeTimestampMs,
		)
		if err != nil {
			return fmt.Errorf("批量保存交易记录失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// FindByID 根据ID查询
func (r *repository) FindByID(ctx context.Context, id int64) (*TradeRecord, error) {
	query := `SELECT 
		id, trader_id, symbol, side, action, quantity, leverage,
		entry_price, exit_price, execution_price,
		pnl, pnl_pct,
		order_id, exchange_order_id, exchange_trade_id, exchange_hash,
		fee, fee_token,
		is_auto_triggered, was_stop_loss, was_take_profit,
		cycle_number, source,
		timestamp, exchange_timestamp, exchange_timestamp_ms,
		created_at, updated_at
		FROM trade_history WHERE id = ?`

	record := &TradeRecord{}
	var entryPrice, exitPrice, pnl, pnlPct sql.NullFloat64
	var exchangeTimestamp sql.NullTime
	var exchangeTimestampMs sql.NullInt64
	var orderID, exchangeOrderID, exchangeTradeID, exchangeHash, feeToken sql.NullString
	var cycleNumber sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.TraderID, &record.Symbol, &record.Side, &record.Action,
		&record.Quantity, &record.Leverage,
		&entryPrice, &exitPrice, &record.ExecutionPrice,
		&pnl, &pnlPct,
		&orderID, &exchangeOrderID, &exchangeTradeID, &exchangeHash,
		&record.Fee, &feeToken,
		&record.IsAutoTriggered, &record.WasStopLoss, &record.WasTakeProfit,
		&cycleNumber, &record.Source,
		&record.Timestamp, &exchangeTimestamp, &exchangeTimestampMs,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("交易记录不存在: %w", err)
		}
		return nil, fmt.Errorf("查询交易记录失败: %w", err)
	}

	// 处理可选字段
	if entryPrice.Valid {
		record.EntryPrice = &entryPrice.Float64
	}
	if exitPrice.Valid {
		record.ExitPrice = &exitPrice.Float64
	}
	if pnl.Valid {
		record.PnL = &pnl.Float64
	}
	if pnlPct.Valid {
		record.PnLPct = &pnlPct.Float64
	}
	if exchangeTimestamp.Valid {
		record.ExchangeTimestamp = &exchangeTimestamp.Time
	}
	if exchangeTimestampMs.Valid {
		ms := exchangeTimestampMs.Int64
		record.ExchangeTimestampMs = &ms
	}
	if orderID.Valid {
		record.OrderID = &orderID.String
	}
	if exchangeOrderID.Valid {
		record.ExchangeOrderID = &exchangeOrderID.String
	}
	if exchangeTradeID.Valid {
		record.ExchangeTradeID = &exchangeTradeID.String
	}
	if exchangeHash.Valid {
		record.ExchangeHash = &exchangeHash.String
	}
	if feeToken.Valid {
		record.FeeToken = &feeToken.String
	}
	if cycleNumber.Valid {
		cn := int(cycleNumber.Int64)
		record.CycleNumber = &cn
	}

	return record, nil
}

// FindByFilter 根据过滤器查询
func (r *repository) FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error) {
	var conditions []string
	var args []interface{}

	// 构建WHERE条件
	if filter.TraderID != "" {
		conditions = append(conditions, "trader_id = ?")
		args = append(args, filter.TraderID)
	}
	if filter.Symbol != "" {
		conditions = append(conditions, "symbol = ?")
		args = append(args, filter.Symbol)
	}
	if filter.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.Side != "" {
		conditions = append(conditions, "side = ?")
		args = append(args, filter.Side)
	}
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}
	if filter.CycleFrom != nil {
		conditions = append(conditions, "cycle_number >= ?")
		args = append(args, *filter.CycleFrom)
	}
	if filter.CycleTo != nil {
		conditions = append(conditions, "cycle_number <= ?")
		args = append(args, *filter.CycleTo)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 设置默认值
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "timestamp DESC"
	}

	query := fmt.Sprintf(`
		SELECT 
			id, trader_id, symbol, side, action, quantity, leverage,
			entry_price, exit_price, execution_price,
			pnl, pnl_pct,
			order_id, exchange_order_id, exchange_trade_id, exchange_hash,
			fee, fee_token,
			is_auto_triggered, was_stop_loss, was_take_profit,
			cycle_number, source,
			timestamp, exchange_timestamp, exchange_timestamp_ms,
			created_at, updated_at
		FROM trade_history
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, whereClause, orderBy)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询交易记录失败: %w", err)
	}
	defer rows.Close()

	var records []*TradeRecord
	for rows.Next() {
		record := &TradeRecord{}
		var entryPrice, exitPrice, pnl, pnlPct sql.NullFloat64
		var exchangeTimestamp sql.NullTime
		var exchangeTimestampMs sql.NullInt64
		var orderID, exchangeOrderID, exchangeTradeID, exchangeHash, feeToken sql.NullString
		var cycleNumber sql.NullInt64

		err := rows.Scan(
			&record.ID, &record.TraderID, &record.Symbol, &record.Side, &record.Action,
			&record.Quantity, &record.Leverage,
			&entryPrice, &exitPrice, &record.ExecutionPrice,
			&pnl, &pnlPct,
			&orderID, &exchangeOrderID, &exchangeTradeID, &exchangeHash,
			&record.Fee, &feeToken,
			&record.IsAutoTriggered, &record.WasStopLoss, &record.WasTakeProfit,
			&cycleNumber, &record.Source,
			&record.Timestamp, &exchangeTimestamp, &exchangeTimestampMs,
			&record.CreatedAt, &record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %w", err)
		}

		// 处理可选字段
		if entryPrice.Valid {
			record.EntryPrice = &entryPrice.Float64
		}
		if exitPrice.Valid {
			record.ExitPrice = &exitPrice.Float64
		}
		if pnl.Valid {
			record.PnL = &pnl.Float64
		}
		if pnlPct.Valid {
			record.PnLPct = &pnlPct.Float64
		}
		if exchangeTimestamp.Valid {
			record.ExchangeTimestamp = &exchangeTimestamp.Time
		}
		if exchangeTimestampMs.Valid {
			ms := exchangeTimestampMs.Int64
			record.ExchangeTimestampMs = &ms
		}
		if orderID.Valid {
			record.OrderID = &orderID.String
		}
		if exchangeOrderID.Valid {
			record.ExchangeOrderID = &exchangeOrderID.String
		}
		if exchangeTradeID.Valid {
			record.ExchangeTradeID = &exchangeTradeID.String
		}
		if exchangeHash.Valid {
			record.ExchangeHash = &exchangeHash.String
		}
		if feeToken.Valid {
			record.FeeToken = &feeToken.String
		}
		if cycleNumber.Valid {
			cn := int(cycleNumber.Int64)
			record.CycleNumber = &cn
		}

		records = append(records, record)
	}

	return records, nil
}

// CountByFilter 统计数量
func (r *repository) CountByFilter(ctx context.Context, filter *TradeRecordFilter) (int, error) {
	var conditions []string
	var args []interface{}

	// 构建WHERE条件
	if filter.TraderID != "" {
		conditions = append(conditions, "trader_id = ?")
		args = append(args, filter.TraderID)
	}
	if filter.Symbol != "" {
		conditions = append(conditions, "symbol = ?")
		args = append(args, filter.Symbol)
	}
	if filter.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.Side != "" {
		conditions = append(conditions, "side = ?")
		args = append(args, filter.Side)
	}
	if filter.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filter.EndTime)
	}
	if filter.CycleFrom != nil {
		conditions = append(conditions, "cycle_number >= ?")
		args = append(args, *filter.CycleFrom)
	}
	if filter.CycleTo != nil {
		conditions = append(conditions, "cycle_number <= ?")
		args = append(args, *filter.CycleTo)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM trade_history %s", whereClause)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计交易记录失败: %w", err)
	}

	return count, nil
}

// ExistsByExchangeID 检查是否存在（通过交易所订单ID或交易ID）
func (r *repository) ExistsByExchangeID(ctx context.Context, exchangeOid *int64, exchangeTid *int64, exchangeHash *string) (bool, error) {
	var conditions []string
	var args []interface{}

	if exchangeOid != nil {
		conditions = append(conditions, "exchange_order_id = ?")
		args = append(args, fmt.Sprintf("%d", *exchangeOid))
	}
	if exchangeTid != nil {
		conditions = append(conditions, "exchange_trade_id = ?")
		args = append(args, fmt.Sprintf("%d", *exchangeTid))
	}
	if exchangeHash != nil && *exchangeHash != "" {
		conditions = append(conditions, "exchange_hash = ?")
		args = append(args, *exchangeHash)
	}

	if len(conditions) == 0 {
		return false, nil
	}

	whereClause := "WHERE " + strings.Join(conditions, " OR ")
	query := fmt.Sprintf("SELECT COUNT(*) FROM trade_history %s LIMIT 1", whereClause)

	var recordCount int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&recordCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("检查交易记录是否存在失败: %w", err)
	}

	return recordCount > 0, nil
}

// UpdateExecutionPrice 更新成交价格
func (r *repository) UpdateExecutionPrice(ctx context.Context, id int64, price float64) error {
	query := `UPDATE trade_history SET execution_price = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, price, id)
	if err != nil {
		return fmt.Errorf("更新成交价格失败: %w", err)
	}
	return nil
}

// GetStatistics 获取统计信息
func (r *repository) GetStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "trader_id = ?")
	args = append(args, traderID)

	// 只统计平仓记录（有PnL的记录）
	conditions = append(conditions, "pnl IS NOT NULL")

	if startTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *endTime)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_trades,
			SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(AVG(CASE WHEN pnl > 0 THEN pnl END), 0) as avg_win,
			COALESCE(AVG(CASE WHEN pnl < 0 THEN pnl END), 0) as avg_loss,
			COALESCE(SUM(fee), 0) as total_fees
		FROM trade_history
		%s
	`, whereClause)

	stats := &TradeStatistics{}
	var winningTrades, losingTrades sql.NullInt64
	var totalPnL, avgWin, avgLoss, totalFees sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalTrades,
		&winningTrades,
		&losingTrades,
		&totalPnL,
		&avgWin,
		&avgLoss,
		&totalFees,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil
		}
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}

	if winningTrades.Valid {
		stats.WinningTrades = int(winningTrades.Int64)
	}
	if losingTrades.Valid {
		stats.LosingTrades = int(losingTrades.Int64)
	}
	if totalPnL.Valid {
		stats.TotalPnL = totalPnL.Float64
	}
	if avgWin.Valid {
		stats.AvgWin = avgWin.Float64
	}
	if avgLoss.Valid {
		stats.AvgLoss = avgLoss.Float64
	}
	if totalFees.Valid {
		stats.TotalFees = totalFees.Float64
	}

	// 计算胜率和盈亏比
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100.0
	}
	if stats.AvgLoss != 0 {
		stats.ProfitFactor = stats.AvgWin / -stats.AvgLoss
	}

	return stats, nil
}

// GetLatestByTrader 获取交易员最近的交易记录
func (r *repository) GetLatestByTrader(ctx context.Context, traderID string, limit int) ([]*TradeRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	filter := &TradeRecordFilter{
		TraderID: traderID,
		Limit:    limit,
		OrderBy:  "timestamp DESC",
	}

	return r.FindByFilter(ctx, filter)
}

// ValidateTraderExists 验证trader是否存在（可选，用于数据一致性检查）
func (r *repository) ValidateTraderExists(ctx context.Context, traderID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM traders WHERE id = ?)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, traderID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("验证trader是否存在失败: %w", err)
	}
	return exists, nil
}


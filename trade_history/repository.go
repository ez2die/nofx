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

	// GetRecentTrades 获取最近的平仓交易
	GetRecentTrades(ctx context.Context, traderID string, limit int, startTime, endTime *time.Time) ([]*TradeOutcome, error)

	// GetSymbolStats 获取币种表现
	GetSymbolStats(ctx context.Context, traderID string, startTime, endTime *time.Time) ([]*SymbolPerformance, error)

	// GetPnLSeries 获取PnL序列（用于Sharpe计算）
	GetPnLSeries(ctx context.Context, traderID string, startTime, endTime *time.Time, limit int) ([]*PnLPoint, error)

	// GetLatestByTrader 获取交易员最近的交易记录
	GetLatestByTrader(ctx context.Context, traderID string, limit int) ([]*TradeRecord, error)

	// GetSyncState 获取同步游标（毫秒时间戳）
	GetSyncState(ctx context.Context, traderID string) (int64, error)

	// UpsertSyncState 写入同步游标
	UpsertSyncState(ctx context.Context, traderID string, timestampMs int64) error
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
			trader_id, symbol, action, side,
			quantity, signed_quantity,
			execution_price, pnl,
			fee, fee_token,
			raw_dir, start_position, builder_fee, exchange_side,
			exchange_order_id, exchange_trade_id, exchange_hash,
			timestamp, exchange_timestamp, exchange_timestamp_ms,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	var signedQty sql.NullFloat64
	if record.SignedQuantity != nil {
		signedQty = sql.NullFloat64{Float64: *record.SignedQuantity, Valid: true}
	}

	var pnl sql.NullFloat64
	if record.PnL != nil {
		pnl = sql.NullFloat64{Float64: *record.PnL, Valid: true}
	}

	var feeToken sql.NullString
	if record.FeeToken != nil {
		feeToken = sql.NullString{String: *record.FeeToken, Valid: true}
	}

	var startPosition, builderFee sql.NullFloat64
	if record.StartPosition != nil {
		startPosition = sql.NullFloat64{Float64: *record.StartPosition, Valid: true}
	}
	if record.BuilderFee != nil {
		builderFee = sql.NullFloat64{Float64: *record.BuilderFee, Valid: true}
	}

	var exchangeSide sql.NullString
	if record.ExchangeSide != nil {
		exchangeSide = sql.NullString{String: *record.ExchangeSide, Valid: true}
	}

	var exchangeOrderID, exchangeTradeID, exchangeHash sql.NullString
	if record.ExchangeOrderID != nil {
		exchangeOrderID = sql.NullString{String: *record.ExchangeOrderID, Valid: true}
	}
	if record.ExchangeTradeID != nil {
		exchangeTradeID = sql.NullString{String: *record.ExchangeTradeID, Valid: true}
	}
	if record.ExchangeHash != nil {
		exchangeHash = sql.NullString{String: *record.ExchangeHash, Valid: true}
	}

	var exchangeTimestamp sql.NullTime
	if record.ExchangeTimestamp != nil {
		exchangeTimestamp = sql.NullTime{Time: *record.ExchangeTimestamp, Valid: true}
	}

	var exchangeTimestampMs sql.NullInt64
	if record.ExchangeTimestampMs != nil {
		exchangeTimestampMs = sql.NullInt64{Int64: *record.ExchangeTimestampMs, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		record.TraderID, record.Symbol, record.Action, record.Side,
		record.Quantity, signedQty,
		record.ExecutionPrice, pnl,
		record.Fee, feeToken,
		record.RawDir, startPosition, builderFee, exchangeSide,
		exchangeOrderID, exchangeTradeID, exchangeHash,
		record.Timestamp, exchangeTimestamp, exchangeTimestampMs,
	)
	if err != nil {
		return fmt.Errorf("保存交易记录失败: %w", err)
	}

	if id, err := result.LastInsertId(); err == nil {
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
			trader_id, symbol, action, side,
			quantity, signed_quantity,
			execution_price, pnl,
			fee, fee_token,
			raw_dir, start_position, builder_fee, exchange_side,
			exchange_order_id, exchange_trade_id, exchange_hash,
			timestamp, exchange_timestamp, exchange_timestamp_ms,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?,
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
		var signedQty sql.NullFloat64
		if record.SignedQuantity != nil {
			signedQty = sql.NullFloat64{Float64: *record.SignedQuantity, Valid: true}
		}

		var pnl sql.NullFloat64
		if record.PnL != nil {
			pnl = sql.NullFloat64{Float64: *record.PnL, Valid: true}
		}

		var feeToken sql.NullString
		if record.FeeToken != nil {
			feeToken = sql.NullString{String: *record.FeeToken, Valid: true}
		}

		var startPosition, builderFee sql.NullFloat64
		if record.StartPosition != nil {
			startPosition = sql.NullFloat64{Float64: *record.StartPosition, Valid: true}
		}
		if record.BuilderFee != nil {
			builderFee = sql.NullFloat64{Float64: *record.BuilderFee, Valid: true}
		}

		var exchangeSide sql.NullString
		if record.ExchangeSide != nil {
			exchangeSide = sql.NullString{String: *record.ExchangeSide, Valid: true}
		}

		var exchangeOrderID, exchangeTradeID, exchangeHash sql.NullString
		if record.ExchangeOrderID != nil {
			exchangeOrderID = sql.NullString{String: *record.ExchangeOrderID, Valid: true}
		}
		if record.ExchangeTradeID != nil {
			exchangeTradeID = sql.NullString{String: *record.ExchangeTradeID, Valid: true}
		}
		if record.ExchangeHash != nil {
			exchangeHash = sql.NullString{String: *record.ExchangeHash, Valid: true}
		}

		var exchangeTimestamp sql.NullTime
		if record.ExchangeTimestamp != nil {
			exchangeTimestamp = sql.NullTime{Time: *record.ExchangeTimestamp, Valid: true}
		}

		var exchangeTimestampMs sql.NullInt64
		if record.ExchangeTimestampMs != nil {
			exchangeTimestampMs = sql.NullInt64{Int64: *record.ExchangeTimestampMs, Valid: true}
		}

		_, err := stmt.ExecContext(ctx,
			record.TraderID, record.Symbol, record.Action, record.Side,
			record.Quantity, signedQty,
			record.ExecutionPrice, pnl,
			record.Fee, feeToken,
			record.RawDir, startPosition, builderFee, exchangeSide,
			exchangeOrderID, exchangeTradeID, exchangeHash,
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
		id, trader_id, symbol, action, side,
		quantity, signed_quantity,
		execution_price, pnl,
		fee, fee_token,
		raw_dir, start_position, builder_fee, exchange_side,
		exchange_order_id, exchange_trade_id, exchange_hash,
		timestamp, exchange_timestamp, exchange_timestamp_ms,
		created_at, updated_at
		FROM trade_history WHERE id = ?`

	record := &TradeRecord{}
	var signedQty, pnl sql.NullFloat64
	var feeToken sql.NullString
	var startPosition, builderFee sql.NullFloat64
	var exchangeSide sql.NullString
	var exchangeOrderID, exchangeTradeID, exchangeHash sql.NullString
	var exchangeTimestamp sql.NullTime
	var exchangeTimestampMs sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.TraderID, &record.Symbol, &record.Action, &record.Side,
		&record.Quantity, &signedQty,
		&record.ExecutionPrice, &pnl,
		&record.Fee, &feeToken,
		&record.RawDir, &startPosition, &builderFee, &exchangeSide,
		&exchangeOrderID, &exchangeTradeID, &exchangeHash,
		&record.Timestamp, &exchangeTimestamp, &exchangeTimestampMs,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("交易记录不存在: %w", err)
		}
		return nil, fmt.Errorf("查询交易记录失败: %w", err)
	}

	if signedQty.Valid {
		record.SignedQuantity = &signedQty.Float64
	}
	if pnl.Valid {
		record.PnL = &pnl.Float64
	}
	if feeToken.Valid {
		record.FeeToken = &feeToken.String
	}
	if startPosition.Valid {
		record.StartPosition = &startPosition.Float64
	}
	if builderFee.Valid {
		record.BuilderFee = &builderFee.Float64
	}
	if exchangeSide.Valid {
		side := exchangeSide.String
		record.ExchangeSide = &side
	}
	if exchangeOrderID.Valid {
		idStr := exchangeOrderID.String
		record.ExchangeOrderID = &idStr
	}
	if exchangeTradeID.Valid {
		idStr := exchangeTradeID.String
		record.ExchangeTradeID = &idStr
	}
	if exchangeHash.Valid {
		hash := exchangeHash.String
		record.ExchangeHash = &hash
	}
	if exchangeTimestamp.Valid {
		record.ExchangeTimestamp = &exchangeTimestamp.Time
	}
	if exchangeTimestampMs.Valid {
		ms := exchangeTimestampMs.Int64
		record.ExchangeTimestampMs = &ms
	}

	return record, nil
}

// FindByFilter 根据过滤器查询
func (r *repository) FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error) {
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

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

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
			id, trader_id, symbol, action, side,
			quantity, signed_quantity,
			execution_price, pnl,
			fee, fee_token,
			raw_dir, start_position, builder_fee, exchange_side,
			exchange_order_id, exchange_trade_id, exchange_hash,
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
		var signedQty, pnl sql.NullFloat64
		var feeToken sql.NullString
		var startPosition, builderFee sql.NullFloat64
		var exchangeSide sql.NullString
		var exchangeOrderID, exchangeTradeID, exchangeHash sql.NullString
		var exchangeTimestamp sql.NullTime
		var exchangeTimestampMs sql.NullInt64

		err := rows.Scan(
			&record.ID, &record.TraderID, &record.Symbol, &record.Action, &record.Side,
			&record.Quantity, &signedQty,
			&record.ExecutionPrice, &pnl,
			&record.Fee, &feeToken,
			&record.RawDir, &startPosition, &builderFee, &exchangeSide,
			&exchangeOrderID, &exchangeTradeID, &exchangeHash,
			&record.Timestamp, &exchangeTimestamp, &exchangeTimestampMs,
			&record.CreatedAt, &record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描交易记录失败: %w", err)
		}

		if signedQty.Valid {
			record.SignedQuantity = &signedQty.Float64
		}
		if pnl.Valid {
			record.PnL = &pnl.Float64
		}
		if feeToken.Valid {
			token := feeToken.String
			record.FeeToken = &token
		}
		if startPosition.Valid {
			record.StartPosition = &startPosition.Float64
		}
		if builderFee.Valid {
			record.BuilderFee = &builderFee.Float64
		}
		if exchangeSide.Valid {
			side := exchangeSide.String
			record.ExchangeSide = &side
		}
		if exchangeOrderID.Valid {
			idStr := exchangeOrderID.String
			record.ExchangeOrderID = &idStr
		}
		if exchangeTradeID.Valid {
			idStr := exchangeTradeID.String
			record.ExchangeTradeID = &idStr
		}
		if exchangeHash.Valid {
			hash := exchangeHash.String
			record.ExchangeHash = &hash
		}
		if exchangeTimestamp.Valid {
			record.ExchangeTimestamp = &exchangeTimestamp.Time
		}
		if exchangeTimestampMs.Valid {
			ms := exchangeTimestampMs.Int64
			record.ExchangeTimestampMs = &ms
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
// 修复：优先使用 exchange_trade_id 作为唯一标识（因为一个订单可能有多个部分成交）
// 如果 exchange_trade_id 不存在，则使用 exchange_order_id + exchange_hash 的组合
func (r *repository) ExistsByExchangeID(ctx context.Context, exchangeOid *int64, exchangeTid *int64, exchangeHash *string) (bool, error) {
	// 优先使用 exchange_trade_id（最精确的唯一标识）
	if exchangeTid != nil {
		query := "SELECT COUNT(*) FROM trade_history WHERE exchange_trade_id = ? LIMIT 1"
		var recordCount int
		err := r.db.QueryRowContext(ctx, query, fmt.Sprintf("%d", *exchangeTid)).Scan(&recordCount)
		if err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, fmt.Errorf("检查交易记录是否存在失败: %w", err)
		}
		if recordCount > 0 {
			return true, nil
		}
	}

	// 如果没有 exchange_trade_id，使用 exchange_order_id + exchange_hash 的组合
	// 注意：不能单独使用 exchange_order_id，因为一个订单可能有多个部分成交
	var conditions []string
	var args []interface{}

	if exchangeOid != nil {
		conditions = append(conditions, "exchange_order_id = ?")
		args = append(args, fmt.Sprintf("%d", *exchangeOid))
	}
	if exchangeHash != nil && *exchangeHash != "" {
		conditions = append(conditions, "exchange_hash = ?")
		args = append(args, *exchangeHash)
	}

	// 如果同时有 exchange_order_id 和 exchange_hash，使用 AND 条件（更精确）
	// 如果只有其中一个，使用该条件
	if len(conditions) == 0 {
		return false, nil
	}

	var whereClause string
	if len(conditions) == 2 {
		// 同时有 order_id 和 hash，使用 AND（必须同时匹配）
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	} else {
		// 只有一个条件，直接使用
		whereClause = "WHERE " + conditions[0]
	}

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

// GetRecentTrades 获取最近的平仓交易
func (r *repository) GetRecentTrades(ctx context.Context, traderID string, limit int, startTime, endTime *time.Time) ([]*TradeOutcome, error) {
	if limit <= 0 {
		limit = 20
	}

	conditions := []string{"trader_id = ?", "pnl IS NOT NULL"}
	args := []interface{}{traderID}

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
		SELECT symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp, exchange_timestamp
		FROM trade_history
		%s
		ORDER BY timestamp DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询最近交易失败: %w", err)
	}
	defer rows.Close()

	var results []*TradeOutcome
	for rows.Next() {
		var (
			outcome      TradeOutcome
			signedQty    sql.NullFloat64
			exchangeTime sql.NullTime
		)
		if err := rows.Scan(
			&outcome.Symbol,
			&outcome.Action,
			&outcome.Side,
			&outcome.Quantity,
			&signedQty,
			&outcome.ExecutionPrice,
			&outcome.PnL,
			&outcome.Fee,
			&outcome.Timestamp,
			&exchangeTime,
		); err != nil {
			return nil, fmt.Errorf("扫描最近交易失败: %w", err)
		}

		if signedQty.Valid {
			val := signedQty.Float64
			outcome.SignedQuantity = &val
		}
		if exchangeTime.Valid {
			t := exchangeTime.Time
			outcome.ExchangeTime = &t
		}

		results = append(results, &outcome)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历最近交易失败: %w", err)
	}

	return results, nil
}

// GetSymbolStats 获取币种表现
func (r *repository) GetSymbolStats(ctx context.Context, traderID string, startTime, endTime *time.Time) ([]*SymbolPerformance, error) {
	conditions := []string{"trader_id = ?", "pnl IS NOT NULL"}
	args := []interface{}{traderID}

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
			symbol,
			COUNT(*) as total_trades,
			SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
			SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades,
			COALESCE(SUM(pnl), 0) as total_pn_l,
			COALESCE(AVG(pnl), 0) as avg_pn_l
		FROM trade_history
		%s
		GROUP BY symbol
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询币种表现失败: %w", err)
	}
	defer rows.Close()

	var results []*SymbolPerformance
	for rows.Next() {
		stat := &SymbolPerformance{}
		if err := rows.Scan(
			&stat.Symbol,
			&stat.TotalTrades,
			&stat.WinningTrades,
			&stat.LosingTrades,
			&stat.TotalPnL,
			&stat.AvgPnL,
		); err != nil {
			return nil, fmt.Errorf("扫描币种表现失败: %w", err)
		}

		results = append(results, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历币种表现失败: %w", err)
	}

	return results, nil
}

// GetPnLSeries 获取PnL序列
func (r *repository) GetPnLSeries(ctx context.Context, traderID string, startTime, endTime *time.Time, limit int) ([]*PnLPoint, error) {
	if limit <= 0 {
		limit = 30
	}

	conditions := []string{"trader_id = ?", "pnl IS NOT NULL"}
	args := []interface{}{traderID}

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
		SELECT pnl, quantity, signed_quantity, execution_price, timestamp
		FROM trade_history
		%s
		ORDER BY timestamp DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询PnL序列失败: %w", err)
	}
	defer rows.Close()

	var points []*PnLPoint
	for rows.Next() {
		point := &PnLPoint{}
		var signedQty sql.NullFloat64
		if err := rows.Scan(
			&point.PnL,
			&point.Quantity,
			&signedQty,
			&point.ExecutionPrice,
			&point.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("扫描PnL序列失败: %w", err)
		}

		if signedQty.Valid {
			val := signedQty.Float64
			point.SignedQuantity = &val
		}

		points = append(points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历PnL序列失败: %w", err)
	}

	return points, nil
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

// GetSyncState 获取同步游标
func (r *repository) GetSyncState(ctx context.Context, traderID string) (int64, error) {
	query := `SELECT last_exchange_timestamp_ms FROM trade_history_sync_state WHERE trader_id = ?`
	var timestamp sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, traderID).Scan(&timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("查询同步游标失败: %w", err)
	}
	if !timestamp.Valid {
		return 0, nil
	}
	return timestamp.Int64, nil
}

// UpsertSyncState 更新或插入同步游标
func (r *repository) UpsertSyncState(ctx context.Context, traderID string, timestampMs int64) error {
	query := `
		INSERT INTO trade_history_sync_state (trader_id, last_exchange_timestamp_ms, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(trader_id) DO UPDATE SET
			last_exchange_timestamp_ms = excluded.last_exchange_timestamp_ms,
			updated_at = CURRENT_TIMESTAMP
	`
	if _, err := r.db.ExecContext(ctx, query, traderID, timestampMs); err != nil {
		return fmt.Errorf("更新同步游标失败: %w", err)
	}
	return nil
}

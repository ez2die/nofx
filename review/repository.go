package review

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// reviewRepository 复盘记录数据访问层实现
type reviewRepository struct {
	db *sql.DB
}

// NewReviewRepository 创建复盘记录Repository实例
func NewReviewRepository(db *sql.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

// Save 保存复盘记录
func (r *reviewRepository) Save(ctx context.Context, record *ReviewRecord) error {
	query := `
		INSERT INTO review_records (
			trader_id, start_time, end_time, report_path,
			summary, metrics,
			total_trades, total_pnl, win_rate, error_count,
			status, error_message, created_at
		) VALUES (
			?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, CURRENT_TIMESTAMP
		)
	`

	result, err := r.db.ExecContext(ctx, query,
		record.TraderID, record.StartTime, record.EndTime, record.ReportPath,
		record.Summary, record.Metrics,
		record.TotalTrades, record.TotalPnL, record.WinRate, record.ErrorCount,
		record.Status, record.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("保存复盘记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取插入ID失败: %w", err)
	}

	record.ID = id
	return nil
}

// FindByID 根据ID查询复盘记录
func (r *reviewRepository) FindByID(ctx context.Context, id int64) (*ReviewRecord, error) {
	query := `
		SELECT 
			id, trader_id, start_time, end_time, report_path,
			summary, metrics,
			total_trades, total_pnl, win_rate, error_count,
			status, error_message, created_at
		FROM review_records
		WHERE id = ?
	`

	record := &ReviewRecord{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.TraderID, &record.StartTime, &record.EndTime, &record.ReportPath,
		&record.Summary, &record.Metrics,
		&record.TotalTrades, &record.TotalPnL, &record.WinRate, &record.ErrorCount,
		&record.Status, &record.ErrorMessage, &record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("复盘记录不存在 (id: %d)", id)
		}
		return nil, fmt.Errorf("查询复盘记录失败: %w", err)
	}

	return record, nil
}

// FindByTraderID 根据交易员ID查询复盘记录
func (r *reviewRepository) FindByTraderID(ctx context.Context, traderID string, limit int) ([]*ReviewRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT 
			id, trader_id, start_time, end_time, report_path,
			summary, metrics,
			total_trades, total_pnl, win_rate, error_count,
			status, error_message, created_at
		FROM review_records
		WHERE trader_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, traderID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询复盘记录失败: %w", err)
	}
	defer rows.Close()

	var records []*ReviewRecord
	for rows.Next() {
		record := &ReviewRecord{}
		err := rows.Scan(
			&record.ID, &record.TraderID, &record.StartTime, &record.EndTime, &record.ReportPath,
			&record.Summary, &record.Metrics,
			&record.TotalTrades, &record.TotalPnL, &record.WinRate, &record.ErrorCount,
			&record.Status, &record.ErrorMessage, &record.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描复盘记录失败: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历复盘记录失败: %w", err)
	}

	return records, nil
}

// FindByTimeRange 根据时间范围查询复盘记录
func (r *reviewRepository) FindByTimeRange(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*ReviewRecord, error) {
	query := `
		SELECT 
			id, trader_id, start_time, end_time, report_path,
			summary, metrics,
			total_trades, total_pnl, win_rate, error_count,
			status, error_message, created_at
		FROM review_records
		WHERE trader_id = ? 
			AND start_time >= ? 
			AND end_time <= ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, traderID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("查询复盘记录失败: %w", err)
	}
	defer rows.Close()

	var records []*ReviewRecord
	for rows.Next() {
		record := &ReviewRecord{}
		err := rows.Scan(
			&record.ID, &record.TraderID, &record.StartTime, &record.EndTime, &record.ReportPath,
			&record.Summary, &record.Metrics,
			&record.TotalTrades, &record.TotalPnL, &record.WinRate, &record.ErrorCount,
			&record.Status, &record.ErrorMessage, &record.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描复盘记录失败: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历复盘记录失败: %w", err)
	}

	return records, nil
}


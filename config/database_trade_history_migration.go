package config

import (
	"database/sql"
	"fmt"
)

func (d *Database) migrateTradeHistoryTable() error {
	rows, err := d.db.Query(`PRAGMA table_info(trade_history)`)
	if err != nil {
		return fmt.Errorf("检查 trade_history 表结构失败: %w", err)
	}
	defer rows.Close()

	hasLeverageColumn := false
	hasSignedQuantityColumn := false

	for rows.Next() {
		var (
			cid          int
			name         string
			ctype        string
			notnull      int
			defaultValue sql.NullString
			pk           int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("读取 trade_history 表结构失败: %w", err)
		}
		if name == "leverage" {
			hasLeverageColumn = true
		}
		if name == "signed_quantity" {
			hasSignedQuantityColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("检查 trade_history 表结构失败: %w", err)
	}

	// 需要迁移的情况：旧表包含 leverage 列，或缺少 signed_quantity 列
	if !hasLeverageColumn && hasSignedQuantityColumn {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("开始 trade_history 迁移失败: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE trade_history RENAME TO trade_history_legacy`); err != nil {
		tx.Rollback()
		return fmt.Errorf("重命名 trade_history 表失败: %w", err)
	}

	if _, err := tx.Exec(createTradeHistoryTableSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("创建新的 trade_history 表失败: %w", err)
	}

	insertStmt := `
	INSERT INTO trade_history (
		trader_id, symbol, action, side,
		quantity, signed_quantity,
		execution_price, pnl,
		fee, fee_token,
		raw_dir, start_position, builder_fee, exchange_side,
		exchange_order_id, exchange_trade_id, exchange_hash,
		timestamp, exchange_timestamp, exchange_timestamp_ms,
		created_at, updated_at
	)
	SELECT
		trader_id, symbol, action, side,
		quantity,
		CASE
			WHEN action IN ('open_short', 'close_short') THEN -ABS(quantity)
			ELSE ABS(quantity)
		END AS signed_quantity,
		execution_price, pnl,
		fee, fee_token,
		'' AS raw_dir, NULL AS start_position, NULL AS builder_fee, NULL AS exchange_side,
		exchange_order_id, exchange_trade_id, exchange_hash,
		timestamp, exchange_timestamp, exchange_timestamp_ms,
		created_at, updated_at
	FROM trade_history_legacy
	`

	if _, err := tx.Exec(insertStmt); err != nil {
		tx.Rollback()
		return fmt.Errorf("迁移 trade_history 数据失败: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE trade_history_legacy`); err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧的 trade_history 表失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 trade_history 迁移失败: %w", err)
	}

	return nil
}

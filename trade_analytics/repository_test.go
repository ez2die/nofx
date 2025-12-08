package trade_analytics

import (
	"context"
	"database/sql"
	"nofx/trade_history"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// 使用临时数据库文件
	dbPath := "/tmp/test_trade_analytics.db"
	os.Remove(dbPath) // 清理旧文件

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	// 创建表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trade_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trader_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			action TEXT NOT NULL,
			side TEXT NOT NULL,
			quantity REAL NOT NULL,
			signed_quantity REAL,
			execution_price REAL NOT NULL,
			pnl REAL,
			fee REAL DEFAULT 0,
			fee_token TEXT,
			raw_dir TEXT DEFAULT '',
			start_position REAL,
			builder_fee REAL,
			exchange_side TEXT,
			exchange_order_id TEXT,
			exchange_trade_id TEXT,
			exchange_hash TEXT,
			timestamp DATETIME NOT NULL,
			exchange_timestamp DATETIME,
			exchange_timestamp_ms INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	// 创建索引
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
		CREATE INDEX IF NOT EXISTS idx_trade_history_symbol ON trade_history(symbol);
		CREATE INDEX IF NOT EXISTS idx_trade_history_action ON trade_history(action);
		CREATE INDEX IF NOT EXISTS idx_trade_history_timestamp ON trade_history(timestamp);
	`)
	if err != nil {
		t.Fatalf("创建索引失败: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

// TestGetOverviewStats 测试概览统计
func TestGetOverviewStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建 repository
	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, ?)
	`, now.Add(-2*time.Hour), now.Add(-1*time.Hour), now)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试查询
	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetOverviewStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取概览统计失败: %v", err)
	}

	// 验证结果
	if stats.TotalTrades != 3 {
		t.Errorf("期望总交易数 3，实际 %d", stats.TotalTrades)
	}
	if stats.OpenTrades != 2 {
		t.Errorf("期望开仓数 2，实际 %d", stats.OpenTrades)
	}
	if stats.CloseTrades != 1 {
		t.Errorf("期望平仓数 1，实际 %d", stats.CloseTrades)
	}
	if stats.CompletedTrades != 1 {
		t.Errorf("期望已完成交易数 1，实际 %d", stats.CompletedTrades)
	}
	if stats.UnclosedTrades != 1 {
		t.Errorf("期望未平仓交易数 1，实际 %d", stats.UnclosedTrades)
	}
}

// TestGetPnLStats 测试盈亏统计
func TestGetPnLStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?)
	`, now.Add(-3*time.Hour), now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetPnLStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取盈亏统计失败: %v", err)
	}

	// 验证结果
	expectedTotalPnL := 25.0 // 10 + 20 - 5
	if stats.TotalPnL != expectedTotalPnL {
		t.Errorf("期望总盈亏 %.2f，实际 %.2f", expectedTotalPnL, stats.TotalPnL)
	}
	if stats.TotalProfit != 30.0 {
		t.Errorf("期望总盈利 30.0，实际 %.2f", stats.TotalProfit)
	}
	if stats.TotalLoss != 5.0 {
		t.Errorf("期望总亏损 5.0，实际 %.2f", stats.TotalLoss)
	}
	if stats.MaxWin != 20.0 {
		t.Errorf("期望最大盈利 20.0，实际 %.2f", stats.MaxWin)
	}
	if stats.MaxLoss != 5.0 {
		t.Errorf("期望最大亏损 5.0，实际 %.2f", stats.MaxLoss)
	}
}


package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestGetActionStats 测试交易类型统计
func TestGetActionStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建 repository
	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.5, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.02, 0.02, 52000, NULL, 2.0, ?)
	`, 
		baseTime,
		baseTime.Add(1*time.Hour),
		baseTime.Add(2*time.Hour),
		baseTime.Add(3*time.Hour),
		baseTime.Add(4*time.Hour),
	)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试查询
	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetActionStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取交易类型统计失败: %v", err)
	}

	// 验证开仓统计
	if stats.OpenStats == nil {
		t.Fatal("期望有开仓统计，实际为 nil")
	}
	// 应该有3个开仓记录
	if stats.OpenStats.TotalTrades != 3 {
		t.Errorf("期望开仓交易数 3，实际 %d", stats.OpenStats.TotalTrades)
	}
	// 开仓总费用：1.0 + 0.5 + 2.0 = 3.5
	expectedOpenFees := 3.5
	if stats.OpenStats.TotalFees != expectedOpenFees {
		t.Errorf("期望开仓总费用 %.2f，实际 %.2f", expectedOpenFees, stats.OpenStats.TotalFees)
	}
	// 开仓没有 PnL
	if stats.OpenStats.TotalPnL != 0 {
		t.Errorf("期望开仓总盈亏 0，实际 %.2f", stats.OpenStats.TotalPnL)
	}

	// 验证平仓统计
	if stats.CloseStats == nil {
		t.Fatal("期望有平仓统计，实际为 nil")
	}
	// 应该有2个平仓记录
	if stats.CloseStats.TotalTrades != 2 {
		t.Errorf("期望平仓交易数 2，实际 %d", stats.CloseStats.TotalTrades)
	}
	// 平仓总费用：1.5 + 0.5 = 2.0
	expectedCloseFees := 2.0
	if stats.CloseStats.TotalFees != expectedCloseFees {
		t.Errorf("期望平仓总费用 %.2f，实际 %.2f", expectedCloseFees, stats.CloseStats.TotalFees)
	}
	// 平仓总盈亏：10.0 + (-5.0) = 5.0
	expectedClosePnL := 5.0
	if stats.CloseStats.TotalPnL != expectedClosePnL {
		t.Errorf("期望平仓总盈亏 %.2f，实际 %.2f", expectedClosePnL, stats.CloseStats.TotalPnL)
	}
	// 平均盈亏：5.0 / 2 = 2.5
	expectedAvgPnL := 2.5
	if stats.CloseStats.AvgPnL != expectedAvgPnL {
		t.Errorf("期望平仓平均盈亏 %.2f，实际 %.2f", expectedAvgPnL, stats.CloseStats.AvgPnL)
	}
}


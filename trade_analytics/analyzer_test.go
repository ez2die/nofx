package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestCalculateRiskMetrics 测试风险指标计算
func TestCalculateRiskMetrics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)
	analyzer := NewAnalyzer(repo)

	ctx := context.Background()

	// 插入测试数据：模拟一个盈亏序列，包含回撤
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 53000, 15.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -10.0, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 3000, -5.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 54000, 25.0, 1.0, ?)
	`, baseTime, baseTime.Add(1*time.Hour), baseTime.Add(2*time.Hour),
		baseTime.Add(3*time.Hour), baseTime.Add(4*time.Hour), baseTime.Add(5*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	metrics, err := analyzer.CalculateRiskMetrics(ctx, filter)
	if err != nil {
		t.Fatalf("计算风险指标失败: %v", err)
	}

	// 验证基本指标存在
	if metrics.MaxDrawdown < 0 {
		t.Error("期望最大回撤 >= 0，实际为负数")
	}
	if metrics.Volatility < 0 {
		t.Error("期望波动率 >= 0，实际为负数")
	}
	// 验证回撤历史
	if metrics.DrawdownHistory == nil {
		t.Error("期望有回撤历史，实际为 nil")
	}
}

// TestCalculateStreakStats 测试连续统计
func TestCalculateStreakStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)
	analyzer := NewAnalyzer(repo)

	ctx := context.Background()

	// 插入测试数据：模拟连胜和连亏序列
	// 序列：赢、赢、输、输、输、赢、赢、赢、输
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 3000, -10.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 50000, -15.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 53000, 15.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 54000, 25.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2800, 30.0, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 3000, -8.0, 0.5, ?)
	`, baseTime, baseTime.Add(1*time.Hour), baseTime.Add(2*time.Hour),
		baseTime.Add(3*time.Hour), baseTime.Add(4*time.Hour), baseTime.Add(5*time.Hour),
		baseTime.Add(6*time.Hour), baseTime.Add(7*time.Hour), baseTime.Add(8*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := analyzer.CalculateStreakStats(ctx, filter)
	if err != nil {
		t.Fatalf("计算连续统计失败: %v", err)
	}

	// 验证最长连胜：应该是3（最后三个赢）
	if stats.LongestWinningStreak < 3 {
		t.Errorf("期望最长连胜 >= 3，实际 %d", stats.LongestWinningStreak)
	}
	// 验证最长连亏：应该是3（中间三个输）
	if stats.LongestLosingStreak < 3 {
		t.Errorf("期望最长连亏 >= 3，实际 %d", stats.LongestLosingStreak)
	}
	// 验证当前连续类型
	if stats.CurrentStreakType != "losing" {
		t.Errorf("期望当前连续类型 losing，实际 %s", stats.CurrentStreakType)
	}
	// 验证当前连续数应该是负数（连亏）
	if stats.CurrentStreak >= 0 {
		t.Errorf("期望当前连续数 < 0（连亏），实际 %d", stats.CurrentStreak)
	}
}


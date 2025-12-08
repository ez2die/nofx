package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestMatchTradePairs 测试交易配对
func TestMatchTradePairs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)
	pairMatcher := NewPairMatcher(repo)

	ctx := context.Background()

	// 插入测试数据：模拟FIFO配对场景
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.02, 0.02, 51000, NULL, 2.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.5, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.02, 0.02, 53000, 40.0, 3.0, ?)
	`, baseTime, baseTime.Add(1*time.Hour), baseTime.Add(2*time.Hour), baseTime.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	pairs, err := pairMatcher.MatchTradePairs(ctx, filter)
	if err != nil {
		t.Fatalf("匹配交易对失败: %v", err)
	}

	// 应该匹配到2个交易对
	if len(pairs) != 2 {
		t.Errorf("期望匹配到2个交易对，实际 %d", len(pairs))
	}

	// 验证第一个配对
	if len(pairs) > 0 {
		firstPair := pairs[0]
		if firstPair.OpenRecord == nil || firstPair.CloseRecord == nil {
			t.Error("期望配对有开仓和平仓记录")
		}
		if firstPair.MatchedQty != 0.01 {
			t.Errorf("期望第一个配对数量 0.01，实际 %.2f", firstPair.MatchedQty)
		}
		if firstPair.HoldingTime <= 0 {
			t.Error("期望持仓时间 > 0")
		}
	}
}

// TestGetPairStatistics 测试配对统计
func TestGetPairStatistics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)
	pairMatcher := NewPairMatcher(repo)

	ctx := context.Background()

	// 插入测试数据
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.02, 0.02, 52000, NULL, 2.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.02, 0.02, 53000, 20.0, 2.0, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.03, 0.03, 54000, NULL, 3.0, ?)
	`, baseTime, baseTime.Add(1*time.Hour), baseTime.Add(2*time.Hour),
		baseTime.Add(3*time.Hour), baseTime.Add(4*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := pairMatcher.GetPairStatistics(ctx, filter)
	if err != nil {
		t.Fatalf("获取配对统计失败: %v", err)
	}

	// 验证配对统计
	if stats.TotalPairs != 2 {
		t.Errorf("期望配对数量 2，实际 %d", stats.TotalPairs)
	}
	// 配对成功率 = 2 / 3 * 100% = 66.67%
	expectedSuccessRate := 66.67
	if stats.PairSuccessRate < expectedSuccessRate-1 || stats.PairSuccessRate > expectedSuccessRate+1 {
		t.Errorf("期望配对成功率约 %.2f%%，实际 %.2f%%", expectedSuccessRate, stats.PairSuccessRate)
	}
	// 未配对交易数 = 3 - 2 = 1
	if stats.UnpairedTrades != 1 {
		t.Errorf("期望未配对交易数 1，实际 %d", stats.UnpairedTrades)
	}
	// 验证持仓时间统计
	if stats.AvgHoldingTime <= 0 {
		t.Error("期望平均持仓时间 > 0")
	}
	if stats.MinHoldingTime <= 0 {
		t.Error("期望最短持仓时间 > 0")
	}
	if stats.MaxHoldingTime <= 0 {
		t.Error("期望最长持仓时间 > 0")
	}
}


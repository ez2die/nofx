package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestGetTrendAnalysis 测试趋势分析
func TestGetTrendAnalysis(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建 repository
	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据（不同时间、不同PnL值）
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.02, 0.02, 52000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.02, 0.02, 53000, 20.0, 1.0, ?)
	`, 
		baseTime,
		baseTime.Add(1*time.Hour),
		baseTime.Add(2*time.Hour),
		baseTime.Add(3*time.Hour),
		baseTime.Add(4*time.Hour),
		baseTime.Add(5*time.Hour),
	)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试查询
	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	analysis, err := repo.GetTrendAnalysis(ctx, filter)
	if err != nil {
		t.Fatalf("获取趋势分析失败: %v", err)
	}

	// 验证累计盈亏曲线
	if analysis.CumulativePnLSeries == nil {
		t.Error("期望有累计盈亏曲线，实际为 nil")
	} else {
		// 应该有3个点（3个有PnL的记录）
		if len(analysis.CumulativePnLSeries) != 3 {
			t.Errorf("期望累计盈亏曲线有3个点，实际 %d", len(analysis.CumulativePnLSeries))
		}
		// 最后一个点的累计盈亏应该是 10.0 + (-5.0) + 20.0 = 25.0
		lastPoint := analysis.CumulativePnLSeries[len(analysis.CumulativePnLSeries)-1]
		expectedCumulative := 25.0
		if lastPoint.CumulativePnL != expectedCumulative {
			t.Errorf("期望最后累计盈亏 %.2f，实际 %.2f", expectedCumulative, lastPoint.CumulativePnL)
		}
	}

	// 验证盈亏分布
	if analysis.PnLDistribution == nil {
		t.Error("期望有盈亏分布，实际为 nil")
	} else {
		// 应该至少有一个区间
		if len(analysis.PnLDistribution) == 0 {
			t.Error("期望盈亏分布至少有一个区间，实际为空")
		}
		// 验证分布区间格式
		for _, bin := range analysis.PnLDistribution {
			if bin.Range == "" {
				t.Error("期望分布区间有范围标识，实际为空")
			}
			if bin.Count <= 0 {
				t.Errorf("期望分布区间计数 > 0，实际 %d", bin.Count)
			}
		}
	}

	// 验证交易频率趋势
	if analysis.TradeFrequencyTrend == nil {
		t.Error("期望有交易频率趋势，实际为 nil")
	} else {
		// 应该有6个交易记录，分布在不同的时间点
		// 由于按小时分组，应该至少有1个点
		if len(analysis.TradeFrequencyTrend) == 0 {
			t.Error("期望交易频率趋势至少有一个点，实际为空")
		}
		// 验证时间点格式
		for _, point := range analysis.TradeFrequencyTrend {
			if point.Timestamp.IsZero() {
				t.Error("期望交易频率点有时间戳，实际为零值")
			}
			if point.TradeCount <= 0 {
				t.Errorf("期望交易频率点计数 > 0，实际 %d", point.TradeCount)
			}
		}
	}
}


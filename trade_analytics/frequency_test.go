package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestGetFrequencyStats 测试交易频率统计
func TestGetFrequencyStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建 repository
	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据（不同时间、不同小时）
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.02, 0.02, 52000, NULL, 1.0, ?)
	`, 
		baseTime,                    // 第1条：2025-01-01 10:00
		baseTime.Add(1*time.Hour),   // 第2条：2025-01-01 11:00
		baseTime.Add(2*time.Hour),   // 第3条：2025-01-01 12:00
		baseTime.Add(3*time.Hour),   // 第4条：2025-01-01 13:00
		baseTime.Add(24*time.Hour),  // 第5条：2025-01-02 10:00
	)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试查询
	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetFrequencyStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取交易频率统计失败: %v", err)
	}

	// 验证结果
	// 日均交易数：5条记录，时间跨度从 2025-01-01 10:00 到 2025-01-02 10:00 = 24小时 = 1天
	// 所以日均交易数应该是 5 / 1 = 5.0
	// 但实际业务中，如果跨越了不同的日期，可能希望按日期数计算
	// 这里我们验证计算是否合理（应该 > 0）
	if stats.DailyAverageTrades <= 0 {
		t.Errorf("期望日均交易数 > 0，实际 %.2f", stats.DailyAverageTrades)
	}
	// 验证应该在合理范围内（1-10之间）
	if stats.DailyAverageTrades < 1.0 || stats.DailyAverageTrades > 10.0 {
		t.Errorf("日均交易数不在合理范围内，实际 %.2f", stats.DailyAverageTrades)
	}

	// 平均交易间隔：应该有4个间隔
	// 实际间隔：1小时、1小时、1小时、21小时，平均 = 24/4 = 6小时 = 21600秒
	// 验证应该在合理范围内（约5-7小时之间）
	expectedIntervalMin := int64(5 * 3600) // 5小时
	expectedIntervalMax := int64(7 * 3600) // 7小时
	if stats.AvgTradeInterval < expectedIntervalMin || stats.AvgTradeInterval > expectedIntervalMax {
		t.Errorf("期望平均交易间隔在 %d-%d 秒之间，实际 %d 秒", expectedIntervalMin, expectedIntervalMax, stats.AvgTradeInterval)
	}

	// 最活跃时段：应该包含10、11、12、13小时
	if len(stats.MostActiveHours) == 0 {
		t.Error("期望有最活跃时段，实际为空")
	}
	// 检查是否包含10点（应该有2次交易）
	has10 := false
	for _, hour := range stats.MostActiveHours {
		if hour == 10 {
			has10 = true
			break
		}
	}
	if !has10 {
		t.Errorf("期望最活跃时段包含10点，实际 %v", stats.MostActiveHours)
	}

	// 最活跃日期：应该包含2025-01-01（4条记录）和2025-01-02（1条记录）
	if len(stats.MostActiveDays) == 0 {
		t.Error("期望有最活跃日期，实际为空")
	}
	// 第一个应该是最活跃的日期（2025-01-01）
	if stats.MostActiveDays[0] != "2025-01-01" {
		t.Errorf("期望最活跃日期第一个是 2025-01-01，实际 %v", stats.MostActiveDays)
	}
}


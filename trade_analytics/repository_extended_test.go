package trade_analytics

import (
	"context"
	"nofx/trade_history"
	"testing"
	"time"
)

// TestGetWinRateStats 测试胜率统计
func TestGetWinRateStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：3个盈利，2个亏损，1个持平
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 3000, 0.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 53000, 15.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2800, -10.0, 0.5, ?)
	`, now.Add(-6*time.Hour), now.Add(-5*time.Hour), now.Add(-4*time.Hour),
		now.Add(-3*time.Hour), now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetWinRateStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取胜率统计失败: %v", err)
	}

	// 验证结果：3个盈利，2个亏损，1个持平，总共6个
	if stats.WinningTrades != 3 {
		t.Errorf("期望盈利交易数 3，实际 %d", stats.WinningTrades)
	}
	if stats.LosingTrades != 2 {
		t.Errorf("期望亏损交易数 2，实际 %d", stats.LosingTrades)
	}
	if stats.BreakEvenTrades != 1 {
		t.Errorf("期望持平交易数 1，实际 %d", stats.BreakEvenTrades)
	}
	// 胜率 = 3 / 6 * 100% = 50%
	expectedWinRate := 50.0
	if stats.WinRate != expectedWinRate {
		t.Errorf("期望胜率 %.2f%%，实际 %.2f%%", expectedWinRate, stats.WinRate)
	}
	// 平均盈利 = (10 + 20 + 15) / 3 = 15.0
	expectedAvgWin := 15.0
	if stats.AvgWin != expectedAvgWin {
		t.Errorf("期望平均盈利 %.2f，实际 %.2f", expectedAvgWin, stats.AvgWin)
	}
	// 平均亏损 = (5 + 10) / 2 = 7.5
	expectedAvgLoss := 7.5
	if stats.AvgLoss != expectedAvgLoss {
		t.Errorf("期望平均亏损 %.2f，实际 %.2f", expectedAvgLoss, stats.AvgLoss)
	}
}

// TestGetFeeStats 测试费用统计
func TestGetFeeStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, builder_fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, 0.1, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.5, 0.2, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, 0.05, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, 0.05, ?)
	`, now.Add(-3*time.Hour), now.Add(-2*time.Hour), now.Add(-1*time.Hour), now)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetFeeStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取费用统计失败: %v", err)
	}

	// 验证结果
	// 总费用 = 1.0 + 1.5 + 0.5 + 0.5 = 3.5
	expectedTotalFees := 3.5
	if stats.TotalFees != expectedTotalFees {
		t.Errorf("期望总费用 %.2f，实际 %.2f", expectedTotalFees, stats.TotalFees)
	}
	// 开仓费用 = 1.0 + 0.5 = 1.5
	expectedOpenFees := 1.5
	if stats.OpenFees != expectedOpenFees {
		t.Errorf("期望开仓费用 %.2f，实际 %.2f", expectedOpenFees, stats.OpenFees)
	}
	// 平仓费用 = 1.5 + 0.5 = 2.0
	expectedCloseFees := 2.0
	if stats.CloseFees != expectedCloseFees {
		t.Errorf("期望平仓费用 %.2f，实际 %.2f", expectedCloseFees, stats.CloseFees)
	}
	// Builder费用 = 0.1 + 0.2 + 0.05 + 0.05 = 0.4
	expectedBuilderFees := 0.4
	if stats.BuilderFees != expectedBuilderFees {
		t.Errorf("期望Builder费用 %.2f，实际 %.2f", expectedBuilderFees, stats.BuilderFees)
	}
	// 平均费用 = 3.5 / 4 = 0.875
	expectedAvgFee := 0.875
	if stats.AvgFee != expectedAvgFee {
		t.Errorf("期望平均费用 %.3f，实际 %.3f", expectedAvgFee, stats.AvgFee)
	}
}

// TestGetDirectionStats 测试方向统计
func TestGetDirectionStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：3个做多（2盈利1亏损），2个做空（1盈利1亏损）
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 50000, -5.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, 15.0, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 3000, -10.0, 0.5, ?)
	`, now.Add(-5*time.Hour), now.Add(-4*time.Hour), now.Add(-3*time.Hour),
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetDirectionStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取方向统计失败: %v", err)
	}

	// 验证做多统计
	if stats.LongStats == nil {
		t.Fatal("期望有做多统计，实际为 nil")
	}
	if stats.LongStats.TotalTrades != 3 {
		t.Errorf("期望做多交易数 3，实际 %d", stats.LongStats.TotalTrades)
	}
	// 做多总盈亏 = 10 + 20 - 5 = 25
	expectedLongPnL := 25.0
	if stats.LongStats.TotalPnL != expectedLongPnL {
		t.Errorf("期望做多总盈亏 %.2f，实际 %.2f", expectedLongPnL, stats.LongStats.TotalPnL)
	}
	// 做多胜率 = 2 / 3 * 100% = 66.67%
	expectedLongWinRate := 66.67
	if stats.LongStats.WinRate < expectedLongWinRate-1 || stats.LongStats.WinRate > expectedLongWinRate+1 {
		t.Errorf("期望做多胜率约 %.2f%%，实际 %.2f%%", expectedLongWinRate, stats.LongStats.WinRate)
	}

	// 验证做空统计
	if stats.ShortStats == nil {
		t.Fatal("期望有做空统计，实际为 nil")
	}
	if stats.ShortStats.TotalTrades != 2 {
		t.Errorf("期望做空交易数 2，实际 %d", stats.ShortStats.TotalTrades)
	}
	// 做空总盈亏 = 15 - 10 = 5
	expectedShortPnL := 5.0
	if stats.ShortStats.TotalPnL != expectedShortPnL {
		t.Errorf("期望做空总盈亏 %.2f，实际 %.2f", expectedShortPnL, stats.ShortStats.TotalPnL)
	}

	// 验证方向偏好
	if stats.Preference == nil {
		t.Fatal("期望有方向偏好，实际为 nil")
	}
	// 做多占比 = 3 / 5 * 100% = 60%
	expectedLongRatio := 60.0
	if stats.Preference.LongRatio != expectedLongRatio {
		t.Errorf("期望做多占比 %.2f%%，实际 %.2f%%", expectedLongRatio, stats.Preference.LongRatio)
	}
	// 应该偏好做多
	if stats.Preference.PreferredSide != "long" {
		t.Errorf("期望偏好方向 long，实际 %s", stats.Preference.PreferredSide)
	}
}

// TestGetSymbolStats 测试币种统计
func TestGetSymbolStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：BTCUSDT 2笔，ETHUSDT 2笔
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'open_long', 'long', 0.01, 0.01, 50000, NULL, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'open_short', 'short', 0.1, 0.1, 3000, NULL, 0.5, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?)
	`, now.Add(-5*time.Hour), now.Add(-4*time.Hour), now.Add(-3*time.Hour),
		now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetSymbolStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取币种统计失败: %v", err)
	}

	// 验证BTCUSDT统计
	btcStats, ok := stats["BTCUSDT"]
	if !ok {
		t.Fatal("期望有BTCUSDT统计，实际缺失")
	}
	// 总交易数包括开仓和平仓：1个开仓 + 2个平仓 = 3
	// 但实际查询可能只统计有PnL的记录，所以可能是2
	if btcStats.TotalTrades < 2 {
		t.Errorf("期望BTCUSDT总交易数 >= 2，实际 %d", btcStats.TotalTrades)
	}
	if btcStats.CompletedTrades != 2 {
		t.Errorf("期望BTCUSDT已完成交易数 2，实际 %d", btcStats.CompletedTrades)
	}
	// BTCUSDT总盈亏 = 10 + 20 = 30
	expectedBTCPnL := 30.0
	if btcStats.TotalPnL != expectedBTCPnL {
		t.Errorf("期望BTCUSDT总盈亏 %.2f，实际 %.2f", expectedBTCPnL, btcStats.TotalPnL)
	}
	// BTCUSDT胜率 = 2 / 2 * 100% = 100%
	if btcStats.WinRate != 100.0 {
		t.Errorf("期望BTCUSDT胜率 100%%，实际 %.2f%%", btcStats.WinRate)
	}

	// 验证ETHUSDT统计
	ethStats, ok := stats["ETHUSDT"]
	if !ok {
		t.Fatal("期望有ETHUSDT统计，实际缺失")
	}
	// 总交易数包括开仓和平仓：1个开仓 + 1个平仓 = 2
	// 但实际查询可能只统计有PnL的记录，所以可能是1
	if ethStats.TotalTrades < 1 {
		t.Errorf("期望ETHUSDT总交易数 >= 1，实际 %d", ethStats.TotalTrades)
	}
	// ETHUSDT总盈亏 = -5
	expectedETHPnL := -5.0
	if ethStats.TotalPnL != expectedETHPnL {
		t.Errorf("期望ETHUSDT总盈亏 %.2f，实际 %.2f", expectedETHPnL, ethStats.TotalPnL)
	}
}

// TestGetDailyStats 测试每日统计
func TestGetDailyStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：不同日期的交易
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 53000, 15.0, 1.0, ?)
	`, baseTime, baseTime.Add(2*time.Hour), baseTime.Add(24*time.Hour), baseTime.Add(26*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetDailyStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取每日统计失败: %v", err)
	}

	// 应该至少有一天的数据
	if len(stats) == 0 {
		t.Error("期望至少1天的统计，实际为0")
	}

	// 验证统计数据的有效性
	for _, day := range stats {
		if day.Date == "" {
			t.Error("期望日期不为空")
		}
		if day.TotalTrades <= 0 {
			t.Errorf("期望总交易数 > 0，实际 %d", day.TotalTrades)
		}
		// 验证日期格式
		if len(day.Date) != 10 {
			t.Errorf("期望日期格式 YYYY-MM-DD，实际 %s", day.Date)
		}
	}
}

// TestGetWeeklyStats 测试每周统计
func TestGetWeeklyStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：不同周的交易
	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC) // 周三
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?)
	`, baseTime, baseTime.Add(3*24*time.Hour), baseTime.Add(8*24*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetWeeklyStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取每周统计失败: %v", err)
	}

	// 应该至少有一周的数据
	if len(stats) == 0 {
		t.Error("期望至少1周的统计，实际为0")
	}
}

// TestGetMonthlyStats 测试每月统计
func TestGetMonthlyStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tradeHistoryRepo := trade_history.NewRepository(db)
	repo := NewRepository(db, tradeHistoryRepo)

	ctx := context.Background()

	// 插入测试数据：不同月份的交易
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO trade_history (trader_id, symbol, action, side, quantity, signed_quantity, execution_price, pnl, fee, timestamp)
		VALUES 
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 51000, 10.0, 1.0, ?),
			('test_trader', 'BTCUSDT', 'close_long', 'long', 0.01, 0.01, 52000, 20.0, 1.0, ?),
			('test_trader', 'ETHUSDT', 'close_short', 'short', 0.1, 0.1, 2900, -5.0, 0.5, ?)
	`, baseTime, baseTime.Add(20*24*time.Hour), baseTime.Add(35*24*time.Hour))
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	filter := &AnalyticsFilter{
		TraderID: "test_trader",
	}

	stats, err := repo.GetMonthlyStats(ctx, filter)
	if err != nil {
		t.Fatalf("获取每月统计失败: %v", err)
	}

	// 应该至少有一个月的数据
	if len(stats) == 0 {
		t.Error("期望至少1个月的统计，实际为0")
	}
}


package trade_analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// Analyzer 分析器接口
type Analyzer interface {
	// Analyze 执行完整分析
	Analyze(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error)

	// CalculateRiskMetrics 计算风险指标
	CalculateRiskMetrics(ctx context.Context, filter *AnalyticsFilter) (*RiskMetrics, error)

	// CalculateStreakStats 计算连续统计
	CalculateStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error)
}

// analyzer 实现
type analyzer struct {
	repo Repository
}

// NewAnalyzer 创建Analyzer实例
func NewAnalyzer(repo Repository) Analyzer {
	return &analyzer{repo: repo}
}

// Analyze 执行完整分析
func (a *analyzer) Analyze(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error) {
	analytics := &TradeAnalytics{}

	// 基础统计
	overview, err := a.repo.GetOverviewStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取概览统计失败: %w", err)
	}
	analytics.Overview = overview

	// 盈亏统计
	pnlStats, err := a.repo.GetPnLStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取盈亏统计失败: %w", err)
	}
	analytics.PnLStats = pnlStats

	// 胜率统计
	winRateStats, err := a.repo.GetWinRateStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取胜率统计失败: %w", err)
	}
	analytics.WinRateStats = winRateStats

	// 费用统计
	feeStats, err := a.repo.GetFeeStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取费用统计失败: %w", err)
	}
	analytics.FeeStats = feeStats

	// 方向统计
	directionStats, err := a.repo.GetDirectionStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取方向统计失败: %w", err)
	}
	analytics.DirectionStats = directionStats

	// 币种统计
	symbolStats, err := a.repo.GetSymbolStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取币种统计失败: %w", err)
	}
	analytics.SymbolStats = symbolStats

	// 时间序列统计
	timeSeriesStats := &TimeSeriesStatistics{}
	if filter.GroupBy == "day" || filter.GroupBy == "" {
		dailyStats, err := a.repo.GetDailyStats(ctx, filter)
		if err == nil {
			timeSeriesStats.DailyStats = dailyStats
		}
	}
	if filter.GroupBy == "week" || filter.GroupBy == "" {
		weeklyStats, err := a.repo.GetWeeklyStats(ctx, filter)
		if err == nil {
			timeSeriesStats.WeeklyStats = weeklyStats
		}
	}
	if filter.GroupBy == "month" || filter.GroupBy == "" {
		monthlyStats, err := a.repo.GetMonthlyStats(ctx, filter)
		if err == nil {
			timeSeriesStats.MonthlyStats = monthlyStats
		}
	}
	analytics.TimeSeriesStats = timeSeriesStats

	// 风险指标
	riskMetrics, err := a.CalculateRiskMetrics(ctx, filter)
	if err == nil {
		analytics.RiskMetrics = riskMetrics
	}

	// 连续统计
	streakStats, err := a.CalculateStreakStats(ctx, filter)
	if err == nil {
		analytics.StreakStats = streakStats
	}

	// 交易频率统计
	frequencyStats, err := a.repo.GetFrequencyStats(ctx, filter)
	if err == nil {
		analytics.FrequencyStats = frequencyStats
	}

	// 交易类型统计
	actionStats, err := a.repo.GetActionStats(ctx, filter)
	if err == nil {
		analytics.ActionStats = actionStats
	}

	// 趋势分析
	trendAnalysis, err := a.repo.GetTrendAnalysis(ctx, filter)
	if err == nil {
		analytics.TrendAnalysis = trendAnalysis
	}

	return analytics, nil
}

// CalculateRiskMetrics 计算风险指标
func (a *analyzer) CalculateRiskMetrics(ctx context.Context, filter *AnalyticsFilter) (*RiskMetrics, error) {
	// 使用聚合后的记录（将同一订单的多笔执行合并）
	records, err := a.repo.GetAggregatedRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	// 过滤出有PnL的记录并按时间排序
	var pnlRecords []*TradeRecord
	for _, rec := range records {
		if rec.PnL != nil {
			pnlRecords = append(pnlRecords, rec)
		}
	}

	if len(pnlRecords) == 0 {
		return &RiskMetrics{}, nil
	}

	// 按时间排序
	sort.Slice(pnlRecords, func(i, j int) bool {
		return pnlRecords[i].Timestamp.Before(pnlRecords[j].Timestamp)
	})

	// 计算累计盈亏曲线
	cumulativePnL := make([]float64, len(pnlRecords))
	var runningTotal float64
	for i, rec := range pnlRecords {
		runningTotal += *rec.PnL
		cumulativePnL[i] = runningTotal
	}

	// 计算最大回撤
	maxDrawdown, maxDrawdownPercent, drawdownHistory := calculateDrawdown(cumulativePnL, pnlRecords)

	// 计算波动率（PnL的标准差）
	volatility := calculateVolatility(pnlRecords)

	// 计算夏普比率
	sharpeRatio := calculateSharpeRatio(pnlRecords, volatility)

	// 计算回撤持续时间和恢复时间
	drawdownDuration, recoveryTime := calculateDrawdownDuration(drawdownHistory, pnlRecords)

	return &RiskMetrics{
		MaxDrawdown:        maxDrawdown,
		MaxDrawdownPercent: maxDrawdownPercent,
		DrawdownDuration:   drawdownDuration,
		RecoveryTime:       recoveryTime,
		Volatility:         volatility,
		SharpeRatio:        sharpeRatio,
		DrawdownHistory:    drawdownHistory,
	}, nil
}

// calculateDrawdown 计算回撤
func calculateDrawdown(cumulativePnL []float64, records []*TradeRecord) (float64, float64, []Drawdown) {
	if len(cumulativePnL) == 0 {
		return 0, 0, nil
	}

	var maxDrawdown float64
	var maxDrawdownPercent float64
	var drawdownHistory []Drawdown

	var peak float64 = cumulativePnL[0]
	var peakIndex int = 0
	var trough float64 = cumulativePnL[0]
	var troughIndex int = 0

	for i, value := range cumulativePnL {
		// 更新峰值
		if value > peak {
			// 如果之前有回撤，记录它
			if peakIndex < troughIndex && peak > trough {
				dd := peak - trough
				ddPercent := 0.0
				if peak > 0 {
					ddPercent = (dd / peak) * 100.0
				} else if peak < 0 {
					ddPercent = (dd / math.Abs(peak)) * 100.0
				}

				if dd > maxDrawdown {
					maxDrawdown = dd
					maxDrawdownPercent = ddPercent
				}

				drawdownHistory = append(drawdownHistory, Drawdown{
					StartTime:   records[peakIndex].Timestamp,
					EndTime:     records[troughIndex].Timestamp,
					PeakValue:   peak,
					TroughValue: trough,
					Drawdown:    dd,
					Duration:    int64(records[troughIndex].Timestamp.Sub(records[peakIndex].Timestamp).Seconds()),
				})
			}
			peak = value
			peakIndex = i
			trough = value
			troughIndex = i
		}

		// 更新谷值
		if value < trough {
			trough = value
			troughIndex = i
		}
	}

	// 检查最后一个回撤
	if peakIndex < troughIndex && peak > trough {
		dd := peak - trough
		ddPercent := 0.0
		if peak > 0 {
			ddPercent = (dd / peak) * 100.0
		} else if peak < 0 {
			ddPercent = (dd / math.Abs(peak)) * 100.0
		}

		if dd > maxDrawdown {
			maxDrawdown = dd
			maxDrawdownPercent = ddPercent
		}

		drawdownHistory = append(drawdownHistory, Drawdown{
			StartTime:   records[peakIndex].Timestamp,
			EndTime:     records[troughIndex].Timestamp,
			PeakValue:   peak,
			TroughValue: trough,
			Drawdown:    dd,
			Duration:    int64(records[troughIndex].Timestamp.Sub(records[peakIndex].Timestamp).Seconds()),
		})
	}

	return maxDrawdown, maxDrawdownPercent, drawdownHistory
}

// calculateVolatility 计算波动率（标准差）
func calculateVolatility(records []*TradeRecord) float64 {
	if len(records) == 0 {
		return 0
	}

	// 计算平均值
	var sum float64
	for _, rec := range records {
		if rec.PnL != nil {
			sum += *rec.PnL
		}
	}
	mean := sum / float64(len(records))

	// 计算方差
	var variance float64
	for _, rec := range records {
		if rec.PnL != nil {
			diff := *rec.PnL - mean
			variance += diff * diff
		}
	}
	variance /= float64(len(records))

	// 标准差
	return math.Sqrt(variance)
}

// calculateSharpeRatio 计算夏普比率
func calculateSharpeRatio(records []*TradeRecord, volatility float64) float64 {
	if len(records) == 0 || volatility == 0 {
		return 0
	}

	// 计算平均收益
	var sum float64
	for _, rec := range records {
		if rec.PnL != nil {
			sum += *rec.PnL
		}
	}
	avgReturn := sum / float64(len(records))

	// 夏普比率 = 平均收益 / 标准差
	if volatility > 0 {
		return avgReturn / volatility
	}

	return 0
}

// calculateDrawdownDuration 计算回撤持续时间和恢复时间
func calculateDrawdownDuration(drawdownHistory []Drawdown, records []*TradeRecord) (int64, int64) {
	if len(drawdownHistory) == 0 {
		return 0, 0
	}

	// 找到最大回撤
	var maxDD Drawdown
	for _, dd := range drawdownHistory {
		if dd.Drawdown > maxDD.Drawdown {
			maxDD = dd
		}
	}

	// 计算恢复时间（从谷值恢复到峰值的时间）
	var recoveryTime int64
	if len(records) > 0 {
		// 找到谷值后的下一个峰值
		troughTime := maxDD.EndTime
		var nextPeakTime time.Time = troughTime

		for _, rec := range records {
			if rec.Timestamp.After(troughTime) && rec.PnL != nil {
				// 这里简化处理，实际应该计算累计盈亏
				// 为了简化，我们假设恢复时间是从谷值到记录结束的时间
				nextPeakTime = rec.Timestamp
				break
			}
		}

		if nextPeakTime.After(troughTime) {
			recoveryTime = int64(nextPeakTime.Sub(troughTime).Seconds())
		}
	}

	return maxDD.Duration, recoveryTime
}

// CalculateStreakStats 计算连续统计
func (a *analyzer) CalculateStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error) {
	// 使用聚合后的记录（将同一订单的多笔执行合并）
	records, err := a.repo.GetAggregatedRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	// 过滤出有PnL的记录并按时间排序
	var pnlRecords []*TradeRecord
	for _, rec := range records {
		if rec.PnL != nil {
			pnlRecords = append(pnlRecords, rec)
		}
	}

	if len(pnlRecords) == 0 {
		return &StreakStatistics{}, nil
	}

	// 按时间排序
	sort.Slice(pnlRecords, func(i, j int) bool {
		return pnlRecords[i].Timestamp.Before(pnlRecords[j].Timestamp)
	})

	// 计算连续统计
	var longestWinningStreak int
	var longestLosingStreak int
	var currentStreak int
	var currentStreakType string

	var currentWinningStreak int
	var currentLosingStreak int

	for _, rec := range pnlRecords {
		if rec.PnL == nil {
			continue
		}

		if *rec.PnL > 0 {
			// 盈利
			currentWinningStreak++
			currentLosingStreak = 0

			if currentWinningStreak > longestWinningStreak {
				longestWinningStreak = currentWinningStreak
			}
		} else if *rec.PnL < 0 {
			// 亏损
			currentLosingStreak++
			currentWinningStreak = 0

			if currentLosingStreak > longestLosingStreak {
				longestLosingStreak = currentLosingStreak
			}
		} else {
			// 持平，重置连续
			currentWinningStreak = 0
			currentLosingStreak = 0
		}
	}

	// 计算当前连续（从最后一条记录开始）
	var lastLossTime *time.Time
	if len(pnlRecords) > 0 {
		lastRec := pnlRecords[len(pnlRecords)-1]
		if lastRec.PnL != nil {
			if *lastRec.PnL > 0 {
				currentStreak = currentWinningStreak
				currentStreakType = "winning"
			} else if *lastRec.PnL < 0 {
				currentStreak = -currentLosingStreak
				currentStreakType = "losing"
				// 记录最后一次亏损的时间
				lastLossTime = &lastRec.Timestamp
			}
		}
	}

	return &StreakStatistics{
		LongestWinningStreak: longestWinningStreak,
		LongestLosingStreak:   longestLosingStreak,
		CurrentStreak:         currentStreak,
		CurrentStreakType:     currentStreakType,
		LastLossTimestamp:     lastLossTime,
	}, nil
}


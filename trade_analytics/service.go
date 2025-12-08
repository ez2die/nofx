package trade_analytics

import (
	"context"
	"fmt"
)

// Service 交易分析服务接口
type Service interface {
	// GetAnalytics 获取完整分析
	GetAnalytics(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error)

	// GetOverview 获取概览统计
	GetOverview(ctx context.Context, filter *AnalyticsFilter) (*TradeOverview, error)

	// GetPnLStats 获取盈亏统计
	GetPnLStats(ctx context.Context, filter *AnalyticsFilter) (*PnLStatistics, error)

	// GetWinRateStats 获取胜率统计
	GetWinRateStats(ctx context.Context, filter *AnalyticsFilter) (*WinRateStatistics, error)

	// GetFeeStats 获取费用统计
	GetFeeStats(ctx context.Context, filter *AnalyticsFilter) (*FeeStatistics, error)

	// GetDirectionStats 获取方向统计
	GetDirectionStats(ctx context.Context, filter *AnalyticsFilter) (*DirectionStatistics, error)

	// GetRiskMetrics 获取风险指标
	GetRiskMetrics(ctx context.Context, filter *AnalyticsFilter) (*RiskMetrics, error)

	// GetSymbolStats 获取币种统计
	GetSymbolStats(ctx context.Context, filter *AnalyticsFilter) (map[string]*SymbolStatistics, error)

	// GetTimeSeriesStats 获取时间序列统计
	GetTimeSeriesStats(ctx context.Context, filter *AnalyticsFilter) (*TimeSeriesStatistics, error)

	// GetPairStats 获取配对统计
	GetPairStats(ctx context.Context, filter *AnalyticsFilter) (*PairStatistics, error)

	// GetFrequencyStats 获取交易频率统计
	GetFrequencyStats(ctx context.Context, filter *AnalyticsFilter) (*TradeFrequencyStats, error)

	// GetActionStats 获取交易类型统计
	GetActionStats(ctx context.Context, filter *AnalyticsFilter) (*ActionStatistics, error)

	// GetTrendAnalysis 获取趋势分析
	GetTrendAnalysis(ctx context.Context, filter *AnalyticsFilter) (*TrendAnalysis, error)

	// GetStreakStats 获取连续统计（新增）
	GetStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error)
}

// service 实现
type service struct {
	repo        Repository
	analyzer    Analyzer
	pairMatcher PairMatcher
}

// NewService 创建Service实例
func NewService(repo Repository, analyzer Analyzer, pairMatcher PairMatcher) Service {
	return &service{
		repo:        repo,
		analyzer:    analyzer,
		pairMatcher: pairMatcher,
	}
}

// GetAnalytics 获取完整分析
func (s *service) GetAnalytics(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error) {
	analytics, err := s.analyzer.Analyze(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("执行分析失败: %w", err)
	}

	// 如果请求包含配对分析，添加配对统计
	if filter.IncludePairs {
		pairStats, err := s.pairMatcher.GetPairStatistics(ctx, filter)
		if err == nil {
			analytics.PairStats = pairStats
		}
	}

	return analytics, nil
}

// GetOverview 获取概览统计
func (s *service) GetOverview(ctx context.Context, filter *AnalyticsFilter) (*TradeOverview, error) {
	return s.repo.GetOverviewStats(ctx, filter)
}

// GetPnLStats 获取盈亏统计
func (s *service) GetPnLStats(ctx context.Context, filter *AnalyticsFilter) (*PnLStatistics, error) {
	return s.repo.GetPnLStats(ctx, filter)
}

// GetWinRateStats 获取胜率统计
func (s *service) GetWinRateStats(ctx context.Context, filter *AnalyticsFilter) (*WinRateStatistics, error) {
	return s.repo.GetWinRateStats(ctx, filter)
}

// GetFeeStats 获取费用统计
func (s *service) GetFeeStats(ctx context.Context, filter *AnalyticsFilter) (*FeeStatistics, error) {
	return s.repo.GetFeeStats(ctx, filter)
}

// GetDirectionStats 获取方向统计
func (s *service) GetDirectionStats(ctx context.Context, filter *AnalyticsFilter) (*DirectionStatistics, error) {
	return s.repo.GetDirectionStats(ctx, filter)
}

// GetRiskMetrics 获取风险指标
func (s *service) GetRiskMetrics(ctx context.Context, filter *AnalyticsFilter) (*RiskMetrics, error) {
	return s.analyzer.CalculateRiskMetrics(ctx, filter)
}

// GetSymbolStats 获取币种统计
func (s *service) GetSymbolStats(ctx context.Context, filter *AnalyticsFilter) (map[string]*SymbolStatistics, error) {
	return s.repo.GetSymbolStats(ctx, filter)
}

// GetTimeSeriesStats 获取时间序列统计
func (s *service) GetTimeSeriesStats(ctx context.Context, filter *AnalyticsFilter) (*TimeSeriesStatistics, error) {
	stats := &TimeSeriesStatistics{}

	// 根据 group_by 参数决定返回哪些统计
	if filter.GroupBy == "day" || filter.GroupBy == "" {
		dailyStats, err := s.repo.GetDailyStats(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("获取每日统计失败: %w", err)
		}
		stats.DailyStats = dailyStats
	}

	if filter.GroupBy == "week" || filter.GroupBy == "" {
		weeklyStats, err := s.repo.GetWeeklyStats(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("获取每周统计失败: %w", err)
		}
		stats.WeeklyStats = weeklyStats
	}

	if filter.GroupBy == "month" || filter.GroupBy == "" {
		monthlyStats, err := s.repo.GetMonthlyStats(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("获取每月统计失败: %w", err)
		}
		stats.MonthlyStats = monthlyStats
	}

	return stats, nil
}

// GetPairStats 获取配对统计
func (s *service) GetPairStats(ctx context.Context, filter *AnalyticsFilter) (*PairStatistics, error) {
	return s.pairMatcher.GetPairStatistics(ctx, filter)
}

// GetFrequencyStats 获取交易频率统计
func (s *service) GetFrequencyStats(ctx context.Context, filter *AnalyticsFilter) (*TradeFrequencyStats, error) {
	return s.repo.GetFrequencyStats(ctx, filter)
}

// GetActionStats 获取交易类型统计
func (s *service) GetActionStats(ctx context.Context, filter *AnalyticsFilter) (*ActionStatistics, error) {
	return s.repo.GetActionStats(ctx, filter)
}

// GetTrendAnalysis 获取趋势分析
func (s *service) GetTrendAnalysis(ctx context.Context, filter *AnalyticsFilter) (*TrendAnalysis, error) {
	return s.repo.GetTrendAnalysis(ctx, filter)
}

// GetStreakStats 获取连续统计
func (s *service) GetStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error) {
	return s.analyzer.CalculateStreakStats(ctx, filter)
}

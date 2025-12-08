package review

import (
	"context"
	"time"

	"nofx/trade_history"
)

// DecisionLogReader 决策日志读取接口
type DecisionLogReader interface {
	GetRecordsByTimeRange(startTime, endTime time.Time) ([]*DecisionRecord, error)
}

// TradeHistoryReader 交易历史读取接口
type TradeHistoryReader interface {
	FindByFilter(ctx context.Context, filter *trade_history.TradeRecordFilter) ([]*trade_history.TradeRecord, error)
}

// DEXDataProvider DEX数据提供接口
type DEXDataProvider interface {
	GetFillsByTimeRange(startTime, endTime time.Time) ([]trade_history.ExchangeFill, error)
}

// TradeAnalyticsService 交易分析服务接口
type TradeAnalyticsService interface {
	GetAnalytics(ctx context.Context, filter interface{}) (interface{}, error)
}

// RuleEngine 规则引擎接口
type RuleEngine interface {
	CheckDecision(decision *DecisionRecord) ([]Violation, error)
	GetRuleDefinitions() []RuleDefinition
}

// PerformanceAnalyzer 表现分析接口
type PerformanceAnalyzer interface {
	AnalyzePerformance(trades []*trade_history.TradeRecord, decisions []*DecisionRecord) (*PerformanceAnalysis, error)
}

// DEXValidationAnalyzer DEX验证分析接口
type DEXValidationAnalyzer interface {
	AnalyzeExecutionQuality(matches []*DecisionMatch) (*ExecutionQuality, error)
	CalculateDEXMetrics(matches []*DecisionMatch, unmatchedDecisions []*DecisionRecord, unmatchedDEXTrades []trade_history.ExchangeFill) (*DEXValidationMetrics, error)
}

// DecisionMatcher 决策匹配器接口
type DecisionMatcher interface {
	Match(decisions []*DecisionRecord, dexTrades []trade_history.ExchangeFill) ([]*DecisionMatch, []*DecisionRecord, []trade_history.ExchangeFill, error)
}

// Reporter 报告生成接口
type Reporter interface {
	GenerateReport(review *ReviewResult) (string, error) // 返回报告文件路径
}

// ReviewRepository 复盘记录数据访问接口
type ReviewRepository interface {
	Save(ctx context.Context, record *ReviewRecord) error
	FindByID(ctx context.Context, id int64) (*ReviewRecord, error)
	FindByTraderID(ctx context.Context, traderID string, limit int) ([]*ReviewRecord, error)
	FindByTimeRange(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*ReviewRecord, error)
}

// ViolationMetricsCalculator 规则违反指标计算器接口
type ViolationMetricsCalculator interface {
	Calculate(violations []Violation, totalDecisions int) (*RuleViolationMetrics, error)
}


package trade_analytics

import (
	"time"
)

// TradeAnalytics 交易分析结果
type TradeAnalytics struct {
	// 基础统计
	Overview *TradeOverview `json:"overview"`

	// 盈亏统计
	PnLStats *PnLStatistics `json:"pnl_stats"`

	// 胜率统计
	WinRateStats *WinRateStatistics `json:"win_rate_stats"`

	// 费用统计
	FeeStats *FeeStatistics `json:"fee_stats"`

	// 方向统计
	DirectionStats *DirectionStatistics `json:"direction_stats"`

	// 风险指标
	RiskMetrics *RiskMetrics `json:"risk_metrics"`

	// 连续统计
	StreakStats *StreakStatistics `json:"streak_stats"`

	// 币种统计
	SymbolStats map[string]*SymbolStatistics `json:"symbol_stats"`

	// 时间维度统计
	TimeSeriesStats *TimeSeriesStatistics `json:"time_series_stats"`

	// 配对统计
	PairStats *PairStatistics `json:"pair_stats"`

	// 交易频率统计
	FrequencyStats *TradeFrequencyStats `json:"frequency_stats"`

	// 趋势分析
	TrendAnalysis *TrendAnalysis `json:"trend_analysis"`

	// 交易类型统计
	ActionStats *ActionStatistics `json:"action_stats"`
}

// TradeOverview 交易概览
type TradeOverview struct {
	TotalTrades     int `json:"total_trades"`      // 总交易笔数
	OpenTrades     int `json:"open_trades"`        // 开仓笔数
	CloseTrades    int `json:"close_trades"`       // 平仓笔数
	CompletedTrades int `json:"completed_trades"`  // 已完成交易数（有PnL）
	UnclosedTrades  int `json:"unclosed_trades"`  // 未平仓交易数
}

// PnLStatistics 盈亏统计
type PnLStatistics struct {
	TotalPnL     float64 `json:"total_pnl"`      // 总盈亏
	TotalProfit  float64 `json:"total_profit"`  // 总盈利
	TotalLoss    float64 `json:"total_loss"`    // 总亏损（绝对值）
	NetPnL       float64 `json:"net_pnl"`       // 净盈亏（总盈亏-总费用）
	AvgPnL       float64 `json:"avg_pnl"`       // 平均盈亏
	MaxWin       float64 `json:"max_win"`       // 最大单笔盈利
	MaxLoss      float64 `json:"max_loss"`      // 最大单笔亏损（绝对值）
	ProfitFactor float64 `json:"profit_factor"` // 盈亏比
}

// WinRateStatistics 胜率统计
type WinRateStatistics struct {
	WinRate         float64 `json:"win_rate"`          // 胜率（%）
	WinningTrades   int     `json:"winning_trades"`    // 盈利交易数
	LosingTrades    int     `json:"losing_trades"`     // 亏损交易数
	BreakEvenTrades int     `json:"break_even_trades"` // 持平交易数
	AvgWin          float64 `json:"avg_win"`           // 平均盈利
	AvgLoss         float64 `json:"avg_loss"`          // 平均亏损（绝对值）
}

// FeeStatistics 费用统计
type FeeStatistics struct {
	TotalFees  float64 `json:"total_fees"`   // 总费用
	OpenFees   float64 `json:"open_fees"`   // 开仓费用
	CloseFees  float64 `json:"close_fees"`  // 平仓费用
	AvgFee     float64 `json:"avg_fee"`     // 平均费用
	FeeRatio   float64 `json:"fee_ratio"`   // 费用占比（%）
	BuilderFees float64 `json:"builder_fees"` // Builder费用总和
}

// DirectionStatistics 方向统计
type DirectionStatistics struct {
	LongStats  *SideStatistics     `json:"long_stats"`  // 做多统计
	ShortStats *SideStatistics     `json:"short_stats"` // 做空统计
	Preference *DirectionPreference `json:"preference"` // 方向偏好
}

// SideStatistics 单方向统计
type SideStatistics struct {
	TotalTrades int     `json:"total_trades"` // 交易数
	TotalPnL    float64 `json:"total_pnl"`    // 总盈亏
	WinRate     float64 `json:"win_rate"`     // 胜率（%）
	AvgPnL      float64 `json:"avg_pnl"`      // 平均盈亏
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	MaxDrawdown        float64    `json:"max_drawdown"`         // 最大回撤
	MaxDrawdownPercent float64    `json:"max_drawdown_percent"` // 最大回撤百分比
	DrawdownDuration   int64      `json:"drawdown_duration"`    // 回撤持续时间（秒）
	RecoveryTime       int64      `json:"recovery_time"`        // 恢复时间（秒）
	Volatility         float64    `json:"volatility"`           // 波动率
	SharpeRatio        float64    `json:"sharpe_ratio"`        // 夏普比率
	DrawdownHistory    []Drawdown `json:"drawdown_history"`     // 回撤历史
}

// Drawdown 回撤记录
type Drawdown struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	PeakValue   float64   `json:"peak_value"`
	TroughValue float64   `json:"trough_value"`
	Drawdown    float64   `json:"drawdown"`
	Duration    int64     `json:"duration"` // 秒
}

// StreakStatistics 连续统计
type StreakStatistics struct {
	LongestWinningStreak int        `json:"longest_winning_streak"` // 最长连胜
	LongestLosingStreak   int        `json:"longest_losing_streak"`  // 最长连亏
	CurrentStreak         int        `json:"current_streak"`         // 当前连胜/连亏（正数=连胜，负数=连亏）
	CurrentStreakType     string     `json:"current_streak_type"`   // "winning" 或 "losing"
	LastLossTimestamp     *time.Time `json:"last_loss_timestamp,omitempty"` // 最后一次亏损的时间（仅当连续亏损时设置）
}

// SymbolStatistics 币种统计
type SymbolStatistics struct {
	Symbol          string  `json:"symbol"`
	TotalTrades     int     `json:"total_trades"`
	CompletedTrades int     `json:"completed_trades"`
	TotalPnL        float64 `json:"total_pnl"`
	WinRate         float64 `json:"win_rate"`
	AvgPnL          float64 `json:"avg_pnl"`
	MaxWin          float64 `json:"max_win"`
	MaxLoss         float64 `json:"max_loss"`
	TotalFees       float64 `json:"total_fees"`
}

// TimeSeriesStatistics 时间序列统计
type TimeSeriesStatistics struct {
	DailyStats   []DailyStatistics   `json:"daily_stats"`   // 每日统计
	WeeklyStats  []WeeklyStatistics  `json:"weekly_stats"`  // 每周统计
	MonthlyStats []MonthlyStatistics `json:"monthly_stats"` // 每月统计
}

// DailyStatistics 每日统计
type DailyStatistics struct {
	Date           string  `json:"date"`            // YYYY-MM-DD
	TotalTrades   int     `json:"total_trades"`
	CompletedTrades int   `json:"completed_trades"`
	TotalPnL      float64 `json:"total_pnl"`
	TotalFees     float64 `json:"total_fees"`
	WinRate       float64 `json:"win_rate"`
}

// WeeklyStatistics 每周统计
type WeeklyStatistics struct {
	Week           string  `json:"week"`            // YYYY-WW
	TotalTrades   int     `json:"total_trades"`
	CompletedTrades int   `json:"completed_trades"`
	TotalPnL      float64 `json:"total_pnl"`
	TotalFees     float64 `json:"total_fees"`
	WinRate       float64 `json:"win_rate"`
}

// MonthlyStatistics 每月统计
type MonthlyStatistics struct {
	Month          string  `json:"month"`          // YYYY-MM
	TotalTrades   int     `json:"total_trades"`
	CompletedTrades int   `json:"completed_trades"`
	TotalPnL      float64 `json:"total_pnl"`
	TotalFees     float64 `json:"total_fees"`
	WinRate       float64 `json:"win_rate"`
}

// PairStatistics 配对统计
type PairStatistics struct {
	TotalPairs      int          `json:"total_pairs"`       // 成功配对数量
	PairSuccessRate float64      `json:"pair_success_rate"` // 配对成功率（%）
	AvgHoldingTime  int64        `json:"avg_holding_time"` // 平均持仓时间（秒）
	MinHoldingTime  int64        `json:"min_holding_time"`  // 最短持仓时间（秒）
	MaxHoldingTime  int64        `json:"max_holding_time"`  // 最长持仓时间（秒）
	UnpairedTrades  int          `json:"unpaired_trades"`   // 未配对记录数
	Pairs           []*TradePair `json:"pairs"`            // 配对详情
}

// TradePair 交易对
type TradePair struct {
	OpenRecord  *TradeRecord `json:"open_record"`  // 开仓记录
	CloseRecord *TradeRecord `json:"close_record"` // 平仓记录
	MatchedQty  float64     `json:"matched_qty"`  // 匹配的数量
	HoldingTime int64       `json:"holding_time"`  // 持仓时间（秒）
	PnL         float64     `json:"pnl"`          // 盈亏（来自CloseRecord.PnL，按比例分配）
	OpenFee     float64     `json:"open_fee"`     // 开仓费用（按比例）
	CloseFee    float64     `json:"close_fee"`    // 平仓费用（按比例）
}

// TradeRecord 交易记录（引用 trade_history 包的类型）
type TradeRecord struct {
	ID                  int64      `json:"id" db:"id"`
	TraderID            string     `json:"trader_id" db:"trader_id"`
	Symbol              string     `json:"symbol" db:"symbol"`
	Action              string     `json:"action" db:"action"`
	Side                string     `json:"side" db:"side"`
	Quantity            float64    `json:"quantity" db:"quantity"`
	SignedQuantity      *float64   `json:"signed_quantity,omitempty" db:"signed_quantity"`
	ExecutionPrice      float64    `json:"execution_price" db:"execution_price"`
	PnL                 *float64   `json:"pnl,omitempty" db:"pnl"`
	Fee                 float64    `json:"fee" db:"fee"`
	FeeToken            *string    `json:"fee_token,omitempty" db:"fee_token"`
	ExchangeOrderID     *string    `json:"exchange_order_id,omitempty" db:"exchange_order_id"`
	ExchangeTradeID     *string    `json:"exchange_trade_id,omitempty" db:"exchange_trade_id"`
	ExchangeHash        *string    `json:"exchange_hash,omitempty" db:"exchange_hash"`
	RawDir              string     `json:"raw_dir" db:"raw_dir"`
	StartPosition       *float64   `json:"start_position,omitempty" db:"start_position"`
	BuilderFee          *float64   `json:"builder_fee,omitempty" db:"builder_fee"`
	ExchangeSide        *string    `json:"exchange_side,omitempty" db:"exchange_side"`
	Timestamp           time.Time  `json:"timestamp" db:"timestamp"`
	ExchangeTimestamp   *time.Time `json:"exchange_timestamp,omitempty" db:"exchange_timestamp"`
	ExchangeTimestampMs *int64     `json:"exchange_timestamp_ms,omitempty" db:"exchange_timestamp_ms"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// TradeFrequencyStats 交易频率统计
type TradeFrequencyStats struct {
	DailyAverageTrades float64  `json:"daily_average_trades"` // 日均交易数
	AvgTradeInterval   int64    `json:"avg_trade_interval"`   // 平均交易间隔（秒）
	MostActiveHours    []int    `json:"most_active_hours"`    // 最活跃时段（小时，0-23）
	MostActiveDays     []string `json:"most_active_days"`     // 最活跃日期（YYYY-MM-DD）
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	CumulativePnLSeries []CumulativePnLPoint `json:"cumulative_pnl_series"` // 累计盈亏曲线
	PnLDistribution     []PnLDistributionBin `json:"pnl_distribution"`     // 盈亏分布
	TradeFrequencyTrend []TradeFrequencyPoint `json:"trade_frequency_trend"` // 交易频率趋势
}

// CumulativePnLPoint 累计盈亏点
type CumulativePnLPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	CumulativePnL float64   `json:"cumulative_pnl"`
}

// PnLDistributionBin 盈亏分布区间
type PnLDistributionBin struct {
	Range string `json:"range"` // 如 "0-10", "10-20", "-10-0"
	Count int    `json:"count"` // 该区间的交易数量
}

// TradeFrequencyPoint 交易频率点
type TradeFrequencyPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	TradeCount int       `json:"trade_count"` // 该时间点的交易数量
}

// ActionStatistics 交易类型统计
type ActionStatistics struct {
	OpenStats  *ActionStats `json:"open_stats"`  // 开仓统计
	CloseStats *ActionStats `json:"close_stats"` // 平仓统计
}

// ActionStats 单类型统计
type ActionStats struct {
	TotalTrades int     `json:"total_trades"` // 交易数
	TotalPnL    float64 `json:"total_pnl"`    // 总盈亏（仅平仓有）
	TotalFees   float64 `json:"total_fees"`   // 总费用
	AvgPnL      float64 `json:"avg_pnl"`      // 平均盈亏（仅平仓有）
}

// DirectionPreference 方向偏好
type DirectionPreference struct {
	PreferredSide string  `json:"preferred_side"` // "long" 或 "short"
	LongRatio     float64 `json:"long_ratio"`     // 做多占比（%）
	ShortRatio    float64 `json:"short_ratio"`    // 做空占比（%）
	TotalTrades   int     `json:"total_trades"`   // 总交易数（用于计算比例）
}

// AnalyticsFilter 分析查询过滤器
type AnalyticsFilter struct {
	TraderID     string     `json:"trader_id"`      // 交易员ID（必需）
	Symbol       string     `json:"symbol"`         // 币种过滤（可选）
	Side         string     `json:"side"`           // 方向过滤：'long' 或 'short'（可选）
	Action       string     `json:"action"`         // 操作过滤：'open_long', 'close_long' 等（可选）
	StartTime    *time.Time `json:"start_time"`     // 开始时间（可选）
	EndTime      *time.Time `json:"end_time"`       // 结束时间（可选）
	GroupBy      string     `json:"group_by"`       // 分组方式：'day', 'week', 'month', 'symbol'（可选）
	IncludePairs bool       `json:"include_pairs"` // 是否包含配对分析（默认false）
}


package review

import (
	"time"

	"nofx/trade_history"
)

// ReviewRecord 复盘记录（数据库存储）
type ReviewRecord struct {
	ID           int64     `json:"id" db:"id"`
	TraderID     string    `json:"trader_id" db:"trader_id"`
	StartTime    time.Time `json:"start_time" db:"start_time"`
	EndTime      time.Time `json:"end_time" db:"end_time"`
	ReportPath   string    `json:"report_path" db:"report_path"`
	Summary      string    `json:"summary" db:"summary"` // JSON
	Metrics      string    `json:"metrics" db:"metrics"` // JSON
	TotalTrades  int       `json:"total_trades" db:"total_trades"`
	TotalPnL     float64   `json:"total_pnl" db:"total_pnl"`
	WinRate      float64   `json:"win_rate" db:"win_rate"`
	ErrorCount   int       `json:"error_count" db:"error_count"`
	Status       string    `json:"status" db:"status"` // success/partial/failed
	ErrorMessage string    `json:"error_message" db:"error_message"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ReviewResult 复盘结果（运行时数据结构）
type ReviewResult struct {
	TraderID         string
	StartTime        time.Time
	EndTime          time.Time
	Status           string
	ErrorMessage     string
	Performance      *PerformanceAnalysis
	Violations       []Violation
	DEXValidation    *DEXValidationMetrics
	ExecutionQuality *ExecutionQuality
	ReportPath       string
	Metrics          *StandardizedMetrics
	Duration         time.Duration
}

// StandardizedMetrics 标准化指标
type StandardizedMetrics struct {
	Performance    *PerformanceAnalysis
	RuleViolations *RuleViolationMetrics
	DEXValidation  *DEXValidationMetrics
}

// DecisionMatch 决策与DEX交易的匹配结果
type DecisionMatch struct {
	DecisionID        string
	DEXTradeID        string
	MatchMethod       string // "order_id" | "time_window" | "quantity_price"
	MatchConfidence   float64
	Slippage          float64
	ExecutionDelay    int64 // 毫秒
	FeeDifference     float64
	QuantityDifference float64
	Decision          *DecisionRecord
	DEXTrade          *trade_history.ExchangeFill
}

// DecisionRecord 决策记录（从logger包导入的类型别名，保持接口一致）
type DecisionRecord struct {
	Timestamp      time.Time
	CycleNumber    int
	SystemPrompt   string
	InputPrompt    string
	CoTTrace       string
	DecisionJSON   string
	AccountState   AccountSnapshot
	Positions      []PositionSnapshot
	CandidateCoins []string
	Decisions      []DecisionAction
	ExecutionLog   []string
	Success        bool
	ErrorMessage   string
}

// AccountSnapshot 账户状态快照
type AccountSnapshot struct {
	TotalBalance          float64
	AvailableBalance      float64
	TotalUnrealizedProfit float64
	PositionCount         int
	MarginUsedPct         float64
}

// PositionSnapshot 持仓快照
type PositionSnapshot struct {
	Symbol           string
	Side             string
	PositionAmt      float64
	EntryPrice       float64
	MarkPrice        float64
	UnrealizedProfit float64
	Leverage         float64
	LiquidationPrice float64
}

// DecisionAction 决策动作
type DecisionAction struct {
	Action          string
	Symbol          string
	Quantity        float64
	Leverage        int
	Price           float64
	OrderID         int64
	Timestamp       time.Time
	Success         bool
	Error           string
	IsAutoTriggered bool
	WasStopLoss     bool
}

// PerformanceAnalysis 交易表现分析
type PerformanceAnalysis struct {
	// 基础统计
	TotalTrades     int
	OpenTrades      int
	CloseTrades     int
	CompletedTrades int

	// 盈亏统计
	TotalPnL float64
	NetPnL   float64 // 扣除手续费
	AvgPnL   float64
	MaxProfit float64
	MaxLoss   float64

	// 胜率统计
	WinRate    float64
	WinTrades  int
	LossTrades int

	// 费用统计
	TotalFees float64
	AvgFee    float64
	FeeRatio  float64 // 费用/总盈亏

	// 风险指标
	MaxDrawdown float64
	Volatility  float64
	SharpeRatio float64

	// 币种表现
	SymbolStats map[string]*SymbolPerformance
}

// SymbolPerformance 币种表现
type SymbolPerformance struct {
	Symbol        string
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnL      float64
	AvgPnL        float64
}

// Violation 规则违反
type Violation struct {
	RuleID      string
	RuleName    string
	Type        string // "hard_constraint" | "risk_control" | "position_management"
	Severity    string // "critical" | "high" | "medium" | "low"
	Description string
	DecisionID  string
	Details     map[string]interface{}
}

// RuleDefinition 规则定义
type RuleDefinition struct {
	ID          string
	Name        string
	Type        string
	Severity    string
	Description string
	PromptRef   string
	Priority    int
}

// DEXValidationMetrics DEX验证指标
type DEXValidationMetrics struct {
	DecisionExecutionRate float64
	UnmatchedDecisions    int
	UnmatchedDEXTrades    int
	AverageSlippage       float64
	AverageExecutionDelay int64 // 毫秒
	FeeDifference         float64
	TotalFundingFees      float64
}

// ExecutionQuality 执行质量
type ExecutionQuality struct {
	TotalMatches             int
	AverageSlippage          float64
	AverageExecutionDelay    int64 // 毫秒
	MaxSlippage              float64
	MaxExecutionDelay        int64 // 毫秒
	SlippageDistribution     map[string]int
	ExecutionDelayDistribution map[string]int
}

// RuleViolationMetrics 规则违反指标
type RuleViolationMetrics struct {
	HardConstraintViolationRate     float64
	RiskControlViolationRate        float64
	PositionManagementViolationRate float64
	TotalViolations                 int
	ViolationSeverityDistribution   map[string]int
}


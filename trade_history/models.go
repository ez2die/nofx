package trade_history

import "time"

// TradeRecord 交易记录模型（仅保留 Hyperliquid userfills 提供的字段及必要的派生字段）
type TradeRecord struct {
	ID                  int64      `json:"id" db:"id"`
	TraderID            string     `json:"trader_id" db:"trader_id"`
	Symbol              string     `json:"symbol" db:"symbol"`
	Action              string     `json:"action" db:"action"` // "open_long", "open_short", "close_long", "close_short"
	Side                string     `json:"side" db:"side"`     // "long", "short" 或 ""
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

// TradeRecordFilter 查询过滤器
type TradeRecordFilter struct {
	TraderID  string
	Symbol    string
	Action    string // "open_long", "close_long", etc.
	Side      string // "long" or "short"
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int    // 默认100
	Offset    int    // 默认0
	OrderBy   string // "timestamp DESC" 或 "timestamp ASC"
}

// TradeStatistics 交易统计
type TradeStatistics struct {
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
	AvgWin        float64 `json:"avg_win"`
	AvgLoss       float64 `json:"avg_loss"`
	ProfitFactor  float64 `json:"profit_factor"`
	TotalFees     float64 `json:"total_fees"`
}

// SymbolPerformance 币种粒度表现
type SymbolPerformance struct {
	Symbol        string  `json:"symbol" db:"symbol"`
	TotalTrades   int     `json:"total_trades" db:"total_trades"`
	WinningTrades int     `json:"winning_trades" db:"winning_trades"`
	LosingTrades  int     `json:"losing_trades" db:"losing_trades"`
	WinRate       float64 `json:"win_rate" db:"-"`
	TotalPnL      float64 `json:"total_pn_l" db:"total_pn_l"`
	AvgPnL        float64 `json:"avg_pn_l" db:"avg_pn_l"`
}

// TradeOutcome 最近平仓交易输出
type TradeOutcome struct {
	Symbol         string     `json:"symbol"`
	Action         string     `json:"action"`
	Side           string     `json:"side"`
	Quantity       float64    `json:"quantity"`
	SignedQuantity *float64   `json:"signed_quantity,omitempty"`
	ExecutionPrice float64    `json:"execution_price"`
	PnL            float64    `json:"pn_l"`
	Fee            float64    `json:"fee"`
	Timestamp      time.Time  `json:"timestamp"`
	ExchangeTime   *time.Time `json:"exchange_timestamp,omitempty"`
}

// PnLPoint Sharpe 序列元素
type PnLPoint struct {
	PnL            float64   `json:"pnl"`
	Quantity       float64   `json:"quantity"`
	SignedQuantity *float64  `json:"signed_quantity,omitempty"`
	ExecutionPrice float64   `json:"execution_price"`
	Timestamp      time.Time `json:"timestamp"`
}

// PerformanceOptions 统计选项
type PerformanceOptions struct {
	StartTime    *time.Time
	EndTime      *time.Time
	RecentLimit  int
	SharpeWindow int
}

// TradePerformance 完整表现视图
type TradePerformance struct {
	TradeStatistics
	SharpeRatio  float64                       `json:"sharpe_ratio"`
	RecentTrades []*TradeOutcome               `json:"recent_trades"`
	SymbolStats  map[string]*SymbolPerformance `json:"symbol_stats"`
	BestSymbol   string                        `json:"best_symbol"`
	WorstSymbol  string                        `json:"worst_symbol"`
	GeneratedAt  time.Time                     `json:"generated_at"`
}

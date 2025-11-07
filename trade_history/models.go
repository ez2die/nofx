package trade_history

import "time"

// TradeRecord 交易记录模型
type TradeRecord struct {
	ID                  int64      `json:"id" db:"id"`
	TraderID            string     `json:"trader_id" db:"trader_id"`
	Symbol              string     `json:"symbol" db:"symbol"`
	Side                string     `json:"side" db:"side"`                    // "long" or "short"
	Action              string     `json:"action" db:"action"`                // "open_long", "open_short", "close_long", "close_short"
	Quantity            float64    `json:"quantity" db:"quantity"`
	Leverage            int        `json:"leverage" db:"leverage"`
	EntryPrice          *float64   `json:"entry_price,omitempty" db:"entry_price"`    // 仅平仓时有效
	ExitPrice           *float64   `json:"exit_price,omitempty" db:"exit_price"`      // 仅平仓时有效
	ExecutionPrice      float64    `json:"execution_price" db:"execution_price"`
	PnL                 *float64   `json:"pnl,omitempty" db:"pnl"`                    // 仅平仓时有效
	PnLPct              *float64   `json:"pnl_pct,omitempty" db:"pnl_pct"`            // 仅平仓时有效
	OrderID             *string    `json:"order_id,omitempty" db:"order_id"`
	ExchangeOrderID     *string    `json:"exchange_order_id,omitempty" db:"exchange_order_id"`
	ExchangeTradeID     *string    `json:"exchange_trade_id,omitempty" db:"exchange_trade_id"`
	ExchangeHash        *string    `json:"exchange_hash,omitempty" db:"exchange_hash"`
	Fee                 float64    `json:"fee" db:"fee"`
	FeeToken            *string    `json:"fee_token,omitempty" db:"fee_token"`
	IsAutoTriggered     bool       `json:"is_auto_triggered" db:"is_auto_triggered"`
	WasStopLoss         bool       `json:"was_stop_loss" db:"was_stop_loss"`
	WasTakeProfit       bool       `json:"was_take_profit" db:"was_take_profit"`
	CycleNumber         *int       `json:"cycle_number,omitempty" db:"cycle_number"`
	Source              string     `json:"source" db:"source"`                // "api" or "sync"
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
	Action    string     // "open_long", "close_long", etc.
	Side      string     // "long" or "short"
	StartTime *time.Time
	EndTime   *time.Time
	CycleFrom *int
	CycleTo   *int
	Limit     int        // 默认100
	Offset    int        // 默认0
	OrderBy   string     // "timestamp DESC" 或 "timestamp ASC"
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


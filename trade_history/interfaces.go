package trade_history

import "time"

// ExchangeFillsProvider 交易所成交记录提供者接口
type ExchangeFillsProvider interface {
	// GetRecentFills 获取最近的成交记录
	GetRecentFills(limit int) ([]ExchangeFill, error)

	// GetFillsByTimeRange 按时间范围获取成交记录
	GetFillsByTimeRange(startTime, endTime time.Time) ([]ExchangeFill, error)
}

// ExchangeFill 交易所成交记录（统一格式）
type ExchangeFill struct {
	// 基本信息
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"` // "long" or "short"
	Dir         string    `json:"dir"`  // "Open" or "Close"
	RawDir      string    `json:"raw_dir"`
	Quantity    float64   `json:"quantity"`
	SignedQty   *float64  `json:"signed_quantity,omitempty"`
	Price       float64   `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
	TimestampMs int64     `json:"timestamp_ms"`

	// 订单信息
	OrderID      string `json:"order_id"`
	ExchangeOid  int64  `json:"exchange_oid"` // 交易所订单ID
	ExchangeTid  int64  `json:"exchange_tid"` // 交易所交易ID
	ExchangeHash string `json:"exchange_hash"`
	ExchangeSide string `json:"exchange_side"`

	// 盈亏信息（仅平仓时有效）
	ClosedPnl *float64 `json:"closed_pnl,omitempty"`

	// 手续费
	Fee        float64  `json:"fee"`
	FeeToken   string   `json:"fee_token"`
	BuilderFee *float64 `json:"builder_fee,omitempty"`

	// 持仓信息
	StartPosition *float64 `json:"start_position,omitempty"`
}

# 交易历史数据库持久化模块设计文档

## 1. 概述

### 1.1 目标
- 将交易历史持久化到数据库，与交易所API保持同步
- 作为独立模块开发，不影响现有功能
- 支持实时写入和定期同步
- 提供查询接口，支持高效统计和分析

### 1.2 设计原则
- **独立性**：模块可独立运行，不依赖现有交易流程
- **可扩展性**：支持多种交易所（Hyperliquid、Binance等）
- **数据一致性**：确保数据库与交易所数据一致
- **向后兼容**：不影响现有JSON日志系统
- **可配置**：支持启用/禁用、同步间隔等配置

## 2. 模块架构

### 2.1 目录结构

```
nofx/
├── trade_history/          # 新模块目录
│   ├── models.go          # 数据模型定义
│   ├── repository.go      # 数据库操作（Repository模式）
│   ├── service.go         # 业务逻辑服务
│   ├── sync.go            # 交易所同步服务
│   ├── events.go          # 事件监听器（可选）
│   └── interfaces.go      # 接口定义
├── config/
│   └── database.go        # 数据库表创建（添加trade_history表）
└── api/
    └── server.go          # API接口（添加trade_history相关接口）
```

### 2.2 模块职责

```
┌─────────────────────────────────────────────────────────┐
│              Trade History Module                       │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐    ┌──────────────┐                 │
│  │  Service     │───▶│  Repository │                 │
│  │  (业务逻辑)   │    │  (数据库操作) │                 │
│  └──────────────┘    └──────────────┘                 │
│         │                    │                          │
│         │                    ▼                          │
│         │            ┌──────────────┐                   │
│         │            │   Database  │                   │
│         │            │  trade_history│                   │
│         │            └──────────────┘                   │
│         │                                              │
│         ▼                                              │
│  ┌──────────────┐                                      │
│  │  Sync        │                                      │
│  │  (交易所同步) │                                      │
│  └──────────────┘                                      │
│         │                                              │
│         ▼                                              │
│  ┌──────────────┐                                      │
│  │  Exchange    │                                      │
│  │  API         │                                      │
│  └──────────────┘                                      │
└─────────────────────────────────────────────────────────┘
```

## 3. 数据库设计

### 3.0 关系维护策略

#### 3.0.1 Trader与交易历史的关系

**设计决策：采用软外键方案（无外键约束）**

**原因**：
1. **数据保留**：交易历史是重要的业务数据，即使trader被删除，也应保留历史记录（符合审计和合规要求）
2. **数据恢复**：如果trader被误删，可以通过历史数据恢复
3. **查询性能**：通过索引优化查询性能，无需外键约束
4. **实现简单**：不需要修改现有删除逻辑

**关系维护方式**：
- `trader_id` 字段：仅作为普通字段，无外键约束
- 索引优化：通过 `idx_trade_history_trader_id` 索引优化查询性能
- 应用层验证：可选性验证trader是否存在（不阻止保存）
- 查询时标记：可选性标记trader状态（是否仍然存在）

**详细说明**：参见 `docs/TRADE_HISTORY_RELATIONSHIP_ANALYSIS.md`

### 3.1 表结构

```sql
CREATE TABLE IF NOT EXISTS trade_history (
    -- 主键
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- 关联信息
    trader_id TEXT NOT NULL,                    -- 交易员ID
    symbol TEXT NOT NULL,                        -- 交易币种（如 "BTCUSDT"）
    
    -- 交易信息
    side TEXT NOT NULL,                          -- 'long' 或 'short'
    action TEXT NOT NULL,                        -- 'open_long', 'open_short', 'close_long', 'close_short'
    quantity REAL NOT NULL,                      -- 交易数量
    leverage INTEGER NOT NULL,                   -- 杠杆倍数
    
    -- 价格信息
    entry_price REAL,                            -- 开仓价格（仅平仓时有效）
    exit_price REAL,                            -- 平仓价格（仅平仓时有效）
    execution_price REAL NOT NULL,               -- 实际成交价格
    
    -- 盈亏信息（仅平仓时计算）
    pnl REAL,                                   -- 盈亏（USDT）
    pnl_pct REAL,                               -- 盈亏百分比（%）
    
    -- 订单信息
    order_id TEXT,                              -- 交易所订单ID（如果有）
    exchange_order_id TEXT,                     -- 交易所内部订单ID（如Hyperliquid的Oid）
    exchange_trade_id TEXT,                     -- 交易所交易ID（如Hyperliquid的Tid）
    exchange_hash TEXT,                         -- 交易所交易哈希
    
    -- 手续费
    fee REAL DEFAULT 0,                         -- 手续费（USDT）
    fee_token TEXT,                             -- 手续费币种
    
    -- 自动触发信息
    is_auto_triggered BOOLEAN DEFAULT 0,        -- 是否自动触发（止盈止损）
    was_stop_loss BOOLEAN DEFAULT 0,            -- 是否止损
    was_take_profit BOOLEAN DEFAULT 0,          -- 是否止盈
    
    -- 上下文信息
    cycle_number INTEGER,                       -- 决策周期编号
    source TEXT DEFAULT 'api',                  -- 数据来源：'api'（API返回）或 'sync'（同步）
    
    -- 时间信息（优先级：交易所时间戳 > 本地时间）
    timestamp DATETIME NOT NULL,                -- 主时间戳（优先使用交易所时间，否则使用本地时间）
    exchange_timestamp DATETIME,                -- 交易所时间戳（如果有，从交易所API获取）
    exchange_timestamp_ms INTEGER,             -- 交易所时间戳（毫秒，用于精确查询和排序）
    
    -- 元数据
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    
    -- 注意：不创建外键约束，采用软外键方案
    -- 理由：
    -- 1. 保留历史数据：即使trader被删除，交易历史仍保留（符合审计和合规要求）
    -- 2. 支持数据恢复：如果trader被误删，可以通过历史数据恢复
    -- 3. 查询性能：通过索引优化查询性能，无需外键约束
    -- 4. 实现简单：不需要修改现有删除逻辑
    -- 应用层可选择性维护数据一致性（通过验证方法）
);

-- 索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
CREATE INDEX IF NOT EXISTS idx_trade_history_symbol ON trade_history(symbol);
CREATE INDEX IF NOT EXISTS idx_trade_history_timestamp ON trade_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_trade_history_cycle_number ON trade_history(cycle_number);
CREATE INDEX IF NOT EXISTS idx_trade_history_action ON trade_history(action);
CREATE INDEX IF NOT EXISTS idx_trade_history_exchange_timestamp_ms ON trade_history(exchange_timestamp_ms);
CREATE INDEX IF NOT EXISTS idx_trade_history_order_id ON trade_history(order_id);
CREATE INDEX IF NOT EXISTS idx_trade_history_exchange_order_id ON trade_history(exchange_order_id);
CREATE INDEX IF NOT EXISTS idx_trade_history_exchange_hash ON trade_history(exchange_hash);

-- 复合索引：支持常用查询
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_time ON trade_history(trader_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_symbol ON trade_history(trader_id, symbol);

-- 触发器：自动更新 updated_at
CREATE TRIGGER IF NOT EXISTS update_trade_history_updated_at
    AFTER UPDATE ON trade_history
    BEGIN
        UPDATE trade_history SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;
```

### 3.2 数据字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | INTEGER | 主键，自增 |
| `trader_id` | TEXT | 交易员ID（软外键，无外键约束，仅通过索引优化查询） |
| `symbol` | TEXT | 交易币种（如 "BTCUSDT"） |
| `side` | TEXT | 方向：'long' 或 'short' |
| `action` | TEXT | 操作类型：'open_long', 'open_short', 'close_long', 'close_short' |
| `quantity` | REAL | 交易数量（币种数量） |
| `leverage` | INTEGER | 杠杆倍数 |
| `entry_price` | REAL | 开仓价格（仅平仓时有效） |
| `exit_price` | REAL | 平仓价格（仅平仓时有效） |
| `execution_price` | REAL | 实际成交价格（所有操作都有） |
| `pnl` | REAL | 盈亏（USDT，仅平仓时计算） |
| `pnl_pct` | REAL | 盈亏百分比（%，仅平仓时计算） |
| `order_id` | TEXT | 订单ID（通用格式，如 "0"） |
| `exchange_order_id` | TEXT | 交易所订单ID（如Hyperliquid的Oid） |
| `exchange_trade_id` | TEXT | 交易所交易ID（如Hyperliquid的Tid） |
| `exchange_hash` | TEXT | 交易所交易哈希 |
| `fee` | REAL | 手续费（USDT） |
| `fee_token` | TEXT | 手续费币种 |
| `is_auto_triggered` | BOOLEAN | 是否自动触发 |
| `was_stop_loss` | BOOLEAN | 是否止损 |
| `was_take_profit` | BOOLEAN | 是否止盈 |
| `cycle_number` | INTEGER | 决策周期编号 |
| `source` | TEXT | 数据来源：'api' 或 'sync' |
| `timestamp` | DATETIME | 主时间戳（交易时间）<br>优先级：1. 交易所时间戳（如果可用）<br>2. 本地时间（备用） |
| `exchange_timestamp` | DATETIME | 交易所时间戳（从交易所API获取，如果有） |
| `exchange_timestamp_ms` | INTEGER | 交易所时间戳（毫秒，用于精确查询和排序） |
| `created_at` | DATETIME | 创建时间（数据库记录创建时间） |
| `updated_at` | DATETIME | 更新时间（数据库记录更新时间） |

### 3.3 关系维护说明

#### 3.3.1 Trader关系维护

**设计决策**：采用软外键方案（无外键约束）

**表结构**：
```sql
-- trader_id字段：仅作为普通字段，无外键约束
trader_id TEXT NOT NULL,  -- 无外键约束

-- 索引优化查询
CREATE INDEX IF NOT EXISTS idx_trade_history_trader_id ON trade_history(trader_id);
```

**应用层维护**：
1. **写入时验证（可选）**：在保存时验证trader是否存在，记录警告日志但不阻止保存
2. **查询时标记（可选）**：查询时标记trader状态，但不影响查询结果
3. **定期清理（可选）**：定期清理孤立记录（如果trader被删除超过一定时间）

**优点**：
- ✅ 保留历史数据：即使trader被删除，交易历史仍保留（符合审计和合规要求）
- ✅ 支持数据恢复：如果trader被误删，可以通过历史数据恢复
- ✅ 查询性能：通过索引优化，不影响查询性能
- ✅ 实现简单：不需要修改现有删除逻辑

**详细分析**：参见 `docs/TRADE_HISTORY_RELATIONSHIP_ANALYSIS.md`

#### 3.3.2 时间戳记录策略

**设计决策**：优先使用交易所时间戳，本地时间作为备用

**策略**：
1. **主时间戳**（`timestamp` 字段）：
   - **优先级1**：交易所时间戳（从 `UserFills` API 获取的 `Fill.Time`）
   - **优先级2**：本地时间（`time.Now()`，当交易所时间戳不可用时）

2. **辅助时间戳**：
   - `exchange_timestamp`：交易所时间戳（DATETIME格式）
   - `exchange_timestamp_ms`：交易所时间戳（毫秒，用于精确查询和排序）

**原因**：
1. **数据一致性**：与交易所数据完全一致，确保审计准确性
2. **可追溯性**：可以追溯到交易所原始记录
3. **同步友好**：与交易所同步时数据一致
4. **向后兼容**：如果交易所时间戳不可用，使用本地时间作为备用

**实施方式**：
- **实时写入**：尝试从 `UserFills` API 获取时间戳，如果不可用则使用本地时间
- **同步写入**：使用 `UserFills` API 返回的时间戳
- **自动触发**：尝试从 `UserFills` API 获取时间戳，如果不可用则使用本地时间

**详细分析**：参见 `docs/TRADE_HISTORY_TIMESTAMP_ANALYSIS.md`

## 4. 数据模型设计

### 4.1 核心模型

```go
// trade_history/models.go

package trade_history

import "time"

// TradeRecord 交易记录模型
type TradeRecord struct {
    ID                  int64     `json:"id" db:"id"`
    TraderID            string    `json:"trader_id" db:"trader_id"`
    Symbol              string    `json:"symbol" db:"symbol"`
    Side                string    `json:"side" db:"side"`                    // "long" or "short"
    Action              string    `json:"action" db:"action"`                // "open_long", "open_short", "close_long", "close_short"
    Quantity            float64   `json:"quantity" db:"quantity"`
    Leverage            int       `json:"leverage" db:"leverage"`
    EntryPrice          *float64  `json:"entry_price,omitempty" db:"entry_price"`    // 仅平仓时有效
    ExitPrice           *float64  `json:"exit_price,omitempty" db:"exit_price"`      // 仅平仓时有效
    ExecutionPrice      float64   `json:"execution_price" db:"execution_price"`
    PnL                 *float64  `json:"pnl,omitempty" db:"pnl"`                    // 仅平仓时有效
    PnLPct              *float64  `json:"pnl_pct,omitempty" db:"pnl_pct"`            // 仅平仓时有效
    OrderID             *string   `json:"order_id,omitempty" db:"order_id"`
    ExchangeOrderID     *string   `json:"exchange_order_id,omitempty" db:"exchange_order_id"`
    ExchangeTradeID     *string   `json:"exchange_trade_id,omitempty" db:"exchange_trade_id"`
    ExchangeHash        *string   `json:"exchange_hash,omitempty" db:"exchange_hash"`
    Fee                 float64   `json:"fee" db:"fee"`
    FeeToken            *string   `json:"fee_token,omitempty" db:"fee_token"`
    IsAutoTriggered     bool      `json:"is_auto_triggered" db:"is_auto_triggered"`
    WasStopLoss         bool      `json:"was_stop_loss" db:"was_stop_loss"`
    WasTakeProfit       bool      `json:"was_take_profit" db:"was_take_profit"`
    CycleNumber         *int      `json:"cycle_number,omitempty" db:"cycle_number"`
    Source              string    `json:"source" db:"source"`                // "api" or "sync"
    Timestamp           time.Time `json:"timestamp" db:"timestamp"`
    ExchangeTimestamp   *time.Time `json:"exchange_timestamp,omitempty" db:"exchange_timestamp"`
    ExchangeTimestampMs *int64    `json:"exchange_timestamp_ms,omitempty" db:"exchange_timestamp_ms"`
    CreatedAt           time.Time `json:"created_at" db:"created_at"`
    UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// TradeRecordFilter 查询过滤器
type TradeRecordFilter struct {
    TraderID    string
    Symbol      string
    Action      string              // "open_long", "close_long", etc.
    Side        string              // "long" or "short"
    StartTime   *time.Time
    EndTime     *time.Time
    CycleFrom   *int
    CycleTo     *int
    Limit       int                 // 默认100
    Offset      int                 // 默认0
    OrderBy     string              // "timestamp DESC" 或 "timestamp ASC"
}

// TradeStatistics 交易统计
type TradeStatistics struct {
    TotalTrades       int     `json:"total_trades"`
    WinningTrades     int     `json:"winning_trades"`
    LosingTrades      int     `json:"losing_trades"`
    WinRate           float64 `json:"win_rate"`
    TotalPnL          float64 `json:"total_pnl"`
    AvgWin            float64 `json:"avg_win"`
    AvgLoss           float64 `json:"avg_loss"`
    ProfitFactor      float64 `json:"profit_factor"`
    TotalFees         float64 `json:"total_fees"`
}
```

### 4.2 交易所适配器接口

```go
// trade_history/interfaces.go

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
    Symbol        string    `json:"symbol"`
    Side          string    `json:"side"`          // "long" or "short"
    Dir           string    `json:"dir"`           // "Open" or "Close"
    Quantity      float64   `json:"quantity"`
    Price         float64   `json:"price"`
    Timestamp     time.Time `json:"timestamp"`
    TimestampMs   int64     `json:"timestamp_ms"`
    
    // 订单信息
    OrderID       string    `json:"order_id"`
    ExchangeOid   int64     `json:"exchange_oid"`   // 交易所订单ID
    ExchangeTid   int64     `json:"exchange_tid"`   // 交易所交易ID
    ExchangeHash  string    `json:"exchange_hash"`
    
    // 盈亏信息（仅平仓时有效）
    ClosedPnl     *float64  `json:"closed_pnl,omitempty"`
    
    // 手续费
    Fee           float64   `json:"fee"`
    FeeToken      string    `json:"fee_token"`
    
    // 持仓信息
    StartPosition *float64  `json:"start_position,omitempty"`
}
```

## 5. Repository 层设计

### 5.1 Repository 接口

```go
// trade_history/repository.go

package trade_history

import (
    "context"
    "time"
)

// Repository 交易历史数据仓库接口
type Repository interface {
    // Save 保存交易记录
    Save(ctx context.Context, record *TradeRecord) error
    
    // SaveBatch 批量保存交易记录
    SaveBatch(ctx context.Context, records []*TradeRecord) error
    
    // FindByID 根据ID查询
    FindByID(ctx context.Context, id int64) (*TradeRecord, error)
    
    // FindByFilter 根据过滤器查询
    FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error)
    
    // CountByFilter 统计数量
    CountByFilter(ctx context.Context, filter *TradeRecordFilter) (int, error)
    
    // ExistsByExchangeID 检查是否存在（通过交易所订单ID或交易ID）
    ExistsByExchangeID(ctx context.Context, exchangeOid *int64, exchangeTid *int64, exchangeHash *string) (bool, error)
    
    // UpdateExecutionPrice 更新成交价格
    UpdateExecutionPrice(ctx context.Context, id int64, price float64) error
    
    // GetStatistics 获取统计信息
    GetStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error)
    
    // GetLatestByTrader 获取交易员最近的交易记录
    GetLatestByTrader(ctx context.Context, traderID string, limit int) ([]*TradeRecord, error)
    
    // ValidateTraderExists 验证trader是否存在（可选，用于数据一致性检查）
    ValidateTraderExists(ctx context.Context, traderID string) (bool, error)
}
```

### 5.2 Repository 实现

```go
// trade_history/repository.go (实现部分)

type repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
    return &repository{db: db}
}

// Save 实现
func (r *repository) Save(ctx context.Context, record *TradeRecord) error {
    // SQL INSERT 语句
    // 处理可选字段（NULL值）
    // 返回插入的ID
}

// ValidateTraderExists 实现（可选，用于数据一致性检查）
func (r *repository) ValidateTraderExists(ctx context.Context, traderID string) (bool, error) {
    var exists bool
    err := r.db.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM traders WHERE id = ?)
    `, traderID).Scan(&exists)
    return exists, err
}

// FindByFilter 实现
func (r *repository) FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error) {
    // 构建动态SQL查询
    // 处理各种过滤条件
    // 返回结果列表
    // 注意：即使trader已被删除，仍然可以查询交易历史
}
```

## 6. Service 层设计

### 6.1 Service 接口

```go
// trade_history/service.go

package trade_history

import (
    "context"
    "time"
)

// Service 交易历史服务接口
type Service interface {
    // RecordTrade 记录交易（从API返回的数据）
    RecordTrade(ctx context.Context, record *TradeRecord) error
    
    // RecordAutoTriggeredClose 记录自动触发的平仓（止盈止损）
    RecordAutoTriggeredClose(ctx context.Context, record *TradeRecord) error
    
    // GetTradeHistory 获取交易历史
    GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, int, error)
    
    // GetTradeStatistics 获取交易统计
    GetTradeStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error)
    
    // SyncFromExchange 从交易所同步交易记录
    SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error
}
```

### 6.2 Service 实现

```go
// trade_history/service.go (实现部分)

type service struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &service{repo: repo}
}

// RecordTrade 实现
func (s *service) RecordTrade(ctx context.Context, record *TradeRecord) error {
    // 1. 可选：验证trader是否存在（不阻止保存）
    exists, _ := s.repo.ValidateTraderExists(ctx, record.TraderID)
    if !exists {
        log.Printf("⚠️ 警告：trader %s 不存在，但交易历史仍会保存", record.TraderID)
    }
    
    // 2. 时间戳处理：优先使用交易所时间戳
    // 如果exchange_timestamp_ms已设置，使用它作为主时间戳
    if record.ExchangeTimestampMs != nil && *record.ExchangeTimestampMs > 0 {
        record.ExchangeTimestamp = time.Unix(*record.ExchangeTimestampMs/1000, (*record.ExchangeTimestampMs%1000)*1000000)
        record.Timestamp = *record.ExchangeTimestamp  // 主时间戳使用交易所时间
    } else if record.Timestamp.IsZero() {
        // 如果都没有设置，使用本地时间作为备用
        record.Timestamp = time.Now()
    }
    
    // 3. 检查是否已存在（通过exchange_order_id或exchange_trade_id）
    existsByExchange, _ := s.repo.ExistsByExchangeID(ctx, record.ExchangeOrderID, record.ExchangeTradeID, record.ExchangeHash)
    if existsByExchange {
        // 如果已存在，更新（避免重复）
        // 可以选择更新或跳过
        return nil
    }
    
    // 4. 如果是平仓，计算PnL（在插入前计算）
    if record.Action == "close_long" || record.Action == "close_short" {
        if record.EntryPrice != nil && record.ExitPrice != nil {
            // 计算PnL
            // ...
        }
    }
    
    // 5. 如果不存在，插入
    return s.repo.Save(ctx, record)
}

// RecordAutoTriggeredClose 实现
func (s *service) RecordAutoTriggeredClose(ctx context.Context, record *TradeRecord) error {
    // 1. 查找对应的开仓记录
    // 2. 计算PnL
    // 3. 保存平仓记录
}
```

## 7. 同步服务设计

### 7.1 Sync Service

```go
// trade_history/sync.go

package trade_history

import (
    "context"
    "time"
)

// SyncService 同步服务
type SyncService struct {
    service  Service
    interval time.Duration
}

// NewSyncService 创建同步服务
func NewSyncService(service Service, interval time.Duration) *SyncService {
    return &SyncService{
        service:  service,
        interval: interval,
    }
}

// Start 启动定期同步
func (s *SyncService) Start(ctx context.Context, traderID string, provider ExchangeFillsProvider) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // 同步交易记录
            s.service.SyncFromExchange(ctx, traderID, provider)
        }
    }
}

// SyncOnce 执行一次同步
func (s *SyncService) SyncOnce(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
    return s.service.SyncFromExchange(ctx, traderID, provider)
}
```

### 7.2 交易所适配器实现

```go
// trade_history/sync_hyperliquid.go

package trade_history

import (
    "github.com/sonirico/go-hyperliquid"
    "time"
)

// HyperliquidFillsProvider Hyperliquid成交记录提供者
type HyperliquidFillsProvider struct {
    exchange *hyperliquid.Exchange
    ctx       context.Context
    walletAddr string
}

// GetRecentFills 实现
func (p *HyperliquidFillsProvider) GetRecentFills(limit int) ([]ExchangeFill, error) {
    // 1. 调用 exchange.Info().UserFills()
    // 2. 转换为统一格式 ExchangeFill
    // 3. 返回最近的limit条记录
}

// GetFillsByTimeRange 实现
func (p *HyperliquidFillsProvider) GetFillsByTimeRange(startTime, endTime time.Time) ([]ExchangeFill, error) {
    // 1. 调用 exchange.Info().UserFillsByTime()
    // 2. 转换为统一格式 ExchangeFill
    // 3. 返回结果
}
```

## 8. 集成点设计

### 8.1 集成策略（最小化影响）

#### 方案A：事件驱动（推荐）

```go
// trade_history/events.go

package trade_history

// TradeEvent 交易事件
type TradeEvent struct {
    Type    string      // "open", "close", "auto_close"
    TraderID string
    Record  *TradeRecord
}

// EventBus 事件总线（简单实现）
type EventBus struct {
    listeners []func(event *TradeEvent)
}

// Publish 发布事件
func (bus *EventBus) Publish(event *TradeEvent) {
    for _, listener := range bus.listeners {
        go listener(event) // 异步执行，不阻塞
    }
}

// Subscribe 订阅事件
func (bus *EventBus) Subscribe(listener func(event *TradeEvent)) {
    bus.listeners = append(bus.listeners, listener)
}
```

#### 方案B：直接调用（简单）

在 `executeOpenLongWithRecord` 等函数中，API成功返回后直接调用：

```go
// trader/auto_trader.go

import "nofx/trade_history"

// 在 executeOpenLongWithRecord 中
func (at *AutoTrader) executeOpenLongWithRecord(...) error {
    // ... 现有代码 ...
    
    // API成功返回后
    if order != nil && err == nil {
        // 创建交易记录
        record := &trade_history.TradeRecord{
            TraderID:       at.id,
            Symbol:         decision.Symbol,
            Side:           "long",
            Action:         "open_long",
            Quantity:       actionRecord.Quantity,
            ExecutionPrice: actionRecord.Price,
            Leverage:       decision.Leverage,
            Timestamp:      time.Now(),
            Source:         "api",
        }
        
        // 写入数据库（如果模块已启用）
        if at.tradeHistoryService != nil {
            go at.tradeHistoryService.RecordTrade(context.Background(), record)
        }
    }
    
    return nil
}
```

### 8.2 AutoTrader 集成

```go
// trader/auto_trader.go

type AutoTrader struct {
    // ... 现有字段 ...
    
    // 新增：交易历史服务（可选）
    tradeHistoryService trade_history.Service
    tradeHistoryEnabled bool
}

// NewAutoTrader 创建时传入服务（如果启用）
func NewAutoTrader(..., tradeHistoryService trade_history.Service) (*AutoTrader, error) {
    // ...
    return &AutoTrader{
        // ...
        tradeHistoryService: tradeHistoryService,
        tradeHistoryEnabled: tradeHistoryService != nil,
    }, nil
}
```

## 9. API 接口设计

### 9.1 REST API

```go
// api/server.go

// handleGetTradeHistory 获取交易历史
func (s *Server) handleGetTradeHistory(c *gin.Context) {
    traderID := c.Query("trader_id")
    symbol := c.Query("symbol")
    action := c.Query("action")
    startTime := c.Query("start_time")
    endTime := c.Query("end_time")
    limit := c.DefaultQuery("limit", "100")
    offset := c.DefaultQuery("offset", "0")
    
    // 构建过滤器
    filter := &trade_history.TradeRecordFilter{
        TraderID: traderID,
        Symbol:   symbol,
        Action:   action,
        // ...
    }
    
    // 查询
    records, total, err := s.tradeHistoryService.GetTradeHistory(c.Request.Context(), filter)
    // ...
}

// handleGetTradeStatistics 获取交易统计
func (s *Server) handleGetTradeStatistics(c *gin.Context) {
    // ...
}

// handleSyncTradeHistory 手动触发同步
func (s *Server) handleSyncTradeHistory(c *gin.Context) {
    // ...
}
```

### 9.2 API 端点

```
GET  /api/trade-history?trader_id=xxx&symbol=xxx&limit=100&offset=0
GET  /api/trade-history/statistics?trader_id=xxx&start_time=xxx&end_time=xxx
POST /api/trade-history/sync?trader_id=xxx
```

## 10. 配置设计

### 10.1 系统配置

```go
// config/database.go 中添加

// GetTradeHistoryConfig 获取交易历史配置
func (d *Database) GetTradeHistoryConfig() (bool, int, error) {
    enabled, _ := d.GetSystemConfig("trade_history_enabled")  // "true" or "false"
    syncInterval, _ := d.GetSystemConfig("trade_history_sync_interval_minutes")  // 默认10分钟
    
    // ...
}
```

### 10.2 配置文件

```json
{
  "trade_history": {
    "enabled": true,
    "sync_interval_minutes": 10,
    "auto_sync": true
  }
}
```

## 11. 实施计划

### 阶段1：基础架构（1-2天）
1. ✅ 创建 `trade_history` 模块目录
2. ✅ 设计数据库表结构
3. ✅ 实现 Repository 层
4. ✅ 实现 Service 层
5. ✅ 添加数据库迁移脚本

### 阶段2：核心功能（2-3天）
1. ✅ 实现交易记录保存功能
2. ✅ 实现查询和统计功能
3. ✅ 实现 Hyperliquid 适配器
4. ✅ 集成到 AutoTrader（可选方式）

### 阶段3：同步功能（1-2天）
1. ✅ 实现同步服务
2. ✅ 实现定期同步
3. ✅ 实现手动同步接口

### 阶段4：API接口（1天）
1. ✅ 实现 REST API
2. ✅ 添加前端接口调用

### 阶段5：测试和优化（1-2天）
1. ✅ 单元测试
2. ✅ 集成测试
3. ✅ 性能优化
4. ✅ 文档完善

## 12. 测试策略

### 12.1 单元测试
- Repository 层测试
- Service 层测试
- 交易所适配器测试

### 12.2 集成测试
- 端到端测试（从API返回 → 数据库写入）
- 同步功能测试
- API接口测试

### 12.3 数据一致性测试
- 对比数据库和交易所数据
- 验证重复数据检测
- 验证时间戳准确性

## 13. 风险控制

### 13.1 数据一致性
- 使用唯一索引防止重复
- 定期同步校验
- 事务保证原子性

### 13.2 性能考虑
- 批量插入优化
- 索引优化查询
- 异步写入（不阻塞交易流程）

### 13.3 向后兼容
- 不影响现有JSON日志系统
- 可配置启用/禁用
- 平滑迁移方案

## 14. 监控和日志

### 14.1 日志记录
- 交易记录写入日志
- 同步操作日志
- 错误日志

### 14.2 监控指标
- 写入成功率
- 同步延迟
- 数据一致性指标


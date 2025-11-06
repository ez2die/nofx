# 交易历史时间戳记录策略分析

## 问题

记录交易历史时，时间戳应该以什么为准？

## 当前设计

### 表结构中的时间戳字段

```sql
-- 时间信息
timestamp DATETIME NOT NULL,                -- 交易时间（本地时间）
exchange_timestamp DATETIME,                  -- 交易所时间戳（如果有，毫秒转datetime）
exchange_timestamp_ms INTEGER,                -- 交易所时间戳（毫秒，用于精确查询）
```

### 当前代码中的时间戳

```go
// trader/auto_trader.go
actionRecord.Timestamp = time.Now()  // 使用本地时间
```

## 时间戳来源分析

### 1. 交易所API返回的时间戳（Hyperliquid）

**来源**：`UserFills` API 返回的 `Fill` 结构体

```go
type Fill struct {
    Time int64 `json:"time"`  // 毫秒时间戳（交易所时间）
    // ...
}
```

**特点**：
- ✅ **最准确**：反映真实交易时间
- ✅ **与交易所一致**：与交易所数据完全一致
- ✅ **毫秒精度**：精确到毫秒
- ✅ **可追溯**：可以追溯到交易所的原始记录

### 2. 本地系统时间

**来源**：`time.Now()`

**特点**：
- ⚠️ **可能不准确**：系统时间可能与交易所时间不同步
- ⚠️ **延迟问题**：API调用到写入数据库有时间差
- ⚠️ **时区问题**：需要考虑时区转换
- ✅ **可用性**：总是可用（即使交易所API不返回时间戳）

### 3. 订单API返回的时间戳

**来源**：`Order` API 返回的 `OrderStatus`

**特点**：
- ✅ **实时性**：订单提交时立即返回
- ⚠️ **可能不完整**：不是所有交易所都返回时间戳
- ⚠️ **精度问题**：可能只有秒级精度

## 时间戳策略对比

### 方案1：优先使用交易所时间戳（推荐）

**策略**：
1. **主时间戳**：使用交易所返回的时间戳（`exchange_timestamp_ms`）
2. **备用时间戳**：如果交易所时间戳不可用，使用本地时间（`timestamp`）
3. **记录来源**：在 `source` 字段中标记时间戳来源

**优点**：
- ✅ **准确性最高**：与交易所数据完全一致
- ✅ **可追溯性**：可以追溯到交易所原始记录
- ✅ **审计友好**：符合审计要求
- ✅ **同步友好**：与交易所同步时数据一致

**缺点**：
- ⚠️ 需要从交易所API获取时间戳（可能增加延迟）

**实现**：
```go
// 从交易所API获取时间戳
if exchangeTimestampMs > 0 {
    record.ExchangeTimestampMs = &exchangeTimestampMs
    record.ExchangeTimestamp = time.Unix(exchangeTimestampMs/1000, (exchangeTimestampMs%1000)*1000000)
    record.Timestamp = record.ExchangeTimestamp  // 主时间戳使用交易所时间
} else {
    record.Timestamp = time.Now()  // 备用：使用本地时间
}
```

### 方案2：使用本地时间（当前实现）

**策略**：
- 使用 `time.Now()` 作为主时间戳
- 可选：记录交易所时间戳作为参考

**优点**：
- ✅ **实现简单**：不需要等待交易所API返回
- ✅ **总是可用**：即使交易所API不返回时间戳

**缺点**：
- ❌ **不准确**：可能与交易所时间不同步
- ❌ **延迟问题**：API调用到写入有时间差
- ❌ **审计问题**：可能与交易所数据不一致

### 方案3：使用订单提交时间（折中方案）

**策略**：
- 使用订单提交时的本地时间
- 记录交易所时间戳作为参考

**优点**：
- ✅ **相对准确**：接近真实交易时间
- ✅ **实现简单**：不需要等待交易所API返回

**缺点**：
- ⚠️ **仍然不准确**：本地时间可能与交易所时间不同步
- ⚠️ **延迟问题**：订单提交到成交有时间差

## 推荐方案：优先使用交易所时间戳

### 实施策略

#### 1. 实时写入时（API返回后）

```go
// 在 executeOpenLongWithRecord 等函数中
func (at *AutoTrader) executeOpenLongWithRecord(...) error {
    // ... 执行订单 ...
    
    // 获取订单返回信息
    order, err := at.trader.OpenLong(...)
    
    // 创建交易记录
    record := &trade_history.TradeRecord{
        // ... 其他字段 ...
        Timestamp: time.Now(),  // 初始值：本地时间
    }
    
    // 优先使用交易所时间戳（如果可用）
    if exchangeTimestampMs > 0 {
        // 从交易所API获取时间戳
        record.ExchangeTimestampMs = &exchangeTimestampMs
        record.ExchangeTimestamp = time.Unix(exchangeTimestampMs/1000, (exchangeTimestampMs%1000)*1000000)
        record.Timestamp = record.ExchangeTimestamp  // 主时间戳使用交易所时间
    }
    
    // 写入数据库
    at.tradeHistoryService.RecordTrade(ctx, record)
}
```

#### 2. 从UserFills获取时间戳

```go
// 在 getFillPriceFromUserFills 中，同时获取时间戳
func (t *HyperliquidTrader) getFillPriceAndTimestamp(symbol string) (float64, int64, error) {
    fills, err := t.exchange.Info().UserFills(t.ctx, t.walletAddr)
    if err != nil {
        return 0, 0, err
    }
    
    // 查找最近的匹配成交
    for i := len(fills) - 1; i >= 0; i-- {
        fill := fills[i]
        if fill.Coin == coin && fill.ClosedPnl != nil {
            if fill.Px != nil {
                price, _ := strconv.ParseFloat(*fill.Px, 64)
                timestampMs := fill.Time  // 交易所时间戳（毫秒）
                return price, timestampMs, nil
            }
        }
    }
    
    return 0, 0, fmt.Errorf("未找到成交记录")
}
```

#### 3. 同步时（从交易所API同步）

```go
// 在 SyncFromExchange 中
func (s *service) SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
    fills, err := provider.GetRecentFills(100)
    if err != nil {
        return err
    }
    
    for _, fill := range fills {
        // 使用交易所时间戳
        record := &TradeRecord{
            TraderID: traderID,
            Symbol: fill.Symbol,
            // ... 其他字段 ...
            ExchangeTimestampMs: &fill.TimestampMs,
            ExchangeTimestamp: fill.Timestamp,
            Timestamp: fill.Timestamp,  // 主时间戳使用交易所时间
            Source: "sync",
        }
        
        // 保存或更新
        s.RecordTrade(ctx, record)
    }
}
```

### 表结构设计（更新）

```sql
CREATE TABLE IF NOT EXISTS trade_history (
    -- ... 其他字段 ...
    
    -- 时间信息（优先级顺序）
    timestamp DATETIME NOT NULL,                -- 主时间戳（优先使用交易所时间，否则使用本地时间）
    exchange_timestamp DATETIME,                -- 交易所时间戳（如果有）
    exchange_timestamp_ms INTEGER,              -- 交易所时间戳（毫秒，用于精确查询和排序）
    
    -- ... 其他字段 ...
);
```

### 时间戳字段说明

| 字段 | 类型 | 说明 | 优先级 |
|------|------|------|--------|
| `timestamp` | DATETIME | 主时间戳（交易时间） | 1. 交易所时间戳（如果可用）<br>2. 本地时间（备用） |
| `exchange_timestamp` | DATETIME | 交易所时间戳（如果有） | 从交易所API获取 |
| `exchange_timestamp_ms` | INTEGER | 交易所时间戳（毫秒） | 从交易所API获取，用于精确查询 |

### 查询策略

```go
// 查询时，优先使用exchange_timestamp_ms（如果可用）
func (r *repository) FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error) {
    // 排序时，优先使用exchange_timestamp_ms
    orderBy := "timestamp DESC"
    if filter.OrderBy != "" {
        orderBy = filter.OrderBy
    }
    
    // 如果有exchange_timestamp_ms，使用它排序（更准确）
    // 如果没有，使用timestamp
    query := `
        SELECT 
            id, trader_id, symbol, side, action, quantity, leverage,
            entry_price, exit_price, execution_price, pnl, pnl_pct,
            order_id, exchange_order_id, exchange_trade_id, exchange_hash,
            fee, fee_token, is_auto_triggered, was_stop_loss, was_take_profit,
            cycle_number, source,
            COALESCE(exchange_timestamp, timestamp) as timestamp,  -- 优先使用交易所时间戳
            exchange_timestamp, exchange_timestamp_ms,
            created_at, updated_at
        FROM trade_history
        WHERE ...
        ORDER BY COALESCE(exchange_timestamp_ms, timestamp) DESC
    `
}
```

## 实施建议

### 1. 实时写入时

**优先级**：
1. **交易所时间戳**（从 `UserFills` API 获取）
2. **订单API返回时间戳**（如果有）
3. **本地时间**（备用）

**实现**：
```go
// 在 executeOpenLongWithRecord 中
func (at *AutoTrader) executeOpenLongWithRecord(...) error {
    // ... 执行订单 ...
    
    // 1. 尝试从UserFills获取时间戳（最准确）
    time.Sleep(2 * time.Second)  // 等待订单成交
    _, timestampMs, err := at.trader.GetFillPriceAndTimestamp(decision.Symbol)
    
    record := &trade_history.TradeRecord{
        // ... 其他字段 ...
    }
    
    if err == nil && timestampMs > 0 {
        // 使用交易所时间戳
        record.ExchangeTimestampMs = &timestampMs
        record.ExchangeTimestamp = time.Unix(timestampMs/1000, (timestampMs%1000)*1000000)
        record.Timestamp = record.ExchangeTimestamp
    } else {
        // 备用：使用本地时间
        record.Timestamp = time.Now()
    }
    
    // 写入数据库
    at.tradeHistoryService.RecordTrade(ctx, record)
}
```

### 2. 同步时

**优先级**：
1. **交易所时间戳**（从 `UserFills` API 获取）

**实现**：
```go
// 在 SyncFromExchange 中
func (s *service) SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
    fills, err := provider.GetRecentFills(100)
    if err != nil {
        return err
    }
    
    for _, fill := range fills {
        record := &TradeRecord{
            // ... 其他字段 ...
            ExchangeTimestampMs: &fill.TimestampMs,
            ExchangeTimestamp: fill.Timestamp,
            Timestamp: fill.Timestamp,  // 主时间戳使用交易所时间
            Source: "sync",
        }
        
        s.RecordTrade(ctx, record)
    }
}
```

### 3. 自动触发平仓时

**优先级**：
1. **交易所时间戳**（从 `UserFills` API 获取）
2. **本地时间**（备用）

**实现**：
```go
// 在 runCycle 中检测到自动触发平仓时
closeAction := logger.DecisionAction{
    // ... 其他字段 ...
    Timestamp: time.Now(),  // 初始值
}

// 尝试从UserFills获取真实时间戳
_, timestampMs, err := at.trader.GetFillPriceAndTimestamp(lastPos.Symbol)
if err == nil && timestampMs > 0 {
    closeAction.Timestamp = time.Unix(timestampMs/1000, (timestampMs%1000)*1000000)
}

// 创建交易记录
record := &trade_history.TradeRecord{
    // ... 其他字段 ...
    ExchangeTimestampMs: &timestampMs,
    ExchangeTimestamp: closeAction.Timestamp,
    Timestamp: closeAction.Timestamp,  // 主时间戳使用交易所时间
}
```

## 最终推荐

### 推荐方案：优先使用交易所时间戳

**主时间戳**：`timestamp` 字段
- **优先级1**：交易所时间戳（从 `UserFills` API 获取）
- **优先级2**：本地时间（备用）

**辅助时间戳**：
- `exchange_timestamp`：交易所时间戳（DATETIME）
- `exchange_timestamp_ms`：交易所时间戳（毫秒，用于精确查询）

**优点**：
1. ✅ **准确性最高**：与交易所数据完全一致
2. ✅ **可追溯性**：可以追溯到交易所原始记录
3. ✅ **审计友好**：符合审计要求
4. ✅ **同步友好**：与交易所同步时数据一致
5. ✅ **向后兼容**：如果交易所时间戳不可用，使用本地时间作为备用

**实施优先级**：
1. **实时写入**：尝试从 `UserFills` API 获取时间戳
2. **同步写入**：使用 `UserFills` API 返回的时间戳
3. **自动触发**：尝试从 `UserFills` API 获取时间戳


# Cycle 中价格数据获取机制详解

## 调用链概览

```
runCycle() 
  → buildTradingContext()
    → decision.FetchTradingContext()
      → fetchMarketDataForContext()
        → market.Get(symbol)
          → WSMonitorCli.GetCurrentKlines(symbol, "3m"/"4h")
```

## 详细流程

### 1. Cycle 启动 (`runCycle()`)

位置：`trader/auto_trader.go:259`

```go
func (at *AutoTrader) runCycle() error {
    // ...
    ctx, err := at.buildTradingContext()
    // ...
}
```

### 2. 构建交易上下文 (`buildTradingContext()`)

位置：`trader/auto_trader.go:551`

```go
func (at *AutoTrader) buildTradingContext() (*decision.Context, error) {
    // 1. 获取账户信息（API调用）
    balance, err := at.trader.GetBalance()
    
    // 2. 获取持仓信息（API调用）
    positions, err := at.trader.GetPositions()
    
    // 3. 获取候选币种列表
    candidateCoins := at.getCandidateCoins()
    
    // 4. 构建 Context 并获取市场数据
    ctx := &decision.Context{
        // ...
    }
    
    // 调用 FetchTradingContext 获取市场数据
    return decision.FetchTradingContext(ctx, ...)
}
```

### 3. 获取市场数据 (`fetchMarketDataForContext()`)

位置：`decision/engine.go:144`

```go
func fetchMarketDataForContext(ctx *Context) error {
    // 收集需要获取数据的币种
    symbolSet := make(map[string]bool)
    
    // 1. 优先获取持仓币种的数据（必须）
    for _, pos := range ctx.Positions {
        symbolSet[pos.Symbol] = true
    }
    
    // 2. 获取候选币种的数据（根据账户状态动态调整）
    maxCandidates := calculateMaxCandidates(ctx)
    for i, coin := range ctx.CandidateCoins {
        if i >= maxCandidates {
            break
        }
        symbolSet[coin.Symbol] = true
    }
    
    // 并发获取市场数据
    for symbol := range symbolSet {
        data, err := market.Get(symbol)  // ⭐ 关键调用
        // ...
    }
}
```

### 4. 获取市场数据 (`market.Get()`)

位置：`market/data.go:16`

```go
func Get(symbol string) (*Data, error) {
    // 标准化symbol
    symbol = Normalize(symbol)
    
    // ⭐ 获取3分钟K线数据（从缓存或API）
    klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m")
    
    // ⭐ 获取4小时K线数据（从缓存或API）
    klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h")
    
    // 计算技术指标（基于K线数据）
    currentPrice := klines3m[len(klines3m)-1].Close
    currentEMA20 := calculateEMA(klines3m, 20)
    currentMACD := calculateMACD(klines3m)
    currentRSI7 := calculateRSI(klines3m, 7)
    
    // ⭐ 获取OI数据（直接API调用）
    apiClient := GetMarketDataClient()
    oiData, err := apiClient.GetOpenInterest(symbol)
    
    // ⭐ 获取Funding Rate（直接API调用）
    fundingRate, err := apiClient.GetFundingRate(symbol)
    
    // 计算日内序列和长期数据
    intradayData := calculateIntradaySeries(klines3m)
    longerTermData := calculateLongerTermData(klines4h)
    
    return &Data{...}
}
```

### 5. 获取K线数据 (`GetCurrentKlines()`)

位置：`market/monitor.go:300`

```go
func (m *WSMonitor) GetCurrentKlines(symbol string, _time string) ([]Kline, error) {
    symbol = strings.ToUpper(symbol)
    klineDataMap := m.getKlineDataMap(_time)      // 获取缓存Map
    updateTimeMap := m.getKlineUpdateTimeMap(_time) // 获取更新时间Map
    
    // ⭐ 步骤1：检查缓存是否存在
    value, exists := klineDataMap.Load(symbol)
    if exists {
        // ⭐ 步骤2：检查缓存时效性
        lastUpdate, updateExists := updateTimeMap.Load(symbol)
        
        // 设置过期时间
        var maxAge time.Duration
        if _time == "3m" {
            maxAge = 5 * time.Minute  // 3分钟K线：5分钟过期
        } else if _time == "4h" {
            maxAge = 10 * time.Minute // 4小时K线：10分钟过期
        }
        
        if updateExists {
            lastUpdateTime := lastUpdate.(time.Time)
            if time.Since(lastUpdateTime) < maxAge {
                // ⭐ 步骤3：缓存有效，直接返回缓存数据
                return value.([]Kline), nil
            } else {
                // 缓存过期，记录日志
                log.Printf("⚠️  %s 的 %s K线数据已过期，重新从API获取", symbol, _time)
            }
        }
    }
    
    // ⭐ 步骤4：缓存不存在或已过期，调用API获取最新数据
    apiClient := GetMarketDataClient()
    klines, err := apiClient.GetKlines(symbol, _time, 100)
    if err != nil {
        // API失败，尝试返回缓存数据（即使可能过期）
        if exists {
            log.Printf("⚠️  从API获取失败，使用可能过期的缓存数据")
            return value.([]Kline), nil
        }
        return nil, fmt.Errorf("获取%v分钟K线失败: %v", _time, err)
    }
    
    // ⭐ 步骤5：更新缓存
    klineDataMap.Store(symbol, klines)
    updateTimeMap.Store(symbol, time.Now())
    log.Printf("✓ 成功从API获取并更新 %s 的 %s K线数据（%d 条）", symbol, _time, len(klines))
    
    // ⭐ 步骤6：尝试订阅WebSocket（仅Binance）
    apiClientForWS := GetMarketDataClient()
    if apiClientForWS.GetDataSourceName() == "binance" {
        // 订阅WebSocket流（如果尚未订阅）
        subStr := m.subscribeSymbol(symbol, _time)
        m.combinedClient.subscribeStreams(subStr)
    }
    
    return klines, nil
}
```

## 总结：价格数据获取机制

### 数据来源

1. **K线数据（3m/4h）**：**优先读缓存，缓存过期或不存在时调用API**
   - 缓存位置：`WSMonitor.klineDataMap3m` 和 `klineDataMap4h` (sync.Map)
   - 缓存时效性：3分钟K线5分钟过期，4小时K线10分钟过期
   - API调用：`apiClient.GetKlines(symbol, _time, 100)`
   - 数据源：根据 `config.json` 中的 `market_data_source` 配置（Binance 或 Hyperliquid）

2. **OI数据（Open Interest）**：**每次直接调用API**
   - API调用：`apiClient.GetOpenInterest(symbol)`
   - 无缓存机制

3. **Funding Rate**：**每次直接调用API**
   - API调用：`apiClient.GetFundingRate(symbol)`
   - 无缓存机制

### 缓存写入机制

#### 1. 初始化时写入（应用启动时）

位置：`market/monitor.go:80` - `initializeHistoricalData()`

```go
func (m *WSMonitor) initializeHistoricalData() error {
    // 为每个symbol获取历史K线数据
    klines, err := apiClient.GetKlines(s, "3m", 100)
    if len(klines) > 0 {
        m.klineDataMap3m.Store(s, klines)        // 写入缓存
        m.klineUpdateTime3m.Store(s, time.Now()) // 记录更新时间
    }
    
    klines4h, err := apiClient.GetKlines(s, "4h", 100)
    if len(klines4h) > 0 {
        m.klineDataMap4h.Store(s, klines4h)      // 写入缓存
        m.klineUpdateTime4h.Store(s, time.Now()) // 记录更新时间
    }
}
```

#### 2. WebSocket更新时写入（仅Binance）

位置：`market/monitor.go:235` - `processKlineUpdate()`

```go
func (m *WSMonitor) processKlineUpdate(symbol string, wsData KlineWSData, _time string) {
    // 转换WebSocket数据为Kline结构
    kline := Kline{...}
    
    // 更新K线数据
    klines := // 从缓存获取现有数据
    if len(klines) > 0 && klines[len(klines)-1].OpenTime == kline.OpenTime {
        // 更新当前K线
        klines[len(klines)-1] = kline
    } else {
        // 添加新K线
        klines = append(klines, kline)
        if len(klines) > 100 {
            klines = klines[1:] // 保持长度
        }
    }
    
    klineDataMap.Store(symbol, klines)  // 写入缓存
    updateTimeMap.Store(symbol, time.Now()) // 更新最后更新时间
}
```

#### 3. 缓存过期或不存在时写入（每次cycle）

位置：`market/monitor.go:300` - `GetCurrentKlines()`

```go
// 当缓存不存在或过期时
klines, err := apiClient.GetKlines(symbol, _time, 100)
klineDataMap.Store(symbol, klines)      // 写入缓存
updateTimeMap.Store(symbol, time.Now()) // 记录更新时间
```

### 缓存更新时机总结

| 时机 | 触发条件 | 数据源 | 更新频率 |
|------|---------|--------|---------|
| **应用启动** | `initializeHistoricalData()` | API | 一次性 |
| **WebSocket推送** | Binance WebSocket数据到达 | WebSocket | 实时（每3分钟或4小时） |
| **缓存过期** | 数据超过有效期（5分钟/10分钟） | API | 按需刷新 |
| **缓存不存在** | 首次请求某个symbol | API | 按需刷新 |

### 当前配置（Hyperliquid）的行为

由于当前配置使用 `market_data_source: "hyperliquid"`：

1. **WebSocket不工作**：Hyperliquid不使用Binance WebSocket，所以不会通过WebSocket更新缓存
2. **完全依赖API**：所有数据都通过REST API获取
3. **缓存刷新机制**：
   - 首次请求：从API获取并缓存
   - 缓存有效期内（5-10分钟）：直接返回缓存
   - 缓存过期：重新从API获取并更新缓存

### 数据获取流程图

```
Cycle 开始
  ↓
buildTradingContext()
  ↓
fetchMarketDataForContext()
  ↓
market.Get(symbol)  ← 对每个symbol调用
  ↓
GetCurrentKlines(symbol, "3m")
  ↓
检查缓存是否存在？
  ├─ 否 → 调用API获取 → 写入缓存 → 返回数据
  └─ 是 → 检查缓存是否过期？
      ├─ 否 → 直接返回缓存数据 ⭐（读缓存）
      └─ 是 → 调用API获取 → 更新缓存 → 返回数据 ⭐（API调用）
```

## 关键结论

1. **价格数据是读缓存的**：K线数据优先从缓存读取，只有在缓存不存在或过期时才调用API
2. **缓存写入时机**：
   - 应用启动时初始化
   - WebSocket推送时更新（仅Binance）
   - 缓存过期时重新获取并更新
3. **当前问题**：使用Hyperliquid时，WebSocket不工作，只能依赖缓存过期机制（5-10分钟）来刷新数据
4. **修复后的改进**：添加了时效性检查，确保数据过期时自动从API刷新


# Trade Analytics 指标计算触发流程

## 1. 系统初始化流程

### 1.1 服务器启动时

```go
// api/server.go::setupRoutes()
func (s *Server) setupRoutes() {
    // ...
    s.setupTradeAnalyticsRoutes(api)  // 初始化交易分析路由
}
```

### 1.2 交易分析模块初始化

```go
// api/server.go::setupTradeAnalyticsRoutes()
func (s *Server) setupTradeAnalyticsRoutes(router *gin.RouterGroup) {
    // 1. 获取数据库连接
    db, err := s.database.GetDB()
    
    // 2. 创建 trade_history repository（底层数据访问）
    tradeHistoryRepo := trade_history.NewRepository(db)
    
    // 3. 创建 trade_analytics repository（扩展数据访问层）
    analyticsRepo := trade_analytics.NewRepository(db, tradeHistoryRepo)
    
    // 4. 创建 analyzer（核心分析逻辑）
    analyticsAnalyzer := trade_analytics.NewAnalyzer(analyticsRepo)
    
    // 5. 创建 pairMatcher（交易配对逻辑）
    analyticsPairMatcher := trade_analytics.NewPairMatcher(analyticsRepo)
    
    // 6. 创建 service（服务层，协调各组件）
    analyticsService := trade_analytics.NewService(
        analyticsRepo, 
        analyticsAnalyzer, 
        analyticsPairMatcher
    )
    
    // 7. 创建 API handler 并注册路由
    analyticsHandler := trade_analytics.NewAPIHandler(analyticsService)
    analyticsHandler.RegisterRoutes(router)
}
```

**初始化顺序**：
```
Database → trade_history.Repository → trade_analytics.Repository 
→ Analyzer + PairMatcher → Service → APIHandler → Routes
```

---

## 2. HTTP 请求触发流程

### 2.1 请求入口

用户发起 HTTP 请求，例如：
```bash
GET /api/trade-analytics?trader_id=xxx&symbol=BTCUSDT&start_time=2025-01-01T00:00:00Z
```

### 2.2 路由匹配

```go
// trade_analytics/api.go::RegisterRoutes()
analytics.GET("", h.handleGetAnalytics)  // 完整分析
analytics.GET("/overview", h.handleGetOverview)  // 概览统计
analytics.GET("/pnl", h.handleGetPnLStats)  // 盈亏统计
// ... 其他端点
```

---

## 3. 完整分析流程（以 GET /api/trade-analytics 为例）

### 3.1 API 层处理

```go
// trade_analytics/api.go::handleGetAnalytics()
func (h *APIHandler) handleGetAnalytics(c *gin.Context) {
    // 1. 解析查询参数为过滤器
    filter, err := parseAnalyticsFilter(c)
    // filter 包含: trader_id, symbol, side, action, start_time, end_time, etc.
    
    // 2. 调用 Service 层
    analytics, err := h.service.GetAnalytics(c.Request.Context(), filter)
    
    // 3. 返回 JSON 响应
    c.JSON(http.StatusOK, analytics)
}
```

**parseAnalyticsFilter 解析的参数**：
- `trader_id` (必需)
- `symbol` (可选)
- `side` (可选: long/short)
- `action` (可选: open_long/close_long/etc.)
- `start_time` (可选: RFC3339格式)
- `end_time` (可选: RFC3339格式)
- `group_by` (可选: day/week/month)
- `include_pairs` (可选: true/false)

### 3.2 Service 层处理

```go
// trade_analytics/service.go::GetAnalytics()
func (s *service) GetAnalytics(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error) {
    // 1. 调用 Analyzer 执行完整分析
    analytics, err := s.analyzer.Analyze(ctx, filter)
    
    // 2. 如果请求包含配对分析，添加配对统计
    if filter.IncludePairs {
        pairStats, err := s.pairMatcher.GetPairStatistics(ctx, filter)
        analytics.PairStats = pairStats
    }
    
    return analytics, nil
}
```

### 3.3 Analyzer 层处理（核心分析逻辑）

```go
// trade_analytics/analyzer.go::Analyze()
func (a *analyzer) Analyze(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error) {
    analytics := &TradeAnalytics{}
    
    // 按顺序计算各项指标：
    
    // 1. 基础统计（Repository层，SQL聚合查询）
    overview, err := a.repo.GetOverviewStats(ctx, filter)
    
    // 2. 盈亏统计（Repository层，SQL聚合查询）
    pnlStats, err := a.repo.GetPnLStats(ctx, filter)
    
    // 3. 胜率统计（Repository层，SQL聚合查询）
    winRateStats, err := a.repo.GetWinRateStats(ctx, filter)
    
    // 4. 费用统计（Repository层，SQL聚合查询）
    feeStats, err := a.repo.GetFeeStats(ctx, filter)
    
    // 5. 方向统计（Repository层，SQL聚合查询 + 方向偏好计算）
    directionStats, err := a.repo.GetDirectionStats(ctx, filter)
    
    // 6. 币种统计（Repository层，SQL GROUP BY查询）
    symbolStats, err := a.repo.GetSymbolStats(ctx, filter)
    
    // 7. 时间序列统计（Repository层，SQL GROUP BY查询）
    timeSeriesStats := &TimeSeriesStatistics{}
    dailyStats, err := a.repo.GetDailyStats(ctx, filter)
    weeklyStats, err := a.repo.GetWeeklyStats(ctx, filter)
    monthlyStats, err := a.repo.GetMonthlyStats(ctx, filter)
    
    // 8. 风险指标（Analyzer层，复杂计算）
    riskMetrics, err := a.CalculateRiskMetrics(ctx, filter)
    
    // 9. 连续统计（Analyzer层，复杂计算）
    streakStats, err := a.CalculateStreakStats(ctx, filter)
    
    // 10. 交易频率统计（Repository层，内存计算）
    frequencyStats, err := a.repo.GetFrequencyStats(ctx, filter)
    
    // 11. 交易类型统计（Repository层，SQL聚合查询）
    actionStats, err := a.repo.GetActionStats(ctx, filter)
    
    // 12. 趋势分析（Repository层，内存计算）
    trendAnalysis, err := a.repo.GetTrendAnalysis(ctx, filter)
    
    return analytics, nil
}
```

---

## 4. 单项指标查询流程（以 GET /api/trade-analytics/pnl 为例）

### 4.1 直接路径（不经过 Analyzer）

```
HTTP Request 
  → API Handler (handleGetPnLStats)
    → Service (GetPnLStats)
      → Repository (GetPnLStats)
        → SQL Query (聚合查询)
          → 返回结果
```

**代码路径**：
```go
// api.go
handleGetPnLStats() 
  → service.GetPnLStats() 
    → repo.GetPnLStats() 
      → SQL: SELECT SUM(pnl), SUM(CASE WHEN pnl > 0...), etc.
```

---

## 5. 计算方式分类

### 5.1 SQL 聚合查询（Repository层）

**特点**：直接在数据库层面计算，性能高

**指标类型**：
- 概览统计（总交易数、开仓/平仓数）
- 盈亏统计（总盈亏、总盈利、总亏损）
- 胜率统计（盈利/亏损交易数）
- 费用统计（总费用、开仓/平仓费用）
- 方向统计（做多/做空统计）
- 币种统计（按币种分组）
- 时间序列统计（按日/周/月分组）
- 交易类型统计（开仓/平仓统计）

**示例**：
```go
// repository.go::GetPnLStats()
query := `
    SELECT 
        COALESCE(SUM(pnl), 0) as total_pnl,
        COALESCE(SUM(CASE WHEN pnl > 0 THEN pnl ELSE 0 END), 0) as total_profit,
        COALESCE(SUM(CASE WHEN pnl < 0 THEN ABS(pnl) ELSE 0 END), 0) as total_loss,
        ...
    FROM trade_history
    WHERE trader_id = ? AND ...
`
```

### 5.2 内存计算（Repository层）

**特点**：从数据库获取原始数据，在内存中计算

**指标类型**：
- 交易频率统计（日均交易数、平均间隔、最活跃时段/日期）
- 趋势分析（累计盈亏曲线、盈亏分布、交易频率趋势）

**示例**：
```go
// repository.go::GetFrequencyStats()
// 1. 获取所有交易记录
records, err := r.GetRecordsByFilter(ctx, filter)

// 2. 在内存中计算
stats.DailyAverageTrades = calculateDailyAverageTrades(records, filter)
stats.AvgTradeInterval = calculateAvgTradeInterval(records)
stats.MostActiveHours = calculateMostActiveHours(records)
stats.MostActiveDays = calculateMostActiveDays(records)
```

### 5.3 复杂算法计算（Analyzer层）

**特点**：需要复杂算法，涉及时间序列分析

**指标类型**：
- 风险指标（最大回撤、波动率、夏普比率）
- 连续统计（最长连胜/连亏、当前连续）

**示例**：
```go
// analyzer.go::CalculateRiskMetrics()
// 1. 获取所有有PnL的记录
records, err := a.repo.GetRecordsByFilter(ctx, filter)

// 2. 计算累计盈亏曲线
cumulativePnL := make([]float64, len(records))
var runningTotal float64
for i, rec := range records {
    runningTotal += *rec.PnL
    cumulativePnL[i] = runningTotal
}

// 3. 计算最大回撤
maxDrawdown, maxDrawdownPercent, drawdownHistory := calculateDrawdown(cumulativePnL, records)

// 4. 计算波动率和夏普比率
volatility := calculateVolatility(records)
sharpeRatio := calculateSharpeRatio(records, volatility)
```

---

## 6. 数据流向图

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request                             │
│  GET /api/trade-analytics?trader_id=xxx&symbol=BTCUSDT    │
└──────────────────────┬────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              API Layer (api.go)                             │
│  • parseAnalyticsFilter() - 解析查询参数                    │
│  • handleGetAnalytics() - 处理请求                          │
└──────────────────────┬────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│           Service Layer (service.go)                        │
│  • GetAnalytics() - 协调各组件                              │
│  • 决定是否包含配对分析                                     │
└──────────────────────┬────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│         Analyzer Layer (analyzer.go)                        │
│  • Analyze() - 执行完整分析                                 │
│  • 按顺序调用各项指标计算                                   │
└──────┬──────────────────────────────┬──────────────────────┘
       │                              │
       ▼                              ▼
┌──────────────────┐        ┌──────────────────────────────┐
│  Repository      │        │  Analyzer 复杂计算            │
│  (SQL聚合查询)    │        │  (内存算法计算)               │
│                  │        │                              │
│  • GetOverview   │        │  • CalculateRiskMetrics     │
│  • GetPnLStats   │        │  • CalculateStreakStats     │
│  • GetWinRate    │        │                              │
│  • GetFeeStats   │        └──────────────────────────────┘
│  • GetDirection  │
│  • GetSymbol     │
│  • GetDaily/     │
│    Weekly/Monthly│
│  • GetFrequency  │
│  • GetAction     │
│  • GetTrend      │
└────────┬─────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│              Database (SQLite)                              │
│  • trade_history 表                                         │
│  • SQL 查询和聚合                                           │
└─────────────────────────────────────────────────────────────┘
```

---

## 7. 计算触发时机

### 7.1 实时计算（当前实现）

**特点**：每次请求都实时计算，无缓存

**触发时机**：
- HTTP 请求到达时
- 立即执行所有计算
- 返回结果

**优点**：
- 数据实时准确
- 支持任意过滤条件
- 实现简单

**缺点**：
- 大数据量时性能可能较慢
- 重复计算相同数据

### 7.2 计算顺序

在 `Analyzer.Analyze()` 中，指标按以下顺序计算：

1. **基础统计**（概览、盈亏、胜率、费用）- 快速SQL查询
2. **方向统计** - SQL查询 + 方向偏好计算
3. **币种统计** - SQL GROUP BY查询
4. **时间序列统计** - SQL GROUP BY查询（按日/周/月）
5. **风险指标** - 复杂内存计算（需要所有PnL记录）
6. **连续统计** - 复杂内存计算（需要所有PnL记录）
7. **交易频率统计** - 内存计算（需要所有记录）
8. **交易类型统计** - SQL查询
9. **趋势分析** - 内存计算（需要所有记录）

**注意**：如果某个指标计算失败（err != nil），会跳过该指标，不影响其他指标的计算。

---

## 8. 性能考虑

### 8.1 数据库查询优化

- 使用索引：`trader_id`, `symbol`, `timestamp`, `action`, `side`
- SQL聚合查询：在数据库层面计算，减少数据传输
- WHERE 子句过滤：根据 filter 条件过滤数据

### 8.2 内存计算优化

- 按时间排序：使用 `ORDER BY timestamp ASC`
- 批量处理：一次性获取所有记录，在内存中计算
- 算法优化：使用高效的排序和统计算法

### 8.3 潜在优化点

- **缓存**：对常用查询结果进行缓存
- **异步计算**：对复杂指标（如风险指标）进行异步计算
- **分页**：大数据量时支持分页查询
- **物化视图**：对常用统计创建物化视图

---

## 9. 示例：完整请求流程

### 9.1 请求示例

```bash
GET /api/trade-analytics?trader_id=trader_123&symbol=BTCUSDT&start_time=2025-01-01T00:00:00Z&include_pairs=true
```

### 9.2 执行步骤

1. **API层**：解析参数，创建 `AnalyticsFilter`
2. **Service层**：调用 `GetAnalytics()`
3. **Analyzer层**：执行 `Analyze()`
   - 调用 `repo.GetOverviewStats()` → SQL查询
   - 调用 `repo.GetPnLStats()` → SQL查询
   - 调用 `repo.GetWinRateStats()` → SQL查询
   - ...（其他指标）
   - 调用 `analyzer.CalculateRiskMetrics()` → 获取记录 + 内存计算
   - 调用 `analyzer.CalculateStreakStats()` → 获取记录 + 内存计算
4. **Service层**：如果 `include_pairs=true`，调用 `pairMatcher.GetPairStatistics()`
5. **API层**：将结果序列化为 JSON 返回

### 9.3 返回结果

```json
{
  "overview": { ... },
  "pnl_stats": { ... },
  "win_rate_stats": { ... },
  "fee_stats": { ... },
  "direction_stats": { ... },
  "risk_metrics": { ... },
  "streak_stats": { ... },
  "symbol_stats": { ... },
  "time_series_stats": { ... },
  "frequency_stats": { ... },
  "action_stats": { ... },
  "trend_analysis": { ... },
  "pair_stats": { ... }
}
```

---

## 10. 总结

**触发方式**：HTTP RESTful API 请求

**计算方式**：
- **SQL聚合**：基础统计、盈亏、胜率、费用、方向、币种、时间序列、交易类型
- **内存计算**：交易频率、趋势分析
- **复杂算法**：风险指标、连续统计

**执行顺序**：按 Analyzer.Analyze() 中定义的顺序依次计算

**数据源**：`trade_history` 数据库表

**实时性**：每次请求实时计算，无缓存


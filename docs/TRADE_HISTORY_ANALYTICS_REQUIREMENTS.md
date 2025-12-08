# Trade History Analytics & Performance Analysis Module
# 交易历史统计与绩效分析模块需求文档

## 1. 概述 (Overview)

### 1.1 目标
开发一个全新的、基于 `trade_history` 数据库的统计和绩效分析模块，提供全面的交易数据分析和绩效评估功能。该模块将独立于现有的 `GetStatistics` 方法，提供更丰富、更深入的分析能力。

### 1.2 设计原则
- **数据驱动**：完全基于 `trade_history` 表中的真实交易记录
- **多维度分析**：支持时间、币种、方向、交易类型等多个维度的统计
- **实时性**：支持实时查询和历史数据回溯
- **可扩展性**：模块化设计，便于后续添加新的分析指标
- **性能优化**：针对大数据量场景进行查询优化

### 1.3 与现有系统的关系
- **独立模块**：不修改现有的 `GetStatistics` 方法
- **数据源**：使用 `trade_history` 表作为唯一数据源
- **补充功能**：提供比现有统计更丰富的分析能力

## 2. 功能需求 (Functional Requirements)

### 2.1 核心统计指标

#### 2.1.1 交易概览统计
- **总交易笔数**：所有交易记录（开仓+平仓）的总数
- **开仓笔数**：`action IN ('open_long', 'open_short')` 的记录数
- **平仓笔数**：`action IN ('close_long', 'close_short')` 的记录数
- **已完成交易数**：有 PnL 的平仓记录数（完整交易对）
- **未平仓交易数**：开仓但未平仓的记录数

#### 2.1.2 盈亏统计
- **总盈亏 (Total PnL)**：所有平仓记录的 PnL 总和
- **总盈利 (Total Profit)**：盈利交易（PnL > 0）的总和
- **总亏损 (Total Loss)**：亏损交易（PnL < 0）的总和（绝对值）
- **净盈亏 (Net PnL)**：总盈亏 - 总费用
- **平均盈亏 (Average PnL)**：总盈亏 / 已完成交易数
- **最大单笔盈利 (Max Win)**：单笔最大盈利
- **最大单笔亏损 (Max Loss)**：单笔最大亏损（绝对值）
- **盈亏比 (Profit Factor)**：总盈利 / 总亏损

#### 2.1.3 胜率统计
- **胜率 (Win Rate)**：盈利交易数 / 已完成交易数 × 100%
- **盈利交易数 (Winning Trades)**：PnL > 0 的交易数
- **亏损交易数 (Losing Trades)**：PnL < 0 的交易数
- **持平交易数 (Break-even Trades)**：PnL = 0 的交易数
- **平均盈利 (Average Win)**：总盈利 / 盈利交易数
- **平均亏损 (Average Loss)**：总亏损 / 亏损交易数（绝对值）

#### 2.1.4 费用统计
- **总费用 (Total Fees)**：所有交易记录的费用总和（开仓+平仓）
- **开仓费用 (Open Fees)**：开仓记录的费用总和
- **平仓费用 (Close Fees)**：平仓记录的费用总和
- **平均费用 (Average Fee)**：总费用 / 总交易笔数
- **费用占比 (Fee Ratio)**：总费用 / 总盈亏（绝对值）× 100%

#### 2.1.5 方向统计
- **做多统计**：
  - 做多交易数（`side = 'long'`）
  - 做多总盈亏
  - 做多胜率
  - 做多平均盈亏
- **做空统计**：
  - 做空交易数（`side = 'short'`）
  - 做空总盈亏
  - 做空胜率
  - 做空平均盈亏

### 2.2 高级分析指标

#### 2.2.1 风险指标
- **最大回撤 (Maximum Drawdown)**：从峰值到谷值的最大跌幅
- **最大回撤百分比 (Max Drawdown %)**：最大回撤 / 峰值 × 100%
- **回撤持续时间 (Drawdown Duration)**：回撤持续的时间段
- **恢复时间 (Recovery Time)**：从回撤恢复到峰值的时间
- **波动率 (Volatility)**：PnL 的标准差
- **夏普比率 (Sharpe Ratio)**：平均收益 / 收益标准差（风险调整后收益）

#### 2.2.2 连续统计
- **最长连胜 (Longest Winning Streak)**：连续盈利交易的最大次数
- **最长连亏 (Longest Losing Streak)**：连续亏损交易的最大次数
- **当前连胜/连亏 (Current Streak)**：当前连续盈利或亏损的次数

#### 2.2.3 交易频率统计
- **日均交易数 (Daily Average Trades)**：平均每天的交易数
- **交易间隔 (Trade Interval)**：平均交易间隔时间
- **最活跃时段 (Most Active Hours)**：交易最频繁的时间段
- **最活跃日期 (Most Active Days)**：交易最频繁的日期

### 2.3 多维度分析

#### 2.3.1 按币种分析
- 每个交易币种的独立统计：
  - 交易次数、盈亏、胜率、平均盈亏
  - 最佳/最差表现币种
  - 币种盈亏分布

#### 2.3.2 按时间维度分析
- **按日统计**：每日的交易统计
- **按周统计**：每周的交易统计
- **按月统计**：每月的交易统计
- **自定义时间范围**：支持任意时间段的统计

#### 2.3.3 按交易类型分析
- **开仓统计**：开仓记录的分析
- **平仓统计**：平仓记录的分析
- **完整交易对统计**：开仓+平仓匹配后的完整交易分析

#### 2.3.4 按方向分析
- **做多 vs 做空**：对比分析
- **方向偏好**：交易者更偏向做多还是做空

### 2.4 交易配对分析

#### 2.4.1 FIFO 配对策略

**核心思想**：
- **FIFO（先进先出）原则**：最早的开仓记录优先与最早的平仓记录匹配
- **匹配条件**：
  1. 同一交易员（`TraderID`）
  2. 同一币种（`Symbol`）
  3. 同一方向（`Side`：`"long"` 或 `"short"`）
  4. 时间顺序：开仓时间 < 平仓时间
  5. 数量匹配：支持完全匹配和部分匹配

**数量处理**：
- **使用 `Side` 字段判断方向**（不依赖 `signed_quantity` 符号）
- **使用绝对值进行数量匹配**：`ABS(SignedQuantity)` 或 `Quantity`
- **当前数据状态**（2025-11-13）：
  - 100% 记录有 `signed_quantity`，且都是正数
  - 100% 使用新 API 格式（所有记录都有 `start_position` 字段）
  - 数据格式统一，配对逻辑更简单

**部分匹配支持**：
- 一个开仓对应多个平仓（部分平仓）
- 多个开仓对应一个平仓（多次开仓一次平仓）
- 按数量比例分配 PnL 和费用

**详细配对策略**：参见 [交易配对策略文档](./TRADE_PAIRING_STRATEGY.md)

#### 2.4.2 配对统计
- **配对成功率**：成功配对的交易数 / 总开仓数
- **平均持仓时间**：开仓到平仓的平均时间（秒）
- **最短/最长持仓时间**：持仓时间的极值
- **配对详情**：每个交易对的完整信息（开仓、平仓、数量、时间、PnL、费用）
- **未配对记录**：开仓但未平仓的记录数

### 2.5 趋势分析

#### 2.5.1 盈亏趋势
- **累计盈亏曲线**：时间序列的累计盈亏变化
- **盈亏分布**：盈亏值的分布直方图
- **盈亏趋势**：盈亏随时间的变化趋势

#### 2.5.2 交易频率趋势
- **交易频率变化**：交易频率随时间的变化
- **活跃度趋势**：交易活跃度的变化趋势

## 3. 数据模型 (Data Models)

### 3.1 核心数据结构

```go
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
}

// TradeOverview 交易概览
type TradeOverview struct {
    TotalTrades      int `json:"total_trades"`       // 总交易笔数
    OpenTrades       int `json:"open_trades"`        // 开仓笔数
    CloseTrades      int `json:"close_trades"`       // 平仓笔数
    CompletedTrades  int `json:"completed_trades"`   // 已完成交易数（有PnL）
    UnclosedTrades   int `json:"unclosed_trades"`    // 未平仓交易数
}

// PnLStatistics 盈亏统计
type PnLStatistics struct {
    TotalPnL      float64 `json:"total_pnl"`       // 总盈亏
    TotalProfit   float64 `json:"total_profit"`   // 总盈利
    TotalLoss     float64 `json:"total_loss"`     // 总亏损（绝对值）
    NetPnL        float64 `json:"net_pnl"`        // 净盈亏（总盈亏-总费用）
    AvgPnL        float64 `json:"avg_pnl"`        // 平均盈亏
    MaxWin        float64 `json:"max_win"`        // 最大单笔盈利
    MaxLoss       float64 `json:"max_loss"`       // 最大单笔亏损（绝对值）
    ProfitFactor  float64 `json:"profit_factor"`  // 盈亏比
}

// WinRateStatistics 胜率统计
type WinRateStatistics struct {
    WinRate          float64 `json:"win_rate"`           // 胜率（%）
    WinningTrades    int     `json:"winning_trades"`      // 盈利交易数
    LosingTrades     int     `json:"losing_trades"`       // 亏损交易数
    BreakEvenTrades  int     `json:"break_even_trades"`  // 持平交易数
    AvgWin           float64 `json:"avg_win"`            // 平均盈利
    AvgLoss          float64 `json:"avg_loss"`           // 平均亏损（绝对值）
}

// FeeStatistics 费用统计
type FeeStatistics struct {
    TotalFees    float64 `json:"total_fees"`     // 总费用
    OpenFees     float64 `json:"open_fees"`      // 开仓费用
    CloseFees    float64 `json:"close_fees"`     // 平仓费用
    AvgFee       float64 `json:"avg_fee"`        // 平均费用
    FeeRatio     float64 `json:"fee_ratio"`      // 费用占比（%）
    BuilderFees  float64 `json:"builder_fees"`   // Builder费用总和
}

// DirectionStatistics 方向统计
type DirectionStatistics struct {
    LongStats  *SideStatistics `json:"long_stats"`  // 做多统计
    ShortStats *SideStatistics `json:"short_stats"` // 做空统计
}

// SideStatistics 单方向统计
type SideStatistics struct {
    TotalTrades int     `json:"total_trades"`  // 交易数
    TotalPnL    float64 `json:"total_pnl"`     // 总盈亏
    WinRate     float64 `json:"win_rate"`      // 胜率（%）
    AvgPnL      float64 `json:"avg_pnl"`       // 平均盈亏
}

// RiskMetrics 风险指标
type RiskMetrics struct {
    MaxDrawdown        float64   `json:"max_drawdown"`         // 最大回撤
    MaxDrawdownPercent float64   `json:"max_drawdown_percent"`  // 最大回撤百分比
    DrawdownDuration   int64     `json:"drawdown_duration"`     // 回撤持续时间（秒）
    RecoveryTime       int64     `json:"recovery_time"`        // 恢复时间（秒）
    Volatility         float64   `json:"volatility"`           // 波动率
    SharpeRatio        float64   `json:"sharpe_ratio"`         // 夏普比率
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
    LongestWinningStreak int `json:"longest_winning_streak"` // 最长连胜
    LongestLosingStreak   int `json:"longest_losing_streak"` // 最长连亏
    CurrentStreak         int `json:"current_streak"`         // 当前连胜/连亏（正数=连胜，负数=连亏）
    CurrentStreakType     string `json:"current_streak_type"` // "winning" 或 "losing"
}

// SymbolStatistics 币种统计
type SymbolStatistics struct {
    Symbol        string  `json:"symbol"`
    TotalTrades   int     `json:"total_trades"`
    CompletedTrades int   `json:"completed_trades"`
    TotalPnL      float64 `json:"total_pnl"`
    WinRate       float64 `json:"win_rate"`
    AvgPnL        float64 `json:"avg_pnl"`
    MaxWin        float64 `json:"max_win"`
    MaxLoss       float64 `json:"max_loss"`
    TotalFees     float64 `json:"total_fees"`
}

// TimeSeriesStatistics 时间序列统计
type TimeSeriesStatistics struct {
    DailyStats  []DailyStatistics  `json:"daily_stats"`   // 每日统计
    WeeklyStats []WeeklyStatistics `json:"weekly_stats"`  // 每周统计
    MonthlyStats []MonthlyStatistics `json:"monthly_stats"` // 每月统计
}

// DailyStatistics 每日统计
type DailyStatistics struct {
    Date         string  `json:"date"`          // YYYY-MM-DD
    TotalTrades  int     `json:"total_trades"`
    CompletedTrades int `json:"completed_trades"`
    TotalPnL     float64 `json:"total_pnl"`
    TotalFees    float64 `json:"total_fees"`
    WinRate      float64 `json:"win_rate"`
}

// WeeklyStatistics 每周统计
type WeeklyStatistics struct {
    Week         string  `json:"week"`         // YYYY-WW
    TotalTrades  int     `json:"total_trades"`
    CompletedTrades int `json:"completed_trades"`
    TotalPnL     float64 `json:"total_pnl"`
    TotalFees    float64 `json:"total_fees"`
    WinRate      float64 `json:"win_rate"`
}

// MonthlyStatistics 每月统计
type MonthlyStatistics struct {
    Month        string  `json:"month"`        // YYYY-MM
    TotalTrades  int     `json:"total_trades"`
    CompletedTrades int `json:"completed_trades"`
    TotalPnL     float64 `json:"total_pnl"`
    TotalFees    float64 `json:"total_fees"`
    WinRate      float64 `json:"win_rate"`
}

// PairStatistics 配对统计
type PairStatistics struct {
    TotalPairs       int     `json:"total_pairs"`        // 成功配对数量
    PairSuccessRate  float64 `json:"pair_success_rate"` // 配对成功率（%）
    AvgHoldingTime    int64   `json:"avg_holding_time"`  // 平均持仓时间（秒）
    MinHoldingTime    int64   `json:"min_holding_time"`  // 最短持仓时间（秒）
    MaxHoldingTime    int64   `json:"max_holding_time"`  // 最长持仓时间（秒）
    Pairs            []TradePair `json:"pairs"`          // 配对详情
}

// TradePair 交易对
type TradePair struct {
    OpenRecord  *TradeRecord `json:"open_record"`   // 开仓记录
    CloseRecord *TradeRecord `json:"close_record"` // 平仓记录
    MatchedQty  float64      `json:"matched_qty"`   // 匹配的数量
    HoldingTime int64        `json:"holding_time"`  // 持仓时间（秒）
    PnL         float64      `json:"pnl"`           // 盈亏（来自CloseRecord.PnL，按比例分配）
    OpenFee     float64      `json:"open_fee"`     // 开仓费用（按比例）
    CloseFee    float64      `json:"close_fee"`    // 平仓费用（按比例）
}
```

### 3.2 查询过滤器

```go
// AnalyticsFilter 分析查询过滤器
type AnalyticsFilter struct {
    TraderID    string     `json:"trader_id"`     // 交易员ID（必需）
    Symbol      string     `json:"symbol"`        // 币种过滤（可选）
    Side        string     `json:"side"`          // 方向过滤：'long' 或 'short'（可选）
    Action      string     `json:"action"`        // 操作过滤：'open_long', 'close_long' 等（可选）
    StartTime   *time.Time `json:"start_time"`   // 开始时间（可选）
    EndTime     *time.Time `json:"end_time"`     // 结束时间（可选）
    GroupBy     string     `json:"group_by"`      // 分组方式：'day', 'week', 'month', 'symbol'（可选）
    IncludePairs bool      `json:"include_pairs"` // 是否包含配对分析（默认false）
}
```

## 4. API 设计 (API Design)

### 4.1 RESTful API 端点

#### 4.1.1 获取完整分析
```
GET /api/trade-analytics
Query Parameters:
  - trader_id (required): 交易员ID
  - symbol (optional): 币种过滤
  - side (optional): 方向过滤
  - start_time (optional): 开始时间 (RFC3339)
  - end_time (optional): 结束时间 (RFC3339)
  - include_pairs (optional): 是否包含配对分析 (true/false, default: false)
  - group_by (optional): 分组方式 (day/week/month/symbol, default: none)

Response: TradeAnalytics
```

#### 4.1.2 获取概览统计
```
GET /api/trade-analytics/overview
Query Parameters: (同 4.1.1)

Response: TradeOverview
```

#### 4.1.3 获取盈亏统计
```
GET /api/trade-analytics/pnl
Query Parameters: (同 4.1.1)

Response: PnLStatistics
```

#### 4.1.4 获取胜率统计
```
GET /api/trade-analytics/win-rate
Query Parameters: (同 4.1.1)

Response: WinRateStatistics
```

#### 4.1.5 获取费用统计
```
GET /api/trade-analytics/fees
Query Parameters: (同 4.1.1)

Response: FeeStatistics
```

#### 4.1.6 获取风险指标
```
GET /api/trade-analytics/risk
Query Parameters: (同 4.1.1)

Response: RiskMetrics
```

#### 4.1.7 获取币种统计
```
GET /api/trade-analytics/symbols
Query Parameters: (同 4.1.1)

Response: map[string]*SymbolStatistics
```

#### 4.1.8 获取时间序列统计
```
GET /api/trade-analytics/time-series
Query Parameters:
  - trader_id (required)
  - start_time (optional)
  - end_time (optional)
  - group_by (required): day/week/month

Response: TimeSeriesStatistics
```

#### 4.1.9 获取交易配对
```
GET /api/trade-analytics/pairs
Query Parameters:
  - trader_id (required)
  - symbol (optional)
  - start_time (optional)
  - end_time (optional)
  - limit (optional): 返回数量限制 (default: 100)

Response: PairStatistics
```

### 4.2 响应格式示例

```json
{
  "overview": {
    "total_trades": 150,
    "open_trades": 75,
    "close_trades": 75,
    "completed_trades": 70,
    "unclosed_trades": 5
  },
  "pnl_stats": {
    "total_pnl": 1250.50,
    "total_profit": 2000.00,
    "total_loss": 749.50,
    "net_pnl": 1200.00,
    "avg_pnl": 17.86,
    "max_win": 150.00,
    "max_loss": 80.00,
    "profit_factor": 2.67
  },
  "win_rate_stats": {
    "win_rate": 65.71,
    "winning_trades": 46,
    "losing_trades": 24,
    "break_even_trades": 0,
    "avg_win": 43.48,
    "avg_loss": 31.23
  },
  "fee_stats": {
    "total_fees": 50.50,
    "open_fees": 25.25,
    "close_fees": 25.25,
    "avg_fee": 0.34,
    "fee_ratio": 4.04,
    "builder_fees": 5.00
  },
  "direction_stats": {
    "long_stats": {
      "total_trades": 40,
      "total_pnl": 800.00,
      "win_rate": 70.00,
      "avg_pnl": 20.00
    },
    "short_stats": {
      "total_trades": 30,
      "total_pnl": 450.50,
      "win_rate": 60.00,
      "avg_pnl": 15.02
    }
  },
  "risk_metrics": {
    "max_drawdown": 200.00,
    "max_drawdown_percent": 15.38,
    "drawdown_duration": 3600,
    "recovery_time": 1800,
    "volatility": 25.50,
    "sharpe_ratio": 0.70
  },
  "streak_stats": {
    "longest_winning_streak": 8,
    "longest_losing_streak": 4,
    "current_streak": 3,
    "current_streak_type": "winning"
  },
  "symbol_stats": {
    "BTCUSDT": {
      "symbol": "BTCUSDT",
      "total_trades": 50,
      "completed_trades": 45,
      "total_pnl": 600.00,
      "win_rate": 66.67,
      "avg_pnl": 13.33,
      "max_win": 100.00,
      "max_loss": 50.00,
      "total_fees": 20.00
    }
  }
}
```

## 5. 实现计划 (Implementation Plan)

### 5.1 模块结构

```
trade_analytics/
├── models.go          # 数据模型定义
├── repository.go      # 数据访问层（扩展 trade_history.Repository）
├── analyzer.go        # 核心分析逻辑
├── pair_matcher.go    # 交易配对逻辑
├── risk_calculator.go # 风险指标计算
├── service.go         # 服务层接口和实现
└── api.go             # API 处理器
```

### 5.2 实现阶段

#### Phase 1: 基础统计（核心功能）
- [ ] 实现基础数据模型
- [ ] 实现概览统计（总交易数、开仓/平仓数等）
- [ ] 实现盈亏统计（总盈亏、平均盈亏、盈亏比等）
- [ ] 实现胜率统计（胜率、盈利/亏损交易数等）
- [ ] 实现费用统计（总费用、平均费用等）
- [ ] 实现基础 API 端点

#### Phase 2: 方向和多维度分析
- [ ] 实现方向统计（做多/做空）
- [ ] 实现币种统计（按币种分组）
- [ ] 实现时间维度统计（按日/周/月）
- [ ] 扩展 API 端点支持过滤和分组

#### Phase 3: 高级分析
- [ ] 实现交易配对逻辑（FIFO）
  - [ ] 实现 `getAbsoluteQuantity()` 辅助函数
  - [ ] 实现 FIFO 配对核心算法
  - [ ] 实现部分匹配处理逻辑
  - [ ] 实现 PnL 和费用按比例分配
  - [ ] 参考：[交易配对策略文档](./TRADE_PAIRING_STRATEGY.md)
- [ ] 实现风险指标计算（最大回撤、夏普比率等）
- [ ] 实现连续统计（连胜/连亏）
- [ ] 实现趋势分析

#### Phase 4: 优化和扩展
- [ ] 查询性能优化（索引、缓存）
- [ ] 大数据量处理优化
- [ ] 前端集成
- [ ] 文档和测试

### 5.3 技术考虑

#### 5.3.1 数据库查询优化
- 为常用查询字段添加索引（trader_id, symbol, timestamp, action, side）
- 使用聚合查询减少数据传输
- 考虑物化视图或缓存常用统计结果

#### 5.3.2 配对算法

**实现要点**：
- **FIFO 配对**：按时间顺序（`Timestamp ASC`）匹配开仓和平仓
- **匹配键**：使用 `Symbol + "_" + Side` 作为分组键（不依赖 `signed_quantity` 符号）
- **数量获取**：使用 `getAbsoluteQuantity()` 函数获取绝对数量
  ```go
  func getAbsoluteQuantity(record *TradeRecord) float64 {
      if record.SignedQuantity != nil {
          return math.Abs(*record.SignedQuantity)
      }
      return record.Quantity
  }
  ```
- **部分匹配处理**：
  - 开仓数量 < 平仓数量：开仓全部匹配，平仓继续匹配下一个开仓
  - 开仓数量 > 平仓数量：平仓全部匹配，开仓保留剩余数量
- **PnL 分配**：按匹配数量比例分配平仓记录的 PnL
- **费用分配**：开仓和平仓费用都按匹配数量比例分配
- **未配对处理**：保留在 `openPositions` 队列中，计入未平仓交易数

**详细实现**：参见 [交易配对策略文档](./TRADE_PAIRING_STRATEGY.md) 第 4 节

#### 5.3.3 风险指标计算
- 最大回撤：需要计算累计盈亏曲线
- 夏普比率：需要计算收益的标准差
- 考虑使用滑动窗口优化计算

#### 5.3.4 性能考虑
- 大数据量时使用分页
- 时间范围查询时使用索引
- 考虑异步计算复杂指标

## 6. 测试需求 (Testing Requirements)

### 6.1 单元测试
- 统计计算逻辑测试
- 配对算法测试
- 风险指标计算测试
- 过滤器测试

### 6.2 集成测试
- API 端点测试
- 数据库查询测试
- 端到端流程测试

### 6.3 性能测试
- 大数据量查询性能
- 并发请求处理
- 内存使用优化

## 7. 前端集成 (Frontend Integration)

### 7.1 数据展示
- 统计概览卡片
- 盈亏趋势图表
- 币种对比图表
- 风险指标仪表盘

### 7.2 交互功能
- 时间范围选择器
- 币种/方向过滤器
- 数据导出功能
- 实时刷新

## 8. 未来扩展 (Future Enhancements)

### 8.1 高级分析
- 机器学习预测
- 交易模式识别
- 异常检测

### 8.2 报告生成
- PDF 报告导出
- 邮件报告推送
- 自定义报告模板

### 8.3 对比分析
- 多交易员对比
- 策略对比
- 时间段对比

## 9. 注意事项 (Notes)

### 9.1 数据完整性
- 确保所有统计基于完整的交易记录
- 处理缺失数据的情况（如 PnL 为 NULL）
- 验证数据一致性

### 9.2 计算准确性
- 验证所有计算公式的正确性
- 处理边界情况（如除零）
- 确保数值精度

### 9.3 性能优化
- 避免全表扫描
- 使用适当的索引
- 考虑缓存策略

### 9.4 向后兼容
- 不修改现有的 `GetStatistics` 方法
- 新模块独立运行
- 保持 API 版本兼容性

## 10. 参考文档 (References)

### 10.1 核心文档
- **交易配对策略**：[TRADE_PAIRING_STRATEGY.md](./TRADE_PAIRING_STRATEGY.md)
  - 详细的 FIFO 配对算法说明
  - 数量处理逻辑和特殊情况处理
  - 伪代码实现和测试用例
- **SignedQuantity 字段分析**：[TRADE_HISTORY_SIGNED_QUANTITY_ANALYSIS.md](./TRADE_HISTORY_SIGNED_QUANTITY_ANALYSIS.md)
- **Hyperliquid Size 符号规则**：[HYPERLIQUID_SIZE_SIGN_RULE.md](./HYPERLIQUID_SIZE_SIGN_RULE.md)
- **数据重新检查报告**：[TRADE_HISTORY_SIGNED_QUANTITY_RECHECK.md](./TRADE_HISTORY_SIGNED_QUANTITY_RECHECK.md)

### 10.2 代码参考
- Trade History 数据模型：`trade_history/models.go`
- 现有统计实现：`trade_history/repository.go::GetStatistics`
- 数据库 Schema：`config/database.go`
- API 路由：`api/server.go`

---

**文档版本**: 1.0  
**创建日期**: 2025-11-13  
**最后更新**: 2025-11-13


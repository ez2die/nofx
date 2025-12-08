# 复盘模块 - 详细实施计划与架构设计

**文档版本**: v1.0  
**创建日期**: 2025-01-XX  
**参考文档**: 
- `REVIEW_MODULE_REQUIREMENTS.md` - 业务需求细化文档
- `REVIEW_MODULE_TECHNICAL_INSIGHTS.md` - 主程序员技术见解

---

## 目录

1. [项目概览](#项目概览)
2. [整体架构设计](#整体架构设计)
3. [Phase I 详细设计](#phase-i-详细设计)
4. [项目计划](#项目计划)
5. [技术规范](#技术规范)
6. [测试策略](#测试策略)

---

## 一、项目概览

### 1.1 项目目标

实现一个自动复盘系统，每6小时自动复盘一次，系统性地检讨过去6小时内的交易历史，通过复盘总结经验、检讨错误，提升交易系统的持续改进能力。

### 1.2 核心价值

- **错误识别与纠正**：及时发现并记录违反约束、时机错误、判断偏差等问题
- **成功模式提取**：识别有效的交易策略和市场条件组合
- **规则优化依据**：为系统提示词(prompt)的优化提供数据支撑
- **风险预警**：识别潜在的系统性问题，提前预警
- **持续改进**：基于数据驱动的复盘结果，持续优化交易系统

### 1.3 实施阶段

- **Phase 1 (MVP)**: 核心功能，快速上线（2-3周）
- **Phase 2**: 深度分析，多维度分析（3-4周）
- **Phase 3**: 高级功能，AI增强（3-4周）
- **Phase 4**: 系统级优化（4-6周）

---

## 二、整体架构设计

### 2.1 架构原则

1. **依赖注入**：通过接口依赖，不直接依赖实现
2. **模块化设计**：清晰的模块边界，便于测试和维护
3. **性能优先**：总体性能目标<5分钟，各阶段<30秒
4. **容错设计**：区分致命/部分/警告错误，优雅降级
5. **可扩展性**：支持未来功能扩展（Phase 2/3/4）

### 2.2 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      AutoTrader (主程序)                      │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │          Review Scheduler (定时调度器)                 │  │
│  │  - 每6小时触发一次                                      │  │
│  │  - 使用 goroutine + ticker                            │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Review Service (复盘服务主逻辑)                   │
│  - 编排整个复盘流程                                          │
│  - 错误处理和容错                                            │
│  - 性能监控                                                  │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   Collector  │   │   Analyzer   │   │   Reporter   │
│  (数据收集)   │   │   (分析引擎)  │   │  (报告生成)   │
└──────────────┘   └──────────────┘   └──────────────┘
        │                   │                   │
        ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────┐
│                    数据源层 (依赖注入)                        │
│  - DecisionLogReader (决策日志)                              │
│  - TradeHistoryReader (交易历史)                             │
│  - DEXDataProvider (DEX数据)                                 │
│  - TradeAnalyticsService (交易分析)                          │
└─────────────────────────────────────────────────────────────┘
```

### 2.3 模块结构

```
review/
├── service.go              # 复盘服务主逻辑（编排）
├── scheduler.go            # 定时调度器
├── collector/              # 数据收集层
│   ├── decision_log.go    # 决策日志收集器
│   ├── trade_history.go   # 交易历史收集器
│   └── dex_data.go        # DEX数据收集器
├── analyzer/               # 分析层
│   ├── performance.go     # 交易表现分析
│   ├── error_detector.go  # 错误识别
│   ├── pattern_extractor.go # 成功模式提取
│   └── rule_engine.go     # 规则引擎
├── metrics/                # 指标计算层
│   ├── violation_metrics.go # 规则违反指标
│   ├── direction_metrics.go # 方向后验指标
│   └── dex_validation.go   # DEX验证指标
├── matcher/                # 数据匹配层
│   └── decision_matcher.go # 决策与DEX交易匹配
├── reporter/               # 报告生成层
│   ├── markdown_reporter.go # Markdown报告生成
│   └── template.go         # 报告模板
├── models.go               # 数据模型
└── repository.go           # 数据访问层（复盘记录存储）
```

### 2.4 数据流设计

```
┌─────────────────────────────────────────────────────────────┐
│                    Phase 1: 数据收集 (30s)                    │
├─────────────────────────────────────────────────────────────┤
│  1. DecisionLogCollector                                    │
│     - 查询决策日志 (按时间范围)                               │
│     - 并行读取JSON文件 (goroutine pool)                      │
│                                                              │
│  2. TradeHistoryCollector                                   │
│     - 查询交易历史 (按时间范围)                               │
│     - 匹配开仓/平仓记录                                      │
│                                                              │
│  3. DEXDataCollector                                        │
│     - 从DEX拉取交易数据 (按时间范围)                          │
│     - 重试机制 (指数退避，最大3次)                            │
└─────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Phase 2: 数据匹配 (10s)                      │
├─────────────────────────────────────────────────────────────┤
│  DecisionMatcher                                            │
│  - 匹配决策与DEX交易 (优先级: OrderID > 时间窗口 > 数量价格)  │
│  - 计算匹配置信度                                            │
│  - 识别未匹配的决策和DEX交易                                  │
└─────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Phase 3: 分析处理 (30s)                      │
├─────────────────────────────────────────────────────────────┤
│  1. PerformanceAnalyzer                                     │
│     - 交易表现分析 (复用 trade_analytics)                     │
│                                                              │
│  2. RuleEngine                                              │
│     - 硬约束检查 (风险回报比、杠杆限制、止损方向)              │
│     - 风险控制检查                                            │
│     - 规则违反指标计算                                        │
│                                                              │
│  3. DEXValidationAnalyzer                                   │
│     - 执行质量分析 (滑点、时间差、费用差异)                   │
│     - DEX验证指标计算                                        │
└─────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Phase 4: 报告生成 (30s)                     │
├─────────────────────────────────────────────────────────────┤
│  MarkdownReporter                                           │
│  - 基于模板生成Markdown报告                                   │
│  - 包含执行摘要、表现概览、详细分析、改进建议                  │
│  - 保存到文件系统 (data/reviews/)                            │
│  - 保存元数据到数据库 (review_records表)                      │
└─────────────────────────────────────────────────────────────┘
```

### 2.5 接口设计

#### 2.5.1 数据收集接口

```go
// DecisionLogReader 决策日志读取接口
type DecisionLogReader interface {
    GetRecordsByTimeRange(startTime, endTime time.Time) ([]*DecisionRecord, error)
}

// TradeHistoryReader 交易历史读取接口
type TradeHistoryReader interface {
    FindByFilter(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, error)
}

// DEXDataProvider DEX数据提供接口
type DEXDataProvider interface {
    GetFillsByTimeRange(startTime, endTime time.Time) ([]ExchangeFill, error)
}

// TradeAnalyticsService 交易分析服务接口
type TradeAnalyticsService interface {
    GetAnalytics(ctx context.Context, filter *AnalyticsFilter) (*TradeAnalytics, error)
}
```

#### 2.5.2 分析引擎接口

```go
// RuleEngine 规则引擎接口
type RuleEngine interface {
    CheckDecision(decision *DecisionRecord) ([]Violation, error)
    GetRuleDefinitions() []RuleDefinition
}

// PerformanceAnalyzer 表现分析接口
type PerformanceAnalyzer interface {
    AnalyzePerformance(trades []*TradeRecord, decisions []*DecisionRecord) (*PerformanceAnalysis, error)
}

// DEXValidationAnalyzer DEX验证分析接口
type DEXValidationAnalyzer interface {
    AnalyzeExecutionQuality(matches []*DecisionMatch) (*ExecutionQuality, error)
    CalculateDEXMetrics(matches []*DecisionMatch, unmatchedDecisions []*DecisionRecord, unmatchedDEXTrades []ExchangeFill) (*DEXValidationMetrics, error)
}
```

#### 2.5.3 报告生成接口

```go
// Reporter 报告生成接口
type Reporter interface {
    GenerateReport(review *ReviewResult) (string, error) // 返回报告文件路径
}
```

---

## 三、Phase I 详细设计

### 3.1 Phase I 目标

**核心功能**：
1. ✅ 数据收集（决策日志、交易历史、DEX数据）
2. ✅ 基础分析（交易表现、硬约束检查）
3. ✅ 决策执行验证（匹配决策与DEX交易）
4. ✅ 规则违反指标计算
5. ✅ Markdown报告生成
6. ✅ 定时调度（每6小时）

**性能目标**：
- 总体性能：< 5分钟
- 数据收集：< 30秒
- DEX数据拉取：< 60秒（正常），< 180秒（含重试）
- 分析处理：< 30秒
- 报告生成：< 30秒

### 3.2 模块详细设计

#### 3.2.1 Review Service (service.go)

**职责**：编排整个复盘流程，错误处理，性能监控

**接口设计**：
```go
type ReviewService interface {
    // RunReview 执行一次复盘
    RunReview(ctx context.Context, traderID string, startTime, endTime time.Time) (*ReviewResult, error)
}

type ReviewResult struct {
    TraderID        string
    StartTime       time.Time
    EndTime         time.Time
    Status          string // "success" | "partial" | "failed"
    ErrorMessage    string
    Performance     *PerformanceAnalysis
    Violations      []Violation
    DEXValidation   *DEXValidationMetrics
    ExecutionQuality *ExecutionQuality
    ReportPath      string
    Metrics         *StandardizedMetrics
    Duration        time.Duration
}
```

**实现要点**：
1. **流程编排**：
   - 数据收集 → 数据匹配 → 分析处理 → 报告生成
   - 各阶段独立，可单独测试
2. **错误处理**：
   - 致命错误：DEX数据拉取失败 → 停止复盘，记录日志
   - 部分错误：部分决策日志文件损坏 → 继续复盘，标注缺失数据
   - 警告：数据不完整 → 继续复盘，标注数据来源
3. **性能监控**：
   - 记录各阶段耗时
   - 如果某个阶段超过目标，记录警告
4. **依赖注入**：
   - 通过构造函数注入所有依赖
   - 便于测试和替换实现

**代码结构**：
```go
type reviewService struct {
    decisionLogReader    DecisionLogReader
    tradeHistoryReader   TradeHistoryReader
    dexDataProvider      DEXDataProvider
    tradeAnalyticsService TradeAnalyticsService
    ruleEngine           RuleEngine
    performanceAnalyzer  PerformanceAnalyzer
    dexValidationAnalyzer DEXValidationAnalyzer
    decisionMatcher      DecisionMatcher
    reporter             Reporter
    repository           ReviewRepository
}

func (s *reviewService) RunReview(ctx context.Context, traderID string, startTime, endTime time.Time) (*ReviewResult, error) {
    // 1. 数据收集
    // 2. 数据匹配
    // 3. 分析处理
    // 4. 报告生成
    // 5. 保存结果
}
```

#### 3.2.2 Scheduler (scheduler.go)

**职责**：定时调度复盘任务

**接口设计**：
```go
type ReviewScheduler interface {
    Start(ctx context.Context, traderID string) error
    Stop() error
}

type ReviewSchedulerConfig struct {
    Interval        time.Duration // 默认6小时
    FirstRunDelay   time.Duration // 首次执行延迟
    Enabled         bool          // 是否启用
}
```

**实现要点**：
1. **调度机制**：
   - 使用 `time.Ticker` + `context.Context`
   - 支持优雅停止（通过context取消）
   - 复用 `trade_history/sync.go` 的模式
2. **启动时机**：
   - 在 `AutoTrader.Run()` 中启动
   - 或在 `main.go` 中独立启动
3. **配置**：
   - 从配置文件读取间隔、首次延迟等
   - 支持动态启用/禁用

**代码结构**：
```go
type reviewScheduler struct {
    service  ReviewService
    interval time.Duration
    config   ReviewSchedulerConfig
}

func (s *reviewScheduler) Start(ctx context.Context, traderID string) error {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    // 首次执行延迟
    if s.config.FirstRunDelay > 0 {
        time.Sleep(s.config.FirstRunDelay)
    }
    
    // 立即执行一次
    go s.runReviewOnce(ctx, traderID)
    
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            go s.runReviewOnce(ctx, traderID)
        }
    }
}
```

#### 3.2.3 Collector - Decision Log (collector/decision_log.go)

**职责**：收集决策日志数据

**接口设计**：
```go
type DecisionLogCollector interface {
    Collect(ctx context.Context, startTime, endTime time.Time) ([]*DecisionRecord, error)
}
```

**实现要点**：
1. **时间范围查询**：
   - 实现 `GetRecordsByTimeRange()` 方法（需要增强 `DecisionLogger`）
   - 遍历 `logDir` 目录下的所有JSON文件
   - 解析文件名中的时间戳（`decision_YYYYMMDD_HHMMSS_cycleN.json`）
   - 过滤时间范围
2. **性能优化**：
   - 并行读取文件（goroutine pool，建议10个并发）
   - 使用 `sync.WaitGroup` 等待所有goroutine完成
   - 有界goroutine池，避免资源耗尽
3. **错误处理**：
   - 文件损坏：跳过该文件，记录警告，继续处理其他文件
   - JSON解析失败：跳过该文件，记录警告

**代码结构**：
```go
type decisionLogCollector struct {
    logger DecisionLogReader
}

func (c *decisionLogCollector) Collect(ctx context.Context, startTime, endTime time.Time) ([]*DecisionRecord, error) {
    // 1. 遍历 logDir 目录
    // 2. 解析文件名，提取时间戳
    // 3. 过滤时间范围
    // 4. 并行读取文件（goroutine pool）
    // 5. 返回结果
}
```

**需要增强的现有模块**：
- `logger/decision_logger.go`：新增 `GetRecordsByTimeRange()` 方法

#### 3.2.4 Collector - Trade History (collector/trade_history.go)

**职责**：收集交易历史数据

**接口设计**：
```go
type TradeHistoryCollector interface {
    Collect(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*TradeRecord, error)
}
```

**实现要点**：
1. **时间范围查询**：
   - 使用 `TradeHistoryReader.FindByFilter()` 方法
   - 构建 `TradeRecordFilter`，设置时间范围
2. **边界处理**：
   - 只统计在复盘窗口内**完整完成**的交易（开仓+平仓都在窗口内）
   - 如果窗口开始时间点的第一笔交易是`close`操作，需要追溯到上一个窗口找到对应的`open`操作
   - 如果窗口结束时间点的最后一笔交易是`open`操作，对应的`close`操作在窗口外，则该交易不计入本次复盘
3. **数据完整性**：
   - 检查开仓是否有对应的平仓
   - 标记未完成的交易

**代码结构**：
```go
type tradeHistoryCollector struct {
    reader TradeHistoryReader
}

func (c *tradeHistoryCollector) Collect(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*TradeRecord, error) {
    // 1. 构建过滤器
    // 2. 查询交易历史
    // 3. 边界处理（追溯上一个窗口的open）
    // 4. 过滤未完成的交易
    // 5. 返回结果
}
```

#### 3.2.5 Collector - DEX Data (collector/dex_data.go)

**职责**：从DEX拉取交易数据

**接口设计**：
```go
type DEXDataCollector interface {
    Collect(ctx context.Context, startTime, endTime time.Time) ([]ExchangeFill, error)
}
```

**实现要点**：
1. **数据拉取**：
   - 使用 `HyperliquidFillsProvider.GetFillsByTimeRange()` 方法
   - 不保存到数据库，仅用于复盘验证
2. **错误处理**：
   - **失败策略**：DEX数据拉取失败时，直接报错停止本轮复盘（不降级）
   - API限流：检测到限流时，等待后重试（指数退避）
   - 最大重试次数：3次
   - 重试间隔：1秒、2秒、4秒
   - 网络错误：重试3次后，如果仍然失败，记录错误日志并停止复盘
   - API错误：记录错误日志并停止复盘
3. **性能优化**：
   - 增量拉取（只拉取复盘窗口内的数据）
   - 如果API支持，可并行拉取多个币种的数据

**代码结构**：
```go
type dexDataCollector struct {
    provider DEXDataProvider
    maxRetries int
    retryIntervals []time.Duration
}

func (c *dexDataCollector) Collect(ctx context.Context, startTime, endTime time.Time) ([]ExchangeFill, error) {
    var lastErr error
    for i := 0; i < c.maxRetries; i++ {
        fills, err := c.provider.GetFillsByTimeRange(startTime, endTime)
        if err == nil {
            return fills, nil
        }
        lastErr = err
        
        // 指数退避
        if i < len(c.retryIntervals) {
            time.Sleep(c.retryIntervals[i])
        }
    }
    return nil, fmt.Errorf("DEX数据拉取失败（重试%d次）: %w", c.maxRetries, lastErr)
}
```

#### 3.2.6 Matcher - Decision Matcher (matcher/decision_matcher.go)

**职责**：匹配决策与DEX交易

**接口设计**：
```go
type DecisionMatcher interface {
    Match(decisions []*DecisionRecord, dexTrades []ExchangeFill) ([]*DecisionMatch, []*DecisionRecord, []ExchangeFill, error)
    // 返回：匹配结果、未匹配的决策、未匹配的DEX交易
}
```

**实现要点**：
1. **匹配策略**（按优先级）：
   - **优先级1：ExchangeOrderID匹配**（最准确）
     - 从 `DecisionAction.OrderID` 提取订单ID
     - 与 `ExchangeFill.ExchangeOid` 匹配
     - 匹配置信度：1.0
   - **优先级2：时间窗口匹配**（备选）
     - 时间窗口：决策时间前后5-10分钟（可配置）
     - 币种：完全匹配
     - 方向：long/short匹配
     - 匹配置信度：0.8
   - **优先级3：数量+价格匹配**（最后备选）
     - 数量差异：< 5%
     - 价格差异：< 1%
     - 匹配置信度：0.6
2. **部分匹配处理**：
   - 一个决策对应多个DEX交易：合并计算（累加数量、平均价格）
   - 多个决策对应一个DEX交易：标记为"系统重试"，需要人工确认
3. **匹配结果**：
   - 计算滑点：`|实际价格 - 决策价格| / 决策价格`
   - 计算执行延迟：`实际执行时间 - 决策时间`
   - 计算费用差异：`实际费用 - 预期费用`

**代码结构**：
```go
type decisionMatcher struct {
    timeWindow      time.Duration
    quantityTolerance float64 // 5%
    priceTolerance   float64 // 1%
}

func (m *decisionMatcher) Match(decisions []*DecisionRecord, dexTrades []ExchangeFill) ([]*DecisionMatch, []*DecisionRecord, []ExchangeFill, error) {
    var matches []*DecisionMatch
    var unmatchedDecisions []*DecisionRecord
    var unmatchedDEXTrades []ExchangeFill
    
    // 优先级1: ExchangeOrderID匹配
    // 优先级2: 时间窗口匹配
    // 优先级3: 数量+价格匹配
    
    return matches, unmatchedDecisions, unmatchedDEXTrades, nil
}
```

#### 3.2.7 Analyzer - Rule Engine (analyzer/rule_engine.go)

**职责**：规则检查，识别违反约束的决策

**接口设计**：
```go
type RuleEngine interface {
    CheckDecision(decision *DecisionRecord) ([]Violation, error)
    GetRuleDefinitions() []RuleDefinition
}

type Violation struct {
    RuleID      string
    RuleName    string
    Type        string // "hard_constraint" | "risk_control" | "position_management"
    Severity    string // "critical" | "high" | "medium" | "low"
    Description string
    DecisionID  string
    Details     map[string]interface{}
}

type RuleDefinition struct {
    ID          string
    Name        string
    Type        string
    Severity    string
    Description string
    PromptRef   string
    Priority    int
}
```

**实现要点**：
1. **规则定义方式**：
   - 在代码中明确定义规则（硬编码），而非从prompt中NLP提取
   - 规则描述从prompt中提取，用于展示
   - 规则检查逻辑在代码中实现
2. **插件化设计**：
   - 采用接口设计，支持规则的动态注册
   - 不同类型的规则检查器独立实现
   - 支持规则的优先级排序
3. **Phase 1 核心规则**：
   - **硬约束检查**：
     - 风险回报比：≥ 3.0:1
     - 杠杆限制：不超过系统配置限制
     - 止损/止盈方向：多仓 `stop_loss < entry < take_profit`，空仓 `take_profit < entry < stop_loss`
     - 置信度要求：开仓置信度 ≥ 70，Sharpe < 0 时 ≥ 85
   - **基础风险控制检查**：
     - 止损宽度检查（BTC/ETH最小0.3-0.5%）
     - 账户风险检查（价格止损距离% × 杠杆 ≤ 8%）
4. **DecisionJSON解析**：
   - 使用宽松的JSON解析（允许字段缺失）
   - 提供默认值
   - 记录解析失败的决策，在报告中标注

**代码结构**：
```go
type ruleEngine struct {
    rules []RuleChecker
}

type RuleChecker interface {
    Check(decision *DecisionRecord) ([]Violation, error)
    GetRuleID() string
    GetSeverity() string
    GetPriority() int
}

// 具体规则检查器
type RiskRewardRatioChecker struct {
    minRatio float64 // 3.0
}

func (c *RiskRewardRatioChecker) Check(decision *DecisionRecord) ([]Violation, error) {
    // 解析 DecisionJSON，提取 stop_loss, take_profit, entry_price
    // 计算风险回报比
    // 返回违反结果
}

// 其他规则检查器...
```

#### 3.2.8 Analyzer - Performance (analyzer/performance.go)

**职责**：交易表现分析

**接口设计**：
```go
type PerformanceAnalyzer interface {
    AnalyzePerformance(trades []*TradeRecord, decisions []*DecisionRecord) (*PerformanceAnalysis, error)
}

type PerformanceAnalysis struct {
    // 基础统计
    TotalTrades      int
    OpenTrades       int
    CloseTrades      int
    CompletedTrades  int
    
    // 盈亏统计
    TotalPnL         float64
    NetPnL           float64 // 扣除手续费
    AvgPnL           float64
    MaxProfit        float64
    MaxLoss          float64
    
    // 胜率统计
    WinRate          float64
    WinTrades        int
    LossTrades       int
    
    // 费用统计
    TotalFees        float64
    AvgFee           float64
    FeeRatio         float64 // 费用/总盈亏
    
    // 风险指标
    MaxDrawdown      float64
    Volatility       float64
    SharpeRatio      float64
    
    // 币种表现
    SymbolStats      map[string]*SymbolPerformance
}
```

**实现要点**：
1. **复用现有模块**：
   - 调用 `TradeAnalyticsService.GetAnalytics()` 获取基础统计
   - 复用 `trade_analytics` 模块的计算逻辑
2. **数据补充**：
   - 补充决策相关的统计（决策数量、决策执行率等）
   - 补充DEX验证相关的统计（执行质量等）
3. **性能优化**：
   - 批量查询，避免多次数据库访问
   - 缓存计算结果

**代码结构**：
```go
type performanceAnalyzer struct {
    analyticsService TradeAnalyticsService
}

func (a *performanceAnalyzer) AnalyzePerformance(trades []*TradeRecord, decisions []*DecisionRecord) (*PerformanceAnalysis, error) {
    // 1. 调用 trade_analytics 获取基础统计
    // 2. 补充决策相关统计
    // 3. 返回结果
}
```

#### 3.2.9 Analyzer - DEX Validation (analyzer/dex_validation.go)

**职责**：DEX验证分析，执行质量分析

**接口设计**：
```go
type DEXValidationAnalyzer interface {
    AnalyzeExecutionQuality(matches []*DecisionMatch) (*ExecutionQuality, error)
    CalculateDEXMetrics(matches []*DecisionMatch, unmatchedDecisions []*DecisionRecord, unmatchedDEXTrades []ExchangeFill) (*DEXValidationMetrics, error)
}

type ExecutionQuality struct {
    TotalMatches           int
    AverageSlippage        float64
    AverageExecutionDelay  int64
    MaxSlippage            float64
    MaxExecutionDelay      int64
    SlippageDistribution   map[string]int
    ExecutionDelayDistribution map[string]int
}

type DEXValidationMetrics struct {
    DecisionExecutionRate    float64
    UnmatchedDecisions       int
    UnmatchedDEXTrades       int
    AverageSlippage         float64
    AverageExecutionDelay    int64
    FeeDifference            float64
    TotalFundingFees         float64
}
```

**实现要点**：
1. **执行质量分析**：
   - 计算平均滑点、最大滑点
   - 计算平均执行延迟、最大执行延迟
   - 统计滑点分布、执行延迟分布
2. **DEX验证指标计算**：
   - 决策执行率：成功匹配的决策数 / 总决策数
   - 未匹配决策数：决策日志中有但DEX没有的交易
   - 未匹配DEX交易数：DEX有但决策日志没有的交易
   - 费用差异：实际费用 - 预期费用
   - 总资金费：从DEX数据中提取的资金费总和

**代码结构**：
```go
type dexValidationAnalyzer struct {
}

func (a *dexValidationAnalyzer) AnalyzeExecutionQuality(matches []*DecisionMatch) (*ExecutionQuality, error) {
    // 1. 计算滑点统计
    // 2. 计算执行延迟统计
    // 3. 计算分布
    // 4. 返回结果
}

func (a *dexValidationAnalyzer) CalculateDEXMetrics(matches []*DecisionMatch, unmatchedDecisions []*DecisionRecord, unmatchedDEXTrades []ExchangeFill) (*DEXValidationMetrics, error) {
    // 1. 计算决策执行率
    // 2. 统计未匹配数量
    // 3. 计算费用差异
    // 4. 提取资金费
    // 5. 返回结果
}
```

#### 3.2.10 Metrics - Violation Metrics (metrics/violation_metrics.go)

**职责**：计算规则违反指标

**接口设计**：
```go
type ViolationMetricsCalculator interface {
    Calculate(violations []Violation, totalDecisions int) (*RuleViolationMetrics, error)
}

type RuleViolationMetrics struct {
    HardConstraintViolationRate     float64
    RiskControlViolationRate        float64
    PositionManagementViolationRate float64
    TotalViolations                 int
    ViolationSeverityDistribution   map[string]int
}
```

**实现要点**：
1. **指标计算**：
   - 硬约束违反率：违反硬约束的决策数 / 总决策数
   - 风险控制违反率：违反风险控制规则的决策数 / 总决策数
   - 持仓管理违反率：违反持仓管理规则的决策数 / 总持仓决策数
   - 规则违反严重程度分布：Critical/High/Medium/Low分布
2. **统计逻辑**：
   - 按规则类型分组统计
   - 按严重程度分组统计

**代码结构**：
```go
type violationMetricsCalculator struct {
}

func (c *violationMetricsCalculator) Calculate(violations []Violation, totalDecisions int) (*RuleViolationMetrics, error) {
    // 1. 按类型统计违反次数
    // 2. 按严重程度统计分布
    // 3. 计算违反率
    // 4. 返回结果
}
```

#### 3.2.11 Reporter - Markdown Reporter (reporter/markdown_reporter.go)

**职责**：生成Markdown格式的复盘报告

**接口设计**：
```go
type Reporter interface {
    GenerateReport(review *ReviewResult) (string, error) // 返回报告文件路径
}
```

**实现要点**：
1. **报告结构**（按需求文档）：
   - 执行摘要（Executive Summary）
   - 表现概览（Performance Overview）
   - 详细分析（Detailed Analysis）
     - 错误分析
     - 成功模式
     - 典型案例
     - 关联分析
     - 标准化指标分析
   - 风险分析（Risk Analysis）
   - 改进建议（Recommendations）
   - 数据附录（Data Appendix）
2. **模板引擎**：
   - 使用 `text/template` 生成报告
   - 模板文件存储在 `reporter/templates/` 目录
   - 支持模板变量替换
3. **报告存储**：
   - 保存到 `data/reviews/` 目录
   - 文件名：`review_{trader_id}_{YYYYMMDD_HHMMSS}.md`
   - 使用缓冲写入，提高性能

**代码结构**：
```go
type markdownReporter struct {
    templateDir string
    reportDir   string
}

func (r *markdownReporter) GenerateReport(review *ReviewResult) (string, error) {
    // 1. 加载模板
    // 2. 渲染报告
    // 3. 保存到文件
    // 4. 返回文件路径
}
```

#### 3.2.12 Repository (repository.go)

**职责**：复盘记录的数据访问层

**接口设计**：
```go
type ReviewRepository interface {
    Save(ctx context.Context, record *ReviewRecord) error
    FindByID(ctx context.Context, id int64) (*ReviewRecord, error)
    FindByTraderID(ctx context.Context, traderID string, limit int) ([]*ReviewRecord, error)
    FindByTimeRange(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*ReviewRecord, error)
}
```

**实现要点**：
1. **数据库表设计**：
   - 表名：`review_records`
   - 字段：ID, TraderID, StartTime, EndTime, ReportPath, Summary, Metrics, TotalTrades, TotalPnL, WinRate, ErrorCount, Status, ErrorMessage, CreatedAt
   - 索引：trader_id, start_time, end_time
2. **数据存储策略**：
   - 报告文件：存储在文件系统（`data/reviews/`）
   - 元数据：存储在数据库（`review_records`表）
   - 指标数据：存储在数据库（JSON字段）
3. **数据库迁移**：
   - 需要设计迁移脚本
   - 考虑索引设计（按trader_id、start_time、end_time索引）

**代码结构**：
```go
type reviewRepository struct {
    db *sql.DB
}

func (r *reviewRepository) Save(ctx context.Context, record *ReviewRecord) error {
    // 1. 序列化 Summary 和 Metrics 为 JSON
    // 2. 插入数据库
    // 3. 返回错误
}
```

#### 3.2.13 Models (models.go)

**职责**：定义所有数据模型

**主要模型**：
```go
// ReviewRecord 复盘记录
type ReviewRecord struct {
    ID              int64     `json:"id" db:"id"`
    TraderID        string    `json:"trader_id" db:"trader_id"`
    StartTime       time.Time `json:"start_time" db:"start_time"`
    EndTime         time.Time `json:"end_time" db:"end_time"`
    ReportPath      string    `json:"report_path" db:"report_path"`
    Summary         string    `json:"summary" db:"summary"` // JSON
    Metrics         string    `json:"metrics" db:"metrics"` // JSON
    TotalTrades     int       `json:"total_trades" db:"total_trades"`
    TotalPnL        float64   `json:"total_pnl" db:"total_pnl"`
    WinRate         float64   `json:"win_rate" db:"win_rate"`
    ErrorCount      int       `json:"error_count" db:"error_count"`
    Status          string    `json:"status" db:"status"` // success/partial/failed
    ErrorMessage    string    `json:"error_message" db:"error_message"`
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
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
    Performance     *PerformanceAnalysis
    RuleViolations  *RuleViolationMetrics
    DEXValidation   *DEXValidationMetrics
}

// DecisionMatch 决策与DEX交易的匹配结果
type DecisionMatch struct {
    DecisionID        string
    DEXTradeID        string
    MatchMethod       string // "order_id" | "time_window" | "quantity_price"
    MatchConfidence   float64
    Slippage          float64
    ExecutionDelay    int64
    FeeDifference     float64
    QuantityDifference float64
}

// Violation 规则违反
type Violation struct {
    RuleID      string
    RuleName    string
    Type        string
    Severity    string
    Description string
    DecisionID  string
    Details     map[string]interface{}
}
```

---

## 四、项目计划

### 4.1 时间线概览

**Phase 1 (MVP)**: 2-3周

| 周次 | 主要任务 | 交付物 |
|------|---------|--------|
| Week 1 | 数据收集 + 基础分析 | 数据收集模块、基础分析模块 |
| Week 2 | 规则引擎 + 报告生成 | 规则引擎、报告生成模块 |
| Week 3 | 定时调度 + 测试优化 | 调度器、测试用例、文档 |

### 4.2 详细任务分解

#### Week 1: 数据收集 + 基础分析

**Day 1-2: 项目初始化**
- [ ] 创建 `review/` 目录结构
- [ ] 定义接口和模型（models.go）
- [ ] 设计数据库表结构（review_records表）
- [ ] 编写数据库迁移脚本

**Day 3-4: 数据收集模块**
- [ ] 增强 `logger/decision_logger.go`：实现 `GetRecordsByTimeRange()`
- [ ] 实现 `collector/decision_log.go`：决策日志收集器
- [ ] 实现 `collector/trade_history.go`：交易历史收集器
- [ ] 实现 `collector/dex_data.go`：DEX数据收集器（含重试机制）
- [ ] 单元测试：数据收集模块

**Day 5: 数据匹配模块**
- [ ] 实现 `matcher/decision_matcher.go`：决策与DEX交易匹配
- [ ] 实现多级匹配策略（OrderID > 时间窗口 > 数量价格）
- [ ] 单元测试：数据匹配模块

**Day 6-7: 基础分析模块**
- [ ] 实现 `analyzer/performance.go`：交易表现分析
- [ ] 集成 `trade_analytics` 模块
- [ ] 单元测试：表现分析模块

#### Week 2: 规则引擎 + 报告生成

**Day 8-10: 规则引擎**
- [ ] 设计规则引擎接口和插件化架构
- [ ] 实现核心规则检查器：
  - `RiskRewardRatioChecker`：风险回报比检查
  - `LeverageLimitChecker`：杠杆限制检查
  - `StopLossDirectionChecker`：止损方向检查
  - `ConfidenceRequirementChecker`：置信度检查
  - `StopLossWidthChecker`：止损宽度检查
  - `AccountRiskChecker`：账户风险检查
- [ ] 实现 `analyzer/rule_engine.go`：规则引擎主逻辑
- [ ] 实现 `metrics/violation_metrics.go`：规则违反指标计算
- [ ] 单元测试：规则引擎

**Day 11-12: DEX验证分析**
- [ ] 实现 `analyzer/dex_validation.go`：DEX验证分析
- [ ] 实现 `metrics/dex_validation.go`：DEX验证指标计算
- [ ] 单元测试：DEX验证模块

**Day 13-14: 报告生成**
- [ ] 设计Markdown报告模板
- [ ] 实现 `reporter/markdown_reporter.go`：报告生成器
- [ ] 实现报告模板（执行摘要、表现概览、详细分析等）
- [ ] 单元测试：报告生成模块

#### Week 3: 定时调度 + 测试优化

**Day 15-16: 复盘服务主逻辑**
- [ ] 实现 `service.go`：复盘服务主逻辑（编排）
- [ ] 实现错误处理和容错机制
- [ ] 实现性能监控
- [ ] 集成测试：完整复盘流程

**Day 17: 定时调度器**
- [ ] 实现 `scheduler.go`：定时调度器
- [ ] 集成到 `AutoTrader.Run()`
- [ ] 配置管理：添加复盘相关配置项

**Day 18-19: Repository + 数据库**
- [ ] 实现 `repository.go`：数据访问层
- [ ] 实现数据库迁移脚本
- [ ] 集成测试：数据持久化

**Day 20-21: 测试与优化**
- [ ] 性能测试：验证<5分钟性能目标
- [ ] 容错测试：验证错误处理机制
- [ ] 集成测试：端到端测试
- [ ] 代码审查和优化
- [ ] 文档完善

### 4.3 关键里程碑

| 里程碑 | 时间 | 验收标准 |
|--------|------|---------|
| M1: 数据收集完成 | Week 1 Day 5 | 能够收集决策日志、交易历史、DEX数据 |
| M2: 基础分析完成 | Week 1 Day 7 | 能够分析交易表现、计算基础指标 |
| M3: 规则引擎完成 | Week 2 Day 10 | 能够检查硬约束违反、计算违反指标 |
| M4: 报告生成完成 | Week 2 Day 14 | 能够生成完整的Markdown报告 |
| M5: 系统集成完成 | Week 3 Day 17 | 能够自动执行复盘、保存结果 |
| M6: 测试完成 | Week 3 Day 21 | 通过所有测试、性能达标 |

### 4.4 风险与应对

| 风险 | 影响 | 应对措施 |
|------|------|---------|
| DEX数据拉取失败 | 高 | 实现重试机制，失败时停止复盘 |
| 性能不达标 | 中 | 并行处理、缓存机制、性能监控 |
| 规则引擎复杂度高 | 中 | 分阶段实现，先实现核心规则 |
| 数据匹配不准确 | 中 | 多级匹配策略，记录置信度 |
| 决策JSON解析失败 | 低 | 宽松解析，提供默认值 |

---

## 五、技术规范

### 5.1 代码规范

1. **命名规范**：
   - 接口：`XxxInterface` 或 `Xxx`（推荐）
   - 实现：`xxx`（小写开头）
   - 结构体：`Xxx`（大写开头）
   - 方法：`Xxx`（大写开头，公开方法）

2. **错误处理**：
   - 使用 `fmt.Errorf` 包装错误
   - 错误信息要清晰，包含上下文
   - 区分致命错误、部分错误、警告

3. **日志记录**：
   - 使用 `log.Printf` 记录关键操作
   - 错误日志包含错误详情
   - 性能日志记录各阶段耗时

### 5.2 性能规范

1. **性能目标**：
   - 总体性能：< 5分钟
   - 数据收集：< 30秒
   - DEX数据拉取：< 60秒（正常），< 180秒（含重试）
   - 分析处理：< 30秒
   - 报告生成：< 30秒

2. **优化策略**：
   - 并行处理：决策日志文件并行读取（goroutine pool）
   - 缓存机制：缓存已解析的决策JSON
   - 批量处理：规则检查使用批量处理
   - 性能监控：记录各阶段耗时

### 5.3 测试规范

1. **单元测试**：
   - 覆盖率目标：> 80%
   - 关键模块：规则引擎、数据匹配、指标计算

2. **集成测试**：
   - 完整复盘流程测试
   - 错误处理测试
   - 性能测试

3. **测试数据**：
   - 使用真实数据样本
   - 创建测试用的决策日志文件
   - 模拟DEX数据拉取

### 5.4 配置规范

1. **配置文件扩展**：
   ```json
   {
     "review": {
       "enabled": true,
       "interval_hours": 6,
       "window_hours": 6,
       "first_run_delay_minutes": 0,
       "report_dir": "data/reviews",
       "dex_validation": {
         "enabled": true,
         "max_retries": 3,
         "retry_intervals_seconds": [1, 2, 4],
         "time_window_minutes": 10
       },
       "metrics": {
         "rule_violations": true,
         "dex_validation": true
       }
     }
   }
   ```

2. **配置验证**：
   - 启动时验证配置的有效性
   - 复盘窗口必须 > 0
   - 复盘间隔必须 >= 复盘窗口

---

## 六、测试策略

### 6.1 单元测试

#### 6.1.1 数据收集模块测试

**测试用例**：
- `TestDecisionLogCollector_Collect`: 测试决策日志收集
- `TestDecisionLogCollector_ParallelRead`: 测试并行读取性能
- `TestTradeHistoryCollector_Collect`: 测试交易历史收集
- `TestTradeHistoryCollector_BoundaryHandling`: 测试边界处理
- `TestDEXDataCollector_Collect`: 测试DEX数据收集
- `TestDEXDataCollector_Retry`: 测试重试机制

#### 6.1.2 数据匹配模块测试

**测试用例**：
- `TestDecisionMatcher_MatchByOrderID`: 测试OrderID匹配
- `TestDecisionMatcher_MatchByTimeWindow`: 测试时间窗口匹配
- `TestDecisionMatcher_MatchByQuantityPrice`: 测试数量价格匹配
- `TestDecisionMatcher_PartialMatch`: 测试部分匹配处理

#### 6.1.3 规则引擎测试

**测试用例**：
- `TestRuleEngine_CheckRiskRewardRatio`: 测试风险回报比检查
- `TestRuleEngine_CheckLeverageLimit`: 测试杠杆限制检查
- `TestRuleEngine_CheckStopLossDirection`: 测试止损方向检查
- `TestRuleEngine_CheckConfidenceRequirement`: 测试置信度检查
- `TestRuleEngine_MultipleViolations`: 测试多个违反情况

#### 6.1.4 指标计算测试

**测试用例**：
- `TestViolationMetricsCalculator_Calculate`: 测试规则违反指标计算
- `TestDEXValidationMetricsCalculator_Calculate`: 测试DEX验证指标计算

#### 6.1.5 报告生成测试

**测试用例**：
- `TestMarkdownReporter_GenerateReport`: 测试报告生成
- `TestMarkdownReporter_TemplateRendering`: 测试模板渲染

### 6.2 集成测试

#### 6.2.1 完整流程测试

**测试用例**：
- `TestReviewService_RunReview_Success`: 测试成功复盘流程
- `TestReviewService_RunReview_PartialError`: 测试部分错误处理
- `TestReviewService_RunReview_FatalError`: 测试致命错误处理

#### 6.2.2 性能测试

**测试用例**：
- `TestReviewService_Performance_DataCollection`: 测试数据收集性能
- `TestReviewService_Performance_Analysis`: 测试分析性能
- `TestReviewService_Performance_ReportGeneration`: 测试报告生成性能
- `TestReviewService_Performance_Total`: 测试总体性能（<5分钟）

#### 6.2.3 容错测试

**测试用例**：
- `TestReviewService_ErrorHandling_DEXDataFailure`: 测试DEX数据拉取失败
- `TestReviewService_ErrorHandling_DecisionLogCorruption`: 测试决策日志文件损坏
- `TestReviewService_ErrorHandling_DatabaseFailure`: 测试数据库失败

### 6.3 测试数据准备

1. **决策日志测试数据**：
   - 创建测试用的决策日志文件（不同时间、不同状态）
   - 包含正常决策、违反约束的决策、执行失败的决策

2. **交易历史测试数据**：
   - 使用现有数据库或创建测试数据库
   - 包含完整的交易记录（开仓+平仓）

3. **DEX数据测试数据**：
   - 模拟DEX数据拉取（mock provider）
   - 包含匹配和不匹配的交易

### 6.4 测试工具

1. **Mock对象**：
   - `MockDecisionLogReader`
   - `MockTradeHistoryReader`
   - `MockDEXDataProvider`
   - `MockTradeAnalyticsService`

2. **测试辅助函数**：
   - `createTestDecisionRecord()`
   - `createTestTradeRecord()`
   - `createTestExchangeFill()`

---

## 七、部署与运维

### 7.1 部署检查清单

- [ ] 数据库迁移脚本已执行（创建 `review_records` 表）
- [ ] 配置文件已更新（添加复盘相关配置）
- [ ] `data/reviews/` 目录已创建
- [ ] 日志目录权限正确
- [ ] 定时调度器已启动

### 7.2 监控指标

1. **性能指标**：
   - 复盘执行时间
   - 各阶段耗时
   - 数据收集耗时
   - 分析处理耗时
   - 报告生成耗时

2. **错误指标**：
   - 复盘失败次数
   - DEX数据拉取失败次数
   - 规则违反次数
   - 数据匹配失败次数

3. **业务指标**：
   - 复盘报告生成数量
   - 规则违反率趋势
   - DEX验证指标趋势

### 7.3 故障排查

1. **复盘失败**：
   - 检查日志：查看错误信息
   - 检查DEX数据拉取：是否网络问题
   - 检查数据库：是否连接问题

2. **性能问题**：
   - 检查各阶段耗时：定位瓶颈
   - 检查数据量：是否数据量过大
   - 检查系统资源：CPU、内存使用情况

3. **数据质量问题**：
   - 检查决策日志文件：是否损坏
   - 检查数据匹配：是否匹配不准确
   - 检查规则引擎：是否规则检查错误

---

## 八、后续扩展（Phase 2+）

### 8.1 Phase 2 功能

1. **深度错误分析**：
   - 持仓管理检查
   - 时机错误识别
   - 系统性偏差检测

2. **成功模式提取**：
   - 市场条件组合分析
   - 信号类型分析
   - 策略维度分析

3. **执行质量分析**：
   - 滑点分析
   - 时间差分析
   - 费用差异分析

4. **决策方向后验指标**：
   - 方向准确率计算
   - 方向准确度分析
   - 方向一致性评估

### 8.2 Phase 3 功能

1. **AI遵循度指标**：
   - Prompt规则遵循率计算
   - 规则遵循度分布分析
   - 自我纠正有效性评估

2. **实时预警**：
   - 严重错误实时通知
   - 风险指标异常预警

3. **历史对比**：
   - 多时间段对比分析
   - 趋势识别

### 8.3 扩展点设计

1. **规则引擎扩展**：
   - 支持动态注册规则
   - 支持规则优先级调整
   - 支持规则配置化

2. **报告模板扩展**：
   - 支持自定义报告模板
   - 支持多格式输出（HTML、PDF等）

3. **数据源扩展**：
   - 支持其他DEX数据源
   - 支持市场数据集成

---

## 九、总结

### 9.1 关键设计决策

1. **依赖注入设计**：通过接口依赖，便于测试和替换实现
2. **插件化规则引擎**：支持规则的动态注册和扩展
3. **多级匹配策略**：提高决策与DEX交易的匹配准确率
4. **容错设计**：区分致命/部分/警告错误，优雅降级
5. **性能优化**：并行处理、缓存机制、批量处理

### 9.2 实施建议

1. **分阶段实施**：先实现核心功能（Phase 1），再逐步扩展
2. **测试驱动**：先编写测试用例，再实现功能
3. **性能监控**：记录各阶段耗时，及时优化
4. **文档完善**：及时更新文档，记录设计决策

### 9.3 成功标准

- ✅ 能够每6小时自动执行复盘
- ✅ 能够识别硬约束违反
- ✅ 能够生成完整的Markdown报告
- ✅ 性能目标：< 5分钟
- ✅ 测试覆盖率：> 80%

---

**文档结束**

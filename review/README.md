# 复盘模块 (Review Module)

## 概述

复盘模块实现了自动复盘功能，每6小时自动复盘一次，系统性地检讨过去6小时内的交易历史。

## 架构说明

为了避免循环导入问题，`review`包使用接口依赖注入模式。子包（analyzer、collector、matcher等）可以导入父包`review`使用类型定义，但`review/service.go`只使用接口而不直接导入子包。

## 使用方法

### 创建复盘服务实例

由于Go的包循环导入限制，需要在主程序中创建所有子组件，然后传递给`NewReviewService`：

```go
import (
    "nofx/review"
    "nofx/review/analyzer"
    "nofx/review/collector"
    "nofx/review/matcher"
    "nofx/review/metrics"
    "nofx/review/reporter"
    "nofx/logger"
)

// 创建依赖
decisionLogger := logger.NewDecisionLogger("decision_logs")
decisionLogAdapter := review.NewDecisionLoggerAdapter(decisionLogger)
tradeHistoryReader := ... // TradeHistoryReader实例
dexDataProvider := ... // DEXDataProvider实例
tradeAnalyticsService := ... // TradeAnalyticsService实例
repo := review.NewReviewRepository(db)

// 创建子组件
decisionLogCollector := collector.NewDecisionLogCollector(decisionLogAdapter)
tradeHistoryCollector := collector.NewTradeHistoryCollector(tradeHistoryReader)
dexDataCollector := collector.NewDEXDataCollector(dexDataProvider)
ruleEngine := analyzer.NewRuleEngine()
performanceAnalyzer := analyzer.NewPerformanceAnalyzer(tradeAnalyticsService)
dexValidationAnalyzer := analyzer.NewDEXValidationAnalyzer()
decisionMatcher := matcher.NewDecisionMatcher(10*time.Minute, 0.05, 0.01)
violationMetricsCalc := metrics.NewViolationMetricsCalculator()
reporter := reporter.NewMarkdownReporter("data/reviews")

// 创建复盘服务
reviewService := review.NewReviewService(
    decisionLogCollector,
    tradeHistoryCollector,
    dexDataCollector,
    ruleEngine,
    performanceAnalyzer,
    dexValidationAnalyzer,
    decisionMatcher,
    violationMetricsCalc,
    reporter,
    repo,
)

// 创建调度器
scheduler := review.NewReviewScheduler(
    reviewService,
    review.ReviewSchedulerConfig{
        Interval:      6 * time.Hour,
        FirstRunDelay: 0,
        Enabled:       true,
    },
)

// 启动调度器
go scheduler.Start(ctx, traderID)
```

## 模块结构

```
review/
├── models.go              # 数据模型定义
├── interfaces.go          # 接口定义
├── adapter.go             # DecisionLogger适配器
├── service.go             # 复盘服务主逻辑（使用接口，不导入子包）
├── scheduler.go           # 定时调度器
├── repository.go          # 数据访问层
├── collector/             # 数据收集层
│   ├── decision_log.go
│   ├── trade_history.go
│   └── dex_data.go
├── matcher/               # 数据匹配层
│   └── decision_matcher.go
├── analyzer/              # 分析层
│   ├── rule_engine.go
│   ├── performance.go
│   └── dex_validation.go
├── metrics/               # 指标计算层
│   └── violation_metrics.go
└── reporter/              # 报告生成层
    └── markdown_reporter.go
```

## 数据库迁移

数据库迁移已添加到`config/database.go`中，包括：
- `review_records`表创建
- 索引创建（trader_id, start_time, end_time等）

## 注意事项

1. **循环导入**：子包（analyzer、collector等）可以导入父包`review`使用类型，但`service.go`使用接口避免循环导入
2. **实例化**：需要在主程序中创建所有子组件实例，然后传递给`NewReviewService`
3. **配置**：从配置文件读取复盘相关配置（间隔、首次延迟等）


# Trade Analytics 集成需求与设计文档

## 1. 需求概述

### 1.1 需求目标

将 `trade_analytics` 模块集成到决策流程中，为 user prompt 提供连续亏损次数追踪数据，并作为未来渐进集成 trade history 数据的基础架构。

### 1.2 业务背景

**当前问题**：
- System Prompt 要求：`连续2次亏损 → 暂停交易（使用"wait"动作）约30分钟（10个决策周期）`
- User Prompt 现状：无法提供连续亏损次数，AI 无法判断是否触发熔断机制
- 数据源现状：
  - `logger.PerformanceAnalysis`：不提供连续亏损次数
  - `trade_analytics.StreakStatistics`：提供连续亏损次数，但未集成到决策流程

**解决方案**：
- 集成 `trade_analytics` 模块到决策流程
- 在 `buildContext` 中获取连续亏损数据
- 在 user prompt 中展示连续亏损信息
- 设计通用架构，支持未来扩展更多 trade history 数据

### 1.3 需求范围

**本次实现**：
- ✅ 集成 `trade_analytics` Service 到 AutoTrader
- ✅ 获取连续亏损次数（`StreakStatistics.CurrentStreak`）
- ✅ 在 user prompt 中展示连续亏损信息
- ✅ 设计通用集成架构，支持未来扩展

**未来扩展**（不在本次范围）：
- 账户回撤百分比（从 RiskMetrics 获取）
- 更多风险指标（Sharpe、最大回撤等）
- 交易频率统计
- 币种表现统计

---

## 2. 现状分析

### 2.1 数据源对比

| 模块 | 数据源 | 连续亏损追踪 | 集成状态 | 数据准确性 |
|------|--------|------------|---------|-----------|
| **logger.PerformanceAnalysis** | 决策日志文件 | ❌ 不支持 | ✅ 已集成 | 基于决策日志，可能不完整 |
| **trade_analytics.StreakStatistics** | trade_history 数据库 | ✅ 支持 | ❌ 未集成 | 基于真实交易记录，更准确 |

### 2.2 当前架构

```
决策流程：
AutoTrader.buildContext()
  └─> logger.AnalyzePerformance(100)
      └─> PerformanceAnalysis (无连续亏损)
          └─> ctx.Performance
              └─> buildUserPrompt()
                  └─> 显示 Sharpe 比率（仅此一项）

trade_analytics 模块：
API Server
  └─> trade_analytics.Service
      └─> Analyzer.CalculateStreakStats()
          └─> StreakStatistics (有连续亏损)
              └─> 仅用于 API 展示，未集成到决策流程
```

### 2.3 问题分析

1. **数据源分离**：
   - 决策流程使用 `logger.PerformanceAnalysis`（基于决策日志）
   - 分析展示使用 `trade_analytics`（基于 trade_history）
   - 两者数据源不同，可能导致不一致

2. **功能缺失**：
   - `PerformanceAnalysis` 缺少连续亏损追踪
   - System Prompt 要求的功能无法实现

3. **架构限制**：
   - `trade_analytics` 服务仅在 API 层初始化
   - AutoTrader 无法访问 `trade_analytics` 服务

---

## 3. 设计方案

### 3.1 设计原则

1. **渐进式集成**：先集成连续亏损，后续逐步扩展
2. **向后兼容**：不影响现有功能，trade_analytics 不可用时优雅降级
3. **通用架构**：设计可扩展的接口，支持未来添加更多指标
4. **数据一致性**：优先使用 trade_history 数据（更准确）

### 3.2 架构设计

#### 3.2.1 服务层集成

```
TraderManager
  ├─> tradeHistoryService (已有)
  └─> tradeAnalyticsService (新增)
      └─> 管理 trade_analytics.Service 实例
          └─> 提供给 AutoTrader 使用

AutoTrader
  ├─> tradeHistoryService (已有)
  └─> tradeAnalyticsService (新增，可选)
      └─> 在 buildContext 中调用
          └─> 获取 StreakStatistics
              └─> 传递给 Context
```

#### 3.2.2 数据流设计

```
决策周期开始
  └─> AutoTrader.buildContext()
      ├─> logger.AnalyzePerformance(100)  [现有]
      │   └─> PerformanceAnalysis
      │       └─> ctx.Performance
      │
      └─> tradeAnalyticsService.GetStreakStats()  [新增]
          └─> StreakStatistics
              └─> ctx.StreakStats (新增字段)
                  └─> buildUserPrompt()
                      └─> 显示连续亏损信息
```

### 3.3 接口设计

#### 3.3.1 扩展 trade_analytics.Service 接口

在 `trade_analytics.Service` 中添加 `GetStreakStats` 方法：

```go
// Service 交易分析服务接口
type Service interface {
    // ... 现有方法 ...
    
    // GetStreakStats 获取连续统计（新增）
    GetStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error)
}
```

#### 3.3.2 扩展 Context 结构

在 `decision.Context` 中添加 `StreakStats` 字段：

```go
type Context struct {
    // ... 现有字段 ...
    Performance   interface{}  `json:"-"` // 历史表现分析（logger.PerformanceAnalysis）
    StreakStats   interface{}  `json:"-"` // 连续统计（trade_analytics.StreakStatistics，可选）
}
```

#### 3.3.3 扩展 AutoTrader 结构

在 `AutoTrader` 中添加 `tradeAnalyticsService` 字段：

```go
type AutoTrader struct {
    // ... 现有字段 ...
    tradeHistoryService   trade_history.Service  // 交易历史服务（可选）
    tradeAnalyticsService trade_analytics.Service // 交易分析服务（可选，新增）
}
```

### 3.4 实现步骤

#### 步骤1：扩展 trade_analytics.Service 接口

**文件**：`trade_analytics/service.go`

```go
// Service 交易分析服务接口
type Service interface {
    // ... 现有方法 ...
    
    // GetStreakStats 获取连续统计
    GetStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error)
}

// GetStreakStats 获取连续统计
func (s *service) GetStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error) {
    return s.analyzer.CalculateStreakStats(ctx, filter)
}
```

#### 步骤2：在 TraderManager 中管理 trade_analytics Service

**文件**：`manager/trader_manager.go`

```go
type TraderManager struct {
    // ... 现有字段 ...
    tradeHistoryService   trade_history.Service
    tradeAnalyticsService trade_analytics.Service // 新增
}

// SetTradeAnalyticsService 设置交易分析服务
func (tm *TraderManager) SetTradeAnalyticsService(service trade_analytics.Service)

// GetTradeAnalyticsService 获取交易分析服务
func (tm *TraderManager) GetTradeAnalyticsService() trade_analytics.Service
```

#### 步骤3：在 API Server 中初始化并设置

**文件**：`api/server.go`

```go
// setupTradeAnalyticsRoutes 设置交易分析路由
func (s *Server) setupTradeAnalyticsRoutes(router *gin.RouterGroup) {
    // ... 现有初始化代码 ...
    
    analyticsService := trade_analytics.NewService(...)
    
    // 设置到 TraderManager（新增）
    s.traderManager.SetTradeAnalyticsService(analyticsService)
    
    // ... 注册路由 ...
}
```

#### 步骤4：在 AutoTrader 中集成

**文件**：`trader/auto_trader.go`

```go
// NewAutoTrader 创建自动交易器
func NewAutoTrader(config AutoTraderConfig, tradeHistoryService trade_history.Service, tradeAnalyticsService trade_analytics.Service) (*AutoTrader, error) {
    // ... 现有代码 ...
    
    return &AutoTrader{
        // ... 现有字段 ...
        tradeAnalyticsService: tradeAnalyticsService, // 新增
    }, nil
}

// buildContext 构建交易上下文
func (at *AutoTrader) buildContext() (*decision.Context, error) {
    // ... 现有代码 ...
    
    // 获取连续统计（新增）
    var streakStats interface{}
    if at.tradeAnalyticsService != nil {
        filter := &trade_analytics.AnalyticsFilter{
            TraderID: at.id,
            // 不设置时间范围，使用所有历史数据
        }
        stats, err := at.tradeAnalyticsService.GetStreakStats(context.Background(), filter)
        if err == nil {
            streakStats = stats
        } else {
            log.Printf("⚠️  获取连续统计失败: %v", err)
        }
    }
    
    ctx := &decision.Context{
        // ... 现有字段 ...
        StreakStats: streakStats, // 新增
    }
    
    return ctx, nil
}
```

#### 步骤5：在 user prompt 中展示

**文件**：`decision/engine.go`

```go
// buildUserPrompt 构建 User Prompt（动态数据）
func buildUserPrompt(ctx *Context) string {
    // ... 现有代码 ...
    
    // 连续亏损信息（新增）
    if ctx.StreakStats != nil {
        type StreakData struct {
            CurrentStreak     int    `json:"current_streak"`
            CurrentStreakType string `json:"current_streak_type"`
            LongestLosingStreak int  `json:"longest_losing_streak"`
        }
        var streakData StreakData
        if jsonData, err := json.Marshal(ctx.StreakStats); err == nil {
            if err := json.Unmarshal(jsonData, &streakData); err == nil {
                if streakData.CurrentStreakType == "losing" {
                    consecutiveLosses := -streakData.CurrentStreak // CurrentStreak 为负数表示连亏
                    sb.WriteString("## 🛡️ 熔断机制状态\n\n")
                    sb.WriteString(fmt.Sprintf("连续亏损: %d次", consecutiveLosses))
                    if consecutiveLosses >= 2 {
                        sb.WriteString(" ⚠️ **已触发熔断机制**（应暂停交易约30分钟）\n")
                    } else {
                        sb.WriteString("\n")
                    }
                    sb.WriteString(fmt.Sprintf("最长连亏: %d次\n\n", streakData.LongestLosingStreak))
                }
            }
        }
    }
    
    // ... 现有代码 ...
}
```

#### 步骤6：更新 TraderManager 的 LoadSingleTrader

**文件**：`manager/trader_manager.go`

```go
// loadSingleTrader 加载单个交易员
func (tm *TraderManager) loadSingleTrader(...) error {
    // ... 现有代码 ...
    
    // 创建trader实例（传递交易历史服务和交易分析服务）
    at, err := trader.NewAutoTrader(traderConfig, tm.tradeHistoryService, tm.tradeAnalyticsService)
    
    // ... 现有代码 ...
}
```

---

## 4. 技术细节

### 4.1 错误处理策略

**原则**：优雅降级，不影响主流程

```go
// 获取连续统计
var streakStats interface{}
if at.tradeAnalyticsService != nil {
    stats, err := at.tradeAnalyticsService.GetStreakStats(ctx, filter)
    if err == nil {
        streakStats = stats
    } else {
        log.Printf("⚠️  获取连续统计失败: %v", err)
        // 不设置 streakStats，user prompt 中不会显示
    }
}
```

**场景处理**：
- `tradeAnalyticsService == nil`：不显示连续亏损信息（向后兼容）
- `GetStreakStats` 失败：记录日志，不显示连续亏损信息
- `trade_history` 数据为空：返回空的 `StreakStatistics`（CurrentStreak = 0）

### 4.2 性能考虑

**查询优化**：
- `GetStreakStats` 只查询有 PnL 的记录（已过滤）
- 使用索引：`trade_history` 表的 `trader_id` 和 `timestamp` 索引
- 不设置时间范围，但数据库查询会使用索引优化

**缓存策略**（未来优化）：
- 可以考虑缓存最近 N 个周期的连续统计
- 每个决策周期（3分钟）更新一次，缓存时间可设置为 1-2 分钟

### 4.3 数据一致性

**数据源优先级**：
1. **trade_history**（优先）：基于真实交易记录，更准确
2. **决策日志**（降级）：如果 trade_history 不可用，可考虑从决策日志计算

**数据同步**：
- `trade_history` 数据在交易执行时实时写入
- 决策日志在决策周期结束时写入
- 可能存在短暂延迟（< 1秒），但影响可忽略

### 4.4 扩展性设计

**未来扩展点**：

1. **更多熔断指标**：
   ```go
   type CircuitBreakerStatus struct {
       ConsecutiveLosses int     // 连续亏损次数
       AccountDrawdown   float64 // 账户回撤百分比
       SharpeRatio       float64 // 夏普比率
       // ... 未来可扩展
   }
   ```

2. **统一数据接口**：
   ```go
   type PerformanceMetrics interface {
       GetConsecutiveLosses() int
       GetAccountDrawdown() float64
       GetSharpeRatio() float64
       // ... 统一接口
   }
   ```

3. **数据聚合层**：
   ```go
   type PerformanceAggregator struct {
       loggerAnalysis    *logger.PerformanceAnalysis
       analyticsData     *trade_analytics.TradeAnalytics
       // 聚合多个数据源
   }
   ```

---

## 5. 实现计划

### 5.1 阶段划分

#### 阶段1：基础集成（本次实现）
- ✅ 扩展 `trade_analytics.Service` 接口
- ✅ 在 `TraderManager` 中管理服务
- ✅ 在 `AutoTrader` 中集成服务
- ✅ 在 `Context` 中添加字段
- ✅ 在 user prompt 中展示连续亏损

#### 阶段2：数据完善（未来）
- 账户回撤百分比（从 `RiskMetrics.MaxDrawdownPercent` 获取）
- 更多风险指标展示
- 数据聚合和统一接口

#### 阶段3：性能优化（未来）
- 缓存机制
- 查询优化
- 批量获取多个指标

### 5.2 测试计划

#### 单元测试
- `trade_analytics.Service.GetStreakStats` 方法测试
- `AutoTrader.buildContext` 集成测试
- `buildUserPrompt` 展示逻辑测试

#### 集成测试
- 完整决策流程测试（包含连续亏损展示）
- 错误处理测试（trade_analytics 不可用时的降级）
- 数据一致性测试（trade_history 数据准确性）

#### 边界测试
- 无交易记录时的处理
- 只有盈利记录时的处理
- 只有亏损记录时的处理
- 交易记录为空时的处理

---

## 6. 风险评估

### 6.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **trade_history 数据不完整** | 中等 | 优雅降级，不显示连续亏损信息 |
| **查询性能问题** | 低 | 使用索引，未来可添加缓存 |
| **服务初始化失败** | 低 | 向后兼容，不影响现有功能 |
| **数据源不一致** | 低 | 优先使用 trade_history（更准确） |

### 6.2 业务风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **连续亏损计算错误** | 高 | 充分测试，验证计算逻辑 |
| **熔断机制误触发** | 高 | 双重验证，记录详细日志 |
| **向后兼容性问题** | 低 | 所有新功能都是可选的 |

---

## 7. 验证方案

### 7.1 功能验证

**验证点1：连续亏损计算准确性**
- 创建测试数据：连续3次亏损
- 验证 `StreakStatistics.CurrentStreak` = -3
- 验证 `CurrentStreakType` = "losing"

**验证点2：User Prompt 展示**
- 验证连续亏损信息出现在 user prompt 中
- 验证格式正确（"连续亏损: N次"）
- 验证触发熔断时的警告提示

**验证点3：错误处理**
- 验证 trade_analytics 不可用时优雅降级
- 验证查询失败时不影响主流程
- 验证无交易记录时的处理

### 7.2 性能验证

**验证点1：查询性能**
- 验证 `GetStreakStats` 查询时间 < 100ms（1000条记录）
- 验证不影响决策周期总时间（< 3秒）

**验证点2：内存使用**
- 验证不增加显著内存开销
- 验证服务实例可复用

### 7.3 集成验证

**验证点1：端到端流程**
- 完整决策周期：从 buildContext 到 user prompt
- 验证数据流正确传递
- 验证 AI 能正确识别连续亏损状态

---

## 8. 未来扩展规划

### 8.1 短期扩展（1-2个版本）

1. **账户回撤百分比**
   - 从 `RiskMetrics.MaxDrawdownPercent` 获取
   - 在 user prompt 中展示
   - 支持 System Prompt 的 `账户回撤 >25%` 检查

2. **统一熔断状态展示**
   - 创建独立的"熔断机制状态"章节
   - 集中展示所有熔断相关指标
   - 提供明确的触发状态

### 8.2 中期扩展（3-6个版本）

1. **数据聚合层**
   - 统一 `logger.PerformanceAnalysis` 和 `trade_analytics` 数据
   - 提供统一的性能指标接口
   - 支持多数据源融合

2. **缓存机制**
   - 缓存最近 N 个周期的统计结果
   - 减少数据库查询频率
   - 提升决策周期性能

### 8.3 长期扩展（6+个版本）

1. **完整迁移**
   - 逐步将 `logger.PerformanceAnalysis` 的功能迁移到 `trade_analytics`
   - 统一使用 trade_history 作为唯一数据源
   - 提升数据一致性和准确性

2. **实时监控**
   - 实时计算和更新性能指标
   - 支持流式数据处理
   - 提供实时预警功能

---

## 9. 总结

### 9.1 设计优势

1. **渐进式集成**：先实现核心功能，后续逐步扩展
2. **向后兼容**：不影响现有功能，优雅降级
3. **通用架构**：设计可扩展的接口，支持未来添加更多指标
4. **数据准确性**：使用 trade_history 作为数据源，更准确可靠

### 9.2 关键决策

1. **选择 trade_analytics 而非扩展 PerformanceAnalysis**：
   - 数据源更准确（基于真实交易记录）
   - 已有完整实现
   - 支持未来扩展更多指标

2. **设计为可选服务**：
   - 向后兼容（trade_analytics 不可用时不影响主流程）
   - 支持渐进式部署
   - 降低集成风险

3. **使用接口而非直接依赖**：
   - 便于测试（可 mock）
   - 支持未来替换实现
   - 降低耦合度

### 9.3 成功标准

- ✅ 连续亏损次数正确计算和展示
- ✅ System Prompt 的熔断机制可以正常工作
- ✅ 不影响现有功能（向后兼容）
- ✅ 性能影响可接受（< 100ms 额外开销）
- ✅ 代码可维护和扩展

---

## 附录

### A. 相关文件清单

**需要修改的文件**：
1. `trade_analytics/service.go` - 添加 `GetStreakStats` 方法
2. `manager/trader_manager.go` - 管理 trade_analytics Service
3. `api/server.go` - 初始化并设置服务
4. `trader/auto_trader.go` - 集成服务到 AutoTrader
5. `decision/engine.go` - 在 Context 中添加字段，在 user prompt 中展示

**需要新增的文件**：
- 无（所有修改都在现有文件中）

### B. 数据结构参考

**StreakStatistics**：
```go
type StreakStatistics struct {
    LongestWinningStreak int    `json:"longest_winning_streak"` // 最长连胜
    LongestLosingStreak  int    `json:"longest_losing_streak"`  // 最长连亏
    CurrentStreak        int    `json:"current_streak"`         // 当前连胜/连亏（正数=连胜，负数=连亏）
    CurrentStreakType    string `json:"current_streak_type"`    // "winning" 或 "losing"
}
```

**使用方式**：
- `CurrentStreakType == "losing"` 且 `CurrentStreak < 0`：表示连续亏损
- 连续亏损次数 = `-CurrentStreak`（取绝对值）

### C. 示例代码

**获取连续亏损**：
```go
if streakStats != nil {
    type StreakData struct {
        CurrentStreak     int    `json:"current_streak"`
        CurrentStreakType string `json:"current_streak_type"`
    }
    var data StreakData
    if jsonData, err := json.Marshal(streakStats); err == nil {
        if err := json.Unmarshal(jsonData, &data); err == nil {
            if data.CurrentStreakType == "losing" {
                consecutiveLosses := -data.CurrentStreak
                // 使用 consecutiveLosses
            }
        }
    }
}
```

---

**文档版本**：v1.0  
**创建日期**：2025-01-XX  
**最后更新**：2025-01-XX  
**作者**：AI Assistant  
**审核状态**：待审核


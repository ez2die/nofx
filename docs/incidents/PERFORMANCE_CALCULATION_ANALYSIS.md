# Performance 计算逻辑分析

## 概述

本系统通过分析交易日志记录来计算交易表现（performance）。核心逻辑在 `logger/decision_logger.go` 的 `AnalyzePerformance()` 函数中实现。

## 关键概念

### 1. Quantity 计算逻辑

**问题：quantity 是什么单位？**

根据代码分析：

```625:627:trader/auto_trader.go
quantity := decision.PositionSizeUSD / marketData.CurrentPrice
actionRecord.Quantity = quantity
actionRecord.Price = marketData.CurrentPrice
```

**结论：`quantity` 是币种的数量（枚/张），而非 USD 价值。**

- `PositionSizeUSD`：用 USD 表示的仓位价值
- `CurrentPrice`：当前币种价格
- `quantity = PositionSizeUSD / CurrentPrice`：计算出的币种数量

**示例**：
- 如果 `PositionSizeUSD = 100` USD
- `CurrentPrice = 50000` USDT
- 则 `quantity = 100 / 50000 = 0.002` BTC

### 2. PnL 计算逻辑

PnL（盈亏）计算有两个层面：**绝对盈亏**和**相对盈亏**。

#### 绝对盈亏（PnL，USDT）

```415:420:logger/decision_logger.go
var pnl float64
if side == "long" {
    pnl = quantity * (action.Price - openPrice)
} else {
    pnl = quantity * (openPrice - action.Price)
}
```

**关键点**：
- PnL = quantity × 价格差
- 杠杆**不影响**绝对盈亏
- 杠杆只影响保证金需求

**示例（Long）**：
- 开仓：1 BTC @ 50000 USDT，杠杆 10x
- 平仓：1 BTC @ 52000 USDT
- PnL = 1 × (52000 - 50000) = **+2000 USDT**

**示例（Short）**：
- 开仓：1 BTC @ 50000 USDT，杠杆 10x
- 平仓：1 BTC @ 48000 USDT
- PnL = 1 × (50000 - 48000) = **+2000 USDT**

#### 相对盈亏（PnLPct，%）

```422:428:logger/decision_logger.go
positionValue := quantity * openPrice
marginUsed := positionValue / float64(leverage)
pnlPct := 0.0
if marginUsed > 0 {
    pnlPct = (pnl / marginUsed) * 100
}
```

**关键点**：
- `positionValue = quantity × openPrice`：仓位价值
- `marginUsed = positionValue / leverage`：保证金使用
- `PnLPct = (PnL / marginUsed) × 100`：相对保证金的盈亏百分比

**示例（Long）**：
- 开仓：1 BTC @ 50000 USDT，杠杆 10x
- 平仓：1 BTC @ 52000 USDT
- positionValue = 1 × 50000 = 50000 USDT
- marginUsed = 50000 / 10 = 5000 USDT
- PnL = 2000 USDT
- PnLPct = (2000 / 5000) × 100 = **+40%**

**重要**：PnLPct 是相对于保证金的收益，考虑了杠杆效应。

### 3. 交易匹配逻辑

系统使用 **FIFO（先进先出）** 匹配开仓和平仓记录。

```336:374:logger/decision_logger.go
// 追踪持仓状态：symbol_side -> {side, openPrice, openTime, quantity, leverage}
openPositions := make(map[string]map[string]interface{})

// 为了避免开仓记录在窗口外导致匹配失败，需要先从所有历史记录中找出未平仓的持仓
// 获取更多历史记录来构建完整的持仓状态（使用更大的窗口）
allRecords, err := l.GetLatestRecords(lookbackCycles * 3) // 扩大3倍窗口
if err == nil && len(allRecords) > len(records) {
    // 先从扩大的窗口中收集所有开仓记录
    for _, record := range allRecords {
        for _, action := range record.Decisions {
            if !action.Success {
                continue
            }

            symbol := action.Symbol
            side := ""
            if action.Action == "open_long" || action.Action == "close_long" {
                side = "long"
            } else if action.Action == "open_short" || action.Action == "close_short" {
                side = "short"
            }
            posKey := symbol + "_" + side

            switch action.Action {
            case "open_long", "open_short":
                // 记录开仓
                openPositions[posKey] = map[string]interface{}{
                    "side":      side,
                    "openPrice": action.Price,
                    "openTime":  action.Timestamp,
                    "quantity":  action.Quantity,
                    "leverage":  action.Leverage,
                }
            case "close_long", "close_short":
                // 移除已平仓记录
                delete(openPositions, posKey)
            }
        }
    }
}
```

**关键点**：
- 使用 `symbol_side` 作为 key，区分同一币种的多空持仓
- 预填充窗口扩大 3 倍，避免开仓记录在分析窗口外
- FIFO 匹配确保历史交易正确关联

## 统计指标计算

### 1. 基础指标

```482:505:logger/decision_logger.go
// 计算统计指标
if analysis.TotalTrades > 0 {
    analysis.WinRate = (float64(analysis.WinningTrades) / float64(analysis.TotalTrades)) * 100

    // 计算总盈利和总亏损
    totalWinAmount := analysis.AvgWin   // 当前是累加的总和
    totalLossAmount := analysis.AvgLoss // 当前是累加的总和（负数）

    if analysis.WinningTrades > 0 {
        analysis.AvgWin /= float64(analysis.WinningTrades)
    }
    if analysis.LosingTrades > 0 {
        analysis.AvgLoss /= float64(analysis.LosingTrades)
    }

    // Profit Factor = 总盈利 / 总亏损（绝对值）
    // 注意：totalLossAmount 是负数，所以取负号得到绝对值
    if totalLossAmount != 0 {
        analysis.ProfitFactor = totalWinAmount / (-totalLossAmount)
    } else if totalWinAmount > 0 {
        // 只有盈利没有亏损的情况，设置为一个很大的值表示完美策略
        analysis.ProfitFactor = 999.0
    }
}
```

**计算公式**：
- **WinRate** = (WinningTrades / TotalTrades) × 100
- **AvgWin** = TotalWin / WinningTrades
- **AvgLoss** = TotalLoss / LosingTrades（注意：这里是负数）
- **ProfitFactor** = TotalWin / |TotalLoss|

**Profit Factor 解释**：
- > 1.0：策略盈利
- = 1.0：盈亏平衡
- < 1.0：策略亏损
- 999.0：只有盈利无亏损（完美策略）

### 2. 夏普比率（Sharpe Ratio）

```546:612:logger/decision_logger.go
// calculateSharpeRatio 计算夏普比率
// 基于账户净值的变化计算风险调整后收益
func (l *DecisionLogger) calculateSharpeRatio(records []*DecisionRecord) float64 {
    if len(records) < 2 {
        return 0.0
    }

    // 提取每个周期的账户净值
    // 注意：TotalBalance字段实际存储的是TotalEquity（账户总净值）
    // TotalUnrealizedProfit字段实际存储的是TotalPnL（相对初始余额的盈亏）
    var equities []float64
    for _, record := range records {
        // 直接使用TotalBalance，因为它已经是完整的账户净值
        equity := record.AccountState.TotalBalance
        if equity > 0 {
            equities = append(equities, equity)
        }
    }

    if len(equities) < 2 {
        return 0.0
    }

    // 计算周期收益率（period returns）
    var returns []float64
    for i := 1; i < len(equities); i++ {
        if equities[i-1] > 0 {
            periodReturn := (equities[i] - equities[i-1]) / equities[i-1]
            returns = append(returns, periodReturn)
        }
    }

    if len(returns) == 0 {
        return 0.0
    }

    // 计算平均收益率
    sumReturns := 0.0
    for _, r := range returns {
        sumReturns += r
    }
    meanReturn := sumReturns / float64(len(returns))

    // 计算收益率标准差
    sumSquaredDiff := 0.0
    for _, r := range returns {
        diff := r - meanReturn
        sumSquaredDiff += diff * diff
    }
    variance := sumSquaredDiff / float64(len(returns))
    stdDev := math.Sqrt(variance)

    // 避免除以零
    if stdDev == 0 {
        if meanReturn > 0 {
            return 999.0 // 无波动的正收益
        } else if meanReturn < 0 {
            return -999.0 // 无波动的负收益
        }
        return 0.0
    }

    // 计算夏普比率（假设无风险利率为0）
    // 注：直接返回周期级别的夏普比率（非年化），正常范围 -2 到 +2
    sharpeRatio := meanReturn / stdDev
    return sharpeRatio
}
```

**计算公式**：
- 周期收益率：`return = (equity[t] - equity[t-1]) / equity[t-1]`
- 平均收益率：`meanReturn = mean(returns)`
- 收益率标准差：`stdDev = sqrt(variance(returns))`
- 夏普比率：`sharpeRatio = meanReturn / stdDev`

**夏普比率解释**：
- > 1.0：良好的风险调整后收益
- 0-1.0：一般表现
- < 0：风险调整后亏损
- ±999.0：无波动（只有收益或只有亏损）

**注意**：这里计算的是周期级别的夏普比率，非年化版本。

### 3. 币种级别统计

```507:524:logger/decision_logger.go
// 计算各币种胜率和平均盈亏
bestPnL := -999999.0
worstPnL := 999999.0
for symbol, stats := range analysis.SymbolStats {
    if stats.TotalTrades > 0 {
        stats.WinRate = (float64(stats.WinningTrades) / float64(stats.TotalTrades)) * 100
        stats.AvgPnL = stats.TotalPnL / float64(stats.TotalTrades)

        if stats.TotalPnL > bestPnL {
            bestPnL = stats.TotalPnL
            analysis.BestSymbol = symbol
        }
        if stats.TotalPnL < worstPnL {
            worstPnL = stats.TotalPnL
            analysis.WorstSymbol = symbol
        }
    }
}
```

每个币种独立统计：
- **TotalTrades**：交易次数
- **WinningTrades / LosingTrades**：盈利/亏损次数
- **WinRate**：胜率
- **TotalPnL**：总盈亏（USDT）
- **AvgPnL**：平均盈亏（USDT）

## 数据流程

### 1. 数据来源

```
Decision Logger → decision_logs/*.json → AnalyzePerformance()
```

- 每次决策后，系统记录完整的 `DecisionRecord`
- 包含开仓/平仓的 `DecisionAction` 记录
- 每笔记录包含：symbol, action, quantity, price, leverage, timestamp

### 2. 数据窗口

```552:559:trader/auto_trader.go
// 5. 分析历史表现（最近100个周期，避免长期持仓的交易记录丢失）
// 假设每3分钟一个周期，100个周期 = 5小时，足够覆盖大部分交易
performance, err := at.decisionLogger.AnalyzePerformance(100)
if err != nil {
    log.Printf("⚠️  分析历史表现失败: %v", err)
    // 不影响主流程，继续执行（但设置performance为nil以避免传递错误数据）
    performance = nil
}
```

**默认窗口**：最近 100 个周期（约 5 小时）

### 3. 处理流程

1. **读取记录**：获取最近 N 个周期的 `DecisionRecord`
2. **预填充持仓**：从 3×N 窗口获取未平仓持仓
3. **匹配交易**：FIFO 匹配开仓和平仓
4. **计算指标**：PnL, WinRate, ProfitFactor, SharpeRatio
5. **币种统计**：按 symbol 聚合统计
6. **结果返回**：返回 `PerformanceAnalysis`

## 验证工具

系统提供了验证工具 `validation/validate_performance.go`：

```bash
go run validation/validate_performance.go [日志目录]
```

**功能**：
- 读取所有历史日志
- 匹配所有完整交易
- 计算统计指标
- 输出 JSON 报告
- 按币种排序展示

可以用来验证 `AnalyzePerformance()` 的计算是否正确。

## 潜在问题

### 1. Quantity 理解混淆

**问题**：quantity 可能被误解为 USD 价值。

**实际情况**：quantity 是币种数量（枚/张）。

**影响**：如果外部系统假设 quantity 是 USD，会产生计算错误。

### 2. PnL 计算正确性

**验证**：PnL 计算基于 quantity × 价格差，这是合约交易的标准计算方式。

✅ **正确**：符合 USDT 永续合约的盈亏计算逻辑。

### 3. Leverage 的使用

**关键点**：
- Leverage **不影响**绝对盈亏（PnL）
- Leverage **只影响**保证金需求（Margin Used）
- PnLPct 才是反映杠杆效应的指标

### 4. 持仓匹配边界

**问题**：如果开仓在窗口外，但平仓在窗口内，是否会遗漏？

**解决方案**：预填充使用 3× 窗口大小，减少遗漏风险。

## 建议改进

### 1. 添加注释说明

在关键计算处添加注释：

```go
// Quantity is the number of coins/contracts, NOT USD value
// Quantity = PositionSizeUSD / CurrentPrice
quantity := decision.PositionSizeUSD / marketData.CurrentPrice
```

### 2. 统一数据单位

考虑在 TradeOutcome 中添加 `PositionSizeUSD` 字段，明确展示 USD 仓位价值。

### 3. 增强验证

在计算过程中添加断言，确保数据一致性：
- quantity > 0
- openPrice > 0
- closePrice > 0
- leverage > 0

### 4. 文档化

建议为每个指标写清晰的文档说明：
- 计算公式
- 取值范围
- 业务含义
- 示例场景

## 总结

Performance 计算逻辑整体上是**正确的**，符合期货合约交易的标准算法：

✅ **正确的点**：
- Quantity 正确计算为币种数量
- PnL 计算符合 USDT 永续合约逻辑
- 杠杆使用正确（不影响绝对盈亏）
- FIFO 匹配逻辑合理
- 统计指标计算公式正确

⚠️ **需要注意的点**：
- 数据单位（quantity vs USD）需要明确文档化
- 窗口边界可能导致部分交易遗漏
- 夏普比率非年化，需在使用时明确说明

🎯 **建议**：
1. 添加更详细的注释和文档
2. 增强数据验证和边界检查
3. 提供可视化的计算验证工具


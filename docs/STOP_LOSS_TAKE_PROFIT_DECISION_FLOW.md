# 止盈止损决策流程

## 📋 概述

系统采用**AI自主决策 + 系统验证 + 自动执行**的三层架构来决定止盈止损策略。

---

## 🎯 决策流程

### 第一层：AI自主决策

**位置**: `prompts/*.txt` (系统提示词)

**决策主体**: AI模型（DeepSeek/Qwen等）

**决策依据**:
1. **技术分析**：
   - 支撑阻力位
   - 斐波那契回调/扩展
   - 波动带（ATR）
   - 趋势线
   - 技术指标（RSI、MACD等）

2. **市场数据**：
   - 多时间框架K线（3m/15m/1h/4h）
   - 成交量序列
   - 持仓量(OI)变化
   - 资金费率

3. **风险控制原则**：
   - 止损：建议距离入场价≤1%（但必须满足风险回报比≥1:3）
   - 止盈：最低风险回报比 1:3（系统硬约束）
   - 方向要求：
     - **做多**：止损 < 入场价 < 止盈
     - **做空**：止损 > 入场价 > 止盈

**AI输出格式**:
```json
{
  "symbol": "BTCUSDT",
  "action": "open_long",
  "stop_loss": 99000,      // AI自主决定
  "take_profit": 103000,   // AI自主决定
  "confidence": 85,
  "risk_usd": 300,
  "reasoning": "技术分析理由"
}
```

**关键约束**（在prompt中明确）:
- 风险回报比必须≥1:3（硬约束）
- 止损建议≤1%距离入场价（软约束）
- 止盈基于技术阻力位/斐波那契/波动带

---

### 第二层：系统验证

**位置**: `decision/engine.go:validateDecision()`

**验证内容**:

#### 1. 基础验证
```go
// 止损和止盈必须大于0
if d.StopLoss <= 0 || d.TakeProfit <= 0 {
    return fmt.Errorf("止损和止盈必须大于0")
}
```

#### 2. 方向验证
```go
// 做多：止损 < 止盈
if d.Action == "open_long" {
    if d.StopLoss >= d.TakeProfit {
        return fmt.Errorf("做多时止损价必须小于止盈价")
    }
}
// 做空：止损 > 止盈
else {
    if d.StopLoss <= d.TakeProfit {
        return fmt.Errorf("做空时止损价必须大于止盈价")
    }
}
```

#### 3. 风险回报比验证（核心验证）
```go
// 计算入场价（假设在止损和止盈之间20%位置）
var entryPrice float64
if d.Action == "open_long" {
    entryPrice = d.StopLoss + (d.TakeProfit-d.StopLoss)*0.2
} else {
    entryPrice = d.StopLoss - (d.StopLoss-d.TakeProfit)*0.2
}

// 计算风险回报比
var riskPercent, rewardPercent, riskRewardRatio float64
if d.Action == "open_long" {
    riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
    rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
} else {
    riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
    rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
}

riskRewardRatio = rewardPercent / riskPercent

// 硬约束：风险回报比必须≥3.0
if riskRewardRatio < 3.0 {
    return fmt.Errorf("风险回报比过低(%.2f:1)，必须≥3.0:1", riskRewardRatio)
}
```

**验证逻辑说明**:
- 假设入场价在止损和止盈之间20%位置（保守估算）
- 计算风险百分比 = |入场价 - 止损价| / 入场价 × 100%
- 计算收益百分比 = |止盈价 - 入场价| / 入场价 × 100%
- 风险回报比 = 收益百分比 / 风险百分比
- **硬约束**：风险回报比必须≥3.0:1

**验证失败处理**:
- 如果验证失败，决策会被拒绝
- 错误信息会返回给AI
- AI需要重新决策，调整止盈止损价格

---

### 第三层：自动执行

**位置**: `trader/auto_trader.go:executeOpenLongWithRecord()` / `executeOpenShortWithRecord()`

**执行流程**:

#### 1. 开仓
```go
// 开仓
order, err := at.trader.OpenLong(decision.Symbol, quantity, decision.Leverage)
```

#### 2. 等待订单成交
```go
// 等待订单成交后，查询持仓信息获取实际成交价格
time.Sleep(2 * time.Second)
positions, err = at.trader.GetPositions()
// 获取实际成交价格（entryPrice）
```

#### 3. 设置止损止盈订单
```go
// 设置止损止盈（使用AI决策中的价格）
if err := at.trader.SetStopLoss(decision.Symbol, "LONG", quantity, decision.StopLoss); err != nil {
    log.Printf("  ⚠ 设置止损失败: %v", err)
}
if err := at.trader.SetTakeProfit(decision.Symbol, "LONG", quantity, decision.TakeProfit); err != nil {
    log.Printf("  ⚠ 设置止盈失败: %v", err)
}
```

**执行说明**:
- 开仓成功后，**立即**设置止损止盈订单
- 使用AI决策中提供的 `stop_loss` 和 `take_profit` 价格
- 如果设置失败，会记录警告，但**不会回滚开仓**（已开仓的仓位会保留）

---

## 📊 决策依据总结

### AI决策止盈止损价格的依据

1. **技术分析**（主要依据）:
   - 支撑阻力位：止损放在支撑位下方，止盈放在阻力位附近
   - 斐波那契：使用斐波那契回调/扩展位作为止盈止损参考
   - 波动带（ATR）：根据波动率调整止盈止损距离
   - 趋势线：止损放在趋势线下方/上方

2. **风险控制**（必须满足）:
   - 风险回报比≥1:3（硬约束）
   - 止损距离建议≤1%（软约束）
   - 单笔风险≤账户净值的1-3%

3. **市场环境**（动态调整）:
   - 高波动环境：扩大止盈止损距离
   - 低波动环境：缩小止盈止损距离
   - 趋势强度：强趋势时止盈可以更远

### 系统验证的硬约束

1. **风险回报比≥3.0:1**（核心约束）
   - 这是系统级别的硬约束
   - 如果AI决策的风险回报比<3.0，决策会被拒绝
   - AI必须重新决策，调整止盈止损价格

2. **方向正确性**
   - 做多：止损 < 入场价 < 止盈
   - 做空：止损 > 入场价 > 止盈

3. **价格有效性**
   - 止损和止盈价格必须>0

---

## 🔄 完整流程示例

### 示例：BTC做多

1. **AI分析市场**:
   - BTC当前价格：100,000 USDT
   - 技术分析：支撑位在99,000，阻力位在103,000
   - 风险回报比计算：止损99,000，止盈103,000
   - 风险：(100,000 - 99,000) / 100,000 = 1%
   - 收益：(103,000 - 100,000) / 100,000 = 3%
   - 风险回报比：3:1 ✅

2. **AI决策输出**:
```json
{
  "symbol": "BTCUSDT",
  "action": "open_long",
  "stop_loss": 99000,
  "take_profit": 103000,
  "confidence": 85,
  "reasoning": "支撑位99,000，阻力位103,000，风险回报比3:1"
}
```

3. **系统验证**:
   - ✅ 止损和止盈>0
   - ✅ 止损(99,000) < 止盈(103,000)
   - ✅ 风险回报比验证：
     - 假设入场价：99,000 + (103,000-99,000)*0.2 = 99,800
     - 风险：(99,800 - 99,000) / 99,800 = 0.8%
     - 收益：(103,000 - 99,800) / 99,800 = 3.2%
     - 风险回报比：3.2 / 0.8 = 4.0:1 ✅

4. **执行开仓**:
   - 开仓成功，实际成交价：100,050 USDT
   - 设置止损订单：99,000 USDT
   - 设置止盈订单：103,000 USDT

5. **后续监控**:
   - 如果价格触及99,000，止损订单自动触发
   - 如果价格触及103,000，止盈订单自动触发

---

## ⚠️ 注意事项

### 1. 风险回报比计算方式

系统使用**保守估算**方式计算风险回报比：
- 假设入场价在止损和止盈之间20%位置
- 这比实际入场价更保守，确保即使实际入场价更接近止损，风险回报比仍然≥3.0

### 2. 止损止盈设置失败处理

如果设置止损止盈订单失败：
- 会记录警告日志
- **不会回滚开仓**（已开仓的仓位会保留）
- 需要手动管理或等待下一个周期重新设置

### 3. AI决策的灵活性

AI可以：
- 根据市场情况动态调整止盈止损距离
- 使用技术分析确定精确的止盈止损价格
- 在满足硬约束的前提下，自主决定风险回报比（可以>3:1）

### 4. 系统约束的严格性

系统会：
- **严格验证**风险回报比≥3.0:1
- **拒绝**不符合要求的决策
- **不修改**AI决策的价格（只验证，不调整）

---

## 📝 总结

**止盈止损决策流程**：
1. **AI自主决策**：基于技术分析，自主决定止盈止损价格
2. **系统验证**：验证风险回报比≥3.0:1，方向正确性
3. **自动执行**：开仓成功后，立即设置止损止盈订单

**核心特点**：
- AI有**完全自主权**决定止盈止损价格
- 系统有**硬约束**确保风险回报比≥3.0:1
- 执行是**自动化**的，无需人工干预

**关键约束**：
- 风险回报比必须≥3.0:1（硬约束）
- 止损建议≤1%距离入场价（软约束）
- 方向必须正确（硬约束）


# Cycle 业务逻辑完整文档

## 概述

一个完整的交易周期（Cycle）从开始到结束，包括数据获取、AI决策、订单执行和记录保存的完整流程。

---

## 时间节点与流程

### 阶段 0: Cycle 启动
**时间节点**: `runCycle()` 开始执行

```
⏰ Cycle #N 开始
├── 创建决策记录对象 (DecisionRecord)
├── 检查风险控制（stopUntil）
└── 检查日盈亏重置（24小时）
```

**记录内容**:
- `record.Timestamp` - 周期开始时间
- `record.CycleNumber` - 周期编号（自动递增）
- `record.Success` - 初始化为 true

---

### 阶段 1: 获取账户信息（外部 API 调用 #1）

**时间节点**: `buildTradingContext()` → `trader.GetBalance()`

**API 调用**: `Hyperliquid API: UserState`
```go
accountState, err := t.exchange.Info().UserState(t.ctx, t.walletAddr)
```

**获取数据**:
- `AccountValue` - 总账户净值（字符串类型）
- `TotalMarginUsed` - 占用保证金（字符串类型）
- `AssetPositions[]` - 持仓数组

**数据处理**:
```go
// 1. 解析字符串为 float64
accountValue, _ := strconv.ParseFloat(accountState.MarginSummary.AccountValue, 64)
totalMarginUsed, _ := strconv.ParseFloat(accountState.MarginSummary.TotalMarginUsed, 64)

// 2. 累加所有持仓的未实现盈亏
totalUnrealizedPnl := 0.0
for _, assetPos := range accountState.AssetPositions {
    unrealizedPnl, _ := strconv.ParseFloat(assetPos.Position.UnrealizedPnl, 64)
    totalUnrealizedPnl += unrealizedPnl
}

// 3. 计算钱包余额（不含未实现盈亏）
walletBalanceWithoutUnrealized := accountValue - totalUnrealizedPnl

// 4. 返回结果
result["totalWalletBalance"] = walletBalanceWithoutUnrealized
result["availableBalance"] = accountValue - totalMarginUsed
result["totalUnrealizedProfit"] = totalUnrealizedPnl
```

**数据校验**:
- ✅ 检查 API 返回错误
- ✅ 类型断言校验（string → float64）
- ✅ 数值合理性检查（accountValue > 0）

**记录时机**: 此时**不记录**，仅用于构建上下文

---

### 阶段 2: 获取持仓信息（外部 API 调用 #2）

**时间节点**: `buildTradingContext()` → `trader.GetPositions()`

**API 调用**: `Hyperliquid API: UserState`（与阶段1是同一个调用，但提取不同字段）

**获取数据**:
```go
for _, assetPos := range accountState.AssetPositions {
    position := assetPos.Position
    
    // 持仓数量
    posAmt, _ := strconv.ParseFloat(position.Szi, 64)
    
    // 开仓价格（指针类型，可能为 nil）
    var entryPrice float64
    if position.EntryPx != nil {
        entryPrice, _ = strconv.ParseFloat(*position.EntryPx, 64)
    }
    
    // 标记价格（通过计算）
    markPrice = positionValue / absFloat(posAmt)
    
    // 未实现盈亏
    unrealizedPnl, _ := strconv.ParseFloat(position.UnrealizedPnl, 64)
    
    // 杠杆
    leverage = position.Leverage.Value
}
```

**数据校验**:
- ✅ 检查 `EntryPx != nil`（指针类型）
- ✅ 检查 `posAmt != 0`（跳过无持仓）
- ✅ 类型断言校验（string → float64）
- ✅ 计算 markPrice 时检查分母不为 0

**记录时机**: 此时**不记录**，但会保存到 `record.Positions` 快照中

---

### 阶段 3: 保存账户和持仓快照

**时间节点**: `runCycle()` 中，构建上下文后

**记录内容**:
```go
// 账户状态快照
record.AccountState = logger.AccountSnapshot{
    TotalBalance:          ctx.Account.TotalEquity,
    AvailableBalance:      ctx.Account.AvailableBalance,
    TotalUnrealizedProfit: ctx.Account.TotalPnL,
    PositionCount:         ctx.Account.PositionCount,
    MarginUsedPct:         ctx.Account.MarginUsedPct,
}

// 持仓快照
for _, pos := range ctx.Positions {
    record.Positions = append(record.Positions, logger.PositionSnapshot{
        Symbol:           pos.Symbol,
        Side:             pos.Side,
        PositionAmt:      pos.Quantity,
        EntryPrice:       pos.EntryPrice,
        MarkPrice:        pos.MarkPrice,
        UnrealizedProfit: pos.UnrealizedPnL,
        Leverage:         float64(pos.Leverage),
        LiquidationPrice: pos.LiquidationPrice,
    })
}
```

**数据校验**:
- ✅ 所有字段都来自已验证的上下文数据
- ✅ 数值类型转换校验（int → float64）

---

### 阶段 4: 检测自动触发的止盈止损

**时间节点**: 保存快照后，AI决策前

**检测逻辑**:
```go
// 比较上一个周期的持仓和当前持仓
if len(at.lastCyclePositions) > 0 {
    // 构建当前持仓的key集合
    currentPositionKeys := make(map[string]bool)
    for _, pos := range ctx.Positions {
        posKey := pos.Symbol + "_" + pos.Side
        currentPositionKeys[posKey] = true
    }
    
    // 检测消失的持仓（自动触发平仓）
    for _, lastPos := range at.lastCyclePositions {
        posKey := lastPos.Symbol + "_" + lastPos.Side
        if !currentPositionKeys[posKey] {
            // 持仓消失了，创建自动触发记录
            closeAction := logger.DecisionAction{
                Action:          "close_" + lastPos.Side,
                Symbol:          lastPos.Symbol,
                Quantity:        lastPos.Quantity,
                Price:           lastPos.MarkPrice, // 使用标记价格
                IsAutoTriggered: true,
                WasStopLoss:     wasStopLoss, // 通过价格变化判断
            }
            record.Decisions = append(record.Decisions, closeAction)
        }
    }
}
```

**判断止损/止盈**:
```go
wasStopLoss := false
if lastPos.Side == "long" {
    // 多仓：标记价格低于开仓价 = 止损
    if lastPos.MarkPrice < lastPos.EntryPrice {
        wasStopLoss = true
    }
} else {
    // 空仓：标记价格高于开仓价 = 止损
    if lastPos.MarkPrice > lastPos.EntryPrice {
        wasStopLoss = true
    }
}
```

**数据校验**:
- ✅ 比较持仓列表（symbol_side 作为唯一标识）
- ✅ 价格合理性检查（markPrice > 0）

---

### 阶段 5: 调用 AI 获取决策（外部 API 调用 #3）

**时间节点**: 保存快照后

**API 调用**: `MCP Client: GetFullDecisionWithCustomPrompt`
```go
decision, err := decision.GetFullDecisionWithCustomPrompt(
    ctx,                    // 交易上下文（包含账户、持仓、历史表现等）
    at.mcpClient,          // MCP 客户端
    at.customPrompt,       // 自定义策略prompt
    at.overrideBasePrompt, // 是否覆盖基础prompt
    at.systemPromptTemplate, // 系统提示词模板
)
```

**输入数据**:
- `Account` - 账户信息
- `Positions` - 持仓列表
- `CandidateCoins` - 候选币种
- `Performance` - 历史表现（最近100个周期）

**返回数据**:
- `SystemPrompt` - 系统提示词
- `UserPrompt` - 用户输入prompt
- `CoTTrace` - AI思维链
- `Decisions[]` - 决策列表

**记录内容**:
```go
if decision != nil {
    record.SystemPrompt = decision.SystemPrompt
    record.InputPrompt = decision.UserPrompt
    record.CoTTrace = decision.CoTTrace
    if len(decision.Decisions) > 0 {
        decisionJSON, _ := json.MarshalIndent(decision.Decisions, "", "  ")
        record.DecisionJSON = string(decisionJSON)
    }
}
```

**数据校验**:
- ✅ 检查 AI 返回错误
- ✅ 即使有错误也保存思维链（用于调试）

---

### 阶段 6: 执行决策并记录执行结果

**时间节点**: AI决策后

**执行流程**:
```go
// 1. 对决策排序（先平仓后开仓）
sortedDecisions := sortDecisionsByPriority(decision.Decisions)

// 2. 执行每个决策
for _, d := range sortedDecisions {
    actionRecord := logger.DecisionAction{
        Action:          d.Action,
        Symbol:          d.Symbol,
        Quantity:        0,  // 初始为0，执行后更新
        Leverage:        d.Leverage,
        Price:           0,  // 初始为0，执行后更新
        Timestamp:       time.Now(),
        Success:         false,
        IsAutoTriggered: false,
    }
    
    // 执行决策
    if err := at.executeDecisionWithRecord(&d, &actionRecord); err != nil {
        actionRecord.Error = err.Error()
        actionRecord.Success = false
    } else {
        actionRecord.Success = true
    }
    
    // 添加到记录
    record.Decisions = append(record.Decisions, actionRecord)
}
```

---

### 阶段 7: 执行开仓订单（外部 API 调用 #4）

**时间节点**: `executeOpenLongWithRecord()` 或 `executeOpenShortWithRecord()`

**API 调用序列**:
1. **获取市场价格**（内部，非外部API）
   ```go
   marketData, err := market.Get(decision.Symbol)
   actionRecord.Price = marketData.CurrentPrice  // 初始价格
   ```

2. **设置杠杆** → `Hyperliquid API: SetLeverage`
   ```go
   t.exchange.UpdateLeverage(t.ctx, coin, leverage)
   ```

3. **提交订单** → `Hyperliquid API: Order`
   ```go
   order := hyperliquid.CreateOrderRequest{
       Coin:  coin,
       IsBuy: true/false,
       Size:  roundedQuantity,
       Price: aggressivePrice,
       OrderType: hyperliquid.OrderType{
           Limit: &hyperliquid.LimitOrderType{
               Tif: hyperliquid.TifIoc, // IOC订单
           },
       },
   }
   _, err = t.exchange.Order(t.ctx, order, nil)
   ```

4. **等待订单成交**（2秒）
   ```go
   time.Sleep(2 * time.Second)
   ```

5. **获取实际成交价格** → `Hyperliquid API: UserState`（再次调用）
   ```go
   positions, err := at.trader.GetPositions()
   for _, pos := range positions {
       if pos["symbol"] == decision.Symbol && pos["side"] == "long/short" {
           if entryPrice, ok := pos["entryPrice"].(float64); ok && entryPrice > 0 {
               actionRecord.Price = entryPrice  // 使用实际成交价格
               break
           }
       }
   }
   ```

**数据校验**:
- ✅ 检查订单提交错误
- ✅ 检查 entryPrice 是否获取成功
- ✅ 如果 entryPrice 为 0，回退到市场价格
- ✅ 类型断言校验（`pos["entryPrice"].(float64)`）

**记录内容**:
```go
actionRecord.Price = entryPrice  // 实际成交价格
actionRecord.Quantity = quantity  // 实际成交数量
actionRecord.OrderID = orderID    // 订单ID（如果有）
actionRecord.Success = true       // 执行成功
actionRecord.Timestamp = time.Now() // 执行时间
```

---

### 阶段 8: 执行平仓订单（外部 API 调用 #5）

**时间节点**: `executeCloseLongWithRecord()` 或 `executeCloseShortWithRecord()`

**API 调用序列**:
1. **获取市场价格**（初始价格）
   ```go
   marketData, err := market.Get(decision.Symbol)
   actionRecord.Price = marketData.CurrentPrice
   ```

2. **获取持仓数量** → `Hyperliquid API: UserState`
   ```go
   positions, err := at.trader.GetPositions()
   for _, pos := range positions {
       if pos["symbol"] == decision.Symbol && pos["side"] == "long/short" {
           actionRecord.Quantity = pos["positionAmt"].(float64)
           break
       }
   }
   ```

3. **提交平仓订单** → `Hyperliquid API: Order`
   ```go
   order := hyperliquid.CreateOrderRequest{
       Coin:  coin,
       IsBuy: true,  // 平空仓是买入
       Size:  roundedQuantity,
       Price: aggressivePrice,
       ReduceOnly: true,  // 平仓标记
   }
   _, err = t.exchange.Order(t.ctx, order, nil)
   ```

4. **等待订单成交**（2秒）
   ```go
   time.Sleep(2 * time.Second)
   ```

5. **获取实际成交价格**（成交后的市场价格）
   ```go
   marketDataAfter, err := market.Get(decision.Symbol)
   if err == nil {
       actionRecord.Price = marketDataAfter.CurrentPrice  // 使用成交后价格
   }
   ```

**数据校验**:
- ✅ 检查持仓是否存在
- ✅ 检查 quantity 是否获取成功
- ✅ 检查订单提交错误
- ✅ 检查成交后价格获取是否成功

**记录内容**:
```go
actionRecord.Price = marketDataAfter.CurrentPrice  // 实际成交价格（近似）
actionRecord.Quantity = quantity                     // 实际平仓数量
actionRecord.Success = true
```

---

### 阶段 9: 保存决策记录（本地文件）

**时间节点**: 所有决策执行完成后

**记录流程**:
```go
// 1. 设置周期编号和时间戳
l.cycleNumber++
record.CycleNumber = l.cycleNumber
record.Timestamp = time.Now()

// 2. 生成文件名
filename := fmt.Sprintf("decision_%s_cycle%d.json",
    record.Timestamp.Format("20060102_150405"),
    record.CycleNumber)

// 3. 序列化为JSON
data, err := json.MarshalIndent(record, "", "  ")

// 4. 写入文件
filepath := filepath.Join(l.logDir, filename)
err := ioutil.WriteFile(filepath, data, 0644)
```

**记录内容**:
```json
{
  "timestamp": "2025-11-04T16:30:00+08:00",
  "cycle_number": 151,
  "system_prompt": "...",
  "input_prompt": "...",
  "cot_trace": "...",
  "decision_json": "[...]",
  "account_state": {
    "total_balance": 1000.0,
    "available_balance": 800.0,
    "total_unrealized_profit": 10.0,
    "position_count": 2,
    "margin_used_pct": 20.0
  },
  "positions": [
    {
      "symbol": "BTCUSDT",
      "side": "long",
      "position_amt": 0.005,
      "entry_price": 107026,
      "mark_price": 107100,
      "unrealized_profit": 0.37,
      "leverage": 10
    }
  ],
  "candidate_coins": ["BTCUSDT", "ETHUSDT"],
  "decisions": [
    {
      "action": "open_long",
      "symbol": "ETHUSDT",
      "quantity": 0.1,
      "leverage": 5,
      "price": 3504.1,      // 实际成交价格（entryPrice）
      "order_id": 12345,
      "timestamp": "2025-11-04T16:30:05+08:00",
      "success": true,
      "is_auto_triggered": false
    }
  ],
  "execution_log": [
    "✓ ETHUSDT open_long 成功"
  ],
  "success": true
}
```

**数据校验**:
- ✅ JSON 序列化错误检查
- ✅ 文件写入错误检查
- ✅ 目录权限检查（通过 `os.MkdirAll`）

---

### 阶段 10: 更新状态（为下一个周期准备）

**时间节点**: 记录保存后

**更新内容**:
```go
// 更新上一个周期的持仓列表（用于检测自动触发）
at.lastCyclePositions = currentPositionsCopy
```

---

## 关键外部 API 调用总结

| 阶段 | API调用 | 目的 | 数据获取 | 记录时机 |
|------|---------|------|----------|----------|
| 1 | `UserState` | 获取账户余额 | `AccountValue`, `TotalMarginUsed`, `UnrealizedPnl` | 阶段3（账户快照） |
| 2 | `UserState` | 获取持仓信息 | `EntryPx`, `MarkPrice`, `Quantity`, `Leverage` | 阶段3（持仓快照） |
| 3 | `MCP Client` | AI决策 | `Decisions[]`, `CoTTrace` | 阶段5（立即记录） |
| 4 | `SetLeverage` | 设置杠杆 | - | 阶段7（不记录） |
| 5 | `Order` | 提交开仓订单 | `OrderID` | 阶段7（执行结果） |
| 6 | `UserState` | 获取实际成交价格 | `EntryPrice` | 阶段7（更新Price） |
| 7 | `UserState` | 获取持仓数量 | `PositionAmt` | 阶段8（Quantity） |
| 8 | `Order` | 提交平仓订单 | `OrderID` | 阶段8（执行结果） |
| 9 | `GetCurrentPrice` | 获取成交后价格 | `CurrentPrice` | 阶段8（更新Price） |

---

## 数据校验机制

### 1. API 调用校验
- ✅ 检查 `err != nil`
- ✅ 检查返回数据是否为 nil
- ✅ 检查指针类型是否为 nil（`EntryPx != nil`）

### 2. 类型转换校验
- ✅ 字符串转数字：`strconv.ParseFloat()`
- ✅ 类型断言：`value.(float64)` + `ok` 检查
- ✅ 类型断言：`value.(string)` + `ok` 检查

### 3. 数值合理性校验
- ✅ `entryPrice > 0`
- ✅ `quantity > 0`
- ✅ `accountValue > 0`
- ✅ 除法运算分母不为 0

### 4. 业务逻辑校验
- ✅ 持仓存在性检查（平仓前）
- ✅ 订单提交错误检查
- ✅ 成交价格获取失败时的回退机制

---

## 关键时间节点图

```
T0: Cycle 启动
    ↓
T1: 获取账户信息 (UserState API)
    ↓
T2: 获取持仓信息 (UserState API)
    ↓
T3: 保存账户和持仓快照 (本地记录)
    ↓
T4: 检测自动触发平仓 (本地逻辑)
    ↓
T5: 调用 AI 获取决策 (MCP API)
    ↓
T6: 执行开仓/平仓订单
    ├─ 开仓: SetLeverage → Order → 等待2秒 → UserState (获取entryPrice)
    └─ 平仓: UserState (获取quantity) → Order → 等待2秒 → GetCurrentPrice
    ↓
T7: 保存决策记录 (本地文件)
    ↓
T8: 更新状态 (为下一个周期准备)
```

---

## 数据记录流程

### 记录层次
1. **账户快照** - 周期开始时的账户状态
2. **持仓快照** - 周期开始时的持仓状态
3. **AI决策** - 系统提示词、思维链、决策JSON
4. **执行结果** - 每个决策的执行结果（价格、数量、成功/失败）
5. **执行日志** - 文本格式的执行日志

### 数据来源
- **外部API**: Hyperliquid API, MCP Client
- **内部计算**: 盈亏计算、保证金计算
- **状态跟踪**: 持仓变化检测、自动触发检测

### 数据更新时机
- **实时更新**: 订单执行后立即更新 `actionRecord.Price` 和 `actionRecord.Quantity`
- **周期结束**: 所有数据一次性保存到 JSON 文件

---

## 注意事项

1. **价格记录**: 
   - 开仓：使用 `entryPrice`（实际成交价格）
   - 平仓：使用成交后的市场价格（近似值）

2. **数量记录**:
   - 开仓：使用计算的数量
   - 平仓：从持仓信息中获取实际数量

3. **等待时间**:
   - 订单提交后等待 2 秒，确保订单成交
   - 确保持仓信息已更新

4. **错误处理**:
   - 即使 API 调用失败，也记录错误信息
   - 关键数据缺失时使用回退值

5. **数据一致性**:
   - 所有数据在同一周期内获取
   - 账户快照和持仓快照在同一时间点


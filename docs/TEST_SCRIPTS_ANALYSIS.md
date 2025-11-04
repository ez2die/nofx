# 测试脚本数据源分析

## 当前状态

### ❌ **测试脚本未使用真实的 Hyperliquid 持仓数据**

**`test_nof1_decision.go` 和 `test_lean_decision.go` 当前行为：**

1. **账户数据** - 使用**模拟数据**：
   ```go
   Account: decision.AccountInfo{
       TotalEquity:      initialBalance,  // 从 config.json 读取
       AvailableBalance: initialBalance,  // 模拟数据
       TotalPnL:         0.0,             // 固定为0
       TotalPnLPct:      0.0,
       MarginUsed:       0.0,
       MarginUsedPct:    0.0,
       PositionCount:   0,                // 固定为0
   }
   ```

2. **持仓数据** - 使用**空列表**：
   ```go
   Positions: []decision.PositionInfo{},  // 空持仓列表
   ```

3. **未调用真实 API**：
   - ❌ 未调用 `trader.GetBalance()` 获取真实账户余额
   - ❌ 未调用 `trader.GetPositions()` 获取真实持仓列表

---

## 真实系统的行为

**`trader/auto_trader.go` 的 `buildTradingContext()` 方法：**

1. **获取真实账户数据**：
   ```go
   // 1. 获取账户信息
   balance, err := at.trader.GetBalance()  // ✅ 调用真实 API
   totalEquity := totalWalletBalance + totalUnrealizedProfit
   ```

2. **获取真实持仓数据**：
   ```go
   // 2. 获取持仓信息
   positions, err := at.trader.GetPositions()  // ✅ 调用真实 API
   
   for _, pos := range positions {
       // 转换为 decision.PositionInfo
       positionInfos = append(positionInfos, ...)
   }
   ```

---

## 问题影响

### ⚠️ **测试脚本的限制：**

1. **无法测试持仓管理逻辑**
   - AI 无法看到真实持仓
   - 无法测试"评估现有持仓"的决策流程
   - 无法测试"关闭持仓"的决策

2. **账户状态不真实**
   - 总权益、可用余额都是模拟值
   - 无法测试 circuit breakers（drawdown >25%）
   - 无法测试真实的风险管理逻辑

3. **测试场景不完整**
   - 只能测试"开新仓"的场景
   - 无法测试"管理现有持仓"的场景

---

## 修改方案

### 方案 A：添加可选的真实数据获取（推荐）

修改测试脚本，添加选项使用真实 Hyperliquid 数据：

```go
// 添加命令行参数或环境变量
useRealData := os.Getenv("USE_REAL_HYPERLIQUID_DATA") == "true"

if useRealData {
    // 初始化 Hyperliquid 交易器
    trader, err := NewHyperliquidTrader(
        config.HyperliquidPrivateKey,
        config.HyperliquidWalletAddr,
        config.HyperliquidTestnet,
    )
    if err != nil {
        log.Fatalf("初始化Hyperliquid交易器失败: %v", err)
    }
    
    // 获取真实账户余额
    balance, err := trader.GetBalance()
    if err != nil {
        log.Fatalf("获取账户余额失败: %v", err)
    }
    
    // 获取真实持仓
    positions, err := trader.GetPositions()
    if err != nil {
        log.Fatalf("获取持仓失败: %v", err)
    }
    
    // 转换为 decision.PositionInfo
    // ... 转换逻辑 ...
} else {
    // 使用模拟数据（当前行为）
    // ... 现有代码 ...
}
```

### 方案 B：创建新的集成测试脚本

创建一个新的测试脚本 `test_with_real_data.go`，专门用于真实数据测试：

```go
// test_with_real_data.go
// 需要真实 Hyperliquid 配置才能运行
// 用于完整测试：包括持仓管理、账户状态等
```

### 方案 C：复用 `buildTradingContext()` 方法

在测试脚本中创建 `AutoTrader` 实例，然后调用其 `buildTradingContext()` 方法：

```go
// 创建 AutoTrader 实例（需要完整的配置）
at := &AutoTrader{
    config: config,
    trader: hyperliquidTrader,
    // ... 其他字段 ...
}

// 调用真实的数据获取方法
ctx, err := at.buildTradingContext()
```

---

## 推荐实施

### ✅ **推荐：方案 A + 方案 C 结合**

1. **添加命令行参数**控制是否使用真实数据
2. **复用 `buildTradingContext()` 逻辑**，避免重复代码
3. **保留模拟数据选项**，用于快速测试 prompt 本身

**修改步骤：**

1. 在测试脚本中添加参数解析：
   ```go
   var useRealData bool
   flag.BoolVar(&useRealData, "real", false, "Use real Hyperliquid data")
   flag.Parse()
   ```

2. 根据参数选择数据源：
   ```go
   if useRealData {
       // 初始化交易器并获取真实数据
       ctx = buildRealTradingContext(...)
   } else {
       // 使用模拟数据（当前行为）
       ctx = buildMockTradingContext(...)
   }
   ```

3. 创建辅助函数复用真实逻辑：
   ```go
   func buildRealTradingContext(config, trader) (*decision.Context, error) {
       // 复用 auto_trader.go 的逻辑
   }
   ```

---

## 当前脚本的使用场景

### ✅ **适合的场景：**
- 测试 prompt 的格式和兼容性
- 测试 AI 是否输出正确的 action 格式
- 测试 JSON 格式验证
- 快速验证 prompt 修改

### ❌ **不适合的场景：**
- 测试持仓管理决策
- 测试账户状态相关的 circuit breakers
- 测试真实的风险管理逻辑
- 测试多币种持仓场景

---

## 建议

1. **保留当前测试脚本** - 用于快速测试 prompt 格式
2. **添加真实数据选项** - 通过命令行参数启用
3. **创建集成测试脚本** - 专门用于完整功能测试

这样可以同时支持：
- 🔧 快速开发测试（模拟数据）
- 🧪 完整功能测试（真实数据）
- 📊 实际决策验证（真实数据）


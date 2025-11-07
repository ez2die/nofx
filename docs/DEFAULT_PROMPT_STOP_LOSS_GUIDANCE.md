# Default.txt Prompt 中关于止盈止损的引导分析

## 📋 概述

`default.txt` prompt 模板本身对止盈止损的引导**相对简洁**，主要通过**间接方式**引导AI选择止盈止损点。

---

## 🔍 Default.txt 模板中的引导

### 1. 分析方法引导（间接引导）

**位置**: `prompts/default.txt:67-68`

```67:68:prompts/default.txt
分析方法（完全由你自主决定）：
- 自由运用序列数据，你可以做但不限于趋势分析、形态识别、支撑阻力、技术阻力位、斐波那契、波动带计算
```

**引导方式**:
- ✅ 明确提到"支撑阻力、技术阻力位、斐波那契、波动带计算"
- ✅ 这些技术分析方法**间接引导**AI使用这些工具来确定止盈止损点
- ⚠️ 但没有明确说明**如何使用**这些工具设置止盈止损

**实际效果**:
- AI知道可以使用这些技术分析方法
- AI需要**自主推断**如何将这些分析方法应用到止盈止损设置上

---

### 2. 风险回报比约束（硬约束）

**位置**: `prompts/default.txt:114`

```114:114:prompts/default.txt
- 风险回报比1:3是底线
```

**引导方式**:
- ✅ 明确要求风险回报比≥1:3
- ✅ 这是**硬约束**，AI必须满足
- ⚠️ 但没有说明**如何计算**或**如何设置**止盈止损来满足这个约束

**实际效果**:
- AI知道必须满足风险回报比≥1:3
- AI需要**自主计算**止盈止损价格来满足这个约束

---

### 3. 交易哲学引导（间接引导）

**位置**: `prompts/default.txt:26`

```26:26:prompts/default.txt
纪律胜于情绪：执行你的退出方案，不随意移动止损或目标
```

**引导方式**:
- ✅ 强调"执行你的退出方案"
- ✅ 强调"不随意移动止损或目标"
- ⚠️ 但没有说明**如何制定**退出方案（止盈止损）

**实际效果**:
- AI知道需要制定退出方案
- AI知道不应该随意移动止损止盈
- 但**如何制定**退出方案需要AI自主决定

---

### 4. 决策流程引导（间接引导）

**位置**: `prompts/default.txt:105`

```105:105:prompts/default.txt
2. 评估持仓: 趋势是否改变？是否该止盈/止损？
```

**引导方式**:
- ✅ 提到"是否该止盈/止损"
- ⚠️ 但这是针对**已有持仓**的评估，不是开仓时的止盈止损设置

**实际效果**:
- AI知道需要评估是否止盈止损
- 但这是**持仓管理**，不是**开仓时的止盈止损设置**

---

## 🔧 系统构建时添加的引导

### 1. 硬约束（动态添加）

**位置**: `decision/engine.go:302-308`

```302:308:decision/engine.go
	// 2. 硬约束（风险控制）- 动态生成
	sb.WriteString("# 硬约束（风险控制）\n\n")
	sb.WriteString("1. 风险回报比: 必须 ≥ 1:3（冒1%风险，赚3%+收益）\n")
	sb.WriteString("2. 最多持仓: 3个币种（质量>数量）\n")
	sb.WriteString(fmt.Sprintf("3. 单币仓位: 山寨%.0f-%.0f U(%dx杠杆) | BTC/ETH %.0f-%.0f U(%dx杠杆)\n",
		accountEquity*0.8, accountEquity*1.5, altcoinLeverage, accountEquity*5, accountEquity*10, btcEthLeverage))
	sb.WriteString("4. 保证金: 总使用率 ≤ 90%\n\n")
```

**引导内容**:
- ✅ 明确说明"风险回报比: 必须 ≥ 1:3（冒1%风险，赚3%+收益）"
- ✅ 这提供了**具体的量化标准**（1%风险，3%+收益）
- ⚠️ 但仍然没有说明**如何设置**止盈止损价格

---

### 2. 输出格式示例（间接引导）

**位置**: `decision/engine.go:315-322`

```315:322:decision/engine.go
	sb.WriteString("```json\n[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 91000, \"confidence\": 85, \"risk_usd\": 300, \"reasoning\": \"下跌趋势+MACD死叉\"},\n", btcEthLeverage, accountEquity*5))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\", \"reasoning\": \"止盈离场\"}\n")
	sb.WriteString("]\n```\n\n")
	sb.WriteString("字段说明:\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `confidence`: 0-100（开仓建议≥75）\n")
	sb.WriteString("- 开仓时必填: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning\n\n")
```

**引导内容**:
- ✅ 示例中显示 `"stop_loss": 97000, "take_profit": 91000`
- ✅ 说明"开仓时必填: stop_loss, take_profit"
- ⚠️ 但示例中的价格是**硬编码的示例值**，不是实际的计算方法

**实际效果**:
- AI知道需要提供 `stop_loss` 和 `take_profit` 字段
- AI知道这些字段是**必填的**
- 但**如何计算**这些价格需要AI自主决定

---

## 📊 总结：Default.txt 的引导方式

### ✅ 有的引导

1. **分析方法引导**（间接）:
   - 提到"支撑阻力、技术阻力位、斐波那契、波动带计算"
   - AI可以使用这些方法来确定止盈止损点

2. **风险回报比约束**（硬约束）:
   - 明确要求风险回报比≥1:3
   - 提供量化标准（1%风险，3%+收益）

3. **输出格式要求**（明确）:
   - 明确要求提供 `stop_loss` 和 `take_profit` 字段
   - 提供示例格式

### ⚠️ 缺失的引导

1. **没有明确说明如何设置止盈止损价格**:
   - 没有说明如何使用支撑阻力位设置止盈止损
   - 没有说明如何使用斐波那契设置止盈止损
   - 没有说明如何使用波动带设置止盈止损

2. **没有明确说明止损距离**:
   - 没有说明止损应该距离入场价多远
   - 没有说明如何根据波动率调整止损距离

3. **没有明确说明止盈目标**:
   - 没有说明止盈应该设置在什么位置
   - 没有说明如何根据技术阻力位设置止盈

4. **没有明确说明方向要求**:
   - 没有明确说明做多时止损<止盈，做空时止损>止盈
   - （这个在系统验证时会检查，但prompt中没有明确说明）

---

## 🔄 对比：其他模板的引导方式

### Lean.txt 模板（更详细）

**位置**: `prompts/lean.txt:23-34`

```23:34:prompts/lean.txt
# RISK MANAGEMENT (MANDATORY)
- stop_loss: Max 1% from entry (recommended, but must satisfy risk/reward ≥3:1)
  - **Directional requirement (CRITICAL)**:
    - For LONG positions: stop_loss must be **LOWER** than entry price (triggers when price falls)
    - For SHORT positions: stop_loss must be **HIGHER** than entry price (triggers when price rises)
  - Example: If opening LONG at $100,000, stop_loss must be < $100,000 (e.g., $99,000)
  - Example: If opening SHORT at $100,000, stop_loss must be > $100,000 (e.g., $101,000)
- profit_target: Min 3:1 reward/risk (HARD CONSTRAINT - engine enforces this)
  - **Directional requirement (CRITICAL)**:
    - For LONG positions: take_profit must be **HIGHER** than entry price (triggers when price rises)
    - For SHORT positions: take_profit must be **LOWER** than entry price (triggers when price falls)
```

**对比**:
- ✅ **更详细**：明确说明方向要求
- ✅ **有示例**：提供具体的价格示例
- ✅ **更清晰**：明确说明止损距离（Max 1% from entry）

### Adaptive.txt 模板（最详细）

**位置**: `prompts/adaptive.txt:364-403`

```364:403:prompts/adaptive.txt
# 风险管理协议 (强制)

每笔交易必须指定：

1. **profit_target** (止盈价格)
   - 最低盈亏比 2:1（盈利 = 2 × 亏损）
   - 基于技术阻力位、斐波那契、或波动带
   - 建议在技术位前 0.1-0.2% 设置（防止未成交）

2. **stop_loss** (止损价格)
   - 限制单笔亏损在账户 1-3%
   - 放置在关键支撑/阻力位之外
   - **滑点调整（V5.5.1 新增）**：
     - 做多：止损价格下移 0.05%（50,000 → 49,975）
     - 做空：止损价格上移 0.05%
     - 预留滑点缓冲，防止实际成交价偏移
```

**对比**:
- ✅ **最详细**：明确说明如何设置止盈止损
- ✅ **有具体方法**：说明基于技术阻力位、斐波那契、波动带
- ✅ **有滑点调整**：说明如何考虑滑点
- ✅ **有具体建议**：建议在技术位前 0.1-0.2% 设置

---

## 📝 结论

### Default.txt 的引导特点

1. **间接引导为主**:
   - 主要通过分析方法引导AI使用技术分析工具
   - 通过风险回报比约束引导AI满足要求
   - 但**没有明确说明**如何设置止盈止损价格

2. **依赖AI自主推断**:
   - AI需要**自主推断**如何使用支撑阻力位设置止盈止损
   - AI需要**自主计算**止盈止损价格来满足风险回报比≥1:3
   - AI需要**自主决定**止损距离和止盈目标

3. **系统验证作为保障**:
   - 系统会验证风险回报比≥3.0:1
   - 系统会验证方向正确性
   - 如果AI决策不符合要求，会被拒绝

### 改进建议

如果需要更明确的引导，可以考虑：

1. **添加明确的止盈止损设置方法**:
   - 说明如何使用支撑阻力位设置止盈止损
   - 说明如何使用斐波那契设置止盈止损
   - 说明如何使用波动带设置止盈止损

2. **添加方向要求说明**:
   - 明确说明做多时止损<止盈，做空时止损>止盈
   - 提供具体示例

3. **添加止损距离建议**:
   - 说明止损建议距离入场价≤1%
   - 说明如何根据波动率调整止损距离

4. **添加止盈目标建议**:
   - 说明止盈应该设置在技术阻力位附近
   - 说明建议在技术位前 0.1-0.2% 设置（防止未成交）

---

## 📌 总结

**Default.txt 的引导方式**：
- ✅ **间接引导**：通过分析方法、风险回报比约束引导AI
- ⚠️ **不够明确**：没有明确说明如何设置止盈止损价格
- ✅ **系统保障**：通过系统验证确保风险回报比≥3.0:1

**实际效果**：
- AI有**完全自主权**决定止盈止损价格
- AI需要**自主推断**如何使用技术分析工具
- 系统会**验证**风险回报比，确保符合要求


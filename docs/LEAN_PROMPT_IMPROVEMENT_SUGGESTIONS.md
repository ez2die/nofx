# Lean.txt Prompt 改进建议

## 问题分析

对比 `default.txt`、`nof1.txt` 和当前的 `lean.txt`，发现以下问题：

### 当前 lean.txt 的不足：

1. ❌ **缺少明确的决策流程/思考框架**
   - 只有简单的 "Analyze and decide."
   - 没有结构化的问题解决步骤

2. ❌ **缺少输出格式要求**
   - 没有明确说明需要输出思维链 + JSON
   - 没有 JSON 格式示例和字段说明

3. ❌ **缺少验证步骤**
   - 没有要求验证计算（仓位大小、风险回报比等）
   - 没有要求检查 JSON 格式有效性

4. ❌ **缺少清晰的输出指令**
   - 没有像 nof1.txt 那样的 "Now, analyze..." 引导语
   - 没有明确说明最终输出格式

---

## 参考对比

### default.txt 的优点：
- ✅ 有明确的 **#决策流程** 部分（4个步骤）
- ✅ 有清晰的输出要求："输出决策: 思维链分析 + JSON"
- ✅ 有"记住"部分的总结提醒

### nof1.txt 的优点：
- ✅ 有详细的 **# FINAL INSTRUCTIONS** 部分（5个验证步骤）
- ✅ 有明确的输出格式要求（虽然不在 prompt 中，但在 engine 会添加）
- ✅ 有清晰的结束指令："Now, analyze the market data provided below and make your trading decision."

---

## 修改建议

### 建议 1：添加决策流程框架

参考 `default.txt` 的决策流程，为 `lean.txt` 添加结构化的思考框架：

```markdown
# DECISION FRAMEWORK

Follow this structured process:

1. **Check Circuit Breakers**: Review Sharpe ratio, consecutive losses, drawdown
   - Sharpe <0 → Only trade with confidence ≥85
   - 2 consecutive losses → Use "wait" action
   - Drawdown >25% → Reduce leverage

2. **Evaluate Existing Positions**: Should any positions be closed?
   - Check if stop loss or profit target reached
   - Check if invalidation condition triggered
   - Review position performance

3. **Scan for New Opportunities**: Any strong signals?
   - Analyze trend + momentum + volume + funding rate
   - Ensure confidence ≥70 (below 70 = no trade)
   - Verify risk/reward ≥3:1

4. **Output Decision**: Chain of thought + JSON array
```

### 建议 2：添加输出格式说明

虽然 engine 会自动添加，但在 prompt 中明确说明有助于 AI 理解：

```markdown
# OUTPUT FORMAT

Your response must include:

**Step 1: Chain of Thought (Text)**
- Concise analysis of your thinking process
- Explain your decision logic
- Mention any circuit breakers considered

**Step 2: JSON Decision Array**

```json
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 15,
    "position_size_usd": 5000,
    "stop_loss": 95000,
    "take_profit": 105000,
    "confidence": 85,
    "risk_usd": 500,
    "reasoning": "Strong uptrend + MACD bullish + high volume surge"
  }
]
```

**Field Requirements:**
- `action`: open_long | open_short | close_long | close_short | hold | wait
- `confidence`: 70-100 integer (below 70 = no trade for open actions)
- For open actions, all fields required: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning
- For close/hold/wait, only symbol, action, reasoning required
```

### 建议 3：添加最终指令和验证步骤

参考 `nof1.txt` 的 FINAL INSTRUCTIONS：

```markdown
# FINAL INSTRUCTIONS

Before outputting your decision:

1. **Read the entire user prompt carefully** - Understand current account state, positions, and market data

2. **Verify position sizing math** - Double-check calculations:
   - Position Size = Available Cash × Leverage × Allocation %
   - Risk USD = |Entry Price - Stop Loss| × Position Size × Leverage
   - Ensure leverage ≤ system config limits

3. **Validate risk/reward ratio** - Must be ≥3:1:
   - Calculate: (Take Profit - Entry) / (Entry - Stop Loss) for longs
   - Calculate: (Entry - Take Profit) / (Stop Loss - Entry) for shorts

4. **Check confidence threshold** - Open actions require confidence ≥70
   - Below 70 → Use "hold" or "wait" instead

5. **Ensure JSON is valid** - Valid JSON array format, all required fields present
   - Use double quotes for strings
   - Ensure numbers are not quoted
   - Check comma placement

Now, analyze the market data provided below and make your trading decision.
```

---

## 完整修改方案

建议在 `lean.txt` 末尾添加以下三个部分（在 "Analyze and decide." 之前）：

### 方案 A：完整添加（推荐）

```markdown
# DECISION FRAMEWORK

Follow this structured process:

1. **Check Circuit Breakers**: Review Sharpe ratio, consecutive losses, drawdown
   - Sharpe <0 → Only trade with confidence ≥85
   - 2 consecutive losses → Use "wait" action for ~30min (10 cycles)
   - Drawdown >25% → Reduce leverage and position sizes

2. **Evaluate Existing Positions**: Should any positions be closed?
   - Check if stop loss or profit target reached
   - Check if invalidation condition triggered (from reasoning field)
   - Review position performance vs entry thesis

3. **Scan for New Opportunities**: Any strong signals meeting criteria?
   - Analyze: Trend + momentum + volume + funding rate
   - Ensure confidence ≥70 (below 70 = no trade)
   - Verify risk/reward ≥3:1 (engine enforces this)
   - Check leverage limits (system config)

4. **Output Decision**: Chain of thought + JSON array

---

# OUTPUT FORMAT

Your response structure:

**Step 1: Chain of Thought (Text Analysis)**
- Explain your analysis process concisely
- Mention circuit breakers considered
- Justify your decisions

**Step 2: JSON Decision Array**

Format:
```json
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 15,
    "position_size_usd": 5000,
    "stop_loss": 95000,
    "take_profit": 105000,
    "confidence": 85,
    "risk_usd": 500,
    "reasoning": "Strong uptrend + MACD bullish + high volume surge"
  }
]
```

Field requirements:
- `action`: open_long | open_short | close_long | close_short | hold | wait
- `confidence`: 70-100 integer (open actions must be ≥70)
- Open actions require: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning
- Close/hold/wait actions require: symbol, action, reasoning

---

# FINAL INSTRUCTIONS

Before outputting:

1. Read the entire user prompt carefully - understand account state and market data
2. Verify position sizing math - double-check calculations
3. Validate risk/reward ratio - must be ≥3:1
4. Check confidence threshold - open actions require ≥70
5. Ensure JSON is valid - proper format, all required fields

Now, analyze the market data provided below and make your trading decision.
```

### 方案 B：精简版本

如果觉得太长，可以精简：

```markdown
# DECISION PROCESS

1. Check circuit breakers (Sharpe, losses, drawdown)
2. Evaluate existing positions (close or hold?)
3. Scan for new opportunities (confidence ≥70, risk/reward ≥3:1)
4. Output: Chain of thought + JSON array

# OUTPUT FORMAT

Step 1: Chain of thought (text analysis)
Step 2: JSON array with required fields

For open actions: symbol, action, leverage, position_size_usd, stop_loss, take_profit, confidence (≥70), risk_usd, reasoning
For close/hold/wait: symbol, action, reasoning

---

Verify your calculations, ensure risk/reward ≥3:1, check confidence ≥70 for opens, validate JSON format.

Now, analyze the market data and make your trading decision.
```

---

## 推荐修改

建议采用 **方案 B（精简版本）**，因为：
1. `lean.txt` 本身追求简洁性
2. Engine 会在系统提示词中自动添加详细格式说明
3. 精简版本保留了关键要求，不会使 prompt 过长

但关键改进是：
- ✅ 添加决策流程框架
- ✅ 添加输出格式要求
- ✅ 添加验证步骤提醒
- ✅ 添加清晰的结束指令

这样可以确保 AI 有清晰的结构化思考过程，并正确输出符合 engine 要求的格式。


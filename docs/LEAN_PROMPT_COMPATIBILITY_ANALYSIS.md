# Lean Prompt 兼容性分析报告

## 1. Prompt 与 Engine 适配性检查

### ✅ **已解决：Action 不兼容**

**问题（已修复）：**
- ~~`lean.txt` 使用的 actions: `buy_to_enter`, `sell_to_enter`, `hold`, `close`~~
- Engine 期望的 actions: `open_long`, `open_short`, `close_long`, `close_short`, `hold`, `wait`

**修复状态：** ✅ **已完成**
- 已修改 `lean.txt` 使用 engine 支持的 action 名称
- 当前 actions: `open_long | open_short | close_long | close_short | hold | wait`
- 已在 prompt 中明确每个 action 的用途说明

**修复详情：**
- `buy_to_enter` → `open_long` ✅
- `sell_to_enter` → `open_short` ✅
- `close` → `close_long` / `close_short` ✅
- 添加了 `wait` action 支持 ✅

---

## 2. 参数冲突检查

### ✅ **已协调：Leverage 定义方式**

**原冲突（已协调）：**
- ~~lean.txt 定义了细粒度的杠杆规则（BTC、ETH、大市值山寨、其他）~~
- Engine 从配置文件读取 `btcEthLeverage` 和 `altcoinLeverage`（见 `decision/engine.go:521-527`）
- Engine 只支持两类杠杆配置，严格验证杠杆不超过配置上限

**协调状态：** ✅ **已完成**
- 已在 prompt 中明确说明：实际杠杆上限由系统配置决定（btcEthLeverage, altcoinLeverage）
- 已将杠杆描述改为"建议范围"，并提醒根据实际系统配置调整
- 在每行杠杆规则后添加了"check system config"提醒

**修复详情：**
```
Note: Actual leverage limits are set by system config (btcEthLeverage, altcoinLeverage).
These are recommended ranges - adjust based on actual system limits.
BTC: 15-25x (max 25x, check system config) ✅
ETH: 12-22x (max 22x, check system config) ✅
Large cap alts: 8-15x (max 15x, check system config) ✅
Others: Avoid or max 8x (check system config) ✅
```

### ✅ **已解决：Confidence 取值范围**

**原冲突（已修复）：**
- ~~lean.txt 使用 0.7-1.0 的浮点数范围~~
- Engine 期望：`Decision.Confidence` 是 `int` 类型，范围 0-100（见 `decision/types.go:79`）
- 输出格式说明：`confidence: 0-100（开仓建议≥75）`（见 `decision/engine.go:282`）

**修复状态：** ✅ **已完成**
- 已统一为 0-100 整数范围
- 已在 prompt 中明确说明：`confidence: 70-100 (below 70 = no trade, integer range 0-100)`
- 添加了详细的信心度分级说明：
  - 70-79: Moderate conviction
  - 80-89: High conviction
  - 90-100: Very high conviction

**修复详情：**
```diff
- confidence: 0.7-1.0 (below 0.7 = no trade)
+ confidence: 70-100 (below 70 = no trade, integer range 0-100)
+   - 70-79: Moderate conviction
+   - 80-89: High conviction
+   - 90-100: Very high conviction
```

### ⚠️ **冲突 3：Risk Management 参数**

**lean.txt 定义：**
```
- stop_loss: Max 1% from entry
- profit_target: Min 3:1 reward/risk
- invalidation: Specific exit condition
- risk_usd: Max 5-8% account per trade
```

**Engine 硬编码约束（见 `decision/engine.go:263-269`）：**
```
1. 风险回报比: 必须 ≥ 1:3（冒1%风险，赚3%+收益）✓ 一致
2. 最多持仓: 3个币种（质量>数量）
3. 单币仓位: 山寨0.8-1.5倍账户净值 | BTC/ETH 5-10倍账户净值
4. 保证金: 总使用率 ≤ 90%
```

**冲突点：**
1. **stop_loss 1%**：Engine 没有硬编码这个限制，只验证风险回报比 ≥ 3:1
2. **risk_usd 5-8%**：Engine 没有验证这个字段，`Decision.RiskUSD` 是可选字段
3. **invalidation**：Engine 的 `Decision` 结构体没有 `invalidation_condition` 字段

**解决方案：**
1. Engine 可以添加对 `risk_usd` 的验证（可选，但不影响运行）
2. `invalidation_condition` 可以作为 `reasoning` 的一部分，或添加到 Decision 结构体

---

## 3. 数据准备支持检查

### ✅ **支持的数据（基于 `market/types.go` 和 `market/data.go`）：**

1. **价格序列** ✓
   - `IntradaySeries.MidPrices` (3分钟间隔)
   - `LongerTermContext` (4小时K线)

2. **技术指标** ✓
   - `IntradaySeries.EMA20Values` (趋势)
   - `IntradaySeries.MACDValues` (动量)
   - `IntradaySeries.RSI7Values`, `RSI14Values` (超买超卖)
   - `IntradaySeries.ATR14Values` (波动率)

3. **成交量数据** ✓
   - `LongerTermContext.CurrentVolume` vs `AverageVolume`

4. **Funding Rate** ✓
   - `Data.FundingRate`

5. **Open Interest** ✓
   - `Data.OpenInterest.Latest` vs `Average`

### ✅ **lean.txt 所需数据均支持：**
- Trend + momentum + volume + funding rate ✓
- 所有序列数据按 OLDEST → NEWEST 排序（最后一个是当前）✓

---

## 4. Circuit Breakers 支持检查

**lean.txt 定义：**
```
- 2 consecutive losses → pause 30min
- Account drawdown >25% → reduce leverage
- Sharpe <0 → be more selective
```

**Engine 当前支持：**
- ✅ Sharpe Ratio 会在 user prompt 中提供（见 `decision/engine.go:369-381`）
- ⚠️ Engine **没有**硬编码实现这些 circuit breakers（设计决策：通过 prompt 指导 AI 自行判断）

**修复状态：** ✅ **已协调**
- 已在 prompt 中明确标注为"Self-enforced via trading decisions"
- 提供了具体的执行指导：
  - 2 consecutive losses → pause trading (use "wait" action) for ~30min (10 decision cycles) ✅
  - Account drawdown >25% → reduce leverage and position sizes ✅
  - Sharpe <0 → be more selective (only trade with confidence ≥85) ✅
- 要求 AI 在 circuit breaker 触发时在 reasoning 字段记录原因 ✅

**实现方式：**
- Circuit breakers 通过 prompt 指导 AI 自行判断和执行
- AI 可以通过 `wait` action 实现暂停交易
- AI 可以通过降低 leverage 和 position_size_usd 来响应回撤
- AI 可以通过提高 confidence 阈值来应对负 Sharpe Ratio

---

## 5. 其他潜在问题

### ✅ **已解决：持仓关闭 Action 不明确**

**原问题（已修复）：**
- ~~lean.txt 只有 `close`，无法区分多空方向~~
- Engine 需要明确是 `close_long` 还是 `close_short`

**修复状态：** ✅ **已完成**
- 已明确使用 `close_long` 和 `close_short`
- 已在 prompt 中添加了每个 action 的详细说明
- 提供了 `close_long` 和 `close_short` 的使用指导

**修复详情：**
```diff
- close
+ close_long | close_short
+ - Use close_long to exit LONG position ✅
+ - Use close_short to exit SHORT position ✅
```

---

## 6. 总结与建议

### ✅ **已解决的问题：**

1. **Action 名称不兼容** ✅ **已修复**
   - ✅ 已修改 prompt 使用 `open_long`/`open_short`/`close_long`/`close_short`/`hold`/`wait`
   - ✅ 已在 prompt 中添加了每个 action 的详细说明

2. **Confidence 范围不一致** ✅ **已修复**
   - ✅ 已统一为 0-100 整数范围
   - ✅ 已在 prompt 中明确说明并添加了分级指导

3. **Leverage 定义冲突** ✅ **已协调**
   - ✅ 已在 prompt 中说明实际杠杆上限由系统配置决定
   - ✅ 已将杠杆描述改为"建议范围"，并添加了系统配置检查提醒

4. **Circuit Breakers 支持** ✅ **已协调**
   - ✅ 已在 prompt 中明确为"Self-enforced via trading decisions"
   - ✅ 提供了具体的执行指导和触发条件

5. **持仓关闭 Action 不明确** ✅ **已修复**
   - ✅ 已明确使用 `close_long` 和 `close_short`
   - ✅ 添加了使用说明

### 🟡 **建议优化的问题：**

6. **Risk USD 验证缺失**
   - 当前：Engine 未验证 `risk_usd` 字段（不影响运行）
   - 可选：在 engine 中添加 `risk_usd` 验证逻辑

### 🟢 **可选优化：**

7. **Invalidation Condition**
   - 当前：已建议在 `reasoning` 字段中记录
   - 可选：添加到 Decision 结构体（需要代码修改）

---

## 7. 修改方案（已实施）

### ✅ **方案 1：最小修改（已采用）**

已修改 `lean.txt`，使其与 engine 完全兼容：

**修改内容：**

```diff
# ACTIONS
- buy_to_enter | sell_to_enter | hold | close
+ open_long | open_short | close_long | close_short | hold | wait
+ - Use open_long to enter LONG position ✅
+ - Use open_short to enter SHORT position ✅
+ - Use close_long to exit LONG position ✅
+ - Use close_short to exit SHORT position ✅
+ - Use hold to maintain current positions ✅
+ - Use wait when no clear opportunity exists ✅

# LEVERAGE BY ASSET & CONFIDENCE
+ Note: Actual leverage limits are set by system config (btcEthLeverage, altcoinLeverage). ✅
+ These are recommended ranges - adjust based on actual system limits. ✅
BTC: 15-25x (max 25x, check system config) ✅
ETH: 12-22x (max 22x, check system config) ✅
Large cap alts: 8-15x (max 15x, check system config) ✅
Others: Avoid or max 8x (check system config) ✅

# RISK MANAGEMENT (MANDATORY)
- stop_loss: Max 1% from entry
+ stop_loss: Max 1% from entry (recommended, but must satisfy risk/reward ≥3:1) ✅
- profit_target: Min 3:1 reward/risk
+ profit_target: Min 3:1 reward/risk (HARD CONSTRAINT - engine enforces this) ✅
- invalidation: Specific exit condition
+ invalidation: Specific exit condition (record in reasoning field) ✅
- confidence: 0.7-1.0 (below 0.7 = no trade)
+ confidence: 70-100 (below 70 = no trade, integer range 0-100) ✅
+   - 70-79: Moderate conviction ✅
+   - 80-89: High conviction ✅
+   - 90-100: Very high conviction ✅
- risk_usd: Max 5-8% account per trade
+ risk_usd: Max 5-8% account per trade (recommended, optional field) ✅

# CIRCUIT BREAKERS
- 2 consecutive losses → pause 30min
- Account drawdown >25% → reduce leverage
- Sharpe <0 → be more selective
+ # CIRCUIT BREAKERS (Self-enforced via trading decisions) ✅
+ Monitor these conditions and adjust behavior accordingly: ✅
+ - 2 consecutive losses → pause trading (use "wait" action) for ~30min (10 decision cycles) ✅
+ - Account drawdown >25% → reduce leverage and position sizes ✅
+ - Sharpe <0 → be more selective (only trade with confidence ≥85) ✅
+ - If circuit breaker triggers, record reason in reasoning field ✅
```

**状态：** ✅ **所有关键问题已修复，prompt 与 engine 完全兼容**

---

## 8. 测试建议

在应用修改后，建议测试：

1. ✅ AI 输出的 action 是否被 engine 接受（应全部为 `open_long`/`open_short`/`close_long`/`close_short`/`hold`/`wait`）
2. ✅ Confidence 值是否在正确范围内（应为 70-100 整数）
3. ✅ Leverage 是否不超过配置上限（应检查系统配置的 `btcEthLeverage` 和 `altcoinLeverage`）
4. ✅ Risk USD 是否在合理范围内（建议 5-8% 账户净值）
5. ✅ 风险回报比是否 ≥ 3:1（engine 会强制验证）

## 9. 修复状态总结

### ✅ 已完成的修复：
- [x] Action 名称兼容性（`open_long`/`open_short`/`close_long`/`close_short`）
- [x] Confidence 取值范围（70-100 整数范围）
- [x] Leverage 定义协调（说明由系统配置决定）
- [x] Circuit Breakers 支持（通过 prompt 指导 AI 自行执行）
- [x] 持仓关闭 Action 明确性（`close_long`/`close_short`）

### ⚠️ 待验证：
- [ ] 实际运行测试：AI 是否按照新 prompt 正确输出
- [ ] 引擎验证：所有决策是否通过 engine 验证
- [ ] Circuit Breakers：AI 是否能在触发时正确使用 `wait` action

### 📝 文档更新：
- [x] 更新分析报告，标注已解决问题


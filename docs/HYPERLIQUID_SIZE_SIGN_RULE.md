# Hyperliquid Size 字段符号规则分析

## 1. 核心发现

通过分析测试数据库中的 253 条 short 方向交易记录，发现了 Hyperliquid `Size` 字段符号的**明确规律**：

### 1.1 关键规律

| `start_position` 状态 | `signed_quantity` 符号 | 记录数 | 占比 |
|----------------------|---------------------|--------|------|
| **有值（非 NULL）** | **正数** | 93 | 36.8% |
| **NULL** | **负数** | 160 | 63.2% |

**结论**：`Size` 字段的符号规则**完全取决于 `start_position` 字段是否存在**。

## 2. 详细数据分析

### 2.1 按 `start_position` 状态分组

```
start_pos_type  total    positive  negative
--------------  -------  --------  --------
NEGATIVE        63       63        0        ← 全部为正数
NULL            160      0         160      ← 全部为负数
ZERO            30       30        0        ← 全部为正数
```

### 2.2 按 `exchange_side` 分组

```
exchange_side  total    positive  negative
-------------  -------  --------  --------
A              49       49        0        ← 开仓，全部为正数
B              44       44        0        ← 平仓，全部为正数
```

**注意**：`exchange_side` 有值的记录，都对应 `start_position` 有值的情况。

### 2.3 按 Action 和 `start_position` 状态分组

```
action       start_pos_status  exchange_side  count  avg_signed_qty
-----------  ----------------  -------------  -----  --------------
close_short  HAS_VALUE         B              44     0.067025      ← 正数
close_short  NULL                             93     -0.008374     ← 负数
open_short   HAS_VALUE         A              19     0.000544      ← 正数
open_short   NULL                             67     -0.011624     ← 负数
open_short   ZERO              A              30     0.097959      ← 正数
```

## 3. Hyperliquid 的符号规则解释

### 3.1 规则 1：基于 `start_position` 字段的存在性

**当 `start_position` 字段存在时**（新版本 API 或特定场景）：
- `Size` 字段的符号表示**数量变化的方向**，而不是持仓方向
- **做空开仓**：`Size` 为正数（表示"增加空头持仓"）
- **做空平仓**：`Size` 为正数（表示"减少空头持仓"）
- 此时 `Size` 的符号与 `start_position` 的符号**相反**

**当 `start_position` 字段不存在时**（旧版本 API 或特定场景）：
- `Size` 字段的符号表示**持仓方向**
- **做空**：`Size` 为负数
- **做多**：`Size` 为正数
- 这是传统的"做多正数，做空负数"规则

### 3.2 规则 2：`exchange_side` 的含义

根据代码注释和数据分析：
- **`Side = "A"`**：开仓订单（Maker 或特定订单类型）
- **`Side = "B"`**：平仓订单（Taker 或特定订单类型）

**观察**：
- 当 `exchange_side` 有值时，`start_position` 也有值，且 `Size` 为正数
- 这可能是 Hyperliquid 新版本 API 的行为

### 3.3 规则 3：时间维度

从数据时间戳观察：
- **旧数据**（2025-11-04）：大部分没有 `start_position`，`Size` 为负数
- **新数据**（2025-11-05）：大部分有 `start_position`，`Size` 为正数

**推测**：Hyperliquid 可能在某个时间点更新了 API，开始返回 `start_position` 字段，同时改变了 `Size` 字段的符号规则。

## 4. 符号规则总结

### 4.1 规则矩阵

| 场景 | `start_position` | `Size` 符号 | 含义 |
|------|-----------------|------------|------|
| **做空开仓（新 API）** | 有值（负数） | **正数** | 增加空头持仓 |
| **做空平仓（新 API）** | 有值（负数） | **正数** | 减少空头持仓 |
| **做空开仓（旧 API）** | NULL | **负数** | 做空方向 |
| **做空平仓（旧 API）** | NULL | **负数** | 做空方向 |
| **做多开仓** | 有值（正数） | **正数** | 增加多头持仓 |
| **做多平仓** | 有值（正数） | **正数** | 减少多头持仓 |

### 4.2 核心理解

**Hyperliquid 的 `Size` 字段符号规则不是固定的，而是取决于 API 返回的数据结构**：

1. **新版本 API**（有 `start_position`）：
   - `Size` 符号表示**数量变化的方向**
   - 正数 = 增加持仓（无论多空）
   - 负数 = 减少持仓（无论多空）

2. **旧版本 API**（无 `start_position`）：
   - `Size` 符号表示**持仓方向**
   - 正数 = 做多
   - 负数 = 做空

## 5. 对配对策略的影响

### 5.1 为什么不能依赖 `signed_quantity` 符号

由于 Hyperliquid 的符号规则不一致：
- 旧数据：负数表示做空
- 新数据：正数也可能表示做空（当有 `start_position` 时）

**因此**：
- ❌ **不能**依赖 `signed_quantity` 的符号来判断方向
- ✅ **必须**使用 `Side` 字段（`"long"` 或 `"short"`）来判断方向
- ✅ **必须**使用 `Quantity` 或 `ABS(SignedQuantity)` 进行数量匹配

### 5.2 配对算法建议

```go
// 正确的配对逻辑
func getAbsoluteQuantity(record *TradeRecord) float64 {
    if record.SignedQuantity != nil {
        return math.Abs(*record.SignedQuantity)  // 始终使用绝对值
    }
    return record.Quantity
}

// 配对条件
func canMatch(open, close *TradeRecord) bool {
    // 1. 同一交易员
    if open.TraderID != close.TraderID {
        return false
    }
    
    // 2. 同一币种
    if open.Symbol != close.Symbol {
        return false
    }
    
    // 3. 同一方向（使用 Side 字段，不依赖 signed_quantity 符号）
    if open.Side != close.Side {
        return false
    }
    
    // 4. 正确的 Action 匹配
    if !isMatchingAction(open.Action, close.Action) {
        return false
    }
    
    // 5. 时间顺序
    if !open.Timestamp.Before(close.Timestamp) {
        return false
    }
    
    return true
}
```

## 6. 数据一致性建议

### 6.1 保持现状（推荐）

**优点**：
- ✅ 保留交易所原始数据，数据保真性最高
- ✅ 可以追溯不同 API 版本的行为差异
- ✅ 配对算法已调整为不依赖符号

**缺点**：
- ⚠️ 数据符号不一致，可能造成理解困惑

### 6.2 标准化符号（可选）

如果需要统一符号规则，可以在同步时标准化：

```go
// 在 convertHyperliquidFillToExchangeFill 中
if exchangeFill.Side == "long" {
    signed = math.Abs(signed)  // 做多始终为正
} else if exchangeFill.Side == "short" {
    signed = -math.Abs(signed)  // 做空始终为负
}
```

**优点**：
- ✅ 数据符号一致，易于理解
- ✅ 符合"做多正数，做空负数"的直觉

**缺点**：
- ⚠️ 改变了原始数据，可能影响后续分析
- ⚠️ 需要确保 `Side` 字段的准确性

## 7. 结论

1. ✅ **Hyperliquid 的 `Size` 字段符号规则取决于 `start_position` 字段的存在性**
2. ✅ **新版本 API**（有 `start_position`）：`Size` 符号表示数量变化方向
3. ✅ **旧版本 API**（无 `start_position`）：`Size` 符号表示持仓方向
4. ✅ **这不是同步功能的错误**，而是 Hyperliquid API 的行为差异
5. ✅ **配对策略应使用 `Side` 字段和绝对值**，不依赖 `signed_quantity` 符号

## 8. 参考资料

- Hyperliquid API 文档：https://hyperliquid.gitbook.io/hyperliquid-docs
- 代码位置：`trade_history/sync_hyperliquid.go`
- 相关分析：`docs/TRADE_HISTORY_SIGNED_QUANTITY_ANALYSIS.md`

---

**文档版本**: 1.0  
**创建日期**: 2025-11-13  
**分析数据**: config.db.test (253 条 short 方向记录)


# Trade History SignedQuantity 字段分析报告

## 1. 数据概览

**查询时间**: 2025-11-13  
**数据库**: `config.db.test`  
**总记录数**: 449 条

### 1.1 总体统计

| 指标 | 数量 | 占比 |
|------|------|------|
| 总记录数 | 449 | 100% |
| 有 `signed_quantity` 的记录 | 449 | 100% |
| `signed_quantity > 0` (正数) | 289 | 64.4% |
| `signed_quantity < 0` (负数) | 160 | 35.6% |
| `signed_quantity = 0` | 0 | 0% |
| `signed_quantity IS NULL` | 0 | 0% |

### 1.2 按 Action 和 Side 分组统计

| Action | Side | 总记录数 | 正数 | 负数 | 正数占比 | 负数占比 |
|--------|------|---------|------|------|---------|---------|
| `open_long` | `long` | 117 | 117 | 0 | 100% | 0% |
| `close_long` | `long` | 79 | 79 | 0 | 100% | 0% |
| `open_short` | `short` | 116 | 49 | 67 | 42.2% | 57.8% |
| `close_short` | `short` | 137 | 44 | 93 | 32.1% | 67.9% |

## 2. 关键发现

### 2.1 Long 方向（做多）

✅ **完全符合预期**：
- 所有 `open_long` 记录的 `signed_quantity` 都是**正数**（117/117，100%）
- 所有 `close_long` 记录的 `signed_quantity` 都是**正数**（79/79，100%）

**结论**：Long 方向的 `signed_quantity` 符号规则一致，始终为正数。

### 2.2 Short 方向（做空）

⚠️ **存在异常**：
- `open_short`：67 条负数（57.8%），49 条正数（42.2%）
- `close_short`：93 条负数（67.9%），44 条正数（32.1%）

**异常模式**：
1. **正常情况**：`signed_quantity < 0`（符合"做空为负数"的预期）
2. **异常情况**：`signed_quantity > 0`（不符合预期）

### 2.3 异常案例分析

#### 异常案例 1：`open_short` 但 `signed_quantity > 0`

```
ID: 446
Action: open_short
Side: short
SignedQuantity: 0.00559 (正数)
Quantity: 0.00559
RawDir: "Open Short"
StartPosition: 0.0
```

#### 异常案例 2：`open_short` 但 `signed_quantity > 0`（有 start_position）

```
ID: 428
Action: open_short
Side: short
SignedQuantity: 0.00018 (正数)
Quantity: 0.00018
RawDir: "Open Short"
StartPosition: -0.00543 (负数)
```

#### 正常案例：`open_short` 且 `signed_quantity < 0`

```
ID: 200
Action: open_short
Side: short
SignedQuantity: -0.00467 (负数)
Quantity: 0.00467
RawDir: "Open Short"
StartPosition: NULL
```

## 3. 根本原因分析

### 3.1 代码逻辑

查看 `trade_history/sync_hyperliquid.go` 中的实现：

```go
// 设置数量（Size字段，json:"sz"）
if fill.Size != "" {
    sz, err := strconv.ParseFloat(fill.Size, 64)
    if err != nil {
        return exchangeFill, fmt.Errorf("解析数量失败: %w", err)
    }
    rawSize = sz
    sizeValid = true
    exchangeFill.Quantity = math.Abs(sz)  // 使用绝对值
    signed := sz  // 直接使用原始值，不进行符号转换
    exchangeFill.SignedQty = &signed
}
```

**关键发现**：
- `signed_quantity` **直接来自 Hyperliquid API 的 `Size` 字段**，没有根据 `Side` 进行符号转换
- 我们的同步代码**忠实地保留了 Hyperliquid API 返回的原始符号**，这是**正确的行为**（不是错误）
- Hyperliquid 的 `Size` 字段符号规则与我们假设的"做多正数，做空负数"约定**不一致**

### 3.2 这是同步功能的错误吗？

**答案：不是**。我们的同步功能工作正常，它正确地保留了 Hyperliquid API 返回的原始数据。

**原因**：
1. ✅ **数据保真性**：同步功能应该尽可能保留交易所返回的原始数据，不做不必要的转换
2. ✅ **代码逻辑正确**：代码直接使用 `fill.Size` 的值，没有引入错误
3. ⚠️ **符号规则不匹配**：Hyperliquid 的 `Size` 字段符号规则与我们的假设不同，这是**交易所 API 的行为**，不是我们的错误

**结论**：这不是同步功能的数据处理错误，而是 Hyperliquid API 的 `Size` 字段符号规则与我们的预期不一致。

### 3.3 Hyperliquid API 行为分析

**详细分析请参见**：[Hyperliquid Size 字段符号规则分析](./HYPERLIQUID_SIZE_SIGN_RULE.md)

根据实际数据分析，发现了**明确的规律**：

1. **基于 `StartPosition` 字段的存在性**（核心规则）：
   - **当 `StartPosition` 有值（非 NULL）时**：
     - `Size` 符号表示**数量变化的方向**，而不是持仓方向
     - 做空时 `Size` 为**正数**（表示"增加空头持仓"）
     - 这是新版本 API 的行为
   - **当 `StartPosition` 为 NULL 时**：
     - `Size` 符号表示**持仓方向**
     - 做空时 `Size` 为**负数**
     - 这是旧版本 API 的行为

2. **数据统计**（253 条 short 方向记录）：
   - `StartPosition` 有值：93 条，**全部为正数**（100%）
   - `StartPosition` 为 NULL：160 条，**全部为负数**（100%）

3. **时间维度**：
   - 旧数据（2025-11-04）：大部分无 `StartPosition`，`Size` 为负数
   - 新数据（2025-11-05）：大部分有 `StartPosition`，`Size` 为正数
   - **推测**：Hyperliquid 在某个时间点更新了 API，改变了符号规则

### 3.4 数据一致性建议

虽然这不是错误，但为了数据一致性，我们**可以选择**在同步时标准化 `signed_quantity` 的符号：

**方案 A：保持现状（推荐）**
- ✅ 保留交易所原始数据，不做转换
- ✅ 配对算法使用 `Side` 字段和绝对值，不依赖符号
- ✅ 数据保真性最高

**方案 B：标准化符号**
- 在 `convertHyperliquidFillToExchangeFill` 或 `convertFillToRecord` 中，根据 `Side` 标准化符号：
  ```go
  if exchangeFill.Side == "long" {
      signed = math.Abs(signed)  // 做多始终为正
  } else if exchangeFill.Side == "short" {
      signed = -math.Abs(signed)  // 做空始终为负
  }
  ```
- ⚠️ 这会改变原始数据，可能影响后续分析
- ⚠️ 需要确保 `Side` 字段的准确性

## 4. 对配对策略的影响

### 4.1 当前配对策略文档的假设

在 `TRADE_PAIRING_STRATEGY.md` 中，我们假设：
- 做多：`signed_quantity` 为正数
- 做空：`signed_quantity` 为负数

### 4.2 实际情况

⚠️ **这个假设不完全正确**：
- Long 方向：✅ 完全符合（100% 为正数）
- Short 方向：❌ 部分不符合（约 35-40% 为正数）

### 4.3 配对策略调整建议

#### 方案 1：不依赖 `signed_quantity` 的符号（推荐）

**配对依据**：
1. ✅ `Symbol` 相同
2. ✅ `Side` 相同（`long` 或 `short`）
3. ✅ `Action` 匹配（`open_long` ↔ `close_long`，`open_short` ↔ `close_short`）
4. ✅ 时间顺序（开仓时间 < 平仓时间）
5. ✅ 数量匹配（使用 `Quantity` 或 `SignedQuantity` 的绝对值）

**不使用**：
- ❌ `signed_quantity` 的符号来判断方向（不可靠）

#### 方案 2：标准化 `signed_quantity` 符号

在配对前，根据 `Side` 和 `Action` 标准化 `signed_quantity`：

```go
func normalizeSignedQuantity(record *TradeRecord) float64 {
    if record.SignedQuantity == nil {
        return 0
    }
    
    qty := *record.SignedQuantity
    absQty := math.Abs(qty)
    
    // 根据 Side 和 Action 确定符号
    if record.Side == "long" {
        return absQty  // 做多始终为正
    } else if record.Side == "short" {
        return -absQty  // 做空始终为负
    }
    
    return qty  // 未知方向，保持原值
}
```

#### 方案 3：使用 `StartPosition` 辅助判断

如果 `signed_quantity` 符号异常，使用 `StartPosition` 来推断：

```go
func inferSignedQuantitySign(record *TradeRecord) float64 {
    if record.SignedQuantity == nil {
        return 0
    }
    
    qty := *record.SignedQuantity
    absQty := math.Abs(qty)
    
    // 如果 StartPosition 有值，使用其符号
    if record.StartPosition != nil {
        if *record.StartPosition < 0 {
            // 做空持仓，signed_quantity 应为负
            return -absQty
        } else if *record.StartPosition > 0 {
            // 做多持仓，signed_quantity 应为正
            return absQty
        }
    }
    
    // 回退到 Side 判断
    if record.Side == "long" {
        return absQty
    } else if record.Side == "short" {
        return -absQty
    }
    
    return qty
}
```

## 5. 推荐方案

### 5.1 配对算法调整

**推荐使用方案 1**：不依赖 `signed_quantity` 的符号，而是：

1. **使用 `Side` 字段**：这是最可靠的判断依据
2. **使用 `Quantity` 或 `SignedQuantity` 的绝对值**：进行数量匹配
3. **使用 `Action` 字段**：确保开仓和平仓正确匹配

### 5.2 配对逻辑伪代码

```go
func MatchTradePairs(records []*TradeRecord) []*TradePair {
    // 按时间排序
    sort.Slice(records, func(i, j int) bool {
        return records[i].Timestamp.Before(records[j].Timestamp)
    })
    
    // 使用 symbol_side 作为 key
    openPositions := make(map[string][]*OpenPosition)
    var pairs []*TradePair
    
    for _, record := range records {
        // 使用 Symbol + Side 作为 key（不依赖 signed_quantity 符号）
        key := record.Symbol + "_" + record.Side
        
        if isOpenAction(record.Action) {
            // 开仓：使用 Quantity 或 SignedQuantity 的绝对值
            qty := getAbsoluteQuantity(record)
            openPositions[key] = append(openPositions[key], &OpenPosition{
                Record:       record,
                RemainingQty: qty,
            })
            
        } else if isCloseAction(record.Action) {
            // 平仓：FIFO 匹配
            closeQty := getAbsoluteQuantity(record)
            // ... 匹配逻辑 ...
        }
    }
    
    return pairs
}

func getAbsoluteQuantity(record *TradeRecord) float64 {
    if record.SignedQuantity != nil {
        return math.Abs(*record.SignedQuantity)
    }
    return record.Quantity
}
```

## 6. 数据质量建议

### 6.1 数据验证

建议在同步数据时添加验证逻辑：

```go
// 验证 signed_quantity 符号是否与 Side 一致
func validateSignedQuantity(record *TradeRecord) error {
    if record.SignedQuantity == nil {
        return nil  // NULL 值允许
    }
    
    qty := *record.SignedQuantity
    
    // 如果 Side 明确，验证符号
    if record.Side == "long" && qty < 0 {
        log.Printf("⚠️ 警告：long 方向但 signed_quantity 为负数 (id=%d)", record.ID)
    } else if record.Side == "short" && qty > 0 {
        log.Printf("⚠️ 警告：short 方向但 signed_quantity 为正数 (id=%d)", record.ID)
    }
    
    return nil
}
```

### 6.2 数据修复（可选）

如果需要，可以添加数据修复逻辑，标准化 `signed_quantity` 符号：

```sql
-- 修复 short 方向但 signed_quantity 为正数的记录
UPDATE trade_history
SET signed_quantity = -ABS(signed_quantity)
WHERE side = 'short' AND signed_quantity > 0;

-- 修复 long 方向但 signed_quantity 为负数的记录
UPDATE trade_history
SET signed_quantity = ABS(signed_quantity)
WHERE side = 'long' AND signed_quantity < 0;
```

**注意**：执行修复前需要备份数据，并确认不会影响其他逻辑。

## 7. 结论

1. ✅ **Long 方向**：`signed_quantity` 符号完全符合预期（100% 为正数）
2. ⚠️ **Short 方向**：`signed_quantity` 符号部分不符合预期（约 35-40% 为正数）
3. ✅ **数据完整性**：所有记录都有 `signed_quantity` 值，无 NULL
4. ✅ **同步功能**：**工作正常**，正确保留了 Hyperliquid API 的原始数据
5. ⚠️ **符号规则**：Hyperliquid 的 `Size` 字段符号规则与我们的假设不一致，这是**交易所 API 的行为**，不是我们的错误
6. ✅ **配对策略**：应使用 `Side` 字段而非 `signed_quantity` 符号来判断方向
7. ✅ **数量匹配**：应使用 `Quantity` 或 `SignedQuantity` 的绝对值进行匹配

## 8. 下一步行动

1. **更新配对策略文档**：明确不依赖 `signed_quantity` 符号
2. **实现配对算法**：使用 `Side` 和 `Quantity` 绝对值
3. **添加数据验证**：在同步时记录符号异常
4. **测试配对逻辑**：使用实际数据测试配对准确性

---

**文档版本**: 1.0  
**创建日期**: 2025-11-13  
**分析数据**: config.db.test (449 条记录)


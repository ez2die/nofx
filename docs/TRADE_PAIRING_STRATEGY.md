# Trade Pairing Strategy
# 交易配对策略文档

## 1. 概述

本文档详细说明如何将 `trade_history` 表中的开仓和平仓记录进行配对，以生成完整的交易对（Trade Pair）用于统计分析。

## 2. 配对目标

- **生成完整交易对**：将开仓记录（`open_long`/`open_short`）与对应的平仓记录（`close_long`/`close_short`）进行匹配
- **计算持仓时间**：从开仓时间到平仓时间的差值
- **验证 PnL 一致性**：确保配对后的 PnL 与平仓记录的 PnL 一致
- **处理复杂场景**：支持部分平仓、多次开仓、多次平仓等情况

## 3. 配对规则

### 3.1 基本匹配条件

两个记录可以配对，必须满足以下**所有条件**：

1. **同一交易员**：`TraderID` 相同
2. **同一币种**：`Symbol` 相同
3. **同一方向**：
   - `open_long` 只能与 `close_long` 配对
   - `open_short` 只能与 `close_short` 配对
   - 或者通过 `Side` 字段判断：`Side = "long"` 或 `Side = "short"`
4. **时间顺序**：开仓时间必须早于平仓时间
5. **数量匹配**：开仓数量与平仓数量可以完全或部分匹配（FIFO 规则）

### 3.2 FIFO（先进先出）原则

- **核心原则**：最早的开仓记录优先与最早的平仓记录匹配
- **处理顺序**：所有记录按 `Timestamp` 升序排序后处理
- **部分匹配**：支持一个开仓记录对应多个平仓记录，或一个平仓记录对应多个开仓记录

### 3.3 数量处理

#### 3.3.1 数量字段说明

- **`Quantity`**：绝对数量（总是正数）
- **`SignedQuantity`**：带符号数量
  - **当前数据状态**（2025-11-13 重新同步后）：
    - **Long 方向**：100% 为正数 ✅
    - **Short 方向**：100% 为正数 ✅（新 API 格式）
    - **所有记录**：100% 有 `signed_quantity` 值，无 NULL ✅
  - **API 版本**：当前使用新版本 API，所有记录都有 `start_position` 字段
  - **符号规则**：在新 API 中，`Size` 符号表示**数量变化方向**，而不是持仓方向
    - Short 方向时，`Size` 为正数表示"增加/减少空头持仓"
  - **详细分析**：参见 [SignedQuantity 字段分析报告](./TRADE_HISTORY_SIGNED_QUANTITY_ANALYSIS.md) 和 [重新检查报告](./TRADE_HISTORY_SIGNED_QUANTITY_RECHECK.md)

#### 3.3.2 数量匹配逻辑

✅ **当前数据格式统一**：所有记录的 `signed_quantity` 都是正数，但配对策略仍应使用绝对值和 `Side` 字段以确保健壮性。

1. **使用 `Side` 字段判断方向**（必需）：
   - **不依赖 `signed_quantity` 的符号**（虽然当前数据都是正数，但为兼容性考虑）
   - 使用 `Side` 字段（`"long"` 或 `"short"`）来判断方向
   - 这是最可靠的判断依据

2. **数量匹配使用绝对值**：
   - 如果 `SignedQuantity` 不为 `NULL`，使用 `ABS(SignedQuantity)` 进行匹配
   - 如果 `SignedQuantity` 为 `NULL`，使用 `Quantity` 字段
   - **匹配时始终使用绝对值**，不依赖符号
   - **当前数据状态**：所有记录都有 `signed_quantity`，且都是正数

3. **数量计算**：
   - 匹配时使用**绝对值**进行比较
   - 部分匹配时，按比例分配 PnL（如果需要）
   - 不依赖 `signed_quantity` 的符号来判断方向

4. **数据质量保证**：
   - 当前数据：100% 记录有 `signed_quantity`，100% 为正数
   - 所有记录都有 `start_position` 字段（新 API 格式）
   - 配对算法可以安全地使用 `ABS(SignedQuantity)` 或 `Quantity`

## 4. 配对算法

### 4.1 算法流程

```
1. 查询所有交易记录（按时间排序）
   ├─ 过滤条件：TraderID, Symbol (可选), Side (可选), 时间范围
   └─ 排序：ORDER BY timestamp ASC

2. 初始化数据结构
   ├─ openPositions: map[symbol_side][]OpenPosition
   └─ completedPairs: []TradePair

3. 遍历每条记录（按时间顺序）
   ├─ 如果是开仓记录（open_long/open_short）
   │  └─ 添加到 openPositions[symbol_side] 队列末尾
   │
   └─ 如果是平仓记录（close_long/close_short）
      └─ 从 openPositions[symbol_side] 队列头部开始匹配（FIFO）
         ├─ 完全匹配：开仓数量 = 平仓数量
         │  └─ 创建 TradePair，从队列移除该开仓记录
         │
         ├─ 部分匹配：开仓数量 < 平仓数量
         │  └─ 创建 TradePair（使用开仓的全部数量），
         │     更新平仓剩余数量，继续匹配下一个开仓
         │
         └─ 部分匹配：开仓数量 > 平仓数量
            └─ 创建 TradePair（使用平仓的全部数量），
               更新开仓剩余数量，保留在队列中

4. 返回 completedPairs
```

### 4.2 数据结构

```go
// OpenPosition 未平仓持仓（用于配对）
type OpenPosition struct {
    Record      *TradeRecord  // 原始开仓记录
    RemainingQty float64     // 剩余未匹配数量
}

// TradePair 配对结果
type TradePair struct {
    OpenRecord   *TradeRecord  // 开仓记录
    CloseRecord  *TradeRecord  // 平仓记录
    MatchedQty   float64       // 匹配的数量
    HoldingTime  int64         // 持仓时间（秒）
    PnL          float64      // 盈亏（来自 CloseRecord.PnL，按比例分配）
    OpenFee      float64       // 开仓费用（按比例）
    CloseFee     float64       // 平仓费用（按比例）
}
```

### 4.3 伪代码实现

```go
// getAbsoluteQuantity 获取绝对数量（用于配对）
func getAbsoluteQuantity(record *TradeRecord) float64 {
    if record.SignedQuantity != nil {
        return math.Abs(*record.SignedQuantity)  // 使用绝对值
    }
    return record.Quantity  // 回退到 Quantity
}

func MatchTradePairs(records []*TradeRecord) []*TradePair {
    // 1. 按时间排序
    sort.Slice(records, func(i, j int) bool {
        return records[i].Timestamp.Before(records[j].Timestamp)
    })
    
    // 2. 初始化
    openPositions := make(map[string][]*OpenPosition)  // key: "symbol_side"
    var pairs []*TradePair
    
    // 3. 遍历记录
    for _, record := range records {
        // 使用 Symbol + Side 作为 key（不依赖 signed_quantity 符号）
        key := record.Symbol + "_" + record.Side
        
        if isOpenAction(record.Action) {
            // 开仓：添加到队列
            qty := getAbsoluteQuantity(record)  // 使用绝对值
            openPositions[key] = append(openPositions[key], &OpenPosition{
                Record:       record,
                RemainingQty: qty,
            })
            
        } else if isCloseAction(record.Action) {
            // 平仓：FIFO 匹配
            closeQty := getAbsoluteQuantity(record)  // 使用绝对值
            remainingCloseQty := closeQty
            
            // 从队列头部开始匹配
            for i := 0; i < len(openPositions[key]) && remainingCloseQty > 0; {
                openPos := openPositions[key][i]
                
                if openPos.RemainingQty <= remainingCloseQty {
                    // 完全匹配或部分匹配（开仓全部用完）
                    matchedQty := openPos.RemainingQty
                    
                    pair := &TradePair{
                        OpenRecord:  openPos.Record,
                        CloseRecord: record,
                        MatchedQty:   matchedQty,
                        HoldingTime:  record.Timestamp.Sub(openPos.Record.Timestamp).Seconds(),
                        PnL:         calculateProportionalPnL(record, matchedQty, closeQty),
                        OpenFee:      calculateProportionalFee(openPos.Record, matchedQty),
                        CloseFee:     calculateProportionalFee(record, matchedQty),
                    }
                    pairs = append(pairs, pair)
                    
                    remainingCloseQty -= matchedQty
                    // 移除已完全匹配的开仓
                    openPositions[key] = append(openPositions[key][:i], openPositions[key][i+1:]...)
                    
                } else {
                    // 部分匹配（平仓全部用完，开仓还有剩余）
                    matchedQty := remainingCloseQty
                    
                    pair := &TradePair{
                        OpenRecord:  openPos.Record,
                        CloseRecord: record,
                        MatchedQty:   matchedQty,
                        HoldingTime:  record.Timestamp.Sub(openPos.Record.Timestamp).Seconds(),
                        PnL:         calculateProportionalPnL(record, matchedQty, closeQty),
                        OpenFee:      calculateProportionalFee(openPos.Record, matchedQty),
                        CloseFee:     calculateProportionalFee(record, matchedQty),
                    }
                    pairs = append(pairs, pair)
                    
                    // 更新开仓剩余数量
                    openPos.RemainingQty -= matchedQty
                    remainingCloseQty = 0
                    i++  // 继续下一个开仓（但当前平仓已用完）
                }
            }
        }
    }
    
    return pairs
}
```

## 5. 特殊情况处理

### 5.1 部分平仓

**场景**：一个开仓记录对应多个平仓记录

**处理**：
- 开仓记录保留在队列中，`RemainingQty` 递减
- 每次平仓只匹配部分数量
- 最后一个平仓完全匹配后，从队列移除

**示例**：
```
开仓：BTCUSDT long 0.01 @ 50000
平仓1：BTCUSDT close_long 0.003 @ 51000  → 配对 0.003，开仓剩余 0.007
平仓2：BTCUSDT close_long 0.005 @ 52000  → 配对 0.005，开仓剩余 0.002
平仓3：BTCUSDT close_long 0.002 @ 53000  → 配对 0.002，开仓完全匹配
```

### 5.2 多次开仓一次平仓

**场景**：多个开仓记录对应一个平仓记录

**处理**：
- 按 FIFO 顺序匹配多个开仓
- 每个开仓生成一个独立的 TradePair
- 平仓的 PnL 按比例分配给各个配对

**示例**：
```
开仓1：BTCUSDT long 0.003 @ 50000
开仓2：BTCUSDT long 0.005 @ 51000
平仓：BTCUSDT close_long 0.008 @ 52000
→ 生成两个 TradePair：
  - Pair1: 开仓1(0.003) + 平仓(0.003)
  - Pair2: 开仓2(0.005) + 平仓(0.005)
```

### 5.3 未平仓记录

**场景**：有开仓记录但没有对应的平仓记录

**处理**：
- 保留在 `openPositions` 队列中
- 不生成 TradePair
- 在统计中计入"未平仓交易数"

### 5.4 PnL 为 NULL

**场景**：平仓记录的 `PnL` 字段为 `NULL`

**处理**：
- 如果 `PnL` 为 `NULL`，尝试计算：
  - `PnL = (平仓价格 - 开仓价格) × 数量 × 方向系数`
  - 方向系数：long = +1, short = -1
- 如果无法计算，`TradePair.PnL` 设为 `0` 或 `NULL`
- 记录警告日志

### 5.5 数量不一致

**场景**：开仓和平仓的数量字段不一致或缺失

**处理**：
1. **优先使用 `SignedQuantity`**：如果两个记录都有，使用 `ABS(SignedQuantity)`
2. **回退到 `Quantity`**：如果 `SignedQuantity` 为 `NULL`，使用 `Quantity`
3. **当前数据状态**：所有记录都有 `signed_quantity`，且都是正数，因此 `ABS(SignedQuantity)` 和 `SignedQuantity` 结果相同
4. **记录警告**：如果无法确定数量，记录警告并跳过配对

### 5.6 时间顺序异常

**场景**：平仓时间早于开仓时间（数据异常）

**处理**：
- **跳过异常配对**：不生成 TradePair
- **记录警告日志**：记录异常情况
- **继续处理**：不影响其他正常配对

## 6. PnL 分配策略

### 6.1 完全匹配

如果开仓数量 = 平仓数量，直接使用平仓记录的 `PnL`：

```go
pair.PnL = closeRecord.PnL  // 如果 PnL 不为 NULL
```

### 6.2 部分匹配

如果开仓数量 ≠ 平仓数量，按比例分配：

```go
// 方法1：按数量比例分配（推荐）
if closeRecord.PnL != nil {
    ratio := matchedQty / totalCloseQty
    pair.PnL = *closeRecord.PnL * ratio
}

// 方法2：按价格计算（如果 PnL 为 NULL）
if closeRecord.PnL == nil {
    priceDiff := closeRecord.ExecutionPrice - openRecord.ExecutionPrice
    direction := 1.0  // long = +1, short = -1
    pair.PnL = priceDiff * matchedQty * direction
}
```

### 6.3 费用分配

费用也按比例分配：

```go
// 开仓费用
openFeeRatio := matchedQty / openRecord.Quantity
pair.OpenFee = openRecord.Fee * openFeeRatio

// 平仓费用
closeFeeRatio := matchedQty / closeRecord.Quantity
pair.CloseFee = closeRecord.Fee * closeFeeRatio
```

## 7. 配对验证

### 7.1 数据完整性检查

配对完成后，验证以下内容：

1. **数量一致性**：
   - 所有配对的开仓数量总和 ≤ 总开仓数量
   - 所有配对的平仓数量总和 ≤ 总平仓数量

2. **PnL 一致性**：
   - 所有配对的 PnL 总和 ≈ 所有平仓记录的 PnL 总和
   - 允许小的浮点误差（< 0.01）

3. **时间顺序**：
   - 所有配对的 `HoldingTime` ≥ 0

### 7.2 统计验证

- **配对成功率** = 成功配对数量 / 平仓记录数量 × 100%
- **未配对开仓数** = 总开仓数 - 已配对开仓数
- **未配对平仓数** = 总平仓数 - 已配对平仓数

## 8. 性能优化

### 8.1 查询优化

- **索引**：确保 `(trader_id, symbol, timestamp)` 有索引
- **过滤**：在数据库层面过滤，减少数据传输
- **排序**：使用数据库 `ORDER BY`，避免内存排序

### 8.2 内存优化

- **流式处理**：对于大数据量，考虑分批处理
- **复用结构**：避免重复创建临时对象

### 8.3 缓存策略

- **配对结果缓存**：对于相同查询条件，缓存配对结果
- **缓存失效**：当有新交易记录时，清除相关缓存

## 9. 实现建议

### 9.1 模块结构

```
trade_analytics/
├── pair_matcher.go      # 配对算法实现
├── pair_validator.go    # 配对验证逻辑
└── pair_calculator.go   # PnL 和费用计算
```

### 9.2 函数签名

```go
// MatchTradePairs 匹配交易对
func MatchTradePairs(
    ctx context.Context,
    repo Repository,
    filter *AnalyticsFilter,
) ([]*TradePair, error)

// ValidatePairs 验证配对结果
func ValidatePairs(
    pairs []*TradePair,
    openRecords []*TradeRecord,
    closeRecords []*TradeRecord,
) (*PairValidationResult, error)
```

### 9.3 错误处理

- **数据异常**：记录警告，继续处理
- **计算错误**：返回错误，不生成部分结果
- **性能问题**：超时保护，限制处理数量

## 10. 测试用例

### 10.1 基础配对

```
输入：
  - 开仓：BTCUSDT long 0.01 @ 50000 (2024-01-01 10:00:00)
  - 平仓：BTCUSDT close_long 0.01 @ 51000 (2024-01-01 11:00:00)

预期输出：
  - 1 个 TradePair
  - HoldingTime = 3600 秒
  - PnL = 平仓记录的 PnL
```

### 10.2 部分平仓

```
输入：
  - 开仓：BTCUSDT long 0.01 @ 50000
  - 平仓1：BTCUSDT close_long 0.003 @ 51000
  - 平仓2：BTCUSDT close_long 0.007 @ 52000

预期输出：
  - 2 个 TradePair
  - 第一个配对数量 = 0.003
  - 第二个配对数量 = 0.007
```

### 10.3 多次开仓一次平仓

```
输入：
  - 开仓1：BTCUSDT long 0.003 @ 50000
  - 开仓2：BTCUSDT long 0.005 @ 51000
  - 平仓：BTCUSDT close_long 0.008 @ 52000

预期输出：
  - 2 个 TradePair
  - 按 FIFO 顺序匹配
```

### 10.4 未平仓

```
输入：
  - 开仓：BTCUSDT long 0.01 @ 50000
  - （无平仓记录）

预期输出：
  - 0 个 TradePair
  - 1 个未平仓记录
```

## 11. 当前数据状态（2025-11-13）

### 11.1 数据质量

- ✅ **总记录数**：200 条
- ✅ **`signed_quantity` 完整性**：100% 记录有值，无 NULL
- ✅ **符号一致性**：100% 记录为正数（Long 和 Short 方向都是正数）
- ✅ **API 版本**：100% 使用新版本 API（所有记录都有 `start_position` 字段）
- ✅ **数据格式统一**：无混合格式，完全符合新 API 规则

### 11.2 配对策略优势

由于数据格式统一：
- ✅ 可以安全地使用 `ABS(SignedQuantity)` 或直接使用 `SignedQuantity`（当前都是正数）
- ✅ 配对逻辑更简单，无需处理符号不一致的情况
- ✅ 数据质量高，配对成功率预期较高

### 11.3 兼容性考虑

虽然当前数据格式统一，但配对算法仍应：
- ✅ 使用 `Side` 字段判断方向（不依赖符号）
- ✅ 使用绝对值进行数量匹配（保持健壮性）
- ✅ 处理 `signed_quantity` 为 NULL 的情况（虽然当前没有）

## 12. 注意事项

1. **数据质量**：确保 `trade_history` 数据完整准确
2. **时区处理**：统一使用 UTC 时间戳
3. **浮点精度**：注意浮点数比较和计算精度问题
4. **并发安全**：如果支持并发查询，确保线程安全
5. **日志记录**：记录配对过程中的异常和警告
6. **API 版本兼容**：虽然当前使用新 API，但算法应兼容可能的格式变化

---

**文档版本**: 1.1  
**创建日期**: 2025-11-13  
**最后更新**: 2025-11-13  
**更新说明**: 根据重新同步后的数据（200 条记录，100% 新 API 格式）更新了数量字段说明和配对逻辑


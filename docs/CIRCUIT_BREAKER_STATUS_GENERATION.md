# 熔断机制状态信息生成流程分析

**分析日期**: 2025-01-18  
**基于**: testdockerv2 决策 cycle 中的熔断机制状态信息

---

## 📋 问题描述

在 testdockerv2 决策 cycle 中，看到以下输入信息：

```
## 🛡️ 熔断机制状态

连续亏损: 2次 ⚠️ **已触发熔断机制**（应暂停交易约30分钟）

最长连亏: 2次
```

需要分析这些信息是如何生成的。

---

## 🔍 数据流追踪

### 1. 数据来源：交易历史记录（trade_history 表）

**位置**: `trade_history` 数据库表

**关键字段**:
- `trader_id`: 交易员ID
- `pnl`: 盈亏金额（NULL 表示未平仓，有值表示已平仓）
- `timestamp`: 交易时间

**数据获取**:
```363:377:trade_analytics/analyzer.go
// CalculateStreakStats 计算连续统计
func (a *analyzer) CalculateStreakStats(ctx context.Context, filter *AnalyticsFilter) (*StreakStatistics, error) {
	// 获取所有有PnL的记录（按时间排序）
	records, err := a.repo.GetRecordsByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录失败: %w", err)
	}

	// 过滤出有PnL的记录并按时间排序
	var pnlRecords []*TradeRecord
	for _, rec := range records {
		if rec.PnL != nil {
			pnlRecords = append(pnlRecords, rec)
		}
	}
```

**说明**: 
- 只统计有 `PnL` 值的记录（已平仓的交易）
- 通过 `AnalyticsFilter` 过滤特定交易员的数据
- 从 `trade_history` 表查询，最多获取 10000 条记录

---

### 2. 计算逻辑：连续统计计算

**位置**: `trade_analytics/analyzer.go::CalculateStreakStats()`

**算法流程**:

```388:444:trade_analytics/analyzer.go
	// 计算连续统计
	var longestWinningStreak int
	var longestLosingStreak int
	var currentStreak int
	var currentStreakType string

	var currentWinningStreak int
	var currentLosingStreak int

	for _, rec := range pnlRecords {
		if rec.PnL == nil {
			continue
		}

		if *rec.PnL > 0 {
			// 盈利
			currentWinningStreak++
			currentLosingStreak = 0

			if currentWinningStreak > longestWinningStreak {
				longestWinningStreak = currentWinningStreak
			}
		} else if *rec.PnL < 0 {
			// 亏损
			currentLosingStreak++
			currentWinningStreak = 0

			if currentLosingStreak > longestLosingStreak {
				longestLosingStreak = currentLosingStreak
			}
		} else {
			// 持平，重置连续
			currentWinningStreak = 0
			currentLosingStreak = 0
		}
	}

	// 计算当前连续（从最后一条记录开始）
	if len(pnlRecords) > 0 {
		lastRec := pnlRecords[len(pnlRecords)-1]
		if lastRec.PnL != nil {
			if *lastRec.PnL > 0 {
				currentStreak = currentWinningStreak
				currentStreakType = "winning"
			} else if *lastRec.PnL < 0 {
				currentStreak = -currentLosingStreak
				currentStreakType = "losing"
			}
		}
	}

	return &StreakStatistics{
		LongestWinningStreak: longestWinningStreak,
		LongestLosingStreak:   longestLosingStreak,
		CurrentStreak:         currentStreak,
		CurrentStreakType:     currentStreakType,
	}, nil
```

**关键逻辑**:
1. **遍历所有交易记录**（按时间升序）:
   - `PnL > 0`: 盈利 → `currentWinningStreak++`, `currentLosingStreak = 0`
   - `PnL < 0`: 亏损 → `currentLosingStreak++`, `currentWinningStreak = 0`
   - `PnL = 0`: 持平 → 重置两个计数器

2. **记录最长连胜/连亏**:
   - 每次更新计数器时，同时更新 `longestWinningStreak` 和 `longestLosingStreak`

3. **确定当前连续状态**:
   - 查看**最后一条记录**的 PnL
   - 如果最后一条是亏损 → `CurrentStreak = -currentLosingStreak`（负数），`CurrentStreakType = "losing"`
   - 如果最后一条是盈利 → `CurrentStreak = currentWinningStreak`（正数），`CurrentStreakType = "winning"`

**数据结构**:
```go
type StreakStatistics struct {
    LongestWinningStreak int    `json:"longest_winning_streak"` // 最长连胜
    LongestLosingStreak  int    `json:"longest_losing_streak"`  // 最长连亏
    CurrentStreak        int    `json:"current_streak"`         // 当前连胜/连亏（正数=连胜，负数=连亏）
    CurrentStreakType    string `json:"current_streak_type"`    // "winning" 或 "losing"
}
```

---

### 3. 获取时机：构建交易上下文时

**位置**: `trader/auto_trader.go::buildTradingContext()`

**调用时机**: 每个决策 cycle 开始时

```801:818:trader/auto_trader.go
	// 5.1 获取连续统计（新增）
	var streakStats interface{}
	if at.tradeAnalyticsService != nil {
		log.Printf("🔍 [DEBUG] tradeAnalyticsService 不为 nil，开始获取连续统计 (trader_id: %s)", at.id)
		filter := &trade_analytics.AnalyticsFilter{
			TraderID: at.id,
			// 不设置时间范围，使用所有历史数据
		}
		stats, err := at.tradeAnalyticsService.GetStreakStats(context.Background(), filter)
		if err == nil {
			streakStats = stats
			log.Printf("🔍 [DEBUG] 成功获取连续统计: %+v", stats)
		} else {
			log.Printf("⚠️  获取连续统计失败: %v", err)
		}
	} else {
		log.Printf("🔍 [DEBUG] tradeAnalyticsService 为 nil，跳过获取连续统计 (trader_id: %s)", at.id)
	}
```

**说明**:
- 通过 `tradeAnalyticsService.GetStreakStats()` 获取统计数据
- 使用 `AnalyticsFilter` 过滤特定交易员的数据
- 不设置时间范围，使用**所有历史数据**
- 如果服务不可用或获取失败，`streakStats` 为 `nil`（优雅降级）

**传递到 Context**:
```go
ctx := &decision.Context{
    // ... 其他字段 ...
    StreakStats: streakStats,  // interface{} 类型
}
```

---

### 4. 格式化展示：构建 User Prompt 时

**位置**: `decision/engine.go::buildUserPrompt()`

**展示逻辑**:

```618:652:decision/engine.go
	// 连续亏损信息（新增）
	log.Printf("🔍 [DEBUG] ctx.StreakStats: %+v", ctx.StreakStats)
	if ctx.StreakStats != nil {
		type StreakData struct {
			CurrentStreak      int    `json:"current_streak"`
			CurrentStreakType string `json:"current_streak_type"`
			LongestLosingStreak int  `json:"longest_losing_streak"`
		}
		var streakData StreakData
		if jsonData, err := json.Marshal(ctx.StreakStats); err == nil {
			if err := json.Unmarshal(jsonData, &streakData); err == nil {
				log.Printf("🔍 [DEBUG] streakData: %+v", streakData)
				if streakData.CurrentStreakType == "losing" {
					consecutiveLosses := -streakData.CurrentStreak // CurrentStreak 为负数表示连亏
					sb.WriteString("## 🛡️ 熔断机制状态\n\n")
					sb.WriteString(fmt.Sprintf("连续亏损: %d次", consecutiveLosses))
					if consecutiveLosses >= 2 {
						sb.WriteString(" ⚠️ **已触发熔断机制**（应暂停交易约30分钟）\n")
					} else {
						sb.WriteString("\n")
					}
					sb.WriteString(fmt.Sprintf("最长连亏: %d次\n\n", streakData.LongestLosingStreak))
					log.Printf("🔍 [DEBUG] 已添加熔断机制状态到prompt (连续亏损: %d次)", consecutiveLosses)
				} else {
					log.Printf("🔍 [DEBUG] CurrentStreakType 不是 'losing'，当前值: %s", streakData.CurrentStreakType)
				}
			} else {
				log.Printf("🔍 [DEBUG] JSON反序列化失败: %v", err)
			}
		} else {
			log.Printf("🔍 [DEBUG] JSON序列化失败: %v", err)
		}
	} else {
		log.Printf("🔍 [DEBUG] ctx.StreakStats 为 nil，跳过添加熔断机制状态")
	}
```

**关键逻辑**:
1. **检查数据是否存在**: `ctx.StreakStats != nil`
2. **JSON 序列化/反序列化**: 因为 `StreakStats` 是 `interface{}` 类型，需要通过 JSON 转换
3. **判断是否连续亏损**: `CurrentStreakType == "losing"`
4. **计算连续亏损次数**: `consecutiveLosses = -CurrentStreak`（因为 `CurrentStreak` 为负数）
5. **判断是否触发熔断**: `consecutiveLosses >= 2` → 显示警告信息
6. **显示最长连亏**: 从 `LongestLosingStreak` 字段获取

**展示条件**:
- ✅ `ctx.StreakStats != nil`
- ✅ `CurrentStreakType == "losing"`（只有连续亏损时才显示）
- ❌ 如果当前是连胜或没有交易记录，不会显示任何内容

---

## 📊 示例：testdockerv2 的情况

### 假设的交易历史

假设 testdockerv2 的交易历史如下（按时间顺序）：

| 交易序号 | PnL | 说明 |
|---------|-----|------|
| 1 | +100 | 盈利 |
| 2 | -50 | 亏损 |
| 3 | -30 | 亏损（连续第2次亏损）|

### 计算过程

1. **遍历交易记录**:
   - 交易1: `PnL = +100` → `currentWinningStreak = 1`, `currentLosingStreak = 0`
   - 交易2: `PnL = -50` → `currentWinningStreak = 0`, `currentLosingStreak = 1`
   - 交易3: `PnL = -30` → `currentWinningStreak = 0`, `currentLosingStreak = 2`

2. **更新最长连亏**:
   - `longestLosingStreak = 2`（因为 `currentLosingStreak = 2`）

3. **确定当前状态**:
   - 最后一条记录（交易3）是亏损 → `CurrentStreak = -2`, `CurrentStreakType = "losing"`

4. **格式化展示**:
   - `consecutiveLosses = -(-2) = 2`
   - `consecutiveLosses >= 2` → 触发熔断机制警告
   - `LongestLosingStreak = 2`

### 最终输出

```
## 🛡️ 熔断机制状态

连续亏损: 2次 ⚠️ **已触发熔断机制**（应暂停交易约30分钟）

最长连亏: 2次
```

---

## 🔄 完整数据流图

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 数据源: trade_history 表                                 │
│    - 查询条件: trader_id = 'testdockerv2'                   │
│    - 过滤: pnl IS NOT NULL (只统计已平仓交易)                │
│    - 排序: timestamp ASC (按时间升序)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. 计算: CalculateStreakStats()                             │
│    - 遍历所有交易记录                                        │
│    - 计算 currentWinningStreak / currentLosingStreak         │
│    - 更新 longestWinningStreak / longestLosingStreak         │
│    - 根据最后一条记录确定 CurrentStreak 和 CurrentStreakType │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. 获取: buildTradingContext()                              │
│    - 调用 tradeAnalyticsService.GetStreakStats()            │
│    - 将结果存储到 ctx.StreakStats (interface{})             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. 展示: buildUserPrompt()                                  │
│    - 检查 ctx.StreakStats != nil                            │
│    - JSON 序列化/反序列化                                    │
│    - 判断 CurrentStreakType == "losing"                     │
│    - 计算 consecutiveLosses = -CurrentStreak                  │
│    - 判断 consecutiveLosses >= 2 → 显示熔断警告               │
│    - 显示最长连亏                                            │
└─────────────────────────────────────────────────────────────┘
```

---

## ⚠️ 注意事项

### 1. 数据范围
- **使用所有历史数据**: 当前实现不设置时间范围，会统计该交易员的**所有历史交易**
- **只统计已平仓交易**: 只有 `pnl IS NOT NULL` 的记录才会被统计

### 2. 计算逻辑
- **按时间顺序**: 交易记录按 `timestamp ASC` 排序，确保计算顺序正确
- **最后一条决定当前状态**: `CurrentStreak` 和 `CurrentStreakType` 由最后一条交易的 PnL 决定
- **负数表示连亏**: `CurrentStreak` 为负数时表示连续亏损，需要取绝对值

### 3. 展示条件
- **只有连续亏损时才显示**: 如果 `CurrentStreakType != "losing"`，不会显示熔断机制状态
- **熔断阈值**: `consecutiveLosses >= 2` 时显示警告信息

### 4. 错误处理
- **服务不可用**: 如果 `tradeAnalyticsService == nil`，`streakStats` 为 `nil`，不会显示任何信息（优雅降级）
- **获取失败**: 如果 `GetStreakStats()` 失败，记录日志但不中断流程

---

## 🔍 调试方法

### 1. 检查数据源
```sql
-- 检查交易历史数据
SELECT trader_id, COUNT(*) as trade_count 
FROM trade_history 
WHERE trader_id = 'testdockerv2' AND pnl IS NOT NULL
GROUP BY trader_id;

-- 查看最近的交易记录（按时间排序）
SELECT timestamp, pnl, symbol, action, side
FROM trade_history 
WHERE trader_id = 'testdockerv2' AND pnl IS NOT NULL
ORDER BY timestamp ASC
LIMIT 10;
```

### 2. 查看日志
```bash
# 查找获取连续统计的日志
grep "获取连续统计\|GetStreakStats\|streakData" <应用日志文件>

# 查找熔断机制状态的日志
grep "熔断机制状态\|已触发熔断机制" <应用日志文件>
```

### 3. 验证计算逻辑
- 手动计算最近几条交易的连续亏损次数
- 对比日志中的 `streakData` 输出
- 确认 `CurrentStreak` 和 `CurrentStreakType` 是否正确

---

## 📝 总结

熔断机制状态信息的生成流程：

1. **数据源**: 从 `trade_history` 表获取有 PnL 的交易记录
2. **计算**: `CalculateStreakStats()` 遍历所有记录，计算连续统计
3. **获取**: 每个决策 cycle 开始时，通过 `GetStreakStats()` 获取统计数据
4. **展示**: 在构建 User Prompt 时，如果当前是连续亏损状态，格式化展示熔断机制状态

**关键点**:
- 只统计已平仓交易（`pnl IS NOT NULL`）
- 使用所有历史数据（不设置时间范围）
- 只有连续亏损时才显示（`CurrentStreakType == "losing"`）
- 连续亏损 ≥ 2 次时触发熔断警告

---

**文档版本**: v1.0  
**创建日期**: 2025-01-18  
**作者**: AI Assistant


# Trade Analytics Prompt 集成检查报告

**检查日期**: 2025-01-16  
**检查目标**: 验证最近的AI决策是否正确从trade analytics获取了数据并作为prompt提交给AI

---

## 1. 检查结果摘要

### 1.1 检查的决策记录

- **最新决策文件**: `decision_20251118_092937_cycle6.json`
- **决策时间**: 2025-11-18 09:29:37
- **Cycle Number**: 6

### 1.2 检查发现

❌ **问题**: 在`input_prompt`中**未找到**trade analytics数据

**具体表现**：
- 未找到"🛡️ 熔断机制状态"标题
- 未找到"连续亏损"相关信息
- 未找到`StreakStats`相关数据

**input_prompt中的章节**（已确认存在）：
- 📚 历史决策参考
- ⏰ 系统状态
- 📊 交易状态追踪
- ₿ BTC市场概览
- 💰 账户信息
- 📈 当前持仓
- 🔍 候选币种
- 📊 绩效指标
- ✅ 输出自检

**缺失的章节**：
- 🛡️ 熔断机制状态（应该在这里显示连续亏损信息）

---

## 2. 代码流程分析

### 2.1 数据流路径

```
1. API Server 启动
   └─> setupTradeAnalyticsRoutes()
       └─> 初始化 trade_analytics.Service
       └─> 设置到 TraderManager.SetTradeAnalyticsService()

2. TraderManager.LoadSingleTrader()
   └─> trader.NewAutoTrader(..., tm.tradeAnalyticsService)
       └─> AutoTrader.tradeAnalyticsService = tradeAnalyticsService

3. AutoTrader.buildTradingContext()
   └─> 检查: if at.tradeAnalyticsService != nil
       └─> at.tradeAnalyticsService.GetStreakStats()
           └─> ctx.StreakStats = stats

4. decision.buildUserPrompt(ctx)
   └─> 检查: if ctx.StreakStats != nil
       └─> 解析 StreakStats
       └─> 检查: if CurrentStreakType == "losing"
           └─> 添加"🛡️ 熔断机制状态"章节
```

### 2.2 可能的问题点

#### 问题1: tradeAnalyticsService 未初始化

**检查点**: `api/server.go::setupTradeAnalyticsRoutes()`

**可能原因**：
1. `trade_history`服务未启用（`tradeHistoryService == nil`）
2. 数据库连接失败
3. `setupTradeAnalyticsRoutes()`未被调用

**代码位置**:
```148:166:api/server.go
// setupTradeAnalyticsRoutes 设置交易分析路由
func (s *Server) setupTradeAnalyticsRoutes(router *gin.RouterGroup) {
	// 初始化 trade_analytics 服务
	db, err := s.database.GetDB()
	if err != nil {
		log.Printf("⚠️ 无法初始化交易分析服务: %v", err)
		// ...
		return
	}

	// 获取 trade_history repository
	tradeHistoryService := s.traderManager.GetTradeHistoryService()
	if tradeHistoryService == nil {
		log.Printf("⚠️ 交易历史服务未启用，交易分析功能不可用")
		// ...
		return
	}
```

#### 问题2: GetStreakStats 调用失败

**检查点**: `trader/auto_trader.go::buildTradingContext()`

**可能原因**：
1. `tradeAnalyticsService`为`nil`
2. `GetStreakStats()`返回错误
3. 错误被静默处理（只记录日志，不中断流程）

**代码位置**:
```795:808:trader/auto_trader.go
	// 5.1 获取连续统计（新增）
	var streakStats interface{}
	if at.tradeAnalyticsService != nil {
		filter := &trade_analytics.AnalyticsFilter{
			TraderID: at.id,
			// 不设置时间范围，使用所有历史数据
		}
		stats, err := at.tradeAnalyticsService.GetStreakStats(context.Background(), filter)
		if err == nil {
			streakStats = stats
		} else {
			log.Printf("⚠️  获取连续统计失败: %v", err)
		}
	}
```

#### 问题3: StreakStats 数据为空或非"losing"

**检查点**: `decision/engine.go::buildUserPrompt()`

**可能原因**：
1. `StreakStats`为`nil`
2. `CurrentStreakType != "losing"`（当前是"winning"或没有数据）
3. JSON序列化/反序列化失败

**代码位置**:
```618:641:decision/engine.go
	// 连续亏损信息（新增）
	if ctx.StreakStats != nil {
		type StreakData struct {
			CurrentStreak      int    `json:"current_streak"`
			CurrentStreakType string `json:"current_streak_type"`
			LongestLosingStreak int  `json:"longest_losing_streak"`
		}
		var streakData StreakData
		if jsonData, err := json.Marshal(ctx.StreakStats); err == nil {
			if err := json.Unmarshal(jsonData, &streakData); err == nil {
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
				}
			}
		}
	}
```

**注意**: 只有当`CurrentStreakType == "losing"`时才会显示。如果当前是连胜或没有交易记录，不会显示任何内容。

---

## 3. 诊断步骤

### 3.1 检查服务初始化

**方法1**: 查看应用启动日志
```bash
# 查找初始化日志
grep -i "交易分析\|trade.*analytics\|GetStreakStats" <应用日志文件>
```

**期望看到的日志**：
- ✅ `交易分析 API 已注册，服务已设置到 TraderManager`
- ❌ `交易历史服务未启用，交易分析功能不可用`
- ❌ `无法初始化交易分析服务: ...`

**方法2**: 检查数据库配置
```sql
-- 检查 trade_history_enabled 配置
SELECT * FROM system_config WHERE key = 'trade_history_enabled';
```

### 3.2 检查运行时调用

**方法1**: 查看运行时日志
```bash
# 查找运行时错误
grep -i "获取连续统计失败" <应用日志文件>
```

**方法2**: 添加调试日志

在`trader/auto_trader.go::buildTradingContext()`中添加：
```go
// 5.1 获取连续统计（新增）
var streakStats interface{}
if at.tradeAnalyticsService != nil {
    log.Printf("🔍 [DEBUG] tradeAnalyticsService 不为 nil，开始获取连续统计")
    filter := &trade_analytics.AnalyticsFilter{
        TraderID: at.id,
    }
    stats, err := at.tradeAnalyticsService.GetStreakStats(context.Background(), filter)
    if err == nil {
        streakStats = stats
        log.Printf("🔍 [DEBUG] 成功获取连续统计: %+v", stats)
    } else {
        log.Printf("⚠️  获取连续统计失败: %v", err)
    }
} else {
    log.Printf("🔍 [DEBUG] tradeAnalyticsService 为 nil，跳过获取连续统计")
}
```

在`decision/engine.go::buildUserPrompt()`中添加：
```go
// 连续亏损信息（新增）
log.Printf("🔍 [DEBUG] ctx.StreakStats: %+v", ctx.StreakStats)
if ctx.StreakStats != nil {
    // ... 现有代码 ...
    log.Printf("🔍 [DEBUG] streakData: %+v", streakData)
}
```

### 3.3 检查数据内容

**方法**: 直接查询数据库
```sql
-- 检查是否有交易记录
SELECT COUNT(*) FROM trade_history WHERE trader_id = '<trader_id>';

-- 检查是否有配对数据
-- 需要调用 trade_analytics API 或直接测试
```

---

## 4. 可能的原因总结

### 4.1 最可能的原因

1. **trade_history服务未启用**
   - `system_config.trade_history_enabled != 'true'`
   - 导致`tradeAnalyticsService`无法初始化

2. **当前没有连续亏损**
   - `CurrentStreakType != "losing"`
   - 代码逻辑正确，但当前状态不需要显示

3. **没有交易历史数据**
   - `trade_history`表中没有该交易员的记录
   - `GetStreakStats()`返回空数据或默认值

### 4.2 需要验证的点

- [ ] 检查`trade_history_enabled`配置
- [ ] 检查应用启动日志
- [ ] 检查运行时错误日志
- [ ] 检查数据库中是否有交易记录
- [ ] 验证`tradeAnalyticsService`是否被正确传递到`AutoTrader`

---

## 5. 建议的修复步骤

### 步骤1: 确认服务初始化

检查应用启动时的日志，确认：
- `setupTradeAnalyticsRoutes()`是否被调用
- `trade_history`服务是否已初始化
- `trade_analytics`服务是否已初始化

### 步骤2: 添加调试日志

在关键位置添加调试日志，确认数据流：
- `buildTradingContext()`中`tradeAnalyticsService`是否为`nil`
- `GetStreakStats()`是否被调用
- 返回值是什么

### 步骤3: 验证数据

- 检查数据库中是否有交易记录
- 直接调用`GetStreakStats()` API，查看返回结果
- 确认`CurrentStreakType`的值

### 步骤4: 测试场景

创建测试场景：
- 确保有交易历史数据
- 确保有连续亏损记录
- 验证prompt中是否显示

---

## 6. 结论

**当前状态**: ❌ Trade analytics数据**未正确**集成到prompt中

**需要进一步调查**:
1. 服务初始化状态
2. 运行时调用情况
3. 数据内容

**下一步行动**:
1. 检查应用启动日志
2. 添加调试日志
3. 验证数据库数据
4. 测试完整流程

---

**检查人**: AI Assistant  
**检查日期**: 2025-01-16


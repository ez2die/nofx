# Trader 04 频繁决策问题诊断报告

## 问题描述

Trader 04连续跑了好几次决策，都记录为cycle 1，间隔仅几秒钟，然后trader被停掉了。

## 问题分析

### 时间线

从日志可以看到：
- **13:51:13** - 第一次自动重启
- **13:51:13-13:54:26** - 频繁的重新加载和自动重启
- **13:54:26** - 最后一次决策（cycle 1）
- **13:54:36** - Trader被停掉

### 决策日志统计

从决策日志文件可以看到：
- `decision_20251105_135442_cycle1.json` - 13:54:42
- `decision_20251105_135438_cycle2.json` - 13:54:38
- `decision_20251105_135435_cycle2.json` - 13:54:35
- `decision_20251105_135431_cycle1.json` - 13:54:31
- `decision_20251105_135427_cycle1.json` - 13:54:27
- `decision_20251105_135422_cycle3.json` - 13:54:22
- `decision_20251105_135419_cycle1.json` - 13:54:19
- `decision_20251105_135416_cycle1.json` - 13:54:16
- `decision_20251105_135410_cycle1.json` - 13:54:10

**观察**：
1. 大部分都是cycle 1，只有少数是cycle 2或cycle 3
2. 时间间隔只有几秒钟（应该按照ScanInterval，通常是3分钟）
3. 每次重新加载后，cycle number都会重置为1

### 根本原因

#### 1. 频繁的重新加载导致状态丢失

**触发LoadUserTraders的API请求**：

当用户在trader详情页面时，前端会同时轮询以下API（使用SWR自动轮询）：

| API端点 | 轮询间隔 | 说明 |
|---------|---------|------|
| `/api/status` | 15秒 | 系统状态 |
| `/api/account` | 15秒 | 账户信息 |
| `/api/positions` | 15秒 | 持仓列表 |
| `/api/decisions/latest` | 30秒 | 最新决策 |
| `/api/statistics` | 30秒 | 统计信息 |
| `/api/performance` | 30秒 | 表现分析 |
| `/api/equity-history` | 30秒 | 收益率历史 |

**这些API的共同点**：
- 都使用`getTraderFromQuery`函数获取trader
- `getTraderFromQuery`中**无条件调用**`LoadUserTraders`
- 没有检查trader是否已加载，或者配置是否变化

**代码位置**：
```go
// api/server.go:186-194
func (s *Server) getTraderFromQuery(c *gin.Context) (*manager.TraderManager, string, error) {
    userID := c.GetString("user_id")
    traderID := c.Query("trader_id")

    // 确保用户的交易员已加载到内存中
    err := s.traderManager.LoadUserTraders(s.database, userID)  // ⚠️ 每次请求都调用
    // ...
}
```

**前端轮询代码**（web/src/App.tsx:108-166）：
```typescript
// 系统状态 - 每15秒轮询
const { data: status } = useSWR(
  `status-${selectedTraderId}`,
  () => api.getStatus(selectedTraderId),
  { refreshInterval: 15000 }
);

// 账户信息 - 每15秒轮询
const { data: account } = useSWR(
  `account-${selectedTraderId}`,
  () => api.getAccount(selectedTraderId),
  { refreshInterval: 15000 }
);

// 持仓列表 - 每15秒轮询
const { data: positions } = useSWR(
  `positions-${selectedTraderId}`,
  () => api.getPositions(selectedTraderId),
  { refreshInterval: 15000 }
);
```

**问题**：
- 每15秒会有3个API请求同时触发（status、account、positions）
- 每个请求都会调用`LoadUserTraders`，导致trader被停止并重新加载
- 从日志可以看到频繁的重新加载：
```
2025/11/05 13:54:22 ⏹  已停止运行中的交易员: trader 04 (将在重新加载后自动重启)
2025/11/05 13:54:23 🔄 自动重启交易员 hyperliquid_26f0a132-0026-4516-b5ef-59a2d0406dec_deepseek_1762221829 (之前正在运行)
2025/11/05 13:54:24 ⏹  已停止运行中的交易员: trader 04 (将在重新加载后自动重启)
2025/11/05 13:54:25 🔄 自动重启交易员 hyperliquid_26f0a132-0026-4516-b5ef-59a2d0406dec_deepseek_1762221829 (之前正在运行)
```

**问题**：每次重新加载都会丢失之前的运行状态（callCount、cycleNumber等）

#### 2. 首次立即执行导致间隔缩短

在`Run()`方法中（trader/auto_trader.go:235-238）：
```go
// 首次立即执行
if err := at.runCycle(); err != nil {
    log.Printf("❌ 执行失败: %v", err)
}
```

每次重启后，都会立即执行一个cycle，而不是等待`ScanInterval`。

#### 3. 频繁的API请求

前端每15秒轮询API，导致：
- 13:54:22 - API请求 → 重新加载 → 自动重启 → 立即执行cycle 1
- 13:54:23 - API请求 → 重新加载 → 自动重启 → 立即执行cycle 1
- 13:54:24 - API请求 → 重新加载 → 自动重启 → 立即执行cycle 1
- ...

### 代码分析

#### AutoTrader状态初始化

```go
// trader/auto_trader.go:203-221
return &AutoTrader{
    // ...
    callCount: 0,  // ⚠️ 每次创建新实例都重置为0
    // ...
}
```

#### DecisionLogger状态初始化

```go
// logger/decision_logger.go:83-87
return &DecisionLogger{
    logDir:      logDir,
    cycleNumber: 0,  // ⚠️ 每次创建新实例都重置为0
}
```

#### Run方法首次立即执行

```go
// trader/auto_trader.go:235-238
// 首次立即执行
if err := at.runCycle(); err != nil {
    log.Printf("❌ 执行失败: %v", err)
}
```

## 解决方案

### 方案1：避免频繁重新加载（推荐）

**核心思路**：只在配置真正变化时才重新加载trader

1. **添加配置比较逻辑**
   - 在重新加载前，比较新配置和旧配置
   - 如果配置没有变化，跳过重新加载
   - 如果配置有变化，才停止并重新加载

2. **实现配置哈希**
   - 对trader配置计算哈希值
   - 比较哈希值判断配置是否变化

### 方案2：恢复运行状态

**核心思路**：在重新加载时，从日志中恢复cycle number

1. **从日志文件恢复cycle number**
   - 在创建新`DecisionLogger`时，读取日志目录中的最新日志
   - 从最新日志中提取cycle number
   - 设置为新的起始cycle number

2. **保存和恢复callCount**
   - 在trader的状态中保存callCount
   - 重新加载时，从状态中恢复callCount

### 方案3：优化重新加载逻辑

**核心思路**：如果trader已在运行且配置未变化，跳过重新加载

1. **检查配置是否变化**
   - 比较新旧配置
   - 如果配置未变化，且trader正在运行，跳过重新加载

2. **只加载不存在的trader**
   - 如果trader已存在且正在运行，跳过重新加载

## 推荐方案

**采用方案1 + 方案2的组合**：

1. **优先避免重新加载**：只在配置真正变化时才重新加载
2. **恢复状态**：如果必须重新加载，从日志中恢复cycle number

这样可以：
- 减少不必要的重新加载
- 保持cycle number的连续性
- 避免频繁的决策执行

## 修复步骤

1. 添加配置比较逻辑
2. 实现从日志恢复cycle number
3. 优化重新加载条件
4. 测试验证


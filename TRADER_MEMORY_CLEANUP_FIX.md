# Trader内存清理问题修复

## 问题描述

前端live页面的排行榜中显示有2个未运行的trader，但数据库查询显示所有trader都是运行状态。

### 问题原因

1. **数据库状态**: 数据库中有3个trader，`is_running`字段都是1（运行中）
2. **内存状态**: 内存中有5个trader，其中2个显示为`is_running=false`（未运行）
3. **不一致原因**: 
   - 这2个trader（trader b和c）在数据库中已被删除
   - 但删除时没有从内存中移除，导致内存中残留
   - `GetCompetitionData()`返回内存中的trader，所以前端显示这2个未运行的trader

### 未运行的trader

- `hyperliquid_d53550af-05cd-494d-8d6e-18fc940d15c9_deepseek_1762221256` (trader b)
- `hyperliquid_d53550af-05cd-494d-8d6e-18fc940d15c9_deepseek_1762221385` (c)

## 修复方案

### 1. 添加RemoveTrader方法

在`manager/trader_manager.go`中添加了`RemoveTrader`方法，用于从内存中移除trader：

```go
// RemoveTrader 从内存中移除trader
func (tm *TraderManager) RemoveTrader(id string) {
    // 停止运行中的trader并从map中删除
}
```

### 2. 修复删除接口

修改`api/server.go`中的`handleDeleteTrader`，在删除数据库记录后，同时从内存中移除：

```go
// 从内存中移除trader（会先停止运行中的trader）
s.traderManager.RemoveTrader(traderID)
```

### 3. 修复加载逻辑

修改`LoadTradersFromDatabase`，在加载前清理数据库中不存在的trader：

- 收集所有数据库中的trader ID
- 遍历内存中的trader，移除不在数据库中的trader
- 确保内存状态与数据库状态一致

## 验证

修复后，需要重启后端服务，然后：

1. 检查API返回的trader数量应该与数据库一致
2. 前端live页面不应该再显示已删除的trader
3. 删除trader时应该从内存中完全移除

## 测试步骤

```bash
# 1. 检查当前状态
./check_api_traders.sh

# 2. 重启后端服务（如果修改了代码）

# 3. 再次检查API返回
./check_api_traders.sh

# 4. 检查数据库状态
sqlite3 config.db "SELECT id, name, is_running FROM traders;"
```

## 修复文件

- `manager/trader_manager.go`: 添加RemoveTrader方法，修复LoadTradersFromDatabase
- `api/server.go`: 修复handleDeleteTrader，调用RemoveTrader


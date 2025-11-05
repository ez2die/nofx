# 当前系统实现 vs 期望业务逻辑详细对比

## 业务逻辑点1：创建Trader

### 期望实现
- 新trader被创建后，将trader配置写入数据库
- Trader状态为**停止**（`is_running = false`）
- **不加载到内存**，不启动

### 当前实现
```go
// api/server.go:350-393
trader := &config.TraderRecord{
    // ...
    IsRunning: false,  // ✅ 状态为停止
}

// 保存到数据库
err := s.database.CreateTrader(trader)

// ⚠️ 立即将新交易员加载到TraderManager中
err = s.traderManager.LoadUserTraders(s.database, userID)
```

**差异**：
- ✅ 状态为停止：符合
- ❌ **立即加载到内存**：不符合期望
- ❌ 虽然加载了，但不会启动（因为`IsRunning=false`）

**问题**：创建trader后立即加载到内存，虽然不启动，但占用了内存资源。

---

## 业务逻辑点2：启动Trader

### 期望实现
- 用户在页面点击启动后
- **此时才加载trader到内存**
- 加载参数从数据库获取
- 启动trader运行

### 当前实现
```go
// api/server.go:555-596
func (s *Server) handleStartTrader(c *gin.Context) {
    // ...
    
    // ⚠️ 直接获取trader，假设已加载
    trader, err := s.traderManager.GetTrader(traderID)
    if err != nil {
        // ❌ 如果未加载，会失败
        c.JSON(http.StatusNotFound, gin.H{"error": "交易员不存在"})
        return
    }
    
    // 启动trader
    go func() {
        trader.Run()
    }()
    
    // 更新数据库状态
    err = s.database.UpdateTraderStatus(userID, traderID, true)
}
```

**差异**：
- ❌ **没有从数据库加载trader的逻辑**
- ❌ 假设trader已加载到内存
- ❌ 如果trader未加载，启动会失败
- ✅ 更新数据库状态：符合

**问题**：如果trader未加载到内存（比如刚创建，或者系统重启后），启动会失败。

---

## 业务逻辑点3：正常运行期间

### 期望实现
- **除非用户触发停止，trader不需要重新加载**
- Trader持续运行，按`ScanInterval`执行决策周期

### 当前实现
```go
// api/server.go:186-194
func (s *Server) getTraderFromQuery(c *gin.Context) {
    // ⚠️ 每次查询都调用LoadUserTraders
    err := s.traderManager.LoadUserTraders(s.database, userID)
}
```

**差异**：
- ❌ **每次查询类API请求都会调用`LoadUserTraders`**
- ❌ 导致trader被频繁停止并重新加载
- ❌ 即使有冷却期机制，仍然会在5秒后重新加载
- ❌ 违反了"正常运行期间不需要重新加载"的原则

**问题**：
- 查询类API（status、account、positions等）每15秒轮询
- 每次请求都会触发`LoadUserTraders`
- 导致trader被频繁停止并重新加载

---

## 业务逻辑点4：Cycle编号恢复

### 期望实现
- 重新加载后如果有历史cycle，继续cycle编号

### 当前实现
```go
// logger/decision_logger.go:83-99
// ✅ 从日志文件中恢复cycle number
cycleNumber := 0
files, err := ioutil.ReadDir(logDir)
if err == nil {
    for _, file := range files {
        // 解析文件名，找到最大的cycle number
        var cycle int
        _, err := fmt.Sscanf(file.Name(), "decision_%*s_cycle%d.json", &cycle)
        if err == nil && cycle > cycleNumber {
            cycleNumber = cycle
        }
    }
}
```

**差异**：
- ✅ **已修复**：从日志文件恢复cycle number
- ✅ 符合期望

---

## 业务逻辑点5：查询类API

### 期望实现
- 前端除了trader启动请求之外，其他的查询类请求：
  - **不应触发加载trader**
  - **不应触发开始/停止trader**
  - 只查询已加载的trader状态

### 当前实现
```go
// 以下API都使用getTraderFromQuery，都会触发LoadUserTraders：
// - GET /api/status
// - GET /api/account
// - GET /api/positions
// - GET /api/decisions/latest
// - GET /api/statistics
// - GET /api/performance
// - GET /api/equity-history
```

**差异**：
- ❌ **所有查询类API都会触发`LoadUserTraders`**
- ❌ 导致trader被频繁重新加载
- ❌ 违反了"查询类请求不应触发加载"的原则

**问题**：
- 用户在trader详情页面时，每15秒会同时发送3个API请求
- 每个请求都会触发`LoadUserTraders`
- 导致trader被频繁停止并重新加载

---

## 核心问题总结

### 问题1：职责混淆

**当前逻辑**：
- `getTraderFromQuery`既负责查询，又负责加载
- 所有查询类API都通过`getTraderFromQuery`获取trader
- 导致所有查询都会触发加载

**期望逻辑**：
- 查询类API：只查询已加载的trader，不触发加载
- 启动类API：确保trader已加载（从数据库加载），然后启动

### 问题2：加载时机错误

**当前逻辑**：
- 创建trader时立即加载（虽然不启动）
- 查询类API时也加载（违反原则）
- 启动时假设已加载（可能失败）

**期望逻辑**：
- 创建trader时不加载
- 启动trader时才加载
- 查询类API不加载

### 问题3：状态管理混乱

**当前逻辑**：
- 数据库状态（`is_running`）：记录trader是否应该运行
- 内存状态（`isRunning`）：trader是否正在运行
- 查询类API会触发重新加载，导致状态不一致

**期望逻辑**：
- 数据库状态：持久化trader应该的状态
- 内存状态：trader当前运行状态
- 查询类API只读取内存状态，不修改状态

---

## 修复方案

### 方案1：分离查询和加载逻辑（推荐）

**核心思路**：
1. `getTraderFromQuery`：只查询已加载的trader，不触发加载
2. `ensureTraderLoaded`：确保trader已加载（只在启动时使用）
3. 查询类API：只查询，不加载
4. 启动类API：确保加载后再启动

**实现步骤**：

1. **修改`getTraderFromQuery`**（查询类API使用）
2. **新增`ensureTraderLoaded`**（启动类API使用）
3. **修改`handleStartTrader`**（启动时确保加载）


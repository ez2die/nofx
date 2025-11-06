# 交易历史持久化模块实施计划

## 实施概览

### 目标
- 将交易历史持久化到数据库
- 与交易所API保持同步
- 作为独立模块，不影响现有功能

### 实施原则
1. **渐进式开发**：分阶段实施，每个阶段可独立测试
2. **向后兼容**：不影响现有JSON日志系统
3. **可配置**：支持启用/禁用功能
4. **最小化影响**：尽量少修改现有代码

## 阶段划分

### 阶段1：基础架构搭建（预计2天）

#### 1.1 创建模块目录结构
```
nofx/
├── trade_history/          # 新建目录
│   ├── models.go          # 数据模型
│   ├── repository.go      # 数据库操作
│   ├── service.go         # 业务逻辑
│   ├── interfaces.go      # 接口定义
│   └── sync.go            # 同步服务
```

#### 1.2 数据库表创建
- 在 `config/database.go` 的 `createTables()` 中添加 `trade_history` 表
- 创建索引和触发器
- 添加数据库迁移逻辑（向后兼容）

#### 1.3 数据模型定义
- 定义 `TradeRecord` 结构体
- 定义 `TradeRecordFilter` 结构体
- 定义 `TradeStatistics` 结构体

#### 1.4 Repository 层实现
- 实现 `Save()` 方法
- 实现 `FindByFilter()` 方法
- 实现 `GetStatistics()` 方法
- 实现 `ExistsByExchangeID()` 方法（去重）

**验收标准**：
- ✅ 数据库表创建成功
- ✅ 可以保存和查询交易记录
- ✅ 单元测试通过

---

### 阶段2：核心功能实现（预计3天）

#### 2.1 Service 层实现
- 实现 `RecordTrade()` 方法（记录交易）
- 实现 `RecordAutoTriggeredClose()` 方法（记录自动平仓）
- 实现 `GetTradeHistory()` 方法（查询历史）
- 实现 `GetTradeStatistics()` 方法（统计）

#### 2.2 交易所适配器实现
- 实现 `HyperliquidFillsProvider`
- 实现 `GetRecentFills()` 方法
- 实现 `GetFillsByTimeRange()` 方法
- 数据格式转换（Hyperliquid Fill → ExchangeFill）

#### 2.3 集成点实现（可选方式）

**方案A：直接调用（推荐，简单）**
- 在 `executeOpenLongWithRecord` 等函数中，API成功返回后调用
- 使用 `if tradeHistoryService != nil` 判断是否启用
- 异步执行，不阻塞主流程

**方案B：事件驱动（可选，更解耦）**
- 实现事件总线
- 在关键位置发布事件
- 事件监听器处理写入

#### 2.4 自动触发平仓记录
- 在 `runCycle()` 中检测到自动触发的平仓后
- 调用 `RecordAutoTriggeredClose()` 记录
- 计算PnL和PnLPct

**验收标准**：
- ✅ 可以记录开仓交易
- ✅ 可以记录平仓交易
- ✅ 可以记录自动触发的平仓
- ✅ 可以查询交易历史
- ✅ 集成测试通过

---

### 阶段3：同步功能实现（预计2天）

#### 3.1 同步服务实现
- 实现 `SyncService`
- 实现 `SyncFromExchange()` 方法
- 实现定期同步逻辑
- 实现去重逻辑（避免重复写入）

#### 3.2 同步策略
- **增量同步**：基于时间范围查询
- **全量同步**：获取所有历史记录
- **去重机制**：通过 `exchange_order_id` 或 `exchange_trade_id` 判断

#### 3.3 定期同步
- 在 `AutoTrader` 中添加定期同步任务
- 可配置同步间隔（默认10分钟）
- 支持手动触发同步

#### 3.4 数据一致性校验
- 对比数据库和交易所数据
- 发现缺失数据时自动补充
- 记录同步日志

**验收标准**：
- ✅ 可以手动触发同步
- ✅ 可以定期自动同步
- ✅ 同步后数据一致性验证通过

---

### 阶段4：API接口实现（预计1天）

#### 4.1 REST API 实现
- `GET /api/trade-history` - 获取交易历史
- `GET /api/trade-history/statistics` - 获取统计信息
- `POST /api/trade-history/sync` - 手动触发同步

#### 4.2 查询参数支持
- `trader_id` - 交易员ID
- `symbol` - 币种
- `action` - 操作类型
- `start_time` / `end_time` - 时间范围
- `limit` / `offset` - 分页

#### 4.3 响应格式
- 统一JSON响应格式
- 包含分页信息（总数、当前页、总页数）
- 错误处理

**验收标准**：
- ✅ API接口可用
- ✅ 查询参数正确解析
- ✅ 响应格式符合预期

---

### 阶段5：测试和优化（预计2天）

#### 5.1 单元测试
- Repository 层测试
- Service 层测试
- 交易所适配器测试

#### 5.2 集成测试
- 端到端测试（API返回 → 数据库写入）
- 同步功能测试
- API接口测试

#### 5.3 性能优化
- 批量插入优化
- 索引优化
- 查询性能优化

#### 5.4 文档完善
- API文档
- 使用示例
- 配置说明

**验收标准**：
- ✅ 单元测试覆盖率 > 80%
- ✅ 集成测试通过
- ✅ 性能测试通过（写入延迟 < 100ms）
- ✅ 文档完善

---

## 详细实施步骤

### 步骤1：创建模块目录和基础文件

```bash
# 创建目录
mkdir -p nofx/trade_history

# 创建文件
touch nofx/trade_history/models.go
touch nofx/trade_history/repository.go
touch nofx/trade_history/service.go
touch nofx/trade_history/interfaces.go
touch nofx/trade_history/sync.go
touch nofx/trade_history/sync_hyperliquid.go
```

### 步骤2：数据库表创建

在 `config/database.go` 的 `createTables()` 中添加：

```go
// 交易历史表
`CREATE TABLE IF NOT EXISTS trade_history (
    -- ... (见设计文档)
)`,
```

### 步骤3：实现 Repository 层

实现 `Save()`, `FindByFilter()`, `GetStatistics()` 等方法。

### 步骤4：实现 Service 层

实现 `RecordTrade()`, `GetTradeHistory()` 等方法。

### 步骤5：实现 Hyperliquid 适配器

实现 `HyperliquidFillsProvider` 接口。

### 步骤6：集成到 AutoTrader

在 `executeOpenLongWithRecord` 等函数中添加写入逻辑。

### 步骤7：实现同步功能

实现定期同步和手动同步。

### 步骤8：实现 API 接口

在 `api/server.go` 中添加相关接口。

### 步骤9：测试

编写单元测试和集成测试。

### 步骤10：文档

更新API文档和使用说明。

---

## 代码修改清单

### 新增文件
- `trade_history/models.go`
- `trade_history/repository.go`
- `trade_history/service.go`
- `trade_history/interfaces.go`
- `trade_history/sync.go`
- `trade_history/sync_hyperliquid.go`

### 修改文件
- `config/database.go` - 添加表创建和迁移逻辑
- `trader/auto_trader.go` - 添加集成点（可选）
- `api/server.go` - 添加API接口
- `manager/trader_manager.go` - 初始化trade_history服务（可选）

### 配置文件
- `config.json` - 添加trade_history配置（可选）

---

## 风险评估

### 风险1：数据一致性
- **影响**：数据库与交易所数据不一致
- **缓解**：定期同步 + 去重机制

### 风险2：性能影响
- **影响**：写入操作影响交易流程
- **缓解**：异步写入 + 批量插入

### 风险3：向后兼容
- **影响**：影响现有功能
- **缓解**：可选集成 + 可配置启用/禁用

---

## 测试计划

### 单元测试
- Repository.Save() - 测试保存功能
- Repository.FindByFilter() - 测试查询功能
- Service.RecordTrade() - 测试记录功能
- HyperliquidFillsProvider - 测试适配器

### 集成测试
- 端到端测试：API返回 → 数据库写入
- 同步测试：交易所API → 数据库同步
- API测试：HTTP请求 → 数据库查询

### 性能测试
- 写入性能：单次写入 < 100ms
- 批量写入：100条 < 1s
- 查询性能：1000条 < 500ms

---

## 验收标准

### 功能验收
- ✅ 可以记录开仓交易
- ✅ 可以记录平仓交易
- ✅ 可以记录自动触发的平仓
- ✅ 可以查询交易历史
- ✅ 可以统计交易数据
- ✅ 可以与交易所同步

### 性能验收
- ✅ 写入延迟 < 100ms
- ✅ 查询延迟 < 500ms
- ✅ 同步延迟 < 5分钟

### 稳定性验收
- ✅ 不影响现有功能
- ✅ 错误处理完善
- ✅ 日志记录完整

---

## 后续优化

### 短期（1-2周）
- 支持更多交易所（Binance、Aster等）
- 优化查询性能
- 添加更多统计指标

### 中期（1-2月）
- 数据归档策略
- 数据导出功能
- 图表展示

### 长期（3-6月）
- 实时数据同步（WebSocket）
- 数据分析和报表
- 机器学习特征工程


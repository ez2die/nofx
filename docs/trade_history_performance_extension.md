## 背景
- 目前 `logger/decision_logger.go` 中的 `PerformanceAnalysis` 结构包含胜率、盈亏比、Sharpe、最近交易、按币种拆分等丰富的维度，用于诊断策略实时表现。
- `trade_history` 模块的 `TradeStatistics` 仅提供基础汇总（总笔数、胜负笔数、胜率、总盈亏、平均盈亏、盈亏比、总手续费），缺少风险指标与拆分维度。
- 用户希望以 trade history 为唯一数据源，把 `PerformanceAnalysis` 所需的指标全部补齐，以便统一来源、持久化和回溯。

## 需求
1. **统一数据源**：所有决策日志和外部展示层的交易表现指标均来源于 `trade_history` 模块。
2. **维度补齐**：在 trade history 中新增以下能力：
   - 最近 N 笔交易明细（含时间、盈亏、方向、币种等）。
   - 按币种（Symbol）聚合的胜率、总盈亏、平均盈亏、盈亏笔数。
   - 自动推导表现最佳 / 最差币种（可按总盈亏或 Sharpe）。
   - 风险调整指标（Sharpe Ratio，未来可扩展 Sortino）。
3. **配置化**：最近交易的数量 N、Sharpe 计算窗口、统计时间范围等需要可配置（系统配置或 API 参数）。
4. **API 输出**：新增或扩展 REST API，使前端/日志可以一次性获取完整 `PerformanceAnalysis` 等价信息。
5. **性能与准确性**：统计查询需可分页/带时间范围筛选，单位时间内多次查询不会对数据库造成显著压力。

## 目标结构
```go
type TradePerformance struct {
    TradeStatistics            // 复用既有字段
    SharpeRatio    float64
    RecentTrades   []TradeOutcome
    SymbolStats    []SymbolPerformance
    BestSymbol     string
    WorstSymbol    string
}
```

## 设计方案

### 1. Repository 层扩展
- 新增查询：
  1. `GetRecentTrades(ctx, traderID string, limit int, startTime, endTime *time.Time) ([]TradeOutcome, error)`
  2. `GetSymbolStats(ctx, traderID string, startTime, endTime *time.Time) ([]SymbolPerformance, error)`
  3. `GetPnLSeries(ctx, traderID string, startTime, endTime *time.Time) ([]PnLPoint, error)` 用于 Sharpe 计算（返回 timestamp + pnl）。
- SQL 实现：
  - RecentTrades：`SELECT ... FROM trade_history WHERE trader_id=? AND timestamp BETWEEN ? ORDER BY timestamp DESC LIMIT ?`
  - SymbolStats：`SELECT symbol, COUNT(*), SUM(CASE WHEN pnl>0 THEN 1 ELSE 0 END) ... GROUP BY symbol`
  - PnLSeries：可直接复用 `trade_history` 表中的 `pnl` 字段（仅平仓单有值），必要时合成盈亏序列。

### 2. Service 层聚合
- 新增 `GetTradePerformance(ctx, traderID string, options PerformanceOptions) (*TradePerformance, error)`。
- 步骤：
  1. 调用现有 `GetTradeStatistics`.
  2. 获取 `recentTrades`（limit 来自 options，默认 20）。
  3. 获取 `symbolStats`，并在 service 里计算 `BestSymbol`/`WorstSymbol`（优先指标：总盈亏；若没有盈亏则按胜率）。
  4. 使用 `GetPnLSeries` 结果计算 Sharpe：`Sharpe = (mean(return) / stddev(return)) * sqrt(periodsPerYear)`；return 可用 `pnl / notional` 或简化为 `pnl` 序列（待确认）。
- Options 结构示例：
```go
type PerformanceOptions struct {
    StartTime  *time.Time
    EndTime    *time.Time
    RecentLimit int
    SharpeWindow int // 最近多少笔/天用于 Sharpe
}
```

### 3. API 层
- 新增 `GET /api/trade-history/performance`（或在 `/statistics` 中通过 query 参数 `detail=performance` 返回扩展结构），但与现有 `PerformanceAnalysis` 使用场景解耦。
- 请求参数：`trader_id`（必填）、`start_time`、`end_time`、`recent_limit`、`sharpe_window`。
- 返回 JSON 包含 `TradePerformance` 所有字段，为外部分析或后续接入做准备；当前决策日志仍沿用原数据源。

### 4. 配置与常量
- `system_config` 新增：
  - `performance_recent_trades_limit`（默认 20）
  - `performance_sharpe_window`（默认 30）
- fallback 到常量，调用方可覆盖。

### 5. 兼容性与迁移
- `TradeStatistics` 保持现有结构，向后兼容。
- 新增 `SymbolPerformance`、`TradeOutcome` 等结构可直接复用 `logger` 中的定义，或迁移这些结构到公共包（例如 `models/performance`），避免重复。
- 更新 `decision_logger`：删除本地统计逻辑，改为依赖 service 返回，为保持稳定可在过渡期双轨验证（flag 控制）。

### 6. 未来扩展
- Sortino Ratio、最大回撤、持仓持有时长分布等指标可在 `GetTradePerformance` 中逐步添加。
- 支持按 exchange / strategy / 子账户等更多维度分组。

## 里程碑
1. **M1 - Repository 扩展**：实现 SQL 查询与模型定义，提供单元测试。
2. **M2 - Service 聚合**：实现 `GetTradePerformance`、Sharpe 计算、最佳/最差币种逻辑。
3. **M3 - API & Logger 集成**：新增 API endpoint，更新 `decision_logger` 使用新接口。
4. **M4 - 配置与监控**：增加系统配置项、指标、必要的 Prometheus 监控。

## 实现步骤细化

### M1 - Repository 扩展
- **Schema/Model**：在 `trade_history/models.go` 中新增 `TradeOutcome`、`SymbolPerformance`、`PnLPoint` 等结构体，并为 SQL 扫描提供 `db` tag。
- **接口定义**：在 `trade_history/repository.go` 定义新方法；更新 `interfaces.go` 保证 service 依赖倒置。
- **SQL 细节**：
  - `GetRecentTrades`：仅返回平仓交易（`pnl IS NOT NULL`，对应 `action IN ('close_long','close_short')`），因为未平仓记录无法确定 PnL；支持 `LIMIT`、`OFFSET`，按 `timestamp DESC`。
  - `GetSymbolStats`：使用聚合统计 `COUNT(*)`, `SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END)`、`SUM(pnl)`、`AVG(pnl)`；NULL `pnl` 需 `COALESCE(pnl,0)`。
  - `GetPnLSeries`：返回平仓单的 `(timestamp, pnl)`；缺失 `pnl` 的记录过滤掉。
- **单元测试**：使用 sqlite 内存库构造数据集，覆盖无数据、有亏损、有手续费、limit 生效等场景。

### M2 - Service 聚合
- **Options 默认值**：`RecentLimit=20`、`SharpeWindow=30`；若 `StartTime/EndTime` 为空则在 service 层自动设为 `[now-30d, now]`。
- **调用顺序**：先查询基础统计，若总交易数为 0 直接返回空结构并跳过其他查询（节省查询成本）。
- **Sharpe 计算**：在 service 内部拉取 PnL 序列并计算（见「计算逻辑」章节）；window 参数生效时，仅取最近 N 条 PnL。
- **最佳/最差币种**：在 service 中对 `[]SymbolPerformance` 排序或单次遍历，优先指标 `TotalPnL`，若相等则按 `WinRate`。
- **错误处理**：单个查询失败即返回错误，避免混合旧数据；可用 `multierr` 聚合。

### M3 - API & Logger 集成
- **API Handler**：在 `api/server.go` 中新增 `handleGetTradePerformance`（或扩展 `handleGetTradeStatistics`）；解析 query 参数映射到 `PerformanceOptions`。
- **路由**：`GET /api/trade-history/performance`，需要 auth，与统计接口相同权限。
- **Logger 集成**：本阶段不改动 `logger/decision_logger.go`，保持 `PerformanceAnalysis` 现有数据来源；新统计可作为独立接口供调试使用。
- **校验**：可通过手动调用新 API 与现有日志比对，但不影响线上逻辑。

### M4 - 配置与监控
- **系统配置**：在 `config/system_config.go` 中注册新 key，提供默认值；`TraderManager` 初始化时注入到 service。
- **Prometheus**：增加查询耗时、Sharpe 计算耗时等指标（例如 `trade_history_perf_latency`）。
- **告警**：若 Sharpe 计算因样本不足返回 NaN，记录 warning。
- **文档/Runbook**：更新运营手册，说明如何通过 API 获取 performance 数据及常见问题。

## 计算逻辑

### 基础统计（复用现有 `TradeStatistics`）
- **TotalTrades**：满足 filter 的交易数量。
- **WinningTrades/LosingTrades**：`pnl > 0` 计胜，`pnl < 0` 计负，`pnl = 0 or NULL` 可计为 `draw`（不纳入胜负）或跟现有行为保持一致。
- **WinRate**：`WinningTrades / (WinningTrades + LosingTrades)`；若分母为 0 返回 0。
- **AvgWin/AvgLoss**：分别对盈利/亏损样本求平均；无样本取 0。
- **ProfitFactor**：`sum(pnl>0)/abs(sum(pnl<0))`；若 denominator=0，则返回 `math.Inf` 或 0（保持现有逻辑）。
- **TotalPnL**：`SUM(COALESCE(pnl,0))`。
- **TotalFees**：`SUM(fee)`。

### RecentTrades
- **字段**：`Symbol`、`Action`、`Side`、`Quantity`、`ExecutionPrice`、`PnL`、`Fee`、`Timestamp`（全部来自平仓记录）。
- **排序**：`timestamp DESC`。
- **过滤**：使用 `StartTime/EndTime`（默认最近 30 天），仅返回 `pnl IS NOT NULL` 的平仓记录；`RecentLimit` 控制数量。
- **用途**：直接填充 `PerformanceAnalysis.RecentTrades`。

### SymbolStats
- **指标**（仅统计平仓记录，与全局统计保持一致）：
  - `TotalTrades`: 每个 symbol 的平仓笔数。
  - `WinningTrades/LosingTrades`: `SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END)`，`SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END)`。
  - `WinRate`: `WinningTrades / max(1, Winning+Losing)`。
  - `TotalPnL`: `SUM(pnl)`。
  - `AvgPnL`: `TotalPnL / TotalTrades`。
- **Best/Worst Symbol**：
  1. 先按 `TotalPnL` 降序/升序。
  2. 如 PnL 相等，按 `WinRate`。
  3. 若仍相等，按交易次数多者优先。
- **返回格式**：Service 层整理为 `map[string]*SymbolPerformance` 以兼容 `PerformanceAnalysis.SymbolStats`，并同时提供列表以便 API 扩展（如有需要）。
- **空结果**：返回空 map、空字符串，避免 null。

### Sharpe Ratio
- **输入**：`[]PnLPoint`（每笔平仓的 pnl + timestamp + signed_quantity + execution_price）。
- **收益序列**：
  - 默认使用 `return_i = pnl_i / notional_i`，其中 `notional_i = abs(signed_quantity * execution_price)`；若 `signed_quantity` 缺失，则退化为 `return_i = pnl_i`。
- **均值与波动率**：
  - `mean = avg(return_i)`。
  - `std = stddev(return_i)`（Welford 算法）。
  - 若 `std == 0`，返回 0。
- **年化/周期调整**：
  - 根据样本的时间跨度：`windowDays = max(1, (last.timestamp - first.timestamp)/24h)`；若跨度不足 1 天则按 1 天处理。
  - `Sharpe = mean / std * sqrt(365 / windowDays)`。
- **样本不足**：当样本数 < 2 时返回 0，并记录 warning。

### 结果封装
- Service 将上述指标填入 `TradePerformance`：
  - `TradeStatistics`：直接引用。
  - `SharpeRatio`：计算结果。
  - `RecentTrades`：repo 查询结果映射为 `[]TradeOutcome`。
  - `SymbolStats`：repo 聚合结果转换为 `map[string]*SymbolPerformance`（提供兼容结构）。
  - `BestSymbol/WorstSymbol`：依据 `SymbolStats` 计算。
- API 返回保持与 `PerformanceAnalysis` 相同的 JSON 字段，同时可增加 `generated_at` 时间戳供调试。

## 配置与默认值
- `system_config` 默认值：
  - `performance_recent_trades_limit = '20'`
  - `performance_sharpe_window = '30'`
- 如果系统未设置，service fallback 到常量；同时提供 SQL 片段供手动写入。

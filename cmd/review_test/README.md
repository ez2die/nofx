# 复盘模块测试程序

这个测试程序用于独立测试复盘模块的所有功能。

## 使用方法

### 一次性运行（执行一次复盘）

```bash
RUN_ONCE=true go run cmd/review_test/main.go <trader_id> [db_path]
```

示例：
```bash
RUN_ONCE=true go run cmd/review_test/main.go hyperliquid_xxx config.db
```

### 持续运行（启动调度器，每6小时自动复盘）

```bash
go run cmd/review_test/main.go <trader_id> [db_path]
```

示例：
```bash
go run cmd/review_test/main.go hyperliquid_xxx config.db
```

## 环境变量

- `RUN_ONCE=true`: 执行一次复盘后退出，否则持续运行调度器
- `USE_MOCK_DEX=true`: 使用模拟的DEX数据提供者（默认使用模拟，因为需要真实的交易所连接）

## 依赖

程序会自动初始化以下依赖：

1. **DecisionLogger**: 从 `decision_logs/<trader_id>/` 目录读取决策日志
2. **TradeHistoryReader**: 从数据库读取交易历史
3. **DEXDataProvider**: 当前使用模拟数据（需要在实际集成时提供真实的交易所接口）
4. **TradeAnalyticsService**: 从交易分析模块获取统计数据
5. **ReviewRepository**: 将复盘结果保存到数据库

## 输出

- 复盘报告会保存到 `data/reviews/` 目录
- 复盘记录会保存到数据库的 `review_records` 表

## 注意事项

1. 确保数据库已初始化（包括 `review_records` 表）
2. 确保 `decision_logs/<trader_id>/` 目录存在且包含决策日志文件
3. 确保交易历史数据已同步到数据库
4. 实际使用时需要提供真实的 DEX 数据提供者（当前使用模拟数据）


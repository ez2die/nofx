# Performance Validation Script

## 功能说明

这是一个独立的验证脚本，用于验证 AI Learning & Reflection 页面显示的交易表现数据的准确性。

## 功能特性

1. **按时间戳排序**: 从 `hyperliquid_deepseek` 目录读取所有决策日志文件，按文件名中的时间戳（而非 cycle 编号）排序
2. **分析所有周期**: 分析目录中的所有交易周期（不限制数量）
3. **识别成交记录**: 自动识别包含成功交易（open_long/open_short/close_long/close_short）的周期
4. **FIFO匹配交易对**: 按照先进先出（FIFO）规则匹配开仓和平仓记录，生成完整交易
5. **计算统计数据**:
   - 总交易数
   - 胜率
   - 平均盈利/亏损
   - 盈亏比
   - 各币种表现统计
   - 最佳/最差币种

## 使用方法

### 编译

```bash
cd validation
go build -o validate_performance validate_performance.go
```

### 运行

```bash
# 从项目根目录运行，使用默认路径（decision_logs/hyperliquid_deepseek）
./validation/validate_performance

# 或指定自定义路径
./validation/validate_performance decision_logs/hyperliquid_deepseek

# 或者在validation目录下运行（需要指定相对路径）
cd validation
./validate_performance ../decision_logs/hyperliquid_deepseek
```

### 输出

脚本会生成 `validation_result.json` 文件，包含完整的验证结果。

同时会在控制台输出：
- 分析的周期数和文件数
- 包含交易的周期数
- 匹配的完整交易数
- 总体统计数据（胜率、平均盈亏、盈亏比等）
- 最佳/最差币种
- 各币种表现统计（Top 10）

## 输出格式

### JSON 结构

```json
{
  "total_cycles": 100,
  "cycles_with_trades": 45,
  "total_trades": 120,
  "winning_trades": 65,
  "losing_trades": 55,
  "win_rate": 54.17,
  "avg_win": 12.34,
  "avg_loss": -8.90,
  "profit_factor": 1.57,
  "completed_trades": [...],
  "symbol_stats": {
    "BTCUSDT": {
      "symbol": "BTCUSDT",
      "total_trades": 20,
      "winning_trades": 12,
      "losing_trades": 8,
      "win_rate": 60.0,
      "total_pn_l": 123.45,
      "avg_pn_l": 6.17
    },
    ...
  },
  "best_symbol": "BTCUSDT",
  "worst_symbol": "ETHUSDT"
}
```

## 与前端数据对比

生成的 `validation_result.json` 可以与前端 `/api/performance` 接口返回的数据进行对比，验证：

1. **Total Trades**: 总交易数是否一致
2. **Win Rate**: 胜率是否一致
3. **Average Win/Loss**: 平均盈亏是否一致
4. **Profit Factor**: 盈亏比是否一致
5. **Symbol Statistics**: 各币种表现统计是否一致
6. **Best/Worst Symbol**: 最佳/最差币种是否一致

## 注意事项

1. **Cycle 编号重置**: 脚本使用文件名中的时间戳排序，不依赖 cycle 编号，因此可以正确处理 cycle 重置的情况
2. **部分平仓**: 脚本支持部分平仓的匹配，会正确拆分持仓进行计算
3. **FIFO 匹配**: 采用先进先出（FIFO）规则匹配开仓和平仓，与实际交易系统保持一致
4. **盈亏计算**: 
   - 做多: PnL = (平仓价 - 开仓价) × 数量 × 杠杆
   - 做空: PnL = (开仓价 - 平仓价) × 数量 × 杠杆

## 故障排除

如果遇到问题：

1. **文件读取失败**: 确保日志目录路径正确
2. **JSON 解析错误**: 检查日志文件格式是否正确
3. **交易匹配异常**: 检查开仓和平仓记录是否完整，确保 `success=true`


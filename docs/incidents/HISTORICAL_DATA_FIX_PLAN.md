# 历史数据修复方案

## 问题描述

历史日志中的平仓记录（`close_long`、`close_short`）的 `quantity` 字段为 0，导致：
1. PL（盈亏）计算为 0
2. 胜率、盈利、亏损、盈亏比无法正确计算
3. 历史成交数据显示 PL 都是 0

## 修复方案

采用**混合方案**修复历史数据：

### 方案1（优先）：从 positions 快照中恢复
- 每个决策记录都包含 `positions` 字段，记录了当前周期的持仓快照
- 对于平仓记录，如果 `quantity == 0`，从同一记录的 `positions` 中查找对应 `symbol` 和 `side` 的持仓
- 使用 `position_amt` 作为平仓的 `quantity`
- **优点**：最准确，因为是平仓时的实际持仓数量
- **适用场景**：positions 中有对应持仓的情况

### 方案2（备用）：从开仓记录中恢复
- 如果 positions 中没有找到对应持仓，向前查找对应的开仓记录
- 使用 FIFO（先进先出）匹配：最早的开仓记录匹配最早的平仓记录
- 使用开仓时的 `quantity` 作为平仓的 `quantity`
- **优点**：总是能找到（如果开仓记录存在）
- **适用场景**：positions 中没有持仓，但有开仓记录的情况

## 修复脚本

创建了 `fix_historical_close_quantity.go` 脚本来批量修复历史数据。

### 使用方法

```bash
# 修复单个trader的历史数据
go run fix_historical_close_quantity.go decision_logs/hyperliquid_a5e99c3c-bef8-4b5b-a19a-01328b593128_deepseek_1762195291

# 修复所有trader的历史数据（需要遍历所有目录）
for dir in decision_logs/*/; do
    echo "修复: $dir"
    go run fix_historical_close_quantity.go "$dir"
done
```

### 脚本功能

1. **按时间排序**：读取所有日志文件并按时间戳排序（从旧到新）
2. **FIFO匹配**：维护开仓记录队列，使用 FIFO 匹配开仓-平仓
3. **双重恢复**：
   - 优先从 positions 快照中恢复
   - 如果 positions 中没有，从开仓记录中恢复
4. **批量修复**：一次性修复所有日志文件
5. **保留完整性**：保留 JSON 文件的所有其他字段，只更新 `decisions` 字段

### 输出示例

```
🔧 开始修复历史数据：decision_logs/hyperliquid_xxx/...

📁 找到 163 个日志文件

🔧 开始修复平仓记录的quantity...
  ✅ decision_20251104_133535_cycle151.json: close_short BTCUSDT quantity=0.00453000 leverage=5
  ✅ decision_20251104_085149_cycle83.json: close_short ETHUSDT quantity=0.13750000 leverage=10
  ...

✅ 修复完成！
   - 处理的平仓记录: 2
   - 成功修复: 2
   - 修改的文件数: 2
```

## 注意事项

1. **备份数据**：修复前建议备份 `decision_logs` 目录
   ```bash
   cp -r decision_logs decision_logs_backup
   ```

2. **验证修复**：修复后可以运行以下命令验证：
   ```bash
   # 检查修复后的quantity是否已更新
   find decision_logs -name "*.json" -exec jq '.decisions[]? | select(.action == "close_long" or .action == "close_short") | {action, symbol, quantity}' {} \;
   ```

3. **重新分析**：修复后需要重新运行 `AnalyzePerformance` 才能看到正确的 PL 计算

4. **部分平仓**：如果存在部分平仓的情况，positions 快照中的数量可能不准确，此时会使用开仓记录中的数量

## 修复后的效果

修复后：
- ✅ 平仓记录的 `quantity` 字段会被正确填充
- ✅ PL 计算会使用正确的 quantity，不再是 0
- ✅ 胜率、盈利、亏损、盈亏比可以正确计算
- ✅ 历史成交数据会显示正确的 PL

## 代码修复（已完成）

已修复的代码：
1. `trader/auto_trader.go`：平仓时记录 quantity
2. `logger/decision_logger.go`：PL 计算时优先使用平仓时的 quantity

这两个修复确保**新产生的平仓记录**会正确记录 quantity，历史数据需要通过脚本修复。


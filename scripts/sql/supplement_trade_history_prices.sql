-- 补充交易历史记录中缺失的 entry_price 和 exit_price
-- 对于平仓记录，exit_price 应该等于 execution_price
-- entry_price 需要从对应的开仓记录中查找

-- 1. 更新 exit_price：对于平仓记录，如果没有 exit_price，使用 execution_price
UPDATE trade_history 
SET exit_price = execution_price 
WHERE action IN ('close_long', 'close_short') 
  AND exit_price IS NULL 
  AND execution_price IS NOT NULL;

-- 2. 对于平仓记录，尝试从同一 trader_id 和 symbol 的最近开仓记录中获取 entry_price
-- 这个方法比较复杂，需要通过子查询来匹配
-- 首先，我们需要为每个平仓记录找到对应的开仓记录
-- 但由于我们没有 cycle_number 或其他关联字段，只能通过时间和数量来匹配

-- 对于有 pnl 数据的平仓记录，我们可以反推 entry_price
-- 对于 close_long: pnl = (exit_price - entry_price) * quantity * leverage
-- 对于 close_short: pnl = (entry_price - exit_price) * quantity * leverage
-- 因此：entry_price = exit_price - (pnl / (quantity * leverage)) 对于 long
--       entry_price = exit_price + (pnl / (quantity * leverage)) 对于 short

-- 更新 close_long 的 entry_price（如果有 pnl 数据）
UPDATE trade_history 
SET entry_price = exit_price - (pnl / (quantity * leverage))
WHERE action = 'close_long' 
  AND entry_price IS NULL 
  AND exit_price IS NOT NULL 
  AND pnl IS NOT NULL 
  AND quantity > 0 
  AND leverage > 0;

-- 更新 close_short 的 entry_price（如果有 pnl 数据）
UPDATE trade_history 
SET entry_price = exit_price + (pnl / (quantity * leverage))
WHERE action = 'close_short' 
  AND entry_price IS NULL 
  AND exit_price IS NOT NULL 
  AND pnl IS NOT NULL 
  AND quantity > 0 
  AND leverage > 0;

-- 验证更新结果
-- SELECT id, action, symbol, entry_price, exit_price, execution_price, pnl 
-- FROM trade_history 
-- WHERE action IN ('close_long', 'close_short') 
-- LIMIT 10;


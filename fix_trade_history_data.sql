-- 修复交易历史数据中的格式问题
-- 1. 修复 action 字段：close_shor -> close_short
-- 2. 修复 side 字段：A/B -> long/short（根据实际情况推断）

-- 首先查看需要修复的数据
-- SELECT id, side, action FROM trade_history WHERE side IN ('A', 'B') OR action LIKE '%shor%';

-- 修复 action 字段格式问题
UPDATE trade_history 
SET action = 'close_short' 
WHERE action = 'close_shor';

-- 修复 side 字段：根据 action 推断正确的 side
-- 如果 action 是 close_long，则 side 应该是 long
UPDATE trade_history 
SET side = 'long' 
WHERE action = 'close_long' AND side IN ('A', 'B');

-- 如果 action 是 close_short，则 side 应该是 short
UPDATE trade_history 
SET side = 'short' 
WHERE action = 'close_short' AND side IN ('A', 'B');

-- 如果 action 是 open_long，则 side 应该是 long
UPDATE trade_history 
SET side = 'long' 
WHERE action = 'open_long' AND side IN ('A', 'B');

-- 如果 action 是 open_short，则 side 应该是 short
UPDATE trade_history 
SET side = 'short' 
WHERE action = 'open_short' AND side IN ('A', 'B');

-- 对于其他情况，根据 closed_pnl 和价格变化推断（如果可能）
-- 这里先不处理，因为需要更多上下文信息

-- 验证修复结果
-- SELECT id, side, action, symbol FROM trade_history WHERE side IN ('A', 'B') OR action LIKE '%shor%';


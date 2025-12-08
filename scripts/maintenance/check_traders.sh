#!/bin/bash

# 检查所有已配置但未运行的trader

DB_PATH="config.db"

echo ""
echo "🔍 检查所有已配置但未运行的trader..."
echo ""

# 检查数据库文件是否存在
if [ ! -f "$DB_PATH" ]; then
    echo "❌ 数据库文件不存在: $DB_PATH"
    exit 1
fi

# 获取统计信息
echo "📊 统计信息:"
echo "----------------------------------------"
sqlite3 "$DB_PATH" <<EOF
SELECT 
    COUNT(*) as '总配置数',
    SUM(CASE WHEN is_running = 1 THEN 1 ELSE 0 END) as '运行中',
    SUM(CASE WHEN is_running = 0 THEN 1 ELSE 0 END) as '未运行'
FROM traders;
EOF

echo ""
echo ""

# 显示所有未运行的trader
echo "📋 未运行的trader (is_running = 0):"
echo "----------------------------------------"
NOT_RUNNING_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM traders WHERE is_running = 0;")

if [ "$NOT_RUNNING_COUNT" -eq 0 ]; then
    echo "✅ 所有trader都在运行中！"
else
    sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    t.id as 'Trader ID',
    t.name as '名称',
    u.email as '用户邮箱',
    t.ai_model_id as 'AI模型',
    t.exchange_id as '交易所',
    t.initial_balance as '初始余额',
    t.scan_interval_minutes as '扫描间隔(分钟)',
    t.created_at as '创建时间'
FROM traders t
LEFT JOIN users u ON t.user_id = u.id
WHERE t.is_running = 0
ORDER BY t.created_at DESC;
EOF
fi

echo ""
echo ""

# 显示所有trader的详细状态
echo "📋 所有trader状态:"
echo "----------------------------------------"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    t.id as 'Trader ID',
    t.name as '名称',
    u.email as '用户邮箱',
    t.ai_model_id as 'AI模型',
    t.exchange_id as '交易所',
    CASE WHEN t.is_running = 1 THEN '✅ 运行中' ELSE '❌ 未运行' END as '状态',
    t.initial_balance as '初始余额',
    t.created_at as '创建时间'
FROM traders t
LEFT JOIN users u ON t.user_id = u.id
ORDER BY t.is_running, t.created_at DESC;
EOF

echo ""
echo ""

# 检查是否有配置但可能缺少依赖的trader
echo "🔍 检查配置完整性:"
echo "----------------------------------------"

# 检查缺少AI模型配置的trader
MISSING_AI=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM traders t LEFT JOIN ai_models a ON t.ai_model_id = a.id WHERE a.id IS NULL;")
if [ "$MISSING_AI" -gt 0 ]; then
    echo "⚠️  发现 $MISSING_AI 个trader缺少AI模型配置:"
    sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    t.id as 'Trader ID',
    t.name as '名称',
    t.ai_model_id as '缺失的AI模型ID'
FROM traders t
LEFT JOIN ai_models a ON t.ai_model_id = a.id
WHERE a.id IS NULL;
EOF
else
    echo "✅ 所有trader的AI模型配置完整"
fi

echo ""

# 检查缺少交易所配置的trader
MISSING_EXCHANGE=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM traders t LEFT JOIN exchanges e ON t.exchange_id = e.id WHERE e.id IS NULL;")
if [ "$MISSING_EXCHANGE" -gt 0 ]; then
    echo "⚠️  发现 $MISSING_EXCHANGE 个trader缺少交易所配置:"
    sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    t.id as 'Trader ID',
    t.name as '名称',
    t.exchange_id as '缺失的交易所ID'
FROM traders t
LEFT JOIN exchanges e ON t.exchange_id = e.id
WHERE e.id IS NULL;
EOF
else
    echo "✅ 所有trader的交易所配置完整"
fi

echo ""
echo "✅ 检查完成"
echo ""


#!/bin/bash

# 检查后端API返回的trader状态

API_BASE="${API_BASE:-http://localhost:8080/api}"

echo ""
echo "🔍 检查后端API返回的trader状态..."
echo "API Base: $API_BASE"
echo ""

# 检查API是否可访问
if ! curl -s -f "$API_BASE/health" > /dev/null 2>&1; then
    echo "❌ 无法连接到后端API: $API_BASE"
    echo "   请确保后端服务正在运行"
    exit 1
fi

echo "✅ API连接正常"
echo ""

# 获取竞赛数据（公开接口，无需认证）
echo "📊 竞赛数据 (/api/competition):"
echo "----------------------------------------"
COMPETITION=$(curl -s "$API_BASE/competition")
if [ $? -eq 0 ]; then
    echo "$COMPETITION" | python3 -m json.tool 2>/dev/null || echo "$COMPETITION"
else
    echo "❌ 获取竞赛数据失败"
fi

echo ""
echo ""

# 获取公开trader列表
echo "📋 公开Trader列表 (/api/traders):"
echo "----------------------------------------"
TRADERS=$(curl -s "$API_BASE/traders")
if [ $? -eq 0 ]; then
    echo "$TRADERS" | python3 -m json.tool 2>/dev/null || echo "$TRADERS"
    
    # 统计未运行的trader
    echo ""
    echo "📊 统计:"
    echo "$TRADERS" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    total = len(data)
    running = sum(1 for t in data if t.get('is_running', False))
    stopped = total - running
    print(f'  总trader数: {total}')
    print(f'  运行中: {running}')
    print(f'  未运行: {stopped}')
    if stopped > 0:
        print()
        print('  未运行的trader:')
        for t in data:
            if not t.get('is_running', False):
                print(f'    - {t.get(\"trader_name\", \"N/A\")} ({t.get(\"trader_id\", \"N/A\")})')
except Exception as e:
    print(f'  解析失败: {e}')
"
else
    echo "❌ 获取trader列表失败"
fi

echo ""
echo "✅ 检查完成"
echo ""


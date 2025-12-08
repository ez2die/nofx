#!/bin/bash

# 检查trader03的数据库配置和内存中的实际配置

TRADER_ID="hyperliquid_d53550af-05cd-494d-8d6e-18fc940d15c9_deepseek_1762221685"
USER_ID="default"

echo "=== 检查 Trader 03 的模板配置 ==="
echo ""

# 1. 从数据库检查配置（更精确的方法）
echo "1. 数据库配置（从数据库文件直接读取）:"
docker exec nofx-trading-v2 sh -c "cat /app/config.db 2>/dev/null | strings" | grep -E "d53550af.*trader 03|trader 03.*d53550af" | grep -oE "(nof1|adaptive|default|lean)" | head -1 || echo "未找到"

echo ""
echo "2. 从日志检查trader03最近使用的模板:"
docker logs nofx-trading-v2 2>&1 | grep -E "\[模板:.*\]" | tail -5

echo ""
echo "3. 从最新决策日志检查实际使用的模板:"
LATEST_LOG=$(find /root/nofx/decision_logs/hyperliquid_d53550af-05cd-494d-8d6e-18fc940d15c9_deepseek_1762221685 -name "*.json" -type f | sort -r | head -1)
if [ -n "$LATEST_LOG" ]; then
    echo "最新日志: $(basename $LATEST_LOG)"
    echo "system_prompt开头:"
    grep -o '"system_prompt": "[^"]*ROLE[^"]*"' "$LATEST_LOG" | head -c 200
    echo ""
    echo ""
    # 检查是哪个模板
    if grep -q "# ROLE & IDENTITY" "$LATEST_LOG"; then
        echo "→ 使用的模板: nof1.txt (特征: '# ROLE & IDENTITY')"
    elif grep -q "你是专业的加密货币交易AI" "$LATEST_LOG"; then
        if grep -q "连续亏损检查\|BTC 状态确认\|滑点调整" "$LATEST_LOG"; then
            echo "→ 使用的模板: adaptive.txt (特征: '你是专业的加密货币交易AI' + '连续亏损检查'等)"
        else
            echo "→ 使用的模板: default.txt (特征: '你是专业的加密货币交易AI')"
        fi
    elif grep -q "^# ROLE$" "$LATEST_LOG" || grep -q "Autonomous crypto trading agent on Hyperliquid" "$LATEST_LOG"; then
        echo "→ 使用的模板: lean.txt (特征: '# ROLE\nAutonomous crypto trading agent')"
    else
        echo "→ 无法确定模板"
    fi
fi

echo ""
echo "4. 从日志中查看trader03加载时的配置:"
docker logs nofx-trading-v2 2>&1 | grep -A 5 "trader 03.*已加载\|trader 03.*已为用户加载" | tail -10


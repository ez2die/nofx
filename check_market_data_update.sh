#!/bin/bash

# 检查Trader 02和Trader 04最近两个cycle的市场数据更新情况

TRADER02_DIR="decision_logs/hyperliquid_26f0a132-0026-4516-b5ef-59a2d0406dec_deepseek_1762352985"
TRADER04_DIR="decision_logs/hyperliquid_a5e99c3c-bef8-4b5b-a19a-01328b593128_deepseek_1762351525"

echo "════════════════════════════════════════════════════════════"
echo "检查 Trader 02 和 Trader 04 最近两个 cycle 的市场数据更新"
echo "════════════════════════════════════════════════════════════"
echo ""

# 获取Trader 02最近两个cycle的日志文件
echo "📊 Trader 02 (26f0a132...)"
echo "────────────────────────────────────────────────────────────"
TRADER02_FILES=$(ls -t ${TRADER02_DIR}/decision_*.json 2>/dev/null | head -2)
if [ -z "$TRADER02_FILES" ]; then
    echo "❌ 未找到 Trader 02 的日志文件"
else
    CYCLE1_FILE=$(echo "$TRADER02_FILES" | tail -1)
    CYCLE2_FILE=$(echo "$TRADER02_FILES" | head -1)
    
    echo "Cycle 1: $(basename $CYCLE1_FILE)"
    echo "Cycle 2: $(basename $CYCLE2_FILE)"
    echo ""
    
    # 提取BTC价格
    CYCLE1_BTC=$(cat "$CYCLE1_FILE" | jq -r '.input_prompt' | grep -oP 'BTCUSDT.*?current_price = \K[\d.]+' | head -1)
    CYCLE2_BTC=$(cat "$CYCLE2_FILE" | jq -r '.input_prompt' | grep -oP 'BTCUSDT.*?current_price = \K[\d.]+' | head -1)
    
    # 提取ETH价格
    CYCLE1_ETH=$(cat "$CYCLE1_FILE" | jq -r '.input_prompt' | grep -oP 'ETHUSDT.*?current_price = \K[\d.]+' | head -1)
    CYCLE2_ETH=$(cat "$CYCLE2_FILE" | jq -r '.input_prompt' | grep -oP 'ETHUSDT.*?current_price = \K[\d.]+' | head -1)
    
    echo "BTCUSDT 价格:"
    echo "  Cycle 1: $CYCLE1_BTC"
    echo "  Cycle 2: $CYCLE2_BTC"
    if [ "$CYCLE1_BTC" != "$CYCLE2_BTC" ] && [ -n "$CYCLE1_BTC" ] && [ -n "$CYCLE2_BTC" ]; then
        DIFF=$(echo "$CYCLE2_BTC - $CYCLE1_BTC" | bc)
        echo "  ✅ 价格已更新 (差异: $DIFF)"
    elif [ "$CYCLE1_BTC" == "$CYCLE2_BTC" ] && [ -n "$CYCLE1_BTC" ]; then
        echo "  ⚠️  价格未更新 (可能相同或数据未刷新)"
    else
        echo "  ⚠️  无法提取价格"
    fi
    
    echo ""
    echo "ETHUSDT 价格:"
    echo "  Cycle 1: $CYCLE1_ETH"
    echo "  Cycle 2: $CYCLE2_ETH"
    if [ "$CYCLE1_ETH" != "$CYCLE2_ETH" ] && [ -n "$CYCLE1_ETH" ] && [ -n "$CYCLE2_ETH" ]; then
        DIFF=$(echo "$CYCLE2_ETH - $CYCLE1_ETH" | bc)
        echo "  ✅ 价格已更新 (差异: $DIFF)"
    elif [ "$CYCLE1_ETH" == "$CYCLE2_ETH" ] && [ -n "$CYCLE1_ETH" ]; then
        echo "  ⚠️  价格未更新 (可能相同或数据未刷新)"
    else
        echo "  ⚠️  无法提取价格"
    fi
fi

echo ""
echo "────────────────────────────────────────────────────────────"
echo ""

# 获取Trader 04最近两个cycle的日志文件
echo "📊 Trader 04 (a5e99c3c...)"
echo "────────────────────────────────────────────────────────────"
TRADER04_FILES=$(ls -t ${TRADER04_DIR}/decision_*.json 2>/dev/null | head -2)
if [ -z "$TRADER04_FILES" ]; then
    echo "❌ 未找到 Trader 04 的日志文件"
else
    CYCLE1_FILE=$(echo "$TRADER04_FILES" | tail -1)
    CYCLE2_FILE=$(echo "$TRADER04_FILES" | head -1)
    
    echo "Cycle 1: $(basename $CYCLE1_FILE)"
    echo "Cycle 2: $(basename $CYCLE2_FILE)"
    echo ""
    
    # 提取BTC价格
    CYCLE1_BTC=$(cat "$CYCLE1_FILE" | jq -r '.input_prompt' | grep -oP 'BTCUSDT.*?current_price = \K[\d.]+' | head -1)
    CYCLE2_BTC=$(cat "$CYCLE2_FILE" | jq -r '.input_prompt' | grep -oP 'BTCUSDT.*?current_price = \K[\d.]+' | head -1)
    
    # 提取ETH价格
    CYCLE1_ETH=$(cat "$CYCLE1_FILE" | jq -r '.input_prompt' | grep -oP 'ETHUSDT.*?current_price = \K[\d.]+' | head -1)
    CYCLE2_ETH=$(cat "$CYCLE2_FILE" | jq -r '.input_prompt' | grep -oP 'ETHUSDT.*?current_price = \K[\d.]+' | head -1)
    
    echo "BTCUSDT 价格:"
    echo "  Cycle 1: $CYCLE1_BTC"
    echo "  Cycle 2: $CYCLE2_BTC"
    if [ "$CYCLE1_BTC" != "$CYCLE2_BTC" ] && [ -n "$CYCLE1_BTC" ] && [ -n "$CYCLE2_BTC" ]; then
        DIFF=$(echo "$CYCLE2_BTC - $CYCLE1_BTC" | bc 2>/dev/null || echo "计算失败")
        echo "  ✅ 价格已更新 (差异: $DIFF)"
    elif [ "$CYCLE1_BTC" == "$CYCLE2_BTC" ] && [ -n "$CYCLE1_BTC" ]; then
        echo "  ⚠️  价格未更新 (可能相同或数据未刷新)"
    else
        echo "  ⚠️  无法提取价格"
    fi
    
    echo ""
    echo "ETHUSDT 价格:"
    echo "  Cycle 1: $CYCLE1_ETH"
    echo "  Cycle 2: $CYCLE2_ETH"
    if [ "$CYCLE1_ETH" != "$CYCLE2_ETH" ] && [ -n "$CYCLE1_ETH" ] && [ -n "$CYCLE2_ETH" ]; then
        DIFF=$(echo "$CYCLE2_ETH - $CYCLE1_ETH" | bc 2>/dev/null || echo "计算失败")
        echo "  ✅ 价格已更新 (差异: $DIFF)"
    elif [ "$CYCLE1_ETH" == "$CYCLE2_ETH" ] && [ -n "$CYCLE1_ETH" ]; then
        echo "  ⚠️  价格未更新 (可能相同或数据未刷新)"
    else
        echo "  ⚠️  无法提取价格"
    fi
fi

echo ""
echo "════════════════════════════════════════════════════════════"
echo ""


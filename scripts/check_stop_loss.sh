#!/bin/bash

echo "=================================================================================="
echo "最近15个Cycle的止损设置验证报告"
echo "=================================================================================="
echo ""

log_dir="/app/decision_logs/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728"
files=$(ls -t "$log_dir"/*.json 2>/dev/null | head -15)

total_decisions=0
valid_sl=0
invalid_sl=0

for file in $files; do
    filename=$(basename "$file")
    
    # 提取开仓决策
    decisions=$(cat "$file" | jq -r '.decisions[]? | select(.action == "OPEN_LONG" or .action == "OPEN_SHORT")' 2>/dev/null)
    
    if [ -n "$decisions" ]; then
        symbol=$(echo "$decisions" | jq -r '.symbol')
        action=$(echo "$decisions" | jq -r '.action')
        entry=$(echo "$decisions" | jq -r '.entry_price')
        sl=$(echo "$decisions" | jq -r '.stop_loss')
        tp=$(echo "$decisions" | jq -r '.take_profit')
        leverage=$(echo "$decisions" | jq -r '.leverage')
        timestamp=$(cat "$file" | jq -r '.timestamp')
        
        # 确定最小止损要求
        if echo "$symbol" | grep -qiE "(BTC|ETH)"; then
            min_sl=0.4
            asset_type="BTC/ETH"
        elif echo "$symbol" | grep -qiE "(SOL|BNB)"; then
            min_sl=0.6
            asset_type="大市值山寨币"
        else
            min_sl=0.8
            asset_type="其他山寨币"
        fi
        
        # 计算止损距离
        if [ "$action" = "OPEN_LONG" ]; then
            sl_distance=$(awk "BEGIN {printf \"%.4f\", (($entry - $sl) / $entry) * 100}")
        else
            sl_distance=$(awk "BEGIN {printf \"%.4f\", (($sl - $entry) / $entry) * 100}")
        fi
        
        # 验证
        status="✅ 符合要求"
        if (( $(echo "$sl_distance < $min_sl" | bc -l 2>/dev/null || echo "1") )); then
            status="❌ 不符合要求 (需要≥${min_sl}%)"
            invalid_sl=$((invalid_sl + 1))
        else
            valid_sl=$((valid_sl + 1))
        fi
        
        total_decisions=$((total_decisions + 1))
        
        echo "文件: $filename"
        echo "时间: $timestamp"
        echo "交易对: $symbol | 方向: $action | 杠杆: ${leverage}x"
        echo "入场价: $entry | 止损价: $sl | 止盈价: $tp"
        echo "止损距离: ${sl_distance}% | 资产类型: $asset_type | 要求: ≥${min_sl}%"
        echo "状态: $status"
        echo "--------------------------------------------------------------------------------"
        echo ""
    fi
done

echo ""
echo "=================================================================================="
echo "总结"
echo "=================================================================================="
echo "总开仓决策数: $total_decisions"
echo "符合止损要求的决策: $valid_sl"
echo "不符合止损要求的决策: $invalid_sl"

if [ $total_decisions -eq 0 ]; then
    echo ""
    echo "⚠️  最近15个cycle中没有开仓决策"
elif [ $invalid_sl -eq 0 ]; then
    echo ""
    echo "✅ 所有开仓决策的止损设置都符合要求！"
else
    echo ""
    echo "⚠️  发现 $invalid_sl 个决策的止损设置不符合要求"
fi


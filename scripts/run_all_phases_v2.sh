#!/bin/bash

# 运行所有Phase分析，使用v2文件名保存结果
# 参数：trader_id log_dir db_path start_cycle end_cycle

TRADER_ID=${1:-"hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1762598543"}
LOG_DIR=${2:-"decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1762598543"}
DB_PATH=${3:-"config.db.test"}
START_CYCLE=${4:-400}
END_CYCLE=${5:-1892}

echo "=== 运行Phase 1-5分析 (v2版本) ==="
echo "Trader ID: $TRADER_ID"
echo "Log目录: $LOG_DIR"
echo "数据库: $DB_PATH"
echo "Cycle范围: $START_CYCLE - $END_CYCLE"
echo ""

# Phase 1
echo "📊 运行Phase 1..."
go run scripts/analyze_negative_profit_phase1.go "$TRADER_ID" "$LOG_DIR" "$DB_PATH" $START_CYCLE $END_CYCLE
if [ $? -eq 0 ]; then
    mv phase1_result_${START_CYCLE}_${END_CYCLE}.json phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json
    mv phase1_report_${START_CYCLE}_${END_CYCLE}.md phase1_report_${START_CYCLE}_${END_CYCLE}_v2.md
    echo "✅ Phase 1完成，文件已重命名为v2版本"
else
    echo "❌ Phase 1失败"
    exit 1
fi
echo ""

# Phase 2
echo "📊 运行Phase 2..."
go run scripts/analyze_negative_profit_phase2.go phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json
if [ $? -eq 0 ]; then
    mv phase2_result_${START_CYCLE}_${END_CYCLE}.json phase2_result_${START_CYCLE}_${END_CYCLE}_v2.json
    mv phase2_report_${START_CYCLE}_${END_CYCLE}.md phase2_report_${START_CYCLE}_${END_CYCLE}_v2.md
    echo "✅ Phase 2完成，文件已重命名为v2版本"
else
    echo "❌ Phase 2失败"
    exit 1
fi
echo ""

# Phase 3
echo "📊 运行Phase 3..."
go run scripts/analyze_negative_profit_phase3.go phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json phase2_result_${START_CYCLE}_${END_CYCLE}_v2.json prompts/lean_optimized.txt
if [ $? -eq 0 ]; then
    mv phase3_result_${START_CYCLE}_${END_CYCLE}.json phase3_result_${START_CYCLE}_${END_CYCLE}_v2.json
    mv phase3_report_${START_CYCLE}_${END_CYCLE}.md phase3_report_${START_CYCLE}_${END_CYCLE}_v2.md
    echo "✅ Phase 3完成，文件已重命名为v2版本"
else
    echo "❌ Phase 3失败"
    exit 1
fi
echo ""

# Phase 4
echo "📊 运行Phase 4..."
go run scripts/analyze_negative_profit_phase4.go phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json phase3_result_${START_CYCLE}_${END_CYCLE}_v2.json prompts/lean_optimized.txt
if [ $? -eq 0 ]; then
    mv phase4_result_${START_CYCLE}_${END_CYCLE}.json phase4_result_${START_CYCLE}_${END_CYCLE}_v2.json
    mv phase4_report_${START_CYCLE}_${END_CYCLE}.md phase4_report_${START_CYCLE}_${END_CYCLE}_v2.md
    echo "✅ Phase 4完成，文件已重命名为v2版本"
else
    echo "❌ Phase 4失败"
    exit 1
fi
echo ""

# Phase 5
echo "📊 运行Phase 5..."
go run scripts/analyze_negative_profit_phase5.go phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json phase3_result_${START_CYCLE}_${END_CYCLE}_v2.json phase4_result_${START_CYCLE}_${END_CYCLE}_v2.json prompts/lean_optimized.txt
if [ $? -eq 0 ]; then
    mv phase5_result_${START_CYCLE}_${END_CYCLE}.json phase5_result_${START_CYCLE}_${END_CYCLE}_v2.json
    mv phase5_report_${START_CYCLE}_${END_CYCLE}.md phase5_report_${START_CYCLE}_${END_CYCLE}_v2.md
    mv prompts/lean_optimized_corrected_${START_CYCLE}_${END_CYCLE}.txt prompts/lean_optimized_corrected_${START_CYCLE}_${END_CYCLE}_v2.txt
    echo "✅ Phase 5完成，文件已重命名为v2版本"
else
    echo "❌ Phase 5失败"
    exit 1
fi
echo ""

echo "=== 所有Phase分析完成 ==="
echo ""
echo "生成的文件："
echo "- phase1_result_${START_CYCLE}_${END_CYCLE}_v2.json"
echo "- phase1_report_${START_CYCLE}_${END_CYCLE}_v2.md"
echo "- phase2_result_${START_CYCLE}_${END_CYCLE}_v2.json"
echo "- phase2_report_${START_CYCLE}_${END_CYCLE}_v2.md"
echo "- phase3_result_${START_CYCLE}_${END_CYCLE}_v2.json"
echo "- phase3_report_${START_CYCLE}_${END_CYCLE}_v2.md"
echo "- phase4_result_${START_CYCLE}_${END_CYCLE}_v2.json"
echo "- phase4_report_${START_CYCLE}_${END_CYCLE}_v2.md"
echo "- phase5_result_${START_CYCLE}_${END_CYCLE}_v2.json"
echo "- phase5_report_${START_CYCLE}_${END_CYCLE}_v2.md"
echo "- prompts/lean_optimized_corrected_${START_CYCLE}_${END_CYCLE}_v2.txt"


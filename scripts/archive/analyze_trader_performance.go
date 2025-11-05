package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nofx/logger"
)

func main() {
	// 三个主要的 trader 目录
	traders := []string{
		"decision_logs/hyperliquid_a5e99c3c-bef8-4b5b-a19a-01328b593128_deepseek_1762195291",
		"decision_logs/hyperliquid_26f0a132-0026-4516-b5ef-59a2d0406dec_deepseek_1762221829",
		"decision_logs/hyperliquid_d53550af-05cd-494d-8d6e-18fc940d15c9_deepseek_1762221685",
	}

	// 使用足够大的窗口来查看所有历史数据
	lookbackCycles := 1000

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("📊 Trader 性能分析")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, traderDir := range traders {
		// 检查目录是否存在
		if _, err := os.Stat(traderDir); os.IsNotExist(err) {
			fmt.Printf("⚠️  Trader %d: 目录不存在: %s\n\n", i+1, traderDir)
			continue
		}

		// 创建 DecisionLogger
		decisionLogger := logger.NewDecisionLogger(traderDir)

		// 分析性能
		performance, err := decisionLogger.AnalyzePerformance(lookbackCycles)
		if err != nil {
			fmt.Printf("❌ Trader %d: 分析失败: %v\n\n", i+1, err)
			continue
		}

		// 获取 trader 名称
		traderName := filepath.Base(traderDir)
		fmt.Printf("📈 Trader %d: %s\n", i+1, traderName)
		fmt.Println(strings.Repeat("-", 80))

		// 打印主要指标
		fmt.Printf("总交易数:     %d\n", performance.TotalTrades)
		fmt.Printf("盈利交易:     %d\n", performance.WinningTrades)
		fmt.Printf("亏损交易:     %d\n", performance.LosingTrades)
		fmt.Printf("胜率:         %.2f%%\n", performance.WinRate)
		fmt.Printf("平均盈利:     %.4f USDT\n", performance.AvgWin)
		fmt.Printf("平均亏损:     %.4f USDT\n", performance.AvgLoss)
		fmt.Printf("盈亏比:       %.4f\n", performance.ProfitFactor)
		fmt.Printf("夏普比率:     %.4f\n", performance.SharpeRatio)
		fmt.Printf("最佳币种:     %s\n", performance.BestSymbol)
		fmt.Printf("最差币种:     %s\n", performance.WorstSymbol)

		// 打印币种统计
		if len(performance.SymbolStats) > 0 {
			fmt.Println("\n币种统计:")
			for symbol, stats := range performance.SymbolStats {
				fmt.Printf("  %s:\n", symbol)
				fmt.Printf("    交易数: %d, 胜率: %.2f%%, 总盈亏: %.4f USDT, 平均盈亏: %.4f USDT\n",
					stats.TotalTrades, stats.WinRate, stats.TotalPnL, stats.AvgPnL)
			}
		}

		// 打印最近几笔交易
		if len(performance.RecentTrades) > 0 {
			fmt.Printf("\n最近 %d 笔交易:\n", len(performance.RecentTrades))
			for j, trade := range performance.RecentTrades {
				if j >= 5 { // 只显示最近5笔
					break
				}
				pnlSign := "+"
				if trade.PnL < 0 {
					pnlSign = ""
				}
				fmt.Printf("  %d. %s %s %.8f: %s%.4f USDT (%.2f%%)\n",
					j+1, trade.Symbol, trade.Side, trade.Quantity, pnlSign, trade.PnL, trade.PnLPct)
			}
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
	}

	// 输出 JSON 格式（便于后续处理）
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		results := make(map[string]interface{})
		for _, traderDir := range traders {
			if _, err := os.Stat(traderDir); os.IsNotExist(err) {
				continue
			}
			decisionLogger := logger.NewDecisionLogger(traderDir)
			performance, err := decisionLogger.AnalyzePerformance(lookbackCycles)
			if err != nil {
				continue
			}
			traderName := filepath.Base(traderDir)
			results[traderName] = performance
		}
		jsonData, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(jsonData))
	}
}

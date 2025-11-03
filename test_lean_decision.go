package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"nofx/decision"
	"nofx/market"
	"nofx/mcp"
	"nofx/trader"
	"os"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 解析命令行参数
	useRealData := flag.Bool("real", false, "Use real Hyperliquid account data (positions and balance)")
	flag.Parse()
	
	log.Println("=== LEAN Prompt 决策测试脚本 ===")
	if *useRealData {
		log.Println("🔴 使用真实 Hyperliquid 账户数据模式")
	} else {
		log.Println("📝 使用模拟数据模式（默认）")
		log.Println("💡 提示: 使用 --real 参数可启用真实账户数据")
	}

	// 1. 从配置文件读取参数
	configFile := "config.json"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var config struct {
		Traders []struct {
			ID                  string  `json:"id"`
			DeepSeekKey         string  `json:"deepseek_key"`
			InitialBalance      float64 `json:"initial_balance"`
			HyperliquidPrivateKey string `json:"hyperliquid_private_key,omitempty"`
			HyperliquidWalletAddr string `json:"hyperliquid_wallet_addr,omitempty"`
			HyperliquidTestnet   bool   `json:"hyperliquid_testnet,omitempty"`
		} `json:"traders"`
		Leverage struct {
			BTCETHLeverage  int `json:"btc_eth_leverage"`
			AltcoinLeverage int `json:"altcoin_leverage"`
		} `json:"leverage"`
		DefaultCoins []string `json:"default_coins"`
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 查找 hyperliquid_deepseek 配置
	var deepseekKey string
	var initialBalance float64
	var hyperliquidPrivateKey string
	var hyperliquidWalletAddr string
	var hyperliquidTestnet bool
	found := false
	for _, trader := range config.Traders {
		if trader.ID == "hyperliquid_deepseek" {
			deepseekKey = trader.DeepSeekKey
			initialBalance = trader.InitialBalance
			hyperliquidPrivateKey = trader.HyperliquidPrivateKey
			hyperliquidWalletAddr = trader.HyperliquidWalletAddr
			hyperliquidTestnet = trader.HyperliquidTestnet
			found = true
			break
		}
	}

	if !found || deepseekKey == "" {
		log.Fatalf("未找到 hyperliquid_deepseek 配置或 deepseek_key 为空")
	}

	log.Printf("✓ 从配置加载: DeepSeek API Key: %s...%s", deepseekKey[:4], deepseekKey[len(deepseekKey)-4:])
	log.Printf("✓ 初始余额: %.2f", initialBalance)

	// 2. 初始化市场监控器（需要先初始化才能获取市场数据）
	log.Println("\n📊 初始化市场监控器...")
	monitor := market.NewWSMonitor(10)
	
	// 使用默认币种列表初始化
	defaultCoins := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}
	if len(config.DefaultCoins) > 0 {
		defaultCoins = config.DefaultCoins
	}
	
	if err := monitor.Initialize(defaultCoins); err != nil {
		log.Fatalf("初始化市场监控器失败: %v", err)
	}

	// 等待市场数据加载
	log.Println("⏳ 等待市场数据加载...")
	time.Sleep(3 * time.Second)

	// 3. 初始化MCP客户端并配置DeepSeek
	log.Println("\n🤖 初始化AI客户端...")
	mcpClient := mcp.New()
	mcpClient.SetDeepSeekAPIKey(deepseekKey, "", "")
	log.Println("✓ DeepSeek客户端已配置")

	// 4. 构建交易上下文（Context）
	log.Println("\n📝 构建交易上下文...")
	
	var ctx *decision.Context
	
	if *useRealData {
		// 使用真实 Hyperliquid 数据
		if hyperliquidPrivateKey == "" || hyperliquidWalletAddr == "" {
			log.Fatalf("❌ 使用 --real 模式需要配置 hyperliquid_private_key 和 hyperliquid_wallet_addr")
		}
		
		log.Println("🔄 正在从 Hyperliquid 获取真实账户数据...")
		
		// 初始化 Hyperliquid 交易器
		hlTrader, err := trader.NewHyperliquidTrader(
			hyperliquidPrivateKey,
			hyperliquidWalletAddr,
			hyperliquidTestnet,
		)
		if err != nil {
			log.Fatalf("❌ 初始化 Hyperliquid 交易器失败: %v", err)
		}
		
		// 获取真实账户余额
		balance, err := hlTrader.GetBalance()
		if err != nil {
			log.Fatalf("❌ 获取账户余额失败: %v", err)
		}
		
		totalWalletBalance := 0.0
		totalUnrealizedProfit := 0.0
		availableBalance := 0.0
		
		if wallet, ok := balance["totalWalletBalance"].(float64); ok {
			totalWalletBalance = wallet
		}
		if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
			totalUnrealizedProfit = unrealized
		}
		if avail, ok := balance["availableBalance"].(float64); ok {
			availableBalance = avail
		}
		
		totalEquity := totalWalletBalance + totalUnrealizedProfit
		totalPnL := totalEquity - initialBalance
		totalPnLPct := 0.0
		if initialBalance > 0 {
			totalPnLPct = (totalPnL / initialBalance) * 100
		}
		
		log.Printf("✓ 真实账户数据: 净值=%.2f, 钱包=%.2f, 未实现盈亏=%.2f, 可用=%.2f",
			totalEquity, totalWalletBalance, totalUnrealizedProfit, availableBalance)
		
		// 获取真实持仓
		positions, err := hlTrader.GetPositions()
		if err != nil {
			log.Fatalf("❌ 获取持仓失败: %v", err)
		}
		
		var positionInfos []decision.PositionInfo
		totalMarginUsed := 0.0
		
		log.Printf("✓ 找到 %d 个真实持仓", len(positions))
		for _, pos := range positions {
			symbol := pos["symbol"].(string)
			side := pos["side"].(string)
			entryPrice := pos["entryPrice"].(float64)
			markPrice := pos["markPrice"].(float64)
			quantity := pos["positionAmt"].(float64)
			if quantity < 0 {
				quantity = -quantity
			}
			unrealizedPnl := pos["unRealizedProfit"].(float64)
			liquidationPrice := pos["liquidationPrice"].(float64)
			
			// 计算盈亏百分比
			pnlPct := 0.0
			if side == "long" {
				pnlPct = ((markPrice - entryPrice) / entryPrice) * 100
			} else {
				pnlPct = ((entryPrice - markPrice) / entryPrice) * 100
			}
			
			// 计算占用保证金
			leverage := 10
			if lev, ok := pos["leverage"].(float64); ok {
				leverage = int(lev)
			}
			marginUsed := (quantity * markPrice) / float64(leverage)
			totalMarginUsed += marginUsed
			
			log.Printf("  - %s %s: 数量=%.4f, 入场价=%.4f, 当前价=%.4f, 盈亏=%.2f (%.2f%%)",
				symbol, side, quantity, entryPrice, markPrice, unrealizedPnl, pnlPct)
			
			positionInfos = append(positionInfos, decision.PositionInfo{
				Symbol:           symbol,
				Side:             side,
				EntryPrice:       entryPrice,
				MarkPrice:        markPrice,
				Quantity:         quantity,
				Leverage:         leverage,
				UnrealizedPnL:    unrealizedPnl,
				UnrealizedPnLPct: pnlPct,
				LiquidationPrice: liquidationPrice,
				MarginUsed:       marginUsed,
				UpdateTime:       time.Now().UnixMilli(),
			})
		}
		
		marginUsedPct := 0.0
		if totalEquity > 0 {
			marginUsedPct = (totalMarginUsed / totalEquity) * 100
		}
		
		ctx = &decision.Context{
			CurrentTime:     time.Now().Format("2006-01-02 15:04:05"),
			RuntimeMinutes:  0,
			CallCount:       1,
			Account: decision.AccountInfo{
				TotalEquity:      totalEquity,
				AvailableBalance: availableBalance,
				TotalPnL:         totalPnL,
				TotalPnLPct:      totalPnLPct,
				MarginUsed:       totalMarginUsed,
				MarginUsedPct:    marginUsedPct,
				PositionCount:    len(positionInfos),
			},
			Positions:      positionInfos,
			CandidateCoins: []decision.CandidateCoin{},
			MarketDataMap:  make(map[string]*market.Data),
			OITopDataMap:   make(map[string]*decision.OITopData),
			Performance:    nil,
			BTCETHLeverage:  config.Leverage.BTCETHLeverage,
			AltcoinLeverage: config.Leverage.AltcoinLeverage,
		}
		
		log.Printf("✓ 交易上下文构建完成: 持仓数=%d, 保证金使用率=%.2f%%", len(positionInfos), marginUsedPct)
	} else {
		// 使用模拟数据（原有逻辑）
		ctx = &decision.Context{
			CurrentTime:     time.Now().Format("2006-01-02 15:04:05"),
			RuntimeMinutes:  0,
			CallCount:       1,
			Account: decision.AccountInfo{
				TotalEquity:      initialBalance,
				AvailableBalance: initialBalance,
				TotalPnL:         0.0,
				TotalPnLPct:      0.0,
				MarginUsed:       0.0,
				MarginUsedPct:    0.0,
				PositionCount:   0,
			},
			Positions:       []decision.PositionInfo{},
			CandidateCoins:  []decision.CandidateCoin{},
			MarketDataMap:   make(map[string]*market.Data),
			OITopDataMap:    make(map[string]*decision.OITopData),
			Performance:    nil,
			BTCETHLeverage:  config.Leverage.BTCETHLeverage,
			AltcoinLeverage: config.Leverage.AltcoinLeverage,
		}
	}

	// 5. 设置候选币种（使用默认币种）
	log.Println("📋 设置候选币种...")
	for _, coin := range defaultCoins {
		ctx.CandidateCoins = append(ctx.CandidateCoins, decision.CandidateCoin{
			Symbol:  coin,
			Sources:  []string{"default"},
		})
		log.Printf("  - %s", coin)
	}

	// 6. 发起决策（使用 lean.txt 作为主 prompt）
	log.Println("\n🚀 发起AI决策请求（使用 lean.txt prompt）...")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 使用 GetFullDecisionWithCustomPrompt
	// templateName: "lean" 表示使用 prompts/lean.txt
	// overrideBase: false 表示不覆盖基础prompt，而是在基础上添加
	// customPrompt: "" 表示不使用自定义prompt
	fullDecision, err := decision.GetFullDecisionWithCustomPrompt(ctx, mcpClient, "", false, "lean")
	if err != nil {
		log.Fatalf("❌ 获取决策失败: %v", err)
	}

	// 7. 输出结果
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("✅ 决策完成！")
	log.Println("\n📄 系统提示词 (System Prompt):")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(fullDecision.SystemPrompt)
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	log.Println("\n📥 用户输入 (User Prompt):")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(fullDecision.UserPrompt)
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	log.Println("\n🤔 AI思维链 (CoT Trace):")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(fullDecision.CoTTrace)
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	log.Println("\n💡 交易决策 (Decisions):")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(fullDecision.Decisions) == 0 {
		log.Println("  (无决策)")
	} else {
		for i, d := range fullDecision.Decisions {
			log.Printf("\n决策 #%d:", i+1)
			log.Printf("  币种: %s", d.Symbol)
			log.Printf("  动作: %s", d.Action)
			
			// 验证 action 是否符合 lean.txt 的要求
			validActions := map[string]bool{
				"open_long":   true,
				"open_short":  true,
				"close_long":  true,
				"close_short": true,
				"hold":        true,
				"wait":        true,
			}
			if validActions[d.Action] {
				log.Printf("  ✓ Action 有效: %s", d.Action)
			} else {
				log.Printf("  ❌ Action 无效: %s (应为 open_long/open_short/close_long/close_short/hold/wait)", d.Action)
			}
			
			if d.Leverage > 0 {
				log.Printf("  杠杆: %dx", d.Leverage)
				
				// 验证杠杆是否在配置范围内
				maxLeverage := config.Leverage.AltcoinLeverage
				if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
					maxLeverage = config.Leverage.BTCETHLeverage
				}
				if d.Leverage <= maxLeverage {
					log.Printf("  ✓ 杠杆在配置范围内 (≤%dx)", maxLeverage)
				} else {
					log.Printf("  ❌ 杠杆超出配置范围 (%dx > %dx)", d.Leverage, maxLeverage)
				}
			}
			if d.PositionSizeUSD > 0 {
				log.Printf("  仓位大小: %.2f USD", d.PositionSizeUSD)
			}
			if d.StopLoss > 0 {
				log.Printf("  止损: %.4f", d.StopLoss)
			}
			if d.TakeProfit > 0 {
				log.Printf("  止盈: %.4f", d.TakeProfit)
				
				// 验证风险回报比
				if d.StopLoss > 0 {
					var entryPrice, risk, reward, riskRewardRatio float64
					if d.Action == "open_long" {
						entryPrice = d.StopLoss + (d.TakeProfit-d.StopLoss)*0.2
						risk = entryPrice - d.StopLoss
						reward = d.TakeProfit - entryPrice
					} else if d.Action == "open_short" {
						entryPrice = d.StopLoss - (d.StopLoss-d.TakeProfit)*0.2
						risk = d.StopLoss - entryPrice
						reward = entryPrice - d.TakeProfit
					}
					if risk > 0 {
						riskRewardRatio = reward / risk
						log.Printf("  风险回报比: %.2f:1", riskRewardRatio)
						if riskRewardRatio >= 3.0 {
							log.Printf("  ✓ 风险回报比符合要求 (≥3:1)")
						} else {
							log.Printf("  ❌ 风险回报比不符合要求 (%.2f:1 < 3:1)", riskRewardRatio)
						}
					}
				}
			}
			if d.Confidence > 0 {
				log.Printf("  信心度: %d/100", d.Confidence)
				
				// 验证信心度是否符合 lean.txt 的要求 (70-100)
				if d.Confidence >= 70 && d.Confidence <= 100 {
					log.Printf("  ✓ 信心度在有效范围内 (70-100)")
				} else {
					log.Printf("  ❌ 信心度不在有效范围内 (%d, 应为 70-100)", d.Confidence)
				}
				
				// 检查是否低于阈值（开仓应≥70）
				if (d.Action == "open_long" || d.Action == "open_short") && d.Confidence < 70 {
					log.Printf("  ⚠️  警告: 开仓信心度低于70，按lean.txt要求不应开仓")
				}
			}
			if d.RiskUSD > 0 {
				log.Printf("  风险金额: %.2f USD (%.2f%% 账户)", d.RiskUSD, (d.RiskUSD/initialBalance)*100)
				
				// 验证风险金额是否符合 lean.txt 的要求 (5-8%)
				riskPct := (d.RiskUSD / initialBalance) * 100
				if riskPct >= 5.0 && riskPct <= 8.0 {
					log.Printf("  ✓ 风险金额在建议范围内 (5-8%%)")
				} else {
					log.Printf("  ⚠️  风险金额不在建议范围内 (%.2f%%, 建议 5-8%%)", riskPct)
				}
			}
			if d.Reasoning != "" {
				log.Printf("  理由: %s", d.Reasoning)
			}
		}
	}
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 8. 输出JSON格式的决策（便于进一步处理）
	log.Println("\n📊 JSON格式决策:")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	jsonData, err := json.MarshalIndent(fullDecision.Decisions, "", "  ")
	if err != nil {
		log.Printf("❌ JSON序列化失败: %v", err)
	} else {
		fmt.Println(string(jsonData))
	}
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 9. 验证总结
	log.Println("\n📋 兼容性验证总结:")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	allValid := true
	
	for i, d := range fullDecision.Decisions {
		valid := true
		
		// 验证 action
		validActions := map[string]bool{
			"open_long": true, "open_short": true, "close_long": true,
			"close_short": true, "hold": true, "wait": true,
		}
		if !validActions[d.Action] {
			log.Printf("决策 #%d: ❌ Action '%s' 无效", i+1, d.Action)
			valid = false
			allValid = false
		}
		
		// 验证 confidence（开仓必须≥70）
		if (d.Action == "open_long" || d.Action == "open_short") {
			if d.Confidence < 70 {
				log.Printf("决策 #%d: ❌ 开仓信心度 %d < 70 (lean.txt要求)", i+1, d.Confidence)
				valid = false
				allValid = false
			}
		}
		
		// 验证杠杆（不能超过配置）
		if d.Leverage > 0 {
			maxLeverage := config.Leverage.AltcoinLeverage
			if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
				maxLeverage = config.Leverage.BTCETHLeverage
			}
			if d.Leverage > maxLeverage {
				log.Printf("决策 #%d: ❌ 杠杆 %dx > 配置上限 %dx", i+1, d.Leverage, maxLeverage)
				valid = false
				allValid = false
			}
		}
		
		if valid {
			log.Printf("决策 #%d: ✅ 所有验证通过", i+1)
		}
	}
	
	if allValid && len(fullDecision.Decisions) > 0 {
		log.Println("\n✅ 所有决策验证通过！lean.txt prompt 与 engine 兼容")
	} else if len(fullDecision.Decisions) == 0 {
		log.Println("\n⚠️  无决策输出（可能是 hold 或 wait）")
	} else {
		log.Println("\n❌ 部分决策验证失败，请检查上述错误")
	}
	log.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	log.Println("\n✅ 测试完成！")
}


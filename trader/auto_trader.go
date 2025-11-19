package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"nofx/decision"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/pool"
	"nofx/trade_analytics"
	"nofx/trade_history"
	"strconv"
	"strings"
	"time"
)

// AutoTraderConfig 自动交易配置（简化版 - AI全权决策）
type AutoTraderConfig struct {
	// Trader标识
	ID      string // Trader唯一标识（用于日志目录等）
	Name    string // Trader显示名称
	AIModel string // AI模型: "qwen" 或 "deepseek"

	// 交易平台选择
	Exchange string // "binance", "hyperliquid" 或 "aster"

	// 币安API配置
	BinanceAPIKey    string
	BinanceSecretKey string

	// Hyperliquid配置
	HyperliquidPrivateKey string
	HyperliquidWalletAddr string
	HyperliquidTestnet    bool

	// Aster配置
	AsterUser       string // Aster主钱包地址
	AsterSigner     string // Aster API钱包地址
	AsterPrivateKey string // Aster API钱包私钥

	CoinPoolAPIURL string

	// AI配置
	UseQwen     bool
	DeepSeekKey string
	QwenKey     string

	// 自定义AI API配置
	CustomAPIURL    string
	CustomAPIKey    string
	CustomModelName string

	// 扫描配置
	ScanInterval time.Duration // 扫描间隔（建议3分钟）

	// 账户配置
	InitialBalance float64 // 初始金额（用于计算盈亏，需手动设置）

	// 杠杆配置
	BTCETHLeverage  int // BTC和ETH的杠杆倍数
	AltcoinLeverage int // 山寨币的杠杆倍数

	// 风险控制（仅作为提示，AI可自主决定）
	MaxDailyLoss    float64       // 最大日亏损百分比（提示）
	MaxDrawdown     float64       // 最大回撤百分比（提示）
	StopTradingTime time.Duration // 触发风控后暂停时长

	// 仓位模式
	IsCrossMargin bool // true=全仓模式, false=逐仓模式

	// 币种配置
	DefaultCoins []string // 默认币种列表（从数据库获取）
	TradingCoins []string // 实际交易币种列表

	// 系统提示词模板
	SystemPromptTemplate string // 系统提示词模板名称（如 "default", "aggressive"）

	// 历史决策配置
	HistoryDecisionCycles int // 历史决策周期数（0=禁用，默认2）
}

// AutoTrader 自动交易器
type AutoTrader struct {
	id                    string // Trader唯一标识
	name                  string // Trader显示名称
	aiModel               string // AI模型名称
	exchange              string // 交易平台名称
	config                AutoTraderConfig
	trader                Trader // 使用Trader接口（支持多平台）
	mcpClient             *mcp.Client
	decisionLogger        *logger.DecisionLogger   // 决策日志记录器
	tradeHistoryService   trade_history.Service   // 交易历史服务（可选）
	tradeHistoryEnabled   bool                    // 是否启用交易历史
	tradeAnalyticsService trade_analytics.Service // 交易分析服务（可选，新增）
	initialBalance        float64
	dailyPnL              float64
	customPrompt          string   // 自定义交易策略prompt
	overrideBasePrompt    bool     // 是否覆盖基础prompt
	systemPromptTemplate  string   // 系统提示词模板名称
	defaultCoins          []string // 默认币种列表（从数据库获取）
	tradingCoins          []string // 实际交易币种列表
	lastResetTime         time.Time
	stopUntil             time.Time
	isRunning             bool
	ctx                   context.Context         // 用于控制goroutine停止的context
	cancel                context.CancelFunc      // 用于取消context的函数
	startTime             time.Time               // 系统启动时间
	callCount             int                     // AI调用次数
	positionFirstSeenTime map[string]int64        // 持仓首次出现时间 (symbol_side -> timestamp毫秒)
	lastCyclePositions    []decision.PositionInfo // 上一个周期的持仓列表（用于检测自动触发的止盈止损）
	lastTradeTime         time.Time               // 最后一次交易时间（开仓或平仓）
	consecutiveWaitCycles int                     // 连续等待周期数（连续 wait 决策的次数）
}

// NewAutoTrader 创建自动交易器
func NewAutoTrader(config AutoTraderConfig, tradeHistoryService trade_history.Service, tradeAnalyticsService trade_analytics.Service) (*AutoTrader, error) {
	// 设置默认值
	if config.ID == "" {
		config.ID = "default_trader"
	}
	if config.Name == "" {
		config.Name = "Default Trader"
	}
	if config.AIModel == "" {
		if config.UseQwen {
			config.AIModel = "qwen"
		} else {
			config.AIModel = "deepseek"
		}
	}

	mcpClient := mcp.New()

	// 初始化AI
	if config.AIModel == "custom" {
		// 使用自定义API
		mcpClient.SetCustomAPI(config.CustomAPIURL, config.CustomAPIKey, config.CustomModelName)
		log.Printf("🤖 [%s] 使用自定义AI API: %s (模型: %s)", config.Name, config.CustomAPIURL, config.CustomModelName)
	} else if config.UseQwen || config.AIModel == "qwen" {
		// 使用Qwen (支持自定义URL和Model)
		mcpClient.SetQwenAPIKey(config.QwenKey, config.CustomAPIURL, config.CustomModelName)
		if config.CustomAPIURL != "" || config.CustomModelName != "" {
			log.Printf("🤖 [%s] 使用阿里云Qwen AI (自定义URL: %s, 模型: %s)", config.Name, config.CustomAPIURL, config.CustomModelName)
		} else {
			log.Printf("🤖 [%s] 使用阿里云Qwen AI", config.Name)
		}
	} else {
		// 默认使用DeepSeek (支持自定义URL和Model)
		mcpClient.SetDeepSeekAPIKey(config.DeepSeekKey, config.CustomAPIURL, config.CustomModelName)
		if config.CustomAPIURL != "" || config.CustomModelName != "" {
			log.Printf("🤖 [%s] 使用DeepSeek AI (自定义URL: %s, 模型: %s)", config.Name, config.CustomAPIURL, config.CustomModelName)
		} else {
			log.Printf("🤖 [%s] 使用DeepSeek AI", config.Name)
		}
	}

	// 初始化币种池API
	if config.CoinPoolAPIURL != "" {
		pool.SetCoinPoolAPI(config.CoinPoolAPIURL)
	}

	// 设置默认交易平台
	if config.Exchange == "" {
		config.Exchange = "binance"
	}

	// 根据配置创建对应的交易器
	var trader Trader
	var err error

	// 记录仓位模式（通用）
	marginModeStr := "全仓"
	if !config.IsCrossMargin {
		marginModeStr = "逐仓"
	}
	log.Printf("📊 [%s] 仓位模式: %s", config.Name, marginModeStr)

	switch config.Exchange {
	case "binance":
		log.Printf("🏦 [%s] 使用币安合约交易", config.Name)
		trader = NewFuturesTrader(config.BinanceAPIKey, config.BinanceSecretKey)
	case "hyperliquid":
		log.Printf("🏦 [%s] 使用Hyperliquid交易", config.Name)
		trader, err = NewHyperliquidTrader(config.HyperliquidPrivateKey, config.HyperliquidWalletAddr, config.HyperliquidTestnet)
		if err != nil {
			return nil, fmt.Errorf("初始化Hyperliquid交易器失败: %w", err)
		}
	case "aster":
		log.Printf("🏦 [%s] 使用Aster交易", config.Name)
		trader, err = NewAsterTrader(config.AsterUser, config.AsterSigner, config.AsterPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("初始化Aster交易器失败: %w", err)
		}
	default:
		return nil, fmt.Errorf("不支持的交易平台: %s", config.Exchange)
	}

	// 验证初始金额配置
	if config.InitialBalance <= 0 {
		return nil, fmt.Errorf("初始金额必须大于0，请在配置中设置InitialBalance")
	}

	// 初始化决策日志记录器（使用trader ID创建独立目录）
	logDir := fmt.Sprintf("decision_logs/%s", config.ID)
	decisionLogger := logger.NewDecisionLogger(logDir)

	// 设置默认系统提示词模板
	systemPromptTemplate := config.SystemPromptTemplate
	if systemPromptTemplate == "" {
		systemPromptTemplate = "default" // 默认使用 default 模板
	}

	// 设置交易历史服务
	tradeHistoryEnabled := tradeHistoryService != nil

	return &AutoTrader{
		id:                    config.ID,
		name:                  config.Name,
		aiModel:               config.AIModel,
		exchange:              config.Exchange,
		config:                config,
		trader:                trader,
		mcpClient:             mcpClient,
		decisionLogger:        decisionLogger,
		tradeHistoryService:   tradeHistoryService,
		tradeHistoryEnabled:   tradeHistoryEnabled,
		tradeAnalyticsService: tradeAnalyticsService, // 新增
		initialBalance:        config.InitialBalance,
		systemPromptTemplate:  systemPromptTemplate,
		defaultCoins:          config.DefaultCoins,
		tradingCoins:          config.TradingCoins,
		lastResetTime:         time.Now(),
		startTime:             time.Now(),
		callCount:             0,
		isRunning:             false,
		positionFirstSeenTime: make(map[string]int64),
		lastTradeTime:         time.Time{}, // 初始化为零值，表示从未交易
		consecutiveWaitCycles: 0,           // 初始化为0
	}, nil
}

// SetTradeAnalyticsService 设置交易分析服务（用于运行时更新）
func (at *AutoTrader) SetTradeAnalyticsService(service trade_analytics.Service) {
	at.tradeAnalyticsService = service
	log.Printf("🔧 [UPDATE] Trader '%s' 的 tradeAnalyticsService 已更新", at.id)
}

// Run 运行自动交易主循环
func (at *AutoTrader) Run() error {
	// 创建context，用于控制goroutine的停止
	at.ctx, at.cancel = context.WithCancel(context.Background())

	at.isRunning = true
	log.Println("🚀 AI驱动自动交易系统启动")
	log.Printf("💰 初始余额: %.2f USDT", at.initialBalance)
	log.Printf("⚙️  扫描间隔: %v", at.config.ScanInterval)
	log.Println("🤖 AI将全权决定杠杆、仓位大小、止损止盈等参数")

	// 启动交易历史定期同步（如果启用）
	if at.tradeHistoryEnabled && at.tradeHistoryService != nil && at.exchange == "hyperliquid" {
		// 获取同步间隔配置（默认10分钟）
		syncInterval := 10 * time.Minute
		// 这里可以从配置中读取，暂时使用默认值

		// 创建同步服务
		syncService := trade_history.NewSyncService(at.tradeHistoryService, syncInterval)

		// 获取HyperliquidTrader的exchange实例
		if hyperliquidTrader, ok := at.trader.(*HyperliquidTrader); ok {
			// 获取wallet地址（需要从HyperliquidTrader获取）
			walletAddr := hyperliquidTrader.walletAddr
			provider := trade_history.NewHyperliquidFillsProvider(
				hyperliquidTrader.exchange,
				at.ctx,
				walletAddr,
			)

			// 启动定期同步（在goroutine中运行）
			go syncService.Start(at.ctx, at.id, provider)
		}
	}

	ticker := time.NewTicker(at.config.ScanInterval)
	defer ticker.Stop()

	// 首次立即执行
	if err := at.runCycle(); err != nil {
		log.Printf("❌ 执行失败: %v", err)
	}

	for {
		select {
		case <-at.ctx.Done():
			// context被取消，立即退出
			log.Println("⏹ 自动交易系统停止（context已取消）")
			return nil
		case <-ticker.C:
			if !at.isRunning {
				// isRunning标志为false，退出
				return nil
			}
			if err := at.runCycle(); err != nil {
				log.Printf("❌ 执行失败: %v", err)
			}
		}
	}
}

// Stop 停止自动交易
func (at *AutoTrader) Stop() {
	at.isRunning = false
	// 取消context，立即停止goroutine
	if at.cancel != nil {
		at.cancel()
	}
	log.Println("⏹ 自动交易系统停止")
}

// runCycle 运行一个交易周期（使用AI全权决策）
func (at *AutoTrader) runCycle() error {
	at.callCount++

	log.Println("\n" + strings.Repeat("=", 70))
	log.Printf("⏰ %s - AI决策周期 #%d", time.Now().Format("2006-01-02 15:04:05"), at.callCount)
	log.Println(strings.Repeat("=", 70))

	// 创建决策记录
	record := &logger.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
	}

	// 1. 检查是否需要停止交易
	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		log.Printf("⏸ 风险控制：暂停交易中，剩余 %.0f 分钟", remaining.Minutes())
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("风险控制暂停中，剩余 %.0f 分钟", remaining.Minutes())
		at.decisionLogger.LogDecision(record)
		return nil
	}

	// 2. 重置日盈亏（每天重置）
	if time.Since(at.lastResetTime) > 24*time.Hour {
		at.dailyPnL = 0
		at.lastResetTime = time.Now()
		log.Println("📅 日盈亏已重置")
	}

	// 3. 收集交易上下文
	ctx, err := at.buildTradingContext()
	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("构建交易上下文失败: %v", err)
		at.decisionLogger.LogDecision(record)
		return fmt.Errorf("构建交易上下文失败: %w", err)
	}

	// 3.1 检测自动触发的止盈止损订单（通过持仓变化）
	// 比较上一个周期的持仓和当前持仓，找出消失的持仓
	if len(at.lastCyclePositions) > 0 {
		// 构建当前持仓的key集合（symbol_side）
		currentPositionKeys := make(map[string]bool)
		for _, pos := range ctx.Positions {
			posKey := pos.Symbol + "_" + pos.Side
			currentPositionKeys[posKey] = true
		}

		// 检测消失的持仓
		for _, lastPos := range at.lastCyclePositions {
			posKey := lastPos.Symbol + "_" + lastPos.Side
			if !currentPositionKeys[posKey] {
				// 持仓消失了，可能是自动触发的止盈止损订单
				// 尝试获取当前市场价格作为实际成交价的近似值
				actualClosePrice := lastPos.MarkPrice // 默认使用上一周期的标记价
				marketData, err := market.Get(lastPos.Symbol)
				if err == nil && marketData != nil {
					// 使用当前市场价格作为实际成交价的近似值（比上一周期的标记价更准确）
					actualClosePrice = marketData.CurrentPrice
				}

				// 判断是止损还是止盈（通过实际成交价与开仓价的对比）
				// 注意：只能通过价格方向推断，无法100%准确判断
				// 但比使用上一周期的标记价更准确
				wasStopLoss := false
				if lastPos.Side == "long" {
					// 多仓：如果实际成交价低于开仓价，可能是止损
					// 设置一个容差，避免价格微小波动导致的误判
					priceDiff := actualClosePrice - lastPos.EntryPrice
					if priceDiff < -0.001 { // 价格低于开仓价超过0.001，判断为止损
						wasStopLoss = true
					}
				} else {
					// 空仓：如果实际成交价高于开仓价，可能是止损
					priceDiff := actualClosePrice - lastPos.EntryPrice
					if priceDiff > 0.001 { // 价格高于开仓价超过0.001，判断为止损
						wasStopLoss = true
					}
				}

				// 估算自动触发平仓的时间（使用持仓更新时间或当前时间减去一个周期作为近似值）
				// 由于我们无法知道确切的平仓时间，使用持仓的UpdateTime（如果可用）或当前时间减去一个周期
				closeTimestamp := time.Now()
				if lastPos.UpdateTime > 0 {
					// 使用持仓更新时间作为平仓时间的近似值（通常更接近实际平仓时间）
					closeTimestamp = time.Unix(0, lastPos.UpdateTime*int64(time.Millisecond))
				} else {
					// 如果没有更新时间，使用当前时间减去一个周期（3分钟）作为近似值
					// 假设平仓发生在周期开始前的某个时间点
					closeTimestamp = time.Now().Add(-3 * time.Minute)
				}

				// 创建自动触发的close决策记录
				closeAction := logger.DecisionAction{
					Action:          "close_" + lastPos.Side,
					Symbol:          lastPos.Symbol,
					Quantity:        lastPos.Quantity,
					Leverage:        lastPos.Leverage,
					Price:           actualClosePrice, // 使用当前市场价格作为实际成交价
					Timestamp:       closeTimestamp,   // 使用估算的平仓时间
					Success:         true,
					IsAutoTriggered: true, // 标记为自动触发
					WasStopLoss:     wasStopLoss,
				}

				if wasStopLoss {
					log.Printf("🛑 检测到自动止损: %s %s (开仓价: %.4f, 成交价: %.4f)",
						lastPos.Symbol, lastPos.Side, lastPos.EntryPrice, actualClosePrice)
				} else {
					log.Printf("🎯 检测到自动止盈: %s %s (开仓价: %.4f, 成交价: %.4f)",
						lastPos.Symbol, lastPos.Side, lastPos.EntryPrice, actualClosePrice)
				}

				// 将自动触发的close决策添加到记录中
				record.Decisions = append(record.Decisions, closeAction)
				record.ExecutionLog = append(record.ExecutionLog,
					fmt.Sprintf("🔄 自动触发: %s %s (数量: %.4f, 价格: %.4f)",
						lastPos.Symbol, closeAction.Action, lastPos.Quantity, actualClosePrice))

				// ⚠️ 关键修复：将自动触发信息写入Context，供AI决策时使用
				ctx.AutoTriggeredCloses = append(ctx.AutoTriggeredCloses, decision.AutoTriggeredClose{
					Symbol:      lastPos.Symbol,
					Side:        lastPos.Side,
					EntryPrice:  lastPos.EntryPrice,
					ClosePrice:  actualClosePrice,
					Quantity:    lastPos.Quantity,
					Leverage:    lastPos.Leverage,
					WasStopLoss: wasStopLoss,
					Timestamp:   closeTimestamp, // 使用估算的平仓时间
				})

			}
		}
	}

	// 更新lastCyclePositions为当前持仓（在周期结束时更新，但这里先保存一份副本）
	// 注意：这里保存的是当前持仓的副本，在周期结束时再更新
	currentPositionsCopy := make([]decision.PositionInfo, len(ctx.Positions))
	copy(currentPositionsCopy, ctx.Positions)

	// 保存账户状态快照
	record.AccountState = logger.AccountSnapshot{
		TotalBalance:          ctx.Account.TotalEquity,
		AvailableBalance:      ctx.Account.AvailableBalance,
		TotalUnrealizedProfit: ctx.Account.TotalPnL,
		PositionCount:         ctx.Account.PositionCount,
		MarginUsedPct:         ctx.Account.MarginUsedPct,
	}

	// 保存持仓快照
	for _, pos := range ctx.Positions {
		record.Positions = append(record.Positions, logger.PositionSnapshot{
			Symbol:           pos.Symbol,
			Side:             pos.Side,
			PositionAmt:      pos.Quantity,
			EntryPrice:       pos.EntryPrice,
			MarkPrice:        pos.MarkPrice,
			UnrealizedProfit: pos.UnrealizedPnL,
			Leverage:         float64(pos.Leverage),
			LiquidationPrice: pos.LiquidationPrice,
		})
	}

	// 保存候选币种列表
	for _, coin := range ctx.CandidateCoins {
		record.CandidateCoins = append(record.CandidateCoins, coin.Symbol)
	}

	log.Printf("📊 账户净值: %.2f USDT | 可用: %.2f USDT | 持仓: %d",
		ctx.Account.TotalEquity, ctx.Account.AvailableBalance, ctx.Account.PositionCount)

	// 4. 调用AI获取完整决策
	log.Printf("🤖 正在请求AI分析并决策... [模板: %s]", at.systemPromptTemplate)
	decision, err := decision.GetFullDecisionWithCustomPrompt(ctx, at.mcpClient, at.customPrompt, at.overrideBasePrompt, at.systemPromptTemplate)

	// 即使有错误，也保存思维链、决策和输入prompt（用于debug）
	if decision != nil {
		record.SystemPrompt = decision.SystemPrompt // 保存系统提示词
		record.InputPrompt = decision.UserPrompt
		record.CoTTrace = decision.CoTTrace
		if len(decision.Decisions) > 0 {
			decisionJSON, _ := json.MarshalIndent(decision.Decisions, "", "  ")
			record.DecisionJSON = string(decisionJSON)
		}
	}

	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("获取AI决策失败: %v", err)

		// 打印系统提示词和AI思维链（即使有错误，也要输出以便调试）
		if decision != nil {
			if decision.SystemPrompt != "" {
				log.Println("\n" + strings.Repeat("=", 70))
				log.Printf("📋 系统提示词 [模板: %s] (错误情况)", at.systemPromptTemplate)
				log.Println(strings.Repeat("=", 70))
				log.Println(decision.SystemPrompt)
				log.Println(strings.Repeat("=", 70))
			}

			if decision.CoTTrace != "" {
				log.Println("\n" + strings.Repeat("-", 70))
				log.Println("💭 AI思维链分析（错误情况）:")
				log.Println(strings.Repeat("-", 70))
				log.Println(decision.CoTTrace)
				log.Println(strings.Repeat("-", 70))
			}
		}

		at.decisionLogger.LogDecision(record)
		return fmt.Errorf("获取AI决策失败: %w", err)
	}

	// // 5. 打印系统提示词
	// log.Printf("\n" + strings.Repeat("=", 70))
	// log.Printf("📋 系统提示词 [模板: %s]", at.systemPromptTemplate)
	// log.Println(strings.Repeat("=", 70))
	// log.Println(decision.SystemPrompt)
	// log.Printf(strings.Repeat("=", 70) + "\n")

	// 6. 打印AI思维链
	// log.Printf("\n" + strings.Repeat("-", 70))
	// log.Println("💭 AI思维链分析:")
	// log.Println(strings.Repeat("-", 70))
	// log.Println(decision.CoTTrace)
	// log.Printf(strings.Repeat("-", 70) + "\n")

	// 7. 打印AI决策
	// log.Printf("📋 AI决策列表 (%d 个):\n", len(decision.Decisions))
	// for i, d := range decision.Decisions {
	// 	log.Printf("  [%d] %s: %s - %s", i+1, d.Symbol, d.Action, d.Reasoning)
	// 	if d.Action == "open_long" || d.Action == "open_short" {
	// 		log.Printf("      杠杆: %dx | 仓位: %.2f USDT | 止损: %.4f | 止盈: %.4f",
	// 			d.Leverage, d.PositionSizeUSD, d.StopLoss, d.TakeProfit)
	// 	}
	// }
	log.Println()

	// 8. 对决策排序：确保先平仓后开仓（防止仓位叠加超限）
	sortedDecisions := sortDecisionsByPriority(decision.Decisions)

	log.Println("🔄 执行顺序（已优化）: 先平仓→后开仓")
	for i, d := range sortedDecisions {
		log.Printf("  [%d] %s %s", i+1, d.Symbol, d.Action)
	}
	log.Println()

	// 执行决策并记录结果
	for _, d := range sortedDecisions {
		actionRecord := logger.DecisionAction{
			Action:          d.Action,
			Symbol:          d.Symbol,
			Quantity:        0,
			Leverage:        d.Leverage,
			Price:           0,
			Timestamp:       time.Now(),
			Success:         false,
			IsAutoTriggered: false, // AI决策触发的，不是自动触发
			WasStopLoss:     false,
		}

		// 检查是否与自动触发的记录冲突（如果AI决策中有相同的close操作，移除自动触发的记录）
		// ⚠️ 修复：从后往前遍历，避免在遍历时删除元素导致索引错乱
		if d.Action == "close_long" || d.Action == "close_short" {
			// 从后往前遍历，查找并移除对应的自动触发记录
			for i := len(record.Decisions) - 1; i >= 0; i-- {
				existingAction := record.Decisions[i]
				if existingAction.IsAutoTriggered &&
					existingAction.Action == d.Action &&
					existingAction.Symbol == d.Symbol {
					// 移除自动触发的记录，因为AI决策会执行相同的操作
					record.Decisions = append(record.Decisions[:i], record.Decisions[i+1:]...)
					log.Printf("ℹ️  AI决策覆盖自动触发: %s %s", d.Symbol, d.Action)
					break // 找到第一个匹配的就退出，因为理论上每个symbol_side只有一个自动触发记录
				}
			}
		}

		if err := at.executeDecisionWithRecord(&d, &actionRecord); err != nil {
			log.Printf("❌ 执行决策失败 (%s %s): %v", d.Symbol, d.Action, err)
			actionRecord.Error = err.Error()
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ %s %s 失败: %v", d.Symbol, d.Action, err))
		} else {
			actionRecord.Success = true
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("✓ %s %s 成功", d.Symbol, d.Action))
			// 成功执行后短暂延迟
			time.Sleep(1 * time.Second)
		}

		record.Decisions = append(record.Decisions, actionRecord)
	}

	// 9. 更新交易状态追踪
	// 检查是否有任何交易操作（开仓或平仓）成功执行
	// 包括AI决策和自动触发的平仓
	for _, actionRecord := range record.Decisions {
		if actionRecord.Success {
			action := actionRecord.Action
			if action == "open_long" || action == "open_short" || action == "close_long" || action == "close_short" {
				// 使用交易操作的时间戳（对于自动触发平仓，使用估算的平仓时间）
				at.lastTradeTime = actionRecord.Timestamp
				break
			}
		}
	}

	// 检查是否有自动触发的平仓
	hasAutoTriggeredClose := len(ctx.AutoTriggeredCloses) > 0

	// 检查所有决策是否都是 wait
	allWait := true
	for _, d := range decision.Decisions {
		if d.Action != "wait" {
			allWait = false
			break
		}
	}

	// 更新连续等待周期数
	// 只有当所有决策都是 wait 且没有自动触发平仓时，才增加连续等待周期数
	if allWait && len(decision.Decisions) > 0 && !hasAutoTriggeredClose {
		// 所有决策都是 wait 且没有自动触发平仓，增加连续等待周期数
		at.consecutiveWaitCycles++
	} else {
		// 有任何非 wait 的决策或自动触发平仓，重置连续等待周期数
		at.consecutiveWaitCycles = 0
	}

	// 10. 保存决策记录
	if err := at.decisionLogger.LogDecision(record); err != nil {
		log.Printf("⚠ 保存决策记录失败: %v", err)
	}

	// 11. 更新lastCyclePositions为当前持仓（用于下一个周期检测持仓变化）
	at.lastCyclePositions = currentPositionsCopy

	return nil
}

// buildTradingContext 构建交易上下文
func (at *AutoTrader) buildTradingContext() (*decision.Context, error) {
	// 1. 获取账户信息
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("获取账户余额失败: %w", err)
	}

	// 获取账户字段
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

	// Total Equity = 钱包余额 + 未实现盈亏
	totalEquity := totalWalletBalance + totalUnrealizedProfit

	// 2. 获取持仓信息
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var positionInfos []decision.PositionInfo
	totalMarginUsed := 0.0

	// 当前持仓的key集合（用于清理已平仓的记录）
	currentPositionKeys := make(map[string]bool)

	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity // 空仓数量为负，转为正数
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

		// 计算占用保证金（估算）
		leverage := 10 // 默认值，实际应该从持仓信息获取
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed

		// 跟踪持仓首次出现时间
		posKey := symbol + "_" + side
		currentPositionKeys[posKey] = true
		if _, exists := at.positionFirstSeenTime[posKey]; !exists {
			// 新持仓，记录当前时间
			at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()
		}
		updateTime := at.positionFirstSeenTime[posKey]

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
			UpdateTime:       updateTime,
		})
	}

	// 清理已平仓的持仓记录
	for key := range at.positionFirstSeenTime {
		if !currentPositionKeys[key] {
			delete(at.positionFirstSeenTime, key)
		}
	}

	// 为当前持仓构建入场快照信息
	entrySnapshots := make(map[string]*decision.PositionEntrySnapshot)
	for _, pos := range positionInfos {
		snapshot, err := at.getPositionEntrySnapshot(pos.Symbol, pos.Side)
		if err != nil {
			log.Printf("⚠️  获取持仓入场快照失败 %s %s: %v", pos.Symbol, pos.Side, err)
			continue
		}
		if snapshot != nil {
			key := fmt.Sprintf("%s_%s", pos.Symbol, pos.Side)
			entrySnapshots[key] = snapshot
		}
	}

	// 3. 获取交易员的候选币种池
	candidateCoins, err := at.getCandidateCoins()
	if err != nil {
		return nil, fmt.Errorf("获取候选币种失败: %w", err)
	}

	// 4. 计算总盈亏
	totalPnL := totalEquity - at.initialBalance
	totalPnLPct := 0.0
	if at.initialBalance > 0 {
		totalPnLPct = (totalPnL / at.initialBalance) * 100
	}

	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	// 5. 分析历史表现（最近100个周期，避免长期持仓的交易记录丢失）
	// 假设每3分钟一个周期，100个周期 = 5小时，足够覆盖大部分交易
	performance, err := at.decisionLogger.AnalyzePerformance(100)
	if err != nil {
		log.Printf("⚠️  分析历史表现失败: %v", err)
		// 不影响主流程，继续执行（但设置performance为nil以避免传递错误数据）
		performance = nil
	}

	// 5.1 获取连续统计和风险指标（新增）
	var streakStats interface{}
	var riskMetrics interface{}
	if at.tradeAnalyticsService != nil {
		log.Printf("🔍 [DEBUG] tradeAnalyticsService 不为 nil，开始获取连续统计和风险指标 (trader_id: %s)", at.id)
		filter := &trade_analytics.AnalyticsFilter{
			TraderID: at.id,
			// 不设置时间范围，使用所有历史数据
		}
		
		// 获取连续统计
		stats, err := at.tradeAnalyticsService.GetStreakStats(context.Background(), filter)
		if err == nil {
			streakStats = stats
			log.Printf("🔍 [DEBUG] 成功获取连续统计: %+v", stats)
		} else {
			log.Printf("⚠️  获取连续统计失败: %v", err)
		}
		
		// 获取风险指标（包含夏普比率）
		risk, err := at.tradeAnalyticsService.GetRiskMetrics(context.Background(), filter)
		if err == nil {
			riskMetrics = risk
			log.Printf("🔍 [DEBUG] 成功获取风险指标，SharpeRatio: %.2f", risk.SharpeRatio)
		} else {
			log.Printf("⚠️  获取风险指标失败: %v", err)
		}
	} else {
		log.Printf("🔍 [DEBUG] tradeAnalyticsService 为 nil，跳过获取连续统计和风险指标 (trader_id: %s)", at.id)
	}

	// 6. 构建上下文
	ctx := &decision.Context{
		CurrentTime:           time.Now().Format("2006-01-02 15:04:05"),
		RuntimeMinutes:        int(time.Since(at.startTime).Minutes()),
		CallCount:             at.callCount,
		BTCETHLeverage:        at.config.BTCETHLeverage,        // 使用配置的杠杆倍数
		AltcoinLeverage:       at.config.AltcoinLeverage,       // 使用配置的杠杆倍数
		LogDir:                at.decisionLogger.GetLogDir(),   // 设置日志目录路径
		HistoryDecisionCycles: at.config.HistoryDecisionCycles, // 历史决策周期数（0=禁用）
		AutoTriggeredCloses:   []decision.AutoTriggeredClose{}, // 初始化为空slice，将在检测到自动触发时填充
		LastTradeTime:         at.lastTradeTime,                // 最后一次交易时间
		ConsecutiveWaitCycles: at.consecutiveWaitCycles,        // 连续等待周期数
		Account: decision.AccountInfo{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			MarginUsed:       totalMarginUsed,
			MarginUsedPct:    marginUsedPct,
			PositionCount:    len(positionInfos),
		},
		Positions:              positionInfos,
		PositionEntrySnapshots: entrySnapshots,
		CandidateCoins:         candidateCoins,
		Performance:            performance, // 添加历史表现分析
		StreakStats:            streakStats, // 添加连续统计（新增）
		RiskMetrics:            riskMetrics, // 添加风险指标（包含夏普比率）
	}

	return ctx, nil
}

// getPositionEntrySnapshot 从历史决策日志中提取指定持仓的建仓快照
func (at *AutoTrader) getPositionEntrySnapshot(symbol, side string) (*decision.PositionEntrySnapshot, error) {
	records, err := at.decisionLogger.GetLatestRecords(500)
	if err != nil {
		return nil, err
	}

	targetAction := ""
	switch strings.ToLower(side) {
	case "long":
		targetAction = "open_long"
	case "short":
		targetAction = "open_short"
	default:
		return nil, fmt.Errorf("未知持仓方向: %s", side)
	}

	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		for j := len(record.Decisions) - 1; j >= 0; j-- {
			action := record.Decisions[j]
			if !action.Success {
				continue
			}
			if !strings.EqualFold(action.Symbol, symbol) {
				continue
			}
			if action.Action != targetAction {
				continue
			}

			snapshot := &decision.PositionEntrySnapshot{
				Symbol:        symbol,
				Side:          side,
				CycleNumber:   record.CycleNumber,
				Timestamp:     record.Timestamp.Format("2006-01-02 15:04:05"),
				EntryPrice:    action.Price,
				DecisionIndex: j,
			}

			// 解析决策JSON以获取更详细的入场信息
			if record.DecisionJSON != "" {
				var decisions []decision.Decision
				if err := json.Unmarshal([]byte(record.DecisionJSON), &decisions); err != nil {
					log.Printf("⚠️  解析决策JSON失败(%s_%s): %v", symbol, side, err)
				} else {
					for idx, d := range decisions {
						if d.Action != targetAction {
							continue
						}
						if !strings.EqualFold(d.Symbol, symbol) {
							continue
						}
						snapshot.StopLoss = d.StopLoss
						snapshot.TakeProfit = d.TakeProfit
						snapshot.Confidence = d.Confidence
						snapshot.RiskUSD = d.RiskUSD
						snapshot.Reasoning = d.Reasoning
						snapshot.DecisionIndex = idx
						break
					}
				}
			}

			// 保存思维链片段（截断以避免过长）
			if record.CoTTrace != "" {
				snapshot.CotTrace = truncateText(record.CoTTrace, 1200)
			}

			return snapshot, nil
		}
	}

	return nil, nil
}

// truncateText 将文本截断到指定长度（保留中文/英文字符）
func truncateText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	trimmed := strings.TrimSpace(string(runes[:maxLen]))
	return trimmed + "…"
}

// executeDecisionWithRecord 执行AI决策并记录详细信息
func (at *AutoTrader) executeDecisionWithRecord(decision *decision.Decision, actionRecord *logger.DecisionAction) error {
	switch decision.Action {
	case "open_long":
		return at.executeOpenLongWithRecord(decision, actionRecord)
	case "open_short":
		return at.executeOpenShortWithRecord(decision, actionRecord)
	case "close_long":
		return at.executeCloseLongWithRecord(decision, actionRecord)
	case "close_short":
		return at.executeCloseShortWithRecord(decision, actionRecord)
	case "hold", "wait":
		// 无需执行，仅记录
		return nil
	default:
		return fmt.Errorf("未知的action: %s", decision.Action)
	}
}

// executeOpenLongWithRecord 执行开多仓并记录详细信息
func (at *AutoTrader) executeOpenLongWithRecord(decision *decision.Decision, actionRecord *logger.DecisionAction) error {
	log.Printf("  📈 开多仓: %s", decision.Symbol)

	// ⚠️ 关键：检查是否已有同币种同方向持仓，如果有则拒绝开仓（防止仓位叠加超限）
	positions, err := at.trader.GetPositions()
	if err == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
				return fmt.Errorf("❌ %s 已有多仓，拒绝开仓以防止仓位叠加超限。如需换仓，请先给出 close_long 决策", decision.Symbol)
			}
		}
	}

	// 获取当前价格
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}

	// 计算数量
	quantity := decision.PositionSizeUSD / marketData.CurrentPrice
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// 设置仓位模式
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		log.Printf("  ⚠️ 设置仓位模式失败: %v", err)
		// 继续执行，不影响交易
	}

	// 开仓
	order, err := at.trader.OpenLong(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// 记录订单ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	log.Printf("  ✓ 开仓成功，订单ID: %v, 数量: %.4f", order["orderId"], quantity)

	// ⚠️ 关键修复：等待订单成交后，查询持仓信息获取实际成交价格（entryPrice）
	// 这是真实成交价格，而不是下单时的市场价格
	time.Sleep(2 * time.Second) // 等待订单成交
	positions, err = at.trader.GetPositions()
	if err == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
				if entryPrice, ok := pos["entryPrice"].(float64); ok && entryPrice > 0 {
					actionRecord.Price = entryPrice // 使用实际成交价格
					log.Printf("  ✅ 已获取实际成交价格: %.2f (entryPrice)", entryPrice)
					break
				}
			}
		}
	}
	if actionRecord.Price == 0 {
		log.Printf("  ⚠️ 无法获取实际成交价格，使用市场价格: %.2f", marketData.CurrentPrice)
	}

	// 记录开仓时间
	posKey := decision.Symbol + "_long"
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()

	// 设置止损止盈
	if err := at.trader.SetStopLoss(decision.Symbol, "LONG", quantity, decision.StopLoss); err != nil {
		log.Printf("  ⚠ 设置止损失败: %v", err)
	}
	if err := at.trader.SetTakeProfit(decision.Symbol, "LONG", quantity, decision.TakeProfit); err != nil {
		log.Printf("  ⚠ 设置止盈失败: %v", err)
	}

	return nil
}

// executeOpenShortWithRecord 执行开空仓并记录详细信息
func (at *AutoTrader) executeOpenShortWithRecord(decision *decision.Decision, actionRecord *logger.DecisionAction) error {
	log.Printf("  📉 开空仓: %s", decision.Symbol)

	// ⚠️ 关键：检查是否已有同币种同方向持仓，如果有则拒绝开仓（防止仓位叠加超限）
	positions, err := at.trader.GetPositions()
	if err == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
				return fmt.Errorf("❌ %s 已有空仓，拒绝开仓以防止仓位叠加超限。如需换仓，请先给出 close_short 决策", decision.Symbol)
			}
		}
	}

	// 获取当前价格
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}

	// 计算数量
	quantity := decision.PositionSizeUSD / marketData.CurrentPrice
	actionRecord.Quantity = quantity
	actionRecord.Price = marketData.CurrentPrice

	// 设置仓位模式
	if err := at.trader.SetMarginMode(decision.Symbol, at.config.IsCrossMargin); err != nil {
		log.Printf("  ⚠️ 设置仓位模式失败: %v", err)
		// 继续执行，不影响交易
	}

	// 开仓
	order, err := at.trader.OpenShort(decision.Symbol, quantity, decision.Leverage)
	if err != nil {
		return err
	}

	// 记录订单ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	log.Printf("  ✓ 开仓成功，订单ID: %v, 数量: %.4f", order["orderId"], quantity)

	// ⚠️ 关键修复：等待订单成交后，查询持仓信息获取实际成交价格（entryPrice）
	// 这是真实成交价格，而不是下单时的市场价格
	time.Sleep(2 * time.Second) // 等待订单成交
	positions, err = at.trader.GetPositions()
	if err == nil {
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
				if entryPrice, ok := pos["entryPrice"].(float64); ok && entryPrice > 0 {
					actionRecord.Price = entryPrice // 使用实际成交价格
					log.Printf("  ✅ 已获取实际成交价格: %.2f (entryPrice)", entryPrice)
					break
				}
			}
		}
	}
	if actionRecord.Price == 0 {
		log.Printf("  ⚠️ 无法获取实际成交价格，使用市场价格: %.2f", marketData.CurrentPrice)
	}

	// 记录开仓时间
	posKey := decision.Symbol + "_short"
	at.positionFirstSeenTime[posKey] = time.Now().UnixMilli()

	// 设置止损止盈
	if err := at.trader.SetStopLoss(decision.Symbol, "SHORT", quantity, decision.StopLoss); err != nil {
		log.Printf("  ⚠ 设置止损失败: %v", err)
	}
	if err := at.trader.SetTakeProfit(decision.Symbol, "SHORT", quantity, decision.TakeProfit); err != nil {
		log.Printf("  ⚠ 设置止盈失败: %v", err)
	}

	return nil
}

// executeCloseLongWithRecord 执行平多仓并记录详细信息
func (at *AutoTrader) executeCloseLongWithRecord(decision *decision.Decision, actionRecord *logger.DecisionAction) error {
	log.Printf("  🔄 平多仓: %s", decision.Symbol)

	// ⚠️ 关键修复：平仓前记录市场价格（作为初始价格），平仓后会更新为实际成交价格
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice // 初始价格，平仓后会更新

	// ⚠️ 关键：在平仓前获取持仓数量并记录
	// 这样在计算PL时才能正确使用quantity
	positions, err := at.trader.GetPositions()
	if err != nil {
		log.Printf("  ⚠️ 获取持仓失败，quantity将无法记录: %v", err)
	} else {
		found := false
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "long" {
				// 类型安全的quantity获取
				positionAmt, ok := pos["positionAmt"]
				if !ok {
					log.Printf("  ⚠️ 持仓信息中没有positionAmt字段")
					break
				}

				var quantity float64
				switch v := positionAmt.(type) {
				case float64:
					quantity = v
				case string:
					var parseErr error
					quantity, parseErr = strconv.ParseFloat(v, 64)
					if parseErr != nil {
						log.Printf("  ⚠️ 无法解析positionAmt: %v", parseErr)
						break
					}
				default:
					log.Printf("  ⚠️ positionAmt类型不支持: %T", v)
					break
				}

				if quantity < 0 {
					quantity = -quantity // 空仓数量为负，转为正数
				}

				actionRecord.Quantity = quantity
				// 获取杠杆（如果有）
				if leverage, ok := pos["leverage"].(float64); ok {
					actionRecord.Leverage = int(leverage)
				}
				found = true
				break
			}
		}
		if !found {
			log.Printf("  ⚠️ 未找到 %s 的多仓，quantity将无法记录", decision.Symbol)
		}
	}

	// 平仓
	order, err := at.trader.CloseLong(decision.Symbol, 0) // 0 = 全部平仓
	if err != nil {
		return err
	}

	// 记录订单ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// ⚠️ 关键修复：优先使用订单返回值中的真实成交价格
	// 如果订单返回了executionPrice，使用它；否则使用成交后的市场价格作为fallback
	if executionPrice, ok := order["executionPrice"].(float64); ok && executionPrice > 0 {
		actionRecord.Price = executionPrice
		log.Printf("  ✅ 已获取真实成交价格: %.2f (来自订单返回值)", executionPrice)
	} else {
		// Fallback: 等待订单成交后，查询当前市场价格作为近似值
		time.Sleep(2 * time.Second) // 等待订单成交
		marketDataAfter, err := market.Get(decision.Symbol)
		if err == nil {
			actionRecord.Price = marketDataAfter.CurrentPrice // 使用实际成交后的市场价格
			log.Printf("  ⚠️ 订单返回值无价格，使用成交后市场价格: %.2f (近似值)", marketDataAfter.CurrentPrice)
		} else {
			log.Printf("  ⚠️ 无法获取成交后价格，使用平仓前价格: %.2f", actionRecord.Price)
		}
	}

	if actionRecord.Quantity > 0 {
		log.Printf("  ✓ 平仓成功，数量: %.4f, 成交价格: %.2f", actionRecord.Quantity, actionRecord.Price)
	} else {
		log.Printf("  ✓ 平仓成功（数量未记录，将使用开仓quantity），成交价格: %.2f", actionRecord.Price)
	}

	return nil
}

// executeCloseShortWithRecord 执行平空仓并记录详细信息
func (at *AutoTrader) executeCloseShortWithRecord(decision *decision.Decision, actionRecord *logger.DecisionAction) error {
	log.Printf("  🔄 平空仓: %s", decision.Symbol)

	// ⚠️ 关键修复：平仓前记录市场价格（作为初始价格），平仓后会更新为实际成交价格
	marketData, err := market.Get(decision.Symbol)
	if err != nil {
		return err
	}
	actionRecord.Price = marketData.CurrentPrice // 初始价格，平仓后会更新

	// ⚠️ 关键：在平仓前获取持仓数量并记录
	// 这样在计算PL时才能正确使用quantity
	positions, err := at.trader.GetPositions()
	if err != nil {
		log.Printf("  ⚠️ 获取持仓失败，quantity将无法记录: %v", err)
	} else {
		found := false
		for _, pos := range positions {
			if pos["symbol"] == decision.Symbol && pos["side"] == "short" {
				// 类型安全的quantity获取
				positionAmt, ok := pos["positionAmt"]
				if !ok {
					log.Printf("  ⚠️ 持仓信息中没有positionAmt字段")
					break
				}

				var quantity float64
				switch v := positionAmt.(type) {
				case float64:
					quantity = v
				case string:
					var parseErr error
					quantity, parseErr = strconv.ParseFloat(v, 64)
					if parseErr != nil {
						log.Printf("  ⚠️ 无法解析positionAmt: %v", parseErr)
						break
					}
				default:
					log.Printf("  ⚠️ positionAmt类型不支持: %T", v)
					break
				}

				if quantity < 0 {
					quantity = -quantity // 空仓数量为负，转为正数
				}

				actionRecord.Quantity = quantity
				// 获取杠杆（如果有）
				if leverage, ok := pos["leverage"].(float64); ok {
					actionRecord.Leverage = int(leverage)
				}
				found = true
				break
			}
		}
		if !found {
			log.Printf("  ⚠️ 未找到 %s 的空仓，quantity将无法记录", decision.Symbol)
		}
	}

	// 平仓
	order, err := at.trader.CloseShort(decision.Symbol, 0) // 0 = 全部平仓
	if err != nil {
		return err
	}

	// 记录订单ID
	if orderID, ok := order["orderId"].(int64); ok {
		actionRecord.OrderID = orderID
	}

	// ⚠️ 关键修复：优先使用订单返回值中的真实成交价格
	// 如果订单返回了executionPrice，使用它；否则使用成交后的市场价格作为fallback
	if executionPrice, ok := order["executionPrice"].(float64); ok && executionPrice > 0 {
		actionRecord.Price = executionPrice
		log.Printf("  ✅ 已获取真实成交价格: %.2f (来自订单返回值)", executionPrice)
	} else {
		// Fallback: 等待订单成交后，查询当前市场价格作为近似值
		time.Sleep(2 * time.Second) // 等待订单成交
		marketDataAfter, err := market.Get(decision.Symbol)
		if err == nil {
			actionRecord.Price = marketDataAfter.CurrentPrice // 使用实际成交后的市场价格
			log.Printf("  ⚠️ 订单返回值无价格，使用成交后市场价格: %.2f (近似值)", marketDataAfter.CurrentPrice)
		} else {
			log.Printf("  ⚠️ 无法获取成交后价格，使用平仓前价格: %.2f", actionRecord.Price)
		}
	}

	if actionRecord.Quantity > 0 {
		log.Printf("  ✓ 平仓成功，数量: %.4f, 成交价格: %.2f", actionRecord.Quantity, actionRecord.Price)
	} else {
		log.Printf("  ✓ 平仓成功（数量未记录，将使用开仓quantity），成交价格: %.2f", actionRecord.Price)
	}

	return nil
}

// GetID 获取trader ID
func (at *AutoTrader) GetID() string {
	return at.id
}

// GetName 获取trader名称
func (at *AutoTrader) GetName() string {
	return at.name
}

// GetAIModel 获取AI模型
func (at *AutoTrader) GetAIModel() string {
	return at.aiModel
}

// GetExchange 获取交易所
func (at *AutoTrader) GetExchange() string {
	return at.exchange
}

// SetCustomPrompt 设置自定义交易策略prompt
func (at *AutoTrader) SetCustomPrompt(prompt string) {
	at.customPrompt = prompt
}

// SetOverrideBasePrompt 设置是否覆盖基础prompt
func (at *AutoTrader) SetOverrideBasePrompt(override bool) {
	at.overrideBasePrompt = override
}

// SetSystemPromptTemplate 设置系统提示词模板
func (at *AutoTrader) SetSystemPromptTemplate(templateName string) {
	at.systemPromptTemplate = templateName
}

// GetSystemPromptTemplate 获取当前系统提示词模板名称
func (at *AutoTrader) GetSystemPromptTemplate() string {
	return at.systemPromptTemplate
}

// GetDecisionLogger 获取决策日志记录器
func (at *AutoTrader) GetDecisionLogger() *logger.DecisionLogger {
	return at.decisionLogger
}

// GetStatus 获取系统状态（用于API）
func (at *AutoTrader) GetStatus() map[string]interface{} {
	aiProvider := "DeepSeek"
	if at.config.UseQwen {
		aiProvider = "Qwen"
	}

	return map[string]interface{}{
		"trader_id":       at.id,
		"trader_name":     at.name,
		"ai_model":        at.aiModel,
		"exchange":        at.exchange,
		"is_running":      at.isRunning,
		"start_time":      at.startTime.Format(time.RFC3339),
		"runtime_minutes": int(time.Since(at.startTime).Minutes()),
		"call_count":      at.callCount,
		"initial_balance": at.initialBalance,
		"scan_interval":   at.config.ScanInterval.String(),
		"stop_until":      at.stopUntil.Format(time.RFC3339),
		"last_reset_time": at.lastResetTime.Format(time.RFC3339),
		"ai_provider":     aiProvider,
	}
}

// GetAccountInfo 获取账户信息（用于API）
func (at *AutoTrader) GetAccountInfo() (map[string]interface{}, error) {
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("获取余额失败: %w", err)
	}

	// 获取账户字段
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

	// Total Equity = 钱包余额 + 未实现盈亏
	totalEquity := totalWalletBalance + totalUnrealizedProfit

	// 获取持仓计算总保证金
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	totalMarginUsed := 0.0
	totalUnrealizedPnL := 0.0
	for _, pos := range positions {
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		totalUnrealizedPnL += unrealizedPnl

		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed
	}

	totalPnL := totalEquity - at.initialBalance
	totalPnLPct := 0.0
	if at.initialBalance > 0 {
		totalPnLPct = (totalPnL / at.initialBalance) * 100
	}

	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	return map[string]interface{}{
		// 核心字段
		"total_equity":      totalEquity,           // 账户净值 = wallet + unrealized
		"wallet_balance":    totalWalletBalance,    // 钱包余额（不含未实现盈亏）
		"unrealized_profit": totalUnrealizedProfit, // 未实现盈亏（从API）
		"available_balance": availableBalance,      // 可用余额

		// 盈亏统计
		"total_pnl":            totalPnL,           // 总盈亏 = equity - initial
		"total_pnl_pct":        totalPnLPct,        // 总盈亏百分比
		"total_unrealized_pnl": totalUnrealizedPnL, // 未实现盈亏（从持仓计算）
		"initial_balance":      at.initialBalance,  // 初始余额
		"daily_pnl":            at.dailyPnL,        // 日盈亏

		// 持仓信息
		"position_count":  len(positions),  // 持仓数量
		"margin_used":     totalMarginUsed, // 保证金占用
		"margin_used_pct": marginUsedPct,   // 保证金使用率
	}, nil
}

// GetPositions 获取持仓列表（用于API）
func (at *AutoTrader) GetPositions() ([]map[string]interface{}, error) {
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var result []map[string]interface{}
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

		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		// 计算占用保证金
		marginUsed := (quantity * markPrice) / float64(leverage)

		// 计算盈亏百分比（基于保证金）
		// 收益率 = 未实现盈亏 / 保证金 × 100%
		pnlPct := 0.0
		if marginUsed > 0 {
			pnlPct = (unrealizedPnl / marginUsed) * 100
		}

		result = append(result, map[string]interface{}{
			"symbol":             symbol,
			"side":               side,
			"entry_price":        entryPrice,
			"mark_price":         markPrice,
			"quantity":           quantity,
			"leverage":           leverage,
			"unrealized_pnl":     unrealizedPnl,
			"unrealized_pnl_pct": pnlPct,
			"liquidation_price":  liquidationPrice,
			"margin_used":        marginUsed,
		})
	}

	return result, nil
}

// sortDecisionsByPriority 对决策排序：先平仓，再开仓，最后hold/wait
// 这样可以避免换仓时仓位叠加超限
func sortDecisionsByPriority(decisions []decision.Decision) []decision.Decision {
	if len(decisions) <= 1 {
		return decisions
	}

	// 定义优先级
	getActionPriority := func(action string) int {
		switch action {
		case "close_long", "close_short":
			return 1 // 最高优先级：先平仓
		case "open_long", "open_short":
			return 2 // 次优先级：后开仓
		case "hold", "wait":
			return 3 // 最低优先级：观望
		default:
			return 999 // 未知动作放最后
		}
	}

	// 复制决策列表
	sorted := make([]decision.Decision, len(decisions))
	copy(sorted, decisions)

	// 按优先级排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if getActionPriority(sorted[i].Action) > getActionPriority(sorted[j].Action) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// getCandidateCoins 获取交易员的候选币种列表
func (at *AutoTrader) getCandidateCoins() ([]decision.CandidateCoin, error) {
	if len(at.tradingCoins) == 0 {
		// 使用数据库配置的默认币种列表
		var candidateCoins []decision.CandidateCoin

		if len(at.defaultCoins) > 0 {
			// 使用数据库中配置的默认币种
			for _, coin := range at.defaultCoins {
				symbol := normalizeSymbol(coin)
				candidateCoins = append(candidateCoins, decision.CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"default"}, // 标记为数据库默认币种
				})
			}
			log.Printf("📋 [%s] 使用数据库默认币种: %d个币种 %v",
				at.name, len(candidateCoins), at.defaultCoins)
			return candidateCoins, nil
		} else {
			// 如果数据库中没有配置默认币种，则使用AI500+OI Top作为fallback
			const ai500Limit = 20 // AI500取前20个评分最高的币种

			mergedPool, err := pool.GetMergedCoinPool(ai500Limit)
			if err != nil {
				return nil, fmt.Errorf("获取合并币种池失败: %w", err)
			}

			// 构建候选币种列表（包含来源信息）
			for _, symbol := range mergedPool.AllSymbols {
				sources := mergedPool.SymbolSources[symbol]
				candidateCoins = append(candidateCoins, decision.CandidateCoin{
					Symbol:  symbol,
					Sources: sources, // "ai500" 和/或 "oi_top"
				})
			}

			log.Printf("📋 [%s] 数据库无默认币种配置，使用AI500+OI Top: AI500前%d + OI_Top20 = 总计%d个候选币种",
				at.name, ai500Limit, len(candidateCoins))
			return candidateCoins, nil
		}
	} else {
		// 使用自定义币种列表
		var candidateCoins []decision.CandidateCoin
		for _, coin := range at.tradingCoins {
			// 确保币种格式正确（转为大写USDT交易对）
			symbol := normalizeSymbol(coin)
			candidateCoins = append(candidateCoins, decision.CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"custom"}, // 标记为自定义来源
			})
		}

		log.Printf("📋 [%s] 使用自定义币种: %d个币种 %v",
			at.name, len(candidateCoins), at.tradingCoins)
		return candidateCoins, nil
	}
}

// normalizeSymbol 标准化币种符号（确保以USDT结尾）
func normalizeSymbol(symbol string) string {
	// 转为大写
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// 确保以USDT结尾
	if !strings.HasSuffix(symbol, "USDT") {
		symbol = symbol + "USDT"
	}

	return symbol
}

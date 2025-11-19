package decision

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/pool"
	"strings"
	"time"
)

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // "long" or "short"
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	LiquidationPrice float64 `json:"liquidation_price"`
	MarginUsed       float64 `json:"margin_used"`
	UpdateTime       int64   `json:"update_time"` // 持仓更新时间戳（毫秒）
}

// PositionEntrySnapshot 持仓入场快照（建仓时的核心信息）
type PositionEntrySnapshot struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	CycleNumber   int     `json:"cycle_number"`
	Timestamp     string  `json:"timestamp"`
	EntryPrice    float64 `json:"entry_price"`
	StopLoss      float64 `json:"stop_loss"`
	TakeProfit    float64 `json:"take_profit"`
	Confidence    int     `json:"confidence"`
	RiskUSD       float64 `json:"risk_usd"`
	Reasoning     string  `json:"reasoning"`
	CotTrace      string  `json:"cot_trace"`
	DecisionIndex int     `json:"decision_index"` // 在决策数组中的索引（用于定位）
}

// AccountInfo 账户信息
type AccountInfo struct {
	TotalEquity      float64 `json:"total_equity"`      // 账户净值
	AvailableBalance float64 `json:"available_balance"` // 可用余额
	TotalPnL         float64 `json:"total_pnl"`         // 总盈亏
	TotalPnLPct      float64 `json:"total_pnl_pct"`     // 总盈亏百分比
	MarginUsed       float64 `json:"margin_used"`       // 已用保证金
	MarginUsedPct    float64 `json:"margin_used_pct"`   // 保证金使用率
	PositionCount    int     `json:"position_count"`    // 持仓数量
}

// CandidateCoin 候选币种（来自币种池）
type CandidateCoin struct {
	Symbol  string   `json:"symbol"`
	Sources []string `json:"sources"` // 来源: "ai500" 和/或 "oi_top"
}

// OITopData 持仓量增长Top数据（用于AI决策参考）
type OITopData struct {
	Rank              int     // OI Top排名
	OIDeltaPercent    float64 // 持仓量变化百分比（1小时）
	OIDeltaValue      float64 // 持仓量变化价值
	PriceDeltaPercent float64 // 价格变化百分比
	NetLong           float64 // 净多仓
	NetShort          float64 // 净空仓
}

// AutoTriggeredClose 自动触发的平仓信息（止盈/止损）
type AutoTriggeredClose struct {
	Symbol      string    `json:"symbol"`        // 币种
	Side        string    `json:"side"`          // "long" 或 "short"
	EntryPrice  float64   `json:"entry_price"`   // 开仓价格
	ClosePrice  float64   `json:"close_price"`   // 平仓价格（成交价）
	Quantity    float64   `json:"quantity"`      // 数量
	Leverage    int       `json:"leverage"`      // 杠杆
	WasStopLoss bool      `json:"was_stop_loss"` // 是否为止损（true=止损, false=止盈）
	Timestamp   time.Time `json:"timestamp"`     // 触发时间
}

// Context 交易上下文（传递给AI的完整信息）
type Context struct {
	CurrentTime            string                            `json:"current_time"`
	RuntimeMinutes         int                               `json:"runtime_minutes"`
	CallCount              int                               `json:"call_count"`
	Account                AccountInfo                       `json:"account"`
	Positions              []PositionInfo                    `json:"positions"`
	PositionEntrySnapshots map[string]*PositionEntrySnapshot `json:"-"`
	CandidateCoins         []CandidateCoin                   `json:"candidate_coins"`
	AutoTriggeredCloses    []AutoTriggeredClose              `json:"-"` // 本周期检测到的自动触发平仓（不序列化，但用于构建prompt）
	MarketDataMap          map[string]*market.Data           `json:"-"` // 不序列化，但内部使用
	OITopDataMap           map[string]*OITopData             `json:"-"` // OI Top数据映射
	Performance            interface{}                       `json:"-"` // 历史表现分析（logger.PerformanceAnalysis）
	StreakStats            interface{}                       `json:"-"` // 连续统计（trade_analytics.StreakStatistics，可选）
	RiskMetrics            interface{}                       `json:"-"` // 风险指标（trade_analytics.RiskMetrics，可选）
	BTCETHLeverage         int                               `json:"-"` // BTC/ETH杠杆倍数（从配置读取）
	AltcoinLeverage        int                               `json:"-"` // 山寨币杠杆倍数（从配置读取）
	LogDir                 string                            `json:"-"` // 决策日志目录路径（用于读取历史思维链）
	HistoryDecisionCycles  int                               `json:"-"` // 历史决策周期数（0=禁用，默认2）
	LastTradeTime          time.Time                         `json:"-"` // 最后一次交易时间（开仓或平仓）
	ConsecutiveWaitCycles  int                               `json:"-"` // 连续等待周期数（连续 wait 决策的次数）
}

// Decision AI的交易决策
type Decision struct {
	Symbol          string  `json:"symbol"`
	Action          string  `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	Leverage        int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss        float64 `json:"stop_loss,omitempty"`
	TakeProfit      float64 `json:"take_profit,omitempty"`
	Confidence      int     `json:"confidence,omitempty"` // 信心度 (0-100)
	RiskUSD         float64 `json:"risk_usd,omitempty"`   // 最大美元风险
	Reasoning       string  `json:"reasoning"`
}

// FullDecision AI的完整决策（包含思维链）
type FullDecision struct {
	SystemPrompt string     `json:"system_prompt"` // 系统提示词（发送给AI的系统prompt）
	UserPrompt   string     `json:"user_prompt"`   // 发送给AI的输入prompt
	CoTTrace     string     `json:"cot_trace"`     // 思维链分析（AI输出）
	Decisions    []Decision `json:"decisions"`     // 具体决策列表
	Timestamp    time.Time  `json:"timestamp"`
}

// GetFullDecision 获取AI的完整交易决策（批量分析所有币种和持仓）
func GetFullDecision(ctx *Context, mcpClient *mcp.Client) (*FullDecision, error) {
	return GetFullDecisionWithCustomPrompt(ctx, mcpClient, "", false, "")
}

// GetFullDecisionWithCustomPrompt 获取AI的完整交易决策（支持自定义prompt和模板选择）
func GetFullDecisionWithCustomPrompt(ctx *Context, mcpClient *mcp.Client, customPrompt string, overrideBase bool, templateName string) (*FullDecision, error) {
	// 1. 为所有币种获取市场数据
	if err := fetchMarketDataForContext(ctx); err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	// 2. 构建 System Prompt（固定规则）和 User Prompt（动态数据）
	systemPrompt := buildSystemPromptWithCustom(ctx.Account.TotalEquity, ctx.BTCETHLeverage, ctx.AltcoinLeverage, customPrompt, overrideBase, templateName)
	userPrompt := buildUserPrompt(ctx)

	// 3. 调用AI API（使用 system + user prompt）并在缺失JSON时进行纠错重试
	const maxDecisionAttempts = 2
	retryPrompt := userPrompt
	var lastDecision *FullDecision
	var lastErr error

	for attempt := 1; attempt <= maxDecisionAttempts; attempt++ {
		if attempt > 1 {
			log.Printf("🔁 正在重新请求AI决策 (尝试 %d/%d)...", attempt, maxDecisionAttempts)
		} else {
			log.Println("🚀 正在调用AI API进行决策分析...")
		}

		aiResponse, err := mcpClient.CallWithMessages(systemPrompt, retryPrompt)
		if err != nil {
			lastErr = fmt.Errorf("调用AI API失败: %w", err)
			continue
		}

		decision, parseErr := parseFullDecisionResponse(aiResponse, ctx.Account.TotalEquity, ctx.BTCETHLeverage, ctx.AltcoinLeverage)
		if parseErr != nil {
			lastDecision = decision
			lastErr = fmt.Errorf("解析AI响应失败: %w", parseErr)

			if errors.Is(parseErr, ErrDecisionJSONNotFound) && attempt < maxDecisionAttempts {
				log.Println("⚠️  AI 输出缺少 JSON 决策数组，已自动追加提醒并重试")
				retryPrompt = userPrompt + "\n\n⚠️  你的上一条输出缺少 `JSON决策数组`。请严格按照输出格式重新回复：先给出思维链，然后在 ```json 代码块中输出一个有效的 JSON 数组（如果没有决策，请输出 `[]`）。不要省略 JSON，也不要添加多余文本。"
				continue
			}

			return decision, lastErr
		}

		decision.Timestamp = time.Now()
		decision.SystemPrompt = systemPrompt // 保存系统prompt
		decision.UserPrompt = retryPrompt    // 保存输入prompt（含可能的纠错提示）
		return decision, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("AI决策生成失败：未知错误")
	}

	return lastDecision, lastErr
}

// fetchMarketDataForContext 为上下文中的所有币种获取市场数据和OI数据
func fetchMarketDataForContext(ctx *Context) error {
	ctx.MarketDataMap = make(map[string]*market.Data)
	ctx.OITopDataMap = make(map[string]*OITopData)

	// 收集所有需要获取数据的币种
	symbolSet := make(map[string]bool)

	// 1. 优先获取持仓币种的数据（这是必须的）
	for _, pos := range ctx.Positions {
		symbolSet[pos.Symbol] = true
	}

	// 2. 候选币种数量根据账户状态动态调整
	maxCandidates := calculateMaxCandidates(ctx)
	for i, coin := range ctx.CandidateCoins {
		if i >= maxCandidates {
			break
		}
		symbolSet[coin.Symbol] = true
	}

	// 并发获取市场数据
	// 持仓币种集合（用于判断是否跳过OI检查）
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	for symbol := range symbolSet {
		data, err := market.Get(symbol)
		if err != nil {
			// 单个币种失败不影响整体，只记录错误
			log.Printf("⚠️  获取 %s 市场数据失败: %v", symbol, err)
			continue
		}

		// ⚠️ 流动性过滤：持仓价值低于15M USD的币种不做（多空都不做）
		// 持仓价值 = 持仓量 × 当前价格
		// 但现有持仓必须保留（需要决策是否平仓）
		// 重要：OI过滤是可选的，如果OI数据不可用（为0或获取失败），不进行过滤，允许进入决策
		isExistingPosition := positionSymbols[symbol]

		// 只有当OI数据存在且大于0时才进行过滤
		shouldFilterOI := !isExistingPosition &&
			data.OpenInterest != nil &&
			data.CurrentPrice > 0 &&
			data.OpenInterest.Latest > 0

		if shouldFilterOI {
			// 计算持仓价值（USD）= 持仓量 × 当前价格
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000 // 转换为百万美元单位
			if oiValueInMillions < 15 {
				log.Printf("⚠️  %s 持仓价值过低(%.2fM USD < 15M)，跳过此币种 [持仓量:%.0f × 价格:%.4f]",
					symbol, oiValueInMillions, data.OpenInterest.Latest, data.CurrentPrice)
				continue
			}
			log.Printf("✓ %s OI过滤通过: 持仓价值=%.2fM USD [持仓量:%.0f × 价格:%.4f]",
				symbol, oiValueInMillions, data.OpenInterest.Latest, data.CurrentPrice)
		} else if !isExistingPosition && data.OpenInterest != nil && data.OpenInterest.Latest == 0 {
			// OI数据为0，可能是数据源不支持，跳过过滤，允许进入决策
			log.Printf("ℹ️  %s OI数据为0（数据源可能不支持），跳过OI过滤，允许进入决策", symbol)
		}

		ctx.MarketDataMap[symbol] = data
		log.Printf("✓ 成功获取 %s 市场数据", symbol)
	}

	// 检查是否获取到任何市场数据
	if len(ctx.MarketDataMap) == 0 {
		return fmt.Errorf("未能获取任何币种的市场数据（尝试获取了 %d 个币种）", len(symbolSet))
	}
	log.Printf("✓ 成功获取 %d/%d 个币种的市场数据", len(ctx.MarketDataMap), len(symbolSet))

	// 加载OI Top数据（不影响主流程）
	oiPositions, err := pool.GetOITopPositions()
	if err == nil {
		for _, pos := range oiPositions {
			// 标准化符号匹配
			symbol := pos.Symbol
			ctx.OITopDataMap[symbol] = &OITopData{
				Rank:              pos.Rank,
				OIDeltaPercent:    pos.OIDeltaPercent,
				OIDeltaValue:      pos.OIDeltaValue,
				PriceDeltaPercent: pos.PriceDeltaPercent,
				NetLong:           pos.NetLong,
				NetShort:          pos.NetShort,
			}
		}
	}

	return nil
}

// calculateMaxCandidates 根据账户状态计算需要分析的候选币种数量
func calculateMaxCandidates(ctx *Context) int {
	// 直接返回候选池的全部币种数量
	// 因为候选池已经在 auto_trader.go 中筛选过了
	// 固定分析前20个评分最高的币种（来自AI500）
	return len(ctx.CandidateCoins)
}

// buildSystemPromptWithCustom 构建包含自定义内容的 System Prompt
func buildSystemPromptWithCustom(accountEquity float64, btcEthLeverage, altcoinLeverage int, customPrompt string, overrideBase bool, templateName string) string {
	// 如果覆盖基础prompt且有自定义prompt，只使用自定义prompt
	if overrideBase && customPrompt != "" {
		return customPrompt
	}

	// 获取基础prompt（使用指定的模板）
	basePrompt := buildSystemPrompt(accountEquity, btcEthLeverage, altcoinLeverage, templateName)

	// 如果没有自定义prompt，直接返回基础prompt
	if customPrompt == "" {
		return basePrompt
	}

	// 添加自定义prompt部分到基础prompt
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")
	sb.WriteString("# 📌 个性化交易策略\n\n")
	sb.WriteString(customPrompt)
	sb.WriteString("\n\n")
	sb.WriteString("注意: 以上个性化策略是对基础规则的补充，不能违背基础风险控制原则。\n")

	return sb.String()
}

// buildSystemPrompt 构建 System Prompt（使用模板+动态部分）
func buildSystemPrompt(accountEquity float64, btcEthLeverage, altcoinLeverage int, templateName string) string {
	var sb strings.Builder

	// 1. 加载提示词模板（核心交易策略部分）
	if templateName == "" {
		templateName = "default" // 默认使用 default 模板
	}

	template, err := GetPromptTemplate(templateName)
	if err != nil {
		// 如果模板不存在，记录错误并使用 default
		log.Printf("⚠️  提示词模板 '%s' 不存在，使用 default: %v", templateName, err)
		template, err = GetPromptTemplate("default")
		if err != nil {
			// 如果连 default 都不存在，使用内置的简化版本
			log.Printf("❌ 无法加载任何提示词模板，使用内置简化版本")
			sb.WriteString("你是专业的加密货币交易AI。请根据市场数据做出交易决策。\n\n")
		} else {
			sb.WriteString(template.Content)
			sb.WriteString("\n\n")
		}
	} else {
		sb.WriteString(template.Content)
		sb.WriteString("\n\n")
	}

	// 2. 硬约束（风险控制）- 动态生成
	sb.WriteString("# 硬约束（风险控制）\n\n")
	sb.WriteString("1. 风险回报比: 必须 ≥ 1:3（冒1%风险，赚3%+收益）\n")
	sb.WriteString("2. 最多持仓: 3个币种（质量>数量）\n")
	sb.WriteString(fmt.Sprintf("3. 单币仓位: 山寨%.0f-%.0f U(%dx杠杆) | BTC/ETH %.0f-%.0f U(%dx杠杆)\n",
		accountEquity*0.8, accountEquity*1.5, altcoinLeverage, accountEquity*5, accountEquity*10, btcEthLeverage))
	sb.WriteString("4. 保证金: 总使用率 ≤ 90%\n\n")

	// 3. 输出格式 - 动态生成
	sb.WriteString("# 输出格式（严格执行）\n\n")
	sb.WriteString("⚠️ **重要**: 你必须严格按照以下格式输出，否则系统无法解析你的决策！\n\n")
	sb.WriteString("第一步: 思维链（纯文本）\n")
	sb.WriteString("简洁分析你的思考过程\n\n")
	sb.WriteString("第二步: JSON决策数组（必须是有效的JSON格式）\n\n")
	sb.WriteString("**严格要求**:\n")
	sb.WriteString("- 必须是有效的JSON数组格式，不能使用markdown格式\n")
	sb.WriteString("- 禁止使用markdown checkbox格式（如 `[x]`）或其他markdown标记\n")
	sb.WriteString("- JSON数组必须包含有效的决策对象或为空数组 `[]`\n")
	sb.WriteString("- JSON必须在代码块中，格式如下：\n\n")
	sb.WriteString("```json\n[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 91000, \"confidence\": 85, \"risk_usd\": 300, \"reasoning\": \"下跌趋势+MACD死叉\"},\n", btcEthLeverage, accountEquity*5))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\", \"reasoning\": \"止盈离场\"}\n")
	sb.WriteString("]\n```\n\n")
	sb.WriteString("如果没有决策，输出空数组：\n")
	sb.WriteString("```json\n[]\n```\n\n")
	sb.WriteString("字段说明:\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `confidence`: 0-100（开仓建议≥75）\n")
	sb.WriteString("- 开仓时必填: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning\n\n")

	return sb.String()
}

// buildUserPrompt 构建 User Prompt（动态数据）
func buildUserPrompt(ctx *Context) string {
	var sb strings.Builder

	// 获取历史决策的思维链（如果日志目录存在且配置启用）
	if ctx.LogDir != "" && ctx.HistoryDecisionCycles > 0 {
		decisionLogger := logger.NewDecisionLogger(ctx.LogDir)
		// 获取最近N个记录（当前cycle还未保存，所以GetLatestRecords会返回最新的N个已保存的cycle）
		records, err := decisionLogger.GetLatestRecords(ctx.HistoryDecisionCycles)
		if err != nil {
			log.Printf("⚠️  读取历史决策记录失败: %v", err)
		}
		if err == nil && len(records) > 0 {
			// 统计实际有效的记录数（有CoTTrace的记录）
			validCount := 0
			for _, record := range records {
				if record.CoTTrace != "" {
					validCount++
				}
			}

			if validCount > 0 {
				// 根据实际数量调整标题
				cycleText := "周期"
				if validCount > 1 {
					cycleText = fmt.Sprintf("前%d个周期", validCount)
				} else {
					cycleText = "前一个周期"
				}

				sb.WriteString(fmt.Sprintf("## 📚 历史决策参考（%s）\n\n", cycleText))
				sb.WriteString("以下是历史决策的思维链，**你必须主动利用这些历史信息进行自我纠正**。请回顾历史决策，识别错误模式和成功经验，并在当前决策中应用这些学习成果。同时，市场情况在不断变化，请基于当前最新的市场数据做出独立判断，但要从历史中学习。\n\n")
				// 从新到旧显示（records已经是按时间从旧到新排列，需要反转）
				for i := len(records) - 1; i >= 0; i-- {
					record := records[i]
					if record.CoTTrace != "" {
						sb.WriteString(fmt.Sprintf("### Cycle #%d (时间: %s)\n\n", record.CycleNumber, record.Timestamp.Format("2006-01-02 15:04:05")))
						sb.WriteString(record.CoTTrace)
						sb.WriteString("\n\n")
					}
				}
				sb.WriteString("---\n\n")
			}
		}
	}

	// ⚠️ 关键修复：显示本周期检测到的自动触发平仓信息（如果有）
	if len(ctx.AutoTriggeredCloses) > 0 {
		sb.WriteString("## ⚠️ 重要提示：本周期检测到自动触发的平仓\n\n")
		sb.WriteString("在本次决策周期开始时，系统检测到以下持仓被自动止盈/止损触发平仓。这些信息可能影响你的决策：\n\n")
		for _, autoClose := range ctx.AutoTriggeredCloses {
			triggerType := "止盈"
			emoji := "🎯"
			if autoClose.WasStopLoss {
				triggerType = "止损"
				emoji = "🛑"
			}

			// 计算盈亏
			var pnlPercent float64
			if autoClose.Side == "long" {
				pnlPercent = ((autoClose.ClosePrice - autoClose.EntryPrice) / autoClose.EntryPrice) * 100
			} else {
				pnlPercent = ((autoClose.EntryPrice - autoClose.ClosePrice) / autoClose.EntryPrice) * 100
			}

			sb.WriteString(fmt.Sprintf("%s **%s %s** 被自动%s触发平仓\n", emoji, autoClose.Symbol, strings.ToUpper(autoClose.Side), triggerType))
			sb.WriteString(fmt.Sprintf("  - 开仓价: %.4f | 平仓价: %.4f | 盈亏: %+.2f%%\n",
				autoClose.EntryPrice, autoClose.ClosePrice, pnlPercent))
			sb.WriteString(fmt.Sprintf("  - 数量: %.4f | 杠杆: %dx | 触发时间: %s\n\n",
				autoClose.Quantity, autoClose.Leverage, autoClose.Timestamp.Format("15:04:05")))
		}
		sb.WriteString("---\n\n")
	}

	// 系统状态
	sb.WriteString("## ⏰ 系统状态\n\n")
	sb.WriteString(fmt.Sprintf("时间: %s | 周期: #%d | 运行: %d分钟\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes))

	// 交易状态追踪信息
	sb.WriteString("## 📊 交易状态追踪\n\n")
	if !ctx.LastTradeTime.IsZero() {
		timeSinceLastTrade := time.Since(ctx.LastTradeTime)
		totalSeconds := int(timeSinceLastTrade.Seconds())
		minutesSinceLastTrade := totalSeconds / 60
		secondsSinceLastTrade := totalSeconds % 60
		hoursSinceLastTrade := minutesSinceLastTrade / 60
		remainingMinutes := minutesSinceLastTrade % 60

		var timeSinceLastTradeStr string
		if hoursSinceLastTrade > 0 {
			timeSinceLastTradeStr = fmt.Sprintf("%d小时%d分钟", hoursSinceLastTrade, remainingMinutes)
		} else if minutesSinceLastTrade > 0 {
			timeSinceLastTradeStr = fmt.Sprintf("%d分钟", minutesSinceLastTrade)
		} else {
			timeSinceLastTradeStr = fmt.Sprintf("%d秒", secondsSinceLastTrade)
		}

		sb.WriteString(fmt.Sprintf("距离上次交易: %s (时间: %s)\n",
			timeSinceLastTradeStr, ctx.LastTradeTime.Format("2006-01-02 15:04:05")))
	} else {
		sb.WriteString("距离上次交易: 尚未进行任何交易\n")
	}

	if ctx.ConsecutiveWaitCycles > 0 {
		waitDurationMinutes := ctx.ConsecutiveWaitCycles * 3 // 每个周期3分钟
		waitDurationHours := waitDurationMinutes / 60
		waitDurationMinutesRemainder := waitDurationMinutes % 60

		var waitDurationStr string
		if waitDurationHours > 0 {
			waitDurationStr = fmt.Sprintf("%d小时%d分钟", waitDurationHours, waitDurationMinutesRemainder)
		} else {
			waitDurationStr = fmt.Sprintf("%d分钟", waitDurationMinutes)
		}

		sb.WriteString(fmt.Sprintf("连续等待: %d个周期 (%s)\n",
			ctx.ConsecutiveWaitCycles, waitDurationStr))
	}
	sb.WriteString("\n")

	// BTC 市场
	if btcData, hasBTC := ctx.MarketDataMap["BTCUSDT"]; hasBTC {
		sb.WriteString("## ₿ BTC市场概览\n\n")
		sb.WriteString(fmt.Sprintf("价格: %.2f | 1h: %+.2f%% | 4h: %+.2f%% | MACD: %.4f | RSI: %.2f\n\n",
			btcData.CurrentPrice, btcData.PriceChange1h, btcData.PriceChange4h,
			btcData.CurrentMACD, btcData.CurrentRSI7))
	}

	// 账户
	sb.WriteString("## 💰 账户信息\n\n")
	sb.WriteString(fmt.Sprintf("净值: %.2f | 余额: %.2f (%.1f%%) | 盈亏: %+.2f%% | 保证金使用率: %.1f%% | 持仓数: %d个\n\n",
		ctx.Account.TotalEquity,
		ctx.Account.AvailableBalance,
		(ctx.Account.AvailableBalance/ctx.Account.TotalEquity)*100,
		ctx.Account.TotalPnLPct,
		ctx.Account.MarginUsedPct,
		ctx.Account.PositionCount))

	// 持仓（完整市场数据）
	if len(ctx.Positions) > 0 {
		sb.WriteString("## 📈 当前持仓\n\n")
		for i, pos := range ctx.Positions {
			// 计算持仓时长
			holdingDuration := ""
			if pos.UpdateTime > 0 {
				durationMs := time.Now().UnixMilli() - pos.UpdateTime
				durationMin := durationMs / (1000 * 60) // 转换为分钟
				if durationMin < 60 {
					holdingDuration = fmt.Sprintf(" | 持仓时长%d分钟", durationMin)
				} else {
					durationHour := durationMin / 60
					durationMinRemainder := durationMin % 60
					holdingDuration = fmt.Sprintf(" | 持仓时长%d小时%d分钟", durationHour, durationMinRemainder)
				}
			}

			sb.WriteString(fmt.Sprintf("%d. %s %s | 入场价%.4f 当前价%.4f | 盈亏%+.2f%% | 杠杆%dx | 保证金%.0f | 强平价%.4f%s\n\n",
				i+1, pos.Symbol, strings.ToUpper(pos.Side),
				pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnLPct,
				pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

			// 使用FormatMarketData输出完整市场数据
			if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
				sb.WriteString(market.Format(marketData))
				sb.WriteString("\n")
			}

			// 入场快照信息
			if ctx.PositionEntrySnapshots != nil {
				posKey := fmt.Sprintf("%s_%s", pos.Symbol, pos.Side)
				if snapshot, ok := ctx.PositionEntrySnapshots[posKey]; ok && snapshot != nil {
					sb.WriteString(fmt.Sprintf("**入场快照**（Cycle #%d，时间 %s）\n", snapshot.CycleNumber, snapshot.Timestamp))
					priceLine := fmt.Sprintf("- 入场价 %.4f", snapshot.EntryPrice)
					if snapshot.StopLoss > 0 || snapshot.TakeProfit > 0 {
						priceLine = fmt.Sprintf("%s | 止损 %.4f | 止盈 %.4f", priceLine, snapshot.StopLoss, snapshot.TakeProfit)
					}
					sb.WriteString(priceLine + "\n")
					if snapshot.Confidence > 0 || snapshot.RiskUSD > 0 {
						sb.WriteString(fmt.Sprintf("- 置信度 %d | 风险敞口 %.2f USD\n", snapshot.Confidence, snapshot.RiskUSD))
					}
					if snapshot.Reasoning != "" {
						sb.WriteString(fmt.Sprintf("- 入场理由：%s\n", truncateForPrompt(snapshot.Reasoning, 280)))
					}
					if snapshot.CotTrace != "" {
						sb.WriteString(fmt.Sprintf("- 思维链摘录：%s\n", truncateForPrompt(snapshot.CotTrace, 400)))
					}
					sb.WriteString("\n")
				}
			}
		}
	} else {
		sb.WriteString("## 📈 当前持仓\n\n")
		sb.WriteString("当前持仓: 无\n\n")
	}

	// 候选币种（完整市场数据）
	sb.WriteString(fmt.Sprintf("## 🔍 候选币种 (%d个)\n\n", len(ctx.MarketDataMap)))
	displayedCount := 0
	for _, coin := range ctx.CandidateCoins {
		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		sourceTags := ""
		if len(coin.Sources) > 1 {
			sourceTags = " (AI500+OI_Top双重信号)"
		} else if len(coin.Sources) == 1 && coin.Sources[0] == "oi_top" {
			sourceTags = " (OI_Top持仓增长)"
		}

		// 使用FormatMarketData输出完整市场数据
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(market.Format(marketData))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// 夏普比率（从 trade_analytics 的 RiskMetrics 获取）
	if ctx.RiskMetrics != nil {
		// 从 RiskMetrics 中提取 SharpeRatio
		type RiskMetricsData struct {
			SharpeRatio float64 `json:"sharpe_ratio"`
		}
		var riskData RiskMetricsData
		if jsonData, err := json.Marshal(ctx.RiskMetrics); err == nil {
			if err := json.Unmarshal(jsonData, &riskData); err == nil {
				sb.WriteString("## 📊 绩效指标\n\n")
				sb.WriteString(fmt.Sprintf("夏普比率: %.2f\n\n", riskData.SharpeRatio))
			}
		}
	} else if ctx.Performance != nil {
		// 降级方案：如果 RiskMetrics 不可用，尝试从 Performance 中获取（向后兼容）
		type PerformanceData struct {
			SharpeRatio float64 `json:"sharpe_ratio"`
		}
		var perfData PerformanceData
		if jsonData, err := json.Marshal(ctx.Performance); err == nil {
			if err := json.Unmarshal(jsonData, &perfData); err == nil {
				sb.WriteString("## 📊 绩效指标\n\n")
				sb.WriteString(fmt.Sprintf("夏普比率: %.2f\n\n", perfData.SharpeRatio))
			}
		}
	}

	// 连续亏损信息（新增）
	log.Printf("🔍 [DEBUG] ctx.StreakStats: %+v", ctx.StreakStats)
	if ctx.StreakStats != nil {
		type StreakData struct {
			CurrentStreak       int     `json:"current_streak"`
			CurrentStreakType   string  `json:"current_streak_type"`
			LongestLosingStreak int     `json:"longest_losing_streak"`
			LastLossTimestamp   *string `json:"last_loss_timestamp,omitempty"` // 使用字符串类型便于JSON解析
		}
		var streakData StreakData
		if jsonData, err := json.Marshal(ctx.StreakStats); err == nil {
			if err := json.Unmarshal(jsonData, &streakData); err == nil {
				log.Printf("🔍 [DEBUG] streakData: %+v", streakData)
				if streakData.CurrentStreakType == "losing" {
					consecutiveLosses := -streakData.CurrentStreak // CurrentStreak 为负数表示连亏
					sb.WriteString("## 🛡️ 熔断机制状态\n\n")
					sb.WriteString(fmt.Sprintf("连续亏损: %d次", consecutiveLosses))
					if consecutiveLosses >= 2 {
						// 计算剩余暂停时间
						if streakData.LastLossTimestamp != nil && *streakData.LastLossTimestamp != "" {
							// 解析时间戳
							lastLossTime, err := time.Parse(time.RFC3339, *streakData.LastLossTimestamp)
							if err == nil {
								elapsed := time.Since(lastLossTime)
								remaining := 30*time.Minute - elapsed
								if remaining > 0 {
									sb.WriteString(fmt.Sprintf(" ⚠️ **已触发熔断机制**（剩余暂停时间: %.0f 分钟，从 %s 开始）\n",
										remaining.Minutes(), lastLossTime.Format("15:04:05")))
								} else {
									sb.WriteString(" ⚠️ **熔断机制已过期**（可恢复交易）\n")
								}
							} else {
								// 解析失败，使用默认提示
								log.Printf("🔍 [DEBUG] 时间戳解析失败: %v", err)
								sb.WriteString(" ⚠️ **已触发熔断机制**（应暂停交易约30分钟）\n")
							}
						} else {
							// 没有时间戳信息，使用默认提示
							sb.WriteString(" ⚠️ **已触发熔断机制**（应暂停交易约30分钟）\n")
						}
					} else {
						sb.WriteString("\n")
					}
					sb.WriteString(fmt.Sprintf("最长连亏: %d次\n\n", streakData.LongestLosingStreak))
					log.Printf("🔍 [DEBUG] 已添加熔断机制状态到prompt (连续亏损: %d次)", consecutiveLosses)
				} else {
					log.Printf("🔍 [DEBUG] CurrentStreakType 不是 'losing'，当前值: %s", streakData.CurrentStreakType)
				}
			} else {
				log.Printf("🔍 [DEBUG] JSON反序列化失败: %v", err)
			}
		} else {
			log.Printf("🔍 [DEBUG] JSON序列化失败: %v", err)
		}
	} else {
		log.Printf("🔍 [DEBUG] ctx.StreakStats 为 nil，跳过添加熔断机制状态")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("**现在请分析并输出决策（思维链 + JSON）**\n\n")
	sb.WriteString("⚠️ 提醒: 确保JSON数组是有效的JSON格式，不能是markdown checkbox或其他格式。如果没有决策，输出空数组 `[]`。\n")
	sb.WriteString("\n## ✅ 输出自检（提交前务必检查）\n\n")
	sb.WriteString("- 在完整回答的最后必须包含一个使用 ```json ... ``` 包裹的有效 JSON 数组；不要在 JSON 代码块后再追加说明文字。\n")
	sb.WriteString("- 如果没有任何动作，也要输出 ```json\\n[]\\n```，不得省略。\n")
	sb.WriteString("- 任何缺少 JSON 数组的回答都会被判为无效并导致重新提问，请务必遵守。\n")

	return sb.String()
}

// truncateForPrompt 截断文本以适配提示输出
func truncateForPrompt(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}

// parseFullDecisionResponse 解析AI的完整决策响应
var ErrDecisionJSONNotFound = errors.New("未在AI响应中找到JSON决策数组")

func parseFullDecisionResponse(aiResponse string, accountEquity float64, btcEthLeverage, altcoinLeverage int) (*FullDecision, error) {
	// 1. 提取思维链
	cotTrace := extractCoTTrace(aiResponse)

	// 2. 提取JSON决策列表
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("提取决策失败: %w", err)
	}

	// 3. 验证决策
	if err := validateDecisions(decisions, accountEquity, btcEthLeverage, altcoinLeverage); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("决策验证失败: %w", err)
	}

	return &FullDecision{
		CoTTrace:  cotTrace,
		Decisions: decisions,
	}, nil
}

// extractCoTTrace 提取思维链分析
func extractCoTTrace(response string) string {
	// 查找JSON数组的开始位置
	jsonStart := strings.Index(response, "[")

	if jsonStart > 0 {
		// 思维链是JSON数组之前的内容
		return strings.TrimSpace(response[:jsonStart])
	}

	// 如果找不到JSON，整个响应都是思维链
	return strings.TrimSpace(response)
}

// extractDecisions 提取JSON决策列表（增强版：支持多种格式和不规范JSON）
func extractDecisions(response string) ([]Decision, error) {
	// 策略1: 首先尝试从 markdown 代码块中提取 JSON
	if jsonContent := extractJSONFromMarkdownCodeBlock(response); jsonContent != "" {
		if decisions, err := tryParseJSON(jsonContent); err == nil {
			log.Printf("✅ 从 markdown 代码块中成功提取 JSON")
			return decisions, nil
		}
	}

	// 策略2: 智能查找JSON数组，跳过markdown格式（如 [x]）
	searchStart := 0
	maxAttempts := 100 // 防止无限循环
	attempts := 0

	for attempts < maxAttempts {
		attempts++

		// 查找JSON数组起始位置
		arrayStart := -1
		for searchStart < len(response) {
			pos := strings.Index(response[searchStart:], "[")
			if pos == -1 {
				break
			}
			actualPos := searchStart + pos

			// 检查是否是markdown checkbox格式 [x] 或 [ ]
			// 使用字符串匹配更简单可靠
			if actualPos+2 < len(response) {
				checkText := response[actualPos : actualPos+3]
				// 检查是否是3字符的markdown checkbox
				runes := []rune(checkText)
				if len(runes) == 3 && runes[0] == '[' && runes[2] == ']' {
					middleRune := runes[1]
					// 跳过markdown checkbox: [x], [X], [ ], [-], [✓], [✔], [☑] 等
					if middleRune == 'x' || middleRune == 'X' || middleRune == ' ' ||
						middleRune == '-' || middleRune == '✓' || middleRune == '✔' || middleRune == '☑' {
						// 确认是markdown checkbox，跳过
						searchStart = actualPos + len(checkText)
						continue
					}
				}
			}

			// 找到可能的JSON数组起始位置
			arrayStart = actualPos
			break
		}

		if arrayStart == -1 {
			// 没有找到 JSON 数组，返回错误让上游处理
			log.Printf("⚠️  未在AI输出中找到 JSON 决策数组")
			return nil, ErrDecisionJSONNotFound
		}

		// 从 [ 开始，匹配括号找到对应的 ]
		arrayEnd := findMatchingBracket(response, arrayStart)
		if arrayEnd == -1 {
			// 找不到匹配的右括号，尝试更宽松的解析
			log.Printf("⚠️  无法找到匹配的 JSON 数组结束括号，尝试宽松解析")
			if decisions, err := tryLooseJSONExtraction(response, arrayStart); err == nil {
				return decisions, nil
			}
			// 如果宽松解析也失败，继续查找下一个可能的 JSON 数组
			searchStart = arrayStart + 1
			continue
		}

		jsonContent := strings.TrimSpace(response[arrayStart : arrayEnd+1])

		// 🔧 首先检查是否是markdown checkbox格式（必须在JSON解析之前）
		jsonContentRunes := []rune(jsonContent)
		if len(jsonContentRunes) == 3 && jsonContentRunes[0] == '[' && jsonContentRunes[2] == ']' {
			middleRune := jsonContentRunes[1]
			// 检查是否是markdown checkbox字符
			if middleRune == 'x' || middleRune == 'X' || middleRune == ' ' ||
				middleRune == '-' || middleRune == '✓' || middleRune == '✔' || middleRune == '☑' {
				// 确认是markdown checkbox，跳过并继续查找下一个可能的JSON数组
				log.Printf("⚠️  检测到markdown checkbox格式: %q，跳过", jsonContent)
				searchStart = arrayStart + len(jsonContent)
				continue
			}
		}

		// 验证内容长度（空数组至少是"[]"，2个字符）
		if len(jsonContent) < 2 {
			// 太短，不可能是有效JSON，尝试查找下一个
			searchStart = arrayStart + 1
			continue
		}

		// 尝试解析JSON
		if decisions, err := tryParseJSON(jsonContent); err == nil {
			return decisions, nil
		}

		// 解析失败，可能是格式错误，尝试查找下一个JSON数组
		searchStart = arrayStart + 1
		continue
	}

	// 所有尝试都失败，返回错误
	log.Printf("⚠️  尝试多次后仍无法提取有效的JSON数组")
	return nil, ErrDecisionJSONNotFound
}

// extractJSONFromMarkdownCodeBlock 从 markdown 代码块中提取 JSON
func extractJSONFromMarkdownCodeBlock(response string) string {
	// 查找 ```json 或 ``` 代码块
	markers := []string{"```json", "```"}

	for _, marker := range markers {
		startIdx := strings.Index(response, marker)
		if startIdx == -1 {
			continue
		}

		// 跳过标记本身
		contentStart := startIdx + len(marker)
		// 跳过可能的换行符
		for contentStart < len(response) && (response[contentStart] == '\n' || response[contentStart] == '\r') {
			contentStart++
		}

		// 查找结束标记 ```
		endIdx := strings.Index(response[contentStart:], "```")
		if endIdx == -1 {
			// 没有找到结束标记，尝试提取到文本末尾
			content := strings.TrimSpace(response[contentStart:])
			if len(content) > 0 {
				return content
			}
			continue
		}

		actualEndIdx := contentStart + endIdx
		content := strings.TrimSpace(response[contentStart:actualEndIdx])
		if len(content) > 0 {
			return content
		}
	}

	return ""
}

// tryParseJSON 尝试解析 JSON（包含修复和验证）
func tryParseJSON(jsonContent string) ([]Decision, error) {
	// 🔧 修复常见的JSON格式错误
	jsonContent = fixMissingQuotes(jsonContent)

	// 尝试解析JSON
	var decisions []Decision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, err
	}

	return decisions, nil
}

// tryLooseJSONExtraction 宽松的 JSON 提取（处理不完整的 JSON）
func tryLooseJSONExtraction(response string, arrayStart int) ([]Decision, error) {
	// 尝试从 arrayStart 开始，查找可能的 JSON 内容
	// 策略：查找下一个换行符或文本结束，然后尝试修复 JSON

	// 查找可能的结束位置（下一个空行、代码块结束、或文本结束）
	endPos := len(response)

	// 查找空行（两个连续的换行符）
	if idx := strings.Index(response[arrayStart:], "\n\n"); idx != -1 {
		endPos = arrayStart + idx
	}

	// 查找代码块结束
	if idx := strings.Index(response[arrayStart:], "```"); idx != -1 && idx < (endPos-arrayStart) {
		endPos = arrayStart + idx
	}

	// 提取内容
	jsonContent := strings.TrimSpace(response[arrayStart:endPos])

	// 如果内容以 [ 开头但没有 ]，尝试添加 ]
	if strings.HasPrefix(jsonContent, "[") && !strings.HasSuffix(jsonContent, "]") {
		// 尝试找到最后一个有效的对象结束
		lastBrace := strings.LastIndex(jsonContent, "}")
		if lastBrace != -1 {
			// 截取到最后一个 }，然后添加 ]
			jsonContent = jsonContent[:lastBrace+1] + "]"
		} else {
			// 如果连 } 都没有，可能是空数组，直接返回 []
			if strings.TrimSpace(jsonContent) == "[" {
				jsonContent = "[]"
			}
		}
	}

	// 尝试解析修复后的 JSON
	return tryParseJSON(jsonContent)
}

// fixMissingQuotes 替换中文引号为英文引号（避免输入法自动转换）
func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")  // '
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")  // '
	return jsonStr
}

// validateDecisions 验证所有决策（需要账户信息和杠杆配置）
func validateDecisions(decisions []Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int) error {
	for i, decision := range decisions {
		if err := validateDecision(&decision, accountEquity, btcEthLeverage, altcoinLeverage); err != nil {
			return fmt.Errorf("决策 #%d 验证失败: %w", i+1, err)
		}
	}
	return nil
}

// findMatchingBracket 查找匹配的右括号
func findMatchingBracket(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// validateDecision 验证单个决策的有效性
func validateDecision(d *Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int) error {
	// 验证action
	validActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("无效的action: %s", d.Action)
	}

	// 开仓操作必须提供完整参数
	if d.Action == "open_long" || d.Action == "open_short" {
		// 根据币种使用配置的杠杆上限
		maxLeverage := altcoinLeverage          // 山寨币使用配置的杠杆
		maxPositionValue := accountEquity * 1.5 // 山寨币最多1.5倍账户净值
		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			maxLeverage = btcEthLeverage          // BTC和ETH使用配置的杠杆
			maxPositionValue = accountEquity * 10 // BTC/ETH最多10倍账户净值
		}

		if d.Leverage <= 0 || d.Leverage > maxLeverage {
			return fmt.Errorf("杠杆必须在1-%d之间（%s，当前配置上限%d倍）: %d", maxLeverage, d.Symbol, maxLeverage, d.Leverage)
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("仓位大小必须大于0: %.2f", d.PositionSizeUSD)
		}
		// 验证仓位价值上限（加1%容差以避免浮点数精度问题）
		tolerance := maxPositionValue * 0.01 // 1%容差
		if d.PositionSizeUSD > maxPositionValue+tolerance {
			if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
				return fmt.Errorf("BTC/ETH单币种仓位价值不能超过%.0f USDT（10倍账户净值），实际: %.0f", maxPositionValue, d.PositionSizeUSD)
			} else {
				return fmt.Errorf("山寨币单币种仓位价值不能超过%.0f USDT（1.5倍账户净值），实际: %.0f", maxPositionValue, d.PositionSizeUSD)
			}
		}
		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("止损和止盈必须大于0")
		}

		// 验证止损止盈的合理性
		if d.Action == "open_long" {
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("做多时止损价必须小于止盈价")
			}
		} else {
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("做空时止损价必须大于止盈价")
			}
		}

		// 验证风险回报比（必须≥1:3）
		// 计算入场价（假设当前市价）
		var entryPrice float64
		if d.Action == "open_long" {
			// 做多：入场价在止损和止盈之间
			entryPrice = d.StopLoss + (d.TakeProfit-d.StopLoss)*0.2 // 假设在20%位置入场
		} else {
			// 做空：入场价在止损和止盈之间
			entryPrice = d.StopLoss - (d.StopLoss-d.TakeProfit)*0.2 // 假设在20%位置入场
		}

		var riskPercent, rewardPercent, riskRewardRatio float64
		if d.Action == "open_long" {
			riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
			rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		} else {
			riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
			rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		}

		// 硬约束：风险回报比必须≥3.0
		if riskRewardRatio < 3.0 {
			return fmt.Errorf("风险回报比过低(%.2f:1)，必须≥3.0:1 [风险:%.2f%% 收益:%.2f%%] [止损:%.2f 止盈:%.2f]",
				riskRewardRatio, riskPercent, rewardPercent, d.StopLoss, d.TakeProfit)
		}
	}

	return nil
}

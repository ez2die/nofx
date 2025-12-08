package analyzer

import (
	"encoding/json"
	"fmt"
	"log"

	"nofx/review"
)

// DecisionJSON 决策JSON结构（用于解析）
type DecisionJSON struct {
	Decisions []Decision `json:"decisions"`
}

// Decision 单个决策（用于解析DecisionJSON）
type Decision struct {
	Symbol        string  `json:"symbol"`
	Action        string  `json:"action"`
	Leverage      int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss      float64 `json:"stop_loss,omitempty"`
	TakeProfit    float64 `json:"take_profit,omitempty"`
	Confidence    int     `json:"confidence,omitempty"`
	RiskUSD       float64 `json:"risk_usd,omitempty"`
	Reasoning     string  `json:"reasoning"`
}

// RuleEngine 规则引擎
type RuleEngine struct {
	rules []RuleChecker
}

// NewRuleEngine 创建规则引擎
func NewRuleEngine() *RuleEngine {
	engine := &RuleEngine{
		rules: []RuleChecker{},
	}

	// 注册核心规则（Phase 1）
	engine.RegisterRule(NewRiskRewardRatioChecker(3.0))           // 风险回报比 ≥ 3.0:1
	engine.RegisterRule(NewLeverageLimitChecker(20))              // 杠杆限制 ≤ 20
	engine.RegisterRule(NewStopLossDirectionChecker())            // 止损方向检查
	engine.RegisterRule(NewConfidenceRequirementChecker(70, 85))  // 置信度要求
	engine.RegisterRule(NewStopLossWidthChecker())                // 止损宽度检查
	engine.RegisterRule(NewAccountRiskChecker(0.08))              // 账户风险检查 ≤ 8%

	return engine
}

// RegisterRule 注册规则检查器
func (e *RuleEngine) RegisterRule(checker RuleChecker) {
	e.rules = append(e.rules, checker)
}

// CheckDecision 检查决策是否违反规则
func (e *RuleEngine) CheckDecision(decision *review.DecisionRecord) ([]review.Violation, error) {
	var allViolations []review.Violation

	// 解析DecisionJSON
	parsedDecisions, err := e.parseDecisionJSON(decision.DecisionJSON)
	if err != nil {
		log.Printf("⚠️ 解析决策JSON失败 (cycle %d): %v", decision.CycleNumber, err)
		// 继续检查，使用默认值
	}

	// 遍历所有规则检查器
	for _, rule := range e.rules {
		violations, err := rule.Check(decision, parsedDecisions)
		if err != nil {
			log.Printf("⚠️ 规则检查失败 (rule: %s, cycle %d): %v", rule.GetRuleID(), decision.CycleNumber, err)
			continue
		}
		allViolations = append(allViolations, violations...)
	}

	return allViolations, nil
}

// GetRuleDefinitions 获取所有规则定义
func (e *RuleEngine) GetRuleDefinitions() []review.RuleDefinition {
	definitions := make([]review.RuleDefinition, len(e.rules))
	for i, rule := range e.rules {
		definitions[i] = review.RuleDefinition{
			ID:          rule.GetRuleID(),
			Name:        rule.GetRuleName(),
			Type:        rule.GetRuleType(),
			Severity:    rule.GetSeverity(),
			Description: rule.GetDescription(),
			Priority:    rule.GetPriority(),
		}
	}
	return definitions
}

// parseDecisionJSON 解析DecisionJSON字符串
func (e *RuleEngine) parseDecisionJSON(jsonStr string) ([]Decision, error) {
	if jsonStr == "" {
		return []Decision{}, nil
	}

	var decisionJSON DecisionJSON
	if err := json.Unmarshal([]byte(jsonStr), &decisionJSON); err != nil {
		// 尝试解析为数组格式
		var decisions []Decision
		if err2 := json.Unmarshal([]byte(jsonStr), &decisions); err2 != nil {
			return nil, fmt.Errorf("解析DecisionJSON失败: %w", err)
		}
		return decisions, nil
	}

	return decisionJSON.Decisions, nil
}

// RuleChecker 规则检查器接口
type RuleChecker interface {
	Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error)
	GetRuleID() string
	GetRuleName() string
	GetRuleType() string
	GetSeverity() string
	GetDescription() string
	GetPriority() int
}

// getEstimatedEntryPrice 获取估算的入场价格（用于未执行的决策）
// 优先使用positions中的markPrice，其次使用stopLoss和takeProfit的中点
func getEstimatedEntryPrice(symbol string, action string, stopLoss, takeProfit float64, positions []review.PositionSnapshot) float64 {
	// 优先从positions中获取markPrice
	for _, pos := range positions {
		if pos.Symbol == symbol {
			if pos.MarkPrice > 0 {
				return pos.MarkPrice
			}
		}
	}

	// 如果没有positions，使用stopLoss和takeProfit的中点作为估算价格
	if stopLoss > 0 && takeProfit > 0 {
		if action == "open_long" {
			// 多仓：stopLoss < entry < takeProfit，使用中点
			if stopLoss < takeProfit {
				return (stopLoss + takeProfit) / 2
			}
		} else if action == "open_short" {
			// 空仓：takeProfit < entry < stopLoss，使用中点
			if takeProfit < stopLoss {
				return (takeProfit + stopLoss) / 2
			}
		}
	}

	return 0
}

// findExecutedAction 查找对应的已执行action
func findExecutedAction(symbol, action string, decisions []review.DecisionAction) *review.DecisionAction {
	for i := range decisions {
		if decisions[i].Symbol == symbol && decisions[i].Action == action && decisions[i].Success {
			return &decisions[i]
		}
	}
	return nil
}

// RiskRewardRatioChecker 风险回报比检查器
type RiskRewardRatioChecker struct {
	minRatio float64
}

// NewRiskRewardRatioChecker 创建风险回报比检查器
func NewRiskRewardRatioChecker(minRatio float64) *RiskRewardRatioChecker {
	return &RiskRewardRatioChecker{
		minRatio: minRatio,
	}
}

func (c *RiskRewardRatioChecker) GetRuleID() string   { return "risk_reward_ratio" }
func (c *RiskRewardRatioChecker) GetRuleName() string { return "风险回报比要求" }
func (c *RiskRewardRatioChecker) GetRuleType() string { return "hard_constraint" }
func (c *RiskRewardRatioChecker) GetSeverity() string { return "critical" }
func (c *RiskRewardRatioChecker) GetDescription() string {
	return fmt.Sprintf("风险回报比必须 ≥ %.1f:1", c.minRatio)
}
func (c *RiskRewardRatioChecker) GetPriority() int { return 1 }

func (c *RiskRewardRatioChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	// 1. 检查已执行的决策
	for i, action := range decision.Decisions {
		if !action.Success {
			continue
		}

		if action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		// 从parsedDecisions中查找对应的决策
		var parsedDecision *Decision
		for j, pd := range parsedDecisions {
			if pd.Symbol == action.Symbol && pd.Action == action.Action {
				parsedDecision = &parsedDecisions[j]
				break
			}
		}

		if parsedDecision == nil {
			continue
		}

		stopLoss := parsedDecision.StopLoss
		takeProfit := parsedDecision.TakeProfit
		entryPrice := action.Price

		if stopLoss <= 0 || takeProfit <= 0 || entryPrice <= 0 {
			continue
		}

		// 计算风险回报比
		var risk, reward float64
		if action.Action == "open_long" {
			// 多仓：风险 = entry - stopLoss, 回报 = takeProfit - entry
			risk = entryPrice - stopLoss
			reward = takeProfit - entryPrice
		} else {
			// 空仓：风险 = stopLoss - entry, 回报 = entry - takeProfit
			risk = stopLoss - entryPrice
			reward = entryPrice - takeProfit
		}

		if risk <= 0 || reward <= 0 {
			continue
		}

		ratio := reward / risk
		if ratio < c.minRatio {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("风险回报比 %.2f:1 低于要求 %.1f:1", ratio, c.minRatio),
				DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":       action.Symbol,
					"entry_price":  entryPrice,
					"stop_loss":    stopLoss,
					"take_profit":  takeProfit,
					"risk":         risk,
					"reward":       reward,
					"ratio":        ratio,
					"min_ratio":    c.minRatio,
					"executed":     true,
				},
			})
		}
	}

	// 2. 检查未执行的决策意图（识别到开单信号但最终没开单）
	for i, pd := range parsedDecisions {
		if pd.Action != "open_long" && pd.Action != "open_short" {
			continue
		}

		// 检查是否有对应的已执行action（已检查过的跳过）
		if findExecutedAction(pd.Symbol, pd.Action, decision.Decisions) != nil {
			continue
		}

		// 这是一个未执行的决策意图，检查是否违反规则
		stopLoss := pd.StopLoss
		takeProfit := pd.TakeProfit
		if stopLoss <= 0 || takeProfit <= 0 {
			continue
		}

		// 获取估算的入场价格
		entryPrice := getEstimatedEntryPrice(pd.Symbol, pd.Action, stopLoss, takeProfit, decision.Positions)
		if entryPrice <= 0 {
			continue // 无法估算价格，跳过
		}

		// 计算风险回报比
		var risk, reward float64
		if pd.Action == "open_long" {
			// 多仓：风险 = entry - stopLoss, 回报 = takeProfit - entry
			risk = entryPrice - stopLoss
			reward = takeProfit - entryPrice
		} else {
			// 空仓：风险 = stopLoss - entry, 回报 = entry - takeProfit
			risk = stopLoss - entryPrice
			reward = entryPrice - takeProfit
		}

		if risk <= 0 || reward <= 0 {
			continue
		}

		ratio := reward / risk
		if ratio < c.minRatio {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("未执行决策：风险回报比 %.2f:1 低于要求 %.1f:1", ratio, c.minRatio),
				DecisionID:  fmt.Sprintf("%d_intent_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":        pd.Symbol,
					"entry_price":   entryPrice,
					"stop_loss":     stopLoss,
					"take_profit":   takeProfit,
					"risk":          risk,
					"reward":        reward,
					"ratio":         ratio,
					"min_ratio":     c.minRatio,
					"executed":      false,
					"is_intent":     true,
				},
			})
		}
	}

	return violations, nil
}

// LeverageLimitChecker 杠杆限制检查器
type LeverageLimitChecker struct {
	maxLeverage int
}

// NewLeverageLimitChecker 创建杠杆限制检查器
func NewLeverageLimitChecker(maxLeverage int) *LeverageLimitChecker {
	return &LeverageLimitChecker{
		maxLeverage: maxLeverage,
	}
}

func (c *LeverageLimitChecker) GetRuleID() string   { return "leverage_limit" }
func (c *LeverageLimitChecker) GetRuleName() string { return "杠杆限制" }
func (c *LeverageLimitChecker) GetRuleType() string { return "hard_constraint" }
func (c *LeverageLimitChecker) GetSeverity() string { return "critical" }
func (c *LeverageLimitChecker) GetDescription() string {
	return fmt.Sprintf("杠杆倍数不能超过 %d", c.maxLeverage)
}
func (c *LeverageLimitChecker) GetPriority() int { return 2 }

func (c *LeverageLimitChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	for i, action := range decision.Decisions {
		if !action.Success || action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		leverage := action.Leverage
		if leverage <= 0 {
			// 从parsedDecisions中查找
			for _, pd := range parsedDecisions {
				if pd.Symbol == action.Symbol && pd.Action == action.Action {
					leverage = pd.Leverage
					break
				}
			}
		}

		if leverage > c.maxLeverage {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("杠杆倍数 %d 超过限制 %d", leverage, c.maxLeverage),
				DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":       action.Symbol,
					"leverage":     leverage,
					"max_leverage": c.maxLeverage,
				},
			})
		}
	}

	return violations, nil
}

// StopLossDirectionChecker 止损方向检查器
type StopLossDirectionChecker struct{}

// NewStopLossDirectionChecker 创建止损方向检查器
func NewStopLossDirectionChecker() *StopLossDirectionChecker {
	return &StopLossDirectionChecker{}
}

func (c *StopLossDirectionChecker) GetRuleID() string   { return "stop_loss_direction" }
func (c *StopLossDirectionChecker) GetRuleName() string { return "止损止盈方向" }
func (c *StopLossDirectionChecker) GetRuleType() string { return "hard_constraint" }
func (c *StopLossDirectionChecker) GetSeverity() string { return "critical" }
func (c *StopLossDirectionChecker) GetDescription() string {
	return "止损/止盈方向必须正确：多仓 stop_loss < entry < take_profit，空仓 take_profit < entry < stop_loss"
}
func (c *StopLossDirectionChecker) GetPriority() int { return 3 }

func (c *StopLossDirectionChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	for i, action := range decision.Decisions {
		if !action.Success || action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		// 查找对应的parsedDecision
		var parsedDecision *Decision
		for j, pd := range parsedDecisions {
			if pd.Symbol == action.Symbol && pd.Action == action.Action {
				parsedDecision = &parsedDecisions[j]
				break
			}
		}

		if parsedDecision == nil {
			continue
		}

		stopLoss := parsedDecision.StopLoss
		takeProfit := parsedDecision.TakeProfit
		entryPrice := action.Price

		if stopLoss <= 0 || takeProfit <= 0 || entryPrice <= 0 {
			continue
		}

		// 检查方向
		if action.Action == "open_long" {
			// 多仓：stop_loss < entry < take_profit
			if stopLoss >= entryPrice || entryPrice >= takeProfit {
				violations = append(violations, review.Violation{
					RuleID:      c.GetRuleID(),
					RuleName:    c.GetRuleName(),
					Type:        c.GetRuleType(),
					Severity:    c.GetSeverity(),
					Description: fmt.Sprintf("多仓止损止盈方向错误：stop_loss (%.2f) < entry (%.2f) < take_profit (%.2f)", stopLoss, entryPrice, takeProfit),
					DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
					Details: map[string]interface{}{
						"symbol":      action.Symbol,
						"action":      action.Action,
						"entry_price": entryPrice,
						"stop_loss":   stopLoss,
						"take_profit": takeProfit,
					},
				})
			}
		} else if action.Action == "open_short" {
			// 空仓：take_profit < entry < stop_loss
			if takeProfit >= entryPrice || entryPrice >= stopLoss {
				violations = append(violations, review.Violation{
					RuleID:      c.GetRuleID(),
					RuleName:    c.GetRuleName(),
					Type:        c.GetRuleType(),
					Severity:    c.GetSeverity(),
					Description: fmt.Sprintf("空仓止损止盈方向错误：take_profit (%.2f) < entry (%.2f) < stop_loss (%.2f)", takeProfit, entryPrice, stopLoss),
					DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
					Details: map[string]interface{}{
						"symbol":      action.Symbol,
						"action":      action.Action,
						"entry_price": entryPrice,
						"stop_loss":   stopLoss,
						"take_profit": takeProfit,
					},
				})
			}
		}
	}

	return violations, nil
}

// ConfidenceRequirementChecker 置信度要求检查器
type ConfidenceRequirementChecker struct {
	minConfidence      int
	minConfidenceHighRisk int
}

// NewConfidenceRequirementChecker 创建置信度要求检查器
func NewConfidenceRequirementChecker(minConfidence, minConfidenceHighRisk int) *ConfidenceRequirementChecker {
	return &ConfidenceRequirementChecker{
		minConfidence:      minConfidence,
		minConfidenceHighRisk: minConfidenceHighRisk,
	}
}

func (c *ConfidenceRequirementChecker) GetRuleID() string   { return "confidence_requirement" }
func (c *ConfidenceRequirementChecker) GetRuleName() string { return "置信度要求" }
func (c *ConfidenceRequirementChecker) GetRuleType() string { return "hard_constraint" }
func (c *ConfidenceRequirementChecker) GetSeverity() string { return "high" }
func (c *ConfidenceRequirementChecker) GetDescription() string {
	return fmt.Sprintf("开仓置信度 ≥ %d，Sharpe < 0 时 ≥ %d", c.minConfidence, c.minConfidenceHighRisk)
}
func (c *ConfidenceRequirementChecker) GetPriority() int { return 4 }

func (c *ConfidenceRequirementChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	// 判断当前账户的Sharpe比率（这里简化处理，实际需要从历史数据计算）
	// Phase 1 暂不实现Sharpe判断，统一使用minConfidence
	sharpeRatio := 0.0 // TODO: 从历史数据计算Sharpe比率
	requiredConfidence := c.minConfidence
	if sharpeRatio < 0 {
		requiredConfidence = c.minConfidenceHighRisk
	}

	for i, action := range decision.Decisions {
		if !action.Success || action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		// 查找对应的parsedDecision
		var parsedDecision *Decision
		for j, pd := range parsedDecisions {
			if pd.Symbol == action.Symbol && pd.Action == action.Action {
				parsedDecision = &parsedDecisions[j]
				break
			}
		}

		if parsedDecision == nil {
			continue
		}

		confidence := parsedDecision.Confidence
		if confidence < requiredConfidence {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("开仓置信度 %d 低于要求 %d", confidence, requiredConfidence),
				DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":             action.Symbol,
					"confidence":         confidence,
					"required_confidence": requiredConfidence,
					"sharpe_ratio":       sharpeRatio,
				},
			})
		}
	}

	return violations, nil
}

// StopLossWidthChecker 止损宽度检查器
type StopLossWidthChecker struct {
	minStopLossWidthBTC float64 // BTC最小止损宽度 0.3%
	minStopLossWidthETH float64 // ETH最小止损宽度 0.5%
	minStopLossWidthOther float64 // 其他币种最小止损宽度 0.5%
}

// NewStopLossWidthChecker 创建止损宽度检查器
func NewStopLossWidthChecker() *StopLossWidthChecker {
	return &StopLossWidthChecker{
		minStopLossWidthBTC:   0.003, // 0.3%
		minStopLossWidthETH:   0.005, // 0.5%
		minStopLossWidthOther: 0.005, // 0.5%
	}
}

func (c *StopLossWidthChecker) GetRuleID() string   { return "stop_loss_width" }
func (c *StopLossWidthChecker) GetRuleName() string { return "止损宽度检查" }
func (c *StopLossWidthChecker) GetRuleType() string { return "risk_control" }
func (c *StopLossWidthChecker) GetSeverity() string { return "medium" }
func (c *StopLossWidthChecker) GetDescription() string {
	return "止损宽度要求：BTC最小0.3%，ETH最小0.5%，其他币种最小0.5%"
}
func (c *StopLossWidthChecker) GetPriority() int { return 5 }

func (c *StopLossWidthChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	for i, action := range decision.Decisions {
		if !action.Success || action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		// 查找对应的parsedDecision
		var parsedDecision *Decision
		for j, pd := range parsedDecisions {
			if pd.Symbol == action.Symbol && pd.Action == action.Action {
				parsedDecision = &parsedDecisions[j]
				break
			}
		}

		if parsedDecision == nil {
			continue
		}

		stopLoss := parsedDecision.StopLoss
		entryPrice := action.Price

		if stopLoss <= 0 || entryPrice <= 0 {
			continue
		}

		// 计算止损宽度（百分比）
		var stopLossWidth float64
		if action.Action == "open_long" {
			stopLossWidth = (entryPrice - stopLoss) / entryPrice
		} else {
			stopLossWidth = (stopLoss - entryPrice) / entryPrice
		}

		// 获取该币种的最小止损宽度要求
		minStopLossWidth := c.getMinStopLossWidth(action.Symbol)
		if stopLossWidth < minStopLossWidth {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("止损宽度 %.2f%% 低于要求 %.2f%%", stopLossWidth*100, minStopLossWidth*100),
				DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":              action.Symbol,
					"stop_loss_width":     stopLossWidth,
					"min_stop_loss_width": minStopLossWidth,
					"entry_price":         entryPrice,
					"stop_loss":           stopLoss,
				},
			})
		}
	}

	return violations, nil
}

func (c *StopLossWidthChecker) getMinStopLossWidth(symbol string) float64 {
	if symbol == "BTC" || symbol == "BTCUSDT" {
		return c.minStopLossWidthBTC
	}
	if symbol == "ETH" || symbol == "ETHUSDT" {
		return c.minStopLossWidthETH
	}
	return c.minStopLossWidthOther
}

// AccountRiskChecker 账户风险检查器
type AccountRiskChecker struct {
	maxAccountRisk float64 // 最大账户风险比例，例如 0.08 表示 8%
}

// NewAccountRiskChecker 创建账户风险检查器
func NewAccountRiskChecker(maxAccountRisk float64) *AccountRiskChecker {
	return &AccountRiskChecker{
		maxAccountRisk: maxAccountRisk,
	}
}

func (c *AccountRiskChecker) GetRuleID() string   { return "account_risk" }
func (c *AccountRiskChecker) GetRuleName() string { return "账户风险检查" }
func (c *AccountRiskChecker) GetRuleType() string { return "risk_control" }
func (c *AccountRiskChecker) GetSeverity() string { return "high" }
func (c *AccountRiskChecker) GetDescription() string {
	return fmt.Sprintf("账户风险要求：价格止损距离%% × 杠杆 ≤ %.0f%%", c.maxAccountRisk*100)
}
func (c *AccountRiskChecker) GetPriority() int { return 6 }

func (c *AccountRiskChecker) Check(decision *review.DecisionRecord, parsedDecisions []Decision) ([]review.Violation, error) {
	var violations []review.Violation

	totalBalance := decision.AccountState.TotalBalance
	if totalBalance <= 0 {
		return violations, nil
	}

	// 1. 检查已执行的决策
	for i, action := range decision.Decisions {
		if !action.Success || action.Action != "open_long" && action.Action != "open_short" {
			continue
		}

		// 查找对应的parsedDecision
		var parsedDecision *Decision
		for j, pd := range parsedDecisions {
			if pd.Symbol == action.Symbol && pd.Action == action.Action {
				parsedDecision = &parsedDecisions[j]
				break
			}
		}

		if parsedDecision == nil {
			continue
		}

		stopLoss := parsedDecision.StopLoss
		entryPrice := action.Price
		leverage := action.Leverage
		if leverage <= 0 {
			leverage = parsedDecision.Leverage
		}

		if stopLoss <= 0 || entryPrice <= 0 || leverage <= 0 {
			continue
		}

		// 计算价格止损距离（百分比）
		var stopLossDistancePct float64
		if action.Action == "open_long" {
			stopLossDistancePct = (entryPrice - stopLoss) / entryPrice
		} else {
			stopLossDistancePct = (stopLoss - entryPrice) / entryPrice
		}

		// 计算账户风险：价格止损距离% × 杠杆
		accountRisk := stopLossDistancePct * float64(leverage)

		if accountRisk > c.maxAccountRisk {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("账户风险 %.2f%% 超过限制 %.0f%%", accountRisk*100, c.maxAccountRisk*100),
				DecisionID:  fmt.Sprintf("%d_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":                 action.Symbol,
					"account_risk":           accountRisk,
					"max_account_risk":       c.maxAccountRisk,
					"stop_loss_distance_pct": stopLossDistancePct,
					"leverage":               leverage,
					"entry_price":            entryPrice,
					"stop_loss":              stopLoss,
					"executed":               true,
				},
			})
		}
	}

	// 2. 检查未执行的决策意图（识别到开单信号但最终没开单）
	for i, pd := range parsedDecisions {
		if pd.Action != "open_long" && pd.Action != "open_short" {
			continue
		}

		// 检查是否有对应的已执行action（已检查过的跳过）
		if findExecutedAction(pd.Symbol, pd.Action, decision.Decisions) != nil {
			continue
		}

		// 这是一个未执行的决策意图，检查是否违反规则
		stopLoss := pd.StopLoss
		if stopLoss <= 0 || pd.Leverage <= 0 {
			continue
		}

		// 获取估算的入场价格
		entryPrice := getEstimatedEntryPrice(pd.Symbol, pd.Action, stopLoss, pd.TakeProfit, decision.Positions)
		if entryPrice <= 0 {
			continue // 无法估算价格，跳过
		}

		// 计算价格止损距离（百分比）
		var stopLossDistancePct float64
		if pd.Action == "open_long" {
			stopLossDistancePct = (entryPrice - stopLoss) / entryPrice
		} else {
			stopLossDistancePct = (stopLoss - entryPrice) / entryPrice
		}

		// 计算账户风险：价格止损距离% × 杠杆
		accountRisk := stopLossDistancePct * float64(pd.Leverage)

		if accountRisk > c.maxAccountRisk {
			violations = append(violations, review.Violation{
				RuleID:      c.GetRuleID(),
				RuleName:    c.GetRuleName(),
				Type:        c.GetRuleType(),
				Severity:    c.GetSeverity(),
				Description: fmt.Sprintf("未执行决策：账户风险 %.2f%% 超过限制 %.0f%%", accountRisk*100, c.maxAccountRisk*100),
				DecisionID:  fmt.Sprintf("%d_intent_%d", decision.CycleNumber, i),
				Details: map[string]interface{}{
					"symbol":                 pd.Symbol,
					"account_risk":           accountRisk,
					"max_account_risk":       c.maxAccountRisk,
					"stop_loss_distance_pct": stopLossDistancePct,
					"leverage":               pd.Leverage,
					"entry_price":            entryPrice,
					"stop_loss":              stopLoss,
					"executed":               false,
					"is_intent":              true,
				},
			})
		}
	}

	return violations, nil
}


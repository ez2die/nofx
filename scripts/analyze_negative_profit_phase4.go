package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Phase 1数据结构（复用）
type Phase1Result struct {
	TraderID        string
	StartCycle      int
	EndCycle        int
	DecisionRecords []DecisionRecord
	MatchedTrades   []MatchedTrade
	BasicStats      BasicStatistics
}

type DecisionRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	CycleNumber  int       `json:"cycle_number"`
	CoTTrace     string    `json:"cot_trace"`
	DecisionJSON string    `json:"decision_json"`
	AccountState AccountSnapshot
	Positions    []PositionSnapshot
	Decisions    []DecisionAction
	Success      bool
}

type AccountSnapshot struct {
	TotalBalance float64 `json:"total_balance"`
}

type PositionSnapshot struct {
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	PositionAmt float64 `json:"position_amt"`
	EntryPrice  float64 `json:"entry_price"`
	MarkPrice   float64 `json:"mark_price"`
}

type DecisionAction struct {
	Action          string    `json:"action"`
	Symbol          string    `json:"symbol"`
	Quantity        float64   `json:"quantity"`
	Leverage        int       `json:"leverage"`
	Price           float64   `json:"price"`
	Timestamp       time.Time `json:"timestamp"`
	Success         bool      `json:"success"`
	IsAutoTriggered bool      `json:"is_auto_triggered"`
	WasStopLoss     bool      `json:"was_stop_loss"`
}

type MatchedTrade struct {
	CycleNumber     int
	OpenAction      DecisionAction
	CloseAction     DecisionAction
	OpenTrade       TradeRecord
	CloseTrade      TradeRecord
	PnL             float64
	Duration        time.Duration
	RiskRewardRatio float64
}

type TradeRecord struct {
	Symbol         string
	Action         string
	Side           string
	ExecutionPrice float64
	PnL            *float64
	Timestamp      time.Time
}

type BasicStatistics struct {
	TotalPnL float64
	WinRate  float64
}

// Phase 3数据结构（复用）
type Phase3Analysis struct {
	MarketBenchmark map[string]*MarketBenchmark
	RuleViolations  []*RuleViolation
	DecisionLogic   []*DecisionLogicAnalysis
	SymbolAnalyses  map[string]*SymbolAnalysis
	AIvsBenchmark   *AIBenchmarkComparison
	ReportPath      string
}

type RuleViolation struct {
	CycleNumber  int
	Symbol       string
	RuleID       string
	RuleName     string
	Severity     string
	Description  string
	Details      map[string]interface{}
	DecisionJSON string
}

type DecisionLogicAnalysis struct {
	CycleNumber  int
	Symbol       string
	Action       string
	CoTTrace     string
	DecisionJSON string
	LogicQuality string
	Issues       []string
	Strengths    []string
}

type SymbolAnalysis struct {
	Symbol            string
	TotalTrades       int
	AllStopLoss       bool
	AverageEntryPrice float64
	AverageExitPrice  float64
	MarketTrend       string
	EntryTimingIssues []string
	StopLossIssues    []string
	TotalPnL          float64
	WinRate           float64
}

type MarketBenchmark struct {
	Symbol        string
	BuyHoldReturn float64
	PriceChange   float64
}

type AIBenchmarkComparison struct {
	TotalTrades int
	AITotalPnL  float64
}

// Phase 4分析结果
type Phase4Analysis struct {
	PromptUnderstanding   PromptUnderstandingAnalysis
	PromptCompliance      PromptComplianceAnalysis
	PromptAmbiguity       PromptAmbiguityAnalysis
	PromptEffectiveness   PromptEffectivenessAnalysis
	PromptRecommendations []PromptRecommendation
	ReportPath            string
}

// Prompt理解度分析
type PromptUnderstandingAnalysis struct {
	RiskRewardMentioned    int     // CoTTrace中提到风险回报比的次数
	StopLossMentioned      int     // CoTTrace中提到止损的次数
	MarketNoiseMentioned   int     // CoTTrace中提到市场噪音的次数
	TrendAnalysisMentioned int     // CoTTrace中提到趋势分析的次数
	EntryTimingMentioned   int     // CoTTrace中提到入场时机的次数
	LeverageMentioned      int     // CoTTrace中提到杠杆的次数
	ConfidenceMentioned    int     // CoTTrace中提到置信度的次数
	TotalDecisions         int
	UnderstandingScore     float64 // 理解度评分 (0-100)
	MentionDetails         []MentionDetail
}

type MentionDetail struct {
	CycleNumber int
	Symbol      string
	MentionType string
	Context     string
}

// Prompt遵循度分析
type PromptComplianceAnalysis struct {
	RiskRewardCompliance  ComplianceStats
	StopLossCompliance    ComplianceStats
	LeverageCompliance    ComplianceStats
	ConfidenceCompliance  ComplianceStats
	EntryTimingCompliance ComplianceStats
	MarketNoiseCompliance ComplianceStats
}

type ComplianceStats struct {
	TotalChecks    int
	PassedChecks   int
	FailedChecks   int
	ComplianceRate float64
	Violations     []ComplianceViolation
}

type ComplianceViolation struct {
	CycleNumber  int
	Symbol       string
	Rule         string
	Expected     string
	Actual       string
	CoTTrace     string
	DecisionJSON string
}

// Prompt模糊性分析
type PromptAmbiguityAnalysis struct {
	AmbiguousRules     []AmbiguousRule
	ContradictoryRules []ContradictoryRule
	MissingClarity      []MissingClarity
}

type AmbiguousRule struct {
	RuleText      string
	AmbiguityType string // "vague", "conflicting", "unclear"
	Examples      []string
	Impact        string
	Frequency     int
}

type ContradictoryRule struct {
	Rule1         string
	Rule2         string
	Contradiction string
	Examples      []string
	Impact        string
}

type MissingClarity struct {
	Topic        string
	Issue        string
	Examples     []string
	SuggestedFix string
}

// Prompt有效性分析
type PromptEffectivenessAnalysis struct {
	EffectiveRules   []RuleEffectiveness
	IneffectiveRules  []RuleEffectiveness
	RuleImpact       map[string]float64
}

type RuleEffectiveness struct {
	RuleName           string
	ComplianceRate     float64
	AvgPnLWhenFollowed float64
	AvgPnLWhenViolated float64
	Impact             float64 // 遵循vs违反的收益差异
	SampleSize         int
}

// Prompt建议
type PromptRecommendation struct {
	Priority       string // "high", "medium", "low"
	Category       string // "clarity", "constraint", "guidance", "structure"
	Issue          string
	CurrentPrompt  string
	SuggestedChange string
	Rationale      string
	ExpectedImpact string
	Evidence       []string
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run analyze_negative_profit_phase4.go <phase1_result.json> <phase3_result.json> <prompt_file.txt>")
		fmt.Println("Example: go run analyze_negative_profit_phase4.go phase1_result_400_1892.json phase3_result_400_1892.json prompts/lean_optimized.txt")
		os.Exit(1)
	}

	phase1File := os.Args[1]
	phase3File := os.Args[2]
	promptFile := os.Args[3]

	fmt.Printf("\n=== Phase 4: Prompt问题诊断 ===\n\n")
	fmt.Printf("读取Phase 1结果: %s\n", phase1File)
	fmt.Printf("读取Phase 3结果: %s\n", phase3File)
	fmt.Printf("读取Prompt文件: %s\n\n", promptFile)

	// 1. 加载数据
	fmt.Println("📊 Step 1: 加载数据...")
	phase1Data, err := loadPhase1Data(phase1File)
	if err != nil {
		log.Fatalf("加载Phase 1数据失败: %v", err)
	}

	phase3Data, err := loadPhase3Data(phase3File)
	if err != nil {
		log.Fatalf("加载Phase 3数据失败: %v", err)
	}

	promptContent, err := ioutil.ReadFile(promptFile)
	if err != nil {
		log.Fatalf("读取Prompt文件失败: %v", err)
	}
	fmt.Printf("✅ 数据加载完成\n\n")

	// 2. Prompt理解度分析
	fmt.Println("📊 Step 2: 分析Prompt理解度...")
	understanding := analyzePromptUnderstanding(phase1Data.DecisionRecords)
	fmt.Printf("✅ 理解度评分: %.1f/100\n", understanding.UnderstandingScore)
	fmt.Printf("   - 风险回报比提及: %d/%d (%.1f%%)\n", understanding.RiskRewardMentioned, understanding.TotalDecisions,
		float64(understanding.RiskRewardMentioned)/float64(understanding.TotalDecisions)*100)
	fmt.Printf("   - 止损提及: %d/%d (%.1f%%)\n", understanding.StopLossMentioned, understanding.TotalDecisions,
		float64(understanding.StopLossMentioned)/float64(understanding.TotalDecisions)*100)
	fmt.Printf("   - 市场噪音提及: %d/%d (%.1f%%)\n", understanding.MarketNoiseMentioned, understanding.TotalDecisions,
		float64(understanding.MarketNoiseMentioned)/float64(understanding.TotalDecisions)*100)
	fmt.Printf("   - 趋势分析提及: %d/%d (%.1f%%)\n", understanding.TrendAnalysisMentioned, understanding.TotalDecisions,
		float64(understanding.TrendAnalysisMentioned)/float64(understanding.TotalDecisions)*100)
	fmt.Printf("   - 入场时机提及: %d/%d (%.1f%%)\n\n", understanding.EntryTimingMentioned, understanding.TotalDecisions,
		float64(understanding.EntryTimingMentioned)/float64(understanding.TotalDecisions)*100)

	// 3. Prompt遵循度分析
	fmt.Println("📊 Step 3: 分析Prompt遵循度...")
	compliance := analyzePromptCompliance(phase1Data.DecisionRecords, phase1Data.MatchedTrades, phase3Data.RuleViolations)
	fmt.Printf("✅ 遵循度分析完成\n")
	fmt.Printf("   - 风险回报比遵循率: %.1f%%\n", compliance.RiskRewardCompliance.ComplianceRate)
	fmt.Printf("   - 止损遵循率: %.1f%%\n", compliance.StopLossCompliance.ComplianceRate)
	fmt.Printf("   - 入场时机遵循率: %.1f%%\n\n", compliance.EntryTimingCompliance.ComplianceRate)

	// 4. Prompt模糊性分析
	fmt.Println("📊 Step 4: 分析Prompt模糊性...")
	ambiguity := analyzePromptAmbiguity(string(promptContent), phase1Data.DecisionRecords, phase3Data.RuleViolations)
	fmt.Printf("✅ 发现 %d 个模糊规则\n", len(ambiguity.AmbiguousRules))
	fmt.Printf("✅ 发现 %d 个矛盾规则\n", len(ambiguity.ContradictoryRules))
	fmt.Printf("✅ 发现 %d 个缺少清晰度的主题\n\n", len(ambiguity.MissingClarity))

	// 5. Prompt有效性分析
	fmt.Println("📊 Step 5: 分析Prompt有效性...")
	effectiveness := analyzePromptEffectiveness(phase1Data.MatchedTrades, phase3Data.RuleViolations, compliance)
	fmt.Printf("✅ 有效性分析完成\n")
	fmt.Printf("   - 有效规则数: %d\n", len(effectiveness.EffectiveRules))
	fmt.Printf("   - 无效规则数: %d\n\n", len(effectiveness.IneffectiveRules))

	// 6. 生成Prompt建议
	fmt.Println("📊 Step 6: 生成Prompt修正建议...")
	recommendations := generatePromptRecommendations(understanding, compliance, ambiguity, effectiveness, string(promptContent), phase1Data, phase3Data)
	fmt.Printf("✅ 生成了 %d 条建议\n\n", len(recommendations))

	// 7. 构建分析结果
	analysis := &Phase4Analysis{
		PromptUnderstanding:   understanding,
		PromptCompliance:      compliance,
		PromptAmbiguity:       ambiguity,
		PromptEffectiveness:   effectiveness,
		PromptRecommendations: recommendations,
	}

	// 8. 生成报告
	fmt.Println("📊 Step 7: 生成分析报告...")
	reportPath, err := generatePhase4Report(phase1Data, phase3Data, analysis, string(promptContent))
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	analysis.ReportPath = reportPath
	fmt.Printf("✅ 报告已生成: %s\n\n", reportPath)

	// 9. 保存结果
	jsonPath := fmt.Sprintf("phase4_result_%d_%d.json", phase1Data.StartCycle, phase1Data.EndCycle)
	jsonData, _ := json.MarshalIndent(analysis, "", "  ")
	ioutil.WriteFile(jsonPath, jsonData, 0644)
	fmt.Printf("✅ 结果已保存到: %s\n\n", jsonPath)

	// 10. 输出摘要
	printPhase4Summary(analysis)
}

func loadPhase1Data(filename string) (*Phase1Result, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result Phase1Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func loadPhase3Data(filename string) (*Phase3Analysis, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result Phase3Analysis
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func analyzePromptUnderstanding(decisions []DecisionRecord) PromptUnderstandingAnalysis {
	analysis := PromptUnderstandingAnalysis{
		TotalDecisions: len(decisions),
		MentionDetails: []MentionDetail{},
	}

	// 关键词模式
	riskRewardPattern := regexp.MustCompile(`(?i)(风险回报|risk.*reward|reward.*risk|3:1|3\.0:1|风险.*回报|风险回报比|risk.*reward.*ratio)`)
	stopLossPattern := regexp.MustCompile(`(?i)(止损|stop.*loss|stop_loss|stop loss)`)
	marketNoisePattern := regexp.MustCompile(`(?i)(市场噪音|market.*noise|波动性|volatility|噪音|noise|K线.*波动|波动性)`)
	trendPattern := regexp.MustCompile(`(?i)(趋势|trend|EMA|方向|direction|趋势.*方向|趋势.*分析)`)
	entryTimingPattern := regexp.MustCompile(`(?i)(入场时机|entry.*timing|timing|时机|信号.*持续|时间.*稳定|持续时间|信号.*稳定)`)
	leveragePattern := regexp.MustCompile(`(?i)(杠杆|leverage|杠杆.*敏感|杠杆.*放大|账户)`)
	confidencePattern := regexp.MustCompile(`(?i)(置信度|confidence|确信度|信心度)`)

	for _, decision := range decisions {
		if decision.CoTTrace == "" {
			continue
		}

		if riskRewardPattern.MatchString(decision.CoTTrace) {
			analysis.RiskRewardMentioned++
			// 提取上下文
			context := extractContext(decision.CoTTrace, riskRewardPattern, 100)
			analysis.MentionDetails = append(analysis.MentionDetails, MentionDetail{
				CycleNumber: decision.CycleNumber,
				MentionType:  "risk_reward",
				Context:      context,
			})
		}
		if stopLossPattern.MatchString(decision.CoTTrace) {
			analysis.StopLossMentioned++
		}
		if marketNoisePattern.MatchString(decision.CoTTrace) {
			analysis.MarketNoiseMentioned++
		}
		if trendPattern.MatchString(decision.CoTTrace) {
			analysis.TrendAnalysisMentioned++
		}
		if entryTimingPattern.MatchString(decision.CoTTrace) {
			analysis.EntryTimingMentioned++
		}
		if leveragePattern.MatchString(decision.CoTTrace) {
			analysis.LeverageMentioned++
		}
		if confidencePattern.MatchString(decision.CoTTrace) {
			analysis.ConfidenceMentioned++
		}
	}

	// 计算理解度评分
	if analysis.TotalDecisions > 0 {
		score := 0.0
		weights := []float64{20, 15, 15, 15, 15, 10, 10} // 权重分配
		mentions := []int{
			analysis.RiskRewardMentioned,
			analysis.StopLossMentioned,
			analysis.MarketNoiseMentioned,
			analysis.TrendAnalysisMentioned,
			analysis.EntryTimingMentioned,
			analysis.LeverageMentioned,
			analysis.ConfidenceMentioned,
		}

		for i, mention := range mentions {
			rate := float64(mention) / float64(analysis.TotalDecisions)
			score += rate * weights[i]
		}
		analysis.UnderstandingScore = math.Min(100, score)
	}

	return analysis
}

func extractContext(text string, pattern *regexp.Regexp, contextLen int) string {
	matches := pattern.FindStringIndex(text)
	if len(matches) == 0 {
		return ""
	}
	start := matches[0]
	contextStart := math.Max(0, float64(start-contextLen))
	contextEnd := math.Min(float64(len(text)), float64(start+contextLen))
	return text[int(contextStart):int(contextEnd)]
}

func analyzePromptCompliance(decisions []DecisionRecord, trades []MatchedTrade, violations []*RuleViolation) PromptComplianceAnalysis {
	// 1. 风险回报比遵循度
	riskRewardStats := ComplianceStats{}
	violationMap := make(map[int]map[string]bool) // cycle -> symbol -> has violation
	for _, v := range violations {
		if v.RuleID == "risk_reward_ratio" {
			if violationMap[v.CycleNumber] == nil {
				violationMap[v.CycleNumber] = make(map[string]bool)
			}
			violationMap[v.CycleNumber][v.Symbol] = true
			riskRewardStats.FailedChecks++
			riskRewardStats.Violations = append(riskRewardStats.Violations, ComplianceViolation{
				CycleNumber:  v.CycleNumber,
				Symbol:       v.Symbol,
				Rule:         "风险回报比",
				Expected:     "≥ 3.0:1",
				Actual:       v.Description,
				DecisionJSON: v.DecisionJSON,
			})
		}
	}

	// 统计所有开仓决策
	for _, decision := range decisions {
		if decision.DecisionJSON == "" {
			continue
		}

		var decisionArray []struct {
			Symbol     string  `json:"symbol"`
			Action     string  `json:"action"`
			StopLoss   float64 `json:"stop_loss"`
			TakeProfit float64 `json:"take_profit"`
		}

		if err := json.Unmarshal([]byte(decision.DecisionJSON), &decisionArray); err != nil {
			continue
		}

		for _, d := range decisionArray {
			if d.Action == "open_long" || d.Action == "open_short" {
				riskRewardStats.TotalChecks++
				if violationMap[decision.CycleNumber] != nil && violationMap[decision.CycleNumber][d.Symbol] {
					// 已经在violations中统计过了
				} else {
					riskRewardStats.PassedChecks++
				}
			}
		}
	}

	if riskRewardStats.TotalChecks > 0 {
		riskRewardStats.ComplianceRate = float64(riskRewardStats.PassedChecks) / float64(riskRewardStats.TotalChecks) * 100
	}

	// 2. 止损遵循度（检查是否考虑了市场噪音最小值）
	stopLossStats := ComplianceStats{}
	for _, trade := range trades {
		if trade.OpenAction.Action != "open_long" && trade.OpenAction.Action != "open_short" {
			continue
		}

		stopLossStats.TotalChecks++

		// 查找对应的决策JSON
		var decisionJSON struct {
			StopLoss float64 `json:"stop_loss"`
		}
		for _, decision := range decisions {
			if decision.CycleNumber == trade.CycleNumber && decision.DecisionJSON != "" {
				var decisionArray []struct {
					Symbol   string  `json:"symbol"`
					Action   string  `json:"action"`
					StopLoss float64 `json:"stop_loss"`
				}
				if err := json.Unmarshal([]byte(decision.DecisionJSON), &decisionArray); err == nil {
					for _, d := range decisionArray {
						if d.Symbol == trade.OpenTrade.Symbol && d.Action == trade.OpenAction.Action {
							decisionJSON.StopLoss = d.StopLoss
							break
						}
					}
				}
				break
			}
		}

		// 检查止损是否过窄（基于市场噪音最小值）
		entryPrice := trade.OpenTrade.ExecutionPrice
		if entryPrice > 0 && decisionJSON.StopLoss > 0 {
			stopLossPct := math.Abs(decisionJSON.StopLoss-entryPrice) / entryPrice * 100

			// BTC/ETH最小0.3-0.5%，大市值山寨币最小0.5-0.7%，其他最小0.7-1%
			var minStopLoss float64
			symbol := trade.OpenTrade.Symbol
			if symbol == "BTCUSDT" || symbol == "ETHUSDT" {
				minStopLoss = 0.3
			} else if strings.Contains(symbol, "USDT") {
				minStopLoss = 0.5
			} else {
				minStopLoss = 0.7
			}

			if stopLossPct < minStopLoss {
				stopLossStats.FailedChecks++
				stopLossStats.Violations = append(stopLossStats.Violations, ComplianceViolation{
					CycleNumber: trade.CycleNumber,
					Symbol:      trade.OpenTrade.Symbol,
					Rule:        "市场噪音最小值",
					Expected:    fmt.Sprintf("≥ %.1f%%", minStopLoss),
					Actual:      fmt.Sprintf("%.2f%%", stopLossPct),
				})
			} else {
				stopLossStats.PassedChecks++
			}
		}
	}

	if stopLossStats.TotalChecks > 0 {
		stopLossStats.ComplianceRate = float64(stopLossStats.PassedChecks) / float64(stopLossStats.TotalChecks) * 100
	}

	// 3. 入场时机遵循度（检查是否检查了信号持续时间）
	entryTimingStats := ComplianceStats{}
	for _, decision := range decisions {
		if decision.CoTTrace == "" {
			continue
		}

		// 检查是否有开仓决策
		hasOpenDecision := false
		if decision.DecisionJSON != "" {
			var decisionArray []struct {
				Action string `json:"action"`
			}
			if err := json.Unmarshal([]byte(decision.DecisionJSON), &decisionArray); err == nil {
				for _, d := range decisionArray {
					if d.Action == "open_long" || d.Action == "open_short" {
						hasOpenDecision = true
						break
					}
				}
			}
		}

		if hasOpenDecision {
			entryTimingStats.TotalChecks++

			// 检查CoTTrace中是否提到信号持续时间或时间稳定性
			timingKeywords := []string{"持续", "稳定", "时间", "分钟", "信号.*持续", "时间.*稳定"}
			mentioned := false
			for _, keyword := range timingKeywords {
				pattern := regexp.MustCompile(`(?i)` + keyword)
				if pattern.MatchString(decision.CoTTrace) {
					mentioned = true
					break
				}
			}

			if mentioned {
				entryTimingStats.PassedChecks++
			} else {
				entryTimingStats.FailedChecks++
				cotTrace := decision.CoTTrace
				if len(cotTrace) > 200 {
					cotTrace = cotTrace[:200]
				}
				entryTimingStats.Violations = append(entryTimingStats.Violations, ComplianceViolation{
					CycleNumber: decision.CycleNumber,
					Rule:        "入场时机检查",
					Expected:    "检查信号持续时间或时间稳定性",
					Actual:      "未在CoTTrace中提及",
					CoTTrace:    cotTrace,
				})
			}
		}
	}

	if entryTimingStats.TotalChecks > 0 {
		entryTimingStats.ComplianceRate = float64(entryTimingStats.PassedChecks) / float64(entryTimingStats.TotalChecks) * 100
	}

	// 4. 杠杆遵循度（简化：检查是否在合理范围内）
	leverageStats := ComplianceStats{}
	for _, trade := range trades {
		if trade.OpenAction.Leverage > 0 {
			leverageStats.TotalChecks++
			// 检查杠杆是否在合理范围内（1-25x）
			if trade.OpenAction.Leverage >= 1 && trade.OpenAction.Leverage <= 25 {
				leverageStats.PassedChecks++
			} else {
				leverageStats.FailedChecks++
			}
		}
	}

	if leverageStats.TotalChecks > 0 {
		leverageStats.ComplianceRate = float64(leverageStats.PassedChecks) / float64(leverageStats.TotalChecks) * 100
	}

	// 5. 置信度遵循度
	confidenceStats := ComplianceStats{}
	for _, decision := range decisions {
		if decision.DecisionJSON == "" {
			continue
		}

		var decisionArray []struct {
			Action     string `json:"action"`
			Confidence int    `json:"confidence"`
		}

		if err := json.Unmarshal([]byte(decision.DecisionJSON), &decisionArray); err != nil {
			continue
		}

		for _, d := range decisionArray {
			if d.Action == "open_long" || d.Action == "open_short" {
				confidenceStats.TotalChecks++
				if d.Confidence >= 70 {
					confidenceStats.PassedChecks++
				} else {
					confidenceStats.FailedChecks++
				}
			}
		}
	}

	if confidenceStats.TotalChecks > 0 {
		confidenceStats.ComplianceRate = float64(confidenceStats.PassedChecks) / float64(confidenceStats.TotalChecks) * 100
	}

	// 6. 市场噪音遵循度（与止损遵循度合并）
	marketNoiseStats := stopLossStats

	return PromptComplianceAnalysis{
		RiskRewardCompliance:  riskRewardStats,
		StopLossCompliance:    stopLossStats,
		LeverageCompliance:    leverageStats,
		ConfidenceCompliance:  confidenceStats,
		EntryTimingCompliance: entryTimingStats,
		MarketNoiseCompliance: marketNoiseStats,
	}
}

func analyzePromptAmbiguity(promptContent string, decisions []DecisionRecord, violations []*RuleViolation) PromptAmbiguityAnalysis {
	_ = violations // 暂时未使用
	ambiguity := PromptAmbiguityAnalysis{}

	// 1. 识别模糊规则
	// 检查"参考原则"vs"硬约束"的混淆
	if strings.Contains(promptContent, "参考原则") && strings.Contains(promptContent, "硬约束") {
		// 查找"参考原则"的上下文
		refPattern := regexp.MustCompile(`(?i)(参考原则|参考案例).*?[\n\r]`)
		refMatches := refPattern.FindAllString(promptContent, -1)
		if len(refMatches) > 0 {
			examples := refMatches
			if len(examples) > 3 {
				examples = examples[:3]
			}
			ambiguity.AmbiguousRules = append(ambiguity.AmbiguousRules, AmbiguousRule{
				RuleText:      "参考原则 vs 硬约束",
				AmbiguityType: "unclear",
				Examples:      examples,
				Impact:        "AI可能将参考原则误认为是硬约束，或反之",
				Frequency:     len(refMatches),
			})
		}
	}

	// 检查"建议"vs"要求"的混淆
	suggestionPattern := regexp.MustCompile(`(?i)(建议|推荐|可以|should|recommend)`)
	requirementPattern := regexp.MustCompile(`(?i)(必须|要求|硬约束|mandatory|required|must)`)
	if suggestionPattern.MatchString(promptContent) && requirementPattern.MatchString(promptContent) {
		suggestionMatches := suggestionPattern.FindAllString(promptContent, -1)
		ambiguity.AmbiguousRules = append(ambiguity.AmbiguousRules, AmbiguousRule{
			RuleText:      "建议 vs 要求",
			AmbiguityType: "vague",
			Examples:      []string{"Prompt中同时包含'建议'和'必须'的表述，可能导致AI混淆"},
			Impact:        "AI可能将建议误认为是要求，或忽略必须遵守的规则",
			Frequency:     len(suggestionMatches),
		})
	}

	// 检查数值范围的模糊性
	rangePattern := regexp.MustCompile(`(\d+\.?\d*)\s*[-~]\s*(\d+\.?\d*)`)
	rangeMatches := rangePattern.FindAllString(promptContent, -1)
	if len(rangeMatches) > 0 {
		examples := rangeMatches
		if len(examples) > 5 {
			examples = examples[:5]
		}
		ambiguity.AmbiguousRules = append(ambiguity.AmbiguousRules, AmbiguousRule{
			RuleText:      "数值范围模糊",
			AmbiguityType: "vague",
			Examples:      examples,
			Impact:        "AI可能选择范围边界值，导致不符合预期",
			Frequency:     len(rangeMatches),
		})
	}

	// 2. 识别矛盾规则
	// 检查"耐心等待"vs"快速响应"的矛盾
	if strings.Contains(promptContent, "耐心等待") && strings.Contains(promptContent, "快速响应") {
		ambiguity.ContradictoryRules = append(ambiguity.ContradictoryRules, ContradictoryRule{
			Rule1:         "耐心等待信号稳定",
			Rule2:         "快速响应入场假设失效",
			Contradiction: "两个规则可能在不同情况下同时适用，但AI可能混淆何时应用哪个规则",
			Examples:      []string{"在弱势市场中要求耐心等待，但入场假设失效时要求快速响应"},
			Impact:        "AI可能在不该等待时等待，或在不该快速响应时快速响应",
		})
	}

	// 检查"趋势优先"vs"多指标综合判断"的矛盾
	if strings.Contains(promptContent, "趋势优先") && strings.Contains(promptContent, "多指标综合判断") {
		ambiguity.ContradictoryRules = append(ambiguity.ContradictoryRules, ContradictoryRule{
			Rule1:         "趋势优先",
			Rule2:         "多指标综合判断",
			Contradiction: "当趋势与其他指标冲突时，AI可能不确定应该优先哪个",
			Examples:      []string{"趋势向下但RSI超卖时，应该优先趋势还是RSI信号"},
			Impact:        "AI可能做出不一致的决策",
		})
	}

	// 3. 识别缺少清晰度的主题
	// 检查入场时机判断的清晰度
	if !strings.Contains(promptContent, "如何判断入场时机") && !strings.Contains(promptContent, "入场时机判断标准") {
		ambiguity.MissingClarity = append(ambiguity.MissingClarity, MissingClarity{
			Topic:        "入场时机判断",
			Issue:        "Prompt中缺少明确的入场时机判断标准",
			Examples:     []string{"AI可能依赖单一指标而非综合判断"},
			SuggestedFix: "添加明确的入场时机判断流程图或检查清单",
		})
	}

	// 检查止损设置的清晰度
	if !strings.Contains(promptContent, "止损设置计算") && !strings.Contains(promptContent, "如何设置止损") {
		ambiguity.MissingClarity = append(ambiguity.MissingClarity, MissingClarity{
			Topic:        "止损设置方法",
			Issue:        "Prompt中缺少明确的止损设置计算方法",
			Examples:     []string{"AI可能设置过窄的止损，导致频繁触发"},
			SuggestedFix: "添加止损设置的步骤化指导",
		})
	}

	return ambiguity
}

func analyzePromptEffectiveness(trades []MatchedTrade, violations []*RuleViolation, compliance PromptComplianceAnalysis) PromptEffectivenessAnalysis {
	effectiveness := PromptEffectivenessAnalysis{
		RuleImpact: make(map[string]float64),
	}

	// 创建违反映射
	violationMap := make(map[int]map[string]bool) // cycle -> symbol -> has violation
	for _, v := range violations {
		if violationMap[v.CycleNumber] == nil {
			violationMap[v.CycleNumber] = make(map[string]bool)
		}
		violationMap[v.CycleNumber][v.Symbol] = true
	}

	// 1. 风险回报比有效性
	riskRewardFollowed := []float64{}
	riskRewardViolated := []float64{}

	for _, trade := range trades {
		if trade.OpenAction.Action == "open_long" || trade.OpenAction.Action == "open_short" {
			if violationMap[trade.CycleNumber] != nil && violationMap[trade.CycleNumber][trade.OpenTrade.Symbol] {
				riskRewardViolated = append(riskRewardViolated, trade.PnL)
			} else {
				riskRewardFollowed = append(riskRewardFollowed, trade.PnL)
			}
		}
	}

	if len(riskRewardFollowed) > 0 && len(riskRewardViolated) > 0 {
		avgFollowed := average(riskRewardFollowed)
		avgViolated := average(riskRewardViolated)
		impact := avgFollowed - avgViolated

		effectiveness.EffectiveRules = append(effectiveness.EffectiveRules, RuleEffectiveness{
			RuleName:           "风险回报比≥3.0:1",
			ComplianceRate:     compliance.RiskRewardCompliance.ComplianceRate,
			AvgPnLWhenFollowed: avgFollowed,
			AvgPnLWhenViolated: avgViolated,
			Impact:             impact,
			SampleSize:         len(riskRewardFollowed) + len(riskRewardViolated),
		})
		effectiveness.RuleImpact["risk_reward"] = impact
	}

	return effectiveness
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func generatePromptRecommendations(understanding PromptUnderstandingAnalysis, compliance PromptComplianceAnalysis,
	ambiguity PromptAmbiguityAnalysis, effectiveness PromptEffectivenessAnalysis, promptContent string,
	phase1Data *Phase1Result, phase3Data *Phase3Analysis) []PromptRecommendation {

	var recommendations []PromptRecommendation

	// 1. 风险回报比理解度低
	if understanding.TotalDecisions > 0 && float64(understanding.RiskRewardMentioned)/float64(understanding.TotalDecisions) < 0.5 {
		recommendations = append(recommendations, PromptRecommendation{
			Priority:        "high",
			Category:        "clarity",
			Issue:           "风险回报比在CoTTrace中提及率低",
			CurrentPrompt:   "风险回报比要求：绝对最小值3.0:1",
			SuggestedChange: "在Prompt开头和决策流程中多次强调风险回报比，要求AI在CoTTrace中明确计算并说明",
			Rationale:       fmt.Sprintf("只有%.1f%%的决策在CoTTrace中提到了风险回报比", float64(understanding.RiskRewardMentioned)/float64(understanding.TotalDecisions)*100),
			ExpectedImpact:  "提高AI对风险回报比的重视，减少违反次数",
			Evidence:        []string{fmt.Sprintf("风险回报比违反: %d次", len(compliance.RiskRewardCompliance.Violations))},
		})
	}

	// 2. 市场噪音理解度低
	if understanding.TotalDecisions > 0 && float64(understanding.MarketNoiseMentioned)/float64(understanding.TotalDecisions) < 0.3 {
		recommendations = append(recommendations, PromptRecommendation{
			Priority:        "high",
			Category:        "guidance",
			Issue:           "市场噪音在CoTTrace中提及率低",
			CurrentPrompt:   "市场噪音考虑：止损应≥2-3×典型K线内波动性",
			SuggestedChange: "在止损设置部分添加强制检查清单，要求AI在设置止损时明确说明是否考虑了市场噪音",
			Rationale:       fmt.Sprintf("只有%.1f%%的决策提到了市场噪音，导致止损设置过窄", float64(understanding.MarketNoiseMentioned)/float64(understanding.TotalDecisions)*100),
			ExpectedImpact:  "减少因市场噪音导致的虚假止损触发",
			Evidence:        []string{fmt.Sprintf("止损遵循率: %.1f%%", compliance.StopLossCompliance.ComplianceRate)},
		})
	}

	// 3. 入场时机检查不足
	if compliance.EntryTimingCompliance.ComplianceRate < 50 {
		recommendations = append(recommendations, PromptRecommendation{
			Priority:        "high",
			Category:        "constraint",
			Issue:           "入场时机检查不足",
			CurrentPrompt:   "信号持续时间验证：偏好信号已经持续了至少5-10分钟",
			SuggestedChange: "将入场时机检查提升为硬约束，要求AI在CoTTrace中明确说明信号持续时间，否则不允许开仓",
			Rationale:       fmt.Sprintf("只有%.1f%%的开仓决策检查了信号持续时间", compliance.EntryTimingCompliance.ComplianceRate),
			ExpectedImpact:  "减少在横盘/下跌趋势中做多的问题",
			Evidence:        []string{"Phase 3显示入场时机不佳的交易占比极高"},
		})
	}

	// 4. 参考原则vs硬约束混淆
	if len(ambiguity.AmbiguousRules) > 0 {
		for _, rule := range ambiguity.AmbiguousRules {
			if rule.AmbiguityType == "unclear" && strings.Contains(rule.RuleText, "参考原则") {
				recommendations = append(recommendations, PromptRecommendation{
					Priority:        "medium",
					Category:        "clarity",
					Issue:           "参考原则与硬约束混淆",
					CurrentPrompt:   "参考原则（非强制规则）",
					SuggestedChange: "明确区分硬约束和参考原则，使用不同的格式（如硬约束用【】标记，参考原则用（）标记）",
					Rationale:       "AI可能将参考原则误认为是硬约束，导致过度遵守或忽略",
					ExpectedImpact:  "提高AI对规则层级的理解",
					Evidence:        rule.Examples,
				})
			}
		}
	}

	// 5. 矛盾规则
	if len(ambiguity.ContradictoryRules) > 0 {
		for _, rule := range ambiguity.ContradictoryRules {
			recommendations = append(recommendations, PromptRecommendation{
				Priority:        "high",
				Category:        "structure",
				Issue:           fmt.Sprintf("规则矛盾: %s vs %s", rule.Rule1, rule.Rule2),
				CurrentPrompt:   "两个规则同时存在但可能冲突",
				SuggestedChange: fmt.Sprintf("明确说明何时应用哪个规则，添加决策树或优先级说明"),
				Rationale:       rule.Contradiction,
				ExpectedImpact:  "减少AI决策的不一致性",
				Evidence:        rule.Examples,
			})
		}
	}

	// 6. 风险回报比违反率高
	if compliance.RiskRewardCompliance.ComplianceRate < 90 {
		recommendations = append(recommendations, PromptRecommendation{
			Priority:        "high",
			Category:        "constraint",
			Issue:           "风险回报比违反率高",
			CurrentPrompt:   "风险回报比要求：绝对最小值3.0:1",
			SuggestedChange: "在预执行检查清单中将风险回报比检查放在第一位，并添加'如果风险回报比<3.0:1，立即拒绝，不要寻找例外'的明确说明",
			Rationale:       fmt.Sprintf("风险回报比遵循率只有%.1f%%，存在%d次违反", compliance.RiskRewardCompliance.ComplianceRate, len(compliance.RiskRewardCompliance.Violations)),
			ExpectedImpact:  "减少风险回报比违反次数",
			Evidence:        []string{fmt.Sprintf("违反次数: %d", len(compliance.RiskRewardCompliance.Violations))},
		})
	}

	// 7. 趋势分析优先性不明确
	if understanding.TotalDecisions > 0 && float64(understanding.TrendAnalysisMentioned)/float64(understanding.TotalDecisions) < 0.6 {
		recommendations = append(recommendations, PromptRecommendation{
			Priority:        "medium",
			Category:        "guidance",
			Issue:           "趋势分析在决策中不够突出",
			CurrentPrompt:   "趋势优先（参考原则，非强制规则）",
			SuggestedChange: "将趋势分析提升为决策流程的第一步，要求AI在分析任何信号前先判断趋势方向",
			Rationale:       "Phase 3显示在下跌/横盘趋势中大量做多，说明趋势分析不够重视",
			ExpectedImpact:  "减少逆势交易",
			Evidence:        []string{"Phase 3显示入场时机不佳的交易占比极高"},
		})
	}

	return recommendations
}

func generatePhase4Report(phase1Data *Phase1Result, phase3Data *Phase3Analysis, analysis *Phase4Analysis, promptContent string) (string, error) {
	reportPath := fmt.Sprintf("phase4_report_%d_%d.md", phase1Data.StartCycle, phase1Data.EndCycle)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Phase 4: Prompt问题诊断报告\n\n")
	fmt.Fprintf(file, "**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**基于Phase 1和Phase 3数据**: Cycle %d - %d\n\n", phase1Data.StartCycle, phase1Data.EndCycle)

	// 1. Prompt理解度分析
	fmt.Fprintf(file, "## 1. Prompt理解度分析\n\n")
	u := analysis.PromptUnderstanding
	fmt.Fprintf(file, "### 1.1 总体理解度评分\n\n")
	fmt.Fprintf(file, "- **理解度评分**: %.1f/100\n\n", u.UnderstandingScore)

	fmt.Fprintf(file, "### 1.2 关键规则提及率\n\n")
	fmt.Fprintf(file, "| 规则 | 提及次数 | 提及率 |\n")
	fmt.Fprintf(file, "|------|----------|--------|\n")
	if u.TotalDecisions > 0 {
		fmt.Fprintf(file, "| 风险回报比 | %d | %.1f%% |\n", u.RiskRewardMentioned, float64(u.RiskRewardMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 止损 | %d | %.1f%% |\n", u.StopLossMentioned, float64(u.StopLossMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 市场噪音 | %d | %.1f%% |\n", u.MarketNoiseMentioned, float64(u.MarketNoiseMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 趋势分析 | %d | %.1f%% |\n", u.TrendAnalysisMentioned, float64(u.TrendAnalysisMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 入场时机 | %d | %.1f%% |\n", u.EntryTimingMentioned, float64(u.EntryTimingMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 杠杆 | %d | %.1f%% |\n", u.LeverageMentioned, float64(u.LeverageMentioned)/float64(u.TotalDecisions)*100)
		fmt.Fprintf(file, "| 置信度 | %d | %.1f%% |\n\n", u.ConfidenceMentioned, float64(u.ConfidenceMentioned)/float64(u.TotalDecisions)*100)
	}

	// 2. Prompt遵循度分析
	fmt.Fprintf(file, "## 2. Prompt遵循度分析\n\n")
	c := analysis.PromptCompliance
	fmt.Fprintf(file, "| 规则 | 总检查数 | 通过数 | 失败数 | 遵循率 |\n")
	fmt.Fprintf(file, "|------|----------|--------|--------|--------|\n")
	fmt.Fprintf(file, "| 风险回报比 | %d | %d | %d | %.1f%% |\n",
		c.RiskRewardCompliance.TotalChecks, c.RiskRewardCompliance.PassedChecks,
		c.RiskRewardCompliance.FailedChecks, c.RiskRewardCompliance.ComplianceRate)
	fmt.Fprintf(file, "| 止损设置 | %d | %d | %d | %.1f%% |\n",
		c.StopLossCompliance.TotalChecks, c.StopLossCompliance.PassedChecks,
		c.StopLossCompliance.FailedChecks, c.StopLossCompliance.ComplianceRate)
	fmt.Fprintf(file, "| 入场时机 | %d | %d | %d | %.1f%% |\n",
		c.EntryTimingCompliance.TotalChecks, c.EntryTimingCompliance.PassedChecks,
		c.EntryTimingCompliance.FailedChecks, c.EntryTimingCompliance.ComplianceRate)
	fmt.Fprintf(file, "| 杠杆 | %d | %d | %d | %.1f%% |\n",
		c.LeverageCompliance.TotalChecks, c.LeverageCompliance.PassedChecks,
		c.LeverageCompliance.FailedChecks, c.LeverageCompliance.ComplianceRate)
	fmt.Fprintf(file, "| 置信度 | %d | %d | %d | %.1f%% |\n\n",
		c.ConfidenceCompliance.TotalChecks, c.ConfidenceCompliance.PassedChecks,
		c.ConfidenceCompliance.FailedChecks, c.ConfidenceCompliance.ComplianceRate)

	// 3. Prompt模糊性分析
	fmt.Fprintf(file, "## 3. Prompt模糊性分析\n\n")
	a := analysis.PromptAmbiguity

	if len(a.AmbiguousRules) > 0 {
		fmt.Fprintf(file, "### 3.1 模糊规则\n\n")
		for i, rule := range a.AmbiguousRules {
			fmt.Fprintf(file, "#### 模糊规则 %d: %s\n\n", i+1, rule.RuleText)
			fmt.Fprintf(file, "- **模糊类型**: %s\n", rule.AmbiguityType)
			fmt.Fprintf(file, "- **出现频率**: %d次\n", rule.Frequency)
			fmt.Fprintf(file, "- **影响**: %s\n", rule.Impact)
			if len(rule.Examples) > 0 {
				fmt.Fprintf(file, "- **示例**:\n")
				for _, ex := range rule.Examples {
					fmt.Fprintf(file, "  - %s\n", ex)
				}
			}
			fmt.Fprintf(file, "\n")
		}
	}

	if len(a.ContradictoryRules) > 0 {
		fmt.Fprintf(file, "### 3.2 矛盾规则\n\n")
		for i, rule := range a.ContradictoryRules {
			fmt.Fprintf(file, "#### 矛盾 %d: %s vs %s\n\n", i+1, rule.Rule1, rule.Rule2)
			fmt.Fprintf(file, "- **矛盾描述**: %s\n", rule.Contradiction)
			fmt.Fprintf(file, "- **影响**: %s\n", rule.Impact)
			if len(rule.Examples) > 0 {
				fmt.Fprintf(file, "- **示例**:\n")
				for _, ex := range rule.Examples {
					fmt.Fprintf(file, "  - %s\n", ex)
				}
			}
			fmt.Fprintf(file, "\n")
		}
	}

	if len(a.MissingClarity) > 0 {
		fmt.Fprintf(file, "### 3.3 缺少清晰度的主题\n\n")
		for i, clarity := range a.MissingClarity {
			fmt.Fprintf(file, "#### 主题 %d: %s\n\n", i+1, clarity.Topic)
			fmt.Fprintf(file, "- **问题**: %s\n", clarity.Issue)
			fmt.Fprintf(file, "- **建议修正**: %s\n", clarity.SuggestedFix)
			if len(clarity.Examples) > 0 {
				fmt.Fprintf(file, "- **示例**:\n")
				for _, ex := range clarity.Examples {
					fmt.Fprintf(file, "  - %s\n", ex)
				}
			}
			fmt.Fprintf(file, "\n")
		}
	}

	// 4. Prompt有效性分析
	fmt.Fprintf(file, "## 4. Prompt有效性分析\n\n")
	e := analysis.PromptEffectiveness

	if len(e.EffectiveRules) > 0 {
		fmt.Fprintf(file, "### 4.1 有效规则\n\n")
		fmt.Fprintf(file, "| 规则 | 遵循率 | 遵循时平均收益 | 违反时平均收益 | 收益差异 |\n")
		fmt.Fprintf(file, "|------|--------|----------------|----------------|----------|\n")
		for _, rule := range e.EffectiveRules {
			fmt.Fprintf(file, "| %s | %.1f%% | %.2f USDT | %.2f USDT | %.2f USDT |\n",
				rule.RuleName, rule.ComplianceRate, rule.AvgPnLWhenFollowed,
				rule.AvgPnLWhenViolated, rule.Impact)
		}
		fmt.Fprintf(file, "\n")
	}

	// 5. Prompt修正建议
	fmt.Fprintf(file, "## 5. Prompt修正建议\n\n")

	// 按优先级排序
	sort.Slice(analysis.PromptRecommendations, func(i, j int) bool {
		priorityMap := map[string]int{"high": 3, "medium": 2, "low": 1}
		return priorityMap[analysis.PromptRecommendations[i].Priority] > priorityMap[analysis.PromptRecommendations[j].Priority]
	})

	for i, rec := range analysis.PromptRecommendations {
		fmt.Fprintf(file, "### 建议 %d: %s [%s优先级]\n\n", i+1, rec.Issue, rec.Priority)
		fmt.Fprintf(file, "**类别**: %s\n\n", rec.Category)
		fmt.Fprintf(file, "**当前Prompt**:\n```\n%s\n```\n\n", rec.CurrentPrompt)
		fmt.Fprintf(file, "**建议修改**:\n```\n%s\n```\n\n", rec.SuggestedChange)
		fmt.Fprintf(file, "**理由**: %s\n\n", rec.Rationale)
		fmt.Fprintf(file, "**预期影响**: %s\n\n", rec.ExpectedImpact)
		if len(rec.Evidence) > 0 {
			fmt.Fprintf(file, "**证据**:\n")
			for _, ev := range rec.Evidence {
				fmt.Fprintf(file, "- %s\n", ev)
			}
		}
		fmt.Fprintf(file, "\n")
	}

	// 6. 关键发现
	fmt.Fprintf(file, "## 6. 关键发现\n\n")
	fmt.Fprintf(file, "### 6.1 Prompt理解度问题\n\n")
	if u.UnderstandingScore < 60 {
		fmt.Fprintf(file, "- ⚠️ **理解度评分偏低**: %.1f/100，说明AI对Prompt关键规则的理解不足\n", u.UnderstandingScore)
		if u.TotalDecisions > 0 {
			fmt.Fprintf(file, "- 风险回报比提及率: %.1f%%（偏低）\n", float64(u.RiskRewardMentioned)/float64(u.TotalDecisions)*100)
			fmt.Fprintf(file, "- 市场噪音提及率: %.1f%%（偏低）\n", float64(u.MarketNoiseMentioned)/float64(u.TotalDecisions)*100)
		}
	} else {
		fmt.Fprintf(file, "- ✅ 理解度评分: %.1f/100（良好）\n", u.UnderstandingScore)
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "### 6.2 Prompt遵循度问题\n\n")
	if c.RiskRewardCompliance.ComplianceRate < 90 {
		fmt.Fprintf(file, "- ⚠️ **风险回报比遵循率偏低**: %.1f%%，存在%d次违反\n",
			c.RiskRewardCompliance.ComplianceRate, len(c.RiskRewardCompliance.Violations))
	}
	if c.EntryTimingCompliance.ComplianceRate < 50 {
		fmt.Fprintf(file, "- ⚠️ **入场时机检查不足**: 只有%.1f%%的开仓决策检查了信号持续时间\n",
			c.EntryTimingCompliance.ComplianceRate)
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "### 6.3 Prompt结构问题\n\n")
	if len(a.AmbiguousRules) > 0 {
		fmt.Fprintf(file, "- ⚠️ **存在%d个模糊规则**，可能导致AI理解偏差\n", len(a.AmbiguousRules))
	}
	if len(a.ContradictoryRules) > 0 {
		fmt.Fprintf(file, "- ⚠️ **存在%d个矛盾规则**，可能导致AI决策不一致\n", len(a.ContradictoryRules))
	}
	fmt.Fprintf(file, "\n")

	// 7. 下一步行动
	fmt.Fprintf(file, "## 7. 下一步行动建议\n\n")
	fmt.Fprintf(file, "1. **立即修正**（高优先级建议）:\n")
	highPriorityCount := 0
	for _, rec := range analysis.PromptRecommendations {
		if rec.Priority == "high" {
			highPriorityCount++
			fmt.Fprintf(file, "   - %s\n", rec.Issue)
		}
	}
	if highPriorityCount == 0 {
		fmt.Fprintf(file, "   - 无高优先级建议\n")
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "2. **Phase 5**: 基于Phase 4的分析结果，生成具体的Prompt修正方案\n")
	fmt.Fprintf(file, "3. **测试验证**: 应用修正后的Prompt，重新运行交易并对比效果\n")

	return reportPath, nil
}

func printPhase4Summary(analysis *Phase4Analysis) {
	fmt.Println("\n=== Phase 4 分析摘要 ===")
	fmt.Printf("Prompt理解度: %.1f/100\n", analysis.PromptUnderstanding.UnderstandingScore)
	fmt.Printf("风险回报比遵循率: %.1f%%\n", analysis.PromptCompliance.RiskRewardCompliance.ComplianceRate)
	fmt.Printf("止损遵循率: %.1f%%\n", analysis.PromptCompliance.StopLossCompliance.ComplianceRate)
	fmt.Printf("入场时机遵循率: %.1f%%\n", analysis.PromptCompliance.EntryTimingCompliance.ComplianceRate)
	fmt.Printf("模糊规则数: %d\n", len(analysis.PromptAmbiguity.AmbiguousRules))
	fmt.Printf("矛盾规则数: %d\n", len(analysis.PromptAmbiguity.ContradictoryRules))
	fmt.Printf("修正建议数: %d\n", len(analysis.PromptRecommendations))

	highPriorityCount := 0
	for _, rec := range analysis.PromptRecommendations {
		if rec.Priority == "high" {
			highPriorityCount++
		}
	}
	fmt.Printf("高优先级建议: %d\n", highPriorityCount)
	fmt.Printf("\n报告路径: %s\n", analysis.ReportPath)
}


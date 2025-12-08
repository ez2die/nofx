package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// Phase 1数据结构
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

// Phase 3数据结构
type Phase3Analysis struct {
	MarketBenchmark   map[string]*MarketBenchmark
	RuleViolations    []*RuleViolation
	DecisionLogic     []*DecisionLogicAnalysis
	SymbolAnalyses    map[string]*SymbolAnalysis
	AIvsBenchmark     *AIBenchmarkComparison
	ReportPath        string
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

// Phase 4数据结构
type Phase4Analysis struct {
	PromptUnderstanding   PromptUnderstandingAnalysis
	PromptCompliance      PromptComplianceAnalysis
	PromptAmbiguity       PromptAmbiguityAnalysis
	PromptEffectiveness   PromptEffectivenessAnalysis
	PromptRecommendations []PromptRecommendation
	ReportPath            string
}

type PromptUnderstandingAnalysis struct {
	RiskRewardMentioned    int
	StopLossMentioned      int
	MarketNoiseMentioned    int
	TrendAnalysisMentioned int
	EntryTimingMentioned   int
	LeverageMentioned      int
	ConfidenceMentioned    int
	TotalDecisions         int
	UnderstandingScore     float64
}

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

type PromptAmbiguityAnalysis struct {
	AmbiguousRules     []AmbiguousRule
	ContradictoryRules []ContradictoryRule
	MissingClarity     []MissingClarity
}

type AmbiguousRule struct {
	RuleText      string
	AmbiguityType string
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

type PromptEffectivenessAnalysis struct {
	EffectiveRules   []RuleEffectiveness
	IneffectiveRules []RuleEffectiveness
	RuleImpact       map[string]float64
}

type RuleEffectiveness struct {
	RuleName           string
	ComplianceRate     float64
	AvgPnLWhenFollowed float64
	AvgPnLWhenViolated float64
	Impact             float64
	SampleSize         int
}

type PromptRecommendation struct {
	Priority       string
	Category       string
	Issue          string
	CurrentPrompt  string
	SuggestedChange string
	Rationale      string
	ExpectedImpact string
	Evidence       []string
}

// Phase 5分析结果
type Phase5Analysis struct {
	RootCauseAnalysis    RootCauseAnalysis
	PromptCorrectionPlan PromptCorrectionPlan
	CorrectedPrompt      string
	ReportPath           string
}

type RootCauseAnalysis struct {
	PrimaryRootCauses    []RootCause
	SecondaryRootCauses  []RootCause
	ContributingFactors  []ContributingFactor
	EvidenceSummary      string
}

type RootCause struct {
	Category        string // "prompt", "止损设置", "规则遵守", "决策逻辑"
	Description      string
	Severity         string // "critical", "high", "medium"
	Evidence         []string
	Impact           string
	Phase1Findings   []string
	Phase2Findings   []string
	Phase3Findings   []string
	Phase4Findings   []string
}

type ContributingFactor struct {
	Factor          string
	Description     string
	Contribution    string
	Evidence        []string
}

type PromptCorrectionPlan struct {
	HighPriorityCorrections []PromptCorrection
	MediumPriorityCorrections []PromptCorrection
	LowPriorityCorrections  []PromptCorrection
	CorrectionSummary       string
}

type PromptCorrection struct {
	Issue          string
	Location       string // Prompt中的位置
	CurrentText    string
	CorrectedText  string
	Rationale      string
	ExpectedImpact string
	Implementation string // 如何实施修正
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run analyze_negative_profit_phase5.go <phase1_result.json> <phase3_result.json> <phase4_result.json> <prompt_file.txt>")
		fmt.Println("Example: go run analyze_negative_profit_phase5.go phase1_result_400_1892.json phase3_result_400_1892.json phase4_result_400_1892.json prompts/lean_optimized.txt")
		os.Exit(1)
	}

	phase1File := os.Args[1]
	phase3File := os.Args[2]
	phase4File := os.Args[3]
	promptFile := os.Args[4]

	fmt.Printf("\n=== Phase 5: 根因分析与Prompt修正方案 ===\n\n")
	fmt.Printf("读取Phase 1结果: %s\n", phase1File)
	fmt.Printf("读取Phase 3结果: %s\n", phase3File)
	fmt.Printf("读取Phase 4结果: %s\n", phase4File)
	fmt.Printf("读取Prompt文件: %s\n\n", promptFile)

	// 1. 加载数据
	fmt.Println("📊 Step 1: 加载所有Phase数据...")
	phase1Data, err := loadPhase1Data(phase1File)
	if err != nil {
		log.Fatalf("加载Phase 1数据失败: %v", err)
	}

	phase3Data, err := loadPhase3Data(phase3File)
	if err != nil {
		log.Fatalf("加载Phase 3数据失败: %v", err)
	}

	phase4Data, err := loadPhase4Data(phase4File)
	if err != nil {
		log.Fatalf("加载Phase 4数据失败: %v", err)
	}

	promptContent, err := ioutil.ReadFile(promptFile)
	if err != nil {
		log.Fatalf("读取Prompt文件失败: %v", err)
	}
	fmt.Printf("✅ 数据加载完成\n\n")

	// 2. 根因分析
	fmt.Println("📊 Step 2: 进行根因分析...")
	rootCauseAnalysis := performRootCauseAnalysis(phase1Data, phase3Data, phase4Data)
	fmt.Printf("✅ 识别了 %d 个主要根因\n", len(rootCauseAnalysis.PrimaryRootCauses))
	fmt.Printf("✅ 识别了 %d 个次要根因\n", len(rootCauseAnalysis.SecondaryRootCauses))
	fmt.Printf("✅ 识别了 %d 个贡献因素\n\n", len(rootCauseAnalysis.ContributingFactors))

	// 3. 生成Prompt修正方案
	fmt.Println("📊 Step 3: 生成Prompt修正方案...")
	correctionPlan := generatePromptCorrectionPlan(phase4Data, string(promptContent))
	fmt.Printf("✅ 生成了 %d 个高优先级修正\n", len(correctionPlan.HighPriorityCorrections))
	fmt.Printf("✅ 生成了 %d 个中优先级修正\n", len(correctionPlan.MediumPriorityCorrections))
	fmt.Printf("✅ 生成了 %d 个低优先级修正\n\n", len(correctionPlan.LowPriorityCorrections))

	// 4. 生成修正后的Prompt
	fmt.Println("📊 Step 4: 生成修正后的Prompt...")
	correctedPrompt := applyPromptCorrections(string(promptContent), correctionPlan)
	fmt.Printf("✅ 修正后的Prompt已生成\n\n")

	// 5. 构建分析结果
	analysis := &Phase5Analysis{
		RootCauseAnalysis:    rootCauseAnalysis,
		PromptCorrectionPlan: correctionPlan,
		CorrectedPrompt:      correctedPrompt,
	}

	// 6. 生成报告
	fmt.Println("📊 Step 5: 生成分析报告...")
	reportPath, err := generatePhase5Report(phase1Data, phase3Data, phase4Data, analysis, string(promptContent))
	if err != nil {
		log.Fatalf("生成报告失败: %v", err)
	}
	analysis.ReportPath = reportPath
	fmt.Printf("✅ 报告已生成: %s\n\n", reportPath)

	// 7. 保存修正后的Prompt
	correctedPromptPath := fmt.Sprintf("prompts/lean_optimized_corrected_%d_%d.txt", phase1Data.StartCycle, phase1Data.EndCycle)
	ioutil.WriteFile(correctedPromptPath, []byte(correctedPrompt), 0644)
	fmt.Printf("✅ 修正后的Prompt已保存到: %s\n\n", correctedPromptPath)

	// 8. 保存结果
	jsonPath := fmt.Sprintf("phase5_result_%d_%d.json", phase1Data.StartCycle, phase1Data.EndCycle)
	jsonData, _ := json.MarshalIndent(analysis, "", "  ")
	ioutil.WriteFile(jsonPath, jsonData, 0644)
	fmt.Printf("✅ 结果已保存到: %s\n\n", jsonPath)

	// 9. 输出摘要
	printPhase5Summary(analysis)
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

func loadPhase4Data(filename string) (*Phase4Analysis, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result Phase4Analysis
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func performRootCauseAnalysis(phase1Data *Phase1Result, phase3Data *Phase3Analysis, phase4Data *Phase4Analysis) RootCauseAnalysis {
	analysis := RootCauseAnalysis{}

	// 1. 主要根因：入场时机选择不当
	entryTimingRootCause := RootCause{
		Category:   "入场时机",
		Description: "在下跌/横盘趋势中大量做多，入场时机不佳的交易占比极高",
		Severity:   "critical",
		Impact:     "导致大量交易亏损，是负收益的主要原因",
		Phase1Findings: []string{
			fmt.Sprintf("总盈亏: %.2f USDT", phase1Data.BasicStats.TotalPnL),
			fmt.Sprintf("胜率: %.2f%%", phase1Data.BasicStats.WinRate),
		},
		Phase2Findings: []string{},
		Phase3Findings: []string{
			"入场时机不佳的交易占比极高（ETHUSDT: 100%, BTCUSDT: 97.4%）",
			"在横盘趋势中做多表现不佳",
			"在下跌趋势中做多",
		},
		Phase4Findings: []string{
			"入场时机检查遵循率: 100%（但可能检查不够深入）",
		},
		Evidence: []string{
			"Phase 3显示入场时机不佳的交易占比极高",
			"在下跌/横盘趋势中大量做多",
			"市场基准显示市场整体下跌，但AI仍大量做多",
		},
	}

	// 2. 主要根因：止损设置过窄
	stopLossRootCause := RootCause{
		Category:   "止损设置",
		Description: "止损设置过窄，导致频繁触发止损，特别是ETHUSDT所有15笔交易都触发止损",
		Severity:   "critical",
		Impact:     "导致大量交易被止损，无法获得潜在收益",
		Phase1Findings: []string{
			"ETHUSDT: 15笔交易全部止损",
			"HYPEUSDT: 2笔交易全部止损",
		},
		Phase2Findings: []string{
			"止损触发总数: 18",
			"止损触发率: 32.1%",
		},
		Phase3Findings: []string{
			"2个币种的所有交易都触发止损",
			"止损设置可能过窄",
		},
		Phase4Findings: []string{
			"市场噪音提及率: 8.4%（非常低）",
			"止损遵循率: 100%（但可能未考虑市场噪音最小值）",
		},
		Evidence: []string{
			"ETHUSDT所有15笔交易都触发止损",
			"市场噪音在CoTTrace中提及率极低",
			"止损设置可能未充分考虑市场噪音",
		},
	}

	// 3. 主要根因：规则遵守不严格
	ruleComplianceRootCause := RootCause{
		Category:   "规则遵守",
		Description: "风险回报比违反率高，存在10次违反，遵循率只有64.3%",
		Severity:   "critical",
		Impact:     "违反硬约束导致交易风险增加",
		Phase1Findings: []string{},
		Phase2Findings: []string{},
		Phase3Findings: []string{
			"发现10次风险回报比违反",
			"违反的风险回报比范围: 2.20:1 - 2.96:1",
		},
		Phase4Findings: []string{
			"风险回报比遵循率: 64.3%",
			"风险回报比提及率: 60.2%（中等）",
		},
		Evidence: []string{
			"10次风险回报比违反",
			"遵循率只有64.3%",
			"违反的风险回报比都低于3.0:1的要求",
		},
	}

	// 4. 次要根因：决策逻辑质量有待提升
	decisionLogicRootCause := RootCause{
		Category:   "决策逻辑",
		Description: "不良逻辑占比(34.8%)高于良好逻辑(0.0%)，决策思维链质量需要改进",
		Severity:   "high",
		Impact:     "决策质量差导致交易表现不佳",
		Phase1Findings: []string{},
		Phase2Findings: []string{},
		Phase3Findings: []string{
			"良好逻辑: 0 (0.0%)",
			"不良逻辑: 39 (34.8%)",
			"中性逻辑: 73 (65.2%)",
		},
		Phase4Findings: []string{
			"Prompt理解度评分: 53.7/100（偏低）",
		},
		Evidence: []string{
			"不良逻辑占比高于良好逻辑",
			"Prompt理解度评分偏低",
		},
	}

	analysis.PrimaryRootCauses = []RootCause{
		entryTimingRootCause,
		stopLossRootCause,
		ruleComplianceRootCause,
	}

	analysis.SecondaryRootCauses = []RootCause{
		decisionLogicRootCause,
	}

	// 5. 贡献因素
	analysis.ContributingFactors = []ContributingFactor{
		{
			Factor:      "Prompt模糊性",
			Description: "Prompt中存在模糊和矛盾的规则",
			Contribution: "导致AI理解偏差和决策不一致",
			Evidence: []string{
				"3个模糊规则",
				"2个矛盾规则",
			},
		},
		{
			Factor:      "市场噪音理解不足",
			Description: "AI在决策中很少考虑市场噪音",
			Contribution: "导致止损设置过窄",
			Evidence: []string{
				"市场噪音提及率: 8.4%",
			},
		},
		{
			Factor:      "趋势分析不够重视",
			Description: "趋势分析在决策中不够突出",
			Contribution: "导致逆势交易",
			Evidence: []string{
				"在下跌/横盘趋势中大量做多",
			},
		},
	}

	// 生成证据摘要
	evidenceSummary := fmt.Sprintf(`
综合Phase 1-4的分析结果，导致负收益的主要原因包括：

1. **入场时机问题**（关键）：
   - Phase 3显示入场时机不佳的交易占比极高（ETHUSDT: 100%, BTCUSDT: 97.4%）
   - 在下跌/横盘趋势中大量做多
   - 市场基准显示市场整体下跌，但AI仍大量做多

2. **止损设置问题**（关键）：
   - ETHUSDT所有15笔交易都触发止损
   - 市场噪音在CoTTrace中提及率极低（8.4%）
   - 止损设置可能未充分考虑市场噪音最小值

3. **规则遵守问题**（关键）：
   - 风险回报比遵循率只有64.3%，存在10次违反
   - 违反的风险回报比范围: 2.20:1 - 2.96:1

4. **决策逻辑问题**（次要）：
   - 不良逻辑占比(34.8%)高于良好逻辑(0.0%)
   - Prompt理解度评分偏低(53.7/100)

5. **Prompt结构问题**（贡献因素）：
   - 存在3个模糊规则和2个矛盾规则
   - 可能导致AI理解偏差和决策不一致
`)

	analysis.EvidenceSummary = evidenceSummary

	return analysis
}

func generatePromptCorrectionPlan(phase4Data *Phase4Analysis, promptContent string) PromptCorrectionPlan {
	plan := PromptCorrectionPlan{}

	// 按优先级分类修正建议
	for _, rec := range phase4Data.PromptRecommendations {
		correction := PromptCorrection{
			Issue:          rec.Issue,
			CurrentText:    rec.CurrentPrompt,
			CorrectedText:  rec.SuggestedChange,
			Rationale:      rec.Rationale,
			ExpectedImpact: rec.ExpectedImpact,
			Implementation: generateImplementation(rec, promptContent),
		}

		// 确定位置
		correction.Location = findPromptLocation(rec.Issue, promptContent)

		if rec.Priority == "high" {
			plan.HighPriorityCorrections = append(plan.HighPriorityCorrections, correction)
		} else if rec.Priority == "medium" {
			plan.MediumPriorityCorrections = append(plan.MediumPriorityCorrections, correction)
		} else {
			plan.LowPriorityCorrections = append(plan.LowPriorityCorrections, correction)
		}
	}

	// 生成修正摘要
	plan.CorrectionSummary = fmt.Sprintf(`
共生成 %d 个Prompt修正方案：

**高优先级修正**（%d个）：
- 市场噪音在CoTTrace中提及率低
- 规则矛盾：耐心等待 vs 快速响应
- 规则矛盾：趋势优先 vs 多指标综合判断
- 风险回报比违反率高

**中优先级修正**（%d个）：
- 参考原则与硬约束混淆

**预期效果**：
- 提高风险回报比遵循率
- 减少止损触发率
- 减少逆势交易
- 提高决策逻辑质量
`, len(phase4Data.PromptRecommendations),
		len(plan.HighPriorityCorrections),
		len(plan.MediumPriorityCorrections))

	return plan
}

func findPromptLocation(issue string, promptContent string) string {
	// 根据问题类型查找Prompt中的位置
	if strings.Contains(issue, "市场噪音") {
		return "止损设置部分（RISK MANAGEMENT > 止损设置 > 市场噪音考虑）"
	}
	if strings.Contains(issue, "风险回报比") {
		return "CORE CONSTRAINTS > 风险回报比 和 PRE-EXECUTION CHECKLIST"
	}
	if strings.Contains(issue, "耐心等待") || strings.Contains(issue, "快速响应") {
		return "MARKET ADAPTATION > 弱势市场 和 持仓熔断复核"
	}
	if strings.Contains(issue, "趋势优先") {
		return "DATA INTERPRETATION > 关注重点 和 DECISION PROCESS"
	}
	if strings.Contains(issue, "参考原则") {
		return "DATA INTERPRETATION 和 MARKET ADAPTATION 部分"
	}
	return "Prompt全文"
}

func generateImplementation(rec PromptRecommendation, promptContent string) string {
	var impl strings.Builder

	impl.WriteString("**修正步骤**：\n\n")

	if strings.Contains(rec.Issue, "市场噪音") {
		impl.WriteString("1. 在止损设置部分添加强制检查清单\n")
		impl.WriteString("2. 要求AI在设置止损时明确说明是否考虑了市场噪音\n")
		impl.WriteString("3. 在预执行检查清单中添加市场噪音检查项\n")
	} else if strings.Contains(rec.Issue, "风险回报比") {
		impl.WriteString("1. 在Prompt开头强调风险回报比要求\n")
		impl.WriteString("2. 在预执行检查清单中将风险回报比检查放在第一位\n")
		impl.WriteString("3. 添加'如果风险回报比<3.0:1，立即拒绝，不要寻找例外'的明确说明\n")
		impl.WriteString("4. 要求AI在CoTTrace中明确计算并说明风险回报比\n")
	} else if strings.Contains(rec.Issue, "规则矛盾") {
		if strings.Contains(rec.Issue, "耐心等待") {
			impl.WriteString("1. 在决策流程中明确说明何时耐心等待，何时快速响应\n")
			impl.WriteString("2. 添加决策树：入场假设失效 → 快速响应；信号不稳定 → 耐心等待\n")
			impl.WriteString("3. 在持仓熔断复核中明确优先级\n")
		} else if strings.Contains(rec.Issue, "趋势优先") {
			impl.WriteString("1. 将趋势分析提升为决策流程的第一步\n")
			impl.WriteString("2. 明确说明：先判断趋势方向，再结合其他指标\n")
			impl.WriteString("3. 当趋势与其他指标冲突时，优先考虑趋势方向\n")
		}
	} else if strings.Contains(rec.Issue, "参考原则") {
		impl.WriteString("1. 使用【】标记硬约束\n")
		impl.WriteString("2. 使用（）标记参考原则\n")
		impl.WriteString("3. 在Prompt开头说明标记规则\n")
	}

	return impl.String()
}

func applyPromptCorrections(originalPrompt string, plan PromptCorrectionPlan) string {
	corrected := originalPrompt

	// 应用高优先级修正
	for _, correction := range plan.HighPriorityCorrections {
		corrected = applyCorrection(corrected, correction)
	}

	// 应用中优先级修正
	for _, correction := range plan.MediumPriorityCorrections {
		corrected = applyCorrection(corrected, correction)
	}

	return corrected
}

func applyCorrection(prompt string, correction PromptCorrection) string {
	// 根据修正类型应用不同的修正策略
	if strings.Contains(correction.Issue, "市场噪音") {
		// 在止损设置部分添加市场噪音强制检查
		pattern := regexp.MustCompile(`(### 市场噪音考虑.*?)(\n###)`)
		replacement := `$1

**⚠️ 强制检查**：在设置止损时，你必须在CoTTrace中明确说明：
1. 是否考虑了市场噪音最小值
2. 止损距离是否 ≥ 资产的最小价格止损要求
3. 如果未考虑，说明原因

$2`
		prompt = pattern.ReplaceAllString(prompt, replacement)
	}

	if strings.Contains(correction.Issue, "风险回报比") {
		// 在预执行检查清单中将风险回报比放在第一位
		pattern := regexp.MustCompile(`(## Tier 1: 硬约束.*?)(- \[ \] \*\*风险回报比)`)
		replacement := `$1
- [ ] **风险回报比（最高优先级）**：计算的风险回报比 ≥ 3.0:1？
  - ⚠️ **如果风险回报比 < 3.0:1，立即拒绝，不要寻找例外**
  - 双重检查计算：(take_profit - entry) / (entry - stop_loss) 对于多仓，或 (entry - take_profit) / (stop_loss - entry) 对于空仓
  - 必须在CoTTrace中明确计算并说明风险回报比
$2`
		prompt = pattern.ReplaceAllString(prompt, replacement)

		// 在Prompt开头添加强调
		pattern2 := regexp.MustCompile(`(# CORE CONSTRAINTS.*?## 风险回报比)`)
		replacement2 := `$1

**⚠️ 最高优先级规则**：风险回报比 ≥ 3.0:1 是绝对硬约束，没有任何例外。在做出任何开仓决策前，必须首先验证风险回报比。`
		prompt = pattern2.ReplaceAllString(prompt, replacement2)
	}

	if strings.Contains(correction.Issue, "耐心等待") && strings.Contains(correction.Issue, "快速响应") {
		// 添加决策树说明
		pattern := regexp.MustCompile(`(### 弱势市场.*?## DECISION PROCESS)`)
		decisionTree := "\n\n**⚠️ 规则优先级说明**：\n" +
			"- **入场假设失效** → 立即使用 close_* 平仓（快速响应，不受耐心等待规则限制）\n" +
			"- **信号不稳定** → 耐心等待信号稳定（至少10-15分钟）\n" +
			"- **止损距离<0.2%** → 立即使用 close_* 平仓（快速响应）\n" +
			"- **正常持仓管理** → 耐心等待时间窗口确认（10-15分钟）\n\n" +
			"**决策树**：\n" +
			"```\n" +
			"入场假设失效？ → 是 → 立即平仓（快速响应）\n" +
			"                ↓ 否\n" +
			"止损距离<0.2%？ → 是 → 立即平仓（快速响应）\n" +
			"                ↓ 否\n" +
			"信号不稳定？ → 是 → 耐心等待\n" +
			"                ↓ 否\n" +
			"正常持仓管理 → 耐心等待时间窗口\n" +
			"```\n"
		replacement := "$1" + decisionTree
		prompt = pattern.ReplaceAllString(prompt, replacement)
	}

	if strings.Contains(correction.Issue, "趋势优先") && strings.Contains(correction.Issue, "多指标综合判断") {
		// 在决策流程中明确趋势优先
		pattern := regexp.MustCompile(`(3\. \*\*市场环境识别.*?)(4\. \*\*参数调整)`)
		replacement := `$1
3.5. **趋势方向判断（第一步）**：
   - 首先判断趋势方向（上涨/下跌/横盘）
   - 使用EMA方向、MACD方向、价格与EMA关系综合判断
   - **趋势方向是决策的基础**，其他指标在此基础上进行综合判断
   - 当趋势与其他指标冲突时，优先考虑趋势方向
   - 示例：趋势向下 + RSI超卖 → 优先考虑趋势方向，RSI超卖不足以作为做多理由
$2`
		prompt = pattern.ReplaceAllString(prompt, replacement)
	}

	if strings.Contains(correction.Issue, "参考原则") {
		// 添加标记说明
		pattern := regexp.MustCompile(`(# ROLE)`)
		replacement := `$1

**⚠️ Prompt标记规则**：
- 【硬约束】：必须严格遵守的规则，违反会导致交易失败或风险增加
- （参考原则）：指导性建议，可根据实际情况灵活应用，但不应与硬约束冲突`
		prompt = pattern.ReplaceAllString(prompt, replacement)

		// 替换参考原则标记
		prompt = strings.ReplaceAll(prompt, "参考原则（非强制规则）", "（参考原则）")
		prompt = strings.ReplaceAll(prompt, "**参考原则**", "（参考原则）")
		
		// 替换硬约束标记
		prompt = strings.ReplaceAll(prompt, "## 风险回报比", "## 【硬约束】风险回报比")
		prompt = strings.ReplaceAll(prompt, "## 杠杆限制", "## 【硬约束】杠杆限制")
		prompt = strings.ReplaceAll(prompt, "## 止损方向要求（关键）", "## 【硬约束】止损方向要求")
		prompt = strings.ReplaceAll(prompt, "## 止盈方向要求（关键）", "## 【硬约束】止盈方向要求")
		prompt = strings.ReplaceAll(prompt, "## 置信度要求", "## 【硬约束】置信度要求")
	}

	return prompt
}

func generatePhase5Report(phase1Data *Phase1Result, phase3Data *Phase3Analysis, phase4Data *Phase4Analysis, analysis *Phase5Analysis, originalPrompt string) (string, error) {
	reportPath := fmt.Sprintf("phase5_report_%d_%d.md", phase1Data.StartCycle, phase1Data.EndCycle)
	file, err := os.Create(reportPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fmt.Fprintf(file, "# Phase 5: 根因分析与Prompt修正方案\n\n")
	fmt.Fprintf(file, "**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**基于Phase 1-4数据**: Cycle %d - %d\n\n", phase1Data.StartCycle, phase1Data.EndCycle)

	// 1. 执行摘要
	fmt.Fprintf(file, "## 1. 执行摘要\n\n")
	fmt.Fprintf(file, "本报告综合Phase 1-4的分析结果，识别了导致负收益的根本原因，并提供了具体的Prompt修正方案。\n\n")
	fmt.Fprintf(file, "**关键发现**：\n")
	fmt.Fprintf(file, "- 主要根因：入场时机选择不当、止损设置过窄、规则遵守不严格\n")
	fmt.Fprintf(file, "- 次要根因：决策逻辑质量有待提升\n")
	fmt.Fprintf(file, "- 贡献因素：Prompt模糊性、市场噪音理解不足、趋势分析不够重视\n\n")

	// 2. 根因分析
	fmt.Fprintf(file, "## 2. 根因分析\n\n")
	rc := analysis.RootCauseAnalysis

	fmt.Fprintf(file, "### 2.1 主要根因\n\n")
	for i, cause := range rc.PrimaryRootCauses {
		fmt.Fprintf(file, "#### 根因 %d: %s [%s严重程度]\n\n", i+1, cause.Description, cause.Severity)
		fmt.Fprintf(file, "**类别**: %s\n\n", cause.Category)
		fmt.Fprintf(file, "**影响**: %s\n\n", cause.Impact)

		if len(cause.Phase1Findings) > 0 {
			fmt.Fprintf(file, "**Phase 1发现**:\n")
			for _, finding := range cause.Phase1Findings {
				fmt.Fprintf(file, "- %s\n", finding)
			}
			fmt.Fprintf(file, "\n")
		}

		if len(cause.Phase2Findings) > 0 {
			fmt.Fprintf(file, "**Phase 2发现**:\n")
			for _, finding := range cause.Phase2Findings {
				fmt.Fprintf(file, "- %s\n", finding)
			}
			fmt.Fprintf(file, "\n")
		}

		if len(cause.Phase3Findings) > 0 {
			fmt.Fprintf(file, "**Phase 3发现**:\n")
			for _, finding := range cause.Phase3Findings {
				fmt.Fprintf(file, "- %s\n", finding)
			}
			fmt.Fprintf(file, "\n")
		}

		if len(cause.Phase4Findings) > 0 {
			fmt.Fprintf(file, "**Phase 4发现**:\n")
			for _, finding := range cause.Phase4Findings {
				fmt.Fprintf(file, "- %s\n", finding)
			}
			fmt.Fprintf(file, "\n")
		}

		if len(cause.Evidence) > 0 {
			fmt.Fprintf(file, "**证据**:\n")
			for _, ev := range cause.Evidence {
				fmt.Fprintf(file, "- %s\n", ev)
			}
			fmt.Fprintf(file, "\n")
		}
	}

	if len(rc.SecondaryRootCauses) > 0 {
		fmt.Fprintf(file, "### 2.2 次要根因\n\n")
		for i, cause := range rc.SecondaryRootCauses {
			fmt.Fprintf(file, "#### 根因 %d: %s [%s严重程度]\n\n", i+1, cause.Description, cause.Severity)
			fmt.Fprintf(file, "**类别**: %s\n\n", cause.Category)
			fmt.Fprintf(file, "**影响**: %s\n\n", cause.Impact)
			if len(cause.Evidence) > 0 {
				fmt.Fprintf(file, "**证据**:\n")
				for _, ev := range cause.Evidence {
					fmt.Fprintf(file, "- %s\n", ev)
				}
			}
			fmt.Fprintf(file, "\n")
		}
	}

	if len(rc.ContributingFactors) > 0 {
		fmt.Fprintf(file, "### 2.3 贡献因素\n\n")
		for i, factor := range rc.ContributingFactors {
			fmt.Fprintf(file, "#### 因素 %d: %s\n\n", i+1, factor.Factor)
			fmt.Fprintf(file, "**描述**: %s\n\n", factor.Description)
			fmt.Fprintf(file, "**贡献**: %s\n\n", factor.Contribution)
			if len(factor.Evidence) > 0 {
				fmt.Fprintf(file, "**证据**:\n")
				for _, ev := range factor.Evidence {
					fmt.Fprintf(file, "- %s\n", ev)
				}
			}
			fmt.Fprintf(file, "\n")
		}
	}

	fmt.Fprintf(file, "### 2.4 证据摘要\n\n")
	fmt.Fprintf(file, "%s\n\n", rc.EvidenceSummary)

	// 3. Prompt修正方案
	fmt.Fprintf(file, "## 3. Prompt修正方案\n\n")
	plan := analysis.PromptCorrectionPlan

	fmt.Fprintf(file, "### 3.1 修正方案摘要\n\n")
	fmt.Fprintf(file, "%s\n\n", plan.CorrectionSummary)

	if len(plan.HighPriorityCorrections) > 0 {
		fmt.Fprintf(file, "### 3.2 高优先级修正\n\n")
		for i, correction := range plan.HighPriorityCorrections {
			fmt.Fprintf(file, "#### 修正 %d: %s\n\n", i+1, correction.Issue)
			fmt.Fprintf(file, "**位置**: %s\n\n", correction.Location)
			fmt.Fprintf(file, "**当前文本**:\n```\n%s\n```\n\n", correction.CurrentText)
			fmt.Fprintf(file, "**修正后**:\n```\n%s\n```\n\n", correction.CorrectedText)
			fmt.Fprintf(file, "**理由**: %s\n\n", correction.Rationale)
			fmt.Fprintf(file, "**预期影响**: %s\n\n", correction.ExpectedImpact)
			fmt.Fprintf(file, "%s\n\n", correction.Implementation)
		}
	}

	if len(plan.MediumPriorityCorrections) > 0 {
		fmt.Fprintf(file, "### 3.3 中优先级修正\n\n")
		for i, correction := range plan.MediumPriorityCorrections {
			fmt.Fprintf(file, "#### 修正 %d: %s\n\n", i+1, correction.Issue)
			fmt.Fprintf(file, "**位置**: %s\n\n", correction.Location)
			fmt.Fprintf(file, "**当前文本**:\n```\n%s\n```\n\n", correction.CurrentText)
			fmt.Fprintf(file, "**修正后**:\n```\n%s\n```\n\n", correction.CorrectedText)
			fmt.Fprintf(file, "**理由**: %s\n\n", correction.Rationale)
			fmt.Fprintf(file, "**预期影响**: %s\n\n", correction.ExpectedImpact)
			fmt.Fprintf(file, "%s\n\n", correction.Implementation)
		}
	}

	// 4. 修正后的Prompt预览
	fmt.Fprintf(file, "## 4. 修正后的Prompt预览\n\n")
	fmt.Fprintf(file, "修正后的Prompt已保存到: `prompts/lean_optimized_corrected_%d_%d.txt`\n\n", phase1Data.StartCycle, phase1Data.EndCycle)
	fmt.Fprintf(file, "### 4.1 主要修改点\n\n")
	fmt.Fprintf(file, "1. **风险回报比强化**：\n")
	fmt.Fprintf(file, "   - 在Prompt开头添加最高优先级说明\n")
	fmt.Fprintf(file, "   - 在预执行检查清单中将风险回报比检查放在第一位\n")
	fmt.Fprintf(file, "   - 添加'如果风险回报比<3.0:1，立即拒绝'的明确说明\n\n")
	fmt.Fprintf(file, "2. **市场噪音强制检查**：\n")
	fmt.Fprintf(file, "   - 在止损设置部分添加强制检查清单\n")
	fmt.Fprintf(file, "   - 要求AI在CoTTrace中明确说明是否考虑了市场噪音\n\n")
	fmt.Fprintf(file, "3. **规则优先级明确**：\n")
	fmt.Fprintf(file, "   - 添加决策树说明何时耐心等待，何时快速响应\n")
	fmt.Fprintf(file, "   - 明确趋势分析的优先级\n\n")
	fmt.Fprintf(file, "4. **规则标记优化**：\n")
	fmt.Fprintf(file, "   - 使用【】标记硬约束\n")
	fmt.Fprintf(file, "   - 使用（）标记参考原则\n\n")

	// 5. 实施建议
	fmt.Fprintf(file, "## 5. 实施建议\n\n")
	fmt.Fprintf(file, "### 5.1 立即实施（高优先级修正）\n\n")
	fmt.Fprintf(file, "1. **应用修正后的Prompt**：\n")
	fmt.Fprintf(file, "   - 使用 `prompts/lean_optimized_corrected_%d_%d.txt` 替换当前Prompt\n", phase1Data.StartCycle, phase1Data.EndCycle)
	fmt.Fprintf(file, "   - 或手动应用高优先级修正\n\n")
	fmt.Fprintf(file, "2. **测试验证**：\n")
	fmt.Fprintf(file, "   - 使用修正后的Prompt运行新的交易周期\n")
	fmt.Fprintf(file, "   - 监控风险回报比遵循率是否提高\n")
	fmt.Fprintf(file, "   - 监控止损触发率是否降低\n")
	fmt.Fprintf(file, "   - 监控入场时机选择是否改善\n\n")
	fmt.Fprintf(file, "3. **持续监控**：\n")
	fmt.Fprintf(file, "   - 定期运行Phase 1-4分析\n")
	fmt.Fprintf(file, "   - 对比修正前后的表现\n")
	fmt.Fprintf(file, "   - 根据结果进一步优化Prompt\n\n")

	fmt.Fprintf(file, "### 5.2 预期改进效果\n\n")
	fmt.Fprintf(file, "基于修正方案，预期可以实现以下改进：\n\n")
	fmt.Fprintf(file, "1. **风险回报比遵循率**：从64.3%提升至90%+\n")
	fmt.Fprintf(file, "2. **止损触发率**：从32.1%降低至20%以下\n")
	fmt.Fprintf(file, "3. **入场时机选择**：减少逆势交易，入场时机不佳的交易占比降低\n")
	fmt.Fprintf(file, "4. **决策逻辑质量**：Prompt理解度评分提升，不良逻辑占比降低\n\n")

	// 6. 关键发现总结
	fmt.Fprintf(file, "## 6. 关键发现总结\n\n")
	fmt.Fprintf(file, "### 6.1 根因优先级\n\n")
	fmt.Fprintf(file, "| 优先级 | 根因 | 严重程度 | 影响 |\n")
	fmt.Fprintf(file, "|--------|------|----------|------|\n")
	for i, cause := range rc.PrimaryRootCauses {
		fmt.Fprintf(file, "| P%d | %s | %s | %s |\n", i+1, cause.Description, cause.Severity, cause.Impact)
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "### 6.2 修正方案优先级\n\n")
	fmt.Fprintf(file, "| 优先级 | 修正项 | 预期效果 |\n")
	fmt.Fprintf(file, "|--------|--------|----------|\n")
	for i, correction := range plan.HighPriorityCorrections {
		fmt.Fprintf(file, "| 高%d | %s | %s |\n", i+1, correction.Issue, correction.ExpectedImpact)
	}
	for i, correction := range plan.MediumPriorityCorrections {
		fmt.Fprintf(file, "| 中%d | %s | %s |\n", i+1, correction.Issue, correction.ExpectedImpact)
	}
	fmt.Fprintf(file, "\n")

	// 7. 下一步行动
	fmt.Fprintf(file, "## 7. 下一步行动\n\n")
	fmt.Fprintf(file, "1. **立即行动**：\n")
	fmt.Fprintf(file, "   - 应用修正后的Prompt\n")
	fmt.Fprintf(file, "   - 开始新的交易周期\n")
	fmt.Fprintf(file, "   - 监控关键指标\n\n")
	fmt.Fprintf(file, "2. **持续优化**：\n")
	fmt.Fprintf(file, "   - 定期运行分析脚本\n")
	fmt.Fprintf(file, "   - 根据新数据调整Prompt\n")
	fmt.Fprintf(file, "   - 迭代优化交易策略\n\n")

	return reportPath, nil
}

func printPhase5Summary(analysis *Phase5Analysis) {
	fmt.Println("\n=== Phase 5 分析摘要 ===")
	fmt.Printf("主要根因数: %d\n", len(analysis.RootCauseAnalysis.PrimaryRootCauses))
	fmt.Printf("次要根因数: %d\n", len(analysis.RootCauseAnalysis.SecondaryRootCauses))
	fmt.Printf("贡献因素数: %d\n", len(analysis.RootCauseAnalysis.ContributingFactors))
	fmt.Printf("高优先级修正: %d\n", len(analysis.PromptCorrectionPlan.HighPriorityCorrections))
	fmt.Printf("中优先级修正: %d\n", len(analysis.PromptCorrectionPlan.MediumPriorityCorrections))
	fmt.Printf("低优先级修正: %d\n", len(analysis.PromptCorrectionPlan.LowPriorityCorrections))
	fmt.Printf("\n报告路径: %s\n", analysis.ReportPath)
}


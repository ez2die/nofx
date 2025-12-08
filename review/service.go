package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"nofx/trade_history"
)

// ReviewService 复盘服务接口
type ReviewService interface {
	RunReview(ctx context.Context, traderID string, startTime, endTime time.Time) (*ReviewResult, error)
}

// DecisionLogCollector 决策日志收集器接口（避免循环导入）
type DecisionLogCollector interface {
	Collect(ctx context.Context, startTime, endTime time.Time) ([]*DecisionRecord, error)
}

// TradeHistoryCollector 交易历史收集器接口
type TradeHistoryCollector interface {
	Collect(ctx context.Context, traderID string, startTime, endTime time.Time) ([]*trade_history.TradeRecord, error)
}

// DEXDataCollector DEX数据收集器接口
type DEXDataCollector interface {
	Collect(ctx context.Context, startTime, endTime time.Time) ([]trade_history.ExchangeFill, error)
}

// reviewService 复盘服务实现
type reviewService struct {
	decisionLogCollector    DecisionLogCollector
	tradeHistoryCollector   TradeHistoryCollector
	dexDataCollector        DEXDataCollector
	ruleEngine              RuleEngine
	performanceAnalyzer     PerformanceAnalyzer
	dexValidationAnalyzer   DEXValidationAnalyzer
	decisionMatcher         DecisionMatcher
	violationMetricsCalc    ViolationMetricsCalculator
	reporter                Reporter
	repository              ReviewRepository
}

// NewReviewService 创建复盘服务（使用接口，避免循环导入）
func NewReviewService(
	decisionLogCollector DecisionLogCollector,
	tradeHistoryCollector TradeHistoryCollector,
	dexDataCollector DEXDataCollector,
	ruleEngine RuleEngine,
	performanceAnalyzer PerformanceAnalyzer,
	dexValidationAnalyzer DEXValidationAnalyzer,
	decisionMatcher DecisionMatcher,
	violationMetricsCalc ViolationMetricsCalculator,
	reporter Reporter,
	repo ReviewRepository,
) ReviewService {
	return &reviewService{
		decisionLogCollector:    decisionLogCollector,
		tradeHistoryCollector:   tradeHistoryCollector,
		dexDataCollector:        dexDataCollector,
		ruleEngine:              ruleEngine,
		performanceAnalyzer:     performanceAnalyzer,
		dexValidationAnalyzer:   dexValidationAnalyzer,
		decisionMatcher:         decisionMatcher,
		violationMetricsCalc:    violationMetricsCalc,
		reporter:                reporter,
		repository:              repo,
	}
}

// RunReview 执行一次复盘
func (s *reviewService) RunReview(ctx context.Context, traderID string, startTime, endTime time.Time) (*ReviewResult, error) {
	startReviewTime := time.Now()
	log.Printf("🔄 开始执行复盘 (TraderID: %s, 时间范围: %s ~ %s)",
		traderID, startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	result := &ReviewResult{
		TraderID:    traderID,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      "success",
		Performance: &PerformanceAnalysis{SymbolStats: make(map[string]*SymbolPerformance)},
		Violations:  []Violation{},
		Metrics: &StandardizedMetrics{
			Performance:    &PerformanceAnalysis{SymbolStats: make(map[string]*SymbolPerformance)},
			RuleViolations: &RuleViolationMetrics{ViolationSeverityDistribution: make(map[string]int)},
			DEXValidation:  &DEXValidationMetrics{},
		},
	}

	// Phase 1: 数据收集 (目标: <30秒)
	phase1Start := time.Now()
	decisions, trades, dexTrades, err := s.collectData(ctx, traderID, startTime, endTime)
	result.Duration = time.Since(phase1Start)
	if err != nil {
		return s.handleError(result, err, "数据收集失败")
	}
	log.Printf("✅ Phase 1 数据收集完成 (耗时: %v, 决策数: %d, 交易数: %d, DEX交易数: %d)",
		result.Duration, len(decisions), len(trades), len(dexTrades))

	// Phase 2: 数据匹配 (目标: <10秒)
	phase2Start := time.Now()
	matches, unmatchedDecisions, unmatchedDEXTrades, err := s.matchData(decisions, dexTrades)
	result.Duration = time.Since(phase2Start)
	if err != nil {
		return s.handleError(result, err, "数据匹配失败")
	}
	log.Printf("✅ Phase 2 数据匹配完成 (耗时: %v, 匹配数: %d, 未匹配决策: %d, 未匹配DEX交易: %d)",
		result.Duration, len(matches), len(unmatchedDecisions), len(unmatchedDEXTrades))

	// Phase 3: 分析处理 (目标: <30秒)
	phase3Start := time.Now()
	err = s.analyzeData(ctx, result, decisions, trades, matches, unmatchedDecisions, unmatchedDEXTrades)
	result.Duration = time.Since(phase3Start)
	if err != nil {
		log.Printf("⚠️ Phase 3 分析处理部分失败: %v，继续执行", err)
		result.Status = "partial"
		result.ErrorMessage = fmt.Sprintf("部分分析失败: %v", err)
	}
	log.Printf("✅ Phase 3 分析处理完成 (耗时: %v)", result.Duration)

	// Phase 4: 报告生成 (目标: <30秒)
	phase4Start := time.Now()
	reportPath, err := s.generateReport(result)
	result.Duration = time.Since(phase4Start)
	if err != nil {
		log.Printf("⚠️ Phase 4 报告生成失败: %v", err)
		result.Status = "partial"
		if result.ErrorMessage != "" {
			result.ErrorMessage += "; "
		}
		result.ErrorMessage += fmt.Sprintf("报告生成失败: %v", err)
	} else {
		result.ReportPath = reportPath
		log.Printf("✅ Phase 4 报告生成完成 (耗时: %v, 报告路径: %s)", result.Duration, reportPath)
	}

	// 计算总耗时
	result.Duration = time.Since(startReviewTime)
	log.Printf("✅ 复盘执行完成 (总耗时: %v, 状态: %s)", result.Duration, result.Status)

	// 保存结果到数据库
	if s.repository != nil {
		if err := s.saveResult(ctx, result); err != nil {
			log.Printf("⚠️ 保存复盘结果到数据库失败: %v", err)
		}
	}

	return result, nil
}

// collectData 收集数据
func (s *reviewService) collectData(
	ctx context.Context,
	traderID string,
	startTime, endTime time.Time,
) ([]*DecisionRecord, []*trade_history.TradeRecord, []trade_history.ExchangeFill, error) {
	// 并行收集三种数据
	type decisionResult struct {
		decisions []*DecisionRecord
		err       error
	}
	type tradeResult struct {
		trades []*trade_history.TradeRecord
		err    error
	}
	type dexResult struct {
		dexTrades []trade_history.ExchangeFill
		err       error
	}

	decisionChan := make(chan decisionResult, 1)
	tradeChan := make(chan tradeResult, 1)
	dexChan := make(chan dexResult, 1)

	// 收集决策日志
	go func() {
		decisions, err := s.decisionLogCollector.Collect(ctx, startTime, endTime)
		decisionChan <- decisionResult{decisions: decisions, err: err}
	}()

	// 收集交易历史
	go func() {
		trades, err := s.tradeHistoryCollector.Collect(ctx, traderID, startTime, endTime)
		tradeChan <- tradeResult{trades: trades, err: err}
	}()

	// 收集DEX数据（这是关键数据，失败会导致复盘停止）
	go func() {
		dexTrades, err := s.dexDataCollector.Collect(ctx, startTime, endTime)
		dexChan <- dexResult{dexTrades: dexTrades, err: err}
	}()

	// 等待所有收集完成
	decisionRes := <-decisionChan
	tradeRes := <-tradeChan
	dexRes := <-dexChan

	// DEX数据收集失败，停止复盘
	if dexRes.err != nil {
		return nil, nil, nil, fmt.Errorf("DEX数据收集失败: %w", dexRes.err)
	}

	// 决策日志和交易历史收集失败，记录警告但继续
	if decisionRes.err != nil {
		log.Printf("⚠️ 决策日志收集失败: %v，使用空数据", decisionRes.err)
		decisionRes.decisions = []*DecisionRecord{}
	}
	if tradeRes.err != nil {
		log.Printf("⚠️ 交易历史收集失败: %v，使用空数据", tradeRes.err)
		tradeRes.trades = []*trade_history.TradeRecord{}
	}

	return decisionRes.decisions, tradeRes.trades, dexRes.dexTrades, nil
}

// matchData 匹配数据
func (s *reviewService) matchData(
	decisions []*DecisionRecord,
	dexTrades []trade_history.ExchangeFill,
) ([]*DecisionMatch, []*DecisionRecord, []trade_history.ExchangeFill, error) {
	return s.decisionMatcher.Match(decisions, dexTrades)
}

// analyzeData 分析数据
func (s *reviewService) analyzeData(
	ctx context.Context,
	result *ReviewResult,
	decisions []*DecisionRecord,
	trades []*trade_history.TradeRecord,
	matches []*DecisionMatch,
	unmatchedDecisions []*DecisionRecord,
	unmatchedDEXTrades []trade_history.ExchangeFill,
) error {
	// 3.1 规则检查
	allViolations := []Violation{}
	for _, decision := range decisions {
		violations, err := s.ruleEngine.CheckDecision(decision)
		if err != nil {
			log.Printf("⚠️ 规则检查失败 (cycle %d): %v", decision.CycleNumber, err)
			continue
		}
		allViolations = append(allViolations, violations...)
	}
	result.Violations = allViolations

	// 3.2 计算规则违反指标
	if len(decisions) > 0 {
		violationMetrics, err := s.violationMetricsCalc.Calculate(allViolations, len(decisions))
		if err == nil {
			result.Metrics.RuleViolations = violationMetrics
		}
	}

	// 3.3 交易表现分析
	if len(trades) > 0 {
		performance, err := s.performanceAnalyzer.AnalyzePerformance(trades, decisions)
		if err == nil {
			result.Performance = performance
			result.Metrics.Performance = performance
		}
	}

	// 3.4 DEX验证分析
	if len(matches) > 0 {
		// 执行质量分析
		executionQuality, err := s.dexValidationAnalyzer.AnalyzeExecutionQuality(matches)
		if err == nil {
			result.ExecutionQuality = executionQuality
		}

		// DEX验证指标计算
		dexMetrics, err := s.dexValidationAnalyzer.CalculateDEXMetrics(matches, unmatchedDecisions, unmatchedDEXTrades)
		if err == nil {
			result.DEXValidation = dexMetrics
			result.Metrics.DEXValidation = dexMetrics
		}
	}

	return nil
}

// generateReport 生成报告
func (s *reviewService) generateReport(result *ReviewResult) (string, error) {
	if s.reporter == nil {
		return "", fmt.Errorf("报告生成器未初始化")
	}
	return s.reporter.GenerateReport(result)
}

// saveResult 保存结果到数据库
func (s *reviewService) saveResult(ctx context.Context, result *ReviewResult) error {
	if s.repository == nil {
		return nil
	}

	// 构建ReviewRecord
	record := &ReviewRecord{
		TraderID:   result.TraderID,
		StartTime:  result.StartTime,
		EndTime:    result.EndTime,
		ReportPath: result.ReportPath,
		Status:     result.Status,
		ErrorMessage: result.ErrorMessage,
		CreatedAt:   time.Now(),
	}

	// 序列化Summary和Metrics
	if result.Metrics != nil {
		summaryJSON, _ := json.Marshal(map[string]interface{}{
			"performance":    result.Performance,
			"violations":     result.Violations,
			"dex_validation": result.DEXValidation,
		})
		record.Summary = string(summaryJSON)

		metricsJSON, _ := json.Marshal(result.Metrics)
		record.Metrics = string(metricsJSON)
	}

	// 填充统计字段
	if result.Performance != nil {
		record.TotalTrades = result.Performance.TotalTrades
		record.TotalPnL = result.Performance.TotalPnL
		record.WinRate = result.Performance.WinRate
	}

	record.ErrorCount = len(result.Violations)

	return s.repository.Save(ctx, record)
}

// handleError 处理错误
func (s *reviewService) handleError(result *ReviewResult, err error, message string) (*ReviewResult, error) {
	result.Status = "failed"
	result.ErrorMessage = fmt.Sprintf("%s: %v", message, err)
	result.Duration = time.Since(result.StartTime)
	log.Printf("❌ 复盘失败: %s", result.ErrorMessage)
	return result, err
}


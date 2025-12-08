package reporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"nofx/review"
)

// MarkdownReporter Markdown报告生成器
type MarkdownReporter struct {
	templateDir string
	reportDir   string
}

// NewMarkdownReporter 创建Markdown报告生成器
func NewMarkdownReporter(reportDir string) *MarkdownReporter {
	if reportDir == "" {
		reportDir = "data/reviews"
	}

	// 确保报告目录存在
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Printf("⚠️ 创建报告目录失败: %v", err)
	}

	return &MarkdownReporter{
		templateDir: "review/reporter/templates",
		reportDir:   reportDir,
	}
}

// GenerateReport 生成复盘报告
func (r *MarkdownReporter) GenerateReport(result *review.ReviewResult) (string, error) {
	log.Printf("📝 开始生成复盘报告 (TraderID: %s, 时间范围: %s ~ %s)",
		result.TraderID, result.StartTime.Format("2006-01-02 15:04:05"), result.EndTime.Format("2006-01-02 15:04:05"))

	// 生成报告内容
	reportContent := r.generateReportContent(result)

	// 生成文件名
	filename := fmt.Sprintf("review_%s_%s.md",
		result.TraderID,
		result.EndTime.Format("20060102_150405"))

	filepath := filepath.Join(r.reportDir, filename)

	// 写入文件
	if err := os.WriteFile(filepath, []byte(reportContent), 0644); err != nil {
		return "", fmt.Errorf("写入报告文件失败: %w", err)
	}

	log.Printf("✅ 复盘报告生成完成: %s", filepath)
	return filepath, nil
}

// generateReportContent 生成报告内容
func (r *MarkdownReporter) generateReportContent(result *review.ReviewResult) string {
	var content string

	// 标题
	content += fmt.Sprintf("# 复盘报告\n\n")
	content += fmt.Sprintf("**交易员ID**: %s\n", result.TraderID)
	content += fmt.Sprintf("**时间范围**: %s ~ %s\n", result.StartTime.Format("2006-01-02 15:04:05"), result.EndTime.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("**生成时间**: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("**状态**: %s\n", result.Status)
	content += fmt.Sprintf("**执行耗时**: %v\n\n", result.Duration)

	if result.ErrorMessage != "" {
		content += fmt.Sprintf("**错误信息**: %s\n\n", result.ErrorMessage)
	}

	// 分隔线
	content += "---\n\n"

	// 1. 执行摘要
	content += r.generateExecutiveSummary(result)

	// 2. 表现概览
	content += r.generatePerformanceOverview(result)

	// 3. 详细分析
	content += r.generateDetailedAnalysis(result)

	// 4. 风险分析
	content += r.generateRiskAnalysis(result)

	// 5. 改进建议
	content += r.generateRecommendations(result)

	// 6. 数据附录
	content += r.generateDataAppendix(result)

	return content
}

// generateExecutiveSummary 生成执行摘要
func (r *MarkdownReporter) generateExecutiveSummary(result *review.ReviewResult) string {
	var summary string
	summary += "## 一、执行摘要\n\n"

	summary += "### 核心指标\n\n"
	if result.Performance != nil {
		summary += fmt.Sprintf("- **总交易数**: %d\n", result.Performance.TotalTrades)
		summary += fmt.Sprintf("- **完成交易数**: %d\n", result.Performance.CompletedTrades)
		summary += fmt.Sprintf("- **总盈亏**: %.2f USDT\n", result.Performance.TotalPnL)
		summary += fmt.Sprintf("- **净盈亏**: %.2f USDT\n", result.Performance.NetPnL)
		summary += fmt.Sprintf("- **胜率**: %.2f%%\n", result.Performance.WinRate)
		summary += fmt.Sprintf("- **平均盈亏**: %.2f USDT\n", result.Performance.AvgPnL)
		summary += fmt.Sprintf("- **最大盈利**: %.2f USDT\n", result.Performance.MaxProfit)
		summary += fmt.Sprintf("- **最大亏损**: %.2f USDT\n\n", result.Performance.MaxLoss)
	}

	summary += "### 规则违反情况\n\n"
	if result.Metrics != nil && result.Metrics.RuleViolations != nil {
		summary += fmt.Sprintf("- **总违反数**: %d\n", result.Metrics.RuleViolations.TotalViolations)
		summary += fmt.Sprintf("- **硬约束违反率**: %.2f%%\n", result.Metrics.RuleViolations.HardConstraintViolationRate)
		summary += fmt.Sprintf("- **风险控制违反率**: %.2f%%\n\n", result.Metrics.RuleViolations.RiskControlViolationRate)
	}

	summary += "### DEX验证情况\n\n"
	if result.DEXValidation != nil {
		summary += fmt.Sprintf("- **决策执行率**: %.2f%%\n", result.DEXValidation.DecisionExecutionRate)
		summary += fmt.Sprintf("- **平均滑点**: %.2f%%\n", result.DEXValidation.AverageSlippage*100)
		summary += fmt.Sprintf("- **平均执行延迟**: %d ms\n\n", result.DEXValidation.AverageExecutionDelay)
	}

	return summary
}

// generatePerformanceOverview 生成表现概览
func (r *MarkdownReporter) generatePerformanceOverview(result *review.ReviewResult) string {
	var overview string
	overview += "## 二、表现概览\n\n"

	if result.Performance == nil {
		overview += "无表现数据\n\n"
		return overview
	}

	perf := result.Performance

	overview += "### 基础统计\n\n"
	overview += fmt.Sprintf("- **总交易数**: %d\n", perf.TotalTrades)
	overview += fmt.Sprintf("- **开仓数**: %d\n", perf.OpenTrades)
	overview += fmt.Sprintf("- **平仓数**: %d\n", perf.CloseTrades)
	overview += fmt.Sprintf("- **完成交易数**: %d\n\n", perf.CompletedTrades)

	overview += "### 盈亏统计\n\n"
	overview += fmt.Sprintf("- **总盈亏**: %.2f USDT\n", perf.TotalPnL)
	overview += fmt.Sprintf("- **净盈亏**: %.2f USDT (扣除手续费)\n", perf.NetPnL)
	overview += fmt.Sprintf("- **平均盈亏**: %.2f USDT\n", perf.AvgPnL)
	overview += fmt.Sprintf("- **最大盈利**: %.2f USDT\n", perf.MaxProfit)
	overview += fmt.Sprintf("- **最大亏损**: %.2f USDT\n\n", perf.MaxLoss)

	overview += "### 胜率统计\n\n"
	overview += fmt.Sprintf("- **胜率**: %.2f%%\n", perf.WinRate)
	overview += fmt.Sprintf("- **盈利交易数**: %d\n", perf.WinTrades)
	overview += fmt.Sprintf("- **亏损交易数**: %d\n\n", perf.LossTrades)

	overview += "### 费用统计\n\n"
	overview += fmt.Sprintf("- **总费用**: %.2f USDT\n", perf.TotalFees)
	overview += fmt.Sprintf("- **平均费用**: %.2f USDT\n", perf.AvgFee)
	overview += fmt.Sprintf("- **费用比率**: %.2f%%\n\n", perf.FeeRatio)

	overview += "### 风险指标\n\n"
	overview += fmt.Sprintf("- **最大回撤**: %.2f USDT\n", perf.MaxDrawdown)
	overview += fmt.Sprintf("- **波动率**: %.2f\n", perf.Volatility)
	overview += fmt.Sprintf("- **Sharpe比率**: %.2f\n\n", perf.SharpeRatio)

	// 币种表现
	if len(perf.SymbolStats) > 0 {
		overview += "### 币种表现\n\n"
		overview += "| 币种 | 交易数 | 胜率 | 总盈亏 | 平均盈亏 |\n"
		overview += "|------|--------|------|--------|----------|\n"
		for symbol, stats := range perf.SymbolStats {
			overview += fmt.Sprintf("| %s | %d | %.2f%% | %.2f | %.2f |\n",
				symbol, stats.TotalTrades, stats.WinRate, stats.TotalPnL, stats.AvgPnL)
		}
		overview += "\n"
	}

	return overview
}

// generateDetailedAnalysis 生成详细分析
func (r *MarkdownReporter) generateDetailedAnalysis(result *review.ReviewResult) string {
	var analysis string
	analysis += "## 三、详细分析\n\n"

	// 错误分析
	analysis += "### 错误分析\n\n"
	if len(result.Violations) > 0 {
		analysis += fmt.Sprintf("共发现 %d 个规则违反。\n\n", len(result.Violations))

		// 按类型分组
		byType := make(map[string][]review.Violation)
		for _, violation := range result.Violations {
			byType[violation.Type] = append(byType[violation.Type], violation)
		}

		for ruleType, violations := range byType {
			analysis += fmt.Sprintf("#### %s\n\n", getRuleTypeName(ruleType))
			for i, violation := range violations {
				if i >= 10 {
					analysis += fmt.Sprintf("... 还有 %d 个违反\n\n", len(violations)-10)
					break
				}
				analysis += fmt.Sprintf("- **%s** (严重程度: %s): %s\n", violation.RuleName, getSeverityName(violation.Severity), violation.Description)
			}
			analysis += "\n"
		}
	} else {
		analysis += "未发现规则违反。\n\n"
	}

	// DEX验证分析
	analysis += "### DEX验证分析\n\n"
	if result.ExecutionQuality != nil {
		eq := result.ExecutionQuality
		analysis += fmt.Sprintf("- **匹配数**: %d\n", eq.TotalMatches)
		analysis += fmt.Sprintf("- **平均滑点**: %.2f%%\n", eq.AverageSlippage*100)
		analysis += fmt.Sprintf("- **最大滑点**: %.2f%%\n", eq.MaxSlippage*100)
		analysis += fmt.Sprintf("- **平均执行延迟**: %d ms\n", eq.AverageExecutionDelay)
		analysis += fmt.Sprintf("- **最大执行延迟**: %d ms\n\n", eq.MaxExecutionDelay)

		// 滑点分布
		if len(eq.SlippageDistribution) > 0 {
			analysis += "**滑点分布**:\n"
			for rangeLabel, count := range eq.SlippageDistribution {
				analysis += fmt.Sprintf("- %s: %d\n", rangeLabel, count)
			}
			analysis += "\n"
		}
	}

	if result.DEXValidation != nil {
		dv := result.DEXValidation
		analysis += fmt.Sprintf("- **决策执行率**: %.2f%%\n", dv.DecisionExecutionRate)
		analysis += fmt.Sprintf("- **未匹配决策数**: %d\n", dv.UnmatchedDecisions)
		analysis += fmt.Sprintf("- **未匹配DEX交易数**: %d\n", dv.UnmatchedDEXTrades)
		analysis += fmt.Sprintf("- **费用差异**: %.2f USDT\n\n", dv.FeeDifference)
	}

	return analysis
}

// generateRiskAnalysis 生成风险分析
func (r *MarkdownReporter) generateRiskAnalysis(result *review.ReviewResult) string {
	var risk string
	risk += "## 四、风险分析\n\n"

	if result.Metrics != nil && result.Metrics.RuleViolations != nil {
		rv := result.Metrics.RuleViolations
		risk += "### 规则违反风险\n\n"
		risk += fmt.Sprintf("- **硬约束违反率**: %.2f%%", rv.HardConstraintViolationRate)
		if rv.HardConstraintViolationRate > 0 {
			risk += " ⚠️"
		}
		risk += "\n"

		risk += fmt.Sprintf("- **风险控制违反率**: %.2f%%", rv.RiskControlViolationRate)
		if rv.RiskControlViolationRate > 0 {
			risk += " ⚠️"
		}
		risk += "\n\n"

		// 严重程度分布
		if len(rv.ViolationSeverityDistribution) > 0 {
			risk += "### 违反严重程度分布\n\n"
			for severity, count := range rv.ViolationSeverityDistribution {
				risk += fmt.Sprintf("- **%s**: %d\n", getSeverityName(severity), count)
			}
			risk += "\n"
		}
	}

	if result.Performance != nil {
		risk += "### 交易风险\n\n"
		risk += fmt.Sprintf("- **最大回撤**: %.2f USDT\n", result.Performance.MaxDrawdown)
		risk += fmt.Sprintf("- **波动率**: %.2f\n", result.Performance.Volatility)
		risk += fmt.Sprintf("- **Sharpe比率**: %.2f\n\n", result.Performance.SharpeRatio)
	}

	return risk
}

// generateRecommendations 生成改进建议
func (r *MarkdownReporter) generateRecommendations(result *review.ReviewResult) string {
	var recommendations string
	recommendations += "## 五、改进建议\n\n"

	// 基于违反情况生成建议
	if len(result.Violations) > 0 {
		recommendations += "### 规则遵守建议\n\n"
		recommendations += "发现规则违反，建议：\n\n"

		// 统计最常见的违反类型
		violationCounts := make(map[string]int)
		for _, violation := range result.Violations {
			violationCounts[violation.RuleName]++
		}

		for ruleName, count := range violationCounts {
			if count > 0 {
				recommendations += fmt.Sprintf("- **%s**: 发现 %d 次违反，建议加强检查和优化\n", ruleName, count)
			}
		}
		recommendations += "\n"
	}

	// 基于表现生成建议
	if result.Performance != nil {
		perf := result.Performance
		if perf.WinRate < 50 {
			recommendations += "### 交易策略建议\n\n"
			recommendations += "- 胜率较低，建议优化入场时机和止损策略\n\n"
		}

		if perf.TotalPnL < 0 {
			recommendations += "### 盈亏优化建议\n\n"
			recommendations += "- 总体亏损，建议减少交易频率，提高交易质量\n\n"
		}
	}

	// 基于DEX验证生成建议
	if result.DEXValidation != nil {
		if result.DEXValidation.DecisionExecutionRate < 90 {
			recommendations += "### 执行优化建议\n\n"
			recommendations += "- 决策执行率较低，建议检查系统执行逻辑\n\n"
		}

		if result.DEXValidation.AverageSlippage > 0.01 {
			recommendations += "### 滑点优化建议\n\n"
			recommendations += "- 平均滑点较高，建议优化订单执行策略\n\n"
		}
	}

	return recommendations
}

// generateDataAppendix 生成数据附录
func (r *MarkdownReporter) generateDataAppendix(result *review.ReviewResult) string {
	var appendix string
	appendix += "## 六、数据附录\n\n"

	appendix += "### 标准化指标\n\n"
	if result.Metrics != nil {
		metricsJSON, err := json.MarshalIndent(result.Metrics, "", "  ")
		if err == nil {
			appendix += "```json\n"
			appendix += string(metricsJSON)
			appendix += "\n```\n\n"
		}
	}

	appendix += "### 复盘结果摘要\n\n"
	summary := map[string]interface{}{
		"trader_id":     result.TraderID,
		"start_time":    result.StartTime.Format(time.RFC3339),
		"end_time":      result.EndTime.Format(time.RFC3339),
		"status":        result.Status,
		"duration_ms":   result.Duration.Milliseconds(),
		"total_trades": 0,
		"total_pnl":     0.0,
		"win_rate":      0.0,
	}

	if result.Performance != nil {
		summary["total_trades"] = result.Performance.TotalTrades
		summary["total_pnl"] = result.Performance.TotalPnL
		summary["win_rate"] = result.Performance.WinRate
	}

	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err == nil {
		appendix += "```json\n"
		appendix += string(summaryJSON)
		appendix += "\n```\n\n"
	}

	return appendix
}

// getRuleTypeName 获取规则类型的中文名称
func getRuleTypeName(ruleType string) string {
	switch ruleType {
	case "hard_constraint":
		return "硬约束"
	case "risk_control":
		return "风险控制"
	case "position_management":
		return "持仓管理"
	default:
		return ruleType
	}
}

// getSeverityName 获取严重程度的中文名称
func getSeverityName(severity string) string {
	switch severity {
	case "critical":
		return "严重"
	case "high":
		return "高"
	case "medium":
		return "中等"
	case "low":
		return "低"
	default:
		return severity
	}
}


package metrics

import (
	"fmt"
	"log"

	"nofx/review"
)

// ViolationMetricsCalculator 规则违反指标计算器
type ViolationMetricsCalculator struct{}

// NewViolationMetricsCalculator 创建规则违反指标计算器
func NewViolationMetricsCalculator() *ViolationMetricsCalculator {
	return &ViolationMetricsCalculator{}
}

// Calculate 计算规则违反指标
func (c *ViolationMetricsCalculator) Calculate(violations []review.Violation, totalDecisions int) (*review.RuleViolationMetrics, error) {
	log.Printf("📊 开始计算规则违反指标 (违反数: %d, 总决策数: %d)", len(violations), totalDecisions)

	if totalDecisions == 0 {
		return &review.RuleViolationMetrics{
			ViolationSeverityDistribution: make(map[string]int),
		}, nil
	}

	metrics := &review.RuleViolationMetrics{
		TotalViolations:               len(violations),
		ViolationSeverityDistribution: make(map[string]int),
	}

	// 按类型统计违反次数
	hardConstraintViolations := 0
	riskControlViolations := 0
	positionManagementViolations := 0

	// 按严重程度统计分布
	for _, violation := range violations {
		// 统计类型
		switch violation.Type {
		case "hard_constraint":
			hardConstraintViolations++
		case "risk_control":
			riskControlViolations++
		case "position_management":
			positionManagementViolations++
		}

		// 统计严重程度分布
		metrics.ViolationSeverityDistribution[violation.Severity]++
	}

	// 计算违反率
	metrics.HardConstraintViolationRate = (float64(hardConstraintViolations) / float64(totalDecisions)) * 100
	metrics.RiskControlViolationRate = (float64(riskControlViolations) / float64(totalDecisions)) * 100
	metrics.PositionManagementViolationRate = (float64(positionManagementViolations) / float64(totalDecisions)) * 100

	log.Printf("✅ 规则违反指标计算完成 (硬约束违反率: %.2f%%, 风险控制违反率: %.2f%%)",
		metrics.HardConstraintViolationRate, metrics.RiskControlViolationRate)

	return metrics, nil
}

// CalculateByRule 按规则统计违反情况
func (c *ViolationMetricsCalculator) CalculateByRule(violations []review.Violation) map[string]*RuleViolationStats {
	statsByRule := make(map[string]*RuleViolationStats)

	for _, violation := range violations {
		ruleID := violation.RuleID
		if _, exists := statsByRule[ruleID]; !exists {
			statsByRule[ruleID] = &RuleViolationStats{
				RuleID:       ruleID,
				RuleName:     violation.RuleName,
				RuleType:     violation.Type,
				Severity:     violation.Severity,
				ViolationCount: 0,
			}
		}

		stats := statsByRule[ruleID]
		stats.ViolationCount++
	}

	return statsByRule
}

// RuleViolationStats 单个规则的违反统计
type RuleViolationStats struct {
	RuleID         string
	RuleName       string
	RuleType       string
	Severity       string
	ViolationCount int
}

// FormatViolationMetrics 格式化违反指标为字符串（用于报告）
func FormatViolationMetrics(metrics *review.RuleViolationMetrics) string {
	if metrics == nil {
		return "无违反指标数据"
	}

	result := fmt.Sprintf(`
## 规则违反指标

- **总违反数**: %d
- **硬约束违反率**: %.2f%%
- **风险控制违反率**: %.2f%%
- **持仓管理违反率**: %.2f%%

### 严重程度分布

`, metrics.TotalViolations,
		metrics.HardConstraintViolationRate,
		metrics.RiskControlViolationRate,
		metrics.PositionManagementViolationRate)

	// 严重程度排序：critical > high > medium > low
	severityOrder := []string{"critical", "high", "medium", "low"}
	for _, severity := range severityOrder {
		count, exists := metrics.ViolationSeverityDistribution[severity]
		if exists {
			result += fmt.Sprintf("- **%s**: %d\n", getSeverityName(severity), count)
		}
	}

	return result
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


package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Phase5Result struct {
	RootCauseAnalysis    RootCauseAnalysis
	PromptCorrectionPlan PromptCorrectionPlan
	ReportPath           string
}

type RootCauseAnalysis struct {
	PrimaryRootCauses   []RootCause
	SecondaryRootCauses []RootCause
	ContributingFactors []ContributingFactor
}

type RootCause struct {
	Category   string
	Description string
	Severity   string
	Impact     string
}

type ContributingFactor struct {
	Factor      string
	Description string
}

type PromptCorrectionPlan struct {
	HighPriorityCorrections   []PromptCorrection
	MediumPriorityCorrections []PromptCorrection
	LowPriorityCorrections    []PromptCorrection
}

type PromptCorrection struct {
	Issue          string
	Location       string
	CurrentText    string
	CorrectedText  string
	Rationale      string
	ExpectedImpact string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run compare_prompt_corrections.go <phase5_result_v1.json> <phase5_result_v2.json>")
		fmt.Println("Example: go run compare_prompt_corrections.go phase5_result_400_1892.json phase5_result_400_1892_v2.json")
		os.Exit(1)
	}

	v1File := os.Args[1]
	v2File := os.Args[2]

	fmt.Println("=== 比较两次Prompt修正结果 ===\n")

	// 加载v1结果
	v1Data, err := ioutil.ReadFile(v1File)
	if err != nil {
		fmt.Printf("❌ 读取v1文件失败: %v\n", err)
		os.Exit(1)
	}

	var v1Result Phase5Result
	if err := json.Unmarshal(v1Data, &v1Result); err != nil {
		fmt.Printf("❌ 解析v1 JSON失败: %v\n", err)
		os.Exit(1)
	}

	// 加载v2结果
	v2Data, err := ioutil.ReadFile(v2File)
	if err != nil {
		fmt.Printf("❌ 读取v2文件失败: %v\n", err)
		os.Exit(1)
	}

	var v2Result Phase5Result
	if err := json.Unmarshal(v2Data, &v2Result); err != nil {
		fmt.Printf("❌ 解析v2 JSON失败: %v\n", err)
		os.Exit(1)
	}

	// 比较根因分析
	fmt.Println("## 1. 根因分析比较\n")
	compareRootCauses(v1Result.RootCauseAnalysis, v2Result.RootCauseAnalysis)

	// 比较修正方案
	fmt.Println("\n## 2. Prompt修正方案比较\n")
	compareCorrectionPlans(v1Result.PromptCorrectionPlan, v2Result.PromptCorrectionPlan)

	// 生成比较报告
	reportPath := "prompt_correction_comparison.md"
	generateComparisonReport(v1Result, v2Result, reportPath)
	fmt.Printf("\n✅ 比较报告已生成: %s\n", reportPath)
}

func compareRootCauses(v1, v2 RootCauseAnalysis) {
	fmt.Println("### 1.1 主要根因比较\n")
	fmt.Printf("| 版本 | 数量 | 根因列表 |\n")
	fmt.Printf("|------|------|----------|\n")
	fmt.Printf("| v1 | %d | ", len(v1.PrimaryRootCauses))
	for i, cause := range v1.PrimaryRootCauses {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", cause.Description)
	}
	fmt.Printf(" |\n")

	fmt.Printf("| v2 | %d | ", len(v2.PrimaryRootCauses))
	for i, cause := range v2.PrimaryRootCauses {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s", cause.Description)
	}
	fmt.Printf(" |\n\n")

	// 检查是否有新增或删除的根因
	v1Map := make(map[string]bool)
	for _, cause := range v1.PrimaryRootCauses {
		v1Map[cause.Description] = true
	}

	v2Map := make(map[string]bool)
	for _, cause := range v2.PrimaryRootCauses {
		v2Map[cause.Description] = true
	}

	var newRootCauses []string
	var removedRootCauses []string

	for desc := range v2Map {
		if !v1Map[desc] {
			newRootCauses = append(newRootCauses, desc)
		}
	}

	for desc := range v1Map {
		if !v2Map[desc] {
			removedRootCauses = append(removedRootCauses, desc)
		}
	}

	if len(newRootCauses) > 0 {
		fmt.Println("**v2新增的根因**:")
		for _, cause := range newRootCauses {
			fmt.Printf("- %s\n", cause)
		}
		fmt.Println()
	}

	if len(removedRootCauses) > 0 {
		fmt.Println("**v2删除的根因**:")
		for _, cause := range removedRootCauses {
			fmt.Printf("- %s\n", cause)
		}
		fmt.Println()
	}

	if len(newRootCauses) == 0 && len(removedRootCauses) == 0 {
		fmt.Println("✅ 主要根因保持一致\n")
	}
}

func compareCorrectionPlans(v1, v2 PromptCorrectionPlan) {
	fmt.Println("### 2.1 修正方案数量比较\n")
	fmt.Printf("| 优先级 | v1数量 | v2数量 | 变化 |\n")
	fmt.Printf("|--------|--------|--------|------|\n")
	fmt.Printf("| 高优先级 | %d | %d | %+d |\n",
		len(v1.HighPriorityCorrections),
		len(v2.HighPriorityCorrections),
		len(v2.HighPriorityCorrections)-len(v1.HighPriorityCorrections))
	fmt.Printf("| 中优先级 | %d | %d | %+d |\n",
		len(v1.MediumPriorityCorrections),
		len(v2.MediumPriorityCorrections),
		len(v2.MediumPriorityCorrections)-len(v1.MediumPriorityCorrections))
	fmt.Printf("| 低优先级 | %d | %d | %+d |\n\n",
		len(v1.LowPriorityCorrections),
		len(v2.LowPriorityCorrections),
		len(v2.LowPriorityCorrections)-len(v1.LowPriorityCorrections))

	// 比较高优先级修正
	fmt.Println("### 2.2 高优先级修正比较\n")
	v1HighMap := make(map[string]PromptCorrection)
	for _, corr := range v1.HighPriorityCorrections {
		v1HighMap[corr.Issue] = corr
	}

	v2HighMap := make(map[string]PromptCorrection)
	for _, corr := range v2.HighPriorityCorrections {
		v2HighMap[corr.Issue] = corr
	}

	var commonIssues []string
	var v1OnlyIssues []string
	var v2OnlyIssues []string

	for issue := range v1HighMap {
		if _, exists := v2HighMap[issue]; exists {
			commonIssues = append(commonIssues, issue)
		} else {
			v1OnlyIssues = append(v1OnlyIssues, issue)
		}
	}

	for issue := range v2HighMap {
		if _, exists := v1HighMap[issue]; !exists {
			v2OnlyIssues = append(v2OnlyIssues, issue)
		}
	}

	fmt.Printf("**共同修正项** (%d个):\n", len(commonIssues))
	for _, issue := range commonIssues {
		fmt.Printf("- %s\n", issue)
	}
	fmt.Println()

	if len(v1OnlyIssues) > 0 {
		fmt.Printf("**仅在v1中的修正项** (%d个\n", len(v1OnlyIssues))
		for _, issue := range v1OnlyIssues {
			fmt.Printf("- %s\n", issue)
		}
		fmt.Println()
	}

	if len(v2OnlyIssues) > 0 {
		fmt.Printf("**仅在v2中的修正项** (%d):\n", len(v2OnlyIssues))
		for _, issue := range v2OnlyIssues {
			fmt.Printf("- %s\n", issue)
		}
		fmt.Println()
	}

	// 比较修正内容的变化
	if len(commonIssues) > 0 {
		fmt.Println("### 2.3 共同修正项的修正内容比较\n")
		for _, issue := range commonIssues {
			v1Corr := v1HighMap[issue]
			v2Corr := v2HighMap[issue]
			fmt.Printf("#### %s\n\n", issue)
			if v1Corr.CorrectedText != v2Corr.CorrectedText {
				fmt.Println("**修正内容有变化**:")
				fmt.Printf("- v1: %s\n", truncate(v1Corr.CorrectedText, 100))
				fmt.Printf("- v2: %s\n", truncate(v2Corr.CorrectedText, 100))
				fmt.Println()
			} else {
				fmt.Println("✅ 修正内容一致\n")
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func generateComparisonReport(v1, v2 Phase5Result, reportPath string) {
	file, err := os.Create(reportPath)
	if err != nil {
		fmt.Printf("❌ 创建报告文件失败: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "# Prompt修正结果比较报告\n\n")
	fmt.Fprintf(file, "**比较时间**: %s\n\n", getCurrentTime())
	fmt.Fprintf(file, "**v1文件**: phase5_result_400_1892.json\n")
	fmt.Fprintf(file, "**v2文件**: phase5_result_400_1892_v2.json\n\n")

	// 根因比较
	fmt.Fprintf(file, "## 1. 根因分析比较\n\n")
	fmt.Fprintf(file, "### 1.1 主要根因\n\n")
	fmt.Fprintf(file, "| 版本 | 数量 |\n")
	fmt.Fprintf(file, "|------|------|\n")
	fmt.Fprintf(file, "| v1 | %d |\n", len(v1.RootCauseAnalysis.PrimaryRootCauses))
	fmt.Fprintf(file, "| v2 | %d |\n\n", len(v2.RootCauseAnalysis.PrimaryRootCauses))

	fmt.Fprintf(file, "**v1主要根因**:\n")
	for i, cause := range v1.RootCauseAnalysis.PrimaryRootCauses {
		fmt.Fprintf(file, "%d. %s [%s]\n", i+1, cause.Description, cause.Severity)
	}
	fmt.Fprintf(file, "\n")

	fmt.Fprintf(file, "**v2主要根因**:\n")
	for i, cause := range v2.RootCauseAnalysis.PrimaryRootCauses {
		fmt.Fprintf(file, "%d. %s [%s]\n", i+1, cause.Description, cause.Severity)
	}
	fmt.Fprintf(file, "\n")

	// 修正方案比较
	fmt.Fprintf(file, "## 2. Prompt修正方案比较\n\n")
	fmt.Fprintf(file, "### 2.1 修正方案数量\n\n")
	fmt.Fprintf(file, "| 优先级 | v1 | v2 | 变化 |\n")
	fmt.Fprintf(file, "|--------|----|----|------|\n")
	fmt.Fprintf(file, "| 高优先级 | %d | %d | %+d |\n",
		len(v1.PromptCorrectionPlan.HighPriorityCorrections),
		len(v2.PromptCorrectionPlan.HighPriorityCorrections),
		len(v2.PromptCorrectionPlan.HighPriorityCorrections)-len(v1.PromptCorrectionPlan.HighPriorityCorrections))
	fmt.Fprintf(file, "| 中优先级 | %d | %d | %+d |\n",
		len(v1.PromptCorrectionPlan.MediumPriorityCorrections),
		len(v2.PromptCorrectionPlan.MediumPriorityCorrections),
		len(v2.PromptCorrectionPlan.MediumPriorityCorrections)-len(v1.PromptCorrectionPlan.MediumPriorityCorrections))
	fmt.Fprintf(file, "| 低优先级 | %d | %d | %+d |\n\n",
		len(v1.PromptCorrectionPlan.LowPriorityCorrections),
		len(v2.PromptCorrectionPlan.LowPriorityCorrections),
		len(v2.PromptCorrectionPlan.LowPriorityCorrections)-len(v1.PromptCorrectionPlan.LowPriorityCorrections))

	// v1高优先级修正
	fmt.Fprintf(file, "### 2.2 v1高优先级修正\n\n")
	for i, corr := range v1.PromptCorrectionPlan.HighPriorityCorrections {
		fmt.Fprintf(file, "%d. **%s**\n", i+1, corr.Issue)
		fmt.Fprintf(file, "   - 位置: %s\n",
			corr.Location)
		fmt.Fprintf(file, "   - 预期影响: %s\n\n", corr.ExpectedImpact)
	}

	// v2高优先级修正
	fmt.Fprintf(file, "### 2.3 v2高优先级修正\n\n")
	for i, corr := range v2.PromptCorrectionPlan.HighPriorityCorrections {
		fmt.Fprintf(file, "%d. **%s**\n", i+1, corr.Issue)
		fmt.Fprintf(file, "   - 在Prompt位置: %s\n",
			corr.Location)
		fmt.Fprintf(file, "   - 预期影响: %s\n\n", corr.ExpectedImpact)
	}

	// 总结
	fmt.Fprintf(file, "## 3. 总结\n\n")
	fmt.Fprintf(file, "### 3.1 一致性分析\n\n")
	if len(v1.RootCauseAnalysis.PrimaryRootCauses) == len(v2.RootCauseAnalysis.PrimaryRootCauses) {
		fmt.Fprintf(file, "✅ 主要根因数量一致\n\n")
	} else {
		fmt.Fprintf(file, "⚠️ 主要根因数量不一致: v1有%d个，v2有%d个\n\n",
			len(v1.RootCauseAnalysis.PrimaryRootCauses),
			len(v2.RootCauseAnalysis.PrimaryRootCauses))
	}

	if len(v1.PromptCorrectionPlan.HighPriorityCorrections) == len(v2.PromptCorrectionPlan.HighPriorityCorrections) {
		fmt.Fprintf(file, "✅ 高优先级修正数量一致\n\n")
	} else {
		fmt.Fprintf(file, "⚠️ 高优先级修正数量不一致: v1有%d个，v2有%d个\n\n",
			len(v1.PromptCorrectionPlan.HighPriorityCorrections),
			len(v2.PromptCorrectionPlan.HighPriorityCorrections))
	}

	fmt.Fprintf(file, "### 3.2 建议\n\n")
	fmt.Fprintf(file, "1. 检查两次分析使用的数据是否一致\n")
	fmt.Fprintf(file, "2. 如果数据一致，差异可能来自分析逻辑的改进\n")
	fmt.Fprintf(file, "3. 综合两次分析的结果，选择最合适的修正方案\n\n")
}

func getCurrentTime() string {
	// 简化版本，实际应该使用time包
	return "2025-11-25"
}


package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

// ReportFormat 报告格式类型
type ReportFormat string

const (
	ReportJSON  ReportFormat = "json"
	ReportJUnit ReportFormat = "junit"
	ReportHTML  ReportFormat = "html"
)

// WriteReport 生成结构化报告
func WriteReport(path string, fmtType ReportFormat, steps []validate.StepResult, spans []trace.Span, gateResult *html.GateResult) error {
	switch fmtType {
	case ReportJSON:
		return writeJSONReport(path, steps)
	case ReportJUnit:
		return writeJUnitReport(path, steps)
	case ReportHTML:
		return writeHTMLReport(path, steps, spans, gateResult)
	default:
		return fmt.Errorf("不支持的报告格式: %s", fmtType)
	}
}

// CoverageSummary 覆盖度总结
type CoverageSummary struct {
	StepsTotal       int               `json:"stepsTotal"`
	StepsPass        int               `json:"stepsPass"`
	StepsFail        int               `json:"stepsFail"`
	StepsSkip        int               `json:"stepsSkip"`
	ConditionsTotal  int               `json:"conditionsTotal"`
	ConditionsPass   int               `json:"conditionsPass"`
	ConditionsFail   int               `json:"conditionsFail"`
	ConditionsSkip   int               `json:"conditionsSkip"`
	UncoveredSteps   []string          `json:"uncoveredSteps"`
	CoverageRate     float64           `json:"coverageRate"`
	ServiceCoverage  map[string]int    `json:"serviceCoverage"`
}

// writeJSONReport 写入 JSON 格式报告
func writeJSONReport(path string, steps []validate.StepResult) error {
	summary := calculateCoverageSummary(steps)
	
	report := struct {
		Timestamp   time.Time             `json:"timestamp"`
		TotalSteps  int                   `json:"totalSteps"`
		PassedSteps int                   `json:"passedSteps"`
		FailedSteps int                   `json:"failedSteps"`
		Success     bool                  `json:"success"`
		Steps       []validate.StepResult `json:"steps"`
		Summary     CoverageSummary       `json:"summary"`
	}{
		Timestamp:   time.Now(),
		TotalSteps:  len(steps),
		PassedSteps: 0,
		FailedSteps: 0,
		Success:     true,
		Steps:       steps,
		Summary:     summary,
	}

	for _, s := range steps {
		if s.Status == "PASS" {
			report.PassedSteps++
		} else {
			report.FailedSteps++
			report.Success = false
		}
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 报告失败: %w", err)
	}

	return os.WriteFile(path, b, 0644)
}

// writeJUnitReport 写入 JUnit XML 格式报告
func writeJUnitReport(path string, steps []validate.StepResult) error {
	var sb strings.Builder
	fails := 0
	for _, s := range steps {
		if s.Status == "FAIL" {
			fails++
		}
	}

	summary := calculateCoverageSummary(steps)

	// JUnit XML header
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`<testsuite name="flowspec-validation" tests="%d" failures="%d" time="0">`, len(steps), fails))
	sb.WriteString("\n")

	// 添加覆盖度总结到 properties
	sb.WriteString("  <properties>\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.stepsTotal" value="%d"/>`, summary.StepsTotal))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.stepsPass" value="%d"/>`, summary.StepsPass))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.stepsFail" value="%d"/>`, summary.StepsFail))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.stepsSkip" value="%d"/>`, summary.StepsSkip))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.conditionsTotal" value="%d"/>`, summary.ConditionsTotal))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.conditionsPass" value="%d"/>`, summary.ConditionsPass))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.conditionsFail" value="%d"/>`, summary.ConditionsFail))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.conditionsSkip" value="%d"/>`, summary.ConditionsSkip))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`    <property name="coverage.coverageRate" value="%.2f"/>`, summary.CoverageRate))
	sb.WriteString("\n")
	sb.WriteString("  </properties>\n")

	// Test cases
	for _, s := range steps {
		sb.WriteString(fmt.Sprintf(`  <testcase name="%s" classname="%s">`, xmlEscape(s.Step), xmlEscape(s.Call)))
		
		if s.Status == "FAIL" {
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf(`    <failure message="%s" type="ValidationFailure">%s</failure>`,
				xmlEscape(s.Message), xmlEscape(s.Message)))
			sb.WriteString("\n  ")
		}
		
		// 添加条件详情到 system-out
		if len(s.Conditions) > 0 {
			sb.WriteString("\n")
			conditionsJSON, _ := json.Marshal(s.Conditions)
			sb.WriteString(fmt.Sprintf(`    <system-out><![CDATA[%s]]></system-out>`, conditionsJSON))
			sb.WriteString("\n  ")
		}
		
		sb.WriteString("</testcase>")
		sb.WriteString("\n")
	}

	// 添加详细的覆盖度总结到 system-out
	if len(summary.UncoveredSteps) > 0 || len(summary.ServiceCoverage) > 0 {
		sb.WriteString("  <system-out><![CDATA[\n")
		summaryJSON, _ := json.MarshalIndent(summary, "", "    ")
		sb.WriteString(string(summaryJSON))
		sb.WriteString("\n  ]]></system-out>\n")
	}

	sb.WriteString("</testsuite>")
	sb.WriteString("\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// writeHTMLReport 写入 HTML 格式报告
func writeHTMLReport(path string, steps []validate.StepResult, spans []trace.Span, gateResult *html.GateResult) error {
	// Convert trace spans to HTML span info
	var spanInfos []html.SpanInfo
	for _, span := range spans {
		spanInfos = append(spanInfos, html.SpanInfo{
			Service:    span.Service,
			Name:       span.Name,
			StartNanos: span.StartNanos,
			EndNanos:   span.EndNanos,
		})
	}

	// Build HTML data with gate result and CE edition
	data := html.BuildHTMLData(steps, spanInfos, gateResult, "CE")

	// Write HTML report
	return html.WriteHTMLReport(path, data)
}

// calculateCoverageSummary 计算覆盖度总结
func calculateCoverageSummary(steps []validate.StepResult) CoverageSummary {
	summary := CoverageSummary{
		ServiceCoverage: make(map[string]int),
		UncoveredSteps:  []string{},
	}

	for _, step := range steps {
		summary.StepsTotal++
		
		switch step.Status {
		case "PASS":
			summary.StepsPass++
		case "FAIL":
			summary.StepsFail++
			summary.UncoveredSteps = append(summary.UncoveredSteps, step.Step)
		case "SKIP":
			summary.StepsSkip++
		}

		// 统计服务覆盖度
		if step.Call != "" {
			parts := strings.Split(step.Call, ".")
			if len(parts) >= 2 {
				service := parts[0]
				summary.ServiceCoverage[service]++
			}
		}

		// 统计条件覆盖度
		for _, condition := range step.Conditions {
			summary.ConditionsTotal++
			switch condition.Status {
			case "PASS":
				summary.ConditionsPass++
			case "FAIL":
				summary.ConditionsFail++
			case "SKIP":
				summary.ConditionsSkip++
			}
		}
	}

	// 计算覆盖率
	if summary.StepsTotal > 0 {
		summary.CoverageRate = float64(summary.StepsPass) / float64(summary.StepsTotal) * 100
	}

	return summary
}

// xmlEscape 转义 XML 特殊字符
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
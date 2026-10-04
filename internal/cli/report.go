// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

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
func WriteReport(path string, fmtType ReportFormat, steps []validate.StepResult, spans []trace.Span, gateResult *html.GateResult, inputs ...*html.InputBinding) error {
	switch fmtType {
	case ReportJSON:
		return writeJSONReport(path, steps, gateResult, inputs...)
	case ReportJUnit:
		return writeJUnitReport(path, steps, gateResult, inputs...)
	case ReportHTML:
		return writeHTMLReport(path, steps, spans, gateResult, inputs...)
	default:
		return fmt.Errorf("unsupported report format: %s", fmtType)
	}
}

// CoverageSummary 覆盖度总结
type CoverageSummary struct {
	StepsTotal      int            `json:"stepsTotal"`
	StepsPass       int            `json:"stepsPass"`
	StepsFail       int            `json:"stepsFail"`
	StepsSkip       int            `json:"stepsSkip"`
	ConditionsTotal int            `json:"conditionsTotal"`
	ConditionsPass  int            `json:"conditionsPass"`
	ConditionsFail  int            `json:"conditionsFail"`
	ConditionsSkip  int            `json:"conditionsSkip"`
	UncoveredSteps  []string       `json:"uncoveredSteps"`
	CoverageRate    float64        `json:"coverageRate"`
	ServiceCoverage map[string]int `json:"serviceCoverage"`
	// Baseline comparison fields
	BaselineStepsCoverage  float64 `json:"baselineStepsCoverage,omitempty"`
	StepsDeltaAbs          float64 `json:"stepsDeltaAbs,omitempty"`
	StepsDeltaPct          float64 `json:"stepsDeltaPct,omitempty"`
	BaselineConditionsRate float64 `json:"baselineConditionsRate,omitempty"`
	ConditionsDeltaAbs     float64 `json:"conditionsDeltaAbs,omitempty"`
	ConditionsDeltaPct     float64 `json:"conditionsDeltaPct,omitempty"`
}

// writeJSONReport 写入 JSON 格式报告
func writeJSONReport(path string, steps []validate.StepResult, gateResult *html.GateResult, inputs ...*html.InputBinding) error {
	summary := calculateCoverageSummary(steps)

	// Add baseline comparison fields if available
	if gateResult != nil && gateResult.Details != nil {
		if val, ok := gateResult.Details["baselineStepsCoverage"].(float64); ok {
			summary.BaselineStepsCoverage = val
		}
		if val, ok := gateResult.Details["stepsDeltaAbs"].(float64); ok {
			summary.StepsDeltaAbs = val
		}
		if val, ok := gateResult.Details["stepsDeltaPct"].(float64); ok {
			summary.StepsDeltaPct = val
		}
		if val, ok := gateResult.Details["baselineConditionsRate"].(float64); ok {
			summary.BaselineConditionsRate = val
		}
		if val, ok := gateResult.Details["conditionsDeltaAbs"].(float64); ok {
			summary.ConditionsDeltaAbs = val
		}
		if val, ok := gateResult.Details["conditionsDeltaPct"].(float64); ok {
			summary.ConditionsDeltaPct = val
		}
	}

	report := struct {
		validate.Outcome
		ExitCode    int                   `json:"exitCode"`
		Inputs      *html.InputBinding    `json:"inputs,omitempty"`
		Timestamp   time.Time             `json:"timestamp"`
		TotalSteps  int                   `json:"totalSteps"`
		PassedSteps int                   `json:"passedSteps"`
		FailedSteps int                   `json:"failedSteps"`
		Steps       []validate.StepResult `json:"steps"`
		Summary     CoverageSummary       `json:"summary"`
		GateResult  *html.GateResult      `json:"gateResult,omitempty"`
	}{
		Outcome:     reportOutcome(steps, gateResult),
		ExitCode:    outcomeExitCode(reportOutcome(steps, gateResult)),
		Timestamp:   time.Now(),
		TotalSteps:  len(steps),
		PassedSteps: 0,
		FailedSteps: 0,
		Steps:       steps,
		Summary:     summary,
		GateResult:  gateResult,
	}
	if len(inputs) > 0 {
		report.Inputs = inputs[0]
	}

	for _, s := range steps {
		if validate.StepPassed(s) {
			report.PassedSteps++
		} else {
			report.FailedSteps++
		}
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON report: %w", err)
	}

	return os.WriteFile(path, b, 0644)
}

// writeJUnitReport 写入 JUnit XML 格式报告
func writeJUnitReport(path string, steps []validate.StepResult, gateResult *html.GateResult, inputs ...*html.InputBinding) error {
	var sb strings.Builder
	fails := 0
	for _, s := range steps {
		if !validate.StepPassed(s) {
			fails++
		}
	}
	gateFailed := gateResult != nil && gateResult.Checked && !gateResult.Passed
	tests := len(steps)
	if len(steps) == 0 {
		fails++
		tests++
	}
	if gateFailed {
		fails++
		tests++
	}

	summary := calculateCoverageSummary(steps)

	// JUnit XML header
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `<testsuite name="flowspec-validation" tests="%d" failures="%d" time="0">`, tests, fails)
	sb.WriteString("\n")

	// 添加覆盖度总结到 properties
	sb.WriteString("  <properties>\n")
	if len(inputs) > 0 && inputs[0] != nil {
		binding, err := json.Marshal(inputs[0])
		if err != nil {
			return fmt.Errorf("failed to serialize report inputs: %w", err)
		}
		fmt.Fprintf(&sb, "    <property name=\"result.inputs\" value=\"%s\"/>\n", xmlEscape(string(binding)))
	}
	fmt.Fprintf(&sb, "    <property name=\"result.exitCode\" value=\"%d\"/>\n", outcomeExitCode(reportOutcome(steps, gateResult)))
	fmt.Fprintf(&sb, `    <property name="coverage.stepsTotal" value="%d"/>`, summary.StepsTotal)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.stepsPass" value="%d"/>`, summary.StepsPass)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.stepsFail" value="%d"/>`, summary.StepsFail)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.stepsSkip" value="%d"/>`, summary.StepsSkip)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.conditionsTotal" value="%d"/>`, summary.ConditionsTotal)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.conditionsPass" value="%d"/>`, summary.ConditionsPass)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.conditionsFail" value="%d"/>`, summary.ConditionsFail)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.conditionsSkip" value="%d"/>`, summary.ConditionsSkip)
	sb.WriteString("\n")
	fmt.Fprintf(&sb, `    <property name="coverage.coverageRate" value="%.2f"/>`, summary.CoverageRate)
	sb.WriteString("\n")

	// Add baseline comparison properties if available
	if gateResult != nil && gateResult.Details != nil {
		if val, ok := gateResult.Details["baselineStepsCoverage"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.stepsCoverage" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
		if val, ok := gateResult.Details["stepsDeltaAbs"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.stepsDeltaAbs" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
		if val, ok := gateResult.Details["stepsDeltaPct"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.stepsDeltaPct" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
		if val, ok := gateResult.Details["baselineConditionsRate"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.conditionsRate" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
		if val, ok := gateResult.Details["conditionsDeltaAbs"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.conditionsDeltaAbs" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
		if val, ok := gateResult.Details["conditionsDeltaPct"].(float64); ok {
			fmt.Fprintf(&sb, `    <property name="baseline.conditionsDeltaPct" value="%.2f"/>`, val)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("  </properties>\n")

	if gateFailed {
		sb.WriteString("  <testcase name=\"threshold-policy\" classname=\"flowspec\"><failure type=\"GateFailure\" message=\"threshold policy failed\"/></testcase>\n")
	}
	if len(steps) == 0 {
		sb.WriteString("  <testcase name=\"validation-result\" classname=\"flowspec\"><failure type=\"ValidationFailure\" message=\"no evaluated steps\"/></testcase>\n")
	}
	// Test cases
	for _, s := range steps {
		fmt.Fprintf(&sb, `  <testcase name="%s" classname="%s">`, xmlEscape(s.Step), xmlEscape(s.Call))

		if !validate.StepPassed(s) {
			sb.WriteString("\n")
			fmt.Fprintf(&sb, `    <failure message="%s" type="ValidationFailure">%s</failure>`,
				xmlEscape(s.Message), xmlEscape(s.Message))
			sb.WriteString("\n  ")
		}

		// 添加条件详情到 system-out
		if len(s.Conditions) > 0 {
			sb.WriteString("\n")
			conditionsJSON, _ := json.Marshal(s.Conditions)
			fmt.Fprintf(&sb, `    <system-out><![CDATA[%s]]></system-out>`, conditionsJSON)
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
func writeHTMLReport(path string, steps []validate.StepResult, spans []trace.Span, gateResult *html.GateResult, inputs ...*html.InputBinding) error {
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
	data := html.BuildHTMLData(steps, spanInfos, gateResult)
	data.ExitCode = outcomeExitCode(reportOutcome(steps, gateResult))
	if len(inputs) > 0 {
		data.Inputs = inputs[0]
	}

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

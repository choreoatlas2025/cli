// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/fileio"
	"strings"

	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/result"
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
func WriteReport(path string, fmtType ReportFormat, steps []validate.StepResult, spans []trace.Span, gateResult *result.GateResult, inputs ...*result.InputBinding) error {
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
type CoverageSummary = result.CoverageSummary

// writeJSONReport 写入 JSON 格式报告
func writeJSONReport(path string, steps []validate.StepResult, gateResult *result.GateResult, inputs ...*result.InputBinding) error {
	var binding *result.InputBinding
	if len(inputs) > 0 {
		binding = inputs[0]
	}
	report := result.New(steps, gateResult, binding)

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON report: %w", err)
	}

	return fileio.WriteFile(path, b, 0644)
}

// writeJUnitReport 写入 JUnit XML 格式报告
func writeJUnitReport(path string, steps []validate.StepResult, gateResult *result.GateResult, inputs ...*result.InputBinding) error {
	var sb strings.Builder
	report := result.New(steps, gateResult, nil)
	fails := report.FailedSteps
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

	summary := report.Summary

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
	fmt.Fprintf(&sb, "    <property name=\"result.exitCode\" value=\"%d\"/>\n", report.ExitCode)
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

	return fileio.WriteFile(path, []byte(sb.String()), 0644)
}

// writeHTMLReport 写入 HTML 格式报告
func writeHTMLReport(path string, steps []validate.StepResult, spans []trace.Span, gateResult *result.GateResult, inputs ...*result.InputBinding) error {
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

	if len(inputs) > 0 {
		data.Inputs = inputs[0]
	}

	// Write HTML report
	return html.WriteHTMLReport(path, data)
}

// calculateCoverageSummary 计算覆盖度总结
func calculateCoverageSummary(steps []validate.StepResult) CoverageSummary {
	return result.Measure(steps)
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

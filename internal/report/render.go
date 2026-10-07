// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package report consumes decided records. It does not match traces, compile
// contracts, evaluate expressions, or decide threshold policy.
package report

import (
	"encoding/json"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/fileio"
	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/result"
	"github.com/choreoatlas2025/cli/internal/verdict"
	"strings"
)

type Format string

const (
	JSON  Format = "json"
	JUnit Format = "junit"
	HTML  Format = "html"
)

func Write(path string, format Format, decided result.Report, spans []html.SpanInfo) error {
	switch format {
	case JSON:
		return writeJSONReport(path, decided)
	case JUnit:
		return writeJUnitReport(path, decided)
	case HTML:
		return writeHTMLReport(path, decided, spans)
	default:
		return fmt.Errorf("unsupported report format: %s", format)
	}
}

// writeJSONReport 写入 JSON 格式报告
func writeJSONReport(path string, report result.Report) error {

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON report: %w", err)
	}

	return fileio.WriteFile(path, b, 0644)
}

// writeJUnitReport 写入 JUnit XML 格式报告
func writeJUnitReport(path string, report result.Report) error {
	var sb strings.Builder
	steps, gateResult := report.Steps, report.GateResult
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
	if report.Inputs != nil {
		binding, err := json.Marshal(report.Inputs)
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

		if !verdict.StepPassed(s) {
			sb.WriteString("\n")
			fmt.Fprintf(&sb, `    <failure message="%s" type="ValidationFailure">%s</failure>`,
				xmlEscape(s.Message), xmlEscape(s.Message))
			sb.WriteString("\n  ")
		}

		// 添加条件详情到 system-out
		if len(s.Conditions) > 0 || s.Evidence != nil || s.Issue != "" {
			sb.WriteString("\n")
			conditionsJSON, err := json.Marshal(s)
			if err != nil {
				return fmt.Errorf("failed to serialize step evidence: %w", err)
			}
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
func writeHTMLReport(path string, report result.Report, spans []html.SpanInfo) error {
	data := html.FromReport(report, spans)
	// Write HTML report
	return html.WriteHTMLReport(path, data)
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

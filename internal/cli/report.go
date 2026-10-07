// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/choreoatlas2025/cli/internal/report"
	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/result"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

type ReportFormat = report.Format

const (
	ReportJSON  = report.JSON
	ReportJUnit = report.JUnit
	ReportHTML  = report.HTML
)

type CoverageSummary = result.CoverageSummary

// WriteReport retains the CLI adapter for callers supplying raw records.
func WriteReport(path string, format ReportFormat, steps []verdict.StepResult, spans []trace.Span, gate *result.GateResult, inputs ...*result.InputBinding) error {
	var binding *result.InputBinding
	if len(inputs) > 0 {
		binding = inputs[0]
	}
	return report.Write(path, format, result.New(steps, gate, binding), reportSpans(spans))
}
func reportSpans(spans []trace.Span) []html.SpanInfo {
	values := make([]html.SpanInfo, len(spans))
	for i, span := range spans {
		values[i] = html.SpanInfo{Service: span.Service, Name: span.Name, StartNanos: span.StartNanos, EndNanos: span.EndNanos}
	}
	return values
}
func calculateCoverageSummary(steps []verdict.StepResult) CoverageSummary {
	return result.Measure(steps)
}

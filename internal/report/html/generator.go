// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package html

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/fileio"
	"github.com/choreoatlas2025/cli/internal/result"

	"github.com/choreoatlas2025/cli/internal/validate"
)

//go:embed template.html
var htmlTemplate string

// HTMLData represents the data structure passed to the HTML template
type HTMLData struct {
	validate.Outcome
	ExitCode   int                   `json:"exitCode"`
	Inputs     *InputBinding         `json:"inputs,omitempty"`
	Summary    CoverageSummary       `json:"summary"`
	Steps      []validate.StepResult `json:"steps"`
	Spans      []SpanInfo            `json:"spans"`
	Graph      interface{}           `json:"graph,omitempty"` // For DAG mode
	GateResult *GateResult           `json:"gateResult,omitempty"`
	Edition    string                `json:"edition"` // Always CE; retained in the report JSON format.
}

type InputBinding = result.InputBinding

// CoverageSummary represents coverage statistics for HTML display
type CoverageSummary struct {
	StepsTotal      int     `json:"stepsTotal"`
	StepsPass       int     `json:"stepsPass"`
	StepsFail       int     `json:"stepsFail"`
	StepsSkip       int     `json:"stepsSkip"`
	StepsCoverage   float64 `json:"stepsCoverage"` // stepsPass / stepsTotal
	ConditionsTotal int     `json:"conditionsTotal"`
	ConditionsPass  int     `json:"conditionsPass"`
	ConditionsFail  int     `json:"conditionsFail"`
	ConditionsSkip  int     `json:"conditionsSkip"`
	ConditionsRate  float64 `json:"conditionsRate"` // conditionsPass / (conditionsPass + conditionsFail)
	DurationNanos   int64   `json:"durationNanos"`
}

// SpanInfo represents span information for timeline rendering
type SpanInfo struct {
	Service    string `json:"service"`
	Name       string `json:"name"`
	StartNanos int64  `json:"startNanos"`
	EndNanos   int64  `json:"endNanos"`
}

// GateResult represents baseline gate evaluation result
type GateResult = result.GateResult

// WriteHTMLReport generates and writes an HTML report file
func WriteHTMLReport(outputPath string, data HTMLData) error {
	data.Edition = "CE"
	// Serialize data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize report data: %w", err)
	}

	// Inject data into template
	content := fmt.Sprintf(`%s<script>window.FLOWREPORT = %s;</script>`,
		htmlTemplate, string(dataJSON))

	// Write to file
	return fileio.WriteFile(outputPath, []byte(content), 0644)
}

// BuildHTMLData creates HTMLData from validation results and spans
func BuildHTMLData(steps []validate.StepResult, spans []SpanInfo, gateResult *GateResult) HTMLData {
	report := result.New(steps, gateResult, nil)
	summary := summaryFromMetrics(report.Summary, spans)

	return HTMLData{
		Outcome:    report.Outcome,
		ExitCode:   report.ExitCode,
		Summary:    summary,
		Steps:      steps,
		Spans:      spans,
		GateResult: gateResult,
		Edition:    "CE",
	}
}

// calculateSummary computes coverage summary from step results
func calculateSummary(steps []validate.StepResult, spans []SpanInfo) CoverageSummary {
	return summaryFromMetrics(result.Measure(steps), spans)
}

func summaryFromMetrics(metrics result.CoverageSummary, spans []SpanInfo) CoverageSummary {
	summary := CoverageSummary{StepsTotal: metrics.StepsTotal, StepsPass: metrics.StepsPass, StepsFail: metrics.StepsFail, StepsSkip: metrics.StepsSkip,
		ConditionsTotal: metrics.ConditionsTotal, ConditionsPass: metrics.ConditionsPass, ConditionsFail: metrics.ConditionsFail, ConditionsSkip: metrics.ConditionsSkip,
		StepsCoverage: metrics.StepsRate(), ConditionsRate: metrics.ConditionsRate(false)}

	// Calculate duration from spans
	if len(spans) > 0 {
		var minStart, maxEnd int64
		for i, span := range spans {
			if i == 0 {
				minStart = span.StartNanos
				maxEnd = span.EndNanos
			} else {
				if span.StartNanos < minStart {
					minStart = span.StartNanos
				}
				if span.EndNanos > maxEnd {
					maxEnd = span.EndNanos
				}
			}
		}
		summary.DurationNanos = maxEnd - minStart
	}

	return summary
}

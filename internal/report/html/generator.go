package html

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/validate"
)

//go:embed template.html
var htmlTemplate string

// HTMLData represents the data structure passed to the HTML template
type HTMLData struct {
	Summary    CoverageSummary     `json:"summary"`
	Steps      []validate.StepResult `json:"steps"`
	Spans      []SpanInfo          `json:"spans"`
	Graph      interface{}         `json:"graph,omitempty"` // For DAG mode
	GateResult *GateResult         `json:"gateResult,omitempty"`
	Edition    string              `json:"edition"`         // Edition badge: CE/Pro/Pro Privacy
}

// CoverageSummary represents coverage statistics for HTML display
type CoverageSummary struct {
	StepsTotal      int     `json:"stepsTotal"`
	StepsPass       int     `json:"stepsPass"`
	StepsFail       int     `json:"stepsFail"`
	StepsSkip       int     `json:"stepsSkip"`
	StepsCoverage   float64 `json:"stepsCoverage"`   // stepsPass / stepsTotal
	ConditionsTotal int     `json:"conditionsTotal"`
	ConditionsPass  int     `json:"conditionsPass"`
	ConditionsFail  int     `json:"conditionsFail"`
	ConditionsSkip  int     `json:"conditionsSkip"`
	ConditionsRate  float64 `json:"conditionsRate"`  // conditionsPass / (conditionsPass + conditionsFail)
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
type GateResult struct {
	Checked bool                   `json:"checked"`
	Passed  bool                   `json:"passed"`
	Details map[string]interface{} `json:"details"`
}

// WriteHTMLReport generates and writes an HTML report file
func WriteHTMLReport(outputPath string, data HTMLData) error {
	// Serialize data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize report data: %w", err)
	}

	// Inject data into template
	content := fmt.Sprintf(`%s<script>window.FLOWREPORT = %s;</script>`, 
		htmlTemplate, string(dataJSON))

	// Write to file
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// BuildHTMLData creates HTMLData from validation results and spans
func BuildHTMLData(steps []validate.StepResult, spans []SpanInfo, gateResult *GateResult, edition string) HTMLData {
	summary := calculateSummary(steps, spans)
	
	return HTMLData{
		Summary:    summary,
		Steps:      steps,
		Spans:      spans,
		GateResult: gateResult,
		Edition:    edition,
	}
}

// calculateSummary computes coverage summary from step results
func calculateSummary(steps []validate.StepResult, spans []SpanInfo) CoverageSummary {
	summary := CoverageSummary{}
	
	// Count steps
	summary.StepsTotal = len(steps)
	for _, step := range steps {
		switch step.Status {
		case "PASS":
			summary.StepsPass++
		case "FAIL":
			summary.StepsFail++
		case "SKIP":
			summary.StepsSkip++
		}
	}
	
	// Count conditions
	for _, step := range steps {
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
	
	// Calculate rates
	if summary.StepsTotal > 0 {
		summary.StepsCoverage = float64(summary.StepsPass) / float64(summary.StepsTotal)
	}
	
	conditionsEvaluated := summary.ConditionsPass + summary.ConditionsFail
	if conditionsEvaluated > 0 {
		summary.ConditionsRate = float64(summary.ConditionsPass) / float64(conditionsEvaluated)
	}
	
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
// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package html

import (
	"os"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestBuildHTMLData(t *testing.T) {
	steps := []validate.StepResult{
		{
			Step:   "test-step",
			Call:   "testService.testOp",
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "success", Status: "PASS"},
				{Kind: "pre", Name: "precond", Status: "FAIL"},
			},
		},
		{
			Step:   "skip-step",
			Call:   "skipService.skipOp",
			Status: "SKIP",
		},
	}

	spans := []SpanInfo{
		{Service: "testService", Name: "testOp", StartNanos: 1000, EndNanos: 2000},
		{Service: "skipService", Name: "skipOp", StartNanos: 3000, EndNanos: 4000},
	}

	data := BuildHTMLData(steps, spans, nil, "CE")

	// Verify CE edition is set
	if data.Edition != "CE" {
		t.Errorf("Expected Edition 'CE', got %s", data.Edition)
	}

	// Verify summary calculations
	if data.Summary.StepsTotal != 2 {
		t.Errorf("Expected StepsTotal 2, got %d", data.Summary.StepsTotal)
	}
	if data.Summary.StepsPass != 1 {
		t.Errorf("Expected StepsPass 1, got %d", data.Summary.StepsPass)
	}
	if data.Summary.StepsSkip != 1 {
		t.Errorf("Expected StepsSkip 1, got %d", data.Summary.StepsSkip)
	}
	if data.Summary.ConditionsTotal != 2 {
		t.Errorf("Expected ConditionsTotal 2, got %d", data.Summary.ConditionsTotal)
	}
	if data.Summary.ConditionsPass != 1 {
		t.Errorf("Expected ConditionsPass 1, got %d", data.Summary.ConditionsPass)
	}
	if data.Summary.ConditionsFail != 1 {
		t.Errorf("Expected ConditionsFail 1, got %d", data.Summary.ConditionsFail)
	}

	// Verify coverage rate calculation
	expectedStepsCoverage := 0.5 // 1 pass out of 2 total
	if data.Summary.StepsCoverage != expectedStepsCoverage {
		t.Errorf("Expected StepsCoverage %f, got %f", expectedStepsCoverage, data.Summary.StepsCoverage)
	}

	expectedConditionsRate := 0.5 // 1 pass out of 2 evaluated (1 pass + 1 fail, SKIP not counted in denominator but PASS and FAIL both counted)
	if data.Summary.ConditionsRate != expectedConditionsRate {
		t.Errorf("Expected ConditionsRate %f, got %f", expectedConditionsRate, data.Summary.ConditionsRate)
	}

	// Verify duration calculation
	expectedDuration := int64(3000) // max(4000) - min(1000)
	if data.Summary.DurationNanos != expectedDuration {
		t.Errorf("Expected DurationNanos %d, got %d", expectedDuration, data.Summary.DurationNanos)
	}

	// Verify data structure
	if len(data.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(data.Steps))
	}
	if len(data.Spans) != 2 {
		t.Errorf("Expected 2 spans, got %d", len(data.Spans))
	}
}

func TestWriteHTMLReport(t *testing.T) {
	data := HTMLData{
		Summary: CoverageSummary{
			StepsTotal:      3,
			StepsPass:       2,
			StepsFail:       1,
			StepsCoverage:   0.67,
			ConditionsTotal: 5,
			ConditionsPass:  4,
			ConditionsFail:  1,
			ConditionsRate:  0.8,
			DurationNanos:   5000000,
		},
		Steps: []validate.StepResult{
			{Step: "test", Call: "test.op", Status: "PASS"},
		},
		Spans: []SpanInfo{
			{Service: "test", Name: "op", StartNanos: 1000, EndNanos: 2000},
		},
		Edition: "CE", // Set CE edition for testing
	}

	tempFile := "/tmp/test-html-report.html"
	defer os.Remove(tempFile)

	err := WriteHTMLReport(tempFile, data)
	if err != nil {
		t.Fatalf("WriteHTMLReport failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	htmlContent := string(content)

	// Verify HTML structure
	if !strings.Contains(htmlContent, "<!doctype html>") {
		t.Error("HTML should contain doctype declaration")
	}
	if !strings.Contains(htmlContent, "<title>FlowSpec Validation Report</title>") {
		t.Error("HTML should contain proper title")
	}
	if !strings.Contains(htmlContent, "window.FLOWREPORT = ") {
		t.Error("HTML should contain embedded report data")
	}

	// Verify embedded data contains key fields
	if !strings.Contains(htmlContent, `"stepsTotal":3`) {
		t.Error("HTML should contain stepsTotal in embedded data")
	}
	if !strings.Contains(htmlContent, `"stepsCoverage":0.67`) {
		t.Error("HTML should contain stepsCoverage in embedded data")
	}
	if !strings.Contains(htmlContent, `"conditionsRate":0.8`) {
		t.Error("HTML should contain conditionsRate in embedded data")
	}
	if !strings.Contains(htmlContent, `"service":"test"`) {
		t.Error("HTML should contain span service information")
	}

	// Verify CE edition badge is present in embedded data
	if !strings.Contains(htmlContent, `"edition":"CE"`) {
		t.Error("HTML should contain CE edition in embedded data")
	}

	// Verify edition badge element exists in template
	if !strings.Contains(htmlContent, `id="edition-badge"`) {
		t.Error("HTML template should contain edition-badge element")
	}
}

func TestCalculateSummary(t *testing.T) {
	steps := []validate.StepResult{
		{
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Status: "PASS"},
				{Status: "PASS"},
			},
		},
		{
			Status: "FAIL",
			Conditions: []validate.ConditionResult{
				{Status: "FAIL"},
				{Status: "SKIP"}, // Should not count in denominator
			},
		},
	}

	spans := []SpanInfo{
		{StartNanos: 1000, EndNanos: 3000},
		{StartNanos: 2000, EndNanos: 5000},
	}

	summary := calculateSummary(steps, spans)

	// Steps statistics
	if summary.StepsTotal != 2 {
		t.Errorf("Expected StepsTotal 2, got %d", summary.StepsTotal)
	}
	if summary.StepsPass != 1 {
		t.Errorf("Expected StepsPass 1, got %d", summary.StepsPass)
	}
	if summary.StepsFail != 1 {
		t.Errorf("Expected StepsFail 1, got %d", summary.StepsFail)
	}

	// Conditions statistics  
	if summary.ConditionsTotal != 4 {
		t.Errorf("Expected ConditionsTotal 4, got %d", summary.ConditionsTotal)
	}
	if summary.ConditionsPass != 2 {
		t.Errorf("Expected ConditionsPass 2, got %d", summary.ConditionsPass)
	}
	if summary.ConditionsFail != 1 {
		t.Errorf("Expected ConditionsFail 1, got %d", summary.ConditionsFail)
	}
	if summary.ConditionsSkip != 1 {
		t.Errorf("Expected ConditionsSkip 1, got %d", summary.ConditionsSkip)
	}

	// Rate calculations
	expectedStepsCoverage := 0.5 // 1 pass / 2 total
	if summary.StepsCoverage != expectedStepsCoverage {
		t.Errorf("Expected StepsCoverage %f, got %f", expectedStepsCoverage, summary.StepsCoverage)
	}

	expectedConditionsRate := 2.0/3.0 // 2 pass / (2 pass + 1 fail), skip not counted
	if summary.ConditionsRate != expectedConditionsRate {
		t.Errorf("Expected ConditionsRate %f, got %f", expectedConditionsRate, summary.ConditionsRate)
	}

	// Duration calculation
	expectedDuration := int64(4000) // max(5000) - min(1000)  
	if summary.DurationNanos != expectedDuration {
		t.Errorf("Expected DurationNanos %d, got %d", expectedDuration, summary.DurationNanos)
	}
}

func TestGateResultEmbedding(t *testing.T) {
	gateResult := &GateResult{
		Checked: true,
		Passed:  false,
		Details: map[string]interface{}{
			"stepsThreshold":      0.9,
			"conditionsThreshold": 0.95,
			"stepsCoverage":       0.8,
			"conditionsRate":      0.9,
		},
	}

	data := HTMLData{
		Summary:    CoverageSummary{},
		Steps:      []validate.StepResult{},
		Spans:      []SpanInfo{},
		GateResult: gateResult,
	}

	tempFile := "/tmp/test-gate-report.html"
	defer os.Remove(tempFile)

	err := WriteHTMLReport(tempFile, data)
	if err != nil {
		t.Fatalf("WriteHTMLReport failed: %v", err)
	}

	// Read and verify gate result is embedded
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	htmlContent := string(content)
	if !strings.Contains(htmlContent, `"checked":true`) {
		t.Error("HTML should contain gate checked status")
	}
	if !strings.Contains(htmlContent, `"passed":false`) {
		t.Error("HTML should contain gate passed status")
	}
	if !strings.Contains(htmlContent, `"stepsThreshold":0.9`) {
		t.Error("HTML should contain gate threshold details")
	}
}

func TestEmptyData(t *testing.T) {
	// Test with empty data
	data := BuildHTMLData([]validate.StepResult{}, []SpanInfo{}, nil, "CE")

	if data.Summary.StepsTotal != 0 {
		t.Errorf("Expected StepsTotal 0 for empty data, got %d", data.Summary.StepsTotal)
	}
	if data.Summary.StepsCoverage != 0 {
		t.Errorf("Expected StepsCoverage 0 for empty data, got %f", data.Summary.StepsCoverage)
	}
	if data.Summary.ConditionsRate != 0 {
		t.Errorf("Expected ConditionsRate 0 for empty data, got %f", data.Summary.ConditionsRate)
	}
	if data.Summary.DurationNanos != 0 {
		t.Errorf("Expected DurationNanos 0 for empty data, got %d", data.Summary.DurationNanos)
	}
}
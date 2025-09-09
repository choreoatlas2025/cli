package baseline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestRecordBaseline(t *testing.T) {
	// Create a test flow spec
	flowSpec := &spec.FlowSpec{
		Info: spec.FlowInfo{
			Title: "Test Flow",
		},
		Flow: []spec.FlowStep{
			{Step: "step1"},
			{Step: "step2"},
			{Step: "step3"},
		},
	}

	// Create test validation results
	results := []validate.StepResult{
		{
			Step:   "step1",
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "cond1", Status: "PASS"},
				{Kind: "post", Name: "cond2", Status: "PASS"},
			},
		},
		{
			Step:   "step2", 
			Status: "PASS",
			Conditions: []validate.ConditionResult{
				{Kind: "post", Name: "cond3", Status: "FAIL"},
			},
		},
		{
			Step:   "step3",
			Status: "FAIL",
		},
	}

	// Create temporary flow file
	tempDir := t.TempDir()
	flowPath := filepath.Join(tempDir, "test.flowspec.yaml")
	err := os.WriteFile(flowPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp flow file: %v", err)
	}

	// Test recording baseline
	baseline, err := RecordBaseline(flowSpec, results, flowPath)
	if err != nil {
		t.Fatalf("RecordBaseline failed: %v", err)
	}

	// Verify baseline data
	if baseline.FlowID != "Test Flow" {
		t.Errorf("Expected FlowID 'Test Flow', got %s", baseline.FlowID)
	}

	if baseline.StepsTotal != 3 {
		t.Errorf("Expected StepsTotal 3, got %d", baseline.StepsTotal)
	}

	if len(baseline.CoveredSteps) != 2 {
		t.Errorf("Expected 2 covered steps, got %d", len(baseline.CoveredSteps))
	}

	// Verify covered steps contain the correct ones
	expectedSteps := map[string]bool{"step1": true, "step2": true}
	for _, step := range baseline.CoveredSteps {
		if !expectedSteps[step] {
			t.Errorf("Unexpected covered step: %s", step)
		}
	}

	// Verify conditions are recorded correctly
	if len(baseline.Conditions) != 2 {
		t.Errorf("Expected 2 steps with conditions, got %d", len(baseline.Conditions))
	}

	step1Conditions := baseline.Conditions["step1"]
	if len(step1Conditions) != 2 {
		t.Errorf("Expected 2 conditions for step1, got %d", len(step1Conditions))
	}

	if !step1Conditions["post:cond1"] || !step1Conditions["post:cond2"] {
		t.Errorf("step1 conditions not recorded correctly")
	}

	step2Conditions := baseline.Conditions["step2"]
	if step2Conditions["post:cond3"] != false {
		t.Errorf("Expected step2 cond3 to be false, got %v", step2Conditions["post:cond3"])
	}
}

func TestEvaluateGate(t *testing.T) {
	tests := []struct {
		name        string
		results     []validate.StepResult
		thresholds  ThresholdConfig
		expectPass  bool
		expectViolations int
	}{
		{
			name: "All thresholds passed",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "PASS"}, {Status: "PASS"},
				}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "PASS"},
				}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold: 0.9,
				ConditionsThreshold: 0.9,
			},
			expectPass: true,
			expectViolations: 0,
		},
		{
			name: "Steps threshold failed",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "PASS"},
				}},
				{Step: "step2", Status: "FAIL"},
			},
			thresholds: ThresholdConfig{
				StepsThreshold: 0.9,
				ConditionsThreshold: 0.5, // Lower threshold so conditions pass
			},
			expectPass: false,
			expectViolations: 1,
		},
		{
			name: "Conditions threshold failed",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "PASS"}, {Status: "FAIL"},
				}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "FAIL"},
				}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold: 0.9,
				ConditionsThreshold: 0.9,
			},
			expectPass: false,
			expectViolations: 1,
		},
		{
			name: "Skip as fail enabled",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{
					{Status: "PASS"}, {Status: "SKIP"},
				}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold: 0.5,
				ConditionsThreshold: 0.9,
				SkipAsFail: true,
			},
			expectPass: false,
			expectViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateGate(tt.results, tt.thresholds, nil)

			if result.Passed != tt.expectPass {
				t.Errorf("Expected pass=%v, got %v", tt.expectPass, result.Passed)
			}

			if len(result.Violations) != tt.expectViolations {
				t.Errorf("Expected %d violations, got %d: %v", 
					tt.expectViolations, len(result.Violations), result.Violations)
			}

			if !result.Checked {
				t.Error("Expected gate to be checked")
			}

			// Verify details are populated
			if result.Details == nil {
				t.Error("Expected details to be populated")
			}
		})
	}
}

func TestSaveAndLoadBaseline(t *testing.T) {
	baseline := &BaselineData{
		SchemaVersion: "1",
		FlowID:        "Test Flow",
		FlowHash:      "sha256:abc123",
		GeneratedAt:   time.Now().UTC(),
		StepsTotal:    3,
		CoveredSteps:  []string{"step1", "step2"},
		Conditions: map[string]map[string]bool{
			"step1": {"post:cond1": true, "post:cond2": false},
		},
	}

	tempDir := t.TempDir()
	baselinePath := filepath.Join(tempDir, "test-baseline.json")

	// Test saving
	err := SaveBaseline(baseline, baselinePath)
	if err != nil {
		t.Fatalf("SaveBaseline failed: %v", err)
	}

	// Test loading
	loadedBaseline, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline failed: %v", err)
	}

	// Verify loaded data
	if loadedBaseline.FlowID != baseline.FlowID {
		t.Errorf("FlowID mismatch: expected %s, got %s", baseline.FlowID, loadedBaseline.FlowID)
	}

	if loadedBaseline.StepsTotal != baseline.StepsTotal {
		t.Errorf("StepsTotal mismatch: expected %d, got %d", baseline.StepsTotal, loadedBaseline.StepsTotal)
	}

	if len(loadedBaseline.CoveredSteps) != len(baseline.CoveredSteps) {
		t.Errorf("CoveredSteps length mismatch: expected %d, got %d", 
			len(baseline.CoveredSteps), len(loadedBaseline.CoveredSteps))
	}

	// Verify conditions
	if len(loadedBaseline.Conditions) != len(baseline.Conditions) {
		t.Errorf("Conditions length mismatch")
	}

	step1Conds := loadedBaseline.Conditions["step1"]
	if step1Conds["post:cond1"] != true || step1Conds["post:cond2"] != false {
		t.Errorf("Conditions not loaded correctly")
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()
	
	if thresholds.StepsThreshold != 0.9 {
		t.Errorf("Expected default StepsThreshold 0.9, got %f", thresholds.StepsThreshold)
	}
	
	if thresholds.ConditionsThreshold != 0.95 {
		t.Errorf("Expected default ConditionsThreshold 0.95, got %f", thresholds.ConditionsThreshold)
	}
	
	if thresholds.SkipAsFail != false {
		t.Errorf("Expected default SkipAsFail false, got %v", thresholds.SkipAsFail)
	}
}
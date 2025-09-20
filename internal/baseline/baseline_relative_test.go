package baseline

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestEvaluateGate_RelativeMode(t *testing.T) {
	tests := []struct {
		name               string
		results            []validate.StepResult
		thresholds         ThresholdConfig
		baseline           *BaselineData
		expectPass         bool
		expectStepsDelta   float64
		expectCondsDelta   float64
	}{
		{
			name: "relative mode - no degradation",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step3", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold:      0.05, // Allow 5% degradation
				ConditionsThreshold: 0.03, // Allow 3% degradation
			},
			baseline: &BaselineData{
				StepsTotal:   3,
				CoveredSteps: []string{"step1", "step2", "step3"},
				Conditions: map[string]map[string]bool{
					"step1": {"cond1": true},
					"step2": {"cond1": true},
					"step3": {"cond1": true},
				},
			},
			expectPass:       true,
			expectStepsDelta: 0.0,
			expectCondsDelta: 0.0,
		},
		{
			name: "relative mode - minor degradation within tolerance",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step3", Status: "FAIL", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold:      0.35, // Allow 35% degradation (baseline: 100%, current: 66.7%, delta: -33.3%)
				ConditionsThreshold: 0.05,
			},
			baseline: &BaselineData{
				StepsTotal:   3,
				CoveredSteps: []string{"step1", "step2", "step3"},
				Conditions: map[string]map[string]bool{
					"step1": {"cond1": true},
					"step2": {"cond1": true},
					"step3": {"cond1": true},
				},
			},
			expectPass:       true,
			expectStepsDelta: -0.333,
			expectCondsDelta: 0.0,
		},
		{
			name: "relative mode - degradation exceeds tolerance",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step2", Status: "FAIL", Conditions: []validate.ConditionResult{{Status: "FAIL"}}},
				{Step: "step3", Status: "FAIL", Conditions: []validate.ConditionResult{{Status: "FAIL"}}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold:      0.30, // Allow 30% degradation (baseline: 100%, current: 33.3%, delta: -66.7%)
				ConditionsThreshold: 0.05,
			},
			baseline: &BaselineData{
				StepsTotal:   3,
				CoveredSteps: []string{"step1", "step2", "step3"},
				Conditions: map[string]map[string]bool{
					"step1": {"cond1": true},
					"step2": {"cond1": true},
					"step3": {"cond1": true},
				},
			},
			expectPass:       false,
			expectStepsDelta: -0.667,
			expectCondsDelta: -0.667,
		},
		{
			name: "relative mode - improvement from baseline",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step3", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold:      0.10,
				ConditionsThreshold: 0.10,
			},
			baseline: &BaselineData{
				StepsTotal:   3,
				CoveredSteps: []string{"step1", "step2"}, // Only 2 were covered in baseline
				Conditions: map[string]map[string]bool{
					"step1": {"cond1": true},
					"step2": {"cond1": false}, // One condition failed in baseline
					"step3": {"cond1": false}, // Another condition failed in baseline
				},
			},
			expectPass:       true,
			expectStepsDelta: 0.5, // 66.7% -> 100%, improvement of 50%
			expectCondsDelta: 2.0, // 33.3% -> 100%, improvement of 200%
		},
		{
			name: "relative mode - baseline missing fallback to absolute",
			results: []validate.StepResult{
				{Step: "step1", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step2", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "PASS"}}},
				{Step: "step3", Status: "FAIL", Conditions: []validate.ConditionResult{{Status: "FAIL"}}},
			},
			thresholds: ThresholdConfig{
				StepsThreshold:      0.60, // Interpreted as absolute 60% when no baseline
				ConditionsThreshold: 0.60,
			},
			baseline:         nil, // No baseline provided
			expectPass:       true,
			expectStepsDelta: 0.0,
			expectCondsDelta: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := EvaluateGate(tc.results, tc.thresholds, tc.baseline)

			if result.Passed != tc.expectPass {
				t.Errorf("Expected pass=%v, got %v", tc.expectPass, result.Passed)
			}

			if tc.baseline != nil {
				// Check delta values when baseline is present
				if stepsDelta, ok := result.Details["stepsDeltaPct"].(float64); ok {
					if !floatEquals(stepsDelta, tc.expectStepsDelta, 0.01) {
						t.Errorf("Expected steps delta %.3f, got %.3f", tc.expectStepsDelta, stepsDelta)
					}
				}

				if condsDelta, ok := result.Details["conditionsDeltaPct"].(float64); ok {
					if !floatEquals(condsDelta, tc.expectCondsDelta, 0.01) {
						t.Errorf("Expected conditions delta %.3f, got %.3f", tc.expectCondsDelta, condsDelta)
					}
				}

				// Verify baseline values are present in details
				if _, ok := result.Details["baselineStepsCoverage"]; !ok && len(tc.baseline.CoveredSteps) > 0 {
					t.Error("Missing baselineStepsCoverage in details")
				}
			}
		})
	}
}

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
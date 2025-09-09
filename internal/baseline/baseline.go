package baseline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

// BaselineData represents a recorded baseline for comparison
type BaselineData struct {
	SchemaVersion string                      `json:"schemaVersion"`
	FlowID        string                      `json:"flowId"`
	FlowHash      string                      `json:"flowHash"`
	GeneratedAt   time.Time                   `json:"generatedAt"`
	StepsTotal    int                         `json:"stepsTotal"`
	CoveredSteps  []string                    `json:"coveredSteps"`
	Conditions    map[string]map[string]bool  `json:"conditions"`
}

// ThresholdConfig represents baseline gate thresholds
type ThresholdConfig struct {
	StepsThreshold      float64 `json:"stepsThreshold"`      // Default 0.9
	ConditionsThreshold float64 `json:"conditionsThreshold"` // Default 0.95
	SkipAsFail          bool    `json:"skipAsFail"`          // Default false
}

// GateResult represents the result of baseline gate evaluation
type GateResult struct {
	Checked    bool                   `json:"checked"`
	Passed     bool                   `json:"passed"`
	Details    map[string]interface{} `json:"details"`
	Violations []string               `json:"violations,omitempty"`
}

// DefaultThresholds returns the default baseline thresholds
func DefaultThresholds() ThresholdConfig {
	return ThresholdConfig{
		StepsThreshold:      0.9,  // 90% step coverage
		ConditionsThreshold: 0.95, // 95% condition pass rate
		SkipAsFail:          false,
	}
}

// RecordBaseline creates a baseline from validation results
func RecordBaseline(flowSpec *spec.FlowSpec, results []validate.StepResult, flowPath string) (*BaselineData, error) {
	// Calculate flow hash for version tracking
	flowContent, err := os.ReadFile(flowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read flow file for hashing: %w", err)
	}
	
	hash := sha256.Sum256(flowContent)
	flowHash := fmt.Sprintf("sha256:%x", hash)

	// Extract covered steps (PASS status)
	var coveredSteps []string
	for _, result := range results {
		if result.Status == "PASS" {
			coveredSteps = append(coveredSteps, result.Step)
		}
	}

	// Extract condition results
	conditions := make(map[string]map[string]bool)
	for _, result := range results {
		if len(result.Conditions) > 0 {
			stepConditions := make(map[string]bool)
			for _, cond := range result.Conditions {
				condKey := fmt.Sprintf("%s:%s", cond.Kind, cond.Name)
				stepConditions[condKey] = cond.Status == "PASS"
			}
			conditions[result.Step] = stepConditions
		}
	}

	baseline := &BaselineData{
		SchemaVersion: "1",
		FlowID:        flowSpec.Info.Title,
		FlowHash:      flowHash,
		GeneratedAt:   time.Now().UTC(),
		StepsTotal:    flowSpec.GetStepsCount(),
		CoveredSteps:  coveredSteps,
		Conditions:    conditions,
	}

	return baseline, nil
}

// SaveBaseline writes baseline data to a JSON file
func SaveBaseline(baseline *BaselineData, outputPath string) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// LoadBaseline reads baseline data from a JSON file
func LoadBaseline(path string) (*BaselineData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline BaselineData
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline: %w", err)
	}

	return &baseline, nil
}

// EvaluateGate checks validation results against baseline thresholds
func EvaluateGate(results []validate.StepResult, thresholds ThresholdConfig, baseline *BaselineData) *GateResult {
	// Calculate current metrics
	stepsTotal := len(results)
	stepsPass := 0
	conditionsTotal := 0
	conditionsPass := 0
	conditionsFail := 0

	for _, result := range results {
		if result.Status == "PASS" {
			stepsPass++
		}

		for _, condition := range result.Conditions {
			conditionsTotal++
			switch condition.Status {
			case "PASS":
				conditionsPass++
			case "FAIL":
				conditionsFail++
			case "SKIP":
				if thresholds.SkipAsFail {
					conditionsFail++
				}
				// Otherwise skip doesn't count in pass/fail
			}
		}
	}

	// Calculate rates
	var stepsCoverage float64
	if stepsTotal > 0 {
		stepsCoverage = float64(stepsPass) / float64(stepsTotal)
	}

	var conditionsRate float64
	conditionsEvaluated := conditionsPass + conditionsFail
	if conditionsEvaluated > 0 {
		conditionsRate = float64(conditionsPass) / float64(conditionsEvaluated)
	}

	// Check thresholds
	stepsPassed := stepsCoverage >= thresholds.StepsThreshold
	conditionsPassed := conditionsRate >= thresholds.ConditionsThreshold
	overallPassed := stepsPassed && conditionsPassed

	// Collect violations
	var violations []string
	if !stepsPassed {
		violations = append(violations, fmt.Sprintf("Steps coverage %.1f%% < required %.1f%%", 
			stepsCoverage*100, thresholds.StepsThreshold*100))
	}
	if !conditionsPassed {
		violations = append(violations, fmt.Sprintf("Conditions pass rate %.1f%% < required %.1f%%", 
			conditionsRate*100, thresholds.ConditionsThreshold*100))
	}

	result := &GateResult{
		Checked: true,
		Passed:  overallPassed,
		Details: map[string]interface{}{
			"stepsTotal":           stepsTotal,
			"stepsPass":            stepsPass,
			"stepsCoverage":        stepsCoverage,
			"stepsThreshold":       thresholds.StepsThreshold,
			"conditionsTotal":      conditionsTotal,
			"conditionsPass":       conditionsPass,
			"conditionsFail":       conditionsFail,
			"conditionsEvaluated":  conditionsEvaluated,
			"conditionsRate":       conditionsRate,
			"conditionsThreshold":  thresholds.ConditionsThreshold,
			"skipAsFail":          thresholds.SkipAsFail,
		},
		Violations: violations,
	}

	return result
}
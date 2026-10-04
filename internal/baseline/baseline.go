// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"strings"
	"time"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

// BaselineData represents a recorded baseline for comparison
type BaselineData struct {
	SchemaVersion string                     `json:"schemaVersion"`
	FlowID        string                     `json:"flowId"`
	FlowHash      string                     `json:"flowHash"`
	ServiceHashes map[string]string          `json:"serviceHashes"`
	GeneratedAt   time.Time                  `json:"generatedAt"`
	StepsTotal    int                        `json:"stepsTotal"`
	CoveredSteps  []string                   `json:"coveredSteps"`
	Conditions    map[string]map[string]bool `json:"conditions"`
}

// ThresholdConfig represents baseline gate thresholds
type ThresholdConfig struct {
	StepsThreshold           float64 `json:"stepsThreshold"`      // Default 0.9
	ConditionsThreshold      float64 `json:"conditionsThreshold"` // Default 0.95
	MaxStepsDegradation      float64 `json:"maxStepsDegradation"`
	MaxConditionsDegradation float64 `json:"maxConditionsDegradation"`
	SkipAsFail               bool    `json:"skipAsFail"` // Default false
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

var ErrIncompleteValidation = errors.New("baseline requires complete successful validation")

func ValidateThresholds(t ThresholdConfig) error {
	for name, value := range map[string]float64{"threshold-steps": t.StepsThreshold, "threshold-conds": t.ConditionsThreshold, "max-steps-degradation": t.MaxStepsDegradation, "max-conds-degradation": t.MaxConditionsDegradation} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return fmt.Errorf("invalid %s: must be finite and between 0 and 1", name)
		}
	}
	return nil
}

// RecordBaseline creates a baseline from validation results
func RecordBaseline(flowSpec *spec.FlowSpec, results []validate.StepResult, flowPath string) (*BaselineData, error) {
	if !validate.AllStepsPassed(results) || len(results) != flowSpec.GetStepsCount() {
		return nil, ErrIncompleteValidation
	}
	expected := map[string]bool{}
	for _, name := range flowSpec.GetStepNames() {
		expected[name] = true
	}
	for _, result := range results {
		if !expected[result.Step] {
			return nil, ErrIncompleteValidation
		}
		delete(expected, result.Step)
		for _, cond := range result.Conditions {
			if cond.Status != "PASS" {
				return nil, ErrIncompleteValidation
			}
		}
	}
	if len(expected) != 0 {
		return nil, ErrIncompleteValidation
	}
	identity, err := spec.IdentifyContract(flowSpec, flowPath)
	if err != nil {
		return nil, err
	}

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
		SchemaVersion: "2",
		FlowID:        identity.FlowID,
		FlowHash:      identity.FlowHash,
		ServiceHashes: identity.ServiceHashes,
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

	if err := validateBaselineData(&baseline); err != nil {
		return nil, err
	}
	return &baseline, nil
}

func validateBaselineData(b *BaselineData) error {
	if b.SchemaVersion != "2" {
		return fmt.Errorf("invalid baseline schema version %q: record a new baseline (version 2)", b.SchemaVersion)
	}
	if b.FlowID == "" || !validHash(b.FlowHash) || b.StepsTotal <= 0 || len(b.CoveredSteps) != b.StepsTotal {
		return fmt.Errorf("invalid baseline: incomplete identity or coverage")
	}
	if b.ServiceHashes == nil {
		return fmt.Errorf("invalid baseline: missing ServiceSpec identities")
	}
	for _, hash := range b.ServiceHashes {
		if !validHash(hash) {
			return fmt.Errorf("invalid baseline: invalid ServiceSpec hash")
		}
	}
	covered := map[string]bool{}
	for _, step := range b.CoveredSteps {
		if step == "" || covered[step] {
			return fmt.Errorf("invalid baseline: empty or duplicate step")
		}
		covered[step] = true
	}
	for step, conditions := range b.Conditions {
		if !covered[step] {
			return fmt.Errorf("invalid baseline: conditions reference unknown step %s", step)
		}
		for name, passed := range conditions {
			if name == "" || !passed {
				return fmt.Errorf("invalid baseline: unsuccessful condition %s", name)
			}
		}
	}
	return nil
}

func validHash(hash string) bool {
	if !strings.HasPrefix(hash, "sha256:") || len(hash) != 71 {
		return false
	}
	for _, c := range hash[7:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func ValidateCompatibility(b *BaselineData, flow *spec.FlowSpec, flowPath string) error {
	if err := validateBaselineData(b); err != nil {
		return err
	}
	identity, err := spec.IdentifyContract(flow, flowPath)
	if err != nil {
		return err
	}
	if b.FlowID != identity.FlowID || b.FlowHash != identity.FlowHash || !maps.Equal(b.ServiceHashes, identity.ServiceHashes) {
		return fmt.Errorf("invalid baseline: contract identity changed; record a new baseline")
	}
	covered := map[string]bool{}
	for _, step := range b.CoveredSteps {
		covered[step] = true
	}
	if len(covered) != flow.GetStepsCount() {
		return fmt.Errorf("invalid baseline: step count does not match contract")
	}
	_, ops, err := flow.BuildOperationIndex(flowPath)
	if err != nil {
		return err
	}
	for _, step := range flow.CallSteps() {
		if !covered[step.Step] {
			return fmt.Errorf("invalid baseline: contract step %s is absent", step.Step)
		}
		parts := strings.SplitN(step.Call, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid contract call: %s", step.Call)
		}
		op := ops[parts[0]][parts[1]]
		conditions := b.Conditions[step.Step]
		if len(conditions) != len(op.Preconditions)+len(op.Postconditions) {
			return fmt.Errorf("invalid baseline: condition set changed for %s", step.Step)
		}
		for name := range op.Preconditions {
			if !conditions["pre:"+name] {
				return fmt.Errorf("invalid baseline: missing precondition %s", name)
			}
		}
		for name := range op.Postconditions {
			if !conditions["post:"+name] {
				return fmt.Errorf("invalid baseline: missing postcondition %s", name)
			}
		}
	}
	return nil
}

// EvaluateGate always applies absolute floors. Baselines add independent limits
// on relative degradation; they never reinterpret or replace the floors.
// EvaluateGate checks validation results against baseline thresholds
func EvaluateGate(results []validate.StepResult, thresholds ThresholdConfig, baseline *BaselineData) *GateResult {
	if err := ValidateThresholds(thresholds); err != nil {
		return &GateResult{Checked: true, Passed: false, Violations: []string{err.Error()}}
	}
	if baseline != nil && baseline.StepsTotal <= 0 {
		return &GateResult{Checked: true, Passed: false, Violations: []string{"invalid baseline: step count must be positive"}}
	}
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

	// Initialize gate checking variables
	stepsPassed := stepsCoverage >= thresholds.StepsThreshold
	conditionsPassed := conditionsRate >= thresholds.ConditionsThreshold
	var violations []string
	details := map[string]interface{}{
		"stepsTotal":          stepsTotal,
		"stepsPass":           stepsPass,
		"stepsCoverage":       stepsCoverage,
		"stepsThreshold":      thresholds.StepsThreshold,
		"conditionsTotal":     conditionsTotal,
		"conditionsPass":      conditionsPass,
		"conditionsFail":      conditionsFail,
		"conditionsEvaluated": conditionsEvaluated,
		"conditionsRate":      conditionsRate,
		"conditionsThreshold": thresholds.ConditionsThreshold,
		"skipAsFail":          thresholds.SkipAsFail,
	}

	if !stepsPassed {
		violations = append(violations, fmt.Sprintf("steps coverage %.1f%% < required %.1f%%", stepsCoverage*100, thresholds.StepsThreshold*100))
	}
	if !conditionsPassed {
		violations = append(violations, fmt.Sprintf("conditions pass rate %.1f%% < required %.1f%%", conditionsRate*100, thresholds.ConditionsThreshold*100))
	}
	if baseline != nil {
		// Relative mode: compare against baseline
		baselineStepsCoverage := float64(len(baseline.CoveredSteps)) / float64(baseline.StepsTotal)
		details["baselineStepsCoverage"] = baselineStepsCoverage

		// Calculate deltas
		stepsDeltaAbs := stepsCoverage - baselineStepsCoverage
		var stepsDeltaPct float64
		if baselineStepsCoverage > 0 {
			stepsDeltaPct = (stepsCoverage - baselineStepsCoverage) / baselineStepsCoverage
		}
		details["stepsDeltaAbs"] = stepsDeltaAbs
		details["stepsDeltaPct"] = stepsDeltaPct

		// For conditions, calculate baseline rate
		baselineConditionsPass := 0
		baselineConditionsTotal := 0
		for _, stepConds := range baseline.Conditions {
			for _, passed := range stepConds {
				baselineConditionsTotal++
				if passed {
					baselineConditionsPass++
				}
			}
		}
		var baselineConditionsRate float64
		if baselineConditionsTotal > 0 {
			baselineConditionsRate = float64(baselineConditionsPass) / float64(baselineConditionsTotal)
		}
		details["baselineConditionsRate"] = baselineConditionsRate

		conditionsDeltaAbs := conditionsRate - baselineConditionsRate
		var conditionsDeltaPct float64
		if baselineConditionsRate > 0 {
			conditionsDeltaPct = (conditionsRate - baselineConditionsRate) / baselineConditionsRate
		}
		details["conditionsDeltaAbs"] = conditionsDeltaAbs
		details["conditionsDeltaPct"] = conditionsDeltaPct

		// Check relative thresholds (delta percentage)
		relativeStepsPassed := stepsDeltaPct >= -thresholds.MaxStepsDegradation
		relativeConditionsPassed := conditionsDeltaPct >= -thresholds.MaxConditionsDegradation

		if !relativeStepsPassed {
			violations = append(violations, fmt.Sprintf("Steps coverage delta %.1f%% < allowed %.1f%%",
				stepsDeltaPct*100, -thresholds.MaxStepsDegradation*100))
		}
		if !relativeConditionsPassed {
			violations = append(violations, fmt.Sprintf("Conditions rate delta %.1f%% < allowed %.1f%%",
				conditionsDeltaPct*100, -thresholds.MaxConditionsDegradation*100))
		}
		stepsPassed = stepsPassed && relativeStepsPassed
		conditionsPassed = conditionsPassed && relativeConditionsPassed
		details["maxStepsDegradation"] = thresholds.MaxStepsDegradation
		details["maxConditionsDegradation"] = thresholds.MaxConditionsDegradation
	}

	overallPassed := stepsPassed && conditionsPassed

	result := &GateResult{
		Checked:    true,
		Passed:     overallPassed,
		Details:    details,
		Violations: violations,
	}

	return result
}

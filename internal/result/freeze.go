// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"maps"
	"slices"

	"github.com/choreoatlas2025/cli/internal/verdict"
)

// Own the records used for the decision so subsequent producer mutations cannot
// make renderers consume a different revision of the evidence.
func freeze(steps []verdict.StepResult, gate *GateResult, inputs *InputBinding) ([]verdict.StepResult, *GateResult, *InputBinding) {
	steps = slices.Clone(steps)
	for i := range steps {
		step := &steps[i]
		step.Conditions = slices.Clone(step.Conditions)
		step.Violations = slices.Clone(step.Violations)
		step.Bindings = slices.Clone(step.Bindings)
		if step.Evidence != nil {
			ref := *step.Evidence
			step.Evidence = &ref
		}
		for j := range step.Conditions {
			if step.Conditions[j].Status == "PASS" && !verdict.ConditionPassed(step.Conditions[j]) {
				step.Conditions[j].Status = "FAIL"
			}
		}
		if step.Status == "PASS" && !verdict.StepPassed(*step) {
			step.Status = "FAIL"
		}
	}
	if gate != nil {
		copied := *gate
		copied.Details = copyFields(gate.Details)
		copied.Violations = slices.Clone(gate.Violations)
		gate = &copied
	}
	if inputs != nil {
		copied := *inputs
		copied.Contract.ServiceHashes = maps.Clone(inputs.Contract.ServiceHashes)
		copied.Policy = copyFields(inputs.Policy)
		if inputs.BaselineProvenance != nil {
			provenance := *inputs.BaselineProvenance
			copied.BaselineProvenance = &provenance
		}
		inputs = &copied
	}
	return steps, gate, inputs
}

func copyFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	copy := make(map[string]any, len(fields))
	for key, value := range fields {
		copy[key] = copyValue(value)
	}
	return copy
}

func copyValue(value any) any {
	switch current := value.(type) {
	case map[string]any:
		return copyFields(current)
	case []any:
		copied := make([]any, len(current))
		for i, child := range current {
			copied[i] = copyValue(child)
		}
		return copied
	default:
		return value
	}
}

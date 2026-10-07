// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

func TestDecisionOwnsProducerRecords(t *testing.T) {
	steps := []verdict.StepResult{{Step: "a", Status: "PASS", Evidence: &verdict.SpanRef{Key: "original"}, Conditions: []verdict.ConditionResult{{Status: "PASS"}}}}
	gate := &GateResult{Checked: true, Passed: true, Details: map[string]any{"nested": map[string]any{"threshold": 0.0}}}
	inputs := &InputBinding{Contract: spec.ContractIdentity{ServiceHashes: map[string]string{"svc": "captured"}}, Policy: gate.Details}
	decided := New(steps, gate, inputs)
	steps[0].Status = "FAIL"
	steps[0].Conditions[0].Status = "FAIL"
	steps[0].Evidence.Key = "different"
	gate.Passed = false
	gate.Details["nested"].(map[string]any)["threshold"] = 1.0
	inputs.Contract.ServiceHashes["svc"] = "different"
	if !decided.Success || !decided.GateResult.Passed || decided.Steps[0].Evidence.Key != "original" || !verdict.StepPassed(decided.Steps[0]) || decided.Inputs.Contract.ServiceHashes["svc"] != "captured" || decided.Inputs.Policy["nested"].(map[string]any)["threshold"] != 0.0 {
		t.Fatalf("producer mutation changed decided revision: %+v", decided)
	}
}

func TestMissingEvidenceCannotCountAsPassedCondition(t *testing.T) {
	steps := []verdict.StepResult{{Status: "PASS", Conditions: []verdict.ConditionResult{{Status: "PASS", Issue: verdict.MissingEvidence}}}}
	decided := New(steps, &GateResult{Checked: true, Passed: true}, nil)
	if decided.Success || decided.ExitCode != 3 || decided.Summary.ConditionsPass != 0 || decided.Summary.ConditionsFail != 1 || decided.Steps[0].Status != "FAIL" || decided.Steps[0].Conditions[0].Status != "FAIL" {
		t.Fatalf("incomplete evidence became passed metrics/display: %+v", decided)
	}
}

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

// Outcome is the shared final decision for command output and report consumers.
// Threshold policy cannot turn a failed runtime validation into success.
type Outcome struct {
	Success          bool   `json:"success"`
	ValidationPassed bool   `json:"validationPassed"`
	GatePassed       bool   `json:"gatePassed"`
	Status           string `json:"status"`
}

func AllStepsPassed(steps []StepResult) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if step.Status != "PASS" {
			return false
		}
		for _, cond := range step.Conditions {
			if cond.Status == "FAIL" {
				return false
			}
		}
	}
	return true
}

func FinalOutcome(steps []StepResult, gateChecked, gatePassed bool) Outcome {
	o := Outcome{ValidationPassed: AllStepsPassed(steps), GatePassed: !gateChecked || gatePassed, Status: "PASS"}
	switch {
	case !o.ValidationPassed:
		o.Status = "VALIDATION_FAILED"
	case !o.GatePassed:
		o.Status = "GATE_FAILED"
	default:
		o.Success = true
	}
	return o
}

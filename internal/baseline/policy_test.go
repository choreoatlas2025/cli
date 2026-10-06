// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package baseline

import (
	"fmt"
	"math"
	"testing"

	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestBaselinePolicyNeverReplacesAbsoluteFloors(t *testing.T) {
	b := &BaselineData{StepsTotal: 20}
	var current []validate.StepResult
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("step%d", i)
		b.CoveredSteps = append(b.CoveredSteps, name)
		if b.Conditions == nil {
			b.Conditions = map[string]map[string]bool{}
		}
		b.Conditions[name] = map[string]bool{"post:ok": true}
		status := "FAIL"
		if i < 2 {
			status = "PASS"
		}
		current = append(current, validate.StepResult{Step: name, Status: status, Conditions: []validate.ConditionResult{{Status: status}}})
	}
	for _, relaxedRelativeLimits := range []bool{false, true} {
		thresholds := DefaultThresholds()
		if relaxedRelativeLimits {
			thresholds.MaxStepsDegradation, thresholds.MaxConditionsDegradation = 1, 1
		}
		if result := EvaluateGate(current, thresholds, b); result.Passed {
			t.Fatalf("18/20 failures passed with relativeRelaxed=%t", relaxedRelativeLimits)
		}
	}
}

func TestBaselinePolicyRejectsInvalidNumbers(t *testing.T) {
	for _, n := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		thresholds := DefaultThresholds()
		thresholds.MaxConditionsDegradation = n
		if err := ValidateThresholds(thresholds); err == nil {
			t.Errorf("accepted invalid threshold %v", n)
		}
	}
	if result := EvaluateGate(nil, DefaultThresholds(), &BaselineData{}); result.Passed {
		t.Fatal("empty baseline passed")
	}
}

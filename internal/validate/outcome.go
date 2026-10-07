// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import "github.com/choreoatlas2025/cli/internal/verdict"

// Compatibility adapters; the authoritative decision belongs to verdict.
type Outcome = verdict.Outcome

func AllStepsPassed(steps []StepResult) bool { return verdict.AllStepsPassed(steps) }
func StepPassed(step StepResult) bool        { return verdict.StepPassed(step) }
func FinalOutcome(steps []StepResult, checked, passed bool) Outcome {
	return verdict.FinalOutcome(steps, checked, passed)
}

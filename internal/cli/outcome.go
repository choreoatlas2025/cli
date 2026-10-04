// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func reportOutcome(steps []validate.StepResult, gate *html.GateResult) validate.Outcome {
	if gate == nil {
		return validate.FinalOutcome(steps, false, true)
	}
	return validate.FinalOutcome(steps, gate.Checked, gate.Passed)
}

func outcomeExitCode(outcome validate.Outcome) int {
	switch outcome.Status {
	case "VALIDATION_FAILED":
		return exitcode.ValidationFailed
	case "GATE_FAILED":
		return exitcode.GateFailed
	case "PASS":
		return exitcode.OK
	default:
		return exitcode.CLIError
	}
}

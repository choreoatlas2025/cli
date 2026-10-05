// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/choreoatlas2025/cli/internal/result"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func outcomeExitCode(outcome validate.Outcome) int {
	return result.ExitCode(outcome)
}

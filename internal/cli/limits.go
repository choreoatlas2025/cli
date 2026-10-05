// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/input"
)

func resourceFlags(fs *flag.FlagSet) *input.Limits {
	l := input.Limits{}.Normalized()
	fs.Int64Var(&l.MaxInputBytes, "max-input-bytes", l.MaxInputBytes, "Maximum bytes per input file (total input budget is twice this)")
	fs.IntVar(&l.MaxSpans, "max-spans", l.MaxSpans, "Maximum trace span count")
	fs.IntVar(&l.MaxSteps, "max-steps", l.MaxSteps, "Maximum contract call count")
	fs.Uint64Var(&l.MaxCELCost, "max-cel-cost", l.MaxCELCost, "Maximum CEL cost per expression and aggregate request cost")
	fs.Int64Var(&l.TimeoutMs, "validation-timeout-ms", l.TimeoutMs, "Cooperative preparation/evaluation timeout in milliseconds")
	return &l
}

func checkResourceFlags(l input.Limits) error {
	if l.MaxInputBytes == 0 || l.MaxSpans == 0 || l.MaxSteps == 0 || l.MaxCELCost == 0 || l.TimeoutMs == 0 {
		return fmt.Errorf("invalid resource limits: flags must be positive")
	}
	return l.Validate()
}

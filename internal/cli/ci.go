// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
)

func runCIGate(args []string) {
	// Dynamic validation includes schema and lint on the same captured contract.
	fs := flag.NewFlagSet("ci-gate", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	tracePath := fs.String("trace", "", "trace.json path")
	_ = fs.Parse(args)

	runValidate([]string{"--flow", *flowPath, "--trace", *tracePath})
}

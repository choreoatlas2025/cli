package cli

import (
	"flag"
)

func runCIGate(args []string) {
	// CI Gate = lint + validate
	fs := flag.NewFlagSet("ci-gate", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	tracePath := fs.String("trace", "", "trace.json path")
	_ = fs.Parse(args)

	runLint([]string{"--flow", *flowPath})
	runValidate([]string{"--flow", *flowPath, "--trace", *tracePath})
}
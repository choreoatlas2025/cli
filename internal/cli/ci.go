package cli

import (
	"flag"
)

func runCIGate(args []string) {
	// CI Gate = lint + validate
	fs := flag.NewFlagSet("ci-gate", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec 文件路径")
	tracePath := fs.String("trace", "", "trace.json 路径")
	_ = fs.Parse(args)

	runLint([]string{"--flow", *flowPath})
	runValidate([]string{"--flow", *flowPath, "--trace", *tracePath})
}
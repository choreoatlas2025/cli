// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/baseline"
	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func runBaseline(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "baseline command requires a subcommand: record\n")
		exitErr(fmt.Errorf("usage: flowspec baseline <record>"))
	}

	subcommand := args[0]
	switch subcommand {
	case "record":
		runBaselineRecord(args[1:])
	default:
		exitErr(fmt.Errorf("unknown baseline subcommand: %s", subcommand))
	}
}

func runBaselineRecord(args []string) {
	fs := flag.NewFlagSet("baseline record", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	tracePath := fs.String("trace", "", "trace.json path")
	outputPath := fs.String("out", "baseline.json", "baseline output file path")
	_ = fs.Parse(args)

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace parameter is required"))
	}

	// Load and validate flow specification
	flow, opIndex, err := loadAndValidateFlow(*flowPath)
	if err != nil {
		exitErr(err)
	}

	// Load trace data
	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	// Perform validation to get results
	results, ok := validate.ValidateAgainstTrace(flow, opIndex, tr)
	if !ok {
		fmt.Fprintln(os.Stderr, "Validation failed; baseline not recorded.")
		os.Exit(exitcode.ValidationFailed)
	}

	// Record baseline
	baselineData, err := baseline.RecordBaseline(flow, results, *flowPath)
	if err != nil {
		exitErr(fmt.Errorf("failed to record baseline: %w", err))
	}

	// Save baseline
	if err := baseline.SaveBaseline(baselineData, *outputPath); err != nil {
		exitErr(fmt.Errorf("failed to save baseline: %w", err))
	}

	fmt.Printf("Baseline recorded: %s\n", *outputPath)
	fmt.Printf("  Flow: %s\n", baselineData.FlowID)
	fmt.Printf("  Steps Total: %d\n", baselineData.StepsTotal)
	fmt.Printf("  Covered Steps: %d\n", len(baselineData.CoveredSteps))
	fmt.Printf("  Coverage: %.1f%%\n", float64(len(baselineData.CoveredSteps))/float64(baselineData.StepsTotal)*100)
}

// loadAndValidateFlow loads flow spec and validates it
func loadAndValidateFlow(flowPath string) (*spec.FlowSpec, map[string]map[string]spec.ServiceOperation, error) {
	flow, err := spec.LoadFlowSpec(flowPath)
	if err != nil {
		return nil, nil, err
	}

	_, opIndex, err := flow.BuildOperationIndex(flowPath)
	if err != nil {
		return nil, nil, err
	}

	return flow, opIndex, nil
}

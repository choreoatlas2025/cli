// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/baseline"
	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
	"github.com/choreoatlas2025/cli/internal/verdict"
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
	semantic := fs.Bool("semantic", true, "Enable semantic validation (CEL)")
	causality := fs.String("causality", "temporal", "Causality mode: strict|temporal|off")
	tolerance := fs.Int64("causality-tolerance", 50, "Causality tolerance in milliseconds")
	limits := resourceFlags(fs)
	_ = fs.Parse(args)
	if err := checkResourceFlags(*limits); err != nil {
		exitErr(err)
	}
	config := spec.ValidationConfig{Semantic: *semantic, Causality: *causality, ToleranceMs: *tolerance, Limits: *limits}
	if err := config.Validate(); err != nil {
		exitErr(err)
	}

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace parameter is required"))
	}

	// Load and validate flow specification
	files := input.NewSnapshotWithLimit(nil, limits.MaxInputBytes)
	contract, err := loadAndValidateContract(*flowPath, files)
	if err != nil {
		exitErr(err)
	}

	// Load trace data
	tr, traceHash, err := loadTraceWithLimits(*tracePath, files, *limits)
	if err != nil {
		exitErr(err)
	}

	// Perform validation to get results
	results, _ := validate.ValidateAgainstTrace(contract.Flow, contract.Operations, tr, config)
	if !verdict.AllStepsPassed(results) {
		fmt.Fprintln(os.Stderr, "Validation failed; baseline not recorded.")
		os.Exit(exitcode.ValidationFailed)
	}

	// Record baseline
	provenance, err := executionIdentity(tr, traceHash, config, files)
	if err != nil {
		exitErr(err)
	}
	baselineData, err := baseline.RecordBaseline(contract, results, provenance)
	if err != nil {
		if errors.Is(err, baseline.ErrIncompleteValidation) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(exitcode.ValidationFailed)
		}
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
	contract, err := loadAndValidateContract(flowPath, input.NewSnapshot(nil))
	if err != nil {
		return nil, nil, err
	}
	return contract.Flow, contract.Operations, nil
}

func loadAndValidateContract(flowPath string, files *input.Snapshot) (*spec.ContractSnapshot, error) {
	contract, issues, err := loadContractWithFiles(flowPath, true, files)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if issue.Level == "ERROR" {
			return nil, fmt.Errorf("invalid contract: %s", issue.Msg)
		}
	}
	return contract, nil
}

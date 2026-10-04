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
	"github.com/choreoatlas2025/cli/internal/report/html"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec file path")
	tracePath := fs.String("trace", "", "trace.json path")
	reportFormat := fs.String("report-format", "", "Report format: json|junit|html")
	reportOut := fs.String("report-out", "", "Report output path")
	semantic := fs.Bool("semantic", true, "Enable semantic validation (CEL)")
	baselinePath := fs.String("baseline", "", "Baseline file path")
	thresholdSteps := fs.Float64("threshold-steps", 0.9, "Step coverage threshold")
	thresholdConds := fs.Float64("threshold-conds", 0.95, "Condition pass rate threshold")
	maxStepsDegradation := fs.Float64("max-steps-degradation", 0, "Maximum relative step coverage degradation with a baseline (0..1)")
	maxCondsDegradation := fs.Float64("max-conds-degradation", 0, "Maximum relative condition pass rate degradation with a baseline (0..1)")
	skipAsFail := fs.Bool("skip-as-fail", false, "Treat SKIP conditions as FAIL")
	causalityMode := fs.String("causality", "temporal", "Causality check mode: strict|temporal|off (default: temporal)")
	causalityTolerance := fs.Int("causality-tolerance", 50, "Causality constraint tolerance in milliseconds (default: 50ms)")
	baselineMissing := fs.String("baseline-missing", "fail", "Baseline missing strategy: fail|treat-as-absolute")
	_ = fs.Parse(args)
	thresholds := baseline.ThresholdConfig{
		StepsThreshold: *thresholdSteps, ConditionsThreshold: *thresholdConds,
		MaxStepsDegradation: *maxStepsDegradation, MaxConditionsDegradation: *maxCondsDegradation, SkipAsFail: *skipAsFail,
	}
	if err := baseline.ValidateThresholds(thresholds); err != nil {
		exitErr(err)
	}
	config := spec.ValidationConfig{Semantic: *semantic, Causality: *causalityMode, ToleranceMs: int64(*causalityTolerance)}
	if err := configureValidation(config); err != nil {
		exitErr(err)
	}
	if *baselineMissing != "fail" && *baselineMissing != "treat-as-absolute" {
		exitErr(fmt.Errorf("invalid baseline-missing strategy: %s", *baselineMissing))
	}

	// Input parameter validation
	if *tracePath == "" {
		exitErr(errors.New("--trace parameter is required"))
	}

	flow, opIndex, issues, err := loadContract(*flowPath, true)
	if err != nil {
		exitErr(err)
	}
	for _, is := range issues {
		fmt.Printf("[LINT-%s] %s\n", is.Level, is.Msg)
	}
	for _, is := range issues {
		if is.Level == "ERROR" {
			fmt.Println("Lint contains ERROR, terminating Validate")
			os.Exit(exitcode.InputError)
		}
	}

	// Load trace data
	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	results, _ := validate.ValidateAgainstTrace(flow, opIndex, tr)

	// Baseline gate check
	var gateResult *baseline.GateResult
	var baselineData *baseline.BaselineData
	baselineExpected := *baselinePath != ""

	if baselineExpected {
		// Load baseline for comparison
		var err error
		baselineData, err = baseline.LoadBaseline(*baselinePath)
		if err != nil {
			// Handle baseline missing according to strategy
			if *baselineMissing == "fail" {
				exitErr(fmt.Errorf("failed to load baseline file %s: %w", *baselinePath, err))
			} else if errors.Is(err, os.ErrNotExist) {
				fmt.Printf("[WARN] Baseline file not available, falling back to absolute threshold mode: %v\n", err)
				baselineData = nil
			} else {
				exitErr(err)
			}
		}
		if baselineData != nil {
			if err := baseline.ValidateCompatibility(baselineData, flow, *flowPath); err != nil {
				exitErr(err)
			}
			current, err := executionIdentity(tr, *tracePath, config)
			if err != nil {
				exitErr(err)
			}
			if err := baseline.ValidateExecution(baselineData, current); err != nil {
				exitErr(err)
			}
		}
	}

	// Execute threshold gate (with optional baseline)
	gateResult = baseline.EvaluateGate(results, thresholds, baselineData)
	outcome := validate.FinalOutcome(results, gateResult.Checked, gateResult.Passed)

	// Generate report (if format and path specified)
	if *reportFormat != "" && *reportOut != "" {
		contract, err := spec.IdentifyContract(flow, *flowPath)
		if err != nil {
			exitErr(err)
		}
		traceHash, err := spec.HashFile(*tracePath)
		if err != nil {
			exitErr(err)
		}
		inputs := &html.InputBinding{Contract: contract, TraceHash: traceHash, Version: Version, GitCommit: GitCommit, Semantic: *semantic, Causality: *causalityMode, ToleranceMs: int64(*causalityTolerance)}
		binaryPath, err := os.Executable()
		if err != nil {
			exitErr(err)
		}
		inputs.ValidatorHash, err = spec.HashFile(binaryPath)
		if err != nil {
			exitErr(err)
		}
		inputs.Policy = gateResult.Details
		// A failed mixed-ID trace remains reportable; expose that identity was not verified.
		inputs.TraceIdentity, err = trace.Identify(tr.Spans)
		if err != nil {
			inputs.TraceIdentity = trace.Identity{Binding: "invalid"}
		}
		if baselineData != nil {
			inputs.BaselineProvenance = &baselineData.Provenance
			inputs.BaselineHash, err = spec.HashFile(*baselinePath)
			if err != nil {
				exitErr(err)
			}
		}
		var format ReportFormat
		switch *reportFormat {
		case "json":
			format = ReportJSON
		case "junit":
			format = ReportJUnit
		case "html":
			format = ReportHTML
		default:
			exitErr(fmt.Errorf("unsupported report format: %s", *reportFormat))
		}

		// Convert baseline GateResult to html.GateResult for report
		var htmlGateResult *html.GateResult
		if gateResult != nil {
			htmlGateResult = &html.GateResult{
				Checked:    gateResult.Checked,
				Passed:     gateResult.Passed,
				Details:    gateResult.Details,
				Violations: gateResult.Violations,
			}
		}

		if err := WriteReport(*reportOut, format, results, tr.Spans, htmlGateResult, inputs); err != nil {
			exitErr(fmt.Errorf("failed to generate report: %w", err))
		}
		fmt.Printf("Report saved: %s (format: %s)\n", *reportOut, *reportFormat)
	}

	// Console output
	for _, r := range results {
		if r.Status == "PASS" {
			fmt.Printf("[PASS] %s (%s)\n", r.Step, r.Call)
		} else {
			fmt.Printf("[FAIL] %s (%s) - %s\n", r.Step, r.Call, r.Message)
		}
	}

	// Gate result output and exit code determination
	if gateResult != nil && gateResult.Checked {
		fmt.Printf("\n[GATE] Baseline Gate: ")
		if gateResult.Passed {
			fmt.Println("PASSED ✓")
			details := gateResult.Details

			// Display current metrics
			if stepsCoverage, ok := details["stepsCoverage"].(float64); ok {
				if stepsThreshold, ok := details["stepsThreshold"].(float64); ok {
					fmt.Printf("  Steps Coverage: %.1f%% (>= %.1f%%)\n", stepsCoverage*100, stepsThreshold*100)
				}
			}
			if conditionsRate, ok := details["conditionsRate"].(float64); ok {
				if conditionsThreshold, ok := details["conditionsThreshold"].(float64); ok {
					fmt.Printf("  Conditions Pass Rate: %.1f%% (>= %.1f%%)\n", conditionsRate*100, conditionsThreshold*100)
				}
			}

			// Display baseline comparison if available
			if baselineData != nil {
				fmt.Println("  Baseline Comparison:")
				if baselineStepsCoverage, ok := details["baselineStepsCoverage"].(float64); ok {
					if stepsDeltaPct, ok := details["stepsDeltaPct"].(float64); ok {
						fmt.Printf("    Steps: %.1f%% baseline → %.1f%% current (delta: %+.1f%%)\n",
							baselineStepsCoverage*100,
							details["stepsCoverage"].(float64)*100,
							stepsDeltaPct*100)
					}
				}
				if baselineConditionsRate, ok := details["baselineConditionsRate"].(float64); ok {
					if conditionsDeltaPct, ok := details["conditionsDeltaPct"].(float64); ok {
						fmt.Printf("    Conditions: %.1f%% baseline → %.1f%% current (delta: %+.1f%%)\n",
							baselineConditionsRate*100,
							details["conditionsRate"].(float64)*100,
							conditionsDeltaPct*100)
					}
				}
			}
		} else {
			fmt.Println("FAILED ✗")
			for _, violation := range gateResult.Violations {
				fmt.Printf("  - %s\n", violation)
			}
		}
	}

	// Exit code determination: validation failure or gate failure should exit non-zero
	if !outcome.Success {
		os.Exit(outcomeExitCode(outcome))
	}

	fmt.Println("Validate: OK")
}

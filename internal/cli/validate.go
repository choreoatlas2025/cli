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
	skipAsFail := fs.Bool("skip-as-fail", false, "Treat SKIP conditions as FAIL")
	causalityMode := fs.String("causality", "temporal", "Causality check mode: strict|temporal|off (default: temporal)")
	causalityTolerance := fs.Int("causality-tolerance", 50, "Causality constraint tolerance in milliseconds (default: 50ms)")
	baselineMissing := fs.String("baseline-missing", "fail", "Baseline missing strategy: fail|treat-as-absolute")
	_ = fs.Parse(args)

	// Input parameter validation
	if *tracePath == "" {
		exitErr(errors.New("--trace parameter is required"))
	}

	flow, err := spec.LoadFlowSpec(*flowPath)
	if err != nil {
		exitErr(err)
	}
	_, opIndex, err := flow.BuildOperationIndex(*flowPath)
	if err != nil {
		exitErr(err)
	}

	// lint 检查
	issues, err := validate.LintFlow(*flowPath, flow, opIndex)
	if err != nil {
		exitErr(err)
	}
	for _, is := range issues {
		fmt.Printf("[LINT-%s] %s\n", is.Level, is.Msg)
	}
	for _, is := range issues {
		if is.Level == "ERROR" {
			fmt.Println("Lint 存在 ERROR，终止 Validate")
			os.Exit(exitcode.InputError)
		}
	}

	// Load trace data
	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	// Set semantic validation switch
	validate.EnableSemantic = *semantic

	// Set causality check mode
	switch validate.CausalityMode(*causalityMode) {
	case validate.CausalityStrict:
		validate.GlobalCausalityMode = validate.CausalityStrict
	case validate.CausalityTemporal:
		validate.GlobalCausalityMode = validate.CausalityTemporal
	case validate.CausalityOff:
		validate.GlobalCausalityMode = validate.CausalityOff
	default:
		exitErr(fmt.Errorf("Invalid causality mode: %s, supported modes: strict|temporal|off", *causalityMode))
	}

	// Set causality tolerance
	validate.GlobalCausalityToleranceMs = int64(*causalityTolerance)

	results, ok := validate.ValidateAgainstTrace(flow, opIndex, tr)

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
				exitErr(fmt.Errorf("Failed to load baseline file %s: %w", *baselinePath, err))
			} else if *baselineMissing == "treat-as-absolute" {
				fmt.Printf("[WARN] Baseline file not available, falling back to absolute threshold mode: %v\n", err)
				baselineData = nil
				baselineExpected = false
			}
		}
	}

	// Execute threshold gate (with optional baseline)
	thresholds := baseline.ThresholdConfig{
		StepsThreshold:      *thresholdSteps,
		ConditionsThreshold: *thresholdConds,
		SkipAsFail:          *skipAsFail,
	}
	gateResult = baseline.EvaluateGate(results, thresholds, baselineData)

	// Generate report (if format and path specified)
	if *reportFormat != "" && *reportOut != "" {
		var format ReportFormat
		switch *reportFormat {
		case "json":
			format = ReportJSON
		case "junit":
			format = ReportJUnit
		case "html":
			format = ReportHTML
		default:
			exitErr(fmt.Errorf("Unsupported report format: %s", *reportFormat))
		}

		// Convert baseline GateResult to html.GateResult for report
		var htmlGateResult *html.GateResult
		if gateResult != nil {
			htmlGateResult = &html.GateResult{
				Checked: gateResult.Checked,
				Passed:  gateResult.Passed,
				Details: gateResult.Details,
			}
		}

		if err := WriteReport(*reportOut, format, results, tr.Spans, htmlGateResult); err != nil {
			exitErr(fmt.Errorf("Failed to generate report: %w", err))
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
	if !ok {
		os.Exit(exitcode.ValidationFailed) // Validation failed
	}
	if gateResult != nil && gateResult.Checked && !gateResult.Passed {
		os.Exit(exitcode.GateFailed) // Gate failed
	}
	
	fmt.Println("Validate: OK")
}
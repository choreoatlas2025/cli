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
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec 文件路径")
	tracePath := fs.String("trace", "", "trace.json 路径")
	reportFormat := fs.String("report-format", "", "报告格式：json|junit|html")
	reportOut := fs.String("report-out", "", "报告输出路径")
	semantic := fs.Bool("semantic", true, "启用语义校验（CEL）")
	baselinePath := fs.String("baseline", "", "基线文件路径")
	thresholdSteps := fs.Float64("threshold-steps", 0.9, "步骤覆盖率阈值")
	thresholdConds := fs.Float64("threshold-conds", 0.95, "条件通过率阈值")
	skipAsFail := fs.Bool("skip-as-fail", false, "将SKIP条件视为FAIL")
	causalityMode := fs.String("causality", "temporal", "因果检查模式：strict|temporal|off（默认 temporal）")
	baselineMissing := fs.String("baseline-missing", "fail", "基线缺失策略：fail|treat-as-absolute")
	_ = fs.Parse(args)

	// 输入参数验证
	if *tracePath == "" {
		exitErr(errors.New("必须指定 --trace 参数"))
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

	// 加载 trace 数据
	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	// 设置语义校验开关
	validate.EnableSemantic = *semantic
	
	// 设置因果检查模式
	switch validate.CausalityMode(*causalityMode) {
	case validate.CausalityStrict:
		validate.GlobalCausalityMode = validate.CausalityStrict
	case validate.CausalityTemporal:
		validate.GlobalCausalityMode = validate.CausalityTemporal
	case validate.CausalityOff:
		validate.GlobalCausalityMode = validate.CausalityOff
	default:
		exitErr(fmt.Errorf("无效的因果检查模式: %s，支持的模式: strict|temporal|off", *causalityMode))
	}

	results, ok := validate.ValidateAgainstTrace(flow, opIndex, tr)

	// 基线门控检查
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
				exitErr(fmt.Errorf("基线文件加载失败 %s: %w", *baselinePath, err))
			} else if *baselineMissing == "treat-as-absolute" {
				fmt.Printf("[WARN] 基线文件不可用，回退到绝对阈值模式: %v\n", err)
				baselineData = nil
				baselineExpected = false
			}
		}
	}

	// 执行阈值门控（可能有基线文件）
	thresholds := baseline.ThresholdConfig{
		StepsThreshold:      *thresholdSteps,
		ConditionsThreshold: *thresholdConds,
		SkipAsFail:          *skipAsFail,
	}
	gateResult = baseline.EvaluateGate(results, thresholds, baselineData)

	// 生成报告（如果指定了格式和路径）
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
			exitErr(fmt.Errorf("不支持的报告格式: %s", *reportFormat))
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
			exitErr(fmt.Errorf("生成报告失败: %w", err))
		}
		fmt.Printf("报告已保存: %s (格式: %s)\n", *reportOut, *reportFormat)
	}

	// 控制台输出
	for _, r := range results {
		if r.Status == "PASS" {
			fmt.Printf("[PASS] %s (%s)\n", r.Step, r.Call)
		} else {
			fmt.Printf("[FAIL] %s (%s) - %s\n", r.Step, r.Call, r.Message)
		}
	}

	// 门控结果输出和退出码判定
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

	// 退出码判定：验证失败或门控失败都应该非零退出
	if !ok {
		os.Exit(exitcode.ValidationFailed) // 验证失败
	}
	if gateResult != nil && gateResult.Checked && !gateResult.Passed {
		os.Exit(exitcode.GateFailed) // 门控失败
	}
	
	fmt.Println("Validate: OK")
}
package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/baseline"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func runBaseline(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "baseline 命令需要子命令：record\n")
		exitErr(fmt.Errorf("usage: flowspec baseline <record>"))
	}

	subcommand := args[0]
	switch subcommand {
	case "record":
		runBaselineRecord(args[1:])
	default:
		exitErr(fmt.Errorf("未知的 baseline 子命令: %s", subcommand))
	}
}

func runBaselineRecord(args []string) {
	fs := flag.NewFlagSet("baseline record", flag.ExitOnError)
	flowPath := fs.String("flow", ".flowspec.yaml", "FlowSpec 文件路径")
	tracePath := fs.String("trace", "", "trace.json 路径")
	outputPath := fs.String("out", "baseline.json", "基线输出文件路径")
	_ = fs.Parse(args)

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace 参数是必需的"))
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
	results, _ := validate.ValidateAgainstTrace(flow, opIndex, tr)

	// Record baseline
	baselineData, err := baseline.RecordBaseline(flow, results, *flowPath)
	if err != nil {
		exitErr(fmt.Errorf("记录基线失败: %w", err))
	}

	// Save baseline
	if err := baseline.SaveBaseline(baselineData, *outputPath); err != nil {
		exitErr(fmt.Errorf("保存基线失败: %w", err))
	}

	fmt.Printf("基线已记录: %s\n", *outputPath)
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
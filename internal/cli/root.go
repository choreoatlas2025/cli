// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
)

// Execute runs the CLI command
func Execute() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(exitcode.CLIError)
	}
	cmd := os.Args[1]

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
		return
	case "version", "-v", "--version":
		runVersion(os.Args[2:])
		return
	case "init":
		runInit(os.Args[2:])
	case "lint":
		runLint(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "discover":
		runDiscover(os.Args[2:])
	case "ci-gate":
		runCIGate(os.Args[2:])
	case "baseline":
		runBaseline(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(exitcode.CLIError)
	}
}

func printHelp() {
	fmt.Print(`ChoreoAtlas CLI - Interactive Logic Governance Platform

Usage:
  choreoatlas <command> [options]
  ca <command> [options]  # alias

Commands:
  init       Bootstrap starter project with FlowSpec/ServiceSpec templates
    --mode string          Bootstrap mode: template|trace
    --trace string         trace.json file path for from-trace mode
    --ci string            GitHub Actions template: none|minimal|combo
    --examples             Copy examples/* starter assets
    --yes                  Accept defaults without interactive prompts
    --force                Overwrite existing files when present
    --out string           Target directory (default ".")
    --title string         Override FlowSpec title

  lint       Static validation of FlowSpec consistency and call coherence
    --flow string          FlowSpec file path (default ".flowspec.yaml")
    --schema               Enable JSON Schema strict validation (default true)

  validate   Dynamic validation against trace.json (Atlas Proof)
    --flow string          FlowSpec file path (default ".flowspec.yaml")
    --trace string         trace.json file path
    --semantic bool        Enable semantic validation (CEL) (default true)
    --causality string     Causality check mode: strict|temporal|off (default "temporal")
    --causality-tolerance int  Causality constraint tolerance in milliseconds (default 50)
    --baseline string      Baseline file path
    --baseline-missing string  Baseline missing strategy: fail|treat-as-absolute (default "fail")
    --threshold-steps float    Step coverage threshold (default 0.9)
    --threshold-conds float    Condition pass threshold (default 0.95)
    --skip-as-fail        Treat SKIP conditions as FAIL
    --report-format string Report format: json|junit|html
    --report-out string    Report output path

  discover   Generate FlowSpec from trace.json exploration (Atlas Scout)
    --trace string         trace.json file path (required)
    --out string          Generated FlowSpec output path (default "discovered.flowspec.yaml")
    --title string        Generated FlowSpec title

  ci-gate    CI scenario combining lint + validate with non-zero exit on failure (Atlas Proof)
    --flow string          FlowSpec file path
    --trace string         trace.json file path

  baseline   Baseline management tool (Atlas Proof)
    record                Record new baseline
      --flow string       FlowSpec file path (required)
      --trace string      trace.json file path (required)
      --out string        Baseline output file path (default "baseline.json")

Examples:
  choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
  ca validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json
  ca discover --trace examples/traces/successful-order.trace.json --out new-flow.yaml
  choreoatlas validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format junit --report-out report.xml

Edition Features:
  ce              Community Edition: Atlas Scout + Atlas Proof basic features
`)
}

var (
	// 这些变量在构建时通过 ldflags 注入
	Version      = "0.7.0-dev"
	GitCommit    = "unknown"
	BuildTime    = "unknown"
	BuildEdition = "ce" // CE版本标识
)

func runVersion(args []string) {
	// Display version with -ce suffix
	fmt.Printf("choreoatlas v%s-ce\n", Version)
	fmt.Printf("Edition: Community Edition (CE)\n")
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s\n", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	// 根据错误类型选择正确的退出码
	exitCode := exitcode.CLIError

	// 检查是否是文件/输入相关错误
	errStr := err.Error()
	if strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "cannot read") ||
		strings.Contains(errStr, "failed to load") ||
		strings.Contains(errStr, "failed to parse") ||
		strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "must specify") ||
		strings.Contains(errStr, "required") {
		exitCode = exitcode.InputError
	}

	os.Exit(exitCode)
}

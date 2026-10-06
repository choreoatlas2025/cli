// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
	"github.com/choreoatlas2025/cli/internal/spec"
	"gopkg.in/yaml.v3"
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
		// Domain-aware help
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "spec":
				printSpecHelp()
				return
			case "run":
				printRunHelp()
				return
			case "system":
				printSystemHelp()
				return
			default:
				fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[2])
				printHelp()
				os.Exit(exitcode.CLIError)
			}
		}
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
	case "flowspec":
		runFlowspec(os.Args[2:])
	case "spec":
		runSpecGroup(os.Args[2:])
	case "run":
		runRunGroup(os.Args[2:])
	case "system":
		runSystemGroup(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(exitcode.CLIError)
	}
}

func printHelp() {
	fmt.Print(`ChoreoAtlas CLI - Interactive Logic Governance Platform (CE)

Usage:
  choreoatlas <command> [options]
  ca <command> [options]  # alias

Domain commands:
  spec        Flow/Service specifications (discover | lint | validate | convert)
  run         Runtime validation (validate)
  system      System utilities (version)

Top-level aliases:
  init       Bootstrap starter project
  lint       ≙ spec lint
  validate   ≙ run validate
  discover   ≙ spec discover
  ci-gate    Composite CI gate (lint + validate, CE)
  baseline   Baseline recorder (record)

Key flags:
  --format <human|json|ndjson|junit|html>  Command-specific machine readable output
  --summary                              Write step summary to $GITHUB_STEP_SUMMARY when present
  --log-level <debug|info|warn|error>    Verbosity for structured logs (stderr)

Examples:
  choreoatlas spec discover --trace examples/traces/successful-order.trace.json --out discovered.flowspec.yaml
  choreoatlas spec lint --flow examples/flows/order-fulfillment.flowspec.yaml --schema
  ca run validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json --report-format junit --report-out report.xml --summary

Exit Codes:
  0  success
  1  generic error (invalid args, etc.)
  2  input/schema errors
  3  validation (trace vs spec) failed
  4  gate thresholds failed
`)
}

var (
	// 这些变量在构建时通过 ldflags 注入
	Version      = "0.8.0-dev"
	GitCommit    = "unknown"
	BuildTime    = "unknown"
	BuildChannel = "source"
	BuildEdition = "ce" // CE版本标识
)

func runVersion(args []string) {
	fmt.Printf("choreoatlas %s\n", formatCEVersion(Version))
	fmt.Printf("Edition: Community Edition (CE)\n")
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Build Channel: %s\n", BuildChannel)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s\n", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}

// formatCEVersion preserves release tags and git-describe metadata while adding
// the version prefix and CE identifier only when missing.
func formatCEVersion(version string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		version = "dev"
	}
	base, metadata, hasMetadata := strings.Cut(version, "+")
	hasCE := false
	for _, suffix := range strings.Split(base, "-")[1:] {
		if suffix == "ce" || strings.HasPrefix(suffix, "ce.") {
			hasCE = true
			break
		}
	}
	if !hasCE {
		base += "-ce"
	}
	if hasMetadata {
		base += "+" + metadata
	}
	return "v" + base
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	// 根据错误类型选择正确的退出码
	exitCode := exitcode.CLIError

	// 检查是否是文件/输入相关错误
	errStr := err.Error()
	if errors.Is(err, errIncompleteEvidence) || strings.Contains(errStr, "no such file") ||
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

// Domain specific help (CE only)
func printSpecHelp() {
	fmt.Print(`Spec domain (CE)

Usage:
  choreoatlas spec <discover|lint|validate|convert> [options]

Commands:
  discover  From trace to initial ServiceSpec + FlowSpec
    --trace <file> --out <path> [--title <text>]
  lint      Static checks (structure + coherence + variables + parallel reachability)
    --flow <file> [--schema]
  validate  Alias of lint for spec-level validation
  convert   graph(DAG) -> flow (CE default)
    --in <file> --to flow --out <file>

Notes:
  - CE defaults to flow format; graph is supported for conversion and linting.
  - Use --format json|ndjson|junit for machine-readable output.
`)
}

func printRunHelp() {
	fmt.Print(`Run domain (CE)

Usage:
  choreoatlas run validate [options]

validate options:
  --flow <file> --trace <file>
  --baseline <file>
  --threshold-steps <float> --threshold-conds <float> [--skip-as-fail]
  --report-format <json|junit|html> --report-out <file> [--summary]
  --causality <strict|temporal|off>

Notes:
  - --summary writes GitHub Step Summary when GITHUB_STEP_SUMMARY is present.
  - With --format json stdout emits exactly one JSON object; with ndjson one JSON object per line.
`)
}

func printSystemHelp() { fmt.Println("Usage: choreoatlas system version") }

// --- Flowspec alias and convert ---
func runFlowspec(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: choreoatlas flowspec <validate|lint|convert> [options]")
		return
	}
	sub := args[0]
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	switch sub {
	case "validate":
		runValidate(rest)
	case "lint":
		runLint(rest)
	case "convert":
		runConvert(rest)
	default:
		fmt.Printf("Unknown flowspec subcommand: %s\n", sub)
	}
}

func runConvert(args []string) {
	fs := flag.NewFlagSet("flowspec convert", flag.ExitOnError)
	in := fs.String("in", ".flowspec.yaml", "Input FlowSpec file")
	out := fs.String("out", "converted.flowspec.yaml", "Output FlowSpec file")
	to := fs.String("to", "flow", "Target format: flow (graph-to-flow conversion)")
	_ = fs.Parse(args)

	if *to != "flow" {
		exitErr(fmt.Errorf("only --to flow is supported currently"))
	}
	fspec, _, err := loadAndValidateFlow(*in)
	if err != nil {
		exitErr(err)
	}
	if !fspec.IsGraphMode() {
		exitErr(fmt.Errorf("input is not in graph(DAG) format"))
	}
	conv, err := spec.ConvertGraphToFlow(fspec)
	if err != nil {
		exitErr(err)
	}
	inputPath, err := filepath.Abs(*in)
	if err != nil {
		exitErr(err)
	}
	outputPath, err := filepath.Abs(*out)
	if err != nil {
		exitErr(err)
	}
	for alias, binding := range conv.Services {
		if !filepath.IsAbs(binding.Spec) {
			binding.Spec, err = filepath.Rel(filepath.Dir(outputPath), spec.ResolvePath(inputPath, binding.Spec))
			if err != nil {
				exitErr(err)
			}
			conv.Services[alias] = binding
		}
	}
	data, err := yaml.Marshal(conv)
	if err != nil {
		exitErr(fmt.Errorf("failed to marshal converted FlowSpec: %w", err))
	}
	if err := validateAndPersistFlow(string(data), *out, filepath.Dir(*out)); err != nil {
		exitErr(err)
	}
	fmt.Printf("Converted graph -> flow: %s\n", *out)
}

// --- Domain routers ---
func runSpecGroup(args []string) {
	if len(args) == 0 {
		printSpecHelp()
		os.Exit(1)
	}
	sub := args[0]
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	switch sub {
	case "discover":
		runDiscover(rest)
	case "lint":
		runLint(rest)
	case "validate":
		runLint(rest)
	case "convert":
		runConvert(rest)
	default:
		fmt.Fprintf(os.Stderr, "Unknown spec subcommand: %s\n\n", sub)
		printSpecHelp()
		os.Exit(1)
	}
}

func runRunGroup(args []string) {
	if len(args) == 0 {
		printRunHelp()
		os.Exit(1)
	}
	sub := args[0]
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	switch sub {
	case "validate":
		runValidate(rest)
	default:
		fmt.Fprintf(os.Stderr, "Unknown run subcommand: %s\n\n", sub)
		printRunHelp()
		os.Exit(1)
	}
}

func runSystemGroup(args []string) {
	if len(args) == 0 {
		printSystemHelp()
		os.Exit(1)
	}
	sub := args[0]
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	switch sub {
	case "version":
		runVersion(rest)
	default:
		fmt.Fprintf(os.Stderr, "Unknown system subcommand: %s\n", sub)
		os.Exit(1)
	}
}

// convertGraphToFlow performs DAG→flow conversion using FlowSpec/GraphSpec types from spec package
// conversion helpers moved to internal/spec/convert.go

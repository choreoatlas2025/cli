// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
    "flag"
    "fmt"
    "os"
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
			case "workspace":
				printWorkspaceHelp()
				return
			case "platform":
				printPlatformHelp()
				return
			case "plugin":
				printPluginHelp()
				return
			case "config":
				printConfigHelp()
				return
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
	case "workspace":
		runWorkspaceGroup(os.Args[2:])
	case "platform":
		runPlatformGroup(os.Args[2:])
	case "plugin":
		runPluginGroup(os.Args[2:])
	case "config":
		runConfigGroup(os.Args[2:])
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
  workspace   Collaboration tooling (not yet available in CE)
  platform    Deployment & governance (not yet available in CE)
  plugin      Plugin management (not yet available in CE)
  config      CLI configuration helpers (not yet available in CE)
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

func printSystemHelp()    { fmt.Println("system domain (CE): version available; upgrade/doctor/cache not included in CE.") }
func printWorkspaceHelp() { fmt.Println("workspace domain not available in CE. Use ci-gate for gate workflows.") }
func printPlatformHelp()  { fmt.Println("platform domain not available in CE build.") }
func printPluginHelp()    { fmt.Println("plugin domain not available in CE build.") }
func printConfigHelp()    { fmt.Println("config domain not available in CE. Use --config or CHOREO_* environment variables.") }

// --- Flowspec alias and convert ---
func runFlowspec(args []string) {
    if len(args) == 0 {
        fmt.Println("Usage: choreoatlas flowspec <validate|lint|convert> [options]")
        return
    }
    sub := args[0]
    rest := []string{}
    if len(args) > 1 { rest = args[1:] }
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
    to := fs.String("to", "flow", "Target format: flow|graph (only flow supported in CE)")
    _ = fs.Parse(args)

    if *to != "flow" {
        exitErr(fmt.Errorf("only --to flow is supported currently"))
    }
    fspec, err := spec.LoadFlowSpec(*in)
    if err != nil { exitErr(err) }
    if !fspec.IsGraphMode() {
        exitErr(fmt.Errorf("input is not in graph(DAG) format"))
    }
    conv := convertGraphToFlow(fspec)
    if err := writeFlowSpec(*out, conv); err != nil {
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
    if len(args) > 1 { rest = args[1:] }
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
    if len(args) > 1 { rest = args[1:] }
    switch sub {
    case "validate":
        runValidate(rest)
    default:
        fmt.Fprintf(os.Stderr, "Unknown run subcommand: %s\n\n", sub)
        printRunHelp()
        os.Exit(1)
    }
}

func runWorkspaceGroup(args []string) {
    _ = args
    fmt.Fprintln(os.Stderr, "workspace domain is not available in CE yet. Use ci-gate or top-level commands where applicable.")
    os.Exit(1)
}

func runPlatformGroup(args []string) {
    _ = args
    fmt.Fprintln(os.Stderr, "platform domain is not available in CE.")
    os.Exit(1)
}

func runPluginGroup(args []string) {
    _ = args
    fmt.Fprintln(os.Stderr, "plugin management is not available in CE.")
    os.Exit(1)
}

func runConfigGroup(args []string) {
    _ = args
    fmt.Fprintln(os.Stderr, "config commands are not available in CE yet. Configure via --config or CHOREO_* environment variables.")
    os.Exit(1)
}

func runSystemGroup(args []string) {
    if len(args) == 0 {
        printSystemHelp()
        os.Exit(1)
    }
    sub := args[0]
    rest := []string{}
    if len(args) > 1 { rest = args[1:] }
    switch sub {
    case "version":
        runVersion(rest)
    default:
        fmt.Fprintf(os.Stderr, "system %s is not implemented in CE\n", sub)
        os.Exit(1)
    }
}

// convertGraphToFlow performs DAG→flow conversion using FlowSpec/GraphSpec types from spec package
func convertGraphToFlow(fs *spec.FlowSpec) *spec.FlowSpec {
    if fs == nil || fs.Graph == nil { return fs }
    g := fs.Graph
    g.EnsureEdges()
    inDeg := map[string]int{}
    adj := map[string][]string{}
    nodes := map[string]spec.GraphNode{}
    for _, n := range g.Nodes {
        nodes[n.ID] = n
        inDeg[n.ID] = 0
    }
    for _, e := range g.Edges {
        adj[e.From] = append(adj[e.From], e.To)
        inDeg[e.To]++
    }
    origIn := map[string]int{}
    for k, v := range inDeg { origIn[k] = v }
    visited := map[string]bool{}
    queue := []string{}
    for id, d := range inDeg { if d == 0 { queue = append(queue, id) } }

    var flow []spec.FlowStep
    for len(queue) > 0 {
        id := queue[0]
        queue = queue[1:]
        if visited[id] { continue }
        visited[id] = true
        n := nodes[id]
        succ := adj[id]
        par := []string{}
        for _, s := range succ { if origIn[s] == 1 { par = append(par, s) } }
        if len(par) > 1 {
            parent := spec.FlowStep{ Step: n.ID, Call: n.Call, Input: n.Input, Output: n.Output, Meta: n.Meta }
            for _, s := range par {
                child := spec.FlowStep{ Step: nodes[s].ID, Call: nodes[s].Call, Input: nodes[s].Input, Output: nodes[s].Output, Meta: nodes[s].Meta }
                parent.Parallel = append(parent.Parallel, child)
                visited[s] = true
                for _, ns := range adj[s] {
                    if inDeg[ns] > 0 { inDeg[ns]-- }
                    if inDeg[ns] == 0 { queue = append(queue, ns) }
                }
            }
            flow = append(flow, parent)
        } else {
            flow = append(flow, spec.FlowStep{ Step: n.ID, Call: n.Call, Input: n.Input, Output: n.Output, Meta: n.Meta })
            for _, s := range succ {
                if inDeg[s] > 0 { inDeg[s]-- }
                if inDeg[s] == 0 { queue = append(queue, s) }
            }
        }
    }
    out := &spec.FlowSpec{ Info: fs.Info, Services: fs.Services, Flow: flow }
    return out
}

// writeFlowSpec writes FlowSpec YAML to file (local helper)
func writeFlowSpec(path string, fs *spec.FlowSpec) error {
    b, err := yaml.Marshal(fs)
    if err != nil { return fmt.Errorf("failed to marshal flowspec: %w", err) }
    return os.WriteFile(path, b, 0644)
}

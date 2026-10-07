// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"
	"fmt"

	"github.com/choreoatlas2025/cli/internal/trace"
	"gopkg.in/yaml.v3"
)

func runDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	tracePath := fs.String("trace", "", "trace.json file path")
	out := fs.String("out", "discovered.flowspec.yaml", "FlowSpec output path")
	outServices := fs.String("out-services", "./services", "ServiceSpec output directory")
	title := fs.String("title", "Flow generated from trace", "FlowSpec title")
	noValidate := fs.Bool("no-validate", false, "Skip schema + lint validation gate (not recommended)")
	_ = fs.Parse(args)

	if *tracePath == "" {
		exitErr(fmt.Errorf("--trace parameter is required"))
	}

	tr, err := trace.LoadFromFile(*tracePath)
	if err != nil {
		exitErr(err)
	}

	if err := discoverAndPersist(tr, *title, *out, *outServices, *noValidate); err != nil {
		exitErr(err)
	}
	fmt.Printf("Generated FlowSpec: %s\n", *out)

	fmt.Println("Dual contract generation complete! Please adjust the generated specifications as needed.")
}

// generateFlowYAML shares the trace-shaped DAG generator across both entry points.
func generateFlowYAML(tr *trace.Trace, title, outServices string) (string, error) {
	flow, err := buildDiscoveredFlow(tr, title, outServices)
	if err != nil {
		return "", err
	}
	if err := qualifyDiscoveredStructure(flow, tr); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(flow)
	if err != nil {
		return "", err
	}
	return string(data) + "# Generated from observed call instances and span relationships.\n# Review sampled conditions and add business requirements and data mappings.\n", nil
}

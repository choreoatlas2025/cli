// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package discovery builds observational drafts without changing files. The
// application owns qualification of the complete output set and its commit.
package discovery

import (
	"github.com/choreoatlas2025/cli/internal/trace"
	"gopkg.in/yaml.v3"
)

// generateFlowYAML shares the trace-shaped DAG generator across both entry points.
func FlowYAML(tr *trace.Trace, title, outServices string) (string, error) {
	flow, err := BuildFlow(tr, title, outServices)
	if err != nil {
		return "", err
	}
	if err := qualifyStructure(flow, tr); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(flow)
	if err != nil {
		return "", err
	}
	return string(data) + "# Generated from observed call instances and span relationships.\n# Review sampled conditions and add business requirements and data mappings.\n", nil
}

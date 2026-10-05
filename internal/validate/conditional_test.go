// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestConditionalEdgesRejectDirectEngineBypass(t *testing.T) {
	previous := GlobalCausalityMode
	GlobalCausalityMode = CausalityOff
	t.Cleanup(func() { GlobalCausalityMode = previous })
	flow := &spec.FlowSpec{Graph: &spec.GraphSpec{
		Nodes: []spec.GraphNode{{ID: "A", Call: "svc.A"}, {ID: "B", Call: "svc.B"}},
		Edges: []spec.GraphEdge{{From: "A", To: "B", Condition: "false"}},
	}}
	tr := &trace.Trace{Spans: []trace.Span{{Name: "A", Service: "svc", StartNanos: 0, EndNanos: 1}, {Name: "B", Service: "svc", StartNanos: 2, EndNanos: 3}}}
	results, passed := ValidateAgainstTrace(flow, nil, tr)
	if passed || len(results) != 1 || results[0].Status != "FAIL" || !strings.Contains(results[0].Message, "unsupported conditional edge") {
		t.Fatalf("engine silently executed a conditional dependency: %+v", results)
	}
	flow.Graph.Edges[0].Condition = ""
	if results, passed := ValidateAgainstTrace(flow, nil, tr); !passed {
		t.Fatalf("unconditional control failed: %+v", results)
	}
}

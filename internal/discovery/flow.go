// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

// Discover a structural draft, not a total order or a set of inferred business
// requirements. Preserve every call instance and every available parent link.
// Timing constraints describe adjacent observed siblings, never a parent's
// completion before the start of its own child.
func BuildFlow(tr *trace.Trace, title, outServices string) (*spec.FlowSpec, error) {
	if tr == nil || len(tr.Spans) == 0 {
		return nil, fmt.Errorf("invalid discovery input: trace contains no spans")
	}
	if err := trace.ValidateTimestamps(tr.Spans, true); err != nil {
		return nil, err
	}
	for _, s := range tr.Spans {
		if s.Service == "" || s.Name == "" {
			return nil, fmt.Errorf("invalid discovery input: every span needs a service and operation name")
		}
	}
	calls, err := validate.BuildCallGraph(tr.Spans)
	if err != nil {
		return nil, fmt.Errorf("invalid discovery input: %w", err)
	}
	if violations := calls.ValidateEdgeConstraints(spec.DefaultValidationConfig().ToleranceMs * 1000000); len(violations) > 0 {
		return nil, fmt.Errorf("invalid discovery input: %s", violations[0].Message)
	}
	nodes := make([]*validate.CallNode, 0, len(calls.Nodes))
	for _, node := range calls.Nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].StartNanos != nodes[j].StartNanos {
			return nodes[i].StartNanos < nodes[j].StartNanos
		}
		if nodes[i].EndNanos != nodes[j].EndNanos {
			return nodes[i].EndNanos < nodes[j].EndNanos
		}
		return nodes[i].SpanID < nodes[j].SpanID
	})
	flow := &spec.FlowSpec{Info: spec.FlowInfo{Title: title}, Services: map[string]spec.ServiceBinding{}, Graph: &spec.GraphSpec{}}
	ids := make(map[string]string, len(nodes))
	for i, node := range nodes {
		id := fmt.Sprintf("step-%04d", i+1)
		ids[node.SpanID] = id
		s := trace.Span{Service: node.Service, Name: node.Operation, Attributes: node.Attributes}
		flow.Graph.Nodes = append(flow.Graph.Nodes, spec.GraphNode{ID: id, Call: node.Service + "." + spec.ComputeOperationID(s)})
		flow.Services[node.Service] = spec.ServiceBinding{Spec: filepath.ToSlash(filepath.Join(outServices, ServiceSpecFilename(node.Service)))}
	}
	// External parent IDs are retained as distinct sibling groups, rather than
	// treating unrelated partial captures as children of a fabricated root.
	previous := map[string]*validate.CallNode{}
	for _, node := range nodes {
		if node.Parent != nil {
			flow.Graph.Edges = append(flow.Graph.Edges, spec.GraphEdge{From: ids[node.Parent.SpanID], To: ids[node.SpanID], Relationship: "parent"})
		}
		parent, _ := node.Attributes["otlp.parent_span_id"].(string)
		if pred := previous[parent]; pred != nil {
			relationship := "follows"
			if pred.EndNanos > node.StartNanos {
				relationship = "concurrent"
			}
			flow.Graph.Edges = append(flow.Graph.Edges, spec.GraphEdge{From: ids[pred.SpanID], To: ids[node.SpanID], Relationship: relationship})
		}
		previous[parent] = node
	}
	if err := flow.Graph.ValidateGraphStructure(); err != nil {
		return nil, err
	}
	return flow, nil
}

// Check structure only. Observations containing failed attempts are legitimate
// inputs for discovery; CEL requirements are qualified separately and reviewed
// by the user. This must run before any generated destination is replaced.
func qualifyStructure(flow *spec.FlowSpec, tr *trace.Trace) error {
	config := spec.DefaultValidationConfig()
	config.Semantic = false
	_, results, ok := validate.ValidateWithPlan(flow, nil, tr, config)
	if !ok {
		for _, result := range results {
			if result.Status != "PASS" {
				return fmt.Errorf("generated structure does not match source trace: %s: %s", result.Step, result.Message)
			}
		}
		return fmt.Errorf("generated structure does not match source trace")
	}
	return nil
}

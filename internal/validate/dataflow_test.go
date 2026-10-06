// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestDataflowTypedInputAndOutput(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{
		{Step: "A", Call: "svc.A", Output: map[string]string{"created": "response.body", "count": "42"}},
		{Step: "B", Call: "svc.B", Input: map[string]any{"body": map[string]any{"payload": "${created}", "id": "id-${created.id}", "count": "${count}"}}},
	}}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"A": {Postconditions: map[string]string{"ok": "true"}}, "B": {Preconditions: map[string]string{"typed": `request.body.payload.items[0] == "x" && request.body.count == 42 && request.body.id == "id-42" && vars.created.id == 42`}}}}
	tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "A", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"response.body": map[string]any{"id": 42, "items": []any{"x"}}}}, {Service: "svc", Name: "B", StartNanos: 3, EndNanos: 4}}}
	if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); !passed {
		t.Fatalf("typed dataflow failed: %+v", results)
	}
	ops["svc"]["A"] = spec.ServiceOperation{Postconditions: map[string]string{"ok": "false"}}
	results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig())
	if passed || results[1].Status != "FAIL" {
		t.Fatalf("failed step exported validated data: %+v", results)
	}
}

func TestDataflowParallelScopes(t *testing.T) {
	for _, graph := range []bool{false, true} {
		flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "group", Parallel: []spec.FlowStep{{Step: "A", Call: "svc.A", Output: map[string]string{"created": "42"}}, {Step: "B", Call: "svc.B"}}}, {Step: "C", Call: "svc.C"}}}
		if graph {
			flow.Flow = nil
			flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "A", Call: "svc.A", Output: map[string]string{"created": "42"}}, {ID: "B", Call: "svc.B"}, {ID: "C", Call: "svc.C"}}, Edges: []spec.GraphEdge{{From: "A", To: "C"}, {From: "B", To: "C"}}}
		}
		ops := map[string]map[string]spec.ServiceOperation{"svc": {"A": {Postconditions: map[string]string{"ok": "true"}}, "B": {Preconditions: map[string]string{"isolated": "!has(vars.created)"}}, "C": {Preconditions: map[string]string{"joined": "vars.created == 42"}}}}
		tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "A", StartNanos: 1, EndNanos: 4}, {Service: "svc", Name: "B", StartNanos: 2, EndNanos: 3}, {Service: "svc", Name: "C", StartNanos: 5, EndNanos: 6}}}
		if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); !passed {
			t.Fatalf("graph=%t: sibling state leaked or join lost data: %+v", graph, results)
		}
		if graph {
			flow.Graph.Nodes[1].Output = map[string]string{"created": "43"}
		} else {
			flow.Flow[0].Parallel[1].Output = map[string]string{"created": "43"}
		}
		if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); passed {
			t.Fatalf("graph=%t: ambiguous independent outputs passed: %+v", graph, results)
		}
	}
}

func TestDataflowUnknownInputIsNotLiteral(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "A", Call: "svc.A", Input: map[string]any{"id": "${customerId}"}}}}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"A": {Preconditions: map[string]string{"present": `request.body.id != ""`}}}}
	tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "A", StartNanos: 1, EndNanos: 2}}}
	if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); passed {
		t.Fatalf("unknown input passed as placeholder text: %+v", results)
	}
	flow.Flow[0].Input = map[string]any{"id": "known"}
	if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); !passed {
		t.Fatalf("literal control failed: %+v", results)
	}
	flow.Flow[0].Output = map[string]string{"missing": "response.body.id"}
	if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); passed {
		t.Fatalf("unresolved output mapping passed: %+v", results)
	}
}

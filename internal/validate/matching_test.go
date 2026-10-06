// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestMatchingBindsInputsAndDistinctSpans(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{
		{Step: "first", Call: "svc.repeat", Input: map[string]any{"expected": 200}},
		{Step: "second", Call: "svc.repeat", Input: map[string]any{"expected": 201}},
	}}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"repeat": {Postconditions: map[string]string{"status": "request.body.expected == response.status"}}}}
	tr := &trace.Trace{Spans: []trace.Span{
		{Service: "svc", Name: "repeat", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"response.status": 200, "otlp.parent_span_id": ""}},
		{Service: "svc", Name: "repeat", StartNanos: 3, EndNanos: 4, Attributes: map[string]any{"response.status": 201}},
	}}
	results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig())
	if !passed || len(results) != 2 {
		t.Fatalf("valid repeated calls failed: %+v", results)
	}
	for _, r := range results {
		if len(r.Conditions) != 1 || r.Conditions[0].Status != "PASS" {
			t.Fatalf("input/span binding lost: %+v", r)
		}
	}
}

func TestMatchingParallelRequiresOverlapAndParentCall(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "parent", Call: "svc.parent", Parallel: []spec.FlowStep{{Step: "B", Call: "svc.B"}, {Step: "C", Call: "svc.C"}}}}}
	tr := &trace.Trace{Spans: []trace.Span{
		{Service: "svc", Name: "parent", StartNanos: 1, EndNanos: 10, Attributes: map[string]any{"otlp.span_id": "parent"}},
		{Service: "svc", Name: "B", StartNanos: 2, EndNanos: 4, Attributes: map[string]any{"otlp.span_id": "B", "otlp.parent_span_id": "parent"}},
		{Service: "svc", Name: "C", StartNanos: 5, EndNanos: 7, Attributes: map[string]any{"otlp.span_id": "C", "otlp.parent_span_id": "parent"}},
	}}
	if r, ok := ValidateAgainstTrace(flow, nil, tr, spec.DefaultValidationConfig()); ok {
		t.Fatalf("serial siblings accepted as parallel: %+v", r)
	}
	tr.Spans[2].StartNanos = 3
	if r, ok := ValidateAgainstTrace(flow, nil, tr, spec.DefaultValidationConfig()); !ok || len(r) != 3 {
		t.Fatalf("valid parent and parallel children failed: %+v", r)
	}
	tr.Spans = tr.Spans[1:]
	if r, ok := ValidateAgainstTrace(flow, nil, tr, spec.DefaultValidationConfig()); ok {
		t.Fatalf("missing parent accepted: %+v", r)
	}
}

func TestMatchingGraphUsesExactPredecessorInstance(t *testing.T) {
	config := spec.DefaultValidationConfig()
	config.ToleranceMs = 0
	flow := &spec.FlowSpec{Graph: &spec.GraphSpec{
		Nodes: []spec.GraphNode{{ID: "first", Call: "svc.repeat"}, {ID: "second", Call: "svc.repeat"}, {ID: "last", Call: "svc.last"}},
		Edges: []spec.GraphEdge{{From: "first", To: "second"}, {From: "second", To: "last"}},
	}}
	tr := &trace.Trace{Spans: []trace.Span{
		{Service: "svc", Name: "repeat", StartNanos: 1, EndNanos: 2},
		{Service: "svc", Name: "repeat", StartNanos: 3, EndNanos: 4},
		{Service: "svc", Name: "last", StartNanos: 2, EndNanos: 3},
	}}
	if r, ok := ValidateAgainstTrace(flow, nil, tr, config); ok {
		t.Fatalf("matched wrong predecessor instance: %+v", r)
	}
	tr.Spans[2].StartNanos, tr.Spans[2].EndNanos = 4, 5
	if r, ok := ValidateAgainstTrace(flow, nil, tr, config); !ok {
		t.Fatalf("valid repeated-call DAG failed: %+v", r)
	}
}

func TestMatchingRejectsDuplicateSpanIdentity(t *testing.T) {
	sp := trace.Span{Service: "svc", Name: "A", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"otlp.span_id": "same"}}
	if _, err := BuildCallGraph([]trace.Span{sp, sp}); err == nil {
		t.Fatal("duplicate span silently overwrote its predecessor")
	}
}

func TestMatchingCausalityModes(t *testing.T) {
	config := spec.DefaultValidationConfig()
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "parent", Call: "svc.A"}, {Step: "child", Call: "svc.B"}}}
	tr := &trace.Trace{Spans: []trace.Span{
		{Service: "svc", Name: "A", StartNanos: 1e9, EndNanos: 4e9, Attributes: map[string]any{"otlp.span_id": "A"}},
		{Service: "svc", Name: "B", StartNanos: 2e9, EndNanos: 3e9, Attributes: map[string]any{"otlp.span_id": "B", "otlp.parent_span_id": "A"}},
	}}
	config.Causality = string(CausalityStrict)
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); !passed {
		t.Fatalf("valid strict dependency failed: %+v", results)
	}
	tr.Spans[1].Attributes["otlp.parent_span_id"] = "different-parent"
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); passed {
		t.Fatalf("wrong strict parent passed: %+v", results)
	}
	config.Causality = string(CausalityTemporal)
	tr.Spans[1].StartNanos, tr.Spans[1].EndNanos = 5e9, 6e9
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); !passed {
		t.Fatalf("valid temporal order failed: %+v", results)
	}
	tr.Spans[1].StartNanos, tr.Spans[1].EndNanos = 0, 1e8
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); passed {
		t.Fatalf("reversed order passed temporal mode: %+v", results)
	}
	config.Causality = string(CausalityOff)
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); !passed {
		t.Fatalf("explicitly disabled order still enforced: %+v", results)
	}
	tr.Spans = tr.Spans[:1]
	if results, passed := ValidateAgainstTrace(flow, nil, tr, config); passed {
		t.Fatalf("missing call passed off mode: %+v", results)
	}
}

func TestMatchingNamingRulesAgreeAcrossFormats(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "A", Call: "svc.operation"}}}
	graph := &spec.FlowSpec{Graph: &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "A", Call: "svc.operation"}}}}
	tr := &trace.Trace{Spans: []trace.Span{{Service: " SVC ", Name: " Operation ", StartNanos: 1, EndNanos: 2}}}
	for _, contract := range []*spec.FlowSpec{flow, graph} {
		if results, passed := ValidateAgainstTrace(contract, nil, tr, spec.DefaultValidationConfig()); !passed {
			t.Fatalf("format-specific naming rule: %+v", results)
		}
	}
}

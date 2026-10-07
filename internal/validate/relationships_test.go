// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"testing"
)

func TestExplicitSpanRelationships(t *testing.T) {
	for _, mode := range []string{"temporal", "strict", "off"} {
		for _, relationship := range []string{"parent", "follows", "concurrent"} {
			t.Run(mode+"/"+relationship, func(t *testing.T) {
				flow := &spec.FlowSpec{Graph: &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "a", Call: "svc.a"}, {ID: "b", Call: "svc.b"}}, Edges: []spec.GraphEdge{{From: "a", To: "b", Relationship: relationship}}}}
				spans := []trace.Span{
					{Service: "svc", Name: "a", StartNanos: 1e9, EndNanos: 2e9, Attributes: map[string]any{"otlp.span_id": "a"}},
					{Service: "svc", Name: "b", StartNanos: 1.1e9, EndNanos: 1.2e9, Attributes: map[string]any{"otlp.span_id": "b", "otlp.parent_span_id": "a"}},
				}
				if relationship == "follows" {
					spans[1].StartNanos, spans[1].EndNanos = 2.1e9, 2.2e9
					delete(spans[1].Attributes, "otlp.parent_span_id")
				}
				config := spec.DefaultValidationConfig()
				config.Causality = mode
				config.Semantic = false
				if results, ok := ValidateAgainstTrace(flow, nil, &trace.Trace{Spans: spans}, config); !ok {
					t.Fatalf("valid explicit relationship rejected: %+v", results)
				}
				switch relationship {
				case "parent":
					delete(spans[1].Attributes, "otlp.parent_span_id")
				case "follows":
					spans[1].StartNanos, spans[1].EndNanos = 1.1e9, 1.2e9
				case "concurrent":
					delete(spans[1].Attributes, "otlp.parent_span_id")
					spans[1].StartNanos, spans[1].EndNanos = 2.1e9, 2.2e9
				}
				_, ok := ValidateAgainstTrace(flow, nil, &trace.Trace{Spans: spans}, config)
				want := mode == "off" && relationship != "concurrent"
				if ok != want {
					t.Fatalf("invalid relationship passed=%v, want %v", ok, want)
				}
			})
		}
	}
}

func TestConcurrentEdgesRequireIntervalsEvenWhenCausalityOff(t *testing.T) {
	flow := &spec.FlowSpec{Graph: &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "a", Call: "svc.a"}, {ID: "b", Call: "svc.b"}}, Edges: []spec.GraphEdge{{From: "a", To: "b", Relationship: "concurrent"}}}}
	tr, err := trace.Parse([]byte(`{"spans":[{"service":"svc","name":"a"},{"service":"svc","name":"b"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	config := spec.DefaultValidationConfig()
	config.Causality = "off"
	config.Semantic = false
	if _, ok := ValidateAgainstTrace(flow, nil, tr, config); ok {
		t.Fatal("missing intervals proved concurrency")
	}
}

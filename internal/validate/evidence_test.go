// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

func TestEvidenceObservedRequestCannotBeReplacedByDeclaration(t *testing.T) {
	for _, graph := range []bool{false, true} {
		flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "pay", Call: "svc.pay", Input: map[string]any{"id": "ORD-A"}}}}
		if graph {
			flow.Flow = nil
			flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "pay", Call: "svc.pay", Input: map[string]any{"id": "ORD-A"}}}}
		}
		ops := map[string]map[string]spec.ServiceOperation{"svc": {"pay": {Preconditions: map[string]string{"same-order": "request.body.id == expected.body.id"}}}}
		for _, tc := range []struct {
			name       string
			attributes map[string]any
			issue      verdict.IssueKind
		}{
			{"correct", map[string]any{"request.body": map[string]any{"id": "ORD-A"}}, ""},
			{"wrong", map[string]any{"request.body": map[string]any{"id": "ORD-B"}}, verdict.RuleViolation},
			{"missing-body", map[string]any{}, verdict.MissingEvidence},
			{"missing-field", map[string]any{"request.body": map[string]any{}}, verdict.MissingEvidence},
		} {
			sp := trace.Span{Service: "svc", Name: "pay", StartNanos: 1, EndNanos: 2, Attributes: tc.attributes}
			sp.Attributes["otlp.span_id"] = "exact-call"
			results, passed := ValidateAgainstTrace(flow, ops, &trace.Trace{Spans: []trace.Span{sp}}, spec.DefaultValidationConfig())
			if len(results) != 1 || passed != (tc.issue == "") || results[0].Issue != tc.issue || results[0].Evidence == nil || results[0].Evidence.Key != "exact-call" {
				t.Fatalf("graph=%t %s: %+v passed=%t", graph, tc.name, results, passed)
			}
			if tc.name == "correct" || tc.name == "wrong" {
				found := false
				for _, field := range results[0].Bindings {
					if field.Variable == "request.body" && field.Attribute == "request.body" && field.Span.Key == "exact-call" {
						found = true
					}
				}
				if !found {
					t.Fatal("observed request has no captured source")
				}
			}
		}
	}
}

func TestEvidenceMissingFieldsAreNotSyntheticValues(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "a", Call: "svc.a"}}}
	tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "a", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"id": 42}}}}
	for _, expr := range []string{"response.status == 0", "response.body.id == 42"} {
		ops := map[string]map[string]spec.ServiceOperation{"svc": {"a": {Postconditions: map[string]string{"required": expr}}}}
		results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig())
		if passed || results[0].Issue != verdict.MissingEvidence {
			t.Fatalf("synthetic observation passed: %s %+v", expr, results)
		}
	}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"a": {Postconditions: map[string]string{"optional": "!has(response.body) && span.attributes.id == 42"}}}}
	if results, passed := ValidateAgainstTrace(flow, ops, tr, spec.DefaultValidationConfig()); !passed {
		t.Fatalf("explicit optional rule failed: %+v", results)
	}
}

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"sync"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

func TestStructuralMatchersDoNotEvaluateOrExport(t *testing.T) {
	sp := trace.Span{Service: "svc", Name: "a", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"response.status": 500}}
	calls, err := BuildCallGraph([]trace.Span{sp})
	if err != nil {
		t.Fatal(err)
	}
	for _, graph := range []bool{false, true} {
		// Invalid CEL cannot affect a structural matcher; qualification and
		// evaluation belong to separate execution boundaries.
		flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "a", Call: "svc.a", Output: map[string]string{"x": "invalid CEL !!!"}}}}
		var matches []matchedStep
		if graph {
			flow.Flow = nil
			flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "a", Call: "svc.a", Output: map[string]string{"x": "invalid CEL !!!"}}}}
			matches = matchGraphSteps(flow, calls, spec.DefaultValidationConfig(), context.Background())
		} else {
			matches = matchFlowSteps(flow, calls, spec.DefaultValidationConfig())
		}
		if len(matches) != 1 || matches[0].result.Status != "PASS" || len(matches[0].result.Conditions) != 0 || matches[0].node.Span.Attributes["response.status"] != 500 {
			t.Fatalf("graph=%t: matcher crossed execution boundary: %+v", graph, matches)
		}
		if _, err := CompilePlan(flow, nil, spec.DefaultValidationConfig()); err == nil {
			t.Fatal("qualification did not reject invalid output")
		}
	}
}

func TestSharedEvaluationKeepsDistinctInvocationsIsolated(t *testing.T) {
	for _, graph := range []bool{false, true} {
		flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "a", Call: "svc.a", Input: map[string]any{"id": "A"}}}}
		if graph {
			flow.Flow = nil
			flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "a", Call: "svc.a", Input: map[string]any{"id": "A"}}}}
		}
		ops := map[string]map[string]spec.ServiceOperation{"svc": {"a": {Preconditions: map[string]string{"id": "request.body.id == expected.body.id"}}}}
		plan, err := CompilePlan(flow, ops, spec.DefaultValidationConfig())
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 40; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				id := "A"
				want := true
				if i%2 != 0 {
					id = "B"
					want = false
				}
				tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "a", StartNanos: 1, EndNanos: 2, Attributes: map[string]any{"otlp.span_id": id, "request.body": map[string]any{"id": id}}}}}
				results, passed := plan.ValidateTrace(tr)
				if passed != want || len(results) != 1 || results[0].Evidence == nil || results[0].Evidence.Key != id {
					t.Errorf("graph=%t invocation %d leaked state: %+v", graph, i, results)
				}
			}(i)
		}
		wg.Wait()
	}
}

func TestContractFailureHasPreparationIdentity(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "a", Call: "svc.a"}}}
	ops := map[string]map[string]spec.ServiceOperation{"svc": {"a": {Postconditions: map[string]string{"broken": "response.status =="}}}}
	_, results, passed := ValidateWithPlan(flow, ops, nil, spec.DefaultValidationConfig())
	if passed || len(results) != 1 || results[0].Issue != verdict.InvalidContract || results[0].Evidence != nil {
		t.Fatalf("invalid contract disguised as runtime observation: %+v", results)
	}
}

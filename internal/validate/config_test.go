// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"io"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestValidationConfigConcurrentIsolation(t *testing.T) {
	for _, dag := range []bool{false, true} {
		flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "A", Call: "svc.A"}, {Step: "B", Call: "svc.B"}}}
		if dag {
			flow.Flow = nil
			flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "A", Call: "svc.A"}, {ID: "B", Call: "svc.B", Depends: []string{"A"}}}}
		}
		tr := &trace.Trace{Spans: []trace.Span{{Service: "svc", Name: "A", StartNanos: 0, EndNanos: 100e6}, {Service: "svc", Name: "B", StartNanos: 90e6, EndNanos: 200e6}}}
		ops := map[string]map[string]spec.ServiceOperation{"svc": {"A": {Postconditions: map[string]string{"ok": "true"}}, "B": {Postconditions: map[string]string{"false": "false"}}}}
		cases := []struct {
			config spec.ValidationConfig
			passed bool
		}{
			{spec.ValidationConfig{Semantic: false, Causality: "temporal", ToleranceMs: 0}, false},
			{spec.ValidationConfig{Semantic: false, Causality: "temporal", ToleranceMs: 50}, true},
			{spec.ValidationConfig{Semantic: true, Causality: "temporal", ToleranceMs: 50}, false},
			{spec.ValidationConfig{Semantic: false, Causality: "strict", ToleranceMs: 50}, false},
			{spec.ValidationConfig{Semantic: false, Causality: "off", ToleranceMs: 0}, true},
		}
		var wg sync.WaitGroup
		for repeat := 0; repeat < 16; repeat++ {
			for _, tc := range cases {
				wg.Add(1)
				go func() {
					defer wg.Done()
					results, passed := ValidateAgainstTrace(flow, ops, tr, tc.config)
					if passed != tc.passed {
						t.Errorf("dag=%t config=%+v: %+v", dag, tc.config, results)
					}
				}()
			}
		}
		wg.Wait()
		if dag && (len(flow.Graph.Edges) != 0) {
			t.Fatal("validation modified shared graph")
		}
	}
}

func TestValidationReturnsDiagnosticsWithoutStdout(t *testing.T) {
	flow := &spec.FlowSpec{Flow: []spec.FlowStep{{Step: "A", Call: "svc.A"}, {Step: "B", Call: "svc.B"}}}
	tr := &trace.Trace{Spans: []trace.Span{
		{Service: "svc", Name: "A", StartNanos: 1e9, EndNanos: 2e9, Attributes: map[string]any{"otlp.span_id": "A"}},
		{Service: "svc", Name: "B", StartNanos: 3e9, EndNanos: 4e9, Attributes: map[string]any{"otlp.span_id": "B", "otlp.parent_span_id": "A"}},
	}}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original; _ = r.Close(); _ = w.Close() })
	results, passed := ValidateAgainstTrace(flow, nil, tr, spec.DefaultValidationConfig())
	os.Stdout = original
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("engine wrote stdout: %s", data)
	}
	if passed || len(results) == 0 || len(results[0].Violations) == 0 {
		t.Fatalf("diagnostic was discarded: %+v", results)
	}
	before := reflect.ValueOf(spec.DefaultValidationConfig()).Interface()
	config := spec.DefaultValidationConfig()
	config.Causality = "invalid"
	if _, passed := ValidateAgainstTrace(flow, nil, tr, config); passed {
		t.Fatal("invalid configuration accepted")
	}
	if !reflect.DeepEqual(before, spec.DefaultValidationConfig()) {
		t.Fatal("default configuration changed")
	}
}

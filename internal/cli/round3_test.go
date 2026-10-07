// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestRound3DuplicateOperationsRejected(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		dir, _, spans := correctnessFixture(t, []string{"A"}, "true")
		spans[0].Attributes["response.status"] = 500
		correctnessTrace(t, dir, spans)
		ops := []spec.ServiceOperation{{OperationId: "A", Postconditions: map[string]string{"status": "response.status == 200"}}, {OperationId: "A", Postconditions: map[string]string{"status": "true"}}}
		if reverse {
			ops[0], ops[1] = ops[1], ops[0]
		}
		correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: ops})
		for _, args := range [][]string{
			{"lint", "--flow", "flow.yaml"},
			{"validate", "--flow", "flow.yaml", "--trace", "trace.json"},
			{"ci-gate", "--flow", "flow.yaml", "--trace", "trace.json"},
			{"baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json"},
		} {
			correctnessCommand(t, dir, 2, args...)
		}
	}
}

func TestRound3HTTPDiscoveryValidatesOwnTrace(t *testing.T) {
	for _, attrs := range []map[string]any{
		{"http.method": "GET", "http.route": "/health", "http.status_code": 200},
		{"http.method": "GET", "http.target": "/health?probe=yes", "http.status_code": 200},
		{"http.request.method": "GET", "url.path": "/health", "http.response.status_code": 200},
		{"http.status_code": 200},
		{"http.method": "POST", "http.status_code": 200},
		{"http.route": "/ready", "http.status_code": 200},
	} {
		dir := t.TempDir()
		correctnessTrace(t, dir, []trace.Span{{Service: "svc", Name: "GET /health", StartNanos: 1e9, EndNanos: 1.1e9, Attributes: attrs}})
		correctnessCommand(t, dir, 0, "discover", "--trace", "trace.json", "--out", "flow.yaml", "--out-services", "services")
		correctnessCommand(t, dir, 0, "lint", "--flow", "flow.yaml")
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		flow, err := spec.LoadFlowSpec(filepath.Join(dir, "flow.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		node := flow.CallSteps()[0]
		flow.Flow = nil
		flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "http", Call: node.Call, Input: node.Input, Output: node.Output}}}
		correctnessWrite(t, filepath.Join(dir, "graph.yaml"), flow)
		correctnessCommand(t, dir, 0, "lint", "--flow", "graph.yaml")
		correctnessCommand(t, dir, 0, "validate", "--flow", "graph.yaml", "--trace", "trace.json")
		// Matching must follow HTTP metadata even when CEL is disabled and the
		// raw span name remains the same.
		correctnessTrace(t, dir, []trace.Span{{Service: "svc", Name: "GET /health", StartNanos: 1e9, EndNanos: 1.1e9, Attributes: map[string]any{"http.method": "DELETE", "http.route": "/other"}}})
		for _, path := range []string{"flow.yaml", "graph.yaml"} {
			correctnessCommand(t, dir, 3, "validate", "--flow", path, "--trace", "trace.json", "--semantic=false", "--threshold-conds=0")
		}
	}
}

func TestRound3DiscoveryRejectsOperationCollision(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		dir := t.TempDir()
		spans := []trace.Span{
			{Service: "svc", Name: "GET /a-b", StartNanos: 1, EndNanos: 2},
			{Service: "svc", Name: "GET /a/b", StartNanos: 3, EndNanos: 4},
		}
		if reverse {
			spans[0], spans[1] = spans[1], spans[0]
		}
		correctnessTrace(t, dir, spans)
		flowPath := filepath.Join(dir, "flow.yaml")
		if err := os.WriteFile(flowPath, []byte("existing"), 0o644); err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 2, "discover", "--trace", "trace.json", "--out", "flow.yaml", "--out-services", "services")
		data, err := os.ReadFile(flowPath)
		if err != nil || string(data) != "existing" {
			t.Fatalf("collision changed existing output: %s, %v", data, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "services", "svc.servicespec.yaml")); !os.IsNotExist(err) {
			t.Fatal("ambiguous identity produced ServiceSpec")
		}
	}
}

func TestRound3CrossStepOutputs(t *testing.T) {
	for _, graph := range []bool{false, true} {
		t.Run(map[bool]string{false: "flow", true: "dag"}[graph], func(t *testing.T) {
			dir, flow, spans := correctnessFixture(t, []string{"A", "B"}, "true")
			flow.Flow[0].Output = map[string]string{"created": "response.body"}
			flow.Flow[1].Input = map[string]any{"id": "${created.id}"}
			if graph {
				correctnessGraph(flow, []string{"A", "B"}, [][2]string{{"A", "B"}})
				flow.Graph.Nodes[0].Output = map[string]string{"created": "response.body"}
				flow.Graph.Nodes[1].Input = map[string]any{"id": "${created.id}"}
			}
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
			spans[0].Attributes["response.body"] = map[string]any{"id": 42}
			correctnessTrace(t, dir, spans)
			correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{
				{OperationId: "A", Postconditions: map[string]string{"ok": "true"}},
				{OperationId: "B", Preconditions: map[string]string{"id": "request.body.id == 42 && vars.created.id == 42"}, Postconditions: map[string]string{"ok": "true"}},
			}})
			correctnessCommand(t, dir, 0, "lint", "--flow", "flow.yaml")
			correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			spans[0].Attributes["response.body"] = map[string]any{}
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
	}
}

func TestRound3SchemaConsistentAtEntries(t *testing.T) {
	for _, invalid := range []string{"title", "service"} {
		dir, flow, _ := correctnessFixture(t, []string{"A"}, "true")
		if invalid == "title" {
			flow.Info.Title = ""
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
		} else {
			correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "", Operations: []spec.ServiceOperation{{OperationId: "A", Postconditions: map[string]string{"ok": "true"}}}})
		}
		for _, args := range [][]string{
			{"lint", "--flow", "flow.yaml"},
			{"validate", "--flow", "flow.yaml", "--trace", "trace.json"},
			{"spec", "validate", "--flow", "flow.yaml"},
			{"run", "validate", "--flow", "flow.yaml", "--trace", "trace.json"},
			{"ci-gate", "--flow", "flow.yaml", "--trace", "trace.json"},
			{"baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json"},
		} {
			correctnessCommand(t, dir, 2, args...)
		}
		if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !os.IsNotExist(err) {
			t.Fatal("invalid contract produced baseline")
		}
	}
}

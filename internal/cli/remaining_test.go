// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

// These files are copied byte-for-byte from the independent d95eabe recheck.
func remainingFixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range []string{"flow.yaml", "svc.yaml", "trace.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", "recheck-d95eabe", name, file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRemainingOriginalFalsePasses(t *testing.T) {
	for _, name := range []string{"remaining-mixed-dependencies", "remaining-precision-flow-large_mismatch", "remaining-precision-dag-large_mismatch"} {
		t.Run(name, func(t *testing.T) {
			dir := remainingFixture(t, name)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			correctnessCommand(t, dir, 3, "ci-gate", "--flow", "flow.yaml", "--trace", "trace.json")
			correctnessCommand(t, dir, 3, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
			if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !os.IsNotExist(err) {
				t.Fatal("false pass was recorded as a baseline")
			}
		})
	}
}

func TestRemainingMixedDependencies(t *testing.T) {
	t.Run("ordered-and-off-controls", func(t *testing.T) {
		dir := remainingFixture(t, "remaining-mixed-dependencies")
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", "off")
		tr, err := trace.LoadFromFile(filepath.Join(dir, "trace.json"))
		if err != nil {
			t.Fatal(err)
		}
		for i := range tr.Spans {
			switch tr.Spans[i].Name {
			case "a":
				tr.Spans[i].StartNanos, tr.Spans[i].EndNanos = 1e9, 1.1e9
			case "b":
				tr.Spans[i].StartNanos, tr.Spans[i].EndNanos = 2e9, 2.1e9
			}
		}
		correctnessTrace(t, dir, tr.Spans)
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
	})
	for _, kind := range []string{"missing-node", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			dir := remainingFixture(t, "remaining-mixed-dependencies")
			flow, err := spec.LoadFlowSpec(filepath.Join(dir, "flow.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if kind == "missing-node" {
				flow.Graph.Nodes[1].Depends = append(flow.Graph.Nodes[1].Depends, "missing")
			} else {
				flow.Graph.Nodes[0].Depends = []string{"B"}
			}
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
			correctnessCommand(t, dir, 2, "lint", "--flow", "flow.yaml")
			correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
	}
	t.Run("dataflow-and-conversion", func(t *testing.T) {
		dir, flow, _ := correctnessFixture(t, []string{"A", "B", "C"}, "true")
		flow.Flow = nil
		flow.Graph = &spec.GraphSpec{
			Nodes: []spec.GraphNode{
				{ID: "A", Call: "svc.A", Output: map[string]string{"id": "42"}},
				{ID: "B", Call: "svc.B", Depends: []string{"A"}, Input: map[string]any{"id": "${id}"}},
				{ID: "C", Call: "svc.C", Depends: []string{"B"}},
			},
			Edges: []spec.GraphEdge{{From: "B", To: "C"}},
		}
		correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
		correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{
			{OperationId: "A", Postconditions: map[string]string{"ok": "true"}},
			{OperationId: "B", Postconditions: map[string]string{"id": "request.body.id == 42 && vars.id == 42"}},
			{OperationId: "C", Postconditions: map[string]string{"ancestor": "vars.id == 42"}},
		}})
		correctnessCommand(t, dir, 0, "lint", "--flow", "flow.yaml")
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		correctnessCommand(t, dir, 0, "spec", "convert", "--in", "flow.yaml", "--out", "converted.yaml")
		if _, err := os.Stat(filepath.Join(dir, "converted.yaml")); err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 0, "validate", "--flow", "converted.yaml", "--trace", "trace.json")
	})
}

func TestRemainingTypedNumberDataflowAndReports(t *testing.T) {
	for _, mode := range []string{"flow", "dag"} {
		t.Run(mode, func(t *testing.T) {
			dir := remainingFixture(t, "remaining-precision-"+mode+"-large_mismatch")
			path := filepath.Join(dir, "trace.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data = bytes.ReplaceAll(data, []byte("9007199254740992"), []byte("9007199254740993"))
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			flow, err := spec.LoadFlowSpec(filepath.Join(dir, "flow.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			output := map[string]string{"identifier": "response.body.id", "payload": "response.body", "ids": "[response.body.id]"}
			input := map[string]any{"id": "${identifier}", "ids": "${ids}", "text": "id-${identifier}"}
			if flow.Graph != nil {
				flow.Graph.Nodes[0].Output, flow.Graph.Nodes[1].Input = output, input
			} else {
				flow.Flow[0].Output, flow.Flow[1].Input = output, input
			}
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
			correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{
				{OperationId: "a", Postconditions: map[string]string{"ok": "true"}},
				{OperationId: "b", Postconditions: map[string]string{"exact": `vars.identifier == response.body.id && vars.payload.id == response.body.id && vars.ids[0] == response.body.id && request.body.id == response.body.id && request.body.ids[0] == response.body.id && request.body.text == "id-9007199254740993"`}},
			}})
			correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
			correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--report-format", "json", "--report-out", "report.json")
			reportData, err := os.ReadFile(filepath.Join(dir, "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			var report struct {
				Success bool `json:"success"`
				Inputs  struct {
					TraceHash string `json:"traceHash"`
				} `json:"inputs"`
			}
			if err := json.Unmarshal(reportData, &report); err != nil {
				t.Fatal(err)
			}
			if !report.Success || report.Inputs.TraceHash != snapshotHash(t, path) {
				t.Fatal("exact numeric control lost report outcome or captured input identity")
			}
		})
	}
	t.Run("double-arithmetic", func(t *testing.T) {
		dir, _, _ := correctnessFixture(t, []string{"A"}, "type(response.status) == double && response.status / 2.0 == 1.5")
		if err := os.WriteFile(filepath.Join(dir, "trace.json"), []byte(`{"spans":[{"name":"A","service":"svc","startNanos":1,"endNanos":2,"attributes":{"response.status":3.0}}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
	})
}

func TestRemainingExactNumbers(t *testing.T) {
	for _, graph := range []bool{false, true} {
		for _, tc := range []struct {
			name, first, second string
			want                int
		}{
			{"large-equal", "9007199254740993", "9007199254740993", 0},
			{"large-unequal", "9007199254740992", "9007199254740993", 3},
			{"negative-unequal", "-9007199254740992", "-9007199254740993", 3},
			{"uint64-unequal", "18446744073709551614", "18446744073709551615", 3},
			{"uint64-equal", "18446744073709551615", "18446744073709551615", 0},
			{"scientific-precision-loss", "9.007199254740992e15", "9.007199254740993e15", 2},
			{"decimal-precision-loss", "9007199254740992.0", "9007199254740993.0", 2},
			{"fraction-equal", "1.25", "1.25", 0},
			{"overflow", "18446744073709551616", "18446744073709551616", 2},
		} {
			mode := "flow"
			if graph {
				mode = "dag"
			}
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				dir := remainingFixture(t, "remaining-precision-"+mode+"-large_mismatch")
				path := filepath.Join(dir, "trace.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data = bytes.ReplaceAll(data, []byte("9007199254740992"), []byte(tc.first))
				data = bytes.ReplaceAll(data, []byte("9007199254740993"), []byte(tc.second))
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
				correctnessCommand(t, dir, tc.want, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			})
		}
	}
}

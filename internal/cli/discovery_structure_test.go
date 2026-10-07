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

func discoverySpan(name, id, parent string, start, end int64) trace.Span {
	return trace.Span{Service: "svc", Name: name, StartNanos: start, EndNanos: end, Attributes: map[string]any{"otlp.trace_id": "one-request", "otlp.span_id": id, "otlp.parent_span_id": parent, "http.status_code": 200}}
}

func TestDiscoverySourceRoundTrip(t *testing.T) {
	fixtures := map[string][]trace.Span{
		"serial":             {discoverySpan("a", "a", "", 1e9, 1.1e9), discoverySpan("b", "b", "", 1.2e9, 1.3e9)},
		"nested":             {discoverySpan("a", "a", "", 1e9, 2e9), discoverySpan("b", "b", "a", 1.1e9, 1.2e9)},
		"overlap":            {discoverySpan("a", "a", "", 1e9, 2e9), discoverySpan("b", "b", "", 1.1e9, 1.9e9)},
		"branch":             {discoverySpan("a", "a", "", 1e9, 2e9), discoverySpan("b", "b", "a", 1.1e9, 1.2e9), discoverySpan("c", "c", "a", 1.3e9, 1.4e9)},
		"repeat":             {discoverySpan("a", "a", "", 1e9, 2e9), discoverySpan("repeat", "b", "a", 1.1e9, 1.2e9), discoverySpan("repeat", "c", "a", 1.3e9, 1.4e9)},
		"equal-start-repeat": {discoverySpan("repeat", "long", "", 1e9, 2e9), discoverySpan("repeat", "short", "", 1e9, 1.9e9)},
		"overlapping-repeat": {discoverySpan("repeat", "a", "", 1e9, 2e9), discoverySpan("repeat", "b", "", 1.1e9, 1.9e9)},
	}
	for name, spans := range fixtures {
		for _, entry := range []string{"init", "discover"} {
			t.Run(name+"/"+entry, func(t *testing.T) {
				dir := t.TempDir()
				correctnessTrace(t, dir, spans)
				flowPath := "generated/.flowspec.yaml"
				if entry == "init" {
					correctnessCommand(t, dir, 0, "init", "--mode", "trace", "--trace", "trace.json", "--out", "generated", "--ci", "none", "--yes")
				} else {
					correctnessCommand(t, dir, 0, "discover", "--trace", "trace.json", "--out", flowPath, "--out-services", "generated/services")
				}
				correctnessCommand(t, dir, 0, "lint", "--flow", flowPath)
				for _, mode := range []string{"temporal", "strict"} {
					correctnessCommand(t, dir, 0, "validate", "--flow", flowPath, "--trace", "trace.json", "--causality", mode)
				}
				flow, err := spec.LoadFlowSpec(filepath.Join(dir, flowPath))
				if err != nil {
					t.Fatal(err)
				}
				if flow.GetStepsCount() != len(spans) {
					t.Fatal("generation dropped call instances")
				}
				// File order is not call identity; equal-start calls must be deterministic.
				shuffled := append([]trace.Span(nil), spans...)
				for i, j := 0, len(shuffled)-1; i < j; i, j = i+1, j-1 {
					shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
				}
				correctnessTrace(t, dir, shuffled)
				correctnessCommand(t, dir, 0, "validate", "--flow", flowPath, "--trace", "trace.json")
				// Changing observations into an actual failed operation must still fail.
				failed := append([]trace.Span(nil), spans...)
				failed[0].Attributes = map[string]any{}
				for k, v := range spans[0].Attributes {
					failed[0].Attributes[k] = v
				}
				failed[0].Attributes["http.status_code"] = 500
				correctnessTrace(t, dir, failed)
				correctnessCommand(t, dir, 3, "validate", "--flow", flowPath, "--trace", "trace.json")
			})
		}
	}
}

func TestDiscoveryRejectsInvalidStructureBeforeReplacement(t *testing.T) {
	for _, entry := range []string{"init", "discover"} {
		t.Run(entry, func(t *testing.T) {
			dir := t.TempDir()
			spans := []trace.Span{discoverySpan("a", "same", "", 1e9, 2e9), discoverySpan("b", "same", "", 1.1e9, 1.2e9)}
			correctnessTrace(t, dir, spans)
			target := filepath.Join(dir, "generated", ".flowspec.yaml")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
				t.Fatal(err)
			}
			if entry == "init" {
				correctnessCommand(t, dir, 2, "init", "--mode", "trace", "--trace", "trace.json", "--out", "generated", "--force", "--ci", "none", "--yes")
			} else {
				correctnessCommand(t, dir, 2, "discover", "--trace", "trace.json", "--out", "generated/.flowspec.yaml", "--out-services", "generated/services", "--no-validate")
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "original" {
				t.Fatal("invalid input replaced a contract")
			}
		})
	}
}

func TestDiscoveryPreservesParentsInsteadOfOnlyAcceptingSource(t *testing.T) {
	dir := t.TempDir()
	spans := []trace.Span{discoverySpan("a", "a", "", 1e9, 2e9), discoverySpan("b", "b", "a", 1.1e9, 1.2e9)}
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 0, "discover", "--trace", "trace.json", "--out", "flow.yaml", "--out-services", "services")
	spans[1].Attributes["otlp.parent_span_id"] = "another-parent"
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--semantic=false", "--threshold-conds=0")
}

func TestTypedStructuralEdgesDoNotSupplyCompletedOutputs(t *testing.T) {
	for _, relationship := range []string{"parent", "concurrent"} {
		dir, flow, _ := correctnessFixture(t, []string{"a", "b"}, "true")
		flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "a", Call: "svc.a", Output: map[string]string{"value": "response.body"}}, {ID: "b", Call: "svc.b", Input: map[string]any{"value": "${value}"}}}, Edges: []spec.GraphEdge{{From: "a", To: "b", Relationship: relationship}}}
		flow.Flow = nil
		correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
		correctnessCommand(t, dir, 2, "lint", "--flow", "flow.yaml")
	}
}

func TestDiscoveryNormalRetryPreservesEveryAttempt(t *testing.T) {
	for _, entry := range []string{"init", "discover"} {
		dir := t.TempDir()
		spans := []trace.Span{discoverySpan("GET /lookup", "first", "", 1e9, 1.1e9), discoverySpan("GET /lookup", "second", "", 1.2e9, 1.3e9)}
		for _, s := range spans {
			s.Attributes["http.method"] = "GET"
		}
		spans[0].Attributes["http.status_code"] = 500
		correctnessTrace(t, dir, spans)
		if entry == "init" {
			correctnessCommand(t, dir, 0, "init", "--mode", "trace", "--trace", "trace.json", "--out", "generated", "--ci", "none", "--yes")
		} else {
			correctnessCommand(t, dir, 0, "discover", "--trace", "trace.json", "--out", "generated/.flowspec.yaml", "--out-services", "generated/services")
		}
		correctnessCommand(t, dir, 0, "validate", "--flow", "generated/.flowspec.yaml", "--trace", "trace.json")
		flow, err := spec.LoadFlowSpec(filepath.Join(dir, "generated/.flowspec.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if flow.GetStepsCount() != 2 {
			t.Fatal("failed attempt was removed")
		}
		// The actual failed attempt remains evidence; an explicit success policy
		// must reject it, even though discovery did not invent such a policy.
		path := filepath.Join(dir, "generated/services/svc.servicespec.yaml")
		ss, err := spec.LoadServiceSpec(path)
		if err != nil {
			t.Fatal(err)
		}
		ss.Operations[0].Postconditions = map[string]string{"require_success": "response.status == 200"}
		correctnessWrite(t, path, ss)
		correctnessCommand(t, dir, 3, "validate", "--flow", "generated/.flowspec.yaml", "--trace", "trace.json")
	}
}

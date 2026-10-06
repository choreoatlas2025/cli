// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestRecheckCompletionDependency(t *testing.T) {
	for _, graph := range []bool{false, true} {
		t.Run(map[bool]string{false: "flow", true: "dag"}[graph], func(t *testing.T) {
			dir, flow, spans := correctnessFixture(t, []string{"A", "B"}, "true")
			if graph {
				correctnessGraph(flow, []string{"A", "B"}, [][2]string{{"A", "B"}})
				correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
			}
			spans[0].EndNanos = spans[1].StartNanos + 100e6
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			spans[0].EndNanos = spans[1].StartNanos + 20e6
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality-tolerance", "0")
		})
	}
}

func TestRecheckParallelJoinWaitsForEveryPredecessor(t *testing.T) {
	for _, graph := range []bool{false, true} {
		t.Run(map[bool]string{false: "flow", true: "dag"}[graph], func(t *testing.T) {
			dir, flow, spans := correctnessFixture(t, []string{"A", "B", "C"}, "true")
			if graph {
				correctnessGraph(flow, []string{"A", "B", "C"}, [][2]string{{"A", "C"}, {"B", "C"}})
			} else {
				flow.Flow = []spec.FlowStep{{Step: "parallel", Parallel: flow.Flow[:2]}, flow.Flow[2]}
			}
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
			spans[0].EndNanos = 5e9
			spans[1].EndNanos = 3e9
			spans[2].StartNanos, spans[2].EndNanos = 4e9, 4.1e9
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
			spans[2].StartNanos, spans[2].EndNanos = 5e9, 5.1e9
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
	}
}

func TestRecheckUnevaluatedResultsFailJUnit(t *testing.T) {
	for _, steps := range [][]validate.StepResult{
		nil,
		{{Step: "A", Call: "svc.A", Status: "PASS", Conditions: []validate.ConditionResult{{Status: "SKIP"}}}},
	} {
		path := filepath.Join(t.TempDir(), "report.xml")
		if err := WriteReport(path, ReportJUnit, steps, nil, nil); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var suite struct {
			Failures int `xml:"failures,attr"`
		}
		if err := xml.Unmarshal(b, &suite); err != nil || suite.Failures == 0 {
			t.Fatalf("unevaluated result passed JUnit: %s, %v", b, err)
		}
	}
}

func TestRecheckTraceIdentity(t *testing.T) {
	for _, mode := range []string{"temporal", "strict", "off"} {
		t.Run(mode, func(t *testing.T) {
			dir, _, spans := correctnessFixture(t, []string{"A", "B"}, "true")
			spans[0].Attributes["otlp.trace_id"] = "first-trace"
			spans[0].Attributes["otlp.span_id"] = "parent"
			spans[1].Attributes["otlp.trace_id"] = "second-trace"
			spans[1].Attributes["otlp.parent_span_id"] = "parent"
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", mode)
			delete(spans[1].Attributes, "otlp.trace_id")
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", mode)
		})
	}
}

func TestRecheckCELErrorsFail(t *testing.T) {
	for _, expr := range []string{"response.status ==", "42", "response.body.missing == true"} {
		t.Run(expr, func(t *testing.T) {
			dir, _, _ := correctnessFixture(t, []string{"A"}, expr)
			service := spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{{OperationId: "A", Postconditions: map[string]string{"healthy": "true", "error": expr}}}}
			correctnessWrite(t, filepath.Join(dir, "svc.yaml"), service)
			for _, format := range []string{"json", "junit", "html"} {
				correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", format, "--report-out", "report."+format)
				b, err := os.ReadFile(filepath.Join(dir, "report."+format))
				if err != nil {
					t.Fatal(err)
				}
				if format == "junit" {
					var suite struct {
						Failures int `xml:"failures,attr"`
					}
					if err := xml.Unmarshal(b, &suite); err != nil || suite.Failures == 0 {
						t.Fatalf("CEL failure missing from JUnit: %s, %v", b, err)
					}
					continue
				}
				if format == "html" {
					_, payload, ok := strings.Cut(string(b), "window.FLOWREPORT = ")
					if !ok {
						t.Fatal("missing HTML payload")
					}
					b = []byte(strings.TrimSuffix(payload, ";</script>"))
				}
				var report struct {
					Success  bool `json:"success"`
					ExitCode int  `json:"exitCode"`
				}
				if err := json.Unmarshal(b, &report); err != nil || report.Success || report.ExitCode != 3 {
					t.Fatalf("CEL failure missing from %s report: %s, %v", format, b, err)
				}
			}
			correctnessCommand(t, dir, 3, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
		})
	}
}

func TestRecheckBaselineExplicitConfig(t *testing.T) {
	dir, _, spans := correctnessFixture(t, []string{"A", "B"}, "true")
	spans[0].EndNanos = 4e9
	spans[0].Attributes["otlp.span_id"] = "parent"
	spans[1].EndNanos = 3e9
	spans[1].Attributes["otlp.parent_span_id"] = "parent"
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json", "--causality", "strict", "--causality-tolerance", "0")
	correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--causality", "strict", "--causality-tolerance", "0")
	correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json")
}

func TestRecheckConversionSchemaBeforeWrite(t *testing.T) {
	dir, flow, _ := correctnessFixture(t, []string{"A"}, "true")
	correctnessGraph(flow, []string{"A"}, nil)
	flow.Info.Title = ""
	correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
	path := filepath.Join(dir, "converted.yaml")
	if err := os.WriteFile(path, []byte("original output"), 0o644); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 2, "spec", "convert", "--in", "flow.yaml", "--out", "converted.yaml")
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "original output" {
		t.Fatalf("invalid conversion changed destination: %s, %v", b, err)
	}
}

func TestRecheckBaselineProvenance(t *testing.T) {
	dir, flow, spans := correctnessFixture(t, []string{"A", "B"}, "true")
	for i := range spans {
		spans[i].Attributes["otlp.trace_id"] = "recorded-trace"
	}
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	b, err := os.ReadFile(filepath.Join(dir, "baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var baseline map[string]any
	if err := json.Unmarshal(b, &baseline); err != nil {
		t.Fatal(err)
	}
	provenance, ok := baseline["provenance"].(map[string]any)
	if !ok || provenance["traceHash"] == nil || provenance["validatorHash"] == nil || provenance["config"] == nil || provenance["traceIdentity"] == nil {
		t.Errorf("baseline lacks recording identity: %s", b)
	}
	// A new successful trace is the point of comparison; it need not be the recording trace.
	for i := range spans {
		spans[i].Attributes["otlp.trace_id"] = "current-trace"
	}
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json")
	correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--semantic=false")
	correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--causality-tolerance", "0")
	if ok {
		provenance["validatorHash"] = "sha256:" + strings.Repeat("0", 64)
		b, err = json.Marshal(baseline)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "baseline.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json")
	}
	flow.Info.Title = ""
	correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
	correctnessCommand(t, dir, 2, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "empty-title.json")
	if _, err := os.Stat(filepath.Join(dir, "empty-title.json")); !os.IsNotExist(err) {
		t.Fatal("recorded an unconsumable empty-title baseline")
	}
}

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"gopkg.in/yaml.v3"
)

// Run the real command router in a child process so os.Exit is observable.
func TestCorrectnessCLIProcess(t *testing.T) {
	if os.Getenv("CHOREOATLAS_CORRECTNESS_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"choreoatlas"}, os.Args[i+1:]...)
			Execute()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func correctnessCommand(t *testing.T, dir string, want int, args ...string) string {
	t.Helper()
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commandArgs := append([]string{"-test.run=^TestCorrectnessCLIProcess$", "--"}, args...)
	if external := os.Getenv("CHOREOATLAS_CORRECTNESS_BINARY"); external != "" {
		bin, commandArgs = external, args
	}
	cmd := exec.Command(bin, commandArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CHOREOATLAS_CORRECTNESS_PROCESS=1")
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("command %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	if code != want {
		t.Errorf("command %v: exit %d, want %d\n%s", args, code, want, out)
	}
	return string(out)
}

func correctnessWrite(t *testing.T, path string, value any) {
	t.Helper()
	b, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func correctnessTrace(t *testing.T, dir string, spans []trace.Span) {
	t.Helper()
	b, err := json.Marshal(trace.Trace{Spans: spans})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trace.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func correctnessFixture(t *testing.T, calls []string, expr string) (string, *spec.FlowSpec, []trace.Span) {
	t.Helper()
	dir := t.TempDir()
	f := &spec.FlowSpec{Info: spec.FlowInfo{Title: "Correctness"}, Services: map[string]spec.ServiceBinding{"svc": {Spec: "svc.yaml"}}}
	ss := spec.ServiceSpecFile{Service: "svc"}
	seen := map[string]bool{}
	var spans []trace.Span
	for i, op := range calls {
		f.Flow = append(f.Flow, spec.FlowStep{Step: fmt.Sprintf("step%d", i), Call: "svc." + op})
		if !seen[op] {
			ss.Operations = append(ss.Operations, spec.ServiceOperation{OperationId: op, Postconditions: map[string]string{"status": expr}})
			seen[op] = true
		}
		spans = append(spans, trace.Span{Service: "svc", Name: op, StartNanos: int64(i+1) * 1e9, EndNanos: int64(i+1)*1e9 + 1e8, Attributes: map[string]any{"response.status": 200}})
	}
	correctnessWrite(t, filepath.Join(dir, "flow.yaml"), f)
	correctnessWrite(t, filepath.Join(dir, "svc.yaml"), ss)
	correctnessTrace(t, dir, spans)
	return dir, f, spans
}

func TestCorrectnessResultSemanticFailure(t *testing.T) {
	calls := make([]string, 20)
	for i := range calls {
		calls[i] = fmt.Sprintf("op%d", i)
	}
	dir, _, spans := correctnessFixture(t, calls, "response.status == 200")
	correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
	spans[0].Attributes["response.status"] = 500
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
	correctnessCommand(t, dir, 3, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed validation must not create a baseline")
	}
}

func TestCorrectnessResultReports(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
	for _, format := range []string{"json", "junit", "html"} {
		t.Run(format, func(t *testing.T) {
			path := "report." + format
			correctnessCommand(t, dir, 4, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--semantic=false", "--report-format", format, "--report-out", path)
			b, err := os.ReadFile(filepath.Join(dir, path))
			if err != nil {
				t.Fatal(err)
			}
			if format == "junit" {
				var suite struct {
					Failures int `xml:"failures,attr"`
				}
				if err := xml.Unmarshal(b, &suite); err != nil {
					t.Fatal(err)
				}
				if suite.Failures == 0 {
					t.Error("failed gate must be a JUnit failure")
				}
				return
			}
			if format == "html" {
				_, tail, ok := strings.Cut(string(b), "window.FLOWREPORT = ")
				if !ok {
					t.Fatal("missing HTML report payload")
				}
				b = []byte(strings.TrimSuffix(tail, ";</script>"))
			}
			var report struct {
				Success  bool `json:"success"`
				ExitCode int  `json:"exitCode"`
			}
			if err := json.Unmarshal(b, &report); err != nil {
				t.Fatal(err)
			}
			if report.Success || report.ExitCode != 4 {
				t.Errorf("inconsistent final report: %+v", report)
			}
		})
	}
}

func TestCorrectnessMatching(t *testing.T) {
	for _, metadata := range []bool{false, true} {
		t.Run(fmt.Sprintf("reuse-parent-%t", metadata), func(t *testing.T) {
			calls := make([]string, 20)
			for i := range calls {
				calls[i] = "repeat"
			}
			dir, _, spans := correctnessFixture(t, calls, "true")
			if metadata {
				spans[0].Attributes["otlp.parent_span_id"] = ""
			}
			correctnessTrace(t, dir, spans[:1])
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
		t.Run(fmt.Sprintf("reverse-parent-%t", metadata), func(t *testing.T) {
			dir, _, spans := correctnessFixture(t, []string{"A", "B"}, "true")
			spans[0].StartNanos, spans[1].StartNanos = spans[1].StartNanos, spans[0].StartNanos
			spans[0].EndNanos, spans[1].EndNanos = spans[1].EndNanos, spans[0].EndNanos
			if metadata {
				for i := range spans {
					spans[i].Attributes["otlp.parent_span_id"] = ""
				}
			}
			correctnessTrace(t, dir, spans)
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
		})
	}
	// The second repeated call must be evaluated against its own matched span.
	t.Run("semantic-span-binding", func(t *testing.T) {
		dir, _, spans := correctnessFixture(t, []string{"repeat", "repeat"}, "response.status == 200")
		spans[0].Attributes["otlp.parent_span_id"] = ""
		spans[1].Attributes["response.status"] = 500
		correctnessTrace(t, dir, spans)
		correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json")
	})
}

func correctnessGraph(f *spec.FlowSpec, nodes []string, edges [][2]string) {
	f.Flow = nil
	f.Graph = &spec.GraphSpec{}
	for _, id := range nodes {
		f.Graph.Nodes = append(f.Graph.Nodes, spec.GraphNode{ID: id, Call: "svc." + id})
	}
	for _, e := range edges {
		f.Graph.Edges = append(f.Graph.Edges, spec.GraphEdge{From: e[0], To: e[1]})
	}
}

func TestCorrectnessConversionRejectsLoss(t *testing.T) {
	for _, tc := range []struct {
		name  string
		nodes []string
		edges [][2]string
	}{
		{"cycle", []string{"A", "B"}, [][2]string{{"A", "B"}, {"B", "A"}}},
		{"fork", []string{"A", "B", "C"}, [][2]string{{"A", "B"}, {"A", "C"}}},
		{"mixed-join", []string{"A", "B", "C", "X", "D"}, [][2]string{{"A", "B"}, {"A", "C"}, {"A", "D"}, {"X", "D"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, f, _ := correctnessFixture(t, tc.nodes, "true")
			correctnessGraph(f, tc.nodes, tc.edges)
			correctnessWrite(t, filepath.Join(dir, "flow.yaml"), f)
			correctnessCommand(t, dir, 2, "spec", "convert", "--in", "flow.yaml", "--out", "converted.yaml")
			if _, err := os.Stat(filepath.Join(dir, "converted.yaml")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed conversion wrote an output")
			}
		})
	}
}

func TestCorrectnessConversionPreservesParent(t *testing.T) {
	dir, f, spans := correctnessFixture(t, []string{"A", "B", "C"}, "true")
	correctnessGraph(f, []string{"A", "B", "C"}, [][2]string{{"A", "B"}, {"B", "C"}})
	correctnessWrite(t, filepath.Join(dir, "flow.yaml"), f)
	correctnessCommand(t, dir, 0, "spec", "convert", "--in", "flow.yaml", "--out", "converted.yaml")
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 0, "lint", "--flow", "converted.yaml", "--schema")
	correctnessCommand(t, dir, 0, "validate", "--flow", "converted.yaml", "--trace", "trace.json")
	correctnessTrace(t, dir, spans[1:])
	correctnessCommand(t, dir, 3, "validate", "--flow", "converted.yaml", "--trace", "trace.json")
}

func TestCorrectnessBaselineIdentity(t *testing.T) {
	for _, field := range []string{"flowId", "flowHash", "schemaVersion", "service-contract"} {
		t.Run(field, func(t *testing.T) {
			dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
			correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
			if field == "service-contract" {
				correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{{OperationId: "A", Postconditions: map[string]string{"status": "response.status == 200"}}}})
			} else {
				path := filepath.Join(dir, "baseline.json")
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var data map[string]any
				if err := json.Unmarshal(b, &data); err != nil {
					t.Fatal(err)
				}
				data[field] = "incorrect"
				b, err = json.Marshal(data)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json")
		})
	}
}

func TestCorrectnessBaselineSevereRegression(t *testing.T) {
	calls := make([]string, 20)
	for i := range calls {
		calls[i] = fmt.Sprintf("op%d", i)
	}
	dir, _, spans := correctnessFixture(t, calls, "response.status == 200")
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json")
	for i := 0; i < 18; i++ {
		spans[i].Attributes["response.status"] = 500
	}
	correctnessTrace(t, dir, spans)
	correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--report-format", "json", "--report-out", "report.json")
	b, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		GatePassed bool `json:"gatePassed"`
		ExitCode   int  `json:"exitCode"`
		Inputs     struct {
			TraceHash    string `json:"traceHash"`
			BaselineHash string `json:"baselineHash"`
		} `json:"inputs"`
	}
	if err := json.Unmarshal(b, &report); err != nil {
		t.Fatal(err)
	}
	if report.GatePassed || report.ExitCode != 3 {
		t.Fatalf("severe regression passed policy: %+v", report)
	}
	if report.Inputs.TraceHash == "" || report.Inputs.BaselineHash == "" {
		t.Fatal("report lacks input binding")
	}
}

func TestCorrectnessBaselineFallbackOnlyForMissingFile(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
	args := []string{"validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "missing.json", "--baseline-missing", "treat-as-absolute"}
	correctnessCommand(t, dir, 0, args...)
	if err := os.WriteFile(filepath.Join(dir, "missing.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 2, args...)
	correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--threshold-steps", "NaN")
}

func TestCorrectnessDiscoveryPreservesExistingFiles(t *testing.T) {
	for _, noValidate := range []bool{false, true} {
		t.Run(fmt.Sprintf("no-validate-%t", noValidate), func(t *testing.T) {
			dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
			services := filepath.Join(dir, "services")
			if err := os.Mkdir(services, 0o755); err != nil {
				t.Fatal(err)
			}
			original := []byte("original contract\n")
			service := filepath.Join(services, "svc.servicespec.yaml")
			if err := os.WriteFile(service, original, 0o644); err != nil {
				t.Fatal(err)
			}
			args := []string{"discover", "--trace", "trace.json", "--out-services", "services"}
			if noValidate {
				// A directory cannot be replaced by the FlowSpec output file.
				args = append(args, "--no-validate", "--out", "services", "--title", "Valid")
			} else {
				args = append(args, "--out", "flow.yaml", "--title", "")
			}
			before, err := os.ReadFile(filepath.Join(dir, "flow.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			out := correctnessCommand(t, dir, 2, args...)
			if !strings.Contains(out, "ERROR") {
				t.Error("expected generation error")
			}
			after, err := os.ReadFile(service)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(original) {
				t.Error("failed discover overwrote existing ServiceSpec")
			}
			flow, err := os.ReadFile(filepath.Join(dir, "flow.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(flow) != string(before) {
				t.Error("failed discover overwrote existing FlowSpec")
			}
		})
	}
}

func TestCorrectnessEndToEnd(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"op0", "op1", "op2"}, "true")
	// Discovery must resolve ServiceSpecs relative to a FlowSpec in another directory.
	correctnessCommand(t, dir, 0, "discover", "--trace", "trace.json", "--out", "contracts/discovered.yaml", "--out-services", "services", "--title", "Quoted \"title\"")
	flowPath := filepath.Join(dir, "contracts", "discovered.yaml")
	flow, err := spec.LoadFlowSpec(flowPath)
	if err != nil {
		t.Fatal(err)
	}
	// Explicit discovered relationships must not be lost in legacy conversion.
	correctnessCommand(t, dir, 2, "spec", "convert", "--in", "contracts/discovered.yaml", "--out", "converted.yaml")
	// Keep the existing conversion/report exercise on a plain serial DAG.
	for i := range flow.Graph.Edges {
		flow.Graph.Edges[i].Relationship = ""
	}
	correctnessWrite(t, filepath.Join(dir, "contracts", "graph.yaml"), flow)
	if err := os.Mkdir(filepath.Join(dir, "converted"), 0o755); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 0, "spec", "convert", "--in", "contracts/graph.yaml", "--out", "converted/flow.yaml")
	correctnessCommand(t, dir, 0, "lint", "--flow", "converted/flow.yaml", "--schema")
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "converted/flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	for _, format := range []string{"json", "junit", "html"} {
		correctnessCommand(t, dir, 0, "validate", "--flow", "converted/flow.yaml", "--trace", "trace.json", "--baseline", "baseline.json", "--report-format", format, "--report-out", "report."+format)
		b, err := os.ReadFile(filepath.Join(dir, "report."+format))
		if err != nil {
			t.Fatal(err)
		}
		if format == "html" {
			_, tail, ok := strings.Cut(string(b), "window.FLOWREPORT = ")
			if !ok {
				t.Fatal("missing HTML data")
			}
			b = []byte(strings.TrimSuffix(tail, ";</script>"))
		}
		if format == "junit" {
			var suite struct {
				Failures   int `xml:"failures,attr"`
				Properties []struct {
					Name  string `xml:"name,attr"`
					Value string `xml:"value,attr"`
				} `xml:"properties>property"`
			}
			if err := xml.Unmarshal(b, &suite); err != nil {
				t.Fatal(err)
			}
			if suite.Failures != 0 {
				t.Fatal("healthy pipeline failed JUnit")
			}
			found := false
			for _, property := range suite.Properties {
				if property.Name == "result.inputs" {
					var binding map[string]any
					if err := json.Unmarshal([]byte(property.Value), &binding); err != nil {
						t.Fatal(err)
					}
					found = binding["validatorHash"] != nil && binding["policy"] != nil
				}
			}
			if !found {
				t.Fatal("JUnit lacks input binding")
			}
			continue
		}
		var report struct {
			Success  bool `json:"success"`
			ExitCode int  `json:"exitCode"`
			Inputs   struct {
				Contract      spec.ContractIdentity `json:"contract"`
				ValidatorHash string                `json:"validatorHash"`
				Policy        map[string]any        `json:"policy"`
			} `json:"inputs"`
		}
		if err := json.Unmarshal(b, &report); err != nil {
			t.Fatal(err)
		}
		if !report.Success || report.ExitCode != 0 || len(report.Inputs.Contract.ServiceHashes) != 1 || report.Inputs.ValidatorHash == "" || report.Inputs.Policy == nil {
			t.Fatalf("incomplete healthy %s report: %+v", format, report)
		}
	}
}

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
)

// Preserve the independent review's inputs, including absent timestamps.
func architectureFixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	for _, nameFile := range []string{"flow.yaml", "svc.yaml", "trace.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", "architecture-df040d7", name, nameFile))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, nameFile), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestArchitectureTimeQualificationModes(t *testing.T) {
	for _, graph := range []bool{false, true} {
		for _, tc := range []struct {
			name, fields  string
			want, offWant int
		}{
			{"missing-both", "", 3, 0},
			{"missing-start", `,"endNanos":10`, 3, 0},
			{"missing-end", `,"startNanos":10`, 3, 0},
			{"zero-interval", `,"startNanos":0,"endNanos":0`, 0, 0},
			{"negative-start", `,"startNanos":-1,"endNanos":1`, 3, 3},
			{"negative-end", `,"startNanos":0,"endNanos":-1`, 3, 3},
			{"reverse", `,"startNanos":10,"endNanos":1`, 3, 3},
			{"null-start", `,"startNanos":null,"endNanos":1`, 2, 2},
			{"null-end", `,"startNanos":0,"endNanos":null`, 2, 2},
			{"string", `,"startNanos":"0","endNanos":1`, 2, 2},
			{"fraction", `,"startNanos":0.5,"endNanos":1`, 2, 2},
		} {
			for _, mode := range []string{"temporal", "strict", "off"} {
				t.Run(fmt.Sprintf("graph=%t/%s/%s", graph, tc.name, mode), func(t *testing.T) {
					dir, flow, _ := correctnessFixture(t, []string{"A"}, "true")
					if graph {
						flow.Flow = nil
						flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "A", Call: "svc.A"}}}
						correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
					}
					data := []byte(`{"spans":[{"service":"svc","name":"A","attributes":{"response.status":200}` + tc.fields + `}]}`)
					if err := os.WriteFile(filepath.Join(dir, "trace.json"), data, 0o644); err != nil {
						t.Fatal(err)
					}
					want := tc.want
					if mode == "off" {
						want = tc.offWant
					}
					correctnessCommand(t, dir, want, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", mode)
				})
			}
		}
	}
}

func TestArchitectureParallelTimeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"parallel-missing-start", 3}, {"parallel-missing-end", 3},
		{"parallel-explicit-zero", 0}, {"parallel-present-start", 3},
	} {
		for _, mode := range []string{"temporal", "off"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				dir := architectureFixture(t, tc.name)
				correctnessCommand(t, dir, tc.want, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", mode, "--causality-tolerance", "0")
			})
		}
	}
}

func TestArchitectureTimeReportsAndBaseline(t *testing.T) {
	dir := architectureFixture(t, "parallel-missing-start")
	correctnessCommand(t, dir, 3, "ci-gate", "--flow", "flow.yaml", "--trace", "trace.json")
	correctnessCommand(t, dir, 3, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !os.IsNotExist(err) {
		t.Error("missing time evidence was recorded as a baseline")
	}
	for _, format := range []string{"json", "junit", "html"} {
		correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", format, "--report-out", "report."+format)
		data, err := os.ReadFile(filepath.Join(dir, "report."+format))
		if err != nil {
			t.Fatal(err)
		}
		switch format {
		case "json":
			var report struct {
				Success bool `json:"success"`
				Inputs  struct {
					TraceHash string `json:"traceHash"`
				} `json:"inputs"`
			}
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Success || report.Inputs.TraceHash != snapshotHash(t, filepath.Join(dir, "trace.json")) {
				t.Error("report outcome or trace identity disagrees with missing evidence")
			}
		case "junit":
			var suite struct {
				Failures int `xml:"failures,attr"`
			}
			if err := xml.Unmarshal(data, &suite); err != nil || suite.Failures == 0 {
				t.Errorf("missing evidence is not a JUnit failure: %v", err)
			}
		case "html":
			if !bytes.Contains(data, []byte("VALIDATION_FAILED")) {
				t.Error("HTML hides validation failure")
			}
		}
	}
	validTrace, err := os.ReadFile(filepath.Join("testdata", "architecture-df040d7", "parallel-explicit-zero", "trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	missingTrace, err := os.ReadFile(filepath.Join(dir, "trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trace.json"), validTrace, 0o644); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "valid-baseline.json")
	if err := os.WriteFile(filepath.Join(dir, "trace.json"), missingTrace, 0o644); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--baseline", "valid-baseline.json")
}

func TestArchitectureInitQuotedTitle(t *testing.T) {
	for _, mode := range []string{"template", "trace"} {
		t.Run(mode, func(t *testing.T) {
			dir := architectureFixture(t, "parallel-explicit-zero")
			title := "Checkout \"verified\"\npath ./services/ stays text"
			args := []string{"init", "--yes", "--mode", mode, "--title", title, "--out", "project", "--examples", "--ci", "minimal"}
			if mode == "trace" {
				args = append(args, "--trace", "trace.json")
			}
			correctnessCommand(t, dir, 0, args...)
			project := filepath.Join(dir, "project")
			flows, err := filepath.Glob(filepath.Join(project, "flows", "*.flowspec.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			flows = append(flows, filepath.Join(project, ".flowspec.yaml"))
			for _, path := range flows {
				correctnessCommand(t, project, 0, "lint", "--flow", path)
				flow, err := spec.LoadFlowSpec(path)
				if err != nil {
					t.Fatal(err)
				}
				if flow.Info.Title != title {
					t.Errorf("title changed: %q", flow.Info.Title)
				}
			}
		})
	}
}

func TestArchitectureInitLateConflict(t *testing.T) {
	for _, mode := range []string{"template", "trace"} {
		for _, force := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/file", true: "/directory"}[force], func(t *testing.T) {
				dir := architectureFixture(t, "parallel-explicit-zero")
				project := filepath.Join(dir, "project")
				workflow := filepath.Join(project, ".github", "workflows", "choreoatlas.yml")
				if err := os.MkdirAll(filepath.Dir(workflow), 0o755); err != nil {
					t.Fatal(err)
				}
				if force {
					if err := os.Mkdir(workflow, 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(project, ".flowspec.yaml"), []byte("original contract"), 0o600); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(workflow, []byte("original workflow"), 0o600); err != nil {
					t.Fatal(err)
				}
				args := []string{"init", "--yes", "--mode", mode, "--out", "project", "--ci", "combo"}
				if mode == "trace" {
					args = append(args, "--trace", "trace.json")
				}
				if force {
					args = append(args, "--force")
				}
				want := 1
				if force {
					want = 2
				}
				correctnessCommand(t, dir, want, args...)
				root := filepath.Join(project, ".flowspec.yaml")
				if force {
					data, err := os.ReadFile(root)
					if err != nil || string(data) != "original contract" {
						t.Errorf("existing contract changed: %s, %v", data, err)
					}
				} else if _, err := os.Stat(root); !os.IsNotExist(err) {
					t.Error("partial initialization left a root contract")
				}
				if paths, _ := filepath.Glob(filepath.Join(project, "services", "*")); len(paths) != 0 {
					t.Error("partial initialization wrote services")
				}
			})
		}
	}
}

func TestArchitectureUnsupportedConditions(t *testing.T) {
	for _, name := range []string{"graph-condition-false", "graph-condition-ordered"} {
		t.Run(name, func(t *testing.T) {
			dir := architectureFixture(t, name)
			if err := os.WriteFile(filepath.Join(dir, "converted.yaml"), []byte("original conversion"), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{
				{"lint", "--flow", "flow.yaml"}, {"lint", "--flow", "flow.yaml", "--schema=false"},
				{"validate", "--flow", "flow.yaml", "--trace", "trace.json"},
				{"validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality", "off", "--semantic=false"},
				{"ci-gate", "--flow", "flow.yaml", "--trace", "trace.json"},
				{"baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json"},
				{"spec", "convert", "--in", "flow.yaml", "--out", "converted.yaml"},
			} {
				correctnessCommand(t, dir, 2, args...)
			}
			if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !os.IsNotExist(err) {
				t.Error("unsupported contract recorded")
			}
			converted, err := os.ReadFile(filepath.Join(dir, "converted.yaml"))
			if err != nil || string(converted) != "original conversion" {
				t.Error("unsupported conversion overwrote output")
			}
			data, err := os.ReadFile(filepath.Join(dir, "flow.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			data = bytes.ReplaceAll(data, []byte(`"condition": "false"`), []byte(`"condition": ""`))
			if err := os.WriteFile(filepath.Join(dir, "flow.yaml"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			correctnessCommand(t, dir, 0, "lint", "--flow", "flow.yaml")
			want := 3
			if name == "graph-condition-ordered" {
				want = 0
			}
			correctnessCommand(t, dir, want, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--causality-tolerance", "0")
		})
	}
}

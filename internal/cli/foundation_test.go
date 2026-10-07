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

	"github.com/choreoatlas2025/cli/internal/result"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

func TestFoundationEvidenceConsumers(t *testing.T) {
	for _, graph := range []bool{false, true} {
		for _, tc := range []struct {
			name, id string
			issue    verdict.IssueKind
		}{
			{"correct", "ORD-A", ""}, {"wrong", "ORD-B", verdict.RuleViolation}, {"missing", "", verdict.MissingEvidence},
		} {
			t.Run(map[bool]string{false: "flow", true: "dag"}[graph]+"/"+tc.name, func(t *testing.T) {
				dir, flow, spans := correctnessFixture(t, []string{"pay"}, "true")
				flow.Flow[0].Input = map[string]any{"id": "ORD-A"}
				if graph {
					flow.Flow = nil
					flow.Graph = &spec.GraphSpec{Nodes: []spec.GraphNode{{ID: "pay", Call: "svc.pay", Input: map[string]any{"id": "ORD-A"}}}}
				}
				correctnessWrite(t, filepath.Join(dir, "flow.yaml"), flow)
				correctnessWrite(t, filepath.Join(dir, "svc.yaml"), spec.ServiceSpecFile{Service: "svc", Operations: []spec.ServiceOperation{{OperationId: "pay", Preconditions: map[string]string{"same-order": "request.body.id == expected.body.id"}}}})
				spans[0].Attributes["otlp.span_id"] = "exact-pay"
				if tc.id != "" {
					spans[0].Attributes["request.body"] = map[string]any{"id": tc.id}
				}
				correctnessTrace(t, dir, spans)
				want := 0
				if tc.issue != "" {
					want = 3
				}
				for _, format := range []string{"json", "junit", "html"} {
					path := filepath.Join(dir, "report."+format)
					correctnessCommand(t, dir, want, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--threshold-steps", "0", "--threshold-conds", "0", "--report-format", format, "--report-out", path)
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					var step verdict.StepResult
					if format == "junit" {
						var suite struct {
							Failures int `xml:"failures,attr"`
							Cases    []struct {
								Out string `xml:"system-out"`
							} `xml:"testcase"`
						}
						if err := xml.Unmarshal(data, &suite); err != nil {
							t.Fatal(err)
						}
						if (suite.Failures != 0) != (want != 0) || len(suite.Cases) != 1 {
							t.Fatalf("JUnit decision drift: %+v", suite)
						}
						if err := json.Unmarshal([]byte(suite.Cases[0].Out), &step); err != nil {
							t.Fatal(err)
						}
					} else {
						if format == "html" {
							text := string(data)
							marker := "window.FLOWREPORT = "
							start := strings.LastIndex(text, marker)
							if start < 0 {
								t.Fatal("HTML payload absent")
							}
							text = text[start+len(marker):]
							end := strings.Index(text, ";</script>")
							if end < 0 {
								t.Fatal("HTML payload incomplete")
							}
							data = []byte(text[:end])
						}
						var rendered struct {
							Success  bool                 `json:"success"`
							ExitCode int                  `json:"exitCode"`
							Steps    []verdict.StepResult `json:"steps"`
							Inputs   *result.InputBinding `json:"inputs"`
						}
						if err := json.Unmarshal(data, &rendered); err != nil {
							t.Fatal(err)
						}
						if rendered.Success != (want == 0) || rendered.ExitCode != want || len(rendered.Steps) != 1 || rendered.Inputs.TraceHash != snapshotHash(t, filepath.Join(dir, "trace.json")) {
							t.Fatalf("%s decision/input drift: %+v", format, rendered)
						}
						step = rendered.Steps[0]
					}
					if step.Issue != tc.issue || step.Evidence == nil || step.Evidence.Key != "exact-pay" {
						t.Fatalf("%s lost cause or matched instance: %+v", format, step)
					}
				}
				correctnessCommand(t, dir, want, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
				_, err := os.Stat(filepath.Join(dir, "baseline.json"))
				if (want == 0 && err != nil) || (want != 0 && !os.IsNotExist(err)) {
					t.Fatalf("baseline consumed unproven evidence: %v", err)
				}
			})
		}
	}
}

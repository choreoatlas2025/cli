// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/baseline"
	"github.com/choreoatlas2025/cli/internal/result"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func TestArchitectureNeutralReportConsistency(t *testing.T) {
	for _, tc := range []struct {
		name, status           string
		conditions             []validate.ConditionResult
		gateFailed             bool
		exit, pass, fail, skip int
	}{
		{"pass", "PASS", nil, false, 0, 1, 0, 0},
		{"condition-fail", "PASS", []validate.ConditionResult{{Status: "FAIL"}}, false, 3, 0, 1, 0},
		{"condition-skip", "PASS", []validate.ConditionResult{{Status: "SKIP"}}, false, 3, 0, 1, 0},
		{"step-skip", "SKIP", nil, false, 3, 0, 0, 1},
		{"gate-fail", "PASS", nil, true, 4, 1, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps := []validate.StepResult{{Step: "A", Call: "svc.A", Status: tc.status, Conditions: tc.conditions}}
			gate := &result.GateResult{Checked: true, Passed: !tc.gateFailed, Details: map[string]any{"stepsThreshold": 0.0, "conditionsThreshold": 0.0}}
			input := &result.InputBinding{Version: "fixture-ce", GitCommit: "fixture", TraceHash: "captured-input"}
			dir := t.TempDir()
			for _, format := range []ReportFormat{ReportJSON, ReportJUnit, ReportHTML} {
				path := filepath.Join(dir, string(format))
				if err := WriteReport(path, format, steps, nil, gate, input); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if format == ReportJUnit {
					var suite struct {
						Failures   int `xml:"failures,attr"`
						Properties []struct {
							Name  string `xml:"name,attr"`
							Value string `xml:"value,attr"`
						} `xml:"properties>property"`
					}
					if err := xml.Unmarshal(data, &suite); err != nil {
						t.Fatal(err)
					}
					properties := map[string]string{}
					for _, p := range suite.Properties {
						properties[p.Name] = p.Value
					}
					if properties["result.exitCode"] != strconv.Itoa(tc.exit) || properties["coverage.stepsPass"] != strconv.Itoa(tc.pass) || properties["coverage.stepsFail"] != strconv.Itoa(tc.fail) || properties["coverage.stepsSkip"] != strconv.Itoa(tc.skip) {
						t.Fatalf("JUnit drift: %+v", properties)
					}
					var binding result.InputBinding
					if err := json.Unmarshal([]byte(properties["result.inputs"]), &binding); err != nil || binding.TraceHash != input.TraceHash {
						t.Fatalf("JUnit identity drift: %v", err)
					}
					if (suite.Failures > 0) != (tc.exit != 0) {
						t.Fatalf("JUnit status drift: %+v", suite)
					}
					continue
				}
				if format == ReportHTML {
					text := string(data)
					marker := "window.FLOWREPORT = "
					start := strings.LastIndex(text, marker)
					if start < 0 {
						t.Fatal("HTML payload missing")
					}
					text = text[start+len(marker):]
					end := strings.Index(text, ";</script>")
					if end < 0 {
						t.Fatal("HTML payload not terminated")
					}
					data = []byte(text[:end])
				}
				var report struct {
					ExitCode int                    `json:"exitCode"`
					Success  bool                   `json:"success"`
					Inputs   *result.InputBinding   `json:"inputs"`
					Summary  result.CoverageSummary `json:"summary"`
				}
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if report.ExitCode != tc.exit || report.Success != (tc.exit == 0) || report.Summary.StepsPass != tc.pass || report.Summary.StepsFail != tc.fail || report.Summary.StepsSkip != tc.skip || report.Inputs.TraceHash != input.TraceHash {
					t.Fatalf("%s drift: %+v", format, report)
				}
			}
			policy := baseline.DefaultThresholds()
			policy.StepsThreshold, policy.ConditionsThreshold = 0, 0
			evaluated := baseline.EvaluateGate(steps, policy, nil)
			if evaluated.Details["stepsPass"] != tc.pass {
				t.Fatalf("gate and report count different successful steps: %+v", evaluated)
			}
		})
	}
}

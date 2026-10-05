// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/result"
)

func TestArchitectureCELQualification(t *testing.T) {
	for _, expr := range []string{"response.status ==", "200", "response.missing == 1", "response.status"} {
		t.Run(expr, func(t *testing.T) {
			dir, _, _ := correctnessFixture(t, []string{"A"}, expr)
			lintWant := 2
			if expr == "response.missing == 1" || expr == "response.status" {
				lintWant = 0
			}
			correctnessCommand(t, dir, lintWant, "lint", "--flow", "flow.yaml")
			correctnessCommand(t, dir, 3, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", "json", "--report-out", "report.json")
			data, err := os.ReadFile(filepath.Join(dir, "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			var r result.Report
			if err := json.Unmarshal(data, &r); err != nil {
				t.Fatal(err)
			}
			if r.Success || r.ExitCode != 3 || r.FailedSteps == 0 {
				t.Fatalf("CEL error hidden: %+v", r)
			}
			correctnessCommand(t, dir, 3, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
			if _, err := os.Stat(filepath.Join(dir, "baseline.json")); !os.IsNotExist(err) {
				t.Fatal("bad CEL was recorded")
			}
		})
	}
}
func TestArchitectureResourceLimits(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"A", "B"}, "response.status == 200")
	for _, tc := range []struct {
		flag, value string
		want        int
	}{
		{"--max-input-bytes", "16", 2}, {"--max-spans", "1", 2}, {"--max-steps", "1", 3},
		{"--max-cel-cost", "1", 3}, {"--max-spans", "0", 2}, {"--validation-timeout-ms", "-1", 2},
		{"--max-input-bytes", "4096", 0}, {"--max-spans", "2", 0}, {"--max-steps", "2", 0}, {"--max-cel-cost", "100", 0},
	} {
		correctnessCommand(t, dir, tc.want, "validate", "--flow", "flow.yaml", "--trace", "trace.json", tc.flag, tc.value)
	}
	correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--max-input-bytes", "4096", "--report-format", "json", "--report-out", "report.json")
	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var r result.Report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if r.Inputs.Limits.MaxInputBytes != 4096 || r.Inputs.PlanHash == "" || r.Inputs.ValidatorHash == "" {
		t.Fatalf("missing bound plan identity: %+v", r.Inputs)
	}
}

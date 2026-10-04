// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package html

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/validate"
)

// CI and the standalone browser test consume the actual Go-generated report.
func TestHTMLUntrustedFieldsFixture(t *testing.T) {
	payload := `</script><img class="ce-external" src="ce-missing.png" onerror="window.ceInjected=true">`
	data := BuildHTMLData([]validate.StepResult{{Step: payload, Call: payload, Status: "FAIL", Message: payload,
		Conditions: []validate.ConditionResult{{Kind: payload, Name: payload, Status: "FAIL"}}}},
		[]SpanInfo{{Service: payload, Name: payload, StartNanos: 1, EndNanos: 2}},
		&GateResult{Checked: true, Passed: false, Details: map[string]any{"stepsThreshold": 0.0, "conditionsThreshold": 0.0}, Violations: []string{payload}})
	path := os.Getenv("CHOREOATLAS_HTML_FIXTURE")
	if path == "" {
		path = filepath.Join(t.TempDir(), "report.html")
	}
	if err := WriteHTMLReport(path, data); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), payload) {
		t.Fatal("raw external markup escaped the JSON payload boundary")
	}
}

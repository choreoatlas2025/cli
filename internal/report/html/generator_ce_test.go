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

func TestCEEditionBadge(t *testing.T) {
	data := BuildHTMLData([]validate.StepResult{
		{Step: "test1", Status: "PASS"},
		{Step: "test2", Status: "PASS"},
	}, []SpanInfo{}, nil)
	output := filepath.Join(t.TempDir(), "report.html")
	if err := WriteHTMLReport(output, data); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	report := string(content)
	if !strings.Contains(report, `"edition":"CE"`) {
		t.Error("report metadata must identify CE")
	}
	if !strings.Contains(report, `<div class="edition-badge ce" id="edition-badge">CE</div>`) {
		t.Error("report must display the CE badge without JavaScript")
	}
	if !strings.Contains(report, `"stepsTotal":2`) {
		t.Error("report must preserve validation results")
	}
}

func TestEmptyEditionHandling(t *testing.T) {
	for _, label := range []string{"", "custom"} {
		t.Run("label="+label, func(t *testing.T) {
			data := HTMLData{Edition: label}
			output := filepath.Join(t.TempDir(), "report.html")
			if err := WriteHTMLReport(output, data); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), `"edition":"CE"`) {
				t.Error("report metadata must identify CE regardless of the input label")
			}
		})
	}
}

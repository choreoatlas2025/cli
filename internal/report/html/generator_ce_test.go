// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package html

import (
	"os"
	"strings"
	"testing"

	"github.com/choreoatlas2025/cli/internal/validate"
)

// TestCEEditionBadge specifically tests CE edition badge rendering
func TestCEEditionBadge(t *testing.T) {
	// Create test data with CE edition
	data := HTMLData{
		Summary: CoverageSummary{
			StepsTotal:    2,
			StepsPass:     2,
			StepsCoverage: 1.0,
		},
		Steps: []validate.StepResult{
			{Step: "test1", Status: "PASS"},
			{Step: "test2", Status: "PASS"},
		},
		Spans:   []SpanInfo{},
		Edition: "CE", // Explicitly set CE edition
	}

	// Generate HTML report
	tempFile := "/tmp/test-ce-badge.html"
	defer os.Remove(tempFile)

	err := WriteHTMLReport(tempFile, data)
	if err != nil {
		t.Fatalf("Failed to write HTML report: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	htmlContent := string(content)

	// Test 1: Verify CE edition in embedded data
	if !strings.Contains(htmlContent, `"edition":"CE"`) {
		t.Error("HTML should contain CE edition in embedded FLOWREPORT data")
	}

	// Test 2: Verify edition badge element exists
	if !strings.Contains(htmlContent, `id="edition-badge"`) {
		t.Error("HTML should contain edition-badge element")
	}

	// Test 3: Verify edition badge CSS class for CE
	if !strings.Contains(htmlContent, `.edition-badge.ce`) {
		t.Error("HTML should contain CE-specific CSS class definition")
	}

	// Test 4: Verify badge styling (gradient for CE)
	if !strings.Contains(htmlContent, `linear-gradient(135deg, #059669, #10b981)`) {
		t.Error("HTML should contain CE edition gradient styling")
	}

	// Test 5: Verify JavaScript that sets the badge
	if !strings.Contains(htmlContent, `const edition = data.edition || 'Unknown'`) {
		t.Error("HTML should contain JavaScript to read edition from data")
	}

	if !strings.Contains(htmlContent, `editionBadge.textContent = edition`) {
		t.Error("HTML should contain JavaScript to set badge text")
	}

	// Test 6: Verify the badge position styling
	if !strings.Contains(htmlContent, `position: absolute`) {
		t.Error("Edition badge should have absolute positioning")
	}
}

// TestEditionBadgeForAllEditions tests badge rendering for different editions
func TestEditionBadgeForAllEditions(t *testing.T) {
	editions := []struct {
		name     string
		cssClass string
	}{
		{"CE", "ce"},
		{"Pro", "pro"},
		{"Pro Privacy", "pro-privacy"},
		{"Cloud", "cloud"},
	}

	for _, edition := range editions {
		t.Run(edition.name, func(t *testing.T) {
			data := HTMLData{
				Summary: CoverageSummary{},
				Steps:   []validate.StepResult{},
				Spans:   []SpanInfo{},
				Edition: edition.name,
			}

			tempFile := "/tmp/test-edition-" + edition.cssClass + ".html"
			defer os.Remove(tempFile)

			err := WriteHTMLReport(tempFile, data)
			if err != nil {
				t.Fatalf("Failed to write HTML report for %s: %v", edition.name, err)
			}

			content, err := os.ReadFile(tempFile)
			if err != nil {
				t.Fatalf("Failed to read HTML file for %s: %v", edition.name, err)
			}

			htmlContent := string(content)

			// Verify edition is in embedded data
			expectedJSON := `"edition":"` + edition.name + `"`
			if !strings.Contains(htmlContent, expectedJSON) {
				t.Errorf("HTML for %s should contain edition in JSON: %s", edition.name, expectedJSON)
			}

			// Verify CSS class exists
			expectedCSS := ".edition-badge." + edition.cssClass
			if !strings.Contains(htmlContent, expectedCSS) {
				t.Errorf("HTML for %s should contain CSS class: %s", edition.name, expectedCSS)
			}
		})
	}
}

// TestEmptyEditionHandling tests handling when edition is not set
func TestEmptyEditionHandling(t *testing.T) {
	data := HTMLData{
		Summary: CoverageSummary{},
		Steps:   []validate.StepResult{},
		Spans:   []SpanInfo{},
		Edition: "", // Empty edition
	}

	tempFile := "/tmp/test-empty-edition.html"
	defer os.Remove(tempFile)

	err := WriteHTMLReport(tempFile, data)
	if err != nil {
		t.Fatalf("Failed to write HTML report: %v", err)
	}

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read HTML file: %v", err)
	}

	htmlContent := string(content)

	// Should still have the badge element
	if !strings.Contains(htmlContent, `id="edition-badge"`) {
		t.Error("HTML should contain edition-badge element even with empty edition")
	}

	// JavaScript should handle empty edition with fallback
	if !strings.Contains(htmlContent, `data.edition || 'Unknown'`) {
		t.Error("HTML should have fallback for empty edition")
	}
}
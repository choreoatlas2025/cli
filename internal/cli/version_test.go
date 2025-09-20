package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run version command
	runVersion([]string{})

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Test for CE suffix in version line
	if !strings.Contains(output, "-ce") {
		t.Errorf("Version output missing -ce suffix: %s", output)
	}

	// Test for Community Edition text
	if !strings.Contains(output, "Community Edition (CE)") {
		t.Errorf("Version output missing 'Community Edition (CE)' text: %s", output)
	}

	// Test for other required fields
	requiredFields := []string{
		"Git Commit:",
		"Build Time:",
		"Go Version:",
		"Platform:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(output, field) {
			t.Errorf("Version output missing required field '%s': %s", field, output)
		}
	}
}

func TestBuildEditionValue(t *testing.T) {
	// Verify BuildEdition is set to "ce"
	if BuildEdition != "ce" {
		t.Errorf("BuildEdition should be 'ce', got: %s", BuildEdition)
	}
}
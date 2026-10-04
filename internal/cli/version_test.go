// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	originalVersion := Version
	t.Cleanup(func() { Version = originalVersion })

	cases := []struct {
		name    string
		version string
		want    string
	}{
		{"plain", "0.8.0", "v0.8.0-ce"},
		{"prefixed", "v0.8.0", "v0.8.0-ce"},
		{"release tag", "v0.8.0-ce", "v0.8.0-ce"},
		{"release version", "0.8.0-ce", "v0.8.0-ce"},
		{"release candidate tag", "v0.8.1-ce.rc.1", "v0.8.1-ce.rc.1"},
		{"release candidate version", "0.8.1-ce.rc.1", "v0.8.1-ce.rc.1"},
		{"development", "0.8.0-dev", "v0.8.0-dev-ce"},
		{"git describe", "v0.8.0-ce-3-gabcdef-dirty", "v0.8.0-ce-3-gabcdef-dirty"},
		{"build metadata", "v0.8.0+build.1", "v0.8.0-ce+build.1"},
		{"tag with build metadata", "v0.8.0-ce+build.1", "v0.8.0-ce+build.1"},
		{"similar prerelease name", "0.8.0-ceiling", "v0.8.0-ceiling-ce"},
		{"empty", "", "vdev-ce"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			Version = tc.version
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = r.Close() }()
			t.Cleanup(func() { os.Stdout = originalStdout })
			os.Stdout = w
			runVersion(nil)
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			os.Stdout = originalStdout

			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			firstLine := strings.SplitN(text, "\n", 2)[0]
			if want := "choreoatlas " + tc.want; firstLine != want {
				t.Errorf("version line = %q, want %q", firstLine, want)
			}
			requiredFields := []string{
				"Edition: Community Edition (CE)",
				"Git Commit:",
				"Build Time:",
				"Go Version:",
				"Platform:",
			}
			for _, field := range requiredFields {
				if !strings.Contains(text, field) {
					t.Errorf("version output missing %q: %s", field, text)
				}
			}
		})
	}
}

func TestBuildEditionValue(t *testing.T) {
	// Verify BuildEdition is set to "ce"
	if BuildEdition != "ce" {
		t.Errorf("BuildEdition should be 'ce', got: %s", BuildEdition)
	}
}

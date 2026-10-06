// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestArchitectureLinkedBuildIdentity(t *testing.T) {
	if os.Getenv("CHOREOATLAS_CORRECTNESS_BINARY") == "" {
		t.Skip("requires linked CLI artifact")
	}
	raw, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(raw))
	out := correctnessCommand(t, t.TempDir(), 0, "version")
	if len(sha) != 40 || !strings.Contains(out, "Git Commit: "+sha+"\n") || !strings.Contains(out, "Build Channel: make\n") {
		t.Fatalf("linked identity differs: %s", out)
	}
}

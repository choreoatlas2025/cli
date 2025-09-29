// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/choreoatlas2025/cli/internal/cli/exitcode"
)

func TestBaselineRecordStopsWhenValidationFails(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	flowPath := filepath.Join(repoRoot, "examples", "flows", "order-fulfillment.flowspec.yaml")
	if _, err := os.Stat(flowPath); err != nil {
		t.Fatalf("flow spec not found: %v", err)
	}

	tracePath := filepath.Join(repoRoot, "examples", "traces", "failed-inventory.trace.json")
	if _, err := os.Stat(tracePath); err != nil {
		t.Fatalf("trace file not found: %v", err)
	}

	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "baseline.json")

	binPath := filepath.Join(tempDir, "choreoatlas.testbin")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/choreoatlas")
	build.Dir = repoRoot
	build.Env = os.Environ()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build choreoatlas binary: %v\n%s", err, output)
	}

	cmd := exec.Command(binPath, "baseline", "record", "--flow", flowPath, "--trace", tracePath, "--out", outPath)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected baseline record to fail validation, got success: %s", output)
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v\noutput: %s", err, err, output)
	}

	if code := exitErr.ExitCode(); code != exitcode.ValidationFailed {
		t.Fatalf("unexpected exit code: got %d, want %d\noutput: %s", code, exitcode.ValidationFailed, output)
	}

	if _, statErr := os.Stat(outPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no baseline file to be written, got stat error: %v", statErr)
	}
}

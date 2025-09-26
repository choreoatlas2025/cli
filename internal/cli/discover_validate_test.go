// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/choreoatlas2025/cli/internal/spec"
    "github.com/choreoatlas2025/cli/internal/trace"
)

func TestValidateAndPersistFlow_FailsOnInvalidCall(t *testing.T) {
    // Prepare temp workspace
    dir := t.TempDir()
    outFlow := filepath.Join(dir, "discovered.flowspec.yaml")
    outServices := filepath.Join(dir, "services")
    _ = os.MkdirAll(outServices, 0o755)

    // Craft a trace with an invalid operation name (contains space and slash)
    traceJSON := `{
        "spans": [
          {
            "name": "GET /health",
            "service": "market-data-service",
            "startNanos": 1,
            "endNanos": 2,
            "attributes": {"http.status_code": 200}
          }
        ]
    }`
    tracePath := filepath.Join(dir, "invalid.trace.json")
    if err := os.WriteFile(tracePath, []byte(traceJSON), 0o644); err != nil {
        t.Fatalf("failed to write trace: %v", err)
    }

    tr, err := trace.LoadFromFile(tracePath)
    if err != nil {
        t.Fatalf("failed to load trace: %v", err)
    }

    // Generate service specs first (required by validation)
    if err := spec.GenerateServiceSpecs(tr.Spans, outServices); err != nil {
        t.Fatalf("failed to generate servicespecs: %v", err)
    }

    // Generate Flow YAML and validate
    yml := generateFlowYAML(tr, "From Invalid Trace", outServices)
    if err := validateAndPersistFlow(yml, outFlow, outServices); err == nil {
        t.Fatalf("expected validation to fail due to invalid call, but got nil error")
    }

    if _, err := os.Stat(outFlow); err == nil {
        t.Fatalf("flow file should not be written on validation failure")
    }
}

func TestValidateAndPersistFlow_PassesOnValidExample(t *testing.T) {
    dir := t.TempDir()
    outFlow := filepath.Join(dir, "discovered.flowspec.yaml")
    outServices := filepath.Join(dir, "services")
    _ = os.MkdirAll(outServices, 0o755)

    // Use repo example trace with clean operation names
    repoRoot, _ := os.Getwd()
    exampleTrace := filepath.Join(repoRoot, "../../examples/traces/successful-order.trace.json")
    // Normalize path relative to this test file location
    if _, err := os.Stat(exampleTrace); err != nil {
        t.Fatalf("example trace not found: %v", err)
    }

    tr, err := trace.LoadFromFile(exampleTrace)
    if err != nil {
        t.Fatalf("failed to load example trace: %v", err)
    }

    // Generate ServiceSpecs required for validation
    if err := spec.GenerateServiceSpecs(tr.Spans, outServices); err != nil {
        t.Fatalf("failed to generate servicespecs: %v", err)
    }

    yml := generateFlowYAML(tr, "From Example", outServices)
    if err := validateAndPersistFlow(yml, outFlow, outServices); err != nil {
        t.Fatalf("unexpected validation failure: %v", err)
    }

    if _, err := os.Stat(outFlow); err != nil {
        t.Fatalf("expected validated flow written, stat error: %v", err)
    }
}


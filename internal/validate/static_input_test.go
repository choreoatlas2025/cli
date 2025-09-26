// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package validate

import (
    "os"
    "path/filepath"
    "testing"
    "strings"

    "github.com/choreoatlas2025/cli/internal/spec"
)

func TestLint_FlagsTelemetryInInput(t *testing.T) {
    dir := t.TempDir()
    // Write ServiceSpec
    svcDir := filepath.Join(dir, "services")
    _ = os.MkdirAll(svcDir, 0o755)
    svcPath := filepath.Join(svcDir, "svc.servicespec.yaml")
    if err := os.WriteFile(svcPath, []byte("service: svc\noperations:\n  - operationId: getHealth\n"), 0o644); err != nil {
        t.Fatalf("write service spec: %v", err)
    }
    // Write FlowSpec with telemetry keys under input
    flowPath := filepath.Join(dir, "flow.yaml")
    flowYAML := "" +
        "info:\n  title: t\n" +
        "services:\n  svc:\n    spec: \"./services/svc.servicespec.yaml\"\n" +
        "flow:\n  - step: s1\n    call: svc.getHealth\n    input:\n      body:\n        http.method: GET\n        otel.status_code: 200\n"
    if err := os.WriteFile(flowPath, []byte(flowYAML), 0o644); err != nil {
        t.Fatalf("write flow: %v", err)
    }
    fs, err := spec.LoadFlowSpec(flowPath)
    if err != nil {
        t.Fatalf("load flow: %v", err)
    }
    _, opIndex, err := fs.BuildOperationIndex(flowPath)
    if err != nil {
        t.Fatalf("build op index: %v", err)
    }
    issues, err := LintFlow(flowPath, fs, opIndex)
    if err != nil {
        t.Fatalf("lint: %v", err)
    }
    // Expect at least one ERROR mentioning telemetry keys
    found := false
    for _, is := range issues {
        if is.Level == "ERROR" && (strings.Contains(is.Msg, "http.method") || strings.Contains(is.Msg, "otel.status_code")) {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected lint to report telemetry keys under input as ERROR, got: %#v", issues)
    }
}


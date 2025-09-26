// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package spec

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/choreoatlas2025/cli/internal/trace"
    "gopkg.in/yaml.v3"
)

func TestGenerateServiceSpecs_HTTPAttributesToConditions(t *testing.T) {
    spans := []trace.Span{
        {
            Name:    "GET /health",
            Service: "svc",
            Attributes: map[string]any{
                "http.method":         "GET",
                "http.route":          "/health",
                "http.status_code":    int64(200),
                "user_agent.original": "curl/8.1",
            },
        },
    }

    dir := t.TempDir()
    if err := GenerateServiceSpecs(spans, dir); err != nil {
        t.Fatalf("GenerateServiceSpecs: %v", err)
    }

    // Load generated YAML
    data, err := os.ReadFile(filepath.Join(dir, "svc.servicespec.yaml"))
    if err != nil {
        t.Fatalf("read servicespec: %v", err)
    }
    var ss ServiceSpecFile
    if err := yaml.Unmarshal(data, &ss); err != nil {
        t.Fatalf("parse servicespec: %v", err)
    }
    if len(ss.Operations) == 0 {
        t.Fatalf("no operations generated")
    }
    op := ss.Operations[0]
    // Preconditions should include http.method and http.route
    foundMethod := false
    foundRoute := false
    for _, expr := range op.Preconditions {
        if expr == "http.method == 'GET'" {
            foundMethod = true
        }
        if expr == "http.route == '/health'" {
            foundRoute = true
        }
    }
    if !foundMethod || !foundRoute {
        t.Fatalf("expected preconditions for http.method and http.route, got: %#v", op.Preconditions)
    }
    // Postconditions should include response.status == 200
    foundStatus := false
    for _, expr := range op.Postconditions {
        if expr == "response.status == 200" {
            foundStatus = true
        }
    }
    if !foundStatus {
        t.Fatalf("expected postcondition for response.status == 200, got: %#v", op.Postconditions)
    }
}


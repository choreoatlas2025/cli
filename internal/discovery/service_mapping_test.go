// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package discovery

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/spec"
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

	files, err := BuildServiceSpecFiles(spans)
	if err != nil {
		t.Fatal(err)
	}
	data := files["svc.servicespec.yaml"]
	var ss spec.ServiceSpecFile
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
		if expr == `span.attributes["http.method"] == "GET"` {
			foundMethod = true
		}
		if expr == `span.attributes["http.route"] == "/health"` {
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

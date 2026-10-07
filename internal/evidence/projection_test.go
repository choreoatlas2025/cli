// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package evidence

import (
	"testing"

	"github.com/choreoatlas2025/cli/internal/trace"
)

func TestProjectionPresenceIdentityAndIsolation(t *testing.T) {
	sp := trace.Span{Service: "svc", Name: "call", StartNanos: 1, Attributes: map[string]any{
		"otlp.trace_id": "trace", "otlp.span_id": "span",
		"request.body": map[string]any{"id": "actual"}, "http.response.status_code": 0,
		"unrelated": "not a body",
	}}
	p, err := Bind(sp)
	if err != nil || p.Source.Key != "span" || p.Source.TraceID != "trace" || p.Response["status"] != 0 {
		t.Fatalf("lost captured identity or explicit zero: %+v %v", p, err)
	}
	if _, ok := p.Response["body"]; ok {
		t.Fatal("attributes fabricated a response body")
	}
	if _, ok := p.Request["headers"]; ok {
		t.Fatal("missing headers fabricated")
	}
	p.Request["body"].(map[string]any)["id"] = "mutated"
	if sp.Attributes["request.body"].(map[string]any)["id"] != "actual" {
		t.Fatal("projection mutated captured source")
	}
	for _, source := range p.Fields {
		if source.Span != p.Source {
			t.Fatalf("source lost exact instance: %+v", source)
		}
	}
	missing, err := Bind(trace.Span{Service: "svc", Name: "call", Attributes: map[string]any{}})
	if err != nil || len(missing.Response) != 0 || len(missing.Request) != 0 || missing.Source.TraceID != "" {
		t.Fatalf("missing evidence fabricated: %+v %v", missing, err)
	}
}

func TestProjectionRejectsConflictingAliases(t *testing.T) {
	sp := trace.Span{Attributes: map[string]any{"response.status": 200, "http.status_code": int64(200)}}
	if _, err := Bind(sp); err != nil {
		t.Fatal("equivalent numeric aliases rejected", err)
	}
	sp.Attributes["http.status_code"] = 500
	if _, err := Bind(sp); err == nil {
		t.Fatal("conflicting aliases silently hid failure")
	}
}

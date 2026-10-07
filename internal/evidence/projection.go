// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package evidence projects captured observations without knowing contracts or
// evaluating rules. It never fills missing fields with declarations or defaults.
package evidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/verdict"
)

const SemanticsVersion = "observed-request-v1"

type Projection struct {
	Source   verdict.SpanRef
	Request  map[string]any
	Response map[string]any
	Span     map[string]any
	Fields   []verdict.FieldSource
}

func Reference(sp trace.Span) verdict.SpanRef {
	spanID, _ := sp.Attributes["otlp.span_id"].(string)
	traceID, _ := sp.Attributes["otlp.trace_id"].(string)
	return verdict.SpanRef{Key: trace.SpanKey(sp), SpanID: spanID, TraceID: traceID}
}

// Bind uses only explicit, documented attribute mappings. Unmapped attributes
// remain accessible through span.attributes with their original names.
func Bind(sp trace.Span) (Projection, error) {
	p := Projection{Source: Reference(sp), Request: map[string]any{}, Response: map[string]any{}}
	attributes := clone(sp.Attributes).(map[string]any)
	p.Span = map[string]any{"name": sp.Name, "service": sp.Service, "attributes": attributes}
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		p.Fields = append(p.Fields, verdict.FieldSource{Variable: fmt.Sprintf("span.attributes[%q]", key), Attribute: key, Span: p.Source})
	}
	for _, mapping := range []struct {
		root  string
		field string
		keys  []string
	}{
		{"request", "body", []string{"request.body"}},
		{"request", "path", []string{"request.path", "url.path"}},
		{"request", "query", []string{"request.query"}},
		{"request", "headers", []string{"request.headers"}},
		{"request", "method", []string{"request.method", "http.request.method", "http.method"}},
		{"response", "body", []string{"response.body"}},
		{"response", "status", []string{"response.status", "http.response.status_code", "http.status_code", "statusCode"}},
	} {
		var selected string
		for _, key := range mapping.keys {
			value, present := attributes[key]
			if !present {
				continue
			}
			if selected != "" {
				a, errA := json.Marshal(attributes[selected])
				b, errB := json.Marshal(value)
				if errA != nil || errB != nil || !bytes.Equal(a, b) {
					return p, fmt.Errorf("conflicting observation aliases %q and %q for %s.%s", selected, key, mapping.root, mapping.field)
				}
			}
			selected = key
			p.Fields = append(p.Fields, verdict.FieldSource{Variable: mapping.root + "." + mapping.field, Attribute: key, Span: p.Source})
		}
		if selected != "" {
			target := p.Request
			if mapping.root == "response" {
				target = p.Response
			}
			target[mapping.field] = attributes[selected]
		}
	}
	return p, nil
}

func clone(value any) any {
	switch current := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(current))
		for key, child := range current {
			copied[key] = clone(child)
		}
		return copied
	case []any:
		copied := make([]any, len(current))
		for i, child := range current {
			copied[i] = clone(child)
		}
		return copied
	default:
		return value
	}
}

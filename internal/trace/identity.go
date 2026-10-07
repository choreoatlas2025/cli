// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"fmt"
	"strings"
)

// Identity describes the evidence available in a single local trace input.
// Legacy inputs without trace IDs are bound by file hash, not a proven trace ID.
type Identity struct {
	Binding string `json:"binding"`
	TraceID string `json:"traceId,omitempty"`
}

// SpanKey identifies an explicit span or a legacy instance within its captured
// input. It is not a fabricated distributed trace ID.
func SpanKey(span Span) string {
	if id, ok := span.Attributes["otlp.span_id"].(string); ok && id != "" {
		return id
	}
	return fmt.Sprintf("%s:%s:%d", span.Service, span.Name, span.StartNanos)
}

func Identify(spans []Span) (Identity, error) {
	id := Identity{Binding: "file-only"}
	labelled := 0
	for _, span := range spans {
		for _, key := range []string{"otlp.trace_id", "otlp.span_id", "otlp.parent_span_id"} {
			value, exists := span.Attributes[key]
			if !exists {
				continue
			}
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) != text || (key != "otlp.parent_span_id" && text == "") {
				return id, fmt.Errorf("invalid trace identity attribute %s", key)
			}
		}
		if traceID, ok := span.Attributes["otlp.trace_id"].(string); ok {
			labelled++
			if id.TraceID != "" && id.TraceID != traceID {
				return id, fmt.Errorf("mixed trace identities: one validation input must contain one trace")
			}
			id.TraceID = traceID
		}
	}
	if labelled > 0 {
		if labelled != len(spans) {
			return id, fmt.Errorf("incomplete trace identity: labelled and unlabelled spans cannot be combined")
		}
		id.Binding = "trace-id"
	}
	return id, nil
}

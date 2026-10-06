// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/input"
	"io"
)

// Trace 表示追踪数据
type Trace struct {
	Spans []Span `json:"spans"`
}

// Span 表示一个追踪片段
type Span struct {
	Name         string                 `json:"name"`
	Service      string                 `json:"service"` // service alias or real name，一般与 FlowSpec.services 的 key 对应
	StartNanos   int64                  `json:"startNanos,omitempty"`
	EndNanos     int64                  `json:"endNanos,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	decoded      bool
	startPresent bool
	endPresent   bool
}

// Go callers supply timestamps directly. JSON inputs retain field presence,
// so an omitted timestamp cannot become evidence for the Unix epoch.
func (s Span) HasStart() bool { return !s.decoded || s.startPresent }
func (s Span) HasEnd() bool   { return !s.decoded || s.endPresent }

func (s *Span) UnmarshalJSON(data []byte) error {
	type plainSpan Span
	var value plainSpan
	wire := struct {
		*plainSpan
		Start json.RawMessage `json:"startNanos"`
		End   json.RawMessage `json:"endNanos"`
	}{plainSpan: &value}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	value.decoded = true
	value.startPresent, value.endPresent = len(wire.Start) != 0, len(wire.End) != 0
	for _, field := range []struct {
		name string
		raw  json.RawMessage
		dest *int64
	}{{"startNanos", wire.Start, &value.StartNanos}, {"endNanos", wire.End, &value.EndNanos}} {
		if len(field.raw) == 0 {
			continue
		}
		if bytes.Equal(bytes.TrimSpace(field.raw), []byte("null")) {
			return fmt.Errorf("%s must be an integer timestamp, not null", field.name)
		}
		if err := json.Unmarshal(field.raw, field.dest); err != nil {
			return fmt.Errorf("invalid %s: %w", field.name, err)
		}
	}
	if _, err := normalizeNumbers(value.Attributes); err != nil {
		return fmt.Errorf("invalid span attributes: %w", err)
	}
	*s = Span(value)
	return nil
}

func (s Span) MarshalJSON() ([]byte, error) {
	type plainSpan Span
	wire := struct {
		plainSpan
		Start *int64 `json:"startNanos,omitempty"`
		End   *int64 `json:"endNanos,omitempty"`
	}{plainSpan: plainSpan(s)}
	if s.HasStart() {
		wire.Start = &s.StartNanos
	}
	if s.HasEnd() {
		wire.End = &s.EndNanos
	}
	return json.Marshal(wire)
}

// ValidateTimestamps qualifies interval evidence before temporal consumers use it.
// Untimed serial matching may opt out of completeness, but supplied fields must
// still be nonnegative and an explicitly supplied interval cannot run backward.
func ValidateTimestamps(spans []Span, required bool) error {
	for i, s := range spans {
		if required && (!s.HasStart() || !s.HasEnd()) {
			return fmt.Errorf("span %d (%s.%s): startNanos and endNanos are required for time validation", i, s.Service, s.Name)
		}
		if (s.HasStart() && s.StartNanos < 0) || (s.HasEnd() && s.EndNanos < 0) ||
			(s.HasStart() && s.HasEnd() && s.EndNanos < s.StartNanos) {
			return fmt.Errorf("invalid span time range: %s.%s", s.Service, s.Name)
		}
	}
	return nil
}

// LoadFromFile 从文件加载追踪数据
func LoadFromFile(path string) (*Trace, error) {
	tb, err := input.ReadFileLimited(path, input.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}
	return Parse(tb)
}

func Parse(tb []byte) (*Trace, error) {
	return ParseWithMaxSpans(tb, input.Limits{}.Normalized().MaxSpans)
}

// Decode spans incrementally so the configured count bounds allocation as well as matching.
func ParseWithMaxSpans(tb []byte, maxSpans int) (*Trace, error) {
	var tr Trace
	if maxSpans < 1 {
		return nil, fmt.Errorf("invalid maximum span count")
	}
	wire := struct {
		Spans spanList `json:"spans"`
	}{Spans: spanList{spans: &tr.Spans, max: maxSpans}}
	decoder := json.NewDecoder(bytes.NewReader(tb))
	decoder.UseNumber()
	if err := decoder.Decode(&wire); err != nil {
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}
	// Keep json.Unmarshal's single-document requirement.
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}
	return &tr, nil
}

// A custom array decoder checks the count before appending the next span.
type spanList struct {
	spans *[]Span
	max   int
}

func (s *spanList) UnmarshalJSON(data []byte) error {
	*s.spans = nil
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('[') {
		return fmt.Errorf("invalid spans: an array is required")
	}
	for decoder.More() {
		if len(*s.spans) >= s.max {
			return fmt.Errorf("invalid trace: span count exceeds %d", s.max)
		}
		var span Span
		if err := decoder.Decode(&span); err != nil {
			return err
		}
		*s.spans = append(*s.spans, span)
	}
	_, err = decoder.Token()
	return err
}

// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Trace 表示追踪数据
type Trace struct {
	Spans []Span `json:"spans"`
}

// Span 表示一个追踪片段
type Span struct {
	Name       string                 `json:"name"`
	Service    string                 `json:"service"` // service alias or real name，一般与 FlowSpec.services 的 key 对应
	StartNanos int64                  `json:"startNanos,omitempty"`
	EndNanos   int64                  `json:"endNanos,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// LoadFromFile 从文件加载追踪数据
func LoadFromFile(path string) (*Trace, error) {
	tb, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}
	return Parse(tb)
}

func Parse(tb []byte) (*Trace, error) {
	var tr Trace
	decoder := json.NewDecoder(bytes.NewReader(tb))
	decoder.UseNumber()
	if err := decoder.Decode(&tr); err != nil {
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}
	// Keep json.Unmarshal's single-document requirement.
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}
	for i := range tr.Spans {
		if _, err := normalizeNumbers(tr.Spans[i].Attributes); err != nil {
			return nil, fmt.Errorf("invalid trace span %d attributes: %w", i, err)
		}
	}
	return &tr, nil
}

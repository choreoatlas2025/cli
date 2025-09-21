// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package trace

import (
	"encoding/json"
	"fmt"
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
	var tr Trace
	if err := json.Unmarshal(tb, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}
	return &tr, nil
}

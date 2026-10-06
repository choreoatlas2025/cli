// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

const DefaultMaxBytes int64 = 32 << 20
const MaxInputFiles = 128
const MaxExpressionBytes = 64 << 10
const MaxExpressionTotalBytes = 1 << 20
const MaxExpressions = 10000

type Limits struct {
	MaxInputBytes int64  `json:"maxInputBytes"`
	MaxSpans      int    `json:"maxSpans"`
	MaxSteps      int    `json:"maxSteps"`
	MaxCELCost    uint64 `json:"maxCELCost"`
	TimeoutMs     int64  `json:"timeoutMs"`
}

func (l Limits) Normalized() Limits {
	if l.MaxInputBytes == 0 {
		l.MaxInputBytes = DefaultMaxBytes
	}
	if l.MaxSpans == 0 {
		l.MaxSpans = 10000
	}
	if l.MaxSteps == 0 {
		l.MaxSteps = 2000
	}
	if l.MaxCELCost == 0 {
		l.MaxCELCost = 100000
	}
	if l.TimeoutMs == 0 {
		l.TimeoutMs = 10000
	}
	return l
}

func (l Limits) Validate() error {
	l = l.Normalized()
	if l.MaxInputBytes < 1 || l.MaxInputBytes > math.MaxInt64/2 || l.MaxSpans < 1 || l.MaxSteps < 1 || l.TimeoutMs < 1 || l.TimeoutMs > int64(time.Hour/time.Millisecond) {
		return fmt.Errorf("invalid resource limits: positive sizes/counts and timeout up to one hour are required")
	}
	return nil
}

func ReadFileLimited(path string, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err == nil && int64(len(data)) > max {
		return nil, fmt.Errorf("invalid input: file exceeds %d byte limit", max)
	}
	return data, err
}

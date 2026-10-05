// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"fmt"
	"github.com/choreoatlas2025/cli/internal/input"
	"math"
	"strings"

	"github.com/choreoatlas2025/cli/internal/trace"
)

type ValidationConfig struct {
	Semantic    bool         `json:"semantic"`
	Causality   string       `json:"causality"`
	ToleranceMs int64        `json:"causalityToleranceMs"`
	Limits      input.Limits `json:"limits"`
}

func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{Semantic: true, Causality: "temporal", ToleranceMs: 50}
}

// ExecutionIdentity records the tool, settings and input used to produce evidence.
type ExecutionIdentity struct {
	BuildChannel  string           `json:"buildChannel,omitempty"`
	Version       string           `json:"version"`
	GitCommit     string           `json:"gitCommit"`
	ValidatorHash string           `json:"validatorHash"`
	Config        ValidationConfig `json:"config"`
	TraceHash     string           `json:"traceHash"`
	TraceIdentity trace.Identity   `json:"traceIdentity"`
}

func (c ValidationConfig) Validate() error {
	if c.Causality != "temporal" && c.Causality != "strict" && c.Causality != "off" {
		return fmt.Errorf("invalid causality mode: %s", c.Causality)
	}
	if c.ToleranceMs < 0 || c.ToleranceMs > math.MaxInt64/1000000 {
		return fmt.Errorf("invalid causality-tolerance: outside supported nonnegative range")
	}
	return c.Limits.Validate()
}

func (e ExecutionIdentity) Validate() error {
	if strings.TrimSpace(e.Version) == "" || strings.TrimSpace(e.GitCommit) == "" || !isSHA256(e.ValidatorHash) || !isSHA256(e.TraceHash) {
		return fmt.Errorf("invalid baseline provenance: missing tool or trace identity; record a new baseline")
	}
	if err := e.Config.Validate(); err != nil {
		return err
	}
	id := e.TraceIdentity
	if (id.Binding == "file-only" && id.TraceID == "") || (id.Binding == "trace-id" && strings.TrimSpace(id.TraceID) != "" && strings.TrimSpace(id.TraceID) == id.TraceID) {
		return nil
	}
	return fmt.Errorf("invalid baseline provenance: incomplete trace identity")
}

func isSHA256(hash string) bool {
	if len(hash) != 71 || !strings.HasPrefix(hash, "sha256:") {
		return false
	}
	for _, c := range hash[7:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

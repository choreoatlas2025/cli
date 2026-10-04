// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"

	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
	"github.com/choreoatlas2025/cli/internal/validate"
)

func configureValidation(config spec.ValidationConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	validate.EnableSemantic = config.Semantic
	validate.GlobalCausalityMode = validate.CausalityMode(config.Causality)
	validate.GlobalCausalityToleranceMs = config.ToleranceMs
	return nil
}

func executionIdentity(tr *trace.Trace, tracePath string, config spec.ValidationConfig) (spec.ExecutionIdentity, error) {
	id := spec.ExecutionIdentity{Version: Version, GitCommit: GitCommit, Config: config}
	var err error
	id.TraceHash, err = spec.HashFile(tracePath)
	if err != nil {
		return id, err
	}
	id.TraceIdentity, err = trace.Identify(tr.Spans)
	if err != nil {
		id.TraceIdentity = trace.Identity{Binding: "invalid"}
	}
	path, err := os.Executable()
	if err != nil {
		return id, err
	}
	id.ValidatorHash, err = spec.HashFile(path)
	return id, err
}

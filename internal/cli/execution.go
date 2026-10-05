// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"

	"github.com/choreoatlas2025/cli/internal/input"
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

func executionIdentity(tr *trace.Trace, traceHash string, config spec.ValidationConfig, files *input.Snapshot) (spec.ExecutionIdentity, error) {
	id := spec.ExecutionIdentity{Version: Version, GitCommit: GitCommit, Config: config, TraceHash: traceHash}
	var err error
	id.TraceIdentity, err = trace.Identify(tr.Spans)
	if err != nil {
		id.TraceIdentity = trace.Identity{Binding: "invalid"}
	}
	path, err := os.Executable()
	if err != nil {
		return id, err
	}
	file, err := files.Read(path)
	if err != nil {
		return id, err
	}
	id.ValidatorHash = file.Hash()
	return id, nil
}

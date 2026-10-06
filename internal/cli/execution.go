// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/choreoatlas2025/cli/internal/input"
	"github.com/choreoatlas2025/cli/internal/spec"
	"github.com/choreoatlas2025/cli/internal/trace"
)

var errIncompleteEvidence = errors.New("evidence incomplete")

func executionIdentity(tr *trace.Trace, traceHash string, config spec.ValidationConfig, files *input.Snapshot) (spec.ExecutionIdentity, error) {
	return executionIdentityWith(tr, traceHash, config, files, os.Executable)
}

func executionIdentityWith(tr *trace.Trace, traceHash string, config spec.ValidationConfig, files *input.Snapshot, executable func() (string, error)) (spec.ExecutionIdentity, error) {
	id := spec.ExecutionIdentity{Version: Version, GitCommit: GitCommit, BuildChannel: BuildChannel, Config: config, TraceHash: traceHash}
	var err error
	id.TraceIdentity, err = trace.Identify(tr.Spans)
	if err != nil {
		id.TraceIdentity = trace.Identity{Binding: "invalid"}
	}
	path, err := executable()
	if err != nil {
		return id, fmt.Errorf("%w: executing tool identity unavailable: %w", errIncompleteEvidence, err)
	}
	hash, err := files.HashExecutable(path)
	if err != nil {
		return id, fmt.Errorf("%w: executing tool identity unavailable: %w", errIncompleteEvidence, err)
	}
	id.ValidatorHash = hash
	return id, nil
}
